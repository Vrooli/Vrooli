package components

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type versionShapePolicy struct {
	Enforcement string    `json:"enforcement"`
	Cutoff      time.Time `json:"cutoff"`
}

// ValidateVersionShape is the shared policy used by the authoring check and
// the catalog gate. It deliberately reports paths relative to the version
// directory so an author can fix the exact missing or forbidden artifact.
func ValidateVersionShape(root, versionDir, assetName string, newOnly bool) ([]string, error) {
	policyBytes, err := os.ReadFile(filepath.Join(root, "scenarios", "react-component-library", "catalog", "version-shape.json"))
	if err != nil {
		// Authoring tests and embedded consumers can provide a deliberately
		// minimal source root. The catalog gate owns enforcement when the
		// repository policy is present; an absent policy is not a malformed
		// version directory.
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var policy versionShapePolicy
	if err := json.Unmarshal(policyBytes, &policy); err != nil {
		return nil, fmt.Errorf("parse version shape policy: %w", err)
	}
	if newOnly {
		info, statErr := os.Stat(versionDir)
		if statErr != nil {
			return nil, statErr
		}
		if !policy.Cutoff.IsZero() && info.ModTime().Before(policy.Cutoff) {
			return nil, nil
		}
	}
	entries, err := os.ReadDir(versionDir)
	if err != nil {
		return nil, err
	}
	allowed := map[string]bool{
		"story.tsx": true, "story.json": true, "dependencies.json": true,
		assetName + ".ts": true, assetName + ".tsx": true,
		assetName + ".css": true, assetName + ".strings.ts": true,
	}
	var problems []string
	hasEntry := false
	hasStorySource, hasStoryContract, hasLock := false, false, false
	for _, entry := range entries {
		if entry.IsDir() {
			problems = append(problems, "./"+entry.Name()+": subdirectories are forbidden")
			continue
		}
		name := entry.Name()
		if !allowed[name] {
			problems = append(problems, "./"+name+": file is not part of the canonical version shape")
			continue
		}
		switch name {
		case "story.tsx":
			hasStorySource = true
		case "story.json":
			hasStoryContract = true
		case "dependencies.json":
			hasLock = true
		default:
			if name == assetName+".ts" || name == assetName+".tsx" {
				hasEntry = true
			}
		}
	}
	if !hasEntry {
		problems = append(problems, "./"+assetName+".tsx: required entry file is missing")
	}
	if !hasStorySource {
		problems = append(problems, "./story.tsx: required authored story source is missing")
	}
	if !hasStoryContract {
		problems = append(problems, "./story.json: required generated story contract is missing")
	}
	if !hasLock {
		problems = append(problems, "./dependencies.json: required generated dependency lock is missing")
	}
	sort.Strings(problems)
	return problems, nil
}

func FormatVersionShapeFindings(problems []string) string { return strings.Join(problems, "; ") }
