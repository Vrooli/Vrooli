//go:build linux

package coreactivation

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	platform "github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
)

// Uses a governed actual CLI, frozen into an isolated home. No serving native
// unit, live registry, source tree or activation is modified by this fixture.
func TestRootCompatibilityCurrentHostFrozenImageAndEntrypointProbe(t *testing.T) {
	if os.Getenv("VROOLI_ROOT_COMPATIBILITY_PROBE") != "1" {
		t.Skip("set VROOLI_ROOT_COMPATIBILITY_PROBE=1 for the isolated actual-CLI compatibility drill")
	}
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	image := filepath.Join(root, ".vrooli", "build", "vrooli")
	if selected := os.Getenv("VROOLI_ROOT_COMPATIBILITY_IMAGE"); selected != "" {
		image = selected
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if err := ValidateRootExecutableRole(image); err != nil {
		t.Fatal(err)
	}
	identity, err := ProbeRootIdentity(ctx, image)
	if err != nil {
		t.Fatal(err)
	}
	if err := QualifyRollbackIdentities(identity, identity); err != nil {
		t.Fatal(err)
	}
	e, r, _, _ := fixture(t)
	e.qualifyFrozen = nil // exercise the production admission gate
	unit := []byte("isolated definition; never installed\n")
	unitSpec, _ := platform.CoreUnitByID(platform.CoreUnitRuntimeSupervisor)
	l := Layout{Home: r.Home, Executable: filepath.Join(r.Home, "bin", "vrooli"), UnitName: unitSpec.NativeName("linux"), UnitPath: platform.RuntimeSupervisorUnitPath("linux", r.Home), LogPath: filepath.Join(r.Home, "logs", "supervisor.log")}
	a := Assets{Store: e.Store}
	var d InputDigests
	err = e.Store.Prepare(ctx, func(p *Preparer) error {
		var err error
		d, err = p.Freeze(ctx, r.ID, l, InputSources{Candidate: image, Prior: image, Worker: image, CandidateUnit: unit, PriorUnit: unit})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	r.LayoutSHA, r.WorkerSHA, r.CandidateSHA, r.PriorSHA = d.Layout, d.Worker, d.Candidate, d.Prior
	r.CandidateUnitSHA, r.PriorUnitSHA = d.CandidateUnit, d.PriorUnit
	r.CandidateBuild, r.PriorBuild = identity.GetBuildIdentity(), identity.GetBuildIdentity()
	r.CandidateStateContract, r.PriorStateContract = identity.GetPersistentStateContractSha256(), identity.GetPersistentStateContractSha256()
	if err := a.QualifyFrozenCompatibility(ctx, r); err != nil {
		t.Fatal(err)
	}
	spoofed := r
	spoofed.PriorStateContract, spoofed.CandidateStateContract = strings.Repeat("9", 64), strings.Repeat("9", 64)
	if err := a.QualifyFrozenCompatibility(ctx, spoofed); err == nil {
		t.Fatal("caller-provided equal hashes substituted for compiled compatibility")
	}
	r, err = e.Begin(ctx, r, false)
	if err != nil {
		t.Fatal(err)
	}
	frozen, err := a.Image(ctx, r, "candidate")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(r.SourceRoot); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(r.Home, "source-unavailable")
	env := []string{"HOME=" + r.Home, "PATH=/usr/local/bin:/usr/bin:/bin", "VROOLI_SOURCE_ROOT=" + missing, "VROOLI_ROOT=" + missing, "VROOLI_RUNTIME_SUPERVISOR=on"}
	inventory := func() []string {
		t.Helper()
		var paths []string
		err := filepath.WalkDir(r.Home, func(path string, _ os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			paths = append(paths, path)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		return paths
	}
	before := inventory()
	for _, args := range [][]string{{"version", "--json"}, {"scenario", "list", "--json"}, {"runtime", "supervisor", "run"}} {
		cmd := exec.CommandContext(ctx, frozen, args...)
		cmd.Env, cmd.Dir = env, r.Home
		output, err := cmd.CombinedOutput()
		if args[0] == "version" {
			if err != nil || !strings.Contains(string(output), r.CandidateStateContract) {
				t.Fatalf("read-only identity failed: %v %s", err, output)
			}
		} else if err == nil || !strings.Contains(string(output), "candidate command must wait for native verification") {
			t.Fatalf("unverified command crossed the initialization fence: %v %v %s", args, err, output)
		}
	}
	if after := inventory(); !reflect.DeepEqual(before, after) {
		t.Fatal("candidate probes initialized user state before verification")
	}
}
