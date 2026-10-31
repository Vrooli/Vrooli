package storage

import (
	"errors"
	"strings"
	"testing"
)

func TestPortableTokensUseTargetPlatformConventions(t *testing.T) {
	cases := []struct {
		platform Platform
		home     string
		want     map[string]string
	}{
		{PlatformLinux, "/home/tester", map[string]string{
			"$USER_HOME":       "/home/tester",
			"$USER_CONFIG_DIR": "/home/tester/.config",
			"$USER_CACHE_DIR":  "/home/tester/.cache",
			"$USER_STATE_DIR":  "/home/tester/.local/state",
			"$USER_DATA_DIR":   "/home/tester/.local/share",
			"$TEMP_DIR":        "/tmp",
		}},
		{PlatformMacOS, "/Users/tester", map[string]string{
			"$USER_HOME":       "/Users/tester",
			"$USER_CONFIG_DIR": "/Users/tester/Library/Application Support",
			"$USER_CACHE_DIR":  "/Users/tester/Library/Caches",
			"$USER_STATE_DIR":  "/Users/tester/Library/Application Support",
			"$USER_DATA_DIR":   "/Users/tester/Library/Application Support",
			"$TEMP_DIR":        "/tmp",
		}},
		{PlatformWindows, `C:\Users\tester`, map[string]string{
			"$USER_HOME":       `C:\Users\tester`,
			"$USER_CONFIG_DIR": `C:\Users\tester\AppData\Roaming`,
			"$USER_CACHE_DIR":  `C:\Users\tester\AppData\Local`,
			"$USER_STATE_DIR":  `C:\Users\tester\AppData\Local`,
			"$USER_DATA_DIR":   `C:\Users\tester\AppData\Local`,
			"$TEMP_DIR":        `C:\Windows\Temp`,
		}},
	}
	for _, tc := range cases {
		t.Run(string(tc.platform), func(t *testing.T) {
			seams := DefaultSeams(tc.platform, UserIdentity{HomeDir: tc.home})
			for token, wantBase := range tc.want {
				got, err := ResolvePortablePath(token, PortablePath{Value: token + "/storage"}, tc.platform, seams)
				if err != nil {
					t.Fatalf("%s: resolve: %v", token, err)
				}
				want := wantBase + "/storage"
				if tc.platform == PlatformWindows {
					want = strings.ReplaceAll(want, "/", `\`)
				}
				if got != want {
					t.Errorf("%s = %q, want %q", token, got, want)
				}
			}
		})
	}
}

func TestPortableWindowsResolutionDoesNotUseHostHome(t *testing.T) {
	got, err := ResolvePortablePath("cache", PortablePath{Value: "$USER_CACHE_DIR/uv"}, PlatformWindows, PlatformSeams{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "/home/") || !strings.HasPrefix(got, `C:\Users\vrooli\AppData\Local`) {
		t.Fatalf("windows cache path = %q", got)
	}
}

func TestPortableWindowsAbsoluteDriveAndUNCFromLinux(t *testing.T) {
	for _, value := range []string{`C:/ProgramData/docker`, `\\server\share\docker`} {
		got, err := ResolvePortablePath("docker", PortablePath{Value: value}, PlatformWindows, PlatformSeams{})
		if err != nil {
			t.Fatalf("%s: %v", value, err)
		}
		if got == "" {
			t.Fatalf("%s resolved to empty path", value)
		}
	}
}

func TestPortableDataAndStateAreDistinctOnLinux(t *testing.T) {
	data, err := ResolvePortablePath("data", PortablePath{Value: "$USER_DATA_DIR/app"}, PlatformLinux, PlatformSeams{})
	if err != nil {
		t.Fatal(err)
	}
	state, err := ResolvePortablePath("state", PortablePath{Value: "$USER_STATE_DIR/app"}, PlatformLinux, PlatformSeams{})
	if err != nil {
		t.Fatal(err)
	}
	if data == state {
		t.Fatalf("data and state resolved identically: %q", data)
	}
}

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
