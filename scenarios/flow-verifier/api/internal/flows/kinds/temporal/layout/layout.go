// Package layout derives all generated and hand-authored file paths
// for a temporal flow from its contract path and runtime language.
//
// Convention (schema v6):
//
//	<feature-dir>/flow/
//	  flow.json                ← the contract (BaseDir is "flow/")
//	  transition.{ts,go}       ← hand-authored wrapper
//	  fixtures.ts              ← TS only: hand-authored fixtures
//	  flow.test.{ts,go}        ← hand-authored thin replay (lint-enforced)
//	  generated/               ← codegen output; package name is "generated"
//	    model.qnt
//	    artifact.json
//	    runtime.{ts,go}
//	    replay.{helper.ts,go}
//
// BaseDir is the flow directory itself (the directory that contains
// flow.json). All paths are relative to the scenario root and use
// forward slashes.
package layout

import (
	"fmt"
	"path/filepath"
)

// Language identifies the runtime emission target.
type Language string

const (
	LanguageGo         Language = "go"
	LanguageTypeScript Language = "typescript"
)

// FlowDirName is the conventional directory name every flow lives in.
const FlowDirName = "flow"

// GeneratedDirName is the conventional name of the generated
// subdirectory; it is also the Go package name of the runtime+replay
// helpers emitted there.
const GeneratedDirName = "generated"

// Layout is the single source of truth for every path used by a flow.
// All fields are paths relative to the scenario root, using forward
// slashes.
type Layout struct {
	Language Language
	// BaseDir is the flow directory itself (parent of flow.json).
	BaseDir string

	ModelPath        string
	ArtifactPath     string
	RuntimePath      string
	ReplayHelperPath string
	TransitionPath   string
	FixturesPath     string
	TestPath         string
}

// Derive builds a Layout from the contract path (relative to root,
// forward slashes) and the runtime language. The contract must live in
// a directory named "flow"; the directory containing flow.json is the
// BaseDir.
func Derive(contractPath string, language Language) (Layout, error) {
	base := filepath.ToSlash(filepath.Dir(filepath.ToSlash(contractPath)))
	if base == "." || base == "" {
		return Layout{}, fmt.Errorf("contract path %q has no base directory", contractPath)
	}
	if filepath.Base(base) != FlowDirName {
		return Layout{}, fmt.Errorf("contract %q must live in a directory named %q; got %q. Run `temporal-model new` to scaffold a new flow", contractPath, FlowDirName, filepath.Base(base))
	}
	subdir := base + "/" + GeneratedDirName
	var runtime, helper, transition, fixtures, test string
	switch language {
	case LanguageGo:
		runtime = subdir + "/runtime.go"
		helper = subdir + "/replay.go"
		transition = base + "/transition.go"
		test = base + "/flow_test.go"
	case LanguageTypeScript:
		runtime = subdir + "/runtime.ts"
		helper = subdir + "/replay.helper.ts"
		transition = base + "/transition.ts"
		fixtures = base + "/fixtures.ts"
		test = base + "/flow.test.ts"
	default:
		return Layout{}, fmt.Errorf("unsupported language %q", language)
	}
	return Layout{
		Language:         language,
		BaseDir:          base,
		ModelPath:        subdir + "/model.qnt",
		ArtifactPath:     subdir + "/artifact.json",
		RuntimePath:      runtime,
		ReplayHelperPath: helper,
		TransitionPath:   transition,
		FixturesPath:     fixtures,
		TestPath:         test,
	}, nil
}

// SubpackageImportPath builds the Go import path of the generated
// subpackage for a contract that lives at api/<...>. The result is
// suitable for use in a Go test file's import block. Returns the
// {{SCENARIO_ID}}-anchored form expected by the template.
func SubpackageImportPath(layout Layout) string {
	dir := layout.BaseDir + "/" + GeneratedDirName
	if len(dir) > 4 && dir[:4] == "api/" {
		dir = dir[4:]
	}
	return "{{SCENARIO_ID}}/" + dir
}
