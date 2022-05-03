/*
Copyright 2022 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// Creates and destroys
func TestAccTenantSpace_basic(t *testing.T) {
	rNameConfig := fmt.Sprintf("tenant_space_test_%d", acctest.RandIntRange(0, 1000))
	rName := "fusion_tenant_space." + rNameConfig
	displayName1 := acctest.RandomWithPrefix("tenant-space-display-name")
	tenantSpaceName := acctest.RandomWithPrefix("test_ts")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckTenantSpaceDestroy,
		Steps: []resource.TestStep{
			// Create Tenant and validate it's fields
			{
				Config: testTenantSpaceConfig(rNameConfig, displayName1, tenantSpaceName, testAccTenant),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "name", tenantSpaceName),
					resource.TestCheckResourceAttr(rName, "display_name", displayName1),
					resource.TestCheckResourceAttr(rName, "tenant_name", testAccTenant),
					testTenantSpaceExists(rName),
				),
			},
		},
	})
}

// Updates display name
func TestAccTenantSpace_update(t *testing.T) {
	rNameConfig := fmt.Sprintf("tenant_space_test_%d", acctest.RandIntRange(0, 1000))
	rName := "fusion_tenant_space." + rNameConfig
	displayName1 := acctest.RandomWithPrefix("tenant-space-display-name")
	displayName2 := acctest.RandomWithPrefix("tenant-space-display-name2")
	buff := make([]byte, 257) // 256 is max length
	rand.Read(buff)
	displayNameTooBig := base64.StdEncoding.EncodeToString(buff)
	tenantSpaceName := acctest.RandomWithPrefix("test_ts")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckTenantSpaceDestroy,
		Steps: []resource.TestStep{
			// Create Tenant and validate it's fields
			{
				Config: testTenantSpaceConfig(rNameConfig, displayName1, tenantSpaceName, testAccTenant),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "name", tenantSpaceName),
					resource.TestCheckResourceAttr(rName, "display_name", displayName1),
					resource.TestCheckResourceAttr(rName, "tenant_name", testAccTenant),
					testTenantSpaceExists(rName),
				),
			},
			// Update the display name, assert that the tf resource got updated, then assert the backend shows the same
			{
				Config: testTenantSpaceConfig(rNameConfig, displayName2, tenantSpaceName, testAccTenant),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "display_name", displayName2),
					testTenantSpaceExists(rName),
				),
			},
			// Bad display name values
			{
				Config: testTenantSpaceConfig(rNameConfig, displayNameTooBig, tenantSpaceName, testAccTenant),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "display_name", displayName2),
					testTenantSpaceExists(rName),
				),
				ExpectError: regexp.MustCompile("display_name must be at most 256 characters"),
			},

			// TODO: HM-2419
			//Can't update certain values
			{
				Config:      testTenantSpaceConfig(rNameConfig, displayName1, "immutable", testAccTenant),
				ExpectError: regexp.MustCompile("attempting to update an immutable field"),
			},
			{
				Config:      testTenantSpaceConfig(rNameConfig, displayName1, tenantSpaceName, "immutable"),
				ExpectError: regexp.MustCompile("attempting to update an immutable field"),
			},
			// When the test tries to destroy the resources at the end, it does not do a refresh first,
			// and therefore the destroy will fail if the state is invalid. Because of this, we need to manually
			// return the state to a valid config. Note that the "terraform destroy" command does do
			// a refresh first, so this issue only applies to acceptance tests.
			{
				Config: testTenantSpaceConfig(rNameConfig, displayName1, tenantSpaceName, testAccTenant),
			},
		},
	})
}

func TestAccTenantSpace_attributes(t *testing.T) {
	rNameConfig := fmt.Sprintf("tenant_space_test_%d", acctest.RandIntRange(0, 1000))
	rName := "fusion_tenant_space." + rNameConfig
	displayName1 := acctest.RandomWithPrefix("tenant-space-display-name")
	tenantSpaceName := acctest.RandomWithPrefix("tenant-space-name")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckTenantSpaceDestroy,
		Steps: []resource.TestStep{
			// Missing required fields
			{
				Config:      testTenantSpaceConfig(rNameConfig, displayName1, "", testAccTenant),
				ExpectError: regexp.MustCompile("Error: name must be specified"),
			},
			{
				Config:      testTenantSpaceConfig(rNameConfig, displayName1, tenantSpaceName, ""),
				ExpectError: regexp.MustCompile("Error: must specify Tenant name"),
			},
			{
				Config:      testTenantSpaceConfig(rNameConfig, displayName1, "bad name here", testAccTenant),
				ExpectError: regexp.MustCompile("name must use alphanumeric characters"),
			},
			//{
			//	Config:      testTenantSpaceConfig(rNameConfig, displayName1, "", ""),
			//	ExpectError: regexp.MustCompile("Error: Name & Tenant Space must be specified"), // TODO: HM-2420 this should be both!
			//},
			// Create without display_name then update
			{
				Config: testTenantSpaceConfigNoDisplayName(rNameConfig, tenantSpaceName, testAccTenant),
				Check: resource.ComposeTestCheckFunc(
					testTenantSpaceExists(rName),
				),
			},
			{
				Config: testTenantSpaceConfig(rNameConfig, displayName1, tenantSpaceName, testAccTenant),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "display_name", displayName1),
					testTenantSpaceExists(rName),
				),
			},
		},
	})
}

func TestAccTenantSpace_multiple(t *testing.T) {
	rNameConfig := fmt.Sprintf("tenant_space_test_%d", acctest.RandIntRange(0, 1000))
	rName := "fusion_tenant_space." + rNameConfig
	displayName1 := acctest.RandomWithPrefix("tenant-space-display-name")
	tenantSpaceName := acctest.RandomWithPrefix("tenant-space-name")

	rNameConfig2 := fmt.Sprintf("tenant_space_test_%d", acctest.RandIntRange(0, 1000))
	rName2 := "fusion_tenant_space." + rNameConfig
	displayName2 := acctest.RandomWithPrefix("tenant-space-display-name")
	tenantSpaceName2 := acctest.RandomWithPrefix("tenant-space-name")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckTenantSpaceDestroy,
		Steps: []resource.TestStep{
			// Sanity check two can be created at once
			{
				Config: testTenantSpaceConfig(rNameConfig, displayName1, tenantSpaceName, testAccTenant) + "\n" +
					testTenantSpaceConfig(rNameConfig2, displayName2, tenantSpaceName2, testAccTenant),
				Check: resource.ComposeTestCheckFunc(
					testTenantSpaceExists(rName),
					testTenantSpaceExists(rName2),
				),
			},
			// Create two with same name
			{
				Config: testTenantSpaceConfig(rNameConfig, displayName1, tenantSpaceName, testAccTenant) + "\n" +
					testTenantSpaceConfig(rNameConfig2, displayName2, tenantSpaceName2, testAccTenant) + "\n" +
					testTenantSpaceConfig("conflictRN", "conflictDN", tenantSpaceName, testAccTenant),
				ExpectError: regexp.MustCompile("already exists"),
			},
		},
	})
}

func testTenantSpaceExists(rName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		tfTenantSpace, ok := s.RootModule().Resources[rName]
		if !ok {
			return fmt.Errorf("resource not found: %s", rName)
		}
		if tfTenantSpace.Type != "fusion_tenant_space" {
			return fmt.Errorf("expected type: fusion_tenant_space. Found: %s", tfTenantSpace.Type)
		}
		attrs := tfTenantSpace.Primary.Attributes

		goclientTenantSpace, _, err := testAccProvider.Meta().(*hmrest.APIClient).TenantSpacesApi.GetTenantSpace(context.Background(), testAccTenant, attrs["name"], nil)
		if err != nil {
			return fmt.Errorf("go client retutrned error while searching for %s. Error: %s", attrs["name"], err)
		}
		if strings.Compare(goclientTenantSpace.Name, attrs["name"]) != 0 ||
			strings.Compare(goclientTenantSpace.DisplayName, attrs["display_name"]) != 0 ||
			strings.Compare(goclientTenantSpace.Tenant.Name, attrs["tenant_name"]) != 0 {
			return fmt.Errorf("terraform tenant space doesnt match goclients tenant space")
		}
		return nil
	}
}

func testCheckTenantSpaceDestroy(s *terraform.State) error {

	client := testAccProvider.Meta().(*hmrest.APIClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "fusion_tenant_space" {
			continue
		}
		attrs := rs.Primary.Attributes

		tenantName := attrs["tenant_name"]
		tenantSpaceName := attrs["name"]

		_, resp, err := client.TenantSpacesApi.GetTenantSpace(context.Background(), tenantName, tenantSpaceName, nil)
		if err != nil && resp.StatusCode == http.StatusNotFound {
			continue
		} else {
			return fmt.Errorf("tenant space may still exist. Expected response code 404, got code %d", resp.StatusCode)
		}
	}
	return nil
}

func testTenantSpaceConfig(rName string, displayName string, tenantSpaceName string, tenantName string) string {
	return fmt.Sprintf(`
	resource "fusion_tenant_space" "%[1]s" {
		name          = "%[2]s"
		display_name  = "%[3]s"
		tenant_name   = "%[4]s"
	}
	`, rName, tenantSpaceName, displayName, tenantName)
}

func testTenantSpaceConfigNoDisplayName(rName string, tenantSpaceName string, tenantName string) string {
	return fmt.Sprintf(`
	resource "fusion_tenant_space" "%[1]s" {
		name          = "%[2]s"
		tenant_name        = "%[3]s"
	}
	`, rName, tenantSpaceName, tenantName)
}
