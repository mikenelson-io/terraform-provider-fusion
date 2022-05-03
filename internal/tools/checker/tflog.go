/*
Copyright 2022 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package checker

import (
	"bytes"
	"go/ast"
	"go/constant"
	"go/printer"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

// TFLogAnalyzer of the linter
var TFLogAnalyzer = &analysis.Analyzer{
	Name:             "tflogChecker",
	Doc:              "Checks for common misuses of tflog module",
	Requires:         []*analysis.Analyzer{inspect.Analyzer},
	Run:              tflogRun,
	RunDespiteErrors: true,
}

var tflogDocsUrl = "https://www.terraform.io/plugin/log/writing"

func tflogRun(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		call := n.(*ast.CallExpr)

		fn, _ := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
		if fn == nil {
			return
		}

		isTflog := false
		for _, fName := range []string{"Trace", "Debug", "Error", "Info", "Warn"} {
			if fn.FullName() == "github.com/hashicorp/terraform-plugin-log/tflog."+fName {
				isTflog = true
				break
			}
		}
		if isTflog {
			if len(call.Args) < 2 {
				pass.Reportf(call.Lparen, "%s missing message argument", fn.Name())
			} else {
				v := pass.TypesInfo.Types[call.Args[1]].Value

				if v != nil && v.Kind() == constant.String {
					if strings.Contains(v.ExactString(), "%") {
						pass.Reportf(call.Lparen, "Parameter: %d: %s: message should be static message not a format string, please see:%s", 1, v.String(), tflogDocsUrl)
					}
				} else {
					pass.Reportf(call.Lparen, "Parameter: %d: %s: message should be string constant, please see %s", 1, render(pass.Fset, call.Args[1]), tflogDocsUrl)
				}
				for argI := 2; argI < len(call.Args); argI += 2 {
					v := pass.TypesInfo.Types[call.Args[argI]].Value
					if !(v != nil && v.Kind() == constant.String) {
						pass.Reportf(call.Lparen, "Parameter: %d: %s: key should be string constant, please see %s", argI, render(pass.Fset, call.Args[argI]), tflogDocsUrl)
					}
					if argI+1 >= len(call.Args) {
						pass.Reportf(call.Lparen, "Parameter: %d: value is missing, please see %s", argI+1, tflogDocsUrl)
					}
				}
			}
		}

	})

	return nil, nil
}

// render returns the pretty-print of the given node
func render(fset *token.FileSet, x interface{}) string {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, x); err != nil {
		panic(err)
	}
	return buf.String()
}
