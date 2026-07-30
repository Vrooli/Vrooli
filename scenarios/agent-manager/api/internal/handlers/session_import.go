package handlers

// Runner-backed transcript discovery.  The browser never receives a host path:
// it selects an opaque key which is resolved beneath a resource-owned session
// root on the server.  This keeps importing evidence local and prevents a
// client from using this surface as a filesystem reader.

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
)

type runnerSessionSource struct {
	RunnerType domain.RunnerType
	Label      string
	Root       string
}

type discoveredSession struct {
	Key           string `json:"key"`
	SessionID     string `json:"sessionId"`
	Title         string `json:"title"`
	Preview       string `json:"preview,omitempty"`
	UpdatedAt     string `json:"updatedAt"`
	ImportedRunID string `json:"importedRunId,omitempty"`
}

// governedRunnerSessionSources mirrors the durable_data session entries owned
// by the coding-agent resources. Add a runner here only when its resource
// declares a non-sensitive, parseable conversation store.
func governedRunnerSessionSources() []runnerSessionSource {
	// These are resource ids and durable-data entry ids, rather than paths.
	// The resource manifests remain the source of truth for where each runner
	// keeps its conversations; Agent Manager only knows which durable entry is
	// a conversation source and which codec can parse it.
	definitions := []struct {
		runner                 domain.RunnerType
		resource, entry, label string
	}{
		{domain.RunnerTypeCodex, "codex", "sessions", "Codex"},
		{domain.RunnerTypeClaudeCode, "claude-code", "projects", "Claude Code"},
		{domain.RunnerTypeOpenCode, "opencode", "storage", "OpenCode"},
		{domain.RunnerTypeGrok, "grok", "sessions", "Grok"},
	}
	root := findWorkspaceRoot()
	home, err := os.UserHomeDir()
	if root == "" || err != nil {
		return nil
	}
	type durableEntry struct {
		Path string `json:"path"`
	}
	type resourceManifest struct {
		DurableData struct {
			Base    string                  `json:"base"`
			Entries map[string]durableEntry `json:"entries"`
		} `json:"durable_data"`
	}
	sources := make([]runnerSessionSource, 0, len(definitions))
	for _, definition := range definitions {
		contents, err := os.ReadFile(filepath.Join(root, "resources", definition.resource, "resource.json"))
		if err != nil {
			continue
		}
		var manifest resourceManifest
		if json.Unmarshal(contents, &manifest) != nil {
			continue
		}
		entry, ok := manifest.DurableData.Entries[definition.entry]
		if !ok || entry.Path == "" {
			continue
		}
		base := strings.Replace(manifest.DurableData.Base, "$HOME", home, 1)
		sources = append(sources, runnerSessionSource{definition.runner, definition.label, filepath.Join(base, entry.Path)})
	}
	return sources
}

func findWorkspaceRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for parent := ""; dir != parent; parent, dir = dir, filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "resources")); err == nil {
			return dir
		}
	}
	return ""
}

func sourceForRunner(value string) (runnerSessionSource, bool) {
	for _, source := range governedRunnerSessionSources() {
		if string(source.RunnerType) == value {
			return source, true
		}
	}
	return runnerSessionSource{}, false
}

func (h *Handler) ListImportSources(w http.ResponseWriter, r *http.Request) {
	type sourceView struct {
		RunnerType   string `json:"runnerType"`
		Label        string `json:"label"`
		State        string `json:"state"`
		SessionCount int    `json:"sessionCount"`
	}
	views := make([]sourceView, 0)
	for _, source := range governedRunnerSessionSources() {
		sessions, err := scanRunnerSessions(source, nil)
		state := "ready"
		if err != nil {
			state = "unavailable"
		}
		views = append(views, sourceView{RunnerType: string(source.RunnerType), Label: source.Label, State: state, SessionCount: len(sessions)})
	}
	writeJSON(w, http.StatusOK, map[string]any{"sources": views})
}

func (h *Handler) ListRunnerSessions(w http.ResponseWriter, r *http.Request) {
	source, ok := sourceForRunner(r.URL.Query().Get("runnerType"))
	if !ok {
		writeSimpleError(w, r, "runnerType", "unknown runner source")
		return
	}
	runs, err := h.svc.ListRuns(r.Context(), orchestration.RunListOptions{ListOptions: orchestration.ListOptions{Limit: 5000}})
	if err != nil {
		writeError(w, r, err)
		return
	}
	associated := make(map[string]string, len(runs))
	for _, run := range runs {
		if run.SessionID != "" {
			associated[run.SessionID] = run.ID.String()
		}
	}
	sessions, err := scanRunnerSessions(source, associated)
	if err != nil {
		writeSimpleError(w, r, "source", "runner sessions are unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"runnerType": source.RunnerType, "sessions": sessions})
}

func (h *Handler) ImportRunnerSession(w http.ResponseWriter, r *http.Request) {
	var request struct {
		RunnerType string `json:"runnerType"`
		SessionKey string `json:"sessionKey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeSimpleError(w, r, "body", "invalid session import request")
		return
	}
	source, ok := sourceForRunner(request.RunnerType)
	if !ok {
		writeSimpleError(w, r, "runnerType", "unknown runner source")
		return
	}
	path, ok := safeSessionPath(source, request.SessionKey)
	if !ok {
		writeSimpleError(w, r, "sessionKey", "invalid runner session")
		return
	}
	if _, err := os.Stat(path); err != nil {
		writeSimpleError(w, r, "sessionKey", "runner session is no longer available")
		return
	}
	run, err := h.svc.ImportTranscript(r.Context(), orchestration.ImportTranscriptRequest{Path: path, RunnerType: source.RunnerType, Label: filepath.Base(path)})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": run.ID.String(), "status": run.Status, "sessionId": run.SessionID})
}

func scanRunnerSessions(source runnerSessionSource, associated map[string]string) ([]discoveredSession, error) {
	if _, err := os.Stat(source.Root); err != nil {
		return nil, err
	}
	var sessions []discoveredSession
	err := filepath.WalkDir(source.Root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !strings.HasSuffix(entry.Name(), ".jsonl") {
			return err
		}
		key, err := filepath.Rel(source.Root, path)
		if err != nil {
			return err
		}
		id, preview := readSessionIdentity(path)
		if id == "" {
			id = strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		session := discoveredSession{Key: filepath.ToSlash(key), SessionID: id, Title: sessionTitle(source.RunnerType, entry.Name(), preview), Preview: preview, UpdatedAt: info.ModTime().UTC().Format(time.RFC3339)}
		if associated != nil {
			session.ImportedRunID = associated[id]
		}
		sessions = append(sessions, session)
		return nil
	})
	sort.Slice(sessions, func(i, j int) bool { return sessions[i].UpdatedAt > sessions[j].UpdatedAt })
	if len(sessions) > 250 {
		sessions = sessions[:250]
	}
	return sessions, err
}

func safeSessionPath(source runnerSessionSource, key string) (string, bool) {
	key = filepath.Clean(strings.TrimSpace(key))
	if key == "." || filepath.IsAbs(key) || strings.HasPrefix(key, ".."+string(filepath.Separator)) || key == ".." || !strings.HasSuffix(key, ".jsonl") {
		return "", false
	}
	path := filepath.Join(source.Root, key)
	rel, err := filepath.Rel(source.Root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	resolvedRoot, err := filepath.EvalSymlinks(source.Root)
	if err != nil {
		return "", false
	}
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", false
	}
	resolvedRel, err := filepath.Rel(resolvedRoot, resolvedPath)
	return resolvedPath, err == nil && resolvedRel != ".." && !strings.HasPrefix(resolvedRel, ".."+string(filepath.Separator))
}

func readSessionIdentity(path string) (string, string) {
	file, err := os.Open(path)
	if err != nil {
		return "", ""
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	var id, preview string
	for lines := 0; scanner.Scan() && lines < 80; lines++ {
		var value any
		if json.Unmarshal(scanner.Bytes(), &value) != nil {
			continue
		}
		if id == "" {
			id = findString(value, "session_id", "sessionId", "id")
		}
		if preview == "" {
			preview = firstUserText(value)
		}
		if id != "" && preview != "" {
			break
		}
	}
	return id, preview
}

func findString(value any, keys ...string) string {
	object, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	for _, key := range keys {
		if text, ok := object[key].(string); ok && text != "" {
			return text
		}
	}
	if payload, ok := object["payload"].(map[string]any); ok {
		for _, key := range keys {
			if text, ok := payload[key].(string); ok && text != "" {
				return text
			}
		}
	}
	return ""
}

func firstUserText(value any) string {
	object, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	message, _ := object["message"].(map[string]any)
	if message == nil {
		message, _ = object["payload"].(map[string]any)
	}
	role, _ := message["role"].(string)
	if role != "user" {
		return ""
	}
	text, _ := message["content"].(string)
	if text == "" {
		text = contentText(message["content"])
	}
	if text == "" {
		text, _ = message["text"].(string)
	}
	text = strings.Join(strings.Fields(text), " ")
	if len(text) > 140 {
		return text[:137] + "…"
	}
	return text
}

func contentText(value any) string {
	items, ok := value.([]any)
	if !ok {
		return ""
	}
	for _, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			continue
		}
		for _, key := range []string{"text", "content"} {
			if text, ok := object[key].(string); ok && text != "" {
				return text
			}
		}
	}
	return ""
}

func sessionTitle(runnerType domain.RunnerType, filename, preview string) string {
	if preview != "" {
		return preview
	}
	return string(runnerType) + " · " + strings.TrimSuffix(filename, filepath.Ext(filename))
}
