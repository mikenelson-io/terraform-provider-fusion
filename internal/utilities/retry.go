/*
Copyright 2022 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package utilities

import (
	"context"
	"reflect"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func reflectFieldAsInterface(i interface{}, name string) interface{} {
	obj := reflect.ValueOf(i)

	// This is a bit overly careful, as we really don't want to cause any panics here
	if obj.Kind() == reflect.Ptr || obj.Kind() == reflect.Interface {
		obj = obj.Elem()
	}

	if obj.Kind() != reflect.Struct {
		return nil
	}

	fieldReflected := obj.FieldByName(name)

	if !fieldReflected.IsValid() {
		return nil
	}

	return fieldReflected.Interface()
}

// Runs f in a loop if it has failed, until attempts reaches attemptLimit
// retryTime is a duration to cooldown between attempts
// retryContext is a string that is included in log output
// backoffFactor is a float that determines how much to increase retryTime
// if backoffFactor is 0.0, then there is no backoff increase
// if backoffFactor is 1.0, then there is a 100% increase, so it doubles each time
// if backoffFactor is 0.5, then there is a 50% increase, ie maybe 100, then 150, then 225...
// f returns a bool (stop) if that is true, we won't retry anymore
func Retry(ctx context.Context, retryTime time.Duration, backoffFactor float32, attemptLimit int, retryContext string, f func() (stop bool, err error)) error {
	var err error
	for attemptI := 0; attemptI < attemptLimit; attemptI++ {
		var stop bool
		stop, err = f()
		if err == nil {
			return nil
		}
		tflog.Warn(ctx, "retry_attempt",
			"context", retryContext,
			"attempt_done_count", attemptI+1,
			"cooldown_ms", retryTime.Milliseconds(),
			"error_message", err.Error(),
		)
		tflog.Trace(ctx, "retry_attempt",
			"attempt_limit", attemptLimit,
		)
		TraceError(ctx, err)
		if stop {
			return err
		}
		time.Sleep(retryTime)
		retryTime += time.Duration(int(float32(retryTime.Milliseconds())*backoffFactor)) * time.Millisecond
	}
	tflog.Error(ctx, "retry_attempt",
		"context", retryContext,
		"attempt_limit", attemptLimit,
		"error_message", err.Error())
	return err
}
