package envkit

import (
	"reflect"
	"runtime"
	"strconv"
	"testing"
)

func TestToolchainAppendsWidthToInheritedGoflags(t *testing.T) {
	got := ToolchainWithPlatform(Env{"GOFLAGS=-mod=mod"}, ToolchainOptions{Width: 4}, Platform{})
	if !containsValue(got, "GOFLAGS", "-mod=mod -p=4") {
		t.Fatalf("GOFLAGS not composed: %#v", got)
	}
}

func TestToolchainPreservesExplicitWidth(t *testing.T) {
	got := ToolchainWithPlatform(Env{"GOFLAGS=-p=2 -trimpath"}, ToolchainOptions{Width: 4}, Platform{})
	if !containsValue(got, "GOFLAGS", "-p=2 -trimpath") {
		t.Fatalf("explicit -p was replaced: %#v", got)
	}
}

func TestToolchainSetsGomaxprocsAndPnpmConcurrency(t *testing.T) {
	got := ToolchainWithPlatform(Env{"HOME=/h"}, ToolchainOptions{Width: 3}, Platform{})
	want := Env{
		"HOME=/h",
		"GOFLAGS=-p=3",
		"GOMAXPROCS=6",
		"npm_config_child_concurrency=3",
		"npm_config_workspace_concurrency=3",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("floor = %#v, want %#v", got, want)
	}
}

func TestToolchainNeverReplacesAnInheritedValue(t *testing.T) {
	parent := Env{"GOMAXPROCS=1", "npm_config_child_concurrency=9", "npm_config_workspace_concurrency=7", "CLAUDE_CODE_SESSION_ID=s"}
	got := ToolchainWithPlatform(parent, ToolchainOptions{Width: 4}, Platform{})
	for _, entry := range parent {
		if !containsEntry(got, entry) {
			t.Fatalf("inherited %q lost: %#v", entry, got)
		}
	}
	if !containsValue(got, "GOFLAGS", "-p=4") {
		t.Fatalf("GOFLAGS missing: %#v", got)
	}
}

func TestToolchainIsCaseFoldedOnWindows(t *testing.T) {
	got := ToolchainWithPlatform(Env{"GoFlags=-mod=vendor", "GoMaxProcs=2"}, ToolchainOptions{Width: 4}, Platform{CaseInsensitive: true})
	if !containsEntry(got, "GoFlags=-mod=vendor -p=4") || !containsEntry(got, "GoMaxProcs=2") || contains(got, "GOMAXPROCS") {
		t.Fatalf("windows floor = %#v", got)
	}
}

func TestToolchainWithoutWidthResolvesTheLeverFromTheEnvironment(t *testing.T) {
	got := ToolchainWithPlatform(Env{"GOFLAGS=-mod=mod", "VROOLI_TUNING_BUILD_WIDTH=2"}, ToolchainOptions{}, Platform{})
	if !containsValue(got, "GOFLAGS", "-mod=mod -p=2") || !containsValue(got, "GOMAXPROCS", "4") {
		t.Fatalf("lever override ignored: %#v", got)
	}
	want := "-p=" + strconv.Itoa(DefaultBuildWidth())
	for _, raw := range []string{"", "abc", "0", "-3"} {
		got := ToolchainWithPlatform(Env{"VROOLI_TUNING_BUILD_WIDTH=" + raw}, ToolchainOptions{}, Platform{})
		if !containsValue(got, "GOFLAGS", want) {
			t.Fatalf("override %q: %#v, want GOFLAGS=%s", raw, got, want)
		}
	}
}

func TestToolchainFillsAnEmptyInheritedValue(t *testing.T) {
	got := ToolchainWithPlatform(Env{"GOMAXPROCS=", "GOFLAGS=", "HOME=/h"}, ToolchainOptions{Width: 4}, Platform{})
	want := Env{"GOFLAGS=-p=4", "HOME=/h", "GOMAXPROCS=8", "npm_config_child_concurrency=4", "npm_config_workspace_concurrency=4"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("floor = %#v, want %#v", got, want)
	}
}

func TestToolchainAppendsSiteFlagsWithoutRepeating(t *testing.T) {
	got := ToolchainWithPlatform(Env{"GOFLAGS=-mod=vendor -p=2"}, ToolchainOptions{Width: 4, GoFlags: []string{"-mod=mod", "-p=2"}}, Platform{})
	if !containsValue(got, "GOFLAGS", "-mod=vendor -p=2 -mod=mod") {
		t.Fatalf("site flags not composed: %#v", got)
	}
	got = ToolchainWithPlatform(Env{}, ToolchainOptions{Width: 4, GoFlags: []string{"-mod=mod"}}, Platform{})
	if !containsValue(got, "GOFLAGS", "-p=4 -mod=mod") {
		t.Fatalf("site flags without inherited GOFLAGS: %#v", got)
	}
}

func TestDefaultBuildWidthIsBoundedByFourAndAtLeastOne(t *testing.T) {
	got := DefaultBuildWidth()
	if got < 1 || got > 4 || got != min(4, max(1, runtime.NumCPU()/4)) {
		t.Fatalf("DefaultBuildWidth() = %d on %d CPUs", got, runtime.NumCPU())
	}
}

func containsEntry(env Env, entry string) bool {
	for _, candidate := range env {
		if candidate == entry {
			return true
		}
	}
	return false
}
