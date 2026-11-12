package home

import (
	"bufio"
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FileFact is one file's size and modification time.
type FileFact struct {
	Name     string
	Path     string
	Bytes    int64
	Modified time.Time
}

// stat returns the fact for a regular file, or false when it is absent.
func stat(path string) (FileFact, bool) {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return FileFact{}, false
	}
	return FileFact{Name: filepath.Base(path), Path: path, Bytes: info.Size(), Modified: info.ModTime()}, true
}

// Unreadable is the evidence for a file or folder that exists but could not
// be read: what it holds is unknown, never empty (rulings E1).
func Unreadable(path string, err error) Evidence {
	return Evidence{Source: path, Reason: "cannot read " + path + ": " + err.Error()}
}

// absent reports whether err says the path does not exist.
func absent(err error) bool { return errors.Is(err, fs.ErrNotExist) }

// topFiles lists the regular files directly inside dir, sorted by name.
// Hidden files are skipped. A missing dir has none; an unreadable one is an
// error.
func topFiles(dir string) ([]FileFact, error) {
	entries, err := os.ReadDir(dir)
	if absent(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := make([]FileFact, 0, len(entries))
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") || entry.IsDir() {
			continue
		}
		if fact, ok := stat(filepath.Join(dir, entry.Name())); ok {
			out = append(out, fact)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// firstHeading returns the text of the first "# " heading in a Markdown file.
// A file that exists but cannot be read gives an unknown title with the
// reason, and true: the title is that file's, not another's.
func firstHeading(path string) (Text, bool) {
	fact, ok := stat(path)
	if !ok {
		return Text{}, false
	}
	file, err := os.Open(path)
	if err != nil {
		return Text{Evidence: Unreadable(path, err)}, true
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "# ") {
			return Text{Value: strings.TrimSpace(line[2:]), Evidence: From(path, fact.Modified)}, true
		}
	}
	if err := scanner.Err(); err != nil {
		return Text{Evidence: Unreadable(path, err)}, true
	}
	return Text{}, false
}

// firstParagraphLine returns the first prose line after the title of a
// Markdown file: the supervisor's own status line. Emphasis markers are
// removed and the line is capped at 300 characters.
func firstParagraphLine(path string) (Text, bool) {
	fact, ok := stat(path)
	if !ok {
		return Text{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Text{Evidence: Unreadable(path, err)}, true
	}
	for _, raw := range bytes.Split(data, []byte("\n")) {
		line := strings.TrimSpace(string(raw))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.ReplaceAll(line, "**", "")
		if runes := []rune(line); len(runes) > 300 {
			line = string(runes[:299]) + "…"
		}
		return Text{Value: line, Evidence: From(path, fact.Modified)}, true
	}
	return Text{}, false
}
