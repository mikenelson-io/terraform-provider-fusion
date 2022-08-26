package fusion

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

func TestAccVolume_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Dont run with units tests because it will try to create the context")
	}

	ctx := setupTestCtx(t)

	// Setup resources we need
	hmClient, err := NewHMClient(ctx, testURL, testIssuer, testPrivKey)
	utilities.TraceError(ctx, err)

	ts := testFusionResource{RName: "ts", Name: acctest.RandomWithPrefix("ts-volTest")}
	pg0 := testFusionResource{RName: "pg0", Name: acctest.RandomWithPrefix("pg0-volTest")}
	pg1 := testFusionResource{RName: "pg1", Name: acctest.RandomWithPrefix("pg1-volTest")}
	host0 := testFusionResource{RName: "host0", Name: acctest.RandomWithPrefix("host0-volTest")}
	host1 := testFusionResource{RName: "host1", Name: acctest.RandomWithPrefix("host1-volTest")}
	host2 := testFusionResource{RName: "host2", Name: acctest.RandomWithPrefix("host2-volTest")}
	storageService0Name := acctest.RandomWithPrefix("ss0-volTest")
	storageService1Name := acctest.RandomWithPrefix("ss1-volTest")
	protectionPolicy0Name := acctest.RandomWithPrefix("pp0-volTest")
	protectionPolicy1Name := acctest.RandomWithPrefix("pp1-volTest")
	storageClass0Name := acctest.RandomWithPrefix("sc0-volTest")
	storageClass1Name := acctest.RandomWithPrefix("sc1-volTest")

	// Initial state
	volState0 := testVolume{
		RName:                "test_volume",
		Name:                 acctest.RandomWithPrefix("test_vol"),
		DisplayName:          "initial display name",
		TenantSpace:          ts,
		ProtectionPolicyName: protectionPolicy0Name,
		StorageClassName:     storageClass0Name,
		PlacementGroup:       pg0,
		Size:                 1 << 20,
	}

	// Change everything
	volState1 := volState0
	volState1.DisplayName = "changed display name"
	volState1.PlacementGroup = pg1
	volState1.StorageClassName = storageClass1Name
	volState1.Size += 1 << 20
	volState1.Hosts = []testFusionResource{host0}

	// Remove and add hosts at the same time, also change protection policy
	volState2 := volState1
	volState2.Hosts = []testFusionResource{host1, host2}
	volState2.ProtectionPolicyName = protectionPolicy1Name

	// Remove a host, and change some other things back
	volState3 := volState2
	volState3.DisplayName = "changed display name again"
	volState3.Hosts = []testFusionResource{host1}
	volState3.PlacementGroup = pg0
	volState3.StorageClassName = storageClass0Name

	commonConfig := "" +
		testTenantSpaceConfig(ts.RName, "ts display name", ts.Name, testAccTenant) +
		testPGConfig("", pg0.RName, pg0.Name, "pg display name", region_name, availability_zone_name, storageService0Name, true) +
		testPGConfig("", pg1.RName, pg1.Name, "pg display name", region_name, availability_zone_name, storageService1Name, true) +
		testHostAccessPolicyConfig(host0.RName, host0.Name, "host display name", randIQN(), "linux") +
		testHostAccessPolicyConfig(host1.RName, host1.Name, "host display name", randIQN(), "linux") +
		testHostAccessPolicyConfig(host2.RName, host2.Name, "host display name", randIQN(), "linux") +
		""

	testVolumeStep := func(vol testVolume) resource.TestStep {
		step := resource.TestStep{}

		step.Config = commonConfig + testVolumeConfig(vol)

		r := "fusion_volume." + vol.RName

		step.Check = resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(r, "name", vol.Name),
			resource.TestCheckResourceAttr(r, "display_name", vol.DisplayName),
			resource.TestCheckResourceAttr(r, "tenant_name", testAccTenant),
			resource.TestCheckResourceAttr(r, "tenant_space_name", vol.TenantSpace.Name),
			resource.TestCheckResourceAttr(r, "protection_policy_name", vol.ProtectionPolicyName),
			resource.TestCheckResourceAttr(r, "storage_class_name", vol.StorageClassName),
			resource.TestCheckResourceAttr(r, "placement_group_name", vol.PlacementGroup.Name),
			resource.TestCheckResourceAttr(r, "host_names.#", fmt.Sprintf("%d", len(vol.Hosts))),
		)
		for _, host := range vol.Hosts {
			step.Check = resource.ComposeTestCheckFunc(step.Check,
				resource.TestCheckTypeSetElemAttr(r, "host_names.*", host.Name),
			)
		}

		step.Check = resource.ComposeTestCheckFunc(step.Check, testVolumeExists(r, t))

		return step
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			doOp := func(userMessage string) func(op hmrest.Operation, _ *http.Response, err error) {
				return func(op hmrest.Operation, _ *http.Response, err error) {
					utilities.TraceError(ctx, err)
					if err != nil {
						t.Errorf("%s: %s", userMessage, err)
					}
					succeeded, err := utilities.WaitOnOperation(ctx, &op, hmClient)
					if !succeeded || err != nil {
						t.Errorf("operation failure %s succeeded:%v error:%v", userMessage, succeeded, err)
					}
				}
			}

			doOp("protectionPolicy0")(hmClient.ProtectionPoliciesApi.CreateProtectionPolicy(ctx, hmrest.ProtectionPolicyPost{
				Name: protectionPolicy0Name,
				Objectives: []hmrest.OneOfProtectionPolicyPostObjectivesItems{
					hmrest.Rpo{Type_: "RPO", Rpo: "PT6H"},
					hmrest.Retention{Type_: "Retention", After: "PT24H"},
				},
			}, nil))

			doOp("protectionPolicy1")(hmClient.ProtectionPoliciesApi.CreateProtectionPolicy(ctx, hmrest.ProtectionPolicyPost{
				Name: protectionPolicy1Name,
				Objectives: []hmrest.OneOfProtectionPolicyPostObjectivesItems{
					hmrest.Rpo{Type_: "RPO", Rpo: "PT6H"},
					hmrest.Retention{Type_: "Retention", After: "PT24H"},
				},
			}, nil))

			doOp("storageService0")(hmClient.StorageServicesApi.CreateStorageService(ctx, hmrest.StorageServicePost{
				Name:          storageService0Name,
				HardwareTypes: []string{"flash-array-x"},
			}, nil))

			doOp("storageService1")(hmClient.StorageServicesApi.CreateStorageService(ctx, hmrest.StorageServicePost{
				Name:          storageService1Name,
				HardwareTypes: []string{"flash-array-c"},
			}, nil))

			doOp("storageClass0")(hmClient.StorageClassesApi.CreateStorageClass(ctx, hmrest.StorageClassPost{
				Name:           storageClass0Name,
				SizeLimit:      1 << 22,
				BandwidthLimit: 1e9,
				IopsLimit:      100,
			}, storageService0Name, nil))

			doOp("storageClass1")(hmClient.StorageClassesApi.CreateStorageClass(ctx, hmrest.StorageClassPost{
				Name:           storageClass1Name,
				SizeLimit:      1 << 22,
				BandwidthLimit: 1e9,
				IopsLimit:      100,
			}, storageService1Name, nil))
		},
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckVolumeDestroy,
		Steps: []resource.TestStep{
			testVolumeStep(volState0),
			testVolumeStep(volState1),
			testVolumeStep(volState2),
			testVolumeStep(volState3),
		},
	})
}

// Verify resource with a direct hmrest call
func testVolumeExists(rName string, t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		volume, ok := s.RootModule().Resources[rName]
		if !ok {
			return fmt.Errorf("resource not found: %s", rName)
		}
		if volume.Type != "fusion_volume" {
			return fmt.Errorf("expected type: fusion_volume. Found: %s", volume.Type)
		}
		attrs := volume.Primary.Attributes

		directVolume, _, err := testAccProvider.Meta().(*hmrest.APIClient).VolumesApi.GetVolume(context.Background(), testAccTenant, attrs["tenant_space_name"], attrs["name"], nil)
		if err != nil {
			return fmt.Errorf("go client retutrned error while searching for %s. Error: %s", attrs["name"], err)
		}
		tfHostNameCount, _ := strconv.Atoi(attrs["host_names.#"])

		directHosts := []string{}
		for _, directHost := range directVolume.HostAccessPolicies {
			directHosts = append(directHosts, directHost.Name)
		}
		sort.Slice(directHosts, func(i, j int) bool { return strings.Compare(directHosts[i], directHosts[j]) < 0 })
		tfHosts := []string{}
		for i := 0; i < tfHostNameCount; i++ {
			tfHosts = append(tfHosts, attrs[fmt.Sprintf("host_names.%d", i)])
		}
		sort.Slice(tfHosts, func(i, j int) bool { return strings.Compare(tfHosts[i], tfHosts[j]) < 0 })

		failed := false

		checkAttr := func(direct, attrName string) {
			if direct != attrs[attrName] {
				t.Errorf("mismatch attr:%s direct:%s tf:%s", attrName, direct, attrs[attrName])
				failed = true
			}
		}

		checkAttr(directVolume.Name, "name")
		checkAttr(directVolume.DisplayName, "display_name")
		checkAttr(directVolume.Tenant.Name, "tenant_name")
		checkAttr(directVolume.TenantSpace.Name, "tenant_space_name")
		checkAttr(directVolume.ProtectionPolicy.Name, "protection_policy_name")
		checkAttr(directVolume.StorageClass.Name, "storage_class_name")
		checkAttr(directVolume.PlacementGroup.Name, "placement_group_name")

		if !reflect.DeepEqual(directHosts, tfHosts) {
			t.Errorf("hosts mismatch")
			for _, h := range directHosts {
				t.Logf("direct host: %s", h)
			}
			for _, h := range tfHosts {
				t.Logf("tf host: %s", h)
			}
			failed = true
		}

		if failed {
			return fmt.Errorf("direct tf mismatch")
		}

		return nil
	}
}

type testVolume struct {
	RName                string
	Name                 string
	DisplayName          string
	ProtectionPolicyName string
	TenantSpace          testFusionResource
	StorageClassName     string
	PlacementGroup       testFusionResource
	Size                 int
	Hosts                []testFusionResource
}

type testFusionResource struct {
	RName string
	Name  string
}

func testVolumeConfig(vol testVolume) string {
	hapList := ""
	for hostI, host := range vol.Hosts {
		if hostI != 0 {
			hapList += ", "
		}
		hapList += fmt.Sprintf(`fusion_host_access_policy.%s.name`, host.RName)
	}

	return fmt.Sprintf(`
resource "fusion_volume" "%[1]s" {
		name          = "%[2]s"
		display_name  = "%[3]s"
		protection_policy_name = "%[4]s"
		tenant_name        = "%[5]s"
		tenant_space_name  = fusion_tenant_space.%[6]s.name
		storage_class_name = "%[7]s"
		size          = %[8]d
		host_names = [%[9]s]
		placement_group_name = fusion_placement_group.%[10]s.name
}`, vol.RName, vol.Name, vol.DisplayName, vol.ProtectionPolicyName,
		testAccTenant, vol.TenantSpace.RName, vol.StorageClassName, vol.Size, hapList, vol.PlacementGroup.RName)
}

func testCheckVolumeDestroy(s *terraform.State) error {

	client := testAccProvider.Meta().(*hmrest.APIClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "fusion_volume" {
			continue
		}
		attrs := rs.Primary.Attributes
		volumeName := attrs["name"]
		tenantName := attrs["tenant_name"]
		tenantSpaceName := attrs["tenan_space_name"]

		_, resp, err := client.VolumesApi.GetVolume(context.Background(), tenantName, tenantSpaceName, volumeName, nil)
		if err != nil && resp.StatusCode == http.StatusNotFound {
			continue
		} else {
			return fmt.Errorf("volume may still exist. Expected response code 404, got code %d", resp.StatusCode)
		}
	}
	return nil
}
