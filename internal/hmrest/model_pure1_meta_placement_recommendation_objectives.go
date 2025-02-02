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

type Pure1MetaPlacementRecommendationObjectives struct {
	AvgPerfUsage float64 `json:"avg_perf_usage,omitempty"`
	AvgCapUsage  float64 `json:"avg_cap_usage,omitempty"`
	VarPerfUsage float64 `json:"var_perf_usage,omitempty"`
	VarCapUsage  float64 `json:"var_cap_usage,omitempty"`
	MaxPerfUsage float64 `json:"max_perf_usage,omitempty"`
	MaxCapUsage  float64 `json:"max_cap_usage,omitempty"`
}
