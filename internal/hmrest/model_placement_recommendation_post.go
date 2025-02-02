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

// (Provider) Request a Placement Recommendation report. If PlacementEngine is set to \"pure1meta\", in addition to Placement Recommendations, load and capacity projections will also be included in the report.
type PlacementRecommendationPost struct {
	// The name of the resource, supplied by the user at creation, and used in the URI path of a resource.
	Name string `json:"name"`
	// The display name of the resource.
	DisplayName string `json:"display_name,omitempty"`
	// Deprecated. Use placement_group instead. The link to the placement group that we would like to generate a placement recommendation report on
	PlacementGroupLink string `json:"placement_group_link,omitempty"`
	// Placement Group you would like to generate a placement recommendation report on. For placement of new placement group, leave this blank, and instead fill in simulated_placement
	PlacementGroup string `json:"placement_group,omitempty"`
	// Tenant that Placement Group belongs to. For placement of new placement group, enter Tenant where the Placement Group would have been created in
	Tenant string `json:"tenant"`
	// Tenant Space that Placement Group belongs to. For placement of new placement group, enter TenantSpace where Placement Group would have been created in
	TenantSpace        string                  `json:"tenant_space"`
	PlacementEngine    *PlacementEngine        `json:"placement_engine,omitempty"`
	SimulatedPlacement *SimulatedPlacementPost `json:"simulated_placement,omitempty"`
	// Optional argument. If provided, specify a list of array names to constraint the list of arrays under consideration for placement recommendations
	TargetArrays []string `json:"target_arrays,omitempty"`
}
