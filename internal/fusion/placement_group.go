/*
Copyright 2022 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"fmt"
	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/antihax/optional"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var validPlacementEngines = []hmrest.PlacementEngine{hmrest.PURE1META_PlacementEngine, hmrest.HEURISTICS_PlacementEngine}
var validPlacementEngineStrings []string

func init() {
	for _, i := range validPlacementEngines {
		validPlacementEngineStrings = append(validPlacementEngineStrings, string(i))
	}
}

var placementGroupResourceFunctions *BaseResourceFunctions

// This is our entry point for the Storage Class resource. Get it movin'
func resourcePlacementGroup() *schema.Resource {
	vp := &placementGroupProvider{BaseResourceProvider{ResourceKind: "PlacementGroup"}}
	placementGroupResourceFunctions = NewBaseResourceFunctions("PlacementGroup", vp)

	placementGroupResourceFunctions.Resource.Schema = map[string]*schema.Schema{
		"name": {
			Type:     schema.TypeString,
			Required: true,
		},
		"display_name": {
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
		},
		"tenant_name": {
			Type:     schema.TypeString,
			Required: true,
		},
		"tenant_space_name": {
			Type:     schema.TypeString,
			Required: true,
		},
		"region_name": {
			Type:     schema.TypeString,
			Required: true,
		},
		"availability_zone_name": {
			Type:     schema.TypeString,
			Required: true,
		},
		"storage_service_name": {
			Type:     schema.TypeString,
			Required: true,
		},
		"placement_engine": {
			Type:         schema.TypeString,
			Optional:     true,
			Computed:     true,
			ValidateFunc: validation.StringInSlice(validPlacementEngineStrings, false),
		},
		"destroy_snapshots_on_delete": {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  false,
			Description: "Before deleting placement group, snapshots within the placement group will be deleted. " +
				"If `false` then any snapshots will need to be deleted as a separate step before removing the placement group",
		},
	}

	return placementGroupResourceFunctions.Resource
}

// Implements ResourceProvider
type placementGroupProvider struct {
	BaseResourceProvider
}

func placementEngineFromString(s string) *hmrest.PlacementEngine {
	if s == "" {
		return nil
	}
	for _, i := range validPlacementEngines {
		if s == string(i) {
			return &i
		}
	}
	// We shouldn't get here, we already prevalidate values comming from the resource parameter
	panic(fmt.Sprintf("Unexpected value: %s", s))
}

func (vp *placementGroupProvider) PrepareCreate(ctx context.Context, d *schema.ResourceData) (InvokeWriteAPI, ResourcePost, error) {
	name := rdString(ctx, d, "name")
	displayName := rdStringDefault(ctx, d, "display_name", name)
	tenantName := rdString(ctx, d, "tenant_name")
	tenantSpaceName := rdString(ctx, d, "tenant_space_name")
	storageService := rdString(ctx, d, "storage_service_name")

	regionName := rdString(ctx, d, "region_name")
	availabilityZone := rdString(ctx, d, "availability_zone_name")

	placementEngine := placementEngineFromString(rdString(ctx, d, "placement_engine"))
	tflog.Debug(ctx, "PlacementGroup.CreateResource()", "ts", tenantSpaceName, "name", name)

	body := hmrest.PlacementGroupPost{
		Name:             name,
		DisplayName:      displayName,
		Region:           regionName,
		AvailabilityZone: availabilityZone,
		PlacementEngine:  placementEngine,
		StorageService:   storageService,
	}

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.PlacementGroupsApi.CreatePlacementGroup(ctx, *body.(*hmrest.PlacementGroupPost), tenantName, tenantSpaceName, nil)
		return &op, err
	}
	return fn, &body, nil
}

func (vp *placementGroupProvider) ReadResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	tflog.Debug(ctx, "PlacementGroup.ReadResource()", "id", d.Id())
	pg, _, err := client.PlacementGroupsApi.GetPlacementGroupById(ctx, d.Id(), nil)
	if err != nil {
		return err
	}

	d.Set("name", pg.Name)
	d.Set("display_name", pg.DisplayName)
	d.Set("tenant_name", pg.Tenant.Name)
	d.Set("tenant_space_name", pg.TenantSpace.Name)
	d.Set("availability_zone_name", pg.AvailabilityZone.Name)
	d.Set("storage_service_name", pg.StorageService.Name)
	d.Set("placement_engine", pg.PlacementEngine)

	az, _, err := client.AvailabilityZonesApi.GetAvailabilityZoneById(ctx, pg.AvailabilityZone.Id, nil)
	if err != nil {
		return err
	}
	d.Set("region_name", az.Region.Name)

	return nil
}

func (vp *placementGroupProvider) PrepareDelete(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, error) {
	placementGroupName := rdString(ctx, d, "name")
	tenantName := rdString(ctx, d, "tenant_name")
	tenantSpaceName := rdString(ctx, d, "tenant_space_name")
	destroySnaps := d.Get("destroy_snapshots_on_delete").(bool)

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		// TODO: HM-2437 this should be patches
		if destroySnaps {
			tflog.Debug(ctx, "Destroying relevant snapshots")
			snapshots, _, err := client.SnapshotsApi.ListSnapshots(ctx, tenantName, tenantSpaceName, &hmrest.SnapshotsApiListSnapshotsOpts{
				PlacementGroup: optional.NewString(placementGroupName),
			})
			if err != nil {
				tflog.Error(ctx, "failed listing snapshots", "tenant_name", tenantName, "tenant_space_name", tenantSpaceName)
				utilities.TraceError(ctx, err)
				return nil, err
			}
			for _, snap := range snapshots.Items {
				tflog.Info(ctx, "Deleting Snapshot", "name", snap.Name)
				op, _, err := client.SnapshotsApi.DeleteSnapshot(ctx, tenantName, tenantSpaceName, snap.Name, nil)
				if err != nil {
					utilities.TraceError(ctx, err)
					return &op, err
				}
				succeeded, err := WaitOnOperation(ctx, &op, client)
				if err != nil {
					utilities.TraceError(ctx, err)
					return &op, err
				}
				if !succeeded {
					tflog.Error(ctx, "failed deleting a snapshot", "snapshot_name", snap.Name)
					return &op, fmt.Errorf("failed deleting a snapshot as part of deleting volume")
				}
			}
		}

		op, _, err := client.PlacementGroupsApi.DeletePlacementGroup(ctx, tenantName, tenantSpaceName, placementGroupName, nil)
		return &op, err
	}
	return fn, nil
}

func (vp *placementGroupProvider) PrepareUpdate(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, []ResourcePatch, error) {

	var patches []ResourcePatch // []*hmrest.TenantSpacePatch

	placementGroupName := rdString(ctx, d, "name")
	tenantName := rdString(ctx, d, "tenant_name")
	tenantSpaceName := rdString(ctx, d, "tenant_space_name")

	if d.HasChangeExcept("display_name") {
		return nil, nil, fmt.Errorf("attempting to update an immutable field")
	} else if d.HasChange("display_name") {
		displayName := d.Get("display_name").(string)
		tflog.Info(ctx, "Updating", "display_name", displayName)
		patches = append(patches, &hmrest.PlacementGroupPatch{
			DisplayName: &hmrest.NullableString{Value: displayName},
		})
	}
	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.PlacementGroupsApi.UpdatePlacementGroup(ctx, *body.(*hmrest.PlacementGroupPatch), tenantName, tenantSpaceName, placementGroupName, nil)
		return &op, err
	}
	return fn, patches, nil
}
