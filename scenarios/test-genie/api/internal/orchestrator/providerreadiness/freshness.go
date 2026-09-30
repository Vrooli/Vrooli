package providerreadiness

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// StalenessVerdict explains why a running provider no longer matches its source.
// A zero verdict means "not stale", which is also what every error path yields:
// the check fails open on purpose. A false positive here restarts a healthy
// provider, and on a branch where shared packages change often that cost
// compounds fast; a false negative only preserves the status quo.
type StalenessVerdict struct {
	Stale  bool
	Reason string
	Detail string
	// Class says what kind of change caused this, which is what makes the
	// verdict actionable: "a shared package moved" and "this provider's own
	// code changed" have very different consequences for the results a phase
	// is about to produce.
	Class StalenessClass
	// File is the first offending input, kept so an operator can see the actual
	// cause rather than only its category.
	File string
}

// StalenessClass categorizes why a provider fell out of date, ordered roughly by
// how much it should worry the reader of a phase result.
type StalenessClass string

const (
	// StalenessOwnCode means the provider's own source changed. Its findings are
	// the ones most likely to be actively wrong, because the code being asked
	// for a verdict is not the code that shipped the verdict.
	StalenessOwnCode StalenessClass = "own_code"
	// StalenessSharedPackage means a package the provider compiles changed.
	// Usually incidental to the provider's own judgment, and the common case on
	// a branch where several agents work at once.
	StalenessSharedPackage StalenessClass = "shared_package"
	// StalenessDependency means a module manifest moved (go.mod / go.sum).
	StalenessDependency StalenessClass = "dependency"
	// StalenessToolchain means a non-file build input changed (Go version, arch,
	// cgo). Every provider sees this at once.
	StalenessToolchain StalenessClass = "toolchain"
	// StalenessRebuilt means the binary was rebuilt but this process was never
	// restarted, so it is serving superseded code regardless of source state.
	StalenessRebuilt StalenessClass = "rebuilt_not_restarted"
	// StalenessOther covers offenders that fit no sharper category.
	StalenessOther StalenessClass = "other"
)

// Describe renders a short operator-facing explanation. It names the category
// and one concrete file rather than listing everything that changed, which for a
// shared-package edit would be thousands of paths and no more informative.
func (v StalenessVerdict) Describe() string {
	if !v.Stale {
		return ""
	}
	switch v.Class {
	case StalenessOwnCode:
		return fmt.Sprintf("this provider's own code changed since its binary was built (%s)", v.File)
	case StalenessSharedPackage:
		return fmt.Sprintf("a shared package it compiles changed (%s)", v.File)
	case StalenessDependency:
		return fmt.Sprintf("its dependency manifest changed (%s)", v.File)
	case StalenessToolchain:
		return fmt.Sprintf("the build toolchain or environment changed (%s)", v.File)
	case StalenessRebuilt:
		return "its binary was rebuilt after this process started, so the running process is superseded"
	default:
		if v.File != "" {
			return fmt.Sprintf("a build input changed (%s)", v.File)
		}
		return "a build input changed"
	}
}

// classifyOffender maps an offending input path to a staleness class. The path
// is already the signal: the freshness layer records inputs relative to the repo
// root, so where a change landed says what kind of change it was.
func classifyOffender(provider, reason, file string) StalenessClass {
	file = strings.TrimSpace(filepath.ToSlash(file))
	if file == "" {
		return StalenessOther
	}
	// A key-input change reports the key name, not a path.
	if reason == "build input changed" && !strings.Contains(file, "/") {
		return StalenessToolchain
	}
	base := path.Base(file)
	if base == "go.mod" || base == "go.sum" {
		return StalenessDependency
	}
	if strings.HasPrefix(file, "scenarios/"+provider+"/") {
		return StalenessOwnCode
	}
	if strings.HasPrefix(file, "packages/") {
		return StalenessSharedPackage
	}
	if strings.HasPrefix(file, "scenarios/") {
		// Another scenario's code compiled into this provider is effectively a
		// shared dependency from this provider's point of view.
		return StalenessSharedPackage
	}
	return StalenessOther
}

// providerAPIDir is where a provider scenario's binary and freshness manifest
// live. Kept in one place so the two checks below cannot disagree.
func providerAPIDir(repoRoot, provider string) string {
	return filepath.Join(repoRoot, "scenarios", provider, "api")
}

// findFreshnessManifest locates the freshness stamp written beside a provider's
// binary. Returns ok=false when the provider has no manifest, which is treated
// as "cannot prove staleness" rather than "stale".
func findFreshnessManifest(repoRoot, provider string) (path string, ok bool) {
	matches, err := filepath.Glob(filepath.Join(providerAPIDir(repoRoot, provider), "*"+cliutil.FreshnessManifestSuffix))
	if err != nil || len(matches) == 0 {
		return "", false
	}
	return matches[0], true
}

// EvaluateProviderStaleness decides whether a running provider is serving code
// that no longer matches the repository. It answers two distinct questions:
//
//  1. Was the binary rebuilt after this process started? The provider reports
//     the manifest digest it read at startup; a different digest on disk means
//     someone rebuilt without restarting, so the live process is superseded.
//
//  2. Has the source changed since the binary was built? This is the ordinary
//     freshness comparison, evaluated against the same manifest the provider's
//     own preflight would use.
//
// Both are exact comparisons over content digests. Neither consults the git
// state: commits do not change file content, and a design keyed to HEAD would
// mark every provider stale on every commit.
func EvaluateProviderStaleness(repoRoot, provider, reportedDigest string) StalenessVerdict {
	repoRoot = strings.TrimSpace(repoRoot)
	provider = strings.TrimSpace(provider)
	if repoRoot == "" || provider == "" {
		return StalenessVerdict{}
	}
	manifestPath, ok := findFreshnessManifest(repoRoot, provider)
	if !ok {
		return StalenessVerdict{}
	}
	manifest, ok, err := cliutil.ReadFreshnessManifest(manifestPath)
	if err != nil || !ok {
		return StalenessVerdict{}
	}

	// (1) Rebuilt but never restarted. Only decidable when the provider actually
	// reported a digest; an empty report means "unknown", never "stale".
	onDisk := strings.TrimSpace(manifest.Digest)
	if reported := strings.TrimSpace(reportedDigest); reported != "" && onDisk != "" && reported != onDisk {
		return StalenessVerdict{
			Stale:  true,
			Class:  StalenessRebuilt,
			Reason: "provider binary was rebuilt after this process started",
			Detail: fmt.Sprintf("running digest %s, on-disk digest %s", shortDigest(reported), shortDigest(onDisk)),
		}
	}

	// (2) Source changed since the binary was built.
	//
	// The provider's OWN tree is evaluated first, on its own. The underlying
	// comparison stops at the first offending path in alphabetical order, so a
	// change under packages/ would otherwise mask a simultaneous change to the
	// provider's own code — and that misclassification matters, because callers
	// treat an own-code change as more urgent than shared-package churn.
	// Checking the smaller own-tree subset first is also cheaper in the common
	// case, since it usually holds a handful of files.
	if v, ok := evaluateSubset(repoRoot, provider, manifestPath, manifest, ownTreePrefix(provider)); ok && v.Class == StalenessOwnCode {
		return v
	}
	if v, ok := evaluateSubset(repoRoot, provider, manifestPath, manifest, ""); ok {
		return v
	}
	return StalenessVerdict{}
}

func ownTreePrefix(provider string) string { return "scenarios/" + provider + "/" }

// evaluateSubset runs the freshness comparison over the manifest entries whose
// path starts with prefix (empty means every entry). The manifest is filtered to
// match, otherwise the excluded entries would be reported as removed inputs.
func evaluateSubset(repoRoot, provider, manifestPath string, manifest cliutil.FreshnessManifest, prefix string) (StalenessVerdict, bool) {
	subset := manifest
	subset.Files = nil
	inputs := make([]string, 0, len(manifest.Files))
	for _, f := range manifest.Files {
		if prefix != "" && !strings.HasPrefix(filepath.ToSlash(f.Rel), prefix) {
			continue
		}
		subset.Files = append(subset.Files, f)
		inputs = append(inputs, f.Rel)
	}
	if len(inputs) == 0 {
		return StalenessVerdict{}, false
	}
	binary := strings.TrimSuffix(manifestPath, cliutil.FreshnessManifestSuffix)
	spec := cliutil.FreshnessSpec{
		SourceRoot:   providerAPIDir(repoRoot, provider),
		ContextRoot:  repoRoot,
		Inputs:       inputs,
		SkipFiles:    []string{filepath.Base(binary)},
		SkipSuffixes: []string{"_test.go", cliutil.FreshnessManifestSuffix},
	}
	verdict, err := cliutil.EvaluateFreshness(spec, subset, subset.KeyInputs)
	if err != nil || !verdict.Stale {
		// An evaluation error is not evidence of staleness. Fail open.
		return StalenessVerdict{}, false
	}
	return StalenessVerdict{
		Stale:  true,
		Class:  classifyOffender(provider, verdict.Reason, verdict.File),
		Reason: "provider source changed since its binary was built",
		Detail: strings.TrimSpace(verdict.Reason + " " + verdict.File),
		File:   verdict.File,
	}, true
}

func shortDigest(d string) string {
	if len(d) > 12 {
		return d[:12]
	}
	return d
}
