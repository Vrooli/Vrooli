package runtime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

func TestUserLocalPackagePublishesOnlyAfterRuntimeQualification(t *testing.T) {
	for _, fails := range []bool{false, true} {
		t.Run(map[bool]string{false: "qualified", true: "missing_runtime_library"}[fails], func(t *testing.T) {
			withArch(t, "amd64")
			bin := withUserBin(t)
			opt := withUserOpt(t)
			withoutCommandOnPath(t)
			payload := hostreqkit.UserLocalAPT{Packages: []string{"example-tool"}, Executable: "usr/bin/example-tool", LibraryDir: "usr/lib/x86_64-linux-gnu", RuntimeEnv: map[string]string{"EXAMPLE_DATA": "usr/share/example"}}
			h := toolHandler{manifest: hostreqkit.ToolManifest{Name: "example-tool", Commands: []string{"example-tool"}, VersionArgs: []string{"--version"}, DefaultPackage: "example-tool", UserLocalAPT: map[string]hostreqkit.UserLocalAPT{"amd64": payload}}}
			original := packagePayloadCommand
			defer func() { packagePayloadCommand = original }()
			var download, qualified bool
			packagePayloadCommand = func(ctx context.Context, dir string, env []string, name string, args ...string) ([]byte, error) {
				switch name {
				case "apt-get":
					download = true
					want := []string{"-o", "APT::Get::AllowUnauthenticated=false", "-o", "Acquire::AllowInsecureRepositories=false", "download", "--", "example-tool"}
					if !reflect.DeepEqual(args, want) {
						t.Fatalf("download must preserve authentication and fixed package identity: %v", args)
					}
					return nil, os.WriteFile(filepath.Join(dir, "example-tool.deb"), []byte("authenticated package fixture"), 0644)
				case "dpkg-deb":
					if args[0] == "--show" {
						return []byte("example-tool 1.0 amd64"), nil
					}
					if args[0] != "--extract" {
						t.Fatalf("package manager install/scripts must not run: %v", args)
					}
					entry := filepath.Join(args[2], payload.Executable)
					if err := os.MkdirAll(filepath.Dir(entry), 0755); err != nil {
						return nil, err
					}
					return nil, os.WriteFile(entry, []byte("#!/bin/sh\necho 'example-tool 1.0'\n"), 0755)
				default:
					qualified = true
					if !strings.HasSuffix(name, payload.Executable) || !strings.Contains(strings.Join(env, "\n"), "LD_LIBRARY_PATH="+filepath.Join(dir, payload.LibraryDir)) {
						t.Fatalf("qualification must use extracted runtime: %q %v", name, env)
					}
					if fails {
						return []byte("missing library"), errors.New("loader failed")
					}
					return []byte("example-tool 1.0\n"), nil
				}
			}
			host := hostreqkit.Host{OS: "linux", PackageManager: "apt"}
			got, err := h.Apply(host, h.Inspect(host, resolved("example-tool")), hostreqkit.EnsureOptions{})
			if err != nil || !download || !qualified {
				t.Fatalf("acquisition/qualification did not run: %+v %v", got, err)
			}
			launcher, readErr := os.ReadFile(filepath.Join(bin, "example-tool"))
			if fails {
				if got.ExecutionState != hostreqkit.ExecutionFailed || readErr == nil {
					t.Fatalf("unqualified payload was published: %+v, %s", got, launcher)
				}
				return
			}
			if got.ExecutionState != hostreqkit.ExecutionInstalled || readErr != nil || !strings.Contains(string(launcher), "EXAMPLE_DATA=") {
				t.Fatalf("qualified runtime absent: %+v %v %s", got, readErr, launcher)
			}
			receipts, _ := filepath.Glob(filepath.Join(opt, "example-tool", "*", "package-receipt.json"))
			if len(receipts) != 1 {
				t.Fatalf("missing durable package receipt: %v", receipts)
			}
		})
	}
}

func TestUserLocalPackageRejectsUnboundedAcquisition(t *testing.T) {
	valid := hostreqkit.UserLocalAPT{Packages: []string{"example-tool"}, Executable: "usr/bin/tool", LibraryDir: "usr/lib"}
	for _, name := range []string{"--unauthenticated", "https://evil.test/x.deb", "../../tool", "tool=unreviewed", "tool; touch x"} {
		payload := valid
		payload.Packages = []string{name}
		if err := validatePackagePayload(payload); err == nil {
			t.Fatalf("accepted unbounded package %q", name)
		}
	}
	valid.Executable = "../../outside"
	if err := validatePackagePayload(valid); err == nil {
		t.Fatal("accepted executable outside payload")
	}
}
