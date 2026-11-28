package agenttranscript

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// TranscriptFileOpener lets consumers supply their native decoder (for
// example, zstd decoding) while sharing identity rules for transcript paths.
type TranscriptFileOpener func(path string) (io.ReadCloser, error)

// TranscriptIdentity describes the native session and its known lineage.
type TranscriptIdentity struct {
	Harness         string
	SessionID       string
	HeaderSessionID string
	ParentSessionID string
	Sidechain       bool
	ParentPath      string
}

var transcriptUUID = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
var codexSpawnParentPattern = regexp.MustCompile(`"thread_spawn":\{"parent_thread_id":"(` + transcriptUUID.String() + `)"`)

// HarnessFamily resolves provider and resource labels through the shared
// harness names in the format registry.
func HarnessFamily(harness string) string {
	lowered := strings.ToLower(harness)
	for name := range formats {
		if strings.Contains(lowered, name) {
			return name
		}
	}
	return "configured"
}

// ReadIdentity applies the shared path and header rules for the native
// transcript formats. Callers provide OpenTranscript when their source has a
// compressed or otherwise decoded representation.
func ReadIdentity(path, harness string, open TranscriptFileOpener) TranscriptIdentity {
	if open == nil {
		open = func(path string) (io.ReadCloser, error) { return os.Open(path) }
	}
	identity := TranscriptIdentity{Harness: HarnessFamily(harness)}
	if identity.Harness == "configured" {
		identity.Harness = sniffTranscriptHarness(path, open)
	}
	if identity.Harness == "antigravity" {
		if sessionID, ok := NativeArchiveSessionID(identity.Harness, path, nil); ok {
			identity.SessionID, identity.HeaderSessionID = sessionID, sessionID
			return identity
		}
	}
	stem := transcriptStem(path)
	if identity.Harness == "gemini-cli" {
		file, err := open(path)
		if err == nil {
			data, readErr := io.ReadAll(io.LimitReader(file, 16<<20+1))
			_ = file.Close()
			if readErr == nil && len(data) <= 16<<20 {
				identity.SessionID = GeminiConversationSessionID(data)
			}
		}
		if identity.SessionID == "" {
			identity.SessionID = stem
		}
		identity.HeaderSessionID = identity.SessionID
		return identity
	}
	if identity.Harness == "opencode" {
		identity.HeaderSessionID = transcriptHeaderSessionID(path, open)
		identity.SessionID = identity.HeaderSessionID
		if identity.SessionID == "" {
			identity.SessionID = stem
		}
		return identity
	}
	identity.HeaderSessionID = transcriptHeaderSessionID(path, open)
	if identity.Harness == "claude" {
		identity.SessionID = stem
		if filepath.Base(filepath.Dir(path)) == "subagents" {
			sessionDir := filepath.Dir(filepath.Dir(path))
			identity.Sidechain = true
			identity.ParentSessionID = filepath.Base(sessionDir)
			identity.ParentPath = filepath.Join(filepath.Dir(sessionDir), identity.ParentSessionID+".jsonl")
		}
	} else {
		ids := transcriptUUID.FindAllString(stem, -1)
		if strings.Contains(stem, "_") && len(ids) >= 2 {
			identity.ParentSessionID, identity.SessionID = ids[len(ids)-2], ids[len(ids)-1]
		} else if identity.HeaderSessionID != "" {
			identity.SessionID = identity.HeaderSessionID
		} else if len(ids) > 0 {
			identity.SessionID = ids[len(ids)-1]
		} else {
			identity.SessionID = stem
		}
		if identity.ParentSessionID == "" && identity.Harness == "codex" {
			identity.ParentSessionID = codexSpawnParent(path, open)
		}
	}
	if identity.HeaderSessionID == "" {
		identity.HeaderSessionID = identity.SessionID
		if identity.Sidechain {
			identity.HeaderSessionID = identity.ParentSessionID
		}
	}
	return identity
}

func transcriptStem(path string) string {
	name := filepath.Base(path)
	for _, suffix := range []string{".jsonl.zst", ".jsonl", ".json"} {
		if strings.HasSuffix(name, suffix) {
			return strings.TrimSuffix(name, suffix)
		}
	}
	return name
}

func transcriptHeaderSessionID(path string, open TranscriptFileOpener) string {
	file, err := open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	reader := bufio.NewReaderSize(file, 64*1024)
	for lines := 0; lines < 40; lines++ {
		line, err := reader.ReadSlice('\n')
		if len(line) > 0 && err != bufio.ErrBufferFull {
			var header struct {
				Type    string `json:"type"`
				Payload struct {
					ID string `json:"id"`
				} `json:"payload"`
				SessionID  string `json:"session_id"`
				SessionID2 string `json:"sessionId"`
				ThreadID   string `json:"thread_id"`
				Params     struct {
					SessionID string `json:"sessionId"`
				} `json:"params"`
				Info struct {
					SessionID string `json:"sessionID"`
				} `json:"info"`
			}
			if json.Unmarshal(line, &header) == nil {
				switch {
				case header.Type == "session_meta" && header.Payload.ID != "":
					return header.Payload.ID
				case header.SessionID2 != "":
					return header.SessionID2
				case header.SessionID != "":
					return header.SessionID
				case header.ThreadID != "":
					return header.ThreadID
				case header.Params.SessionID != "":
					return header.Params.SessionID
				case header.Info.SessionID != "":
					return header.Info.SessionID
				}
			}
		}
		if err == bufio.ErrBufferFull {
			for err == bufio.ErrBufferFull {
				_, err = reader.ReadSlice('\n')
			}
			continue
		}
		if err != nil {
			return ""
		}
	}
	return ""
}

func codexSpawnParent(path string, open TranscriptFileOpener) string {
	file, err := open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	head := make([]byte, 8192)
	n, _ := io.ReadFull(file, head)
	if match := codexSpawnParentPattern.FindSubmatch(head[:n]); match != nil {
		return string(match[1])
	}
	return ""
}

func sniffTranscriptHarness(path string, open TranscriptFileOpener) string {
	file, err := open(path)
	if err != nil {
		return "configured"
	}
	defer file.Close()
	head := make([]byte, 8192)
	n, _ := io.ReadFull(file, head)
	text := string(head[:n])
	switch {
	case strings.Contains(text, `"type":"session_meta"`) || strings.Contains(text, `"type":"thread.started"`):
		return "codex"
	case strings.Contains(text, `"sessionId"`) || strings.Contains(text, `"type":"system"`):
		return "claude"
	}
	return "configured"
}
