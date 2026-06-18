package validation

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/packages/proto/genmanifest"
)

const envSkipGenSync = "PROTO_HEALTH_SKIP_GEN_SYNC"

type ManifestVerifier struct {
	repoRoot  string
	protoRoot string
}

func NewManifestVerifier(repoRoot string) *ManifestVerifier {
	return &ManifestVerifier{
		repoRoot:  repoRoot,
		protoRoot: filepath.Join(repoRoot, "packages", "proto"),
	}
}

func (v *ManifestVerifier) CheckScenario(_ context.Context, scenario string) (GenSyncStatus, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return GenSyncStatus{}, fmt.Errorf("scenario name is required")
	}
	if truthy(os.Getenv(envSkipGenSync)) {
		return GenSyncStatus{InSync: true, Skipped: true, SkipMessage: envSkipGenSync + " is set"}, nil
	}

	opts := genmanifest.Options{RepoRoot: v.repoRoot, ProtoRoot: v.protoRoot}
	manifest, err := genmanifest.LoadManifest(v.protoRoot, scenario)
	if err != nil {
		if os.IsNotExist(err) {
			return GenSyncStatus{
				ManifestMissing: true,
				Detail:          "generation manifest is missing",
				Drift:           []string{filepath.ToSlash(filepath.Join("packages", "proto", "gen", "manifests", scenario+".lock.json"))},
			}, nil
		}
		return GenSyncStatus{}, fmt.Errorf("load generation manifest: %w", err)
	}

	var drift []string
	inputFiles, inputDigest, err := genmanifest.InputClosure(opts, scenario)
	if err != nil {
		return GenSyncStatus{}, err
	}
	if inputDigest != manifest.InputDigest {
		drift = append(drift, inputFiles...)
	}

	actualOutputs, err := genmanifest.OutputDigests(opts, scenario)
	if err != nil {
		return GenSyncStatus{}, err
	}
	for rel, want := range manifest.Outputs {
		got, ok := actualOutputs[rel]
		if !ok || got != want {
			drift = append(drift, rel)
		}
	}
	for rel := range actualOutputs {
		if _, ok := manifest.Outputs[rel]; !ok {
			drift = append(drift, rel)
		}
	}

	toolchain, err := genmanifest.ToolchainFingerprint(opts)
	if err != nil {
		return GenSyncStatus{}, err
	}
	toolchainDrift := !sameToolchain(toolchain, manifest.Toolchain)

	drift = uniqueSortedPackageProtoPaths(drift)
	if len(drift) == 0 && !toolchainDrift {
		return GenSyncStatus{InSync: true}, nil
	}
	return GenSyncStatus{
		InSync:         len(drift) == 0,
		Drift:          drift,
		Detail:         driftDetail(drift, toolchainDrift),
		ToolchainDrift: toolchainDrift,
	}, nil
}

func truthy(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "t", "true", "y", "yes", "on":
		return true
	default:
		return false
	}
}

func sameToolchain(a, b genmanifest.Toolchain) bool {
	if a.Buf != b.Buf || a.BufGenYAMLDigest != b.BufGenYAMLDigest || len(a.Plugins) != len(b.Plugins) {
		return false
	}
	for name, version := range a.Plugins {
		if b.Plugins[name] != version {
			return false
		}
	}
	return true
}

func uniqueSortedPackageProtoPaths(paths []string) []string {
	seen := map[string]bool{}
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		path = strings.TrimPrefix(filepath.ToSlash(path), "packages/proto/")
		seen[filepath.ToSlash(filepath.Join("packages", "proto", path))] = true
	}
	out := make([]string, 0, len(seen))
	for path := range seen {
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

func driftDetail(drift []string, toolchainDrift bool) string {
	switch {
	case len(drift) > 0 && toolchainDrift:
		return fmt.Sprintf("%d manifest-tracked file(s) drifted; toolchain pins also changed", len(drift))
	case len(drift) > 0:
		return fmt.Sprintf("%d manifest-tracked file(s) drifted", len(drift))
	case toolchainDrift:
		return "toolchain pins changed since the generation manifest was written"
	default:
		return ""
	}
}
