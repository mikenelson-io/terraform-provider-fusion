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

type Performance struct {
	Resource *ResourceReference `json:"resource,omitempty"`
	// Reads per second
	ReadsPerSec int64 `json:"reads_per_sec,omitempty"`
	// Read Latency in microseconds
	ReadLatencyUs int64 `json:"read_latency_us,omitempty"`
	// Read Bandwidth in bytes per second
	ReadBandwidth int64 `json:"read_bandwidth,omitempty"`
	// Writes per second
	WritesPerSec int64 `json:"writes_per_sec,omitempty"`
	// Write Latency in microseconds
	WriteLatencyUs int64 `json:"write_latency_us,omitempty"`
	// Write Bandwidth in bytes per second
	WriteBandwidth int64 `json:"write_bandwidth,omitempty"`
}
