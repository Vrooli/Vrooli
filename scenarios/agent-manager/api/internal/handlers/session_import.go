package handlers

// Runner-backed transcript discovery.  The browser never receives a host path:
// it selects an opaque key which is resolved beneath a resource-owned session
// root on the server.  This keeps importing evidence local and prevents a
// client from using this surface as a filesystem reader.

import (
	"bufio"
	"encoding/json"
	"fmt"
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
	Harness    string
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
		sources = append(sources, runnerSessionSource{RunnerType: definition.runner, Label: definition.label, Root: filepath.Join(base, entry.Path), Harness: "resource:" + definition.resource + "/" + definition.entry})
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
	sessionID, _ := readSessionIdentity(path)
	if sessionID == "" {
		sessionID = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	run, err := h.svc.ImportTranscript(r.Context(), orchestration.ImportTranscriptRequest{Path: path, RunnerType: source.RunnerType, Label: filepath.Base(path), SourceHarness: source.Harness, SourceSessionID: sessionID})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": run.ID.String(), "status": run.Status, "sessionId": run.SessionID})
}

func scanRunnerSessions(source runnerSessionSource, associated map[string]string) ([]discoveredSession, error) {
	return scanRunnerSessionsLimit(source, associated, 250)
}

// scanRunnerSessionsLimit is the server-side discovery primitive. Browser
// listing remains deliberately capped, while corpus import uses the uncapped
// form so pagination cannot silently bias the selected evidence set.
func scanRunnerSessionsLimit(source runnerSessionSource, associated map[string]string, limit int) ([]discoveredSession, error) {
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
	if limit > 0 && len(sessions) > limit {
		sessions = sessions[:limit]
	}
	return sessions, err
}

type corpusImportRequest struct {
	RunnerTypes []string `json:"runnerTypes"`
	Strategy    string   `json:"strategy,omitempty"`
	From        string   `json:"from,omitempty"`
	To          string   `json:"to,omitempty"`
	PerMonth    int      `json:"perMonth,omitempty"`
	Limit       int      `json:"limit,omitempty"`
}

type corpusImportCoverage struct {
	SelectionRule   string         `json:"selectionRule"`
	Strategy        string         `json:"strategy"`
	CandidateCount  int            `json:"candidateCount"`
	Omitted         int            `json:"omitted"`
	Bounded         bool           `json:"bounded"`
	Checkpoint      string         `json:"checkpoint,omitempty"`
	Selected        int            `json:"selected"`
	Imported        int            `json:"imported"`
	AlreadyImported int            `json:"alreadyImported"`
	Replayed        int            `json:"replayed"`
	Unreplayable    int            `json:"unreplayable"`
	Failed          int            `json:"failed"`
	Skipped         map[string]int `json:"skipped"`
}

// ImportSessionCorpus adopts a bounded, reproducible sample of governed local
// sessions. Every selected item has either an imported/existing run or a named
// failure reason in the response: no source file is silently skipped.
func (h *Handler) ImportSessionCorpus(w http.ResponseWriter, r *http.Request) {
	var request corpusImportRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeSimpleError(w, r, "body", "invalid corpus import request")
		return
	}
	from, to, err := corpusWindow(request.From, request.To)
	if err != nil {
		writeSimpleError(w, r, "time_window", err.Error())
		return
	}
	perMonth, limit := request.PerMonth, request.Limit
	if perMonth <= 0 {
		perMonth = 1
	}
	strategy := strings.ToLower(strings.TrimSpace(request.Strategy))
	if strategy == "" {
		strategy = "stratified"
	}
	if strategy != "deterministic-per-month" && strategy != "stratified" && strategy != "recent" && strategy != "all" {
		writeSimpleError(w, r, "strategy", "strategy must be deterministic-per-month, stratified, recent, or all")
		return
	}
	if perMonth > 500 || limit > 500 || limit < 0 {
		writeSimpleError(w, r, "selection", "perMonth must be 1..500 and limit must be 1..500")
		return
	}
	if limit == 0 {
		limit = 24
	}
	sources, invalid := corpusSources(request.RunnerTypes)
	coverage := corpusImportCoverage{SelectionRule: fmt.Sprintf("%s selection over governed runner sessions (up to %d per month, %d total)", strategy, perMonth, limit), Strategy: strategy, Bounded: true, Skipped: map[string]int{}}
	for _, runnerType := range invalid {
		coverage.Skipped["unknown_runner:"+runnerType]++
	}
	candidates := make([]corpusCandidate, 0)
	for _, source := range sources {
		sessions, scanErr := scanRunnerSessionsLimit(source, nil, 0)
		if scanErr != nil {
			coverage.Skipped["source_unavailable:"+string(source.RunnerType)]++
			continue
		}
		for _, session := range sessions {
			updated, parseErr := time.Parse(time.RFC3339, session.UpdatedAt)
			if parseErr != nil || (from != nil && updated.Before(*from)) || (to != nil && !updated.Before(*to)) {
				continue
			}
			candidates = append(candidates, corpusCandidate{source: source, session: session, month: updated.UTC().Format("2006-01")})
		}
	}
	coverage.CandidateCount = len(candidates)
	selected := selectCorpusCandidatesWithStrategy(candidates, perMonth, limit, strategy)
	coverage.Omitted = len(candidates) - len(selected)
	coverage.Selected = len(selected)
	for _, candidate := range selected {
		path, ok := safeSessionPath(candidate.source, candidate.session.Key)
		if !ok {
			coverage.Skipped["invalid_session_path"]++
			continue
		}
		existing, lookupErr := h.svc.GetRunByImportProvenance(r.Context(), candidate.source.Harness, candidate.session.SessionID)
		if lookupErr != nil {
			coverage.Failed++
			coverage.Skipped["provenance_lookup_failed"]++
			continue
		}
		if existing != nil {
			coverage.AlreadyImported++
			continue
		}
		run, importErr := h.svc.ImportTranscript(r.Context(), orchestration.ImportTranscriptRequest{Path: path, RunnerType: candidate.source.RunnerType, Label: candidate.source.Label + "-corpus-" + candidate.month, SourceHarness: candidate.source.Harness, SourceSessionID: candidate.session.SessionID})
		if importErr != nil {
			coverage.Failed++
			coverage.Skipped["import_failed"]++
			continue
		}
		coverage.Imported++
		replay, replayErr := h.svc.ReplayInvocationFacts(r.Context(), run.ID)
		if replayErr != nil {
			coverage.Failed++
			coverage.Skipped["projection_failed"]++
			continue
		}
		if replay.Status == "unreplayable" {
			coverage.Unreplayable++
		} else {
			coverage.Replayed++
		}
		coverage.Checkpoint = candidate.source.Harness + ":" + candidate.session.SessionID
	}
	writeJSON(w, http.StatusOK, coverage)
}

type corpusCandidate struct {
	source  runnerSessionSource
	session discoveredSession
	month   string
}

func corpusWindow(rawFrom, rawTo string) (*time.Time, *time.Time, error) {
	parse := func(raw string) (*time.Time, error) {
		if raw == "" {
			return nil, nil
		}
		value, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return nil, fmt.Errorf("from and to must be RFC3339 timestamps")
		}
		return &value, nil
	}
	from, err := parse(rawFrom)
	if err != nil {
		return nil, nil, err
	}
	to, err := parse(rawTo)
	if err != nil {
		return nil, nil, err
	}
	if from != nil && to != nil && !from.Before(*to) {
		return nil, nil, fmt.Errorf("from must precede to")
	}
	return from, to, nil
}

func corpusSources(requested []string) ([]runnerSessionSource, []string) {
	all := governedRunnerSessionSources()
	if len(requested) == 0 {
		requested = []string{string(domain.RunnerTypeCodex), string(domain.RunnerTypeClaudeCode)}
	}
	allowed := make(map[string]bool, len(requested))
	for _, item := range requested {
		allowed[item] = true
	}
	seen := make(map[string]bool, len(requested))
	sources := make([]runnerSessionSource, 0, len(requested))
	for _, source := range all {
		if allowed[string(source.RunnerType)] {
			sources = append(sources, source)
			seen[string(source.RunnerType)] = true
		}
	}
	invalid := make([]string, 0)
	for _, item := range requested {
		if !seen[item] {
			invalid = append(invalid, item)
		}
	}
	return sources, invalid
}

func selectCorpusCandidates(candidates []corpusCandidate, perMonth, limit int) []corpusCandidate {
	return selectCorpusCandidatesWithStrategy(candidates, perMonth, limit, "stratified")
}

func selectCorpusCandidatesWithStrategy(candidates []corpusCandidate, perMonth, limit int, strategy string) []corpusCandidate {
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].source.RunnerType != candidates[j].source.RunnerType {
			return candidates[i].source.RunnerType < candidates[j].source.RunnerType
		}
		if candidates[i].month != candidates[j].month {
			return candidates[i].month < candidates[j].month
		}
		return candidates[i].session.Key < candidates[j].session.Key
	})
	if strategy == "recent" {
		sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].session.UpdatedAt > candidates[j].session.UpdatedAt })
	}
	if strategy == "all" {
		if limit > len(candidates) {
			limit = len(candidates)
		}
		return append([]corpusCandidate(nil), candidates[:limit]...)
	}
	byRunner := make(map[domain.RunnerType][]corpusCandidate)
	runners := make([]domain.RunnerType, 0)
	counts := map[string]int{}
	for _, candidate := range candidates {
		key := candidate.month + "\x00" + string(candidate.source.RunnerType)
		if counts[key] >= perMonth {
			continue
		}
		if _, exists := byRunner[candidate.source.RunnerType]; !exists {
			runners = append(runners, candidate.source.RunnerType)
		}
		counts[key]++
		byRunner[candidate.source.RunnerType] = append(byRunner[candidate.source.RunnerType], candidate)
	}
	// Round-robin runners rather than globally sorting all months. This keeps a
	// long-lived store from crowding out another requested runner at the limit.
	selected := make([]corpusCandidate, 0, limit)
	for index := 0; len(selected) < limit; index++ {
		added := false
		for _, runnerType := range runners {
			items := byRunner[runnerType]
			if index >= len(items) {
				continue
			}
			selected = append(selected, items[index])
			added = true
			if len(selected) == limit {
				break
			}
		}
		if !added {
			break
		}
	}
	return selected
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
