/*
Copyright 2022 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

// Code generated DO NOT EDIT.
/*
 * Pure Fusion API
 *
 * Pure Fusion is fully API-driven. Most APIs which change the system (POST, PATCH, DELETE) return an Operation in status \"Pending\" or \"Running\". You can poll (GET) the operation to check its status, waiting for it to change to \"Succeeded\" or \"Failed\".
 *
 * API version: 1.1
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */
package fusion

type StorageClass struct {
	// An immutable, globally unique, system generated identifier.
	Id string `json:"id"`
	// The name of the resource, supplied by the user at creation, and used in the URI path of a resource.
	Name string `json:"name"`
	// The URI of the resource.
	SelfLink string `json:"self_link"`
	// The display name of the resource.
	DisplayName    string             `json:"display_name,omitempty"`
	StorageService *StorageServiceRef `json:"storage_service,omitempty"`
	// The maximum size allowed
	SizeLimit int64 `json:"size_limit"`
	// The maximum IOPS allowed
	IopsLimit int64 `json:"iops_limit"`
	// The maximum bandwidth allowed
	BandwidthLimit int64 `json:"bandwidth_limit"`
}
