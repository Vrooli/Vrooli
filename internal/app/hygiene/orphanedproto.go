package hygiene

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Scenario proto contracts do not live inside the scenario. `make generate`
// writes them into the SHARED packages/proto tree across six outputs, so
// deleting scenarios/<name>/ leaves all six behind with nothing linking them
// back. Three throwaway isolation probes plus two pilot surfaces left 274 files
// of residue this way, invisible until someone went looking.
//
// The residue is not inert: orphaned surfaces still appear in dependency graphs,
// proto-health sweeps, and codegen time for everyone, because packages/proto is
// a shared surface.
//
// Ownership is deliberately NOT an allowlist. A surface is legitimate if some
// directory owns it OR some code consumes it; the cross-cutting contracts
// (common, cli, measures, scenario-validation, architecture, dev-routing) have
// no owning directory but plenty of consumers, and templates own their surface
// from templates/scenarios/. That keeps the check self-maintaining -- a new
// shared contract needs no registration, and a genuinely dead surface cannot
// hide behind one.

// protoOwnerDirs are the directory roots that can own a proto surface.
var protoOwnerDirs = [][]string{
	{"scenarios"},
	{"packages"},
	{"resources"},
	{"templates", "scenarios"},
}

// protoConsumerScanExts are the source extensions worth scanning for imports.
var protoConsumerScanExts = map[string]struct{}{
	".go": {}, ".ts": {}, ".tsx": {}, ".js": {}, ".jsx": {}, ".mjs": {}, ".py": {},
}

// protoConsumerSkipDirs are trees that cannot hold a genuine consumer: the
// generated output itself (which always references its own surface) and
// installed/build directories.
var protoConsumerSkipDirs = map[string]struct{}{
	"node_modules": {}, "dist": {}, "build": {}, "coverage": {},
	"vendor": {}, ".git": {}, "bundle": {},
}

// checkOrphanedProtoSurfaces reports proto surfaces with no owner and no consumer.
func (s Service) checkOrphanedProtoSurfaces(report *Report) {
	root := report.Root
	schemasDir := filepath.Join(root, "packages", "proto", "schemas")

	entries, err := os.ReadDir(schemasDir)
	if err != nil {
		// No proto tree here; nothing to assert.
		report.addCheck("orphaned_proto_surfaces", true, SeverityInfo, "skipped: no packages/proto/schemas")
		return
	}

	var unowned []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if !protoSurfaceHasOwner(root, entry.Name()) {
			unowned = append(unowned, entry.Name())
		}
	}

	orphans := unowned
	if len(unowned) > 0 {
		consumed := protoSurfacesWithConsumers(root, unowned)
		orphans = orphans[:0]
		for _, name := range unowned {
			if _, ok := consumed[name]; !ok {
				orphans = append(orphans, name)
			}
		}
	}
	sort.Strings(orphans)

	if len(orphans) == 0 {
		report.addCheck("orphaned_proto_surfaces", true, SeverityInfo, "no orphaned proto surfaces")
		return
	}

	var locations []string
	for _, name := range orphans {
		locations = append(locations, protoSurfaceFootprint(name)...)
	}

	message := strconv.Itoa(len(orphans)) + " orphaned proto surface(s) with no owning directory and no consumer: " + strings.Join(orphans, ", ")
	report.addCheck("orphaned_proto_surfaces", false, SeverityWarning, message)
	report.addFinding(Finding{
		Severity:   SeverityWarning,
		Code:       "orphaned_proto_surface",
		Locations:  locations,
		Message:    message,
		Why:        "Scenario deletion does not reap packages/proto; the schemas plus all generated Go/TypeScript/Python output and the lock manifest survive the scenario. Because packages/proto is shared, the residue slows codegen and pollutes dependency graphs for every scenario.",
		Fixability: FixabilityGuided,
		NextActions: []Action{{
			Code:       "destroy_orphaned_proto_surface",
			Message:    "Remove the surface with the command that reaps the whole footprint, then regenerate.",
			Command:    "template-manager lifecycle destroy <surface> --proto-only && (cd packages/proto && make generate)",
			Fixability: FixabilityGuided,
		}},
	})
}

// protoSurfaceHasOwner reports whether a directory that could own the surface exists.
func protoSurfaceHasOwner(root, name string) bool {
	for _, parts := range protoOwnerDirs {
		candidate := filepath.Join(append([]string{root}, append(parts, name)...)...)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return true
		}
	}
	return false
}

// protoSurfaceFootprint returns every repo-relative path a surface occupies,
// covering both name forms (protoc-gen-python rewrites hyphens to underscores)
// and the lock manifest, which is a file beside the gen trees rather than a
// directory inside one.
func protoSurfaceFootprint(name string) []string {
	python := strings.ReplaceAll(name, "-", "_")
	gen := filepath.Join("packages", "proto", "gen")
	return []string{
		filepath.ToSlash(filepath.Join("packages", "proto", "schemas", name)),
		filepath.ToSlash(filepath.Join(gen, "go", name)),
		filepath.ToSlash(filepath.Join(gen, "typescript", name)),
		filepath.ToSlash(filepath.Join(gen, "typescript", "js", name)),
		filepath.ToSlash(filepath.Join(gen, "python", python)),
		filepath.ToSlash(filepath.Join(gen, "manifests", name+".lock.json")),
	}
}

// protoSurfacesWithConsumers scans source files once and returns which candidate
// surfaces are imported somewhere. Only surfaces that already failed the owner
// check are searched, so the candidate set is small.
func protoSurfacesWithConsumers(root string, candidates []string) map[string]struct{} {
	needles := make(map[string][]string, len(candidates))
	for _, name := range candidates {
		python := strings.ReplaceAll(name, "-", "_")
		needles[name] = []string{
			"proto/gen/go/" + name + "/",
			"proto/gen/typescript/" + name + "/",
			"proto/gen/python/" + python + "/",
			// Buf-style and bundler-style references to the same surface.
			"proto/" + name + "/v1",
			name + "/v1/",
		}
	}

	found := make(map[string]struct{}, len(candidates))
	genRoot := filepath.Join(root, "packages", "proto", "gen")
	schemasRoot := filepath.Join(root, "packages", "proto", "schemas")

	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // an unreadable subtree must not fail hygiene
		}
		if d.IsDir() {
			if _, skip := protoConsumerSkipDirs[d.Name()]; skip {
				return filepath.SkipDir
			}
			// Generated output and the schemas themselves always reference their
			// own surface; counting them would make every surface self-consuming.
			if path == genRoot || path == schemasRoot {
				return filepath.SkipDir
			}
			return nil
		}
		if _, ok := protoConsumerScanExts[strings.ToLower(filepath.Ext(path))]; !ok {
			return nil
		}
		if len(found) == len(candidates) {
			return filepath.SkipAll
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		text := string(data)
		for name, patterns := range needles {
			if _, already := found[name]; already {
				continue
			}
			for _, pattern := range patterns {
				if strings.Contains(text, pattern) {
					found[name] = struct{}{}
					break
				}
			}
		}
		return nil
	})
	return found
}
