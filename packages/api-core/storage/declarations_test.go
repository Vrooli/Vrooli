package storage

import (
	"errors"
	"strings"
	"testing"
)

func TestPortablePathUsesInjectedPlatformSeams(t *testing.T) {
	path := PortablePath{Value: "$USER_CONFIG_DIR/vrooli"}
	seams := PlatformSeams{
		UserHomeDir:   func() (string, error) { return "/Users/alice", nil },
		UserConfigDir: func() (string, error) { return "/Users/alice/Library/Application Support", nil },
		UserCacheDir:  func() (string, error) { return "/Users/alice/Library/Caches", nil },
	}
	got, err := ResolvePortablePath("config", path, PlatformMacOS, seams)
	if err != nil || got != "/Users/alice/Library/Application Support/vrooli" {
		t.Fatalf("got %q, err %v", got, err)
	}
}

func TestPortablePathRejectsXDGAndReportsAbsence(t *testing.T) {
	if _, err := ResolvePortablePath("cache", PortablePath{Value: "$XDG_CACHE_HOME/vrooli"}, PlatformLinux, PlatformSeams{}); err == nil || !strings.Contains(err.Error(), "$USER_CACHE_DIR") {
		t.Fatalf("unexpected XDG error: %v", err)
	}
	_, err := ResolvePortablePath("optional", PortablePath{ByOS: map[Platform]*string{PlatformLinux: nil}}, PlatformLinux, PlatformSeams{})
	var absent *NotApplicable
	if !errors.As(err, &absent) {
		t.Fatalf("error %v is not typed NotApplicable", err)
	}
}
