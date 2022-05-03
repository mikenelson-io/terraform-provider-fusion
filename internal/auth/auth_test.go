/*
Copyright 2022 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/
package auth_test

import (
	"context"
	"os"
	"testing"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/auth"
)

// Performs a test against production pure1 auth endpoint
func TestProduction(t *testing.T) {
	t.Setenv("PURE1_AUTHENTICATION_ENDPOINT", "")
	_, err := auth.GetPure1SelfSignedAccessTokenGoodForOneHour(
		context.Background(),
		os.Getenv("TEST_PURE1_PROD_ISSUERID"),
		os.Getenv("TEST_PURE1_PROD_PRIVATE_KEY_PATH"),
	)
	if err != nil {
		t.Errorf("main: %s", err)
	}
}

// Performs a test against endpoint specified by PURE1_AUTHENTICATION_ENDPOINT if its set...
func TestStaging(t *testing.T) {
	if os.Getenv("PURE1_AUTHENTICATION_ENDPOINT") == "" {
		t.Skip("PURE1_AUTHENTICATION_ENDPOINT not set")
	}
	_, err := auth.GetPure1SelfSignedAccessTokenGoodForOneHour(
		context.Background(),
		os.Getenv("TEST_PURE1_STAGING_ISSUERID"),
		os.Getenv("TEST_PURE1_STAGING_PRIVATE_KEY_PATH"),
	)
	if err != nil {
		t.Errorf("main: %s", err)
	}
}
