/*
Copyright 2022 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/
package utilities_test

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-log/tfsdklog"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
)

var ptrLogMatcher = regexp.MustCompile(`\(0x[0-9a-f]+\)`)

func checkTFLog(t *testing.T, expectedString string, testFunc func(context.Context)) {
	ctx := context.TODO()
	tmpFile, err := os.CreateTemp("/tmp", "mockLog")
	if err != nil {
		t.FailNow()
	}
	defer tmpFile.Close()

	t.Setenv("TF_LOG", "TRACE")
	t.Setenv("TF_LOG_PATH", tmpFile.Name())
	t.Setenv("TF_ACC_LOG_PATH", "")
	t.Setenv("TF_LOG_PATH_MASK", "")

	ctx = tfsdklog.RegisterTestSink(ctx, t)
	os.Remove(tmpFile.Name())
	ctx = tfsdklog.NewRootProviderLogger(ctx)

	testFunc(ctx)

	tmpFile.Seek(0, 0)
	lineScanner := bufio.NewScanner(tmpFile)
	gotString := ""
	for lineScanner.Scan() {
		// Strip date, add to string buffer
		line := lineScanner.Text()[strings.Index(lineScanner.Text(), " ")+1:]
		// Replace all pointers to constant value
		line = ptrLogMatcher.ReplaceAllString(line, "(PTR)")
		gotString += line + "\n"
	}
	if lineScanner.Err() != nil {
		t.Fatalf("Scan failed %v", lineScanner.Err())
	}
	gotString = strings.TrimSpace(gotString)
	expectedString = strings.TrimSpace(expectedString)
	if gotString != expectedString {
		t.Errorf("Strings do not match\nExpected:\n%s\nGot:\n%s\n", expectedString, gotString)
	}
}

func TestTraceErrorBasic(t *testing.T) {
	checkTFLog(t, `
[TRACE] provider: trace_error: error_message=test_error error_type="*errors.errorString" error_dump="&errors.errorString{s:\"test_error\"}" error_package_path=""
`, func(ctx context.Context) {
		utilities.TraceError(ctx, fmt.Errorf("test_error"))
	})
}

type testErrorWithResponse struct {
	Response *http.Response
}

func (e *testErrorWithResponse) Error() string {
	return "testErrorWithResponse"
}

func TestTraceErrorResponseHttpNull(t *testing.T) {
	checkTFLog(t, `
[TRACE] provider: trace_error: error_message="test_error: testErrorWithResponse" error_type="*fmt.wrapError" error_dump="&fmt.wrapError{msg:\"test_error: testErrorWithResponse\", err:(*utilities_test.testErrorWithResponse)(PTR)}" error_package_path=""
[TRACE] provider: trace_error: error_message=testErrorWithResponse error_type="*utilities_test.testErrorWithResponse" error_dump="&utilities_test.testErrorWithResponse{Response:(*http.Response)(nil)}" error_package_path=""
[TRACE] provider: trace_error: response=<nil>
`, func(ctx context.Context) {
		utilities.TraceError(ctx, fmt.Errorf("test_error: %w", &testErrorWithResponse{}))
	})
}

func TestTraceErrorResponseHttpNotNil(t *testing.T) {
	checkTFLog(t, `
[TRACE] provider: trace_error: error_message="test_error: testErrorWithResponse" error_type="*fmt.wrapError" error_dump="&fmt.wrapError{msg:\"test_error: testErrorWithResponse\", err:(*utilities_test.testErrorWithResponse)(PTR)}" error_package_path=""
[TRACE] provider: trace_error: error_message=testErrorWithResponse error_type="*utilities_test.testErrorWithResponse" error_dump="&utilities_test.testErrorWithResponse{Response:(*http.Response)(PTR)}" error_package_path=""
[TRACE] provider: trace_error: response_status_code=420 response_status="some status"
`, func(ctx context.Context) {
		utilities.TraceError(ctx, fmt.Errorf("test_error: %w", &testErrorWithResponse{
			Response: &http.Response{
				Status:     "some status",
				StatusCode: 420,
			},
		}))
	})
}

func TestTraceErrorResponseHttpNotNilWithRequest(t *testing.T) {
	checkTFLog(t, `
[TRACE] provider: trace_error: error_message="test_error: testErrorWithResponse" error_type="*fmt.wrapError" error_dump="&fmt.wrapError{msg:\"test_error: testErrorWithResponse\", err:(*utilities_test.testErrorWithResponse)(PTR)}" error_package_path=""
[TRACE] provider: trace_error: error_message=testErrorWithResponse error_type="*utilities_test.testErrorWithResponse" error_dump="&utilities_test.testErrorWithResponse{Response:(*http.Response)(PTR)}" error_package_path=""
[TRACE] provider: trace_error: response_status_code=420 response_status="some status"
[TRACE] provider: trace_error: request_uri=test_uri request_host=test_host
`, func(ctx context.Context) {
		utilities.TraceError(ctx, fmt.Errorf("test_error: %w", &testErrorWithResponse{
			Response: &http.Response{
				Status:     "some status",
				StatusCode: 420,
				Request: &http.Request{
					RequestURI: "test_uri",
					Host:       "test_host",
				},
			},
		}))
	})
}

type testErrorWithBody struct {
	Body []byte
}

func (e *testErrorWithBody) Error() string {
	return "testErrorWithBody"
}
func TestTraceErrorBodyEmpty(t *testing.T) {
	checkTFLog(t, `
[TRACE] provider: trace_error: error_message="test_error: testErrorWithBody" error_type="*fmt.wrapError" error_dump="&fmt.wrapError{msg:\"test_error: testErrorWithBody\", err:(*utilities_test.testErrorWithBody)(PTR)}" error_package_path=""
[TRACE] provider: trace_error: error_message=testErrorWithBody error_type="*utilities_test.testErrorWithBody" error_dump="&utilities_test.testErrorWithBody{Body:[]uint8(nil)}" error_package_path=""
[TRACE] provider: trace_error: body=""
`, func(ctx context.Context) {
		utilities.TraceError(ctx, fmt.Errorf("test_error: %w", &testErrorWithBody{}))
	})
}

func TestTraceErrorPathError(t *testing.T) {
	checkTFLog(t, `
[TRACE] provider: trace_error: error_message="test_error: open fileThatDoesNotExist...: no such file or directory" error_type="*fmt.wrapError" error_dump="&fmt.wrapError{msg:\"test_error: open fileThatDoesNotExist...: no such file or directory\", err:(*fs.PathError)(PTR)}" error_package_path=""
[TRACE] provider: trace_error: error_message="open fileThatDoesNotExist...: no such file or directory" error_type="*fs.PathError" error_dump="&fs.PathError{Op:\"open\", Path:\"fileThatDoesNotExist...\", Err:0x2}" error_package_path=""
[TRACE] provider: trace_error: error_message="no such file or directory" error_type=syscall.Errno error_dump=0x2 error_package_path=syscall
`, func(ctx context.Context) {
		_, err := os.Open("fileThatDoesNotExist...")
		utilities.TraceError(ctx, fmt.Errorf("test_error: %w", err))
	})
}

func TestTraceErrorBodyNotEmpty(t *testing.T) {
	checkTFLog(t, `
[TRACE] provider: trace_error: error_message="test_error: testErrorWithBody" error_type="*fmt.wrapError" error_dump="&fmt.wrapError{msg:\"test_error: testErrorWithBody\", err:(*utilities_test.testErrorWithBody)(PTR)}" error_package_path=""
[TRACE] provider: trace_error: error_message=testErrorWithBody error_type="*utilities_test.testErrorWithBody" error_dump="&utilities_test.testErrorWithBody{Body:[]uint8{0x74, 0x65, 0x73, 0x74, 0x5f, 0x62, 0x6f, 0x64, 0x79}}" error_package_path=""
[TRACE] provider: trace_error: body=test_body
`, func(ctx context.Context) {
		utilities.TraceError(ctx, fmt.Errorf("test_error: %w", &testErrorWithBody{Body: []byte("test_body")}))
	})
}

func TestRetry0(t *testing.T) {
	checkTFLog(t, `
[INFO]  provider: TestRetry_testfunc
[WARN]  provider: retry_attempt: context=test_retry attempt_done_count=1 cooldown_ms=10 error_message=test_error
[TRACE] provider: retry_attempt: attempt_limit=5
[TRACE] provider: trace_error: error_message=test_error error_type="*errors.errorString" error_dump="&errors.errorString{s:\"test_error\"}" error_package_path=""
[INFO]  provider: TestRetry_testfunc
[WARN]  provider: retry_attempt: context=test_retry attempt_done_count=2 cooldown_ms=10 error_message=test_error
[TRACE] provider: retry_attempt: attempt_limit=5
[TRACE] provider: trace_error: error_message=test_error error_type="*errors.errorString" error_dump="&errors.errorString{s:\"test_error\"}" error_package_path=""
[INFO]  provider: TestRetry_testfunc
[WARN]  provider: retry_attempt: context=test_retry attempt_done_count=3 cooldown_ms=10 error_message=test_error
[TRACE] provider: retry_attempt: attempt_limit=5
[TRACE] provider: trace_error: error_message=test_error error_type="*errors.errorString" error_dump="&errors.errorString{s:\"test_error\"}" error_package_path=""
[INFO]  provider: TestRetry_testfunc
[WARN]  provider: retry_attempt: context=test_retry attempt_done_count=4 cooldown_ms=10 error_message=test_error
[TRACE] provider: retry_attempt: attempt_limit=5
[TRACE] provider: trace_error: error_message=test_error error_type="*errors.errorString" error_dump="&errors.errorString{s:\"test_error\"}" error_package_path=""
[INFO]  provider: TestRetry_testfunc
[WARN]  provider: retry_attempt: context=test_retry attempt_done_count=5 cooldown_ms=10 error_message=test_error
[TRACE] provider: retry_attempt: attempt_limit=5
[TRACE] provider: trace_error: error_message=test_error error_type="*errors.errorString" error_dump="&errors.errorString{s:\"test_error\"}" error_package_path=""
[ERROR] provider: retry_attempt: context=test_retry attempt_limit=5 error_message=test_error
`, func(ctx context.Context) {
		err := utilities.Retry(ctx, time.Millisecond*10, 0.0, 5, "test_retry", func() (bool, error) {
			tflog.Info(ctx, "TestRetry_testfunc")
			return false, fmt.Errorf("test_error")
		})
		if err.Error() != "test_error" {
			t.Fail()
		}
	})
}

func TestRetry1(t *testing.T) {
	checkTFLog(t, `
[INFO]  provider: TestRetry_testfunc
[WARN]  provider: retry_attempt: context=test_retry attempt_done_count=1 cooldown_ms=10 error_message=test_error
[TRACE] provider: retry_attempt: attempt_limit=5
[TRACE] provider: trace_error: error_message=test_error error_type="*errors.errorString" error_dump="&errors.errorString{s:\"test_error\"}" error_package_path=""
[INFO]  provider: TestRetry_testfunc
[WARN]  provider: retry_attempt: context=test_retry attempt_done_count=2 cooldown_ms=20 error_message=test_error
[TRACE] provider: retry_attempt: attempt_limit=5
[TRACE] provider: trace_error: error_message=test_error error_type="*errors.errorString" error_dump="&errors.errorString{s:\"test_error\"}" error_package_path=""
[INFO]  provider: TestRetry_testfunc
[WARN]  provider: retry_attempt: context=test_retry attempt_done_count=3 cooldown_ms=40 error_message=test_error
[TRACE] provider: retry_attempt: attempt_limit=5
[TRACE] provider: trace_error: error_message=test_error error_type="*errors.errorString" error_dump="&errors.errorString{s:\"test_error\"}" error_package_path=""
[INFO]  provider: TestRetry_testfunc
[WARN]  provider: retry_attempt: context=test_retry attempt_done_count=4 cooldown_ms=80 error_message=test_error
[TRACE] provider: retry_attempt: attempt_limit=5
[TRACE] provider: trace_error: error_message=test_error error_type="*errors.errorString" error_dump="&errors.errorString{s:\"test_error\"}" error_package_path=""
[INFO]  provider: TestRetry_testfunc
[WARN]  provider: retry_attempt: context=test_retry attempt_done_count=5 cooldown_ms=160 error_message=test_error
[TRACE] provider: retry_attempt: attempt_limit=5
[TRACE] provider: trace_error: error_message=test_error error_type="*errors.errorString" error_dump="&errors.errorString{s:\"test_error\"}" error_package_path=""
[ERROR] provider: retry_attempt: context=test_retry attempt_limit=5 error_message=test_error
`, func(ctx context.Context) {
		err := utilities.Retry(ctx, time.Millisecond*10, 1.0, 5, "test_retry", func() (bool, error) {
			tflog.Info(ctx, "TestRetry_testfunc")
			return false, fmt.Errorf("test_error")
		})
		if err.Error() != "test_error" {
			t.Fail()
		}

	})
}

func TestRetry2(t *testing.T) {
	checkTFLog(t, `
[INFO]  provider: TestRetry_testfunc
[WARN]  provider: retry_attempt: context=test_retry attempt_done_count=1 cooldown_ms=10 error_message=test_error
[TRACE] provider: retry_attempt: attempt_limit=5
[TRACE] provider: trace_error: error_message=test_error error_type="*errors.errorString" error_dump="&errors.errorString{s:\"test_error\"}" error_package_path=""
[INFO]  provider: TestRetry_testfunc
[WARN]  provider: retry_attempt: context=test_retry attempt_done_count=2 cooldown_ms=15 error_message=test_error
[TRACE] provider: retry_attempt: attempt_limit=5
[TRACE] provider: trace_error: error_message=test_error error_type="*errors.errorString" error_dump="&errors.errorString{s:\"test_error\"}" error_package_path=""
[INFO]  provider: TestRetry_testfunc
[WARN]  provider: retry_attempt: context=test_retry attempt_done_count=3 cooldown_ms=22 error_message=test_error
[TRACE] provider: retry_attempt: attempt_limit=5
[TRACE] provider: trace_error: error_message=test_error error_type="*errors.errorString" error_dump="&errors.errorString{s:\"test_error\"}" error_package_path=""
[INFO]  provider: TestRetry_testfunc
[WARN]  provider: retry_attempt: context=test_retry attempt_done_count=4 cooldown_ms=33 error_message=test_error
[TRACE] provider: retry_attempt: attempt_limit=5
[TRACE] provider: trace_error: error_message=test_error error_type="*errors.errorString" error_dump="&errors.errorString{s:\"test_error\"}" error_package_path=""
[INFO]  provider: TestRetry_testfunc
[WARN]  provider: retry_attempt: context=test_retry attempt_done_count=5 cooldown_ms=49 error_message=test_error
[TRACE] provider: retry_attempt: attempt_limit=5
[TRACE] provider: trace_error: error_message=test_error error_type="*errors.errorString" error_dump="&errors.errorString{s:\"test_error\"}" error_package_path=""
[ERROR] provider: retry_attempt: context=test_retry attempt_limit=5 error_message=test_error
`, func(ctx context.Context) {
		err := utilities.Retry(ctx, time.Millisecond*10, 0.5, 5, "test_retry", func() (bool, error) {
			tflog.Info(ctx, "TestRetry_testfunc")
			return false, fmt.Errorf("test_error")
		})
		if err.Error() != "test_error" {
			t.Fail()
		}
	})
}

func TestRetryGiveup(t *testing.T) {
	checkTFLog(t, `
[INFO]  provider: TestRetry_testfunc
[WARN]  provider: retry_attempt: context=test_retry attempt_done_count=1 cooldown_ms=10 error_message="test_error attempt:1"
[TRACE] provider: retry_attempt: attempt_limit=5
[TRACE] provider: trace_error: error_message="test_error attempt:1" error_type="*errors.errorString" error_dump="&errors.errorString{s:\"test_error attempt:1\"}" error_package_path=""
[INFO]  provider: TestRetry_testfunc
[WARN]  provider: retry_attempt: context=test_retry attempt_done_count=2 cooldown_ms=10 error_message="test_error attempt:2"
[TRACE] provider: retry_attempt: attempt_limit=5
[TRACE] provider: trace_error: error_message="test_error attempt:2" error_type="*errors.errorString" error_dump="&errors.errorString{s:\"test_error attempt:2\"}" error_package_path=""
[INFO]  provider: TestRetry_testfunc
[WARN]  provider: retry_attempt: context=test_retry attempt_done_count=3 cooldown_ms=10 error_message=test_error_give_up
[TRACE] provider: retry_attempt: attempt_limit=5
[TRACE] provider: trace_error: error_message=test_error_give_up error_type="*errors.errorString" error_dump="&errors.errorString{s:\"test_error_give_up\"}" error_package_path=""
`, func(ctx context.Context) {

		attemptCount := 0

		err := utilities.Retry(ctx, time.Millisecond*10, 0.0, 5, "test_retry", func() (bool, error) {
			tflog.Info(ctx, "TestRetry_testfunc")

			attemptCount++

			if attemptCount == 3 {
				return true, fmt.Errorf("test_error_give_up")
			}

			return false, fmt.Errorf("test_error attempt:%d", attemptCount)
		})
		if err.Error() != "test_error_give_up" {
			t.Fail()
		}
	})
}

func TestRetryPassAfter2(t *testing.T) {
	checkTFLog(t, `
[INFO]  provider: TestRetry_testfunc
[WARN]  provider: retry_attempt: context=test_retry attempt_done_count=1 cooldown_ms=10 error_message="test_error attempt:1"
[TRACE] provider: retry_attempt: attempt_limit=5
[TRACE] provider: trace_error: error_message="test_error attempt:1" error_type="*errors.errorString" error_dump="&errors.errorString{s:\"test_error attempt:1\"}" error_package_path=""
[INFO]  provider: TestRetry_testfunc
[WARN]  provider: retry_attempt: context=test_retry attempt_done_count=2 cooldown_ms=10 error_message="test_error attempt:2"
[TRACE] provider: retry_attempt: attempt_limit=5
[TRACE] provider: trace_error: error_message="test_error attempt:2" error_type="*errors.errorString" error_dump="&errors.errorString{s:\"test_error attempt:2\"}" error_package_path=""
[INFO]  provider: TestRetry_testfunc
`, func(ctx context.Context) {

		attemptCount := 0

		err := utilities.Retry(ctx, time.Millisecond*10, 0.0, 5, "test_retry", func() (bool, error) {
			tflog.Info(ctx, "TestRetry_testfunc")

			attemptCount++

			if attemptCount == 3 {
				return false, nil
			}

			return false, fmt.Errorf("test_error attempt:%d", attemptCount)
		})
		if err != nil {
			t.Fail()
		}
	})
}
