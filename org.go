package org

type Account struct {
	ID     string
	Name   string
	Email  string
	Active bool
	Tags   map[string]string
}

type AccountInfo struct {
	ID         string
	Name       string
	Email      string
	Active     bool
	OUNamePath []string
	OUIdPath   []string
	Org        string
	Tags       map[string]string
}

type OrgTree struct {
	ID                string
	ManagementAccount Account
	Tags              map[string]string
	OrgUnits          map[string]OrgUnit
}

type OrgUnitInfo struct {
	Name       string
	ID         string
	ParentPath []string
	Tags       map[string]string
}

type OrgUnit struct {
	Name     string
	ID       string
	Tags     map[string]string
	OrgUnits map[string]OrgUnit
	Accounts map[string]Account
}

func (t *OrgTree) OrgUnitsInfo() []OrgUnitInfo {
	return walkOUs(t.OrgUnits, nil)
}

func walkOUs(orgUnits map[string]OrgUnit, parentPath []string) []OrgUnitInfo {
	units := []OrgUnitInfo{}
	for _, ou := range orgUnits {
		units = append(units, OrgUnitInfo{
			Name:       ou.Name,
			ID:         ou.ID,
			ParentPath: parentPath,
		})

		childPath := parentPath[:]
		childPath = append(childPath, ou.Name)

		units = append(units, walkOUs(ou.OrgUnits, childPath)...)
	}

	return units
}

func (t *OrgTree) Accounts() map[string]AccountInfo {
	return walkAccounts(t.OrgUnits, nil)
}

func walkAccounts(orgUnits map[string]OrgUnit, parentPath []string) map[string]AccountInfo {
	accounts := map[string]AccountInfo{}
	for _, ou := range orgUnits {
		for _, account := range ou.Accounts {
			accounts[account.Name] = AccountInfo{
				Name:       account.Name,
				ID:         account.ID,
				Active:     account.Active,
				Email:      account.Email,
				Tags:       account.Tags,
				OUNamePath: parentPath,
			}
		}

		childPath := parentPath[:]
		childPath = append(childPath, ou.Name)

		for _, account := range walkAccounts(ou.OrgUnits, childPath) {
			accounts[account.Name] = AccountInfo{
				Name:       account.Name,
				ID:         account.ID,
				Active:     account.Active,
				Email:      account.Email,
				Tags:       account.Tags,
				OUNamePath: childPath,
			}
		}
	}

	return accounts
}
