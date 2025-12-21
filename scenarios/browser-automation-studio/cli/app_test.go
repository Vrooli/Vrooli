package main

import "testing"

func TestAppConstants(t *testing.T) {
	if appName != "browser-automation-studio" {
		t.Errorf("appName = %q, want 'browser-automation-studio'", appName)
	}
	if appVersion == "" {
		t.Errorf("appVersion should not be empty")
	}
}

func TestBuildVariables(t *testing.T) {
	if buildFingerprint == "" {
		t.Log("buildFingerprint is empty (expected at build time)")
	}
	if buildTimestamp == "" {
		t.Log("buildTimestamp is empty (expected at build time)")
	}
}
