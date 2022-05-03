/*
Copyright 2022 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package checker_test

import (
	"fmt"
	"testing"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/tools/checker"
	"golang.org/x/tools/go/analysis/analysistest"
)

func testTFLogLine(t *testing.T, testName string, line string) {
	dir, cleanup, err := analysistest.WriteFiles(map[string]string{
		"github.com/hashicorp/terraform-plugin-log/tflog/mock.go": `
package tflog
func Info(_ ...interface{}) {}
`,
		fmt.Sprintf("package0/%s.go", testName): fmt.Sprintf(`
package package0

import (
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"context"
)
func f0(ctx context.Context) {
	%s
}
		`, line),
	})
	defer cleanup()
	if err != nil {
		panic(err)
	}

	analysistest.Run(t, dir, checker.TFLogAnalyzer, "package0")

}

func Test(t *testing.T) {
	testTFLogLine(t, "test0", `tflog.Info(ctx, "test", 234) // want "key should be string constant" "value is missing"`)
	testTFLogLine(t, "test1", `tflog.Info(ctx, "test", "test", 123)`)
	testTFLogLine(t, "test2", `tflog.Info(ctx, "test")`)
	testTFLogLine(t, "test3", `tflog.Info(ctx, "%d", "test", "234") // want "message should be static message not a format string"`)
	testTFLogLine(t, "test4", `tflog.Info(ctx, "test", 3234, 234) // want "key should be string constant"`)
	testTFLogLine(t, "test5", `tflog.Info(ctx, "test", "test") // want "value is missing"`)
}
