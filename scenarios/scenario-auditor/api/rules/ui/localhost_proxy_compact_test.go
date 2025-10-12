//go:build ruletests
// +build ruletests

package ui

import "testing"

func TestProxyCompatibleUIBaseDocCases(t *testing.T) {
	runDocTestsViolations(t, "localhost_proxy_compact.go", "ui/index.js", CheckProxyCompatibleUIAssets)
}
