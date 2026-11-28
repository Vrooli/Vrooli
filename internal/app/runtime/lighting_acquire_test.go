package runtimeapp

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vrooli/api-core/storage/storagetest"
	"github.com/vrooli/vrooli/internal/resources"
)

func TestGovernedAcquisitionRejectsBlockedOrMalformedSuccessfulCLIReply(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("fixture targets the qualified pinned linux-amd64 profile")
	}
	for _, tc := range []struct{ name, reply, want string }{
		{"blocked", `{"installed":false,"blocked":true,"verdict":"unrecorded","message":"selected package is not admitted"}`, "unrecorded"},
		{"dry run cannot install", `{"installed":true,"dryRun":true,"verdict":"approved"}`, "did not admit"},
		{"malformed", `{"installed":true,"unexpected":"field"}`, "invalid governed installation receipt"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			storagetest.IsolateStorage(t)
			home := t.TempDir()
			binary := filepath.Join(home, ".vrooli/bin/scenario-dependency-analyzer")
			if err := os.MkdirAll(filepath.Dir(binary), 0o700); err != nil {
				t.Fatal(err)
			}
			argsPath := filepath.Join(t.TempDir(), "args")
			t.Setenv("LIGHTING_SETUP_TEST_ARGS", argsPath)
			script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$LIGHTING_SETUP_TEST_ARGS\"\ncat <<'_LIGHTING_RECEIPT_'\n" + tc.reply + "\n_LIGHTING_RECEIPT_\n"
			if err := os.WriteFile(binary, []byte(script), 0o700); err != nil {
				t.Fatal(err)
			}
			root, err := filepath.Abs("../../..")
			if err != nil {
				t.Fatal(err)
			}
			controller := resources.NewController(root, home)
			_, err = acquireManagedLighting(context.Background(), controller, home, "1.1")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("blocked receipt admitted: %v", err)
			}
			args, err := os.ReadFile(argsPath)
			if err != nil {
				t.Fatal(err)
			}
			if string(args) != strings.Join([]string{"deps", "install", "source/openrgb@1.1", "--scenario", "openrgb", "--surface", "host-tool", "--exact", "--apply", "--json"}, "\n")+"\n" {
				t.Fatalf("unexpected command scope: %s", args)
			}
		})
	}
}

func TestGovernedAcquisitionRejectsChangedReleaseBeforeInvokingInstaller(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	if _, err := acquireManagedLighting(context.Background(), resources.NewController(root, home), home, "latest"); err == nil || !strings.Contains(err.Error(), "release changed") {
		t.Fatalf("unpinned acquisition admitted: %v", err)
	}
}

func TestNativeLightingRootAcquisitionNeverRunsUserOwnedSDA(t *testing.T) {
	storagetest.IsolateStorage(t)
	home := t.TempDir()
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(home, "hostile-sda-was-run")
	binary := filepath.Join(home, ".vrooli", "bin", "scenario-dependency-analyzer")
	if err = os.MkdirAll(filepath.Dir(binary), 0o700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(binary, []byte("#!/bin/sh\nprintf compromised > '"+marker+"'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	_, err = acquireManagedLightingWith(context.Background(), resources.NewController(root, home), home, "1.1", lightingAcquisitionEffects{nativeRoot: true, userInstall: func(context.Context, *resources.Controller, string, string) error {
		t.Fatal("privileged setup entered user-owned SDA execution")
		return nil
	}})
	if err == nil || !strings.Contains(err.Error(), "ordinary operator") {
		t.Fatalf("privileged acquisition did not explain user-owned acquisition: %v", err)
	}
	if _, err = os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("hostile user-owned SDA executed")
	}
}
