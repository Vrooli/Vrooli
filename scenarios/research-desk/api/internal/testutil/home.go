// Package testutil holds the frozen research-home layouts the unit tests run
// on: one archive of excerpts in Go's txtar format, expanded into
// a temporary folder. No byte copy of the research home is kept; the real
// home is checked read-only by internal/realhome.
package testutil

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// homeArchive holds the smallest excerpts the tests need of four commissions
// (one per layout generation: a hand-made charter with a numbered needs list,
// a prepared packet, an older charter with STATUS.md, a packet with a
// MISSION.md link), five topics and ATTENTION.md. {{research}} and {{home}}
// stand for the research home and the user's home.
//
//go:embed home.txtar
var homeArchive []byte

// Base is the modification time of every file NewHome writes; each test sets
// the times it depends on.
var Base = time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)

// Home is one expanded layout.
type Home struct {
	// Root is the temporary folder; it stands for the user's home.
	Root string
	// Research is the research home, Root/research.
	Research string
	// Codex is an empty Web Console Codex session folder, Root/codex.
	Codex string
	// State is the desk state folder, Root/state (not created).
	State string
}

// NewHome expands the archive into a temporary folder. The folder's name is
// short, like a real research home's: packet.py counts the commission path
// inside the kernel against its 600-character gate, and t.TempDir's name
// carries the whole test name.
func NewHome(t testing.TB) Home {
	t.Helper()
	root, err := os.MkdirTemp("", "rd")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	h := Home{Root: root, Research: filepath.Join(root, "research"), Codex: filepath.Join(root, "codex"), State: filepath.Join(root, "state")}
	replacer := strings.NewReplacer("{{research}}", h.Research, "{{home}}", root)
	for _, f := range ReadArchive(homeArchive) {
		path := filepath.Join(root, filepath.FromSlash(f.Name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(replacer.Replace(string(f.Data))), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(h.Codex, 0o755); err != nil {
		t.Fatal(err)
	}
	err = filepath.WalkDir(root, func(path string, _ os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return os.Chtimes(path, Base, Base)
	})
	if err != nil {
		t.Fatal(err)
	}
	return h
}

// File is one archive member.
type File struct {
	Name string
	Data []byte
}

// ReadArchive reads Go's txtar format: a member starts at a line
// "-- <name> --" and runs to the next such line; text before the first is a
// comment and is dropped.
func ReadArchive(data []byte) []File {
	var files []File
	for _, line := range bytes.SplitAfter(data, []byte("\n")) {
		trimmed := bytes.TrimRight(line, "\n")
		if name, ok := bytes.CutPrefix(trimmed, []byte("-- ")); ok && bytes.HasSuffix(name, []byte(" --")) {
			files = append(files, File{Name: string(bytes.TrimSuffix(name, []byte(" --")))})
			continue
		}
		if len(files) > 0 {
			files[len(files)-1].Data = append(files[len(files)-1].Data, line...)
		}
	}
	return files
}

// EntryChars reads entry_chars_max from the scenario's own limits.json, the
// one source the desk reads in production. The file is found by walking up
// from the test's working directory (its package folder), because source
// paths are module-relative when tests are built with -trimpath.
func EntryChars() (int, error) {
	path, err := scenarioFile(filepath.Join("skills", "research-coordination", "templates", "limits.json"))
	if err != nil {
		return 0, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var limits struct {
		EntryCharsMax int `json:"entry_chars_max"`
	}
	if err := json.Unmarshal(data, &limits); err != nil || limits.EntryCharsMax <= 0 {
		return 0, fmt.Errorf("%s has no entry_chars_max: %v", path, err)
	}
	return limits.EntryCharsMax, nil
}

// scenarioFile finds rel under the nearest folder at or above the working
// directory that holds it.
func scenarioFile(rel string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		path := filepath.Join(dir, rel)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("%s not found at or above the working directory", rel)
		}
		dir = parent
	}
}
