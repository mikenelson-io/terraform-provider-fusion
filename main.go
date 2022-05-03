/*
Copyright 2022 PureStorage Inc
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	root "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/fusion"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() *schema.Provider {
			return root.Provider()
		},
	})
}
