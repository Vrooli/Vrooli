// Package hosthome keeps agent-manager tests away from the operator's real
// harness logins.
//
// Production seeds each run's private CODEX_HOME from <HOME>/.codex/auth.json
// (orchestration.PrepareCodecSessionHome). Any test that executes a Codex run
// therefore copies whatever login HOME points at. A test package calls Main
// from TestMain so HOME, CODEX_HOME and the temp root all live in a private
// directory seeded with a dummy login, and so a leaked or non-dummy credential
// fails the package instead of lingering on disk.
package hosthome

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// DummyAuth is the only credential tests may hold. It is not a real token.
const DummyAuth = `{"OPENAI_API_KEY":null,"tokens":{"refresh_token":"agent-manager-test-dummy-refresh-token"}}`

// Leak describes one credential file that violates the test contract.
// Fingerprint is a truncated hash so a real token is never printed.
type Leak struct {
	Path        string
	Reason      string
	Fingerprint string
}

func (l Leak) String() string {
	if l.Fingerprint == "" {
		return fmt.Sprintf("%s: %s", l.Path, l.Reason)
	}
	return fmt.Sprintf("%s: %s (sha256:%s)", l.Path, l.Reason, l.Fingerprint)
}

// FindLeaks reports every run-scoped Codex credential file under root
// (<...>/runtime/codex/auth.json). With survivors set, any such file is a
// leak, because the test that created it has finished. Otherwise only files
// that are not the dummy login are leaks.
func FindLeaks(root string, survivors bool) []Leak {
	var leaks []Leak
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || entry.Name() != "auth.json" {
			return nil
		}
		codexDir := filepath.Dir(path)
		if filepath.Base(codexDir) != "codex" || filepath.Base(filepath.Dir(codexDir)) != "runtime" {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			leaks = append(leaks, Leak{Path: path, Reason: "unreadable credential file: " + readErr.Error()})
			return nil
		}
		if !bytes.Equal(bytes.TrimSpace(data), []byte(DummyAuth)) {
			sum := sha256.Sum256(data)
			leaks = append(leaks, Leak{Path: path, Reason: "credential is not the dummy login", Fingerprint: hex.EncodeToString(sum[:6])})
			return nil
		}
		if survivors {
			leaks = append(leaks, Leak{Path: path, Reason: "credential file survived its test"})
		}
		return nil
	})
	return leaks
}

// isolatedParent is where the private root lives. It must be outside /tmp:
// the run-state root validator refuses process-ephemeral locations, and some
// fixtures create their state under HOME. GOTMPDIR is the operator's existing
// scratch location for Go work; the home directory's path (never its contents)
// is the fallback.
func isolatedParent() string {
	if dir := os.Getenv("GOTMPDIR"); dir != "" {
		return dir
	}
	if home, err := os.UserHomeDir(); err == nil {
		if dir := filepath.Join(home, ".vrooli", "tmp"); dirExists(dir) {
			return dir
		}
	}
	return ""
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// Main runs a package's tests under an isolated HOME, CODEX_HOME and TMPDIR,
// then fails the package if any test left a credential file behind or held a
// non-dummy one. Use it as: func TestMain(m *testing.M) { os.Exit(hosthome.Main(m)) }
func Main(m *testing.M) int {
	base, err := os.MkdirTemp(isolatedParent(), "amh-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "hosthome: create isolated root:", err)
		return 1
	}
	defer os.RemoveAll(base)
	home, tmp := filepath.Join(base, "home"), filepath.Join(base, "t")
	codexHome := filepath.Join(home, ".codex")
	for _, dir := range []string{codexHome, tmp} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			fmt.Fprintln(os.Stderr, "hosthome:", err)
			return 1
		}
	}
	if err := os.WriteFile(filepath.Join(codexHome, "auth.json"), []byte(DummyAuth), 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "hosthome:", err)
		return 1
	}
	for key, value := range map[string]string{"HOME": home, "CODEX_HOME": codexHome, "TMPDIR": tmp} {
		if err := os.Setenv(key, value); err != nil {
			fmt.Fprintln(os.Stderr, "hosthome:", err)
			return 1
		}
	}
	// A real XDG data home would let OpenCode runs link the operator's login.
	_ = os.Unsetenv("XDG_DATA_HOME")

	code := m.Run()
	if leaks := FindLeaks(tmp, true); len(leaks) > 0 {
		fmt.Fprintf(os.Stderr, "hosthome: %d test credential file(s) violate the isolation contract:\n", len(leaks))
		for _, leak := range leaks {
			fmt.Fprintln(os.Stderr, "  "+leak.String())
		}
		return 1
	}
	return code
}
