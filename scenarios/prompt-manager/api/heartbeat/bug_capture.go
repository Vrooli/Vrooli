package heartbeat

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"prompt-manager/store"

	"github.com/gorilla/mux"
)

const scenarioQATeamID = "scenario-qa"

var bugSignalAliases = map[string]string{
	"code-defect": "code-defect", "code_defect": "code-defect", "code defect": "code-defect",
	"regression":       "regression",
	"prompt-confusion": "prompt-confusion", "prompt_confusion": "prompt-confusion", "prompt confusion": "prompt-confusion",
	"data-shape-mismatch": "data-shape-mismatch", "data_shape_mismatch": "data-shape-mismatch", "data shape mismatch": "data-shape-mismatch",
	"unexpected-error": "unexpected-error", "unexpected_error": "unexpected-error", "unexpected error": "unexpected-error",
	"unknown": "unknown",
}

var bugSeverityAliases = map[string]string{
	"blocker": "blocker", "critical": "blocker", "high": "blocker",
	"major": "major", "medium": "major",
	"minor": "minor", "low": "minor",
}

var bugHonestyFlags = map[string]struct{}{
	"repro-not-attempted": {}, "speculative-cause": {}, "minimal-context": {}, "ai-generated-summary": {},
}

// CaptureBug accepts a partial taxonomy report. A valid payload is published
// immediately; otherwise all supplied data is retained privately as a draft.
func (h *Handlers) CaptureBug(w http.ResponseWriter, r *http.Request) {
	teamID := mux.Vars(r)["id"]
	if !requireScenarioQATeam(w, teamID) {
		return
	}
	info, ok := h.bugAttribution(w, r, teamID)
	if !ok {
		return
	}
	var req BugCaptureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	resp, err := h.captureBug(r, teamID, info, req, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// RepairBugCapture merges the supplied fields into a durable private draft and
// publishes it under the same draft id as soon as it becomes taxonomy-valid.
func (h *Handlers) RepairBugCapture(w http.ResponseWriter, r *http.Request) {
	teamID := mux.Vars(r)["id"]
	if !requireScenarioQATeam(w, teamID) {
		return
	}
	if _, ok := h.bugAttribution(w, r, teamID); !ok {
		return
	}
	draftID := mux.Vars(r)["draftId"]
	draft, err := h.teamStore.GetBugDraft(r.Context(), teamID, draftID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	var patch BugCaptureRequest
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req := mergeBugCapture(bugCaptureFromRaw(draft.Raw), patch)
	resp, err := h.captureBug(r, teamID, draft.Attribution, req, draftID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if resp.Disposition == "published" {
		if err := h.teamStore.DeleteBugDraft(r.Context(), teamID, draftID); err != nil {
			http.Error(w, fmt.Sprintf("published bug but could not remove draft: %v", err), http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func requireScenarioQATeam(w http.ResponseWriter, teamID string) bool {
	if teamID == scenarioQATeamID {
		return true
	}
	http.Error(w, "typed bug capture is owned by the scenario-qa team", http.StatusBadRequest)
	return false
}

func (h *Handlers) bugAttribution(w http.ResponseWriter, r *http.Request, teamID string) (store.AttributionInfo, bool) {
	info, err := parseAttributionHeader(r.Header.Get(attributionHeaderName))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return store.AttributionInfo{}, false
	}
	if err := validateAttribution(info, teamID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return store.AttributionInfo{}, false
	}
	return info, true
}

func (h *Handlers) captureBug(r *http.Request, teamID string, info store.AttributionInfo, req BugCaptureRequest, draftID string) (BugCaptureResponse, error) {
	accepted, needs, invalid := assessBugCapture(req)
	if len(needs) > 0 || len(invalid) > 0 {
		id := draftID
		now := time.Now().UTC().Format(time.RFC3339)
		if id == "" {
			id = "bug-" + generateID()
		}
		draft := store.BugDraft{ID: id, CreatedAt: now, UpdatedAt: now, Raw: bugCaptureRaw(req), Accepted: accepted, Needs: needs, Invalid: invalid, Warnings: []string{"Draft saved privately; it is not in bug-inbox or knowledge search."}, Attribution: info}
		if draftID != "" {
			old, err := h.teamStore.GetBugDraft(r.Context(), teamID, draftID)
			if err != nil {
				return BugCaptureResponse{}, err
			}
			draft.CreatedAt = old.CreatedAt
		}
		if err := h.teamStore.SaveBugDraft(r.Context(), teamID, draft); err != nil {
			return BugCaptureResponse{}, err
		}
		return BugCaptureResponse{Disposition: "draft", DraftID: id, Accepted: accepted, Needs: needs, Invalid: invalid, Warnings: draft.Warnings, NextAction: bugRepairCommand(id)}, nil
	}

	topic := "bug-inbox/" + accepted["signal_type"] + "/" + bugSlug(req.Title)
	if existing, err := h.teamStore.ListTeamCorpus(r.Context(), teamID, topic, "", 0); err != nil {
		return BugCaptureResponse{}, err
	} else if len(existing) > 0 {
		entry := existing[len(existing)-1]
		return BugCaptureResponse{Disposition: "published", Knowledge: &entry, Accepted: accepted, Warnings: []string{"Returned the existing report for this deterministic topic."}}, nil
	}
	entry := &store.KnowledgeEntry{ID: "knw-" + generateID(), At: time.Now().UTC().Format(time.RFC3339), Topic: topic, Content: renderBugReport(req, accepted, info), Source: "prompt-manager bug capture", Caller: deriveCaller(info, teamID), Attribution: info}
	if err := h.teamStore.AppendTeamCorpus(r.Context(), teamID, entry); err != nil {
		return BugCaptureResponse{}, err
	}
	return BugCaptureResponse{Disposition: "published", Knowledge: entry, Accepted: accepted}, nil
}

func assessBugCapture(req BugCaptureRequest) (map[string]string, []string, []store.FieldDiagnostic) {
	accepted := map[string]string{}
	needs := []string{}
	invalid := []store.FieldDiagnostic{}
	if value := normalizeBugEnum(req.SignalType, bugSignalAliases); value != "" {
		accepted["signal_type"] = value
	} else if strings.TrimSpace(req.SignalType) == "" {
		needs = append(needs, "signal_type")
	} else {
		invalid = append(invalid, store.FieldDiagnostic{Field: "signal_type", Value: req.SignalType, Message: "expected code-defect, regression, prompt-confusion, data-shape-mismatch, unexpected-error, or unknown"})
	}
	if value := normalizeBugEnum(req.Severity, bugSeverityAliases); value != "" {
		accepted["severity"] = value
	} else if strings.TrimSpace(req.Severity) == "" {
		needs = append(needs, "severity")
	} else {
		invalid = append(invalid, store.FieldDiagnostic{Field: "severity", Value: req.Severity, Message: "expected blocker, major, or minor"})
	}
	for _, required := range []struct{ field, value string }{{"title", req.Title}, {"expected", req.Expected}, {"actual", req.Actual}} {
		if strings.TrimSpace(required.value) == "" {
			needs = append(needs, required.field)
		} else {
			accepted[required.field] = strings.TrimSpace(required.value)
		}
	}
	flags := normalizedFlags(req.HonestyFlags, &invalid)
	if len(trimmedStrings(req.Repro)) > 0 {
		accepted["repro"] = "provided"
	} else if containsFlag(flags, "repro-not-attempted") {
		accepted["repro"] = "not-attempted"
	} else {
		needs = append(needs, "repro (or honesty_flags=repro-not-attempted)")
	}
	if strings.TrimSpace(req.Description) != "" {
		accepted["description"] = "provided"
	} else if containsFlag(flags, "minimal-context") {
		accepted["description"] = "minimal-context"
	} else {
		needs = append(needs, "description (or honesty_flags=minimal-context)")
	}
	accepted["honesty_flags"] = strings.Join(flags, ",")
	return accepted, needs, invalid
}

func normalizeBugEnum(value string, aliases map[string]string) string {
	return aliases[strings.ToLower(strings.TrimSpace(value))]
}

func normalizedFlags(flags []string, invalid *[]store.FieldDiagnostic) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(flags))
	for _, raw := range flags {
		flag := strings.ToLower(strings.TrimSpace(raw))
		if flag == "" {
			continue
		}
		if _, ok := bugHonestyFlags[flag]; !ok {
			*invalid = append(*invalid, store.FieldDiagnostic{Field: "honesty_flags", Value: raw, Message: "expected repro-not-attempted, speculative-cause, minimal-context, or ai-generated-summary"})
			continue
		}
		if _, ok := seen[flag]; !ok {
			seen[flag] = struct{}{}
			out = append(out, flag)
		}
	}
	sort.Strings(out)
	return out
}

func containsFlag(flags []string, want string) bool {
	for _, flag := range flags {
		if flag == want {
			return true
		}
	}
	return false
}

func trimmedStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
	}
	return out
}

func bugCaptureRaw(req BugCaptureRequest) map[string]any {
	return map[string]any{"title": req.Title, "signal_type": req.SignalType, "severity": req.Severity, "repro": req.Repro, "expected": req.Expected, "actual": req.Actual, "description": req.Description, "context": req.Context, "honesty_flags": req.HonestyFlags, "idempotency_key": req.IdempotencyKey}
}

func bugCaptureFromRaw(raw map[string]any) BugCaptureRequest {
	bytes, _ := json.Marshal(raw)
	var req BugCaptureRequest
	_ = json.Unmarshal(bytes, &req)
	return req
}

func mergeBugCapture(old, patch BugCaptureRequest) BugCaptureRequest {
	if strings.TrimSpace(patch.Title) != "" {
		old.Title = patch.Title
	}
	if strings.TrimSpace(patch.SignalType) != "" {
		old.SignalType = patch.SignalType
	}
	if strings.TrimSpace(patch.Severity) != "" {
		old.Severity = patch.Severity
	}
	if patch.Repro != nil {
		old.Repro = patch.Repro
	}
	if strings.TrimSpace(patch.Expected) != "" {
		old.Expected = patch.Expected
	}
	if strings.TrimSpace(patch.Actual) != "" {
		old.Actual = patch.Actual
	}
	if strings.TrimSpace(patch.Description) != "" {
		old.Description = patch.Description
	}
	if patch.Context != nil {
		old.Context = patch.Context
	}
	if patch.HonestyFlags != nil {
		old.HonestyFlags = patch.HonestyFlags
	}
	if strings.TrimSpace(patch.IdempotencyKey) != "" {
		old.IdempotencyKey = patch.IdempotencyKey
	}
	return old
}

func bugRepairCommand(id string) []string {
	return []string{"prompt-manager", "team", "bug-repair", scenarioQATeamID, id, "--signal-type", "<signal-type>", "--severity", "<severity>", "--expected", "<expected>", "--actual", "<actual>"}
}

func renderBugReport(req BugCaptureRequest, accepted map[string]string, info store.AttributionInfo) string {
	context := req.Context
	if context == nil {
		context = map[string]string{}
	}
	lines := []string{"---", "severity: " + accepted["severity"], "reporter: " + yamlScalar(deriveCaller(info, scenarioQATeamID)), "reporter_team: " + scenarioQATeamID, "observed_at: " + time.Now().UTC().Format("2006-01-02"), "context:", "  scenario: " + yamlScalar(context["scenario"]), "  skill: " + yamlScalar(context["skill"]), "  member: " + yamlScalar(context["member"]), "  command: " + yamlScalar(context["command"]), "repro:"}
	for _, step := range trimmedStrings(req.Repro) {
		lines = append(lines, "  - "+yamlScalar(step))
	}
	if len(trimmedStrings(req.Repro)) == 0 {
		lines = append(lines, "  - null")
	}
	lines = append(lines, "expected: "+yamlScalar(req.Expected), "actual: "+yamlScalar(req.Actual), "description: |")
	for _, line := range strings.Split(strings.TrimSpace(req.Description), "\n") {
		lines = append(lines, "  "+line)
	}
	if strings.TrimSpace(req.Description) == "" {
		lines = append(lines, "  Minimal context captured; follow the honesty flag before investigation.")
	}
	lines = append(lines, "honesty_flags: ["+strings.Join(normalizedFlags(req.HonestyFlags, &[]store.FieldDiagnostic{}), ", ")+"]", "---", "", "# "+strings.TrimSpace(req.Title))
	return strings.Join(lines, "\n") + "\n"
}

func yamlScalar(value string) string {
	if strings.TrimSpace(value) == "" {
		return "null"
	}
	encoded, _ := json.Marshal(strings.TrimSpace(value))
	return string(encoded)
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func bugSlug(title string) string {
	slug := strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(strings.TrimSpace(title)), "-"), "-")
	if slug == "" {
		return "untitled-bug"
	}
	parts := strings.Split(slug, "-")
	if len(parts) > 8 {
		slug = strings.Join(parts[:8], "-")
	}
	return slug
}
