package proposals

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
)

// Digest binds a proposal revision to its base, exact file contents and
// rendered message. Apply intents include it, so an approval cannot apply a
// different revision.
func Digest(baseHead string, files []File, rendered string) string {
	sorted := append([]File(nil), files...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })
	hash := sha256.New()
	hash.Write([]byte("gct-proposal-v1\x00" + baseHead + "\x00"))
	for _, file := range sorted {
		hash.Write([]byte(file.Path + "\x00" + file.ContentID() + "\x00"))
	}
	hash.Write([]byte("\x00" + rendered))
	return hex.EncodeToString(hash.Sum(nil))
}

// SubjectContext is the human-control subject context of one revision.
func SubjectContext(id string, revision int) string {
	return "proposal:" + strings.TrimSpace(id) + "@" + strconv.Itoa(revision)
}

// ParseSubjectContext splits "proposal:<id>@<revision>".
func ParseSubjectContext(value string) (string, int, bool) {
	rest, ok := strings.CutPrefix(strings.TrimSpace(value), "proposal:")
	if !ok {
		return "", 0, false
	}
	id, rawRevision, ok := strings.Cut(rest, "@")
	if !ok || id == "" {
		return "", 0, false
	}
	revision, err := strconv.Atoi(rawRevision)
	if err != nil || revision <= 0 {
		return "", 0, false
	}
	return id, revision, true
}

func newID(prefix string) string {
	var raw [6]byte
	if _, err := rand.Read(raw[:]); err != nil {
		panic("proposals: crypto/rand unavailable: " + err.Error())
	}
	return prefix + hex.EncodeToString(raw[:])
}
