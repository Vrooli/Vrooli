package targetpack

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"structure-health/internal/rules"
)

// schemaScanExcludedDirs are trees whose schema files are not authoritative
// declarations: build and dependency outputs, and evidence directories, which
// hold captured byte-for-byte snapshots (pre-images, prewrite bytes) of the
// schema they recorded rather than forks of it.
var schemaScanExcludedDirs = map[string]struct{}{
	".git": {}, ".cache": {}, "build": {}, "coverage": {}, "dist": {}, "evidence": {}, "node_modules": {}, "vendor": {},
}

// schemaScanExcludedPaths are root-relative evidence areas that must not be
// confused with authoritative source trees. `.vrooli/work` retains delivery
// snapshots, including source-before copies of schemas that are still owned by
// their original paths.
var schemaScanExcludedPaths = map[string]struct{}{
	filepath.Join(".vrooli", "work"): {},
}

func projectSchemaIDRules(root string) []rules.Finding {
	byID := map[string][]string{}
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if entry.IsDir() {
			if path != root {
				if relative, err := filepath.Rel(root, path); err == nil {
					if _, excluded := schemaScanExcludedPaths[filepath.Clean(relative)]; excluded {
						return filepath.SkipDir
					}
				}
				if _, excluded := schemaScanExcludedDirs[entry.Name()]; excluded {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !isJSONSchemaPath(path) {
			return nil
		}
		raw, err := os.ReadFile(path) // #nosec G304 -- path comes from a walk rooted at the validated repository root.
		if err != nil {
			return nil
		}
		var header struct {
			ID string `json:"$id"`
		}
		if json.Unmarshal(raw, &header) != nil {
			return nil
		}
		id := strings.TrimSpace(header.ID)
		if id == "" {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		byID[id] = append(byID[id], filepath.ToSlash(relative))
		return nil
	})

	ids := make([]string, 0, len(byID))
	for id, paths := range byID {
		if len(paths) > 1 {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	findings := make([]rules.Finding, 0, len(ids))
	for _, id := range ids {
		paths := byID[id]
		sort.Strings(paths)
		findings = append(findings, finding(
			"REPO_SCHEMA_ID_UNIQUE",
			"error",
			fmt.Sprintf("schema $id %q is declared by %s", id, strings.Join(paths, ", ")),
			paths[0],
			"Keep one authoritative schema document for this $id and remove the forks.",
		))
	}
	return findings
}

func isJSONSchemaPath(path string) bool {
	slashed := filepath.ToSlash(path)
	return strings.HasSuffix(slashed, ".schema.json") || strings.Contains(slashed, "/schemas/") && strings.HasSuffix(slashed, ".json")
}
