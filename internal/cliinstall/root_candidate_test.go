package cliinstall

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	mode := os.Getenv("VROOLI_ROOT_IDENTITY_TEST_FIXTURE")
	if mode == "" {
		if executable, err := os.Executable(); err == nil {
			if data, err := os.ReadFile(filepath.Join(filepath.Dir(executable), ".root-identity-fixture")); err == nil {
				mode = string(data)
			}
		}
	}
	if mode != "" {
		identity := `{"cli_version":"1.0.0","platform_version":"2.0.0","root":"","build_identity":"` + strings.Repeat("a", 64) + `","runtime_home_contract_sha256":"` + strings.Repeat("b", 64) + `"}`
		switch mode {
		case "valid":
			fmt.Print(identity)
		case "unstamped":
			fmt.Print(strings.Replace(identity, strings.Repeat("a", 64), "", 1))
		case "source-bound":
			fmt.Print(strings.Replace(identity, strings.Repeat("b", 64), "", 1))
		case "duplicate":
			fmt.Print(strings.Replace(identity, `"build_identity":`, `"build_identity":"wrong","build_identity":`, 1))
		case "overflow":
			fmt.Print(strings.Repeat("x", 16384))
		case "timeout":
			time.Sleep(time.Minute)
		default:
			fmt.Print("not-json")
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestRootCLIRejectedCandidatePreservesCurrentAndFallback(t *testing.T) {
	for _, mode := range []string{"wrong executable", "wrong Go role", "script", "empty", "oversized", "symlink"} {
		t.Run(mode, func(t *testing.T) {
			home := t.TempDir()
			bin := filepath.Join(home, "bin")
			if err := os.Mkdir(bin, 0o755); err != nil {
				t.Fatal(err)
			}
			dst := filepath.Join(bin, "vrooli")
			src := filepath.Join(home, "candidate")
			fallback := filepath.Join(home, "libexec", "vrooli.previous")
			if err := os.WriteFile(dst, []byte("working-current"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Dir(fallback), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(fallback, []byte("retained-prior"), 0o755); err != nil {
				t.Fatal(err)
			}
			marker := filepath.Join(home, "unexpected-execution")
			body := []byte("wrong image")
			if mode == "script" {
				body = []byte("#!/bin/sh\ntouch '" + marker + "'\n")
			}
			if mode == "empty" {
				body = nil
			}
			if err := os.WriteFile(src, body, 0o755); err != nil {
				t.Fatal(err)
			}
			if mode == "wrong Go role" {
				executable, err := os.Executable()
				if err != nil {
					t.Fatal(err)
				}
				if _, err := copyRootCLIImage(executable, src, "", 0o755); err != nil {
					t.Fatal(err)
				}
				t.Setenv("VROOLI_ROOT_IDENTITY_TEST_FIXTURE", "valid")
				if err := os.WriteFile(filepath.Join(bin, ".root-identity-fixture"), []byte("valid"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "oversized" {
				if err := os.Truncate(src, rootCLIImageLimit+1); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "symlink" {
				if err := os.Remove(src); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(dst, src); err != nil {
					t.Fatal(err)
				}
			}
			if err := AtomicInstall(src, dst); err == nil {
				t.Fatal("unqualified root candidate installed")
			}
			for path, want := range map[string]string{dst: "working-current", fallback: "retained-prior"} {
				got, err := os.ReadFile(path)
				if err != nil || string(got) != want {
					t.Fatalf("rejected candidate changed %s: %q %v", path, got, err)
				}
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatal("wrong executable was invoked before its role was checked")
			}
			if _, err := os.Stat(dst + ".staged"); !os.IsNotExist(err) {
				t.Fatal("rejected candidate staging slot was not retired")
			}
		})
	}
}

func TestRootCLIChangedCanaryBytesCannotReplaceCurrentImage(t *testing.T) {
	dir := t.TempDir()
	source, destination := filepath.Join(dir, "stage"), filepath.Join(dir, "current")
	if err := os.WriteFile(source, []byte("changed after canary"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, []byte("prior"), 0o755); err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256([]byte("admitted bytes"))
	if _, err := copyRootCLIImage(source, destination, hex.EncodeToString(h[:]), 0o755); err == nil {
		t.Fatal("unprobed bytes installed")
	}
	data, err := os.ReadFile(destination)
	if err != nil || string(data) != "prior" {
		t.Fatal("failed checksum publication changed current image")
	}
}

func TestRootCLICandidateCurrentHostPreparationProbe(t *testing.T) {
	if os.Getenv("VROOLI_ROOT_CANDIDATE_PROBE") != "1" {
		t.Skip("set VROOLI_ROOT_CANDIDATE_PROBE=1 for installed root candidate preparation and side-effect-free identity proof")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(home, ".vrooli", "bin", "vrooli")
	targetHome := t.TempDir()
	bin := filepath.Join(targetHome, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(bin, "vrooli")
	if err := os.WriteFile(destination, []byte("prior fixture bytes"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := AtomicInstall(source, destination); err != nil {
		t.Fatal(err)
	}
	prior, err := os.ReadFile(filepath.Join(targetHome, "libexec", "vrooli.previous"))
	if err != nil || string(prior) != "prior fixture bytes" {
		t.Fatalf("prior snapshot not retained: %q %v", prior, err)
	}
	info, err := os.Stat(destination)
	if err != nil || info.Mode().Perm() != 0o755 {
		t.Fatal("published root CLI permission contract changed")
	}
	queryHome := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	query := exec.CommandContext(ctx, destination, "version", "--json")
	query.Dir = queryHome
	query.Env = []string{"HOME=" + queryHome, "XDG_STATE_HOME=" + filepath.Join(queryHome, "state"), "PATH=" + os.Getenv("PATH")}
	query.WaitDelay = time.Second
	out, err := query.Output()
	if err != nil {
		t.Fatalf("prepared root image identity query failed: %v", err)
	}
	var identity map[string]any
	if err := json.Unmarshal(out, &identity); err != nil {
		t.Fatal(err)
	}
	id, _ := identity["build_identity"].(string)
	if !rootCLIBuildIdentity.MatchString(id) {
		t.Fatal("actual root entry did not export its stamped build identity")
	}
	if digest, _ := identity["runtime_home_contract_sha256"].(string); !rootCLIBuildIdentity.MatchString(digest) {
		t.Fatal("actual root image has no compiled runtime-home authority identity")
	}
	entries, err := os.ReadDir(queryHome)
	if err != nil || len(entries) != 0 {
		t.Fatalf("identity probe initialized managed home state: %v %v", entries, err)
	}
	if _, err := os.Stat(destination + ".staged"); !os.IsNotExist(err) {
		t.Fatal("qualified candidate staging slot retained after installation")
	}
	t.Logf("actual root CLI admitted through frozen Go-role/native-target/stamped-identity canary; prior fixture bytes retained; version query left fresh HOME empty; build_identity=%s", id)
}

func TestRootCLIIdentityProbeRejectsUnstampedDuplicateAndUnboundedResponses(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"valid", "unstamped", "source-bound", "duplicate", "overflow", "invalid", "timeout"} {
		t.Run(mode, func(t *testing.T) {
			fixtureDirectory := t.TempDir()
			fixtureImage := filepath.Join(fixtureDirectory, "probe")
			if err := os.Link(executable, fixtureImage); err != nil {
				if _, err := copyRootCLIImage(executable, fixtureImage, "", 0o700); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(fixtureDirectory, ".root-identity-fixture"), []byte(mode), 0o600); err != nil {
				t.Fatal(err)
			}
			deadline := 3 * time.Second
			if mode == "timeout" {
				deadline = 150 * time.Millisecond
			}
			ctx, cancel := context.WithTimeout(context.Background(), deadline)
			defer cancel()
			identity, err := probeRootCLIIdentity(ctx, fixtureImage)
			if mode != "timeout" && ctx.Err() != nil {
				t.Fatalf("response-shape case was not exercised before deadline: %v", ctx.Err())
			}
			if mode == "valid" {
				if err != nil || identity.GetBuildIdentity() != strings.Repeat("a", 64) {
					t.Fatalf("valid artifact identity lost: %+v %v", identity, err)
				}
			} else if err == nil {
				t.Fatal("invalid root identity canary accepted")
			}
		})
	}
}
