/*
Copyright 2022 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	context "context"
	"fmt"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var tenantSpaceResourceFunctions *BaseResourceFunctions

// Implements ResourceProvider
type tenantSpaceProvider struct {
	BaseResourceProvider
}

// This is our entry point for the Storage Class resource. Get it movin'
func resourceTenantSpace() *schema.Resource {

	vp := &tenantSpaceProvider{BaseResourceProvider{ResourceKind: "TenantSpace"}}
	tenantSpaceResourceFunctions = NewBaseResourceFunctions("TenantSpace", vp)

	tenantSpaceResourceFunctions.Resource.Schema = map[string]*schema.Schema{
		"tenant_name": {
			Type:     schema.TypeString,
			Required: true,
		},
		"name": {
			Type:     schema.TypeString,
			Required: true,
		},
		"display_name": {
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
		},
	}

	return tenantSpaceResourceFunctions.Resource
}

func (vp *tenantSpaceProvider) PrepareCreate(ctx context.Context, d *schema.ResourceData) (InvokeWriteAPI, ResourcePost, error) {
	name := rdString(ctx, d, "name")
	tenant := rdString(ctx, d, "tenant_name")
	displayName := rdString(ctx, d, "display_name")

	body := hmrest.TenantSpacePost{
		Name:        name,
		DisplayName: displayName,
	}

	// REVIEW: Should we return an interface instead? What does that look like? The closure lets us use variables above.
	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.TenantSpacesApi.CreateTenantSpace(ctx, *body.(*hmrest.TenantSpacePost), tenant, nil)
		return &op, err
	}
	return fn, &body, nil
}

func (vp *tenantSpaceProvider) ReadResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	ts, _, err := client.TenantSpacesApi.GetTenantSpaceById(ctx, d.Id(), nil)
	if err != nil {
		return err
	}

	d.Set("name", ts.Name)
	d.Set("display_name", ts.DisplayName)
	d.Set("tenant_name", ts.Tenant.Name)
	return nil
}

func (vp *tenantSpaceProvider) PrepareDelete(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, error) {
	tenant := d.Get("tenant_name").(string)
	tenantSpaceName := rdString(ctx, d, "name")

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.TenantSpacesApi.DeleteTenantSpace(ctx, tenant, tenantSpaceName, nil)
		return &op, err
	}
	return fn, nil
}

func (vp *tenantSpaceProvider) PrepareUpdate(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, []ResourcePatch, error) {
	var patches []ResourcePatch // []*hmrest.TenantSpacePatch

	tenant := d.Get("tenant_name").(string)
	tenantSpaceName := d.Get("name").(string)
	if d.HasChangeExcept("display_name") {
		return nil, nil, fmt.Errorf("attempting to update an immutable field")
	} else {
		displayName := d.Get("display_name").(string)
		tflog.Info(ctx, "Updating", "display_name", displayName)
		patches = append(patches, &hmrest.TenantSpacePatch{
			DisplayName: &hmrest.NullableString{Value: displayName},
		})
	}

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.TenantSpacesApi.UpdateTenantSpace(ctx, *body.(*hmrest.TenantSpacePatch), tenant, tenantSpaceName, nil)
		return &op, err
	}

	return fn, patches, nil
}
