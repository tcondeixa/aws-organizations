package org

import (
	"context"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
)

type OrgTreeProvider interface {
	OrgTree(ctx context.Context) (*OrgTree, error)
}

type AWSOrgTreeProvider struct {
	org  Organizations
	tag  ResourceGroupsTagging
	lock sync.RWMutex
}

func NewAWSOrgTreeProvider(cfg aws.Config) *AWSOrgTreeProvider {
	return &AWSOrgTreeProvider{
		org: organizations.NewFromConfig(cfg),
		tag: resourcegroupstaggingapi.NewFromConfig(cfg, func(o *resourcegroupstaggingapi.Options) {
			o.Region = "us-east-1"
		}),
		lock: sync.RWMutex{},
	}
}

type TagsAccounts struct {
	err  error
	tags map[string]map[string]string
}

func (a *AWSOrgTreeProvider) OrgTree(ctx context.Context) (*OrgTree, error) {

	channel := make(chan TagsAccounts)
	go func(c chan TagsAccounts) {
		//Get Account Tags
		tags, err := a.getTagsForAccounts(ctx)
		if err != nil {
			c <- TagsAccounts{
				err:  err,
				tags: nil,
			}
		}

		c <- TagsAccounts{
			err:  nil,
			tags: tags,
		}
	}(channel)

	// Get Org tree
	tree, err := a.getOrgTree(ctx)
	if err != nil {
		return nil, err
	}

	result := <-channel
	if result.err != nil {
		return nil, result.err
	}

	//Enrich Account Structure in the Org Tree
	addTagsAccounts(tree.OrgUnits, result.tags)

	return tree, nil
}

func (r *AWSOrgTreeProvider) getTagsForAccounts(ctx context.Context) (map[string]map[string]string, error) {

	accountsTags := make(map[string]map[string]string)

	paginator := resourcegroupstaggingapi.NewGetResourcesPaginator(r.tag, &resourcegroupstaggingapi.GetResourcesInput{
		ResourceTypeFilters:      []string{"organizations:account"},
		IncludeComplianceDetails: aws.Bool(false),
	})

	for paginator.HasMorePages() {
		resp, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, resource := range resp.ResourceTagMappingList {
			accountTags := make(map[string]string)

			splitArn := strings.Split(aws.ToString(resource.ResourceARN), "/")
			accountId := splitArn[len(splitArn)-1]
			for _, tag := range resource.Tags {
				accountTags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
			}
			accountsTags[accountId] = accountTags
		}
	}

	return accountsTags, nil
}

func addTagsAccounts(orgUnits map[string]OrgUnit, tags map[string]map[string]string) {
	for k1, ou := range orgUnits {
		for k2, account := range ou.Accounts {
			if val, ok := tags[account.ID]; ok {
				account.Tags = val
				orgUnits[k1].Accounts[k2] = account
			}
		}

		addTagsAccounts(ou.OrgUnits, tags)

	}
}

type OrgDescription struct {
	err    error
	output *organizations.DescribeOrganizationOutput
}

func (a *AWSOrgTreeProvider) getOrgTree(ctx context.Context) (*OrgTree, error) {

	channel := make(chan OrgDescription)
	go func(c chan OrgDescription) {
		resp, err := a.org.DescribeOrganization(ctx, &organizations.DescribeOrganizationInput{})

		c <- OrgDescription{
			err:    err,
			output: resp,
		}

	}(channel)

	p := organizations.NewListRootsPaginator(a.org, &organizations.ListRootsInput{})
	var roots []types.Root
	for p.HasMorePages() {
		resp, err := p.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		roots = append(roots, resp.Roots...)
	}

	result := <-channel
	if result.err != nil {
		return nil, result.err
	}

	tree := &OrgTree{
		ID:       aws.ToString(result.output.Organization.Id),
		OrgUnits: make(map[string]OrgUnit, len(roots)),
	}
	for _, root := range roots {
		orgUnits, accounts, managementAccount, err := a.walkOUTree(ctx, root.Id, aws.ToString(result.output.Organization.Id), aws.ToString(result.output.Organization.MasterAccountId), []string{aws.ToString(root.Name)})
		if err != nil {
			return nil, err
		}

		if managementAccount != nil {
			tree.ManagementAccount = *managementAccount
		}

		tree.OrgUnits[aws.ToString(root.Name)] = OrgUnit{
			Name:     aws.ToString(root.Name),
			ID:       aws.ToString(root.Id),
			OrgUnits: orgUnits,
			Accounts: accounts,
		}
	}

	return tree, nil
}

func (a *AWSOrgTreeProvider) walkOUTree(ctx context.Context, parentID *string, orgId, managementAccountID string, OU []string) (map[string]OrgUnit, map[string]Account, *Account, error) {

	channel := make(chan OrgAccount)
	go func(c chan OrgAccount) {

		var managementAccount *Account
		pa := organizations.NewListAccountsForParentPaginator(a.org, &organizations.ListAccountsForParentInput{
			ParentId: parentID,
		})

		var accountsList []types.Account
		for pa.HasMorePages() {
			resp, err := pa.NextPage(ctx)
			if err != nil {
				c <- OrgAccount{
					err:               err,
					accounts:          nil,
					managementAccount: nil,
				}
			}
			accountsList = append(accountsList, resp.Accounts...)
		}

		accounts := map[string]Account{}
		for _, account := range accountsList {
			account := Account{
				Name:   aws.ToString(account.Name),
				ID:     aws.ToString(account.Id),
				Email:  aws.ToString(account.Email),
				Active: account.Status == types.AccountStatusActive,
				Tags:   map[string]string{},
			}
			accounts[account.Name] = account

			if account.ID == managementAccountID {
				managementAccount = &account
			}
		}

		c <- OrgAccount{
			err:               nil,
			accounts:          accounts,
			managementAccount: managementAccount,
		}

	}(channel)

	p := organizations.NewListChildrenPaginator(a.org, &organizations.ListChildrenInput{
		ChildType: types.ChildTypeOrganizationalUnit,
		ParentId:  parentID,
	})
	var children []types.Child
	for p.HasMorePages() {
		resp, err := p.NextPage(ctx)
		if err != nil {
			return nil, nil, nil, err
		}
		children = append(children, resp.Children...)
	}

	channel2 := make(chan OrgUnitAccountResponse, len(children))

	orgUnits := make(map[string]OrgUnit)
	wg := sync.WaitGroup{}
	wg.Add(len(children))
	for _, child := range children {
		go func(child types.Child, c chan OrgUnitAccountResponse, wg *sync.WaitGroup) {
			defer wg.Done()
			resp, err := a.org.DescribeOrganizationalUnit(ctx, &organizations.DescribeOrganizationalUnitInput{
				OrganizationalUnitId: child.Id,
			})
			if err != nil {
				c <- OrgUnitAccountResponse{
					err:        err,
					management: nil,
					accounts:   nil,
					orgUnit:    nil,
				}
			}

			childOU := OU[:]
			childOU = append(childOU, aws.ToString(resp.OrganizationalUnit.Name))

			childOrgUnits, childAccounts, childManagementAccount, err := a.walkOUTree(ctx, child.Id, orgId, managementAccountID, childOU)
			if err != nil {
				c <- OrgUnitAccountResponse{
					err:        err,
					management: nil,
					accounts:   nil,
					orgUnit:    nil,
				}
			}

			var managementAccount *Account
			if childManagementAccount != nil {
				managementAccount = childManagementAccount
			}

			orgUnits[aws.ToString(resp.OrganizationalUnit.Name)] = OrgUnit{
				Name:     aws.ToString(resp.OrganizationalUnit.Name),
				ID:       aws.ToString(resp.OrganizationalUnit.Id),
				OrgUnits: childOrgUnits,
				Accounts: childAccounts,
			}

			c <- OrgUnitAccountResponse{
				err:        err,
				management: managementAccount,
				accounts:   childAccounts,
				orgUnit:    childOrgUnits,
			}

		}(child, channel2, &wg)
	}

	var managementAccount *Account
	allAccount := map[string]Account{}
	allOrgUnits := map[string]OrgUnit{}

	wg.Wait()
	for i := 0; i < len(children); i++ {
		r := <-channel2
		if r.err != nil {
			return nil, nil, nil, r.err
		}

		for k, v := range r.accounts {
			allAccount[k] = v
		}

		for k, v := range r.orgUnit {
			allOrgUnits[k] = v
		}

		if r.management != nil {
			managementAccount = r.management
		}
	}

	result := <-channel
	if result.err != nil {
		return nil, nil, nil, result.err
	}

	if result.managementAccount != nil {
		managementAccount = result.managementAccount
	}

	for k, v := range result.accounts {
		allAccount[k] = v
	}

	return orgUnits, allAccount, managementAccount, nil
}

type OrgAccount struct {
	err               error
	accounts          map[string]Account
	managementAccount *Account
}

type OrgUnitAccountResponse struct {
	err        error
	accounts   map[string]Account
	orgUnit    map[string]OrgUnit
	management *Account
}
