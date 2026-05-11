// Package layout derives all generated and hand-authored file paths
// for a temporal flow from its contract path, flow ID, and runtime
// language. There is exactly one convention: every codegen artifact
// lives under <baseDir>/generated/<folderName>/. Hand-authored files
// stay at <baseDir>.
package layout

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Language identifies the runtime emission target.
type Language string

const (
	LanguageGo         Language = "go"
	LanguageTypeScript Language = "typescript"
)

// Layout is the single source of truth for every path used by a flow.
// All fields are paths relative to the scenario root, using forward
// slashes.
type Layout struct {
	Language   Language
	FolderName string
	BaseDir    string

	ModelPath        string
	ArtifactPath     string
	RuntimePath      string
	ReplayHelperPath string
}

// Derive builds a Layout from the contract path (relative to root,
// forward slashes) and the runtime language. The flowId convention is
// "<domain>.<name>.<surface>"; the folder name comes from the middle
// dotted segments, lowercased, with dashes stripped.
func Derive(contractPath string, flowID string, language Language) (Layout, error) {
	folder, err := FolderName(flowID)
	if err != nil {
		return Layout{}, err
	}
	base := filepath.ToSlash(filepath.Dir(filepath.ToSlash(contractPath)))
	if base == "." || base == "" {
		return Layout{}, fmt.Errorf("contract path %q has no base directory", contractPath)
	}
	subdir := base + "/generated/" + folder
	var runtime, helper string
	switch language {
	case LanguageGo:
		runtime = subdir + "/runtime.go"
		helper = subdir + "/replay.go"
	case LanguageTypeScript:
		runtime = subdir + "/runtime.ts"
		helper = subdir + "/replay.helper.ts"
	default:
		return Layout{}, fmt.Errorf("unsupported language %q", language)
	}
	return Layout{
		Language:         language,
		FolderName:       folder,
		BaseDir:          base,
		ModelPath:        subdir + "/model.qnt",
		ArtifactPath:     subdir + "/artifact.json",
		RuntimePath:      runtime,
		ReplayHelperPath: helper,
	}, nil
}

// FolderName derives the canonical generated-subpackage name from a
// flowId of the form "<domain>.<name>.<surface>". It strips the first
// and last dotted segments, lowercases, and removes dashes/underscores.
// Examples:
//
//	notes.attachment-upload.ui  → "attachmentupload"
//	billing.refund-flow.api     → "refundflow"
//	core.send-message-v2.api    → "sendmessagev2"
//
// The result must match Go's lowercase no-separator package convention.
func FolderName(flowID string) (string, error) {
	parts := strings.Split(flowID, ".")
	if len(parts) < 3 {
		return "", fmt.Errorf("flowId %q must have at least three dotted segments", flowID)
	}
	middle := strings.Join(parts[1:len(parts)-1], "")
	middle = strings.ToLower(middle)
	var b strings.Builder
	for _, r := range middle {
		if r == '-' || r == '_' {
			continue
		}
		b.WriteRune(r)
	}
	out := b.String()
	if out == "" {
		return "", fmt.Errorf("flowId %q produced empty folder name", flowID)
	}
	return out, nil
}

// SubpackageImportPath builds the Go import path of the generated
// subpackage for a contract that lives at api/<...>. The result is
// suitable for use in a Go test file's import block. Returns the
// {{SCENARIO_ID}}-anchored form expected by the template.
func SubpackageImportPath(layout Layout) string {
	dir := strings.TrimPrefix(layout.BaseDir+"/generated/"+layout.FolderName, "api/")
	return "{{SCENARIO_ID}}/" + dir
}
