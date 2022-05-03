/*
Copyright 2022 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/tools/checker"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(checker.TFLogAnalyzer)
}
