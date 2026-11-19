package evals

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domain "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

type callerHarvestClient interface {
	GetRun(context.Context, *connect.Request[api.GetRunRequest]) (*connect.Response[api.GetRunResponse], error)
	GetRunEvents(context.Context, *connect.Request[api.GetRunEventsRequest]) (*connect.Response[api.GetRunEventsResponse], error)
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
		agentManager := apiconnect.NewAgentManagerServiceClient(httpClient, managerURL)
		requestCtx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
		defer cancel()
		bundle, err := harvestSelectedCallers(requestCtx, agentManager, runIDs)
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
		return nil, fmt.Errorf("--run-ids is required; select IDs from Agent Manager run or conversation search")
	}
	sort.Strings(ids)
	return ids, nil
}

func harvestSelectedCallers(ctx context.Context, client callerHarvestClient, selected []string) (*callerHarvestBundle, error) {
	if client == nil {
		return nil, fmt.Errorf("generated Agent Manager Connect client is required")
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("selected Agent Manager run IDs are required")
	}
	bundle := &callerHarvestBundle{SchemaVersion: "agent-manager-caller-audit/v1", Source: "Agent Manager run events", CreatedAt: time.Now().UTC()}
	for _, runID := range selected {
		run, err := getAgentManagerRun(ctx, client, runID)
		if err != nil {
			return nil, fmt.Errorf("validate selected Agent Manager run %s: %w", runID, err)
		}
		records, err := readAgentManagerCallers(ctx, client, runID, run.GetHarnessSessionId())
		if err != nil {
			return nil, fmt.Errorf("harvest Agent Manager run %s: %w", runID, err)
		}
		bundle.Records = append(bundle.Records, records...)
	}
	return bundle, nil
}

func getAgentManagerRun(ctx context.Context, client callerHarvestClient, runID string) (*domain.Run, error) {
	response, err := client.GetRun(ctx, connect.NewRequest(&api.GetRunRequest{RunId: runID}))
	if err != nil {
		return nil, fmt.Errorf("read Agent Manager run: %w", err)
	}
	if response.Msg.GetRun() == nil || response.Msg.GetRun().GetId() != runID {
		return nil, fmt.Errorf("Agent Manager returned mismatched run identity")
	}
	return response.Msg.GetRun(), nil
}

func readAgentManagerCallers(ctx context.Context, client callerHarvestClient, runID, harnessSessionID string) ([]callerHarvestRecord, error) {
	var events []*domain.RunEvent
	var after int64 = -1
	for {
		limit := int32(1000)
		response, err := client.GetRunEvents(ctx, connect.NewRequest(&api.GetRunEventsRequest{
			RunId: runID, AfterSequence: &after, Limit: &limit,
			EventTypes: []domain.RunEventType{domain.RunEventType_RUN_EVENT_TYPE_MESSAGE, domain.RunEventType_RUN_EVENT_TYPE_MESSAGE_DELETED},
		}))
		if err != nil {
			return nil, fmt.Errorf("read Agent Manager run events: %w", err)
		}
		page := response.Msg
		if len(page.GetEvents()) == 0 && page.GetHasMore() {
			return nil, fmt.Errorf("Agent Manager returned an empty event page with has_more")
		}
		for _, event := range page.GetEvents() {
			if event.GetId() == "" || event.GetRunId() != runID || event.GetSequence() <= after {
				return nil, fmt.Errorf("Agent Manager event page contains invalid source identity")
			}
			after = event.GetSequence()
			events = append(events, event)
		}
		if !page.GetHasMore() {
			break
		}
	}
	deleted := map[string]bool{}
	conversationIDs := map[string]bool{}
	for _, event := range events {
		if event.GetMessageDeleted() != nil {
			deleted[event.GetMessageDeleted().GetTargetEventId()] = true
		}
		if message := event.GetMessage(); message != nil && strings.TrimSpace(message.GetConversationId()) != "" {
			conversationIDs[strings.TrimSpace(message.GetConversationId())] = true
		}
	}
	var records []callerHarvestRecord
	for _, event := range events {
		message := event.GetMessage()
		if message == nil || message.GetRole() != "user" || message.GetSource() != "operator" || message.GetEvidenceOnly() || deleted[event.GetId()] || strings.TrimSpace(message.GetContent()) == "" {
			continue
		}
		conversationID := strings.TrimSpace(message.GetConversationId())
		if conversationID == "" {
			// Older Agent Manager imports omit conversation_id on operator
			// messages but retain it on assistant messages in the same run. Use
			// that identity only when the run has exactly one; otherwise preserve
			// the canonical harness session ID when available.
			if len(conversationIDs) == 1 {
				for id := range conversationIDs {
					conversationID = id
				}
			} else if len(conversationIDs) == 0 {
				conversationID = strings.TrimSpace(harnessSessionID)
			}
		}
		if conversationID == "" {
			continue
		}
		records = append(records, callerHarvestRecord{RunID: runID, EventID: event.GetId(), Sequence: uint64(event.GetSequence()), ConversationID: conversationID, MessageRole: message.GetRole(), MessageSource: message.GetSource(), Query: strings.TrimSpace(message.GetContent()), Status: "candidate", AcceptanceEligible: false})
	}
	return records, nil
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
