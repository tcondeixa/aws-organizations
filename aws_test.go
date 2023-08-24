package org

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/stretchr/testify/require"
)

func TestOrgTree(t *testing.T) {

	ctx := context.Background()
	awsCfg, err := config.LoadDefaultConfig(ctx)
	require.NoError(t, err)

	orgTreeProvider := NewAWSOrgTreeProvider(awsCfg)
	orgTree, err := orgTreeProvider.OrgTree(ctx)
	require.NoError(t, err)

	orgUnits := orgTree.OrgUnitsInfo()
	fmt.Println(orgUnits)

	accounts := orgTree.Accounts()
	fmt.Println(accounts)
}
