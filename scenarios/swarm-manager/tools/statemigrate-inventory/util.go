package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var roundFileRE = regexp.MustCompile(`^round-\d{3}\.json$`)

func isRoundFile(base string) bool { return roundFileRE.MatchString(base) }

// relOf returns the forward-slash relative path of p under root.
func relOf(root, p string) string {
	r, err := filepath.Rel(root, p)
	if err != nil {
		return filepath.ToSlash(p)
	}
	return filepath.ToSlash(r)
}

// redactHome replaces the user's home dir prefix with "~" so paths are portable
// and byte-stable across machines.
func redactHome(p string) string {
	if p == "" {
		return ""
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if p == home {
			return "~"
		}
		if strings.HasPrefix(p, home+string(os.PathSeparator)) {
			return "~/" + filepath.ToSlash(strings.TrimPrefix(p, home+string(os.PathSeparator)))
		}
	}
	return filepath.ToSlash(p)
}

func contains(seg []string, want string) bool {
	for _, s := range seg {
		if s == want {
			return true
		}
	}
	return false
}

// containsSeq reports whether a and b appear adjacently (a immediately before b).
func containsSeq(seg []string, a, b string) bool {
	for i := 0; i+1 < len(seg); i++ {
		if seg[i] == a && seg[i+1] == b {
			return true
		}
	}
	return false
}

// dirName returns the parent directory name of a rel path (the item/entity
// folder name for "<kind>/<name>/spec.json").
func dirName(rel string) string {
	return path.Base(path.Dir(rel))
}

func base(rel string) string { return path.Base(rel) }

// jsonQuote renders a value in Go-quoted form for stable, escaped detail strings.
func jsonQuote(v string) string { return strconv.Quote(v) }

func uniqueSorted(in []string) []string {
	if len(in) == 0 {
		return in
	}
	sort.Strings(in)
	out := in[:0:0]
	var last string
	for i, v := range in {
		if i == 0 || v != last {
			out = append(out, v)
		}
		last = v
	}
	return out
}

// hashLines returns the hex sha256 over newline-joined sorted lines.
func hashLines(lines []string) string {
	h := sha256.New()
	for _, l := range lines {
		h.Write([]byte(l))
		h.Write([]byte{'\n'})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func sortFindings(f []Finding) {
	sort.Slice(f, func(i, j int) bool {
		if f[i].Type != f[j].Type {
			return f[i].Type < f[j].Type
		}
		if f[i].From != f[j].From {
			return f[i].From < f[j].From
		}
		return f[i].To < f[j].To
	})
}

func sortAnomalies(a []Anomaly) {
	sort.Slice(a, func(i, j int) bool {
		if a[i].Type != a[j].Type {
			return a[i].Type < a[j].Type
		}
		if a[i].RelPath != a[j].RelPath {
			return a[i].RelPath < a[j].RelPath
		}
		return a[i].Detail < a[j].Detail
	})
}
