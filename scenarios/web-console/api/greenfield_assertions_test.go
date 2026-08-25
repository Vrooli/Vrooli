package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func walkModuleGoFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && (entry.Name() == "node_modules" || entry.Name() == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".go" && !strings.HasSuffix(path, "_test.go") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func findEnclosingFunc(lines []string, lineIdx int) string {
	re := regexp.MustCompile(`^func\s+(?:\([^)]*\)\s+)?([A-Za-z_][A-Za-z0-9_]*)`)
	for i := lineIdx; i >= 0; i-- {
		if m := re.FindStringSubmatch(lines[i]); m != nil {
			return m[1]
		}
	}
	return ""
}

func TestGreenfield_NoRawSetSizeOutsideGatedPaths(t *testing.T) {
	files, err := walkModuleGoFiles("session")
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`s\.pty\.SetSize\(|creackpty\.Setsize\(`)
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if !re.MatchString(line) {
				continue
			}
			switch enclosing := findEnclosingFunc(lines, i); enclosing {
			case "maybeSIGWINCHRecovery", "Resize", "applyResizeLocked":
			default:
				t.Errorf("%s:%d SetSize outside gated path (enclosing func=%q)", file, i+1, enclosing)
			}
		}
	}
}

func TestGreenfield_NoRawPtmxWriteOutsidePTYFiles(t *testing.T) {
	allowed := map[string]bool{"pty.go": true, "pty_tmux.go": true}
	re := regexp.MustCompile(`\bptmx\.Write\(`)
	files, err := walkModuleGoFiles(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if allowed[filepath.ToSlash(file)] {
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if re.Match(data) {
			t.Errorf("%s calls ptmx.Write; route through PTY.WriteInput", file)
		}
	}
}

func TestGreenfield_PTYInterfaceHasNoLegacyWrite(t *testing.T) {
	data, err := os.ReadFile("../../../packages/session-core/session.go")
	if err != nil {
		t.Fatal(err)
	}
	if regexp.MustCompile(`\n\s*Write\(p \[\]byte\) \(int, error\)`).Match(data) {
		t.Fatal("session-core PTY interface still declares legacy Write")
	}
}

func TestGreenfield_AudioToolsDependencyIsLazyDegraded(t *testing.T) {
	type dep struct {
		Required bool   `json:"required"`
		Policy   string `json:"startup_policy"`
		Degraded string `json:"degraded_behavior"`
	}
	var service struct {
		Dependencies struct {
			Scenarios map[string]dep `json:"scenarios"`
		} `json:"dependencies"`
	}
	raw, err := os.ReadFile("../.vrooli/service.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &service); err != nil {
		t.Fatal(err)
	}
	audio, ok := service.Dependencies.Scenarios["audio-tools"]
	if !ok || audio.Required || audio.Policy != "try_start" || !strings.Contains(strings.ToLower(audio.Degraded), "terminal workspace boots") {
		t.Fatalf("audio-tools dependency is not lazy-degraded: %+v", audio)
	}
}

func TestGreenfield_InternalAudioDomainsDoNotImportHandlers(t *testing.T) {
	re := regexp.MustCompile(`"web-console/handlers/[^\"]+"`)
	for _, dir := range []string{"internal/audio", "internal/audioports"} {
		files, err := walkModuleGoFiles(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range files {
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if re.Match(data) {
				t.Errorf("%s imports a handler package", file)
			}
		}
	}
}

func TestGreenfield_OrchestrationRoutesThroughAudioPorts(t *testing.T) {
	forbidden := []*regexp.Regexp{
		regexp.MustCompile(`\binttts\.(?:NormalizeTextForSpeech|SplitIntoSpeechParagraphs)\(`),
		regexp.MustCompile(`\bs\.voiceService\.Transcribe\b`),
		regexp.MustCompile(`internal/voice\.Service\.Transcribe`),
	}
	files, err := walkModuleGoFiles(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if filepath.ToSlash(file) == "main.go" {
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, pattern := range forbidden {
			if pattern.Match(data) {
				t.Errorf("%s bypasses the audioports seam: %s", file, pattern)
			}
		}
	}
}
