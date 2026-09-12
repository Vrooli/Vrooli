package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
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

// TestGreenfield_PTYTestsDeclarePlatformRequirement keeps host-dependent test
// coverage honest. A test that reaches a real PTY or tmux command must declare
// its platform boundary at the test entry point; otherwise a cross-platform
// run can report a misleading implementation failure instead of a typed skip.
func TestGreenfield_PTYTestsDeclarePlatformRequirement(t *testing.T) {
	// Walk from the API module root so the check is recursive and remains
	// scoped to this scenario's tests on every supported Go host.
	var files []string
	err := filepath.WalkDir(".", func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != "." && (entry.Name() == "node_modules" || entry.Name() == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, "_test.go") && filepath.Base(path) != "greenfield_assertions_test.go" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	needGuard := regexp.MustCompile(`\b(?:tmuxCmd|tmuxAttach|defaultPTYFactory|tmuxPTYFactory)\s*\(|\bcreackpty\b`)
	hasGuard := regexp.MustCompile(`\brequire(?:LocalPTY|Tmux|UnixTools)\s*\(`)
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, file, data, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, declaration := range parsed.Decls {
			fn, ok := declaration.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Body == nil || fn.Name == nil || !strings.HasPrefix(fn.Name.Name, "Test") {
				continue
			}
			start := fset.Position(fn.Pos()).Offset
			end := fset.Position(fn.End()).Offset
			body := string(data[start:end])
			if needGuard.MatchString(body) && !hasGuard.MatchString(body) {
				t.Errorf("%s:%s reaches a PTY/tmux seam without a require helper", file, fn.Name.Name)
			}
		}
	}
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
			case "Resize", "applyResizeLocked":
			default:
				t.Errorf("%s:%d SetSize outside gated path (enclosing func=%q)", file, i+1, enclosing)
			}
		}
	}
}

func TestGreenfield_NoSessionLockHeldAcrossExec(t *testing.T) {
	files, err := walkModuleGoFiles("session")
	if err != nil {
		t.Fatal(err)
	}
	lockCall := regexp.MustCompile(`s\.(?:emuMu|clientsMu|ptyMu)\.(?:Lock|RLock)\(\)`)
	unlockCall := regexp.MustCompile(`s\.(?:emuMu|clientsMu|ptyMu)\.(?:Unlock|RUnlock)\(\)`)
	backendCall := regexp.MustCompile(`\.(?:Read|WriteInput|SetSize|Close|Kill|ExitCode|ProbeReady|CurrentDir|TerminalEchoState|Exec|Run|Output|CombinedOutput)\(`)
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		depth := 0
		for lineNumber, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "func ") {
				depth = 0
			}
			if lockCall.MatchString(line) {
				depth++
			}
			if depth > 0 && backendCall.MatchString(line) {
				t.Errorf("%s:%d calls backend I/O while a session state lock is held: %s", file, lineNumber+1, strings.TrimSpace(line))
			}
			if unlockCall.MatchString(line) && !strings.Contains(line, "defer ") {
				depth--
			}
		}
	}
}

func TestGreenfield_NoRawPtmxWriteOutsidePTYFiles(t *testing.T) {
	allowed := map[string]bool{
		"pty_local_unix.go":    true,
		"pty_local_windows.go": true,
		"pty_tmux.go":          true,
	}
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

func TestGreenfield_NoMouseRoutingPredicateInAPI(t *testing.T) {
	files, err := walkModuleGoFiles(".")
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`isMouseTrackingSequence`)
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if re.Match(data) {
			t.Errorf("%s contains browser-owned mouse routing predicate", file)
		}
	}
}

func TestGreenfield_TerminalHasOneWebSocketUpgrade(t *testing.T) {
	files, err := walkModuleGoFiles(".")
	if err != nil {
		t.Fatal(err)
	}

	terminalUpgradeFunctions := 0
	terminalUpgradeCalls := 0
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if !strings.Contains(line, "websocket.Upgrader") && !strings.Contains(line, ".Upgrade(w, r, nil)") {
				continue
			}
			fn := findEnclosingFunc(lines, i)
			if strings.Contains(fn, "Terminal") || fn == "handleTerminalWS" {
				if strings.Contains(line, "websocket.Upgrader") {
					terminalUpgradeFunctions++
				}
				if strings.Contains(line, ".Upgrade(w, r, nil)") {
					terminalUpgradeCalls++
				}
			}
		}
	}
	if terminalUpgradeFunctions != 1 || terminalUpgradeCalls != 1 {
		t.Fatalf("terminal WebSocket upgrade sites = declarations %d, calls %d; want one of each", terminalUpgradeFunctions, terminalUpgradeCalls)
	}

	routeRegistrations := 0
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		routeRegistrations += strings.Count(string(data), "WS:     s.handleTerminalWS")
	}
	if routeRegistrations != 1 {
		t.Fatalf("terminal WebSocket route registrations = %d, want exactly one", routeRegistrations)
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

func TestGreenfield_UIWritesOnlyServerTerminalBytes(t *testing.T) {
	root := filepath.Join("..", "ui", "src")
	operatorWrite := regexp.MustCompile(`\b(?:terminal|t)\.write\(\s*["'\x60]`)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || (filepath.Ext(path) != ".ts" && filepath.Ext(path) != ".tsx") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if operatorWrite.Match(data) {
			t.Errorf("%s writes an operator string into the terminal buffer", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
