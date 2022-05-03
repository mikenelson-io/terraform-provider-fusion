/*
Copyright 2022 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"fmt"
	"time"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

//
// ResourceProvider is implemented for each resource.
//

type ResourcePatch interface {
}
type ResourcePost interface { // could have: id, name
}
type RequestSpec interface{}

//type InvokeReadMultiAPI func(ctx context.Context, client *hmrest.APIClient) (resource []interface{}, err error)
//type InvokeReadSingleAPI func(ctx context.Context, client *hmrest.APIClient) (resource interface{}, err error)
type InvokeWriteAPI func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (operation *hmrest.Operation, err error)

// This is what you need to implement as the owner of a resource. Use the BaseResourceFunctions to build a schema.
type ResourceProvider interface {
	// PrepareCreate returns a function which will call the Create REST API on this object and return an operation;
	// also returns the post body to pass to that function. Invoke the function using the post body.
	PrepareCreate(ctx context.Context, d *schema.ResourceData) (fn InvokeWriteAPI, post ResourcePost, err error)

	// ReadResource synchronously reads the resource via its REST API.
	ReadResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (err error)

	// PrepareUpdate returns a function which will call the Update REST API on this object and return an operation.
	// Invoke it with each of the patches.
	PrepareUpdate(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (fn InvokeWriteAPI, patches []ResourcePatch, err error)

	// PrepareDelete returns a function which will call the Delete REST API on this object and return an operation. Invoke it.
	PrepareDelete(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (fn InvokeWriteAPI, err error)
}

// Actually, an empty implementation which returns "not implemented" errors. :-)
type BaseResourceProvider struct {
	ResourceKind string
}

func (p *BaseResourceProvider) PrepareCreate(ctx context.Context, d *schema.ResourceData) (fn InvokeWriteAPI, post ResourcePost, err error) {
	return nil, nil, fmt.Errorf("unsupported operation: create %s", p.ResourceKind)
}

func (p *BaseResourceProvider) ReadResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (err error) {
	return fmt.Errorf("unsupported operation: read %s", p.ResourceKind)
}

func (p *BaseResourceProvider) PrepareUpdate(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (fn InvokeWriteAPI, patches []ResourcePatch, err error) {
	return nil, nil, fmt.Errorf("unsupported operation: update %s", p.ResourceKind)
}

func (p *BaseResourceProvider) PrepareDelete(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (fn InvokeWriteAPI, err error) {
	return nil, fmt.Errorf("unsupported operation: delete %s", p.ResourceKind)
}

//
// Resource functions internally implement the interface defined by Terraform.
//

// Implements interface to Terraform: resource-CRUD
type BaseResourceFunctions struct {
	*schema.Resource
	ResourceKind string // We're for volume, tenant space, storage class, etc. More likely to come.
	Provider     ResourceProvider
}

func NewBaseResourceFunctions(resourceKind string, provider ResourceProvider) *BaseResourceFunctions {
	result := &BaseResourceFunctions{&schema.Resource{}, resourceKind, provider}
	result.Resource.CreateContext = result.resourceCreate
	result.Resource.ReadContext = result.resourceRead
	result.Resource.UpdateContext = result.resourceUpdate
	result.Resource.DeleteContext = result.resourceDelete
	result.Resource.Importer = &schema.ResourceImporter{
		StateContext: result.resourceImport,
	}
	return result
}

// resourceCreate creates a HM Resource, generically, relying on ResourceProvider.ComputePost
func (f *BaseResourceFunctions) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client, ctx := f.resourceBoilerplate(ctx, "Create", d, m)

	// invoke the interface
	callAPI, body, err := f.Provider.PrepareCreate(ctx, d)
	if err != nil {
		tflog.Error(ctx, "in preparing post", "error_message", err)
		return diag.FromErr(err)
	}
	tflog.Debug(ctx, "Post", "body", body)
	op, err := callAPI(ctx, client, body)
	if err != nil {
		utilities.TraceError(ctx, err)
		return processClientError(ctx, "create", err)
	}

	// Wait on Operation
	succeeded, err := WaitOnOperation(ctx, op, client) // updates op with latest
	if err != nil {
		utilities.TraceError(ctx, err)
		return processClientError(ctx, "get wait status", err)
	}

	if !succeeded {
		tflog.Error(ctx, "REST create failed", "error_message", op.Error_.Message,
			"PureCode", op.Error_.PureCode, "HttpCode", op.Error_.HttpCode)
		return diag.Errorf(op.Error_.Message)
	}

	// succeeded!
	tflog.Debug(ctx, "created successfully", "operation_result", op.Result)
	d.SetId(op.Result.Resource.Id)
	return f.resourceRead(ctx, d, m)
}

func (f *BaseResourceFunctions) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client, _ := f.resourceBoilerplate(ctx, "Read", d, m)
	err := f.Provider.ReadResource(ctx, client, d)
	return processClientError(ctx, "read", err)
}

func (f *BaseResourceFunctions) resourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client, ctx := f.resourceBoilerplate(ctx, "Update", d, m)

	callAPI, patches, err := f.Provider.PrepareUpdate(ctx, client, d)
	if err != nil {
		return diag.FromErr(err)
	}

	// Start operations for each update
	for i, p := range patches {
		ctx := tflog.With(ctx, "patch_idx", i)
		tflog.Debug(ctx, "Start Operation to apply update", "patch_num", i, "patch", p)
		op, err := callAPI(ctx, client, p)
		traceOperation(ctx, op, "resourceUpdate_patch")
		if err != nil {
			utilities.TraceError(ctx, err)
			return processClientError(ctx, "update", err)
		}

		// Right now we are forcing all the operations to complete serially
		// because there are certain patch operations that need to happen
		// in order.  Later on we can get more clever and try to come up
		// with patch groups that can be done in parallel together
		succeeded, err := WaitOnOperation(ctx, op, client)
		if err != nil {
			utilities.TraceError(ctx, err)
			return processClientError(ctx, "get wait status", err)
		}
		if !succeeded {
			return diag.Errorf("Operation failed Message:%s ID:%s", op.Error_.Message, op.Id)
		}
	}

	return f.resourceRead(ctx, d, m)
}

func (f *BaseResourceFunctions) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client, ctx := f.resourceBoilerplate(ctx, "Delete", d, m)

	callAPI, err := f.Provider.PrepareDelete(ctx, client, d)
	if err != nil {
		tflog.Error(ctx, "in compute delete or volume: REST DELETE volume failed", "error_message", err)
		return diag.FromErr(err)
	}

	op, err := callAPI(ctx, client, nil) // no body for delete
	if err != nil {
		utilities.TraceError(ctx, err)
		return processClientError(ctx, "delete", err)
	}

	succeeded, err := WaitOnOperation(ctx, op, client)
	if err != nil {
		utilities.TraceError(ctx, err)
		return processClientError(ctx, "get wait status", err)
	}

	if !succeeded {
		tflog.Error(ctx, "REST delete failed", "error_message", op.Error_.Message,
			"PureCode", op.Error_.PureCode, "HttpCode", op.Error_.HttpCode)
		return diag.Errorf(op.Error_.Message)
	}

	return nil
}

// resourcePureVolumeImport imports a volume into Terraform.
// TODO: when is this used?
func (f *BaseResourceFunctions) resourceImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	client, ctx := f.resourceBoilerplate(ctx, "Import", d, m)
	err := f.Provider.ReadResource(ctx, client, d)
	if err != nil {
		tflog.Error(ctx, "in reading resource", "error_message", err)
		return nil, err
	}
	return []*schema.ResourceData{d}, nil // TODO: We return one item. Looks like this API can do lists.
}

// A function used at the top of each CRUD function to grab stuff we need. Belongs in resource_functions.
func (f *BaseResourceFunctions) resourceBoilerplate(ctx context.Context, action string, d *schema.ResourceData, m interface{}) (*hmrest.APIClient, context.Context) {
	ctx = tflog.With(ctx, "resource_kind", f.ResourceKind)
	tflog.Debug(ctx, "resource", "action", action, "state", d.State())

	client := m.(*hmrest.APIClient)

	return client, ctx
}

// Wait on an operation until its status reaches Succeeded (or Completed) or Failed.
// Return succeeded = true if status reaches Succeeded (or Completed), Failed if status reached Failed, and err otherwise.
// On return,
//  op will be up to date with the most recent GET of the operation, EVEN when we're returning an error.
//	if err != nil, then we have an error. Ignore succeeded (it will be false, but it doesn't mean the operation failed.)
//  If err == nil, then check succeeded. It is true iff (op.Status == "Succeeded" || op.Status == "Completed") && op.Status != "Failed"
func WaitOnOperation(ctx context.Context, op *hmrest.Operation, client *hmrest.APIClient) (succeeded bool, err error) {
	traceOperation(ctx, op, "waitOnOperation")
	if op.Status == "" && op.Id == "" && op.RetryIn == 0 {
		tflog.Error(ctx, "waitOnOperation for null op")
		return false, fmt.Errorf("waitOnOperation for null op")
	}
	for op.Status != "Succeeded" && op.Status != "Completed" && op.Status != "Failed" {
		tflog.Debug(ctx, "Waiting for operation", "op_type", op.RequestType, "op_id", op.Id, "op_status", op.Status, "op_retry_in", op.RetryIn)
		time.Sleep(time.Duration(op.RetryIn) * time.Millisecond)
		opNew, _, err := client.OperationsApi.GetOperation(ctx, op.Id, nil)
		traceOperation(ctx, &opNew, "waitOnOperation")
		utilities.TraceError(ctx, err)
		if err != nil {
			return false, err
		}
		*op = opNew
	}

	// Now op.Status must be Succeeded or Completed or Failed.
	if op.Status == "Failed" {
		tflog.Error(ctx, "waitOnOperation FAILED with Error", "error_message", op.Error_, "operation", op)
		return false, nil
	}

	// op.Status must be Succeeded or Completed.
	traceOperation(ctx, op, "waitOnOperation Succeeded")
	return true, nil
}

func traceOperation(ctx context.Context, op *hmrest.Operation, userMessage string) {
	tflog.Trace(ctx, "trace_operation",
		"user_message", userMessage,
		"op_id", op.Id,
		"op_request_type", op.RequestType,
		"op_error_dump", fmt.Sprintf("%#v", op.Error_),
		"op_status", op.Status,
		"op_retry_in", op.RetryIn,
	)
}

//
// These RD wrappers don't do much yet, just ensure we get good logging on errors.
// But for future's sake ... good fences make good neighbors.
//
func rdString(ctx context.Context, d *schema.ResourceData, key string) string {
	value := d.Get(key)
	if value == nil {
		return ""
	}
	s, ok := value.(string) // If not set, provides empty string.
	if !ok {
		tflog.Error(ctx, "Got unexpected type value", "key", key, "type", fmt.Sprintf("%T", value), "value", value)
		return value.(string) // Force the runtime error if not ok.
	}
	return s
}
func rdStringDefault(ctx context.Context, d *schema.ResourceData, key string, defaultValue string) string {
	value := rdString(ctx, d, key)
	if value == "" {
		return defaultValue
	}
	return value
}
func rdInt(d *schema.ResourceData, key string) int { return d.Get(key).(int) }

func processClientError(ctx context.Context, op string, err error) diag.Diagnostics {
	modelError, convError := hmrest.ToModelError(err)
	if convError != nil {
		tflog.Warn(ctx, "Error while converting error", "error_message", convError.Error(),
			"unconverted error", err, "operation", op)
		return diag.FromErr(err)
	} else {
		tflog.Error(ctx, "REST ", "operation", op, "error_message", modelError.Message)
		return diag.Errorf(modelError.Message)
	}
}
