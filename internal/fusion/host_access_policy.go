/*
Copyright 2022 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var hostAccessPolicyResourceFunctions *BaseResourceFunctions

func resourceHostAccessPolicy() *schema.Resource {

	vp := &hostAccessPolicyProvider{BaseResourceProvider{ResourceKind: "host_access_policy"}}
	hostAccessPolicyResourceFunctions = NewBaseResourceFunctions("host_access_policy", vp)

	hostAccessPolicyResourceFunctions.Resource.Schema = map[string]*schema.Schema{
		"name": {
			Type:     schema.TypeString,
			Required: true,
		},
		"display_name": {
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
		},
		"iqn": {
			Type:     schema.TypeString,
			Required: true,
		},
		"personality": {
			Type:     schema.TypeString,
			Required: true,
		},
	}
	return hostAccessPolicyResourceFunctions.Resource
}

// Implements ResourceProvider
type hostAccessPolicyProvider struct {
	BaseResourceProvider
}

func (vp *hostAccessPolicyProvider) PrepareCreate(ctx context.Context, d *schema.ResourceData) (InvokeWriteAPI, ResourcePost, error) {
	hostAccessPolicyName := rdString(ctx, d, "name")
	displayName := rdString(ctx, d, "display_name")
	iqn := rdString(ctx, d, "iqn")
	personality := rdString(ctx, d, "personality")

	body := hmrest.HostAccessPoliciesPost{
		Name:        hostAccessPolicyName,
		DisplayName: displayName,
		Iqn:         iqn,
		Personality: personality,
	}

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.HostAccessPoliciesApi.CreateHostAccessPolicy(ctx, *body.(*hmrest.HostAccessPoliciesPost), nil)
		return &op, err
	}
	return fn, &body, nil
}

func (vp *hostAccessPolicyProvider) ReadResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	hap, _, err := client.HostAccessPoliciesApi.GetHostAccessPolicyById(ctx, d.Id(), nil)
	if err != nil {
		return err
	}

	d.Set("name", hap.Name)
	d.Set("display_name", hap.DisplayName)
	d.Set("iqn", hap.Iqn)
	d.Set("personality", hap.Personality)

	return nil
}

func (vp *hostAccessPolicyProvider) PrepareDelete(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, error) {
	hostAccessPolicyName := rdString(ctx, d, "name")

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.HostAccessPoliciesApi.DeleteHostAccessPolicy(ctx, hostAccessPolicyName, nil)
		return &op, err
	}
	return fn, nil
}
