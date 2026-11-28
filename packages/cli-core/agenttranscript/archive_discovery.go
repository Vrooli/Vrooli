package agenttranscript

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
)

// IsNativeArchiveFile applies the filename and parent-directory rules stored
// in a harness format descriptor.
func IsNativeArchiveFile(harness, path string) bool {
	format, ok := FormatFor(harness)
	if !ok || format.ArchiveID == "" {
		return false
	}
	base := filepath.Base(path)
	matched := false
	if format.ArchiveFileName != "" {
		if base != format.ArchiveFileName {
			return false
		}
		matched = true
	}
	if format.ArchivePrefix != "" {
		if !strings.HasPrefix(base, format.ArchivePrefix) {
			return false
		}
		matched = true
	}
	if format.ArchiveSuffix != "" {
		if !strings.HasSuffix(strings.ToLower(base), strings.ToLower(format.ArchiveSuffix)) {
			return false
		}
		matched = true
	}
	if format.ArchiveParent != "" {
		if filepath.Base(filepath.Dir(path)) != format.ArchiveParent {
			if !archiveParentPathMatches(path, format.ArchiveParents) {
				return false
			}
		}
		matched = true
	}
	if format.ArchiveParent == "" && len(format.ArchiveParents) > 0 {
		if !archiveParentPathMatches(path, format.ArchiveParents) {
			return false
		}
		matched = true
	}
	return matched
}

func archiveParentPathMatches(path string, parents []string) bool {
	parentPath := filepath.ToSlash(filepath.Clean(filepath.Dir(path)))
	for _, parent := range parents {
		parent = strings.Trim(filepath.ToSlash(filepath.Clean(parent)), "/")
		if parent != "" && (parentPath == parent || strings.HasSuffix(parentPath, "/"+parent)) {
			return true
		}
	}
	return false
}

// NativeArchiveSessionID returns the native session identity for an archive
// accepted by the shared format descriptor. File reads are limited to the
// harness-specific discovery policy recorded in that descriptor.
func NativeArchiveSessionID(harness, path string, info os.FileInfo) (string, bool) {
	format, ok := FormatFor(harness)
	if !ok || !IsNativeArchiveFile(harness, path) {
		return "", false
	}
	switch format.ArchiveID {
	case "file-stem":
		return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)), true
	case "parent-directory":
		return filepath.Base(filepath.Dir(path)), true
	case "codex-rollout":
		if id := codexArchiveSessionID(path); id != "" {
			return id, true
		}
		base := filepath.Base(path)
		return strings.TrimSuffix(strings.TrimPrefix(base, "rollout-"), filepath.Ext(base)), true
	case "gemini-document":
		if info == nil || info.Size() <= 0 || format.ArchiveMaxBytes <= 0 || info.Size() > format.ArchiveMaxBytes {
			return "", false
		}
		data, err := os.ReadFile(path)
		if err != nil || int64(len(data)) != info.Size() {
			return "", false
		}
		return GeminiConversationSessionID(data), true
	case "directory-hash":
		sum := sha256.Sum256([]byte(filepath.Dir(path)))
		return hex.EncodeToString(sum[:8]), true
	case "antigravity-brain":
		// Native Antigravity CLI transcripts are nested under
		// <conversation-id>/.system_generated/logs/transcript_full.jsonl.
		directory := filepath.Dir(path)
		for depth := 0; depth < 2; depth++ {
			directory = filepath.Dir(directory)
		}
		id := filepath.Base(directory)
		return id, id != "" && id != "." && id != string(filepath.Separator)
	default:
		return "", false
	}
}

func codexArchiveSessionID(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	for scanner.Scan() {
		meta, ok := ParseCodexRolloutSessionMeta(scanner.Bytes())
		if !ok {
			continue
		}
		if meta.ID != "" {
			return meta.ID
		}
		return meta.SessionID
	}
	return ""
}
