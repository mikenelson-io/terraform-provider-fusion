/*
Copyright 2022 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

// This is a hand written file
// We can put useful helper function. Most old frontdoor clients logic went here
package fusion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/antihax/optional"
)

func ToModelError(err error) (*ModelError, error) {
	// Check the error document: http code, pure code, message.
	var swagErr GenericSwaggerError
	if ok := errors.As(err, &swagErr); !ok {
		return nil, fmt.Errorf("cannot convert error to GenericSwaggerError")
	}
	respErr := &ErrorResponse{}
	err = json.Unmarshal(swagErr.Body(), respErr)
	if err != nil {
		return nil, err
	}
	return respErr.Error_, nil
}

func (a *SnapshotsApiService) CreateSnapshotBy(ctx context.Context, body SnapshotPost, tenantName, tenantSpaceName, requestId string) (Operation, error) {
	var empty Operation
	op, _, err := a.CreateSnapshot(ctx, body, tenantName, tenantSpaceName, &SnapshotsApiCreateSnapshotOpts{XRequestID: optional.NewString(requestId)})
	if err != nil {
		return empty, fmt.Errorf("Error creating snapshots '%v': %w", body, err)
	}
	return op, nil
}

// Create a snapshot for a given list of volume names
func (a *SnapshotsApiService) CreateSnapshotByVolumes(
	ctx context.Context,
	tenantName string,
	tenantSpaceName string,
	volumeNames []string,
	snapshotName string,
	displayName string,
	requestId string) (Operation, error) {
	body := SnapshotPost{
		Name:        snapshotName,
		DisplayName: displayName,
		Volumes:     volumeNames,
	}
	return a.CreateSnapshotBy(ctx, body, tenantName, tenantSpaceName, requestId)
}

// Create a snapshot for a given placement group
func (a *SnapshotsApiService) CreateSnapshotByPlacementGroup(
	ctx context.Context,
	tenantName string,
	tenantSpaceName string,
	placemenGroupName string,
	snapshotName string,
	displayName string,
	requestId string) (Operation, error) {
	body := SnapshotPost{
		Name:           snapshotName,
		DisplayName:    displayName,
		PlacementGroup: placemenGroupName,
	}
	return a.CreateSnapshotBy(ctx, body, tenantName, tenantSpaceName, requestId)
}

func (a *VolumesApiService) CreateVolumeFromSnapshot(
	ctx context.Context,
	tenantName string,
	tenantSpaceName string,
	volumeName string,
	displayName string,
	storageClass string,
	protectionPolicy string,
	placementGroup string,
	sourceVolumeSnapshotLink string,
	requestId string) (Operation, error) {
	body := VolumePost{
		Name:             volumeName,
		DisplayName:      displayName,
		StorageClass:     storageClass,
		ProtectionPolicy: protectionPolicy,
		PlacementGroup:   placementGroup,
		SourceLink:       sourceVolumeSnapshotLink,
	}
	op, _, err := a.CreateVolume(ctx, body, tenantName, tenantSpaceName, &VolumesApiCreateVolumeOpts{XRequestID: optional.NewString(requestId)})
	if err != nil {
		var empty Operation
		return empty, fmt.Errorf("error creating volumes from snapshot: %w", err)
	}
	return op, nil
}

func (a *VolumesApiService) UpdateVolumeBy(
	ctx context.Context,
	tenantName string,
	tenantSpaceName string,
	volumeName string,
	patch VolumePatch,
	requestId string) (Operation, error) {
	var empty Operation
	op, _, err := a.UpdateVolume(ctx, patch, tenantName, tenantSpaceName, volumeName, &VolumesApiUpdateVolumeOpts{XRequestID: optional.NewString(requestId)})
	if err != nil {
		return empty, fmt.Errorf("error Updating Volumes: %w", err)
	}
	return op, nil
}

func (a *VolumesApiService) UpdateVolumeDisplayName(
	ctx context.Context,
	tenantName string,
	tenantSpaceName string,
	volumeName string,
	displayName string,
	requestId string) (Operation, error) {
	patch := VolumePatch{DisplayName: &NullableString{displayName}}
	return a.UpdateVolumeBy(ctx, tenantName, tenantSpaceName, volumeName, patch, requestId)
}

func (a *VolumesApiService) UpdateVolumeStorageClass(
	ctx context.Context,
	tenantName string,
	tenantSpaceName string,
	volumeName string,
	storageClassName string,
	requestId string) (Operation, error) {
	patch := VolumePatch{StorageClass: &NullableString{storageClassName}}
	return a.UpdateVolumeBy(ctx, tenantName, tenantSpaceName, volumeName, patch, requestId)
}

func (a *VolumesApiService) UpdateVolumePlacementGroup(
	ctx context.Context,
	tenantName string,
	tenantSpaceName string,
	volumeName string,
	placementGroupName string,
	requestId string) (Operation, error) {
	patch := VolumePatch{PlacementGroup: &NullableString{placementGroupName}}
	return a.UpdateVolumeBy(ctx, tenantName, tenantSpaceName, volumeName, patch, requestId)
}

func (a *VolumesApiService) UpdateVolumeStorageClassPlacementGroup(
	ctx context.Context,
	tenantName string,
	tenantSpaceName string,
	volumeName string,
	storageClassName string,
	placementGroupName string,
	requestId string) (Operation, error) {
	patch := VolumePatch{StorageClass: &NullableString{storageClassName},
		PlacementGroup: &NullableString{placementGroupName}}
	return a.UpdateVolumeBy(ctx, tenantName, tenantSpaceName, volumeName, patch, requestId)
}

func (a *VolumesApiService) UpdateVolumeProtectionPolicy(
	ctx context.Context,
	tenantName string,
	tenantSpaceName string,
	volumeName string,
	protectionPolicyName string,
	requestId string) (Operation, error) {
	patch := VolumePatch{ProtectionPolicy: &NullableString{protectionPolicyName}}
	return a.UpdateVolumeBy(ctx, tenantName, tenantSpaceName, volumeName, patch, requestId)
}

func (a *VolumesApiService) UpdateVolumeFromSnapshot(
	ctx context.Context,
	tenantName string,
	tenantSpaceName string,
	volumeName string,
	volumeSnapshotLink string,
	requestId string) (Operation, error) {
	patch := VolumePatch{SourceVolumeSnapshotLink: &NullableString{volumeSnapshotLink}}
	return a.UpdateVolumeBy(ctx, tenantName, tenantSpaceName, volumeName, patch, requestId)
}

func (a *VolumesApiService) UpdateVolumeSize(
	ctx context.Context,
	tenantName string,
	tenantSpaceName string,
	volumeName string,
	size int64,
	requestId string) (Operation, error) {
	patch := VolumePatch{Size: &NullableSize{Value: size}}
	return a.UpdateVolumeBy(ctx, tenantName, tenantSpaceName, volumeName, patch, requestId)
}
