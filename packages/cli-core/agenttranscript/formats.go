package agenttranscript

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// Format describes native files and invocation shape for one coding harness.
// Keep it data-only so discovery and resume surfaces share one contract.
type Format struct {
	Harness          string   `json:"harness"`
	Transcript       string   `json:"transcript"`
	MessageDialect   string   `json:"message_dialect,omitempty"`
	HomeRoots        []string `json:"home_roots,omitempty"`
	RunRoots         []string `json:"run_roots,omitempty"`
	WebConsoleRoots  []string `json:"web_console_roots,omitempty"`
	ArchiveFileName  string   `json:"archive_file_name,omitempty"`
	ArchivePrefix    string   `json:"archive_prefix,omitempty"`
	ArchiveSuffix    string   `json:"archive_suffix,omitempty"`
	ArchiveParent    string   `json:"archive_parent,omitempty"`
	ArchiveParents   []string `json:"archive_parents,omitempty"`
	ArchiveID        string   `json:"archive_id,omitempty"`
	ArchiveOnly      bool     `json:"archive_only,omitempty"`
	ArchiveSnapshot  bool     `json:"archive_snapshot,omitempty"`
	ArchiveMaxBytes  int64    `json:"archive_max_bytes,omitempty"`
	ResumeFlag       string   `json:"resume_flag"`
	ResumeCommand    string   `json:"resume_command,omitempty"`
	ResumeBinary     string   `json:"resume_binary,omitempty"`
	ResumePromptFlag string   `json:"resume_prompt_flag,omitempty"`
	// ResumeLayout describes the argument ordering around a native prompt and
	// selected session. Empty keeps the session-first layout.
	ResumeLayout string `json:"resume_layout,omitempty"`
	// ResumeSelection states which value follows ResumeFlag. Empty preserves
	// the existing harness-specific native session ID behavior.
	ResumeSelection string `json:"resume_selection,omitempty"`
}

// ResumeInvocation is an argument-vector-safe native resume command. Command
// and Args are separate so callers never need to interpolate session IDs or
// prompts into shell text.
type ResumeInvocation struct {
	Command string
	Args    []string
}

// BuildNativeResumeInvocation builds the data-backed native resume invocation
// for a selected session. Only descriptors with an explicit ResumeBinary are
// executable through this helper; other harnesses keep their caller-specific
// argument policies.
func BuildNativeResumeInvocation(harness, sessionID, prompt string) (ResumeInvocation, error) {
	format, ok := FormatFor(harness)
	if !ok || strings.TrimSpace(format.ResumeBinary) == "" || strings.TrimSpace(format.ResumeFlag) == "" {
		return ResumeInvocation{}, fmt.Errorf("native resume invocation is not described for %q", harness)
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return ResumeInvocation{}, fmt.Errorf("native resume for %q requires a selected session ID", harness)
	}
	var args []string
	switch format.ResumeLayout {
	case "", "session-first":
		args = []string{format.ResumeFlag, sessionID}
		if prompt != "" {
			if strings.TrimSpace(format.ResumePromptFlag) == "" {
				return ResumeInvocation{}, fmt.Errorf("native resume prompt is not described for %q", harness)
			}
			args = append(args, format.ResumePromptFlag, prompt)
		}
	case "prompt-first":
		if strings.TrimSpace(format.ResumePromptFlag) == "" {
			return ResumeInvocation{}, fmt.Errorf("native resume prompt is not described for %q", harness)
		}
		args = []string{format.ResumePromptFlag, prompt, format.ResumeFlag, sessionID}
	default:
		return ResumeInvocation{}, fmt.Errorf("unsupported native resume layout %q for %q", format.ResumeLayout, harness)
	}
	return ResumeInvocation{Command: format.ResumeBinary, Args: args}, nil
}

//go:embed format-definitions.json
var formatDefinitions []byte

var formats = loadFormats()

func loadFormats() map[string]Format {
	var definitions []Format
	if err := json.Unmarshal(formatDefinitions, &definitions); err != nil {
		panic(fmt.Errorf("decode transcript format definitions: %w", err))
	}
	result := make(map[string]Format, len(definitions))
	for _, definition := range definitions {
		if definition.Harness == "" || definition.Transcript == "" {
			panic("transcript format definition needs harness and transcript")
		}
		if _, exists := result[definition.Harness]; exists {
			panic("duplicate transcript format definition: " + definition.Harness)
		}
		result[definition.Harness] = definition
	}
	return result
}

// Formats returns a copy of the supported native format descriptors.
func Formats() []Format {
	out := make([]Format, 0, len(formats))
	for _, format := range formats {
		out = append(out, format)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Harness < out[j].Harness })
	return out
}

// FormatFor returns the shared native format descriptor for harness.
func FormatFor(harness string) (Format, bool) {
	f, ok := formats[strings.ToLower(strings.TrimSpace(harness))]
	return f, ok
}

// RequiresNativeArchive reports whether discovery should reject ordinary
// transcript-name matches for a harness with one exact native archive shape.
func RequiresNativeArchive(harness string) bool {
	format, ok := FormatFor(harness)
	return ok && format.ArchiveOnly
}

// HomeRootPaths resolves the descriptor's home-root-relative paths under a
// caller-selected home directory.
func HomeRootPaths(home, harness string) []string {
	format, ok := FormatFor(harness)
	if !ok {
		return nil
	}
	paths := make([]string, 0, len(format.HomeRoots))
	for _, root := range format.HomeRoots {
		paths = append(paths, filepath.Join(home, filepath.FromSlash(root)))
	}
	return paths
}
