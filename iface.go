package org

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
)

type Organizations interface {
	ListRoots(ctx context.Context, params *organizations.ListRootsInput, optFns ...func(*organizations.Options)) (*organizations.ListRootsOutput, error)
	ListChildren(ctx context.Context, params *organizations.ListChildrenInput, optFns ...func(*organizations.Options)) (*organizations.ListChildrenOutput, error)
	DescribeAccount(ctx context.Context, params *organizations.DescribeAccountInput, optFns ...func(*organizations.Options)) (*organizations.DescribeAccountOutput, error)
	DescribeOrganization(ctx context.Context, params *organizations.DescribeOrganizationInput, optFns ...func(*organizations.Options)) (*organizations.DescribeOrganizationOutput, error)
	DescribeOrganizationalUnit(ctx context.Context, params *organizations.DescribeOrganizationalUnitInput, optFns ...func(*organizations.Options)) (*organizations.DescribeOrganizationalUnitOutput, error)
	ListAccountsForParent(ctx context.Context, params *organizations.ListAccountsForParentInput, optFns ...func(*organizations.Options)) (*organizations.ListAccountsForParentOutput, error)
}

type ResourceGroupsTagging interface {
	GetResources(ctx context.Context, params *resourcegroupstaggingapi.GetResourcesInput, optFns ...func(*resourcegroupstaggingapi.Options)) (*resourcegroupstaggingapi.GetResourcesOutput, error)
}
