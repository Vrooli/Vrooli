package evals

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

type callerHarvestHTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type callerHarvestRun struct {
	ID string `json:"id"`
}

type callerHarvestRunPage struct {
	Runs    []callerHarvestRun `json:"runs"`
	HasMore bool               `json:"has_more"`
}

type callerHarvestEventPage struct {
	Events  []callerHarvestEvent `json:"events"`
	HasMore bool                 `json:"has_more"`
}

type callerHarvestEvent struct {
	ID             string                 `json:"id"`
	RunID          string                 `json:"run_id"`
	Sequence       json.RawMessage        `json:"sequence"`
	ParsedSequence int64                  `json:"-"`
	EventType      string                 `json:"event_type"`
	Message        *callerHarvestMessage  `json:"message"`
	MessageDeleted *callerHarvestDeletion `json:"message_deleted"`
}

type callerHarvestMessage struct {
	Role           string `json:"role"`
	Source         string `json:"source"`
	Content        string `json:"content"`
	ConversationID string `json:"conversation_id"`
	EvidenceOnly   bool   `json:"evidence_only"`
}

type callerHarvestDeletion struct {
	TargetEventID string `json:"target_event_id"`
}

type callerHarvestRecord struct {
	RunID              string `json:"run_id"`
	EventID            string `json:"event_id"`
	Sequence           uint64 `json:"sequence"`
	ConversationID     string `json:"conversation_id"`
	MessageRole        string `json:"message_role"`
	MessageSource      string `json:"message_source"`
	Query              string `json:"query"`
	Status             string `json:"status"`
	AcceptanceEligible bool   `json:"acceptance_eligible"`
}

type callerHarvestBundle struct {
	SchemaVersion string                `json:"schema_version"`
	Source        string                `json:"source"`
	CreatedAt     time.Time             `json:"created_at"`
	Records       []callerHarvestRecord `json:"records"`
}

// bindCallerHarvest registers the local export action. The caller text is
// written only to an operator-selected 0600 file; it never enters CLI output or
// Search Hub API responses.
func bindCallerHarvest(core *cliapp.ScenarioApp) cliapp.PrimitiveHandler {
	return cliapp.ContentAction(func(ctx cliapp.RunContext) error {
		runIDs, err := parseCallerRunIDs(ctx.Flag("run-ids"))
		if err != nil {
			return err
		}
		outputPath := strings.TrimSpace(ctx.Flag("output-file"))
		if outputPath == "" {
			return fmt.Errorf("--output-file is required")
		}
		managerURL, err := configuredAgentManagerBase(ctx.Flag("agent-manager-url"))
		if err != nil {
			return err
		}
		httpClient, _ := cliapp.NewConnectHTTPClient(core)
		requestCtx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
		defer cancel()
		bundle, err := harvestSelectedCallers(requestCtx, httpClient, managerURL, runIDs)
		if err != nil {
			return err
		}
		if err := writeCallerHarvestFile(outputPath, bundle); err != nil {
			return err
		}
		return ctx.RenderMutation(cliapp.MutationReport{
			Result:      []string{fmt.Sprintf("Harvested %d real caller candidate(s) from %d selected Agent Manager run(s).", len(bundle.Records), len(runIDs))},
			Changes:     []string{fmt.Sprintf("Wrote private candidate bundle to %s with mode 0600.", outputPath), "Every record remains acceptance_eligible=false until it is joined to Gateway role evidence and independently reviewed."},
			NextCommand: []string{"Review caller labels and route-role evidence before importing cases into an evaluation suite."},
		})
	})
}

func configuredAgentManagerBase(override string) (string, error) {
	base := strings.TrimSpace(override)
	if base == "" {
		base = strings.TrimSpace(os.Getenv("AGENT_MANAGER_API_URL"))
	}
	if base == "" {
		base = strings.TrimSpace(os.Getenv("AGENT_MANAGER_API_BASE"))
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed == nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return "", fmt.Errorf("set --agent-manager-url or AGENT_MANAGER_API_URL to the local Agent Manager API base")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("Agent Manager API base must use http or https")
	}
	host := strings.Trim(parsed.Hostname(), "[]")
	ip := net.ParseIP(host)
	if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return "", fmt.Errorf("caller transcripts are restricted to a loopback Agent Manager API")
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func parseCallerRunIDs(raw string) ([]string, error) {
	seen := map[string]bool{}
	var ids []string
	for _, value := range strings.Split(raw, ",") {
		id := strings.TrimSpace(value)
		if id == "" {
			continue
		}
		if seen[id] {
			return nil, fmt.Errorf("--run-ids must not contain duplicates")
		}
		seen[id] = true
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("--run-ids is required; use Agent Manager's real run inventory to select runs")
	}
	sort.Strings(ids)
	return ids, nil
}

func harvestSelectedCallers(ctx context.Context, client callerHarvestHTTPClient, base string, selected []string) (*callerHarvestBundle, error) {
	if client == nil {
		return nil, fmt.Errorf("authenticated Agent Manager HTTP client is required")
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("selected Agent Manager run IDs are required")
	}
	knownRuns, err := listAgentManagerRunIDs(ctx, client, base)
	if err != nil {
		return nil, err
	}
	for _, runID := range selected {
		if !knownRuns[runID] {
			return nil, fmt.Errorf("selected run %s is absent from the current Agent Manager run inventory", runID)
		}
	}
	bundle := &callerHarvestBundle{SchemaVersion: "agent-manager-caller-audit/v1", Source: "Agent Manager run events", CreatedAt: time.Now().UTC()}
	for _, runID := range selected {
		records, err := readAgentManagerCallers(ctx, client, base, runID)
		if err != nil {
			return nil, fmt.Errorf("harvest Agent Manager run %s: %w", runID, err)
		}
		bundle.Records = append(bundle.Records, records...)
	}
	return bundle, nil
}

func listAgentManagerRunIDs(ctx context.Context, client callerHarvestHTTPClient, base string) (map[string]bool, error) {
	known := map[string]bool{}
	for offset := 0; ; offset += 5000 {
		query := url.Values{"limit": {"5000"}, "offset": {strconv.Itoa(offset)}}
		var page callerHarvestRunPage
		if err := callerGETJSON(ctx, client, base+"/api/v1/runs?"+query.Encode(), &page); err != nil {
			return nil, fmt.Errorf("read Agent Manager run inventory: %w", err)
		}
		for _, run := range page.Runs {
			if run.ID != "" {
				known[run.ID] = true
			}
		}
		if !page.HasMore {
			return known, nil
		}
		if len(page.Runs) == 0 {
			return nil, fmt.Errorf("Agent Manager run inventory returned an empty page with has_more")
		}
	}
}

func readAgentManagerCallers(ctx context.Context, client callerHarvestHTTPClient, base, runID string) ([]callerHarvestRecord, error) {
	var events []callerHarvestEvent
	var after int64 = -1
	for {
		query := url.Values{"after_sequence": {strconv.FormatInt(after, 10)}, "limit": {"1000"}, "event_types": {"message,message_deleted"}}
		var page callerHarvestEventPage
		endpoint := base + "/api/v1/runs/" + url.PathEscape(runID) + "/events?" + query.Encode()
		if err := callerGETJSON(ctx, client, endpoint, &page); err != nil {
			return nil, err
		}
		if len(page.Events) == 0 && page.HasMore {
			return nil, fmt.Errorf("Agent Manager returned an empty event page with has_more")
		}
		for _, event := range page.Events {
			sequence, err := callerSequence(event.Sequence)
			if len(event.Sequence) == 0 && after == -1 && len(events) == 0 {
				// Agent Manager's proto3 RunEvent.sequence field defaults to
				// zero, which protojson omits when EmitUnpopulated is disabled.
				sequence, err = 0, nil
			}
			if err != nil || event.ID == "" || event.RunID != runID || sequence <= after {
				return nil, fmt.Errorf("Agent Manager event page contains invalid source identity")
			}
			after = sequence
			event.ParsedSequence = sequence
			events = append(events, event)
		}
		if !page.HasMore {
			break
		}
	}
	deleted := map[string]bool{}
	for _, event := range events {
		if event.EventType == "RUN_EVENT_TYPE_MESSAGE_DELETED" && event.MessageDeleted != nil {
			deleted[event.MessageDeleted.TargetEventID] = true
		}
	}
	var records []callerHarvestRecord
	for _, event := range events {
		message := event.Message
		sequence := event.ParsedSequence
		if event.EventType != "RUN_EVENT_TYPE_MESSAGE" || message == nil || message.Role != "user" || message.Source != "operator" || message.EvidenceOnly || deleted[event.ID] || strings.TrimSpace(message.ConversationID) == "" || strings.TrimSpace(message.Content) == "" {
			continue
		}
		records = append(records, callerHarvestRecord{RunID: runID, EventID: event.ID, Sequence: uint64(sequence), ConversationID: message.ConversationID, MessageRole: message.Role, MessageSource: message.Source, Query: strings.TrimSpace(message.Content), Status: "candidate", AcceptanceEligible: false})
	}
	return records, nil
}

func callerSequence(raw json.RawMessage) (int64, error) {
	value := strings.TrimSpace(string(raw))
	if strings.HasPrefix(value, "\"") {
		var decoded string
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return 0, err
		}
		value = decoded
	}
	return strconv.ParseInt(value, 10, 64)
}

func callerGETJSON(ctx context.Context, client callerHarvestHTTPClient, endpoint string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build Agent Manager read request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("request Agent Manager read: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("Agent Manager read returned HTTP %d", response.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 64<<20)).Decode(target); err != nil {
		return fmt.Errorf("decode Agent Manager read response: %w", err)
	}
	return nil
}

func writeCallerHarvestFile(path string, bundle *callerHarvestBundle) error {
	if bundle == nil {
		return fmt.Errorf("caller harvest bundle is required")
	}
	path = filepath.Clean(path)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create caller candidate bundle: %w", err)
	}
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(bundle); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return fmt.Errorf("write caller candidate bundle: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return fmt.Errorf("close caller candidate bundle: %w", err)
	}
	return nil
}
