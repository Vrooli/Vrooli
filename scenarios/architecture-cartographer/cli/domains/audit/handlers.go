package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"connectrpc.com/connect"

	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/audit"
	auditconnect "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/audit/audit_v1connect"
	conflictsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/conflicts"

	"github.com/vrooli/cli-core/cliapp"
)

// Exit codes (per L5-readiness plan §7 phase 3 step 4).
const (
	ExitClean      = 0
	ExitFindings   = 1
	ExitToolError  = 2
	ExitUsageError = 3
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client auditconnect.AuditServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: auditconnect.NewAuditServiceClient(httpClient, baseURL),
	}
}

// run orchestrates an audit and translates the response outcome into
// a process exit code. On a successful RPC, this function calls
// os.Exit directly with one of {0,1,2}; it returns an error (which
// the CLI runtime translates to exit 1) only on a transport-level
// failure where we have no Outcome to map. Usage errors (invalid
// flag values) bypass the RPC and exit 3.
func (h *handlers) run(ctx cliapp.RunContext) error {
	scenario := strings.TrimSpace(ctx.Positional("scenario"))
	if scenario == "" {
		fmt.Fprintln(os.Stderr, "audit run: <scenario> is required")
		os.Exit(ExitUsageError)
	}
	failOnStr := strings.ToLower(strings.TrimSpace(ctx.Flag("fail-on")))
	failOn, ok := parseFailOn(failOnStr)
	if !ok {
		fmt.Fprintf(os.Stderr, "audit run: invalid --fail-on=%q (want info|warn|error|blocker)\n", failOnStr)
		os.Exit(ExitUsageError)
	}
	includeTypes := splitCSV(ctx.Flag("include-types"))
	excludeTypes := splitCSV(ctx.Flag("exclude-types"))
	asJSON := ctx.Flag("json") != ""

	resp, err := h.client.Run(context.Background(), connect.NewRequest(&auditv1.AuditRunRequest{
		Scenario:     scenario,
		FailOn:       failOn,
		IncludeTypes: includeTypes,
		ExcludeTypes: excludeTypes,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("audit %q", scenario), err, nil)
	}
	msg := resp.Msg

	if asJSON {
		out, mErr := json.MarshalIndent(jsonReport(msg), "", "  ")
		if mErr != nil {
			return fmt.Errorf("marshal audit report: %w", mErr)
		}
		fmt.Println(string(out))
	} else {
		renderHuman(msg)
	}

	os.Exit(exitCodeFor(msg.GetOutcome()))
	return nil // unreachable
}

func exitCodeFor(o auditv1.AuditOutcome) int {
	switch o {
	case auditv1.AuditOutcome_AUDIT_OUTCOME_CLEAN:
		return ExitClean
	case auditv1.AuditOutcome_AUDIT_OUTCOME_FINDINGS:
		return ExitFindings
	case auditv1.AuditOutcome_AUDIT_OUTCOME_TOOL_ERROR:
		return ExitToolError
	default:
		return ExitToolError
	}
}

func parseFailOn(in string) (conflictsv1.Severity, bool) {
	switch in {
	case "", "warn":
		return conflictsv1.Severity_SEVERITY_WARN, true
	case "info":
		return conflictsv1.Severity_SEVERITY_INFO, true
	case "error":
		return conflictsv1.Severity_SEVERITY_ERROR, true
	case "blocker":
		return conflictsv1.Severity_SEVERITY_BLOCKER, true
	default:
		return conflictsv1.Severity_SEVERITY_UNSPECIFIED, false
	}
}

func splitCSV(in string) []string {
	in = strings.TrimSpace(in)
	if in == "" {
		return nil
	}
	parts := strings.Split(in, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func renderHuman(msg *auditv1.AuditRunResponse) {
	fmt.Printf("Audit %s — outcome=%s findings=%d duration=%s\n",
		msg.GetScenario(), outcomeName(msg.GetOutcome()), msg.GetTotalFindings(), msg.GetDuration().AsDuration())
	if msg.GetError() != "" {
		fmt.Printf("  error: %s\n", msg.GetError())
		return
	}
	g := msg.GetGraph()
	d := msg.GetDomains()
	fmt.Printf("  graph: snapshot=%s files=%d packages=%d imports=%d\n",
		g.GetSnapshotId(), g.GetFileCount(), g.GetPackageCount(), g.GetImportEdgeCount())
	fmt.Printf("  domains: authority=%s confidence=%s count=%d\n",
		d.GetAuthority(), d.GetConfidence(), d.GetDomainCount())
	if len(msg.GetBySeverity()) > 0 {
		fmt.Print("  by severity:")
		for _, sev := range []string{"blocker", "error", "warn", "info"} {
			if c := msg.GetBySeverity()[sev]; c > 0 {
				fmt.Printf(" %s=%d", sev, c)
			}
		}
		fmt.Println()
	}
	for _, f := range msg.GetFindings() {
		fmt.Printf("  [%s] %s — %s\n", severityName(f.GetSeverity()), f.GetType(), f.GetHeadline())
	}
}

func outcomeName(o auditv1.AuditOutcome) string {
	switch o {
	case auditv1.AuditOutcome_AUDIT_OUTCOME_CLEAN:
		return "clean"
	case auditv1.AuditOutcome_AUDIT_OUTCOME_FINDINGS:
		return "findings"
	case auditv1.AuditOutcome_AUDIT_OUTCOME_TOOL_ERROR:
		return "tool_error"
	default:
		return "unspecified"
	}
}

func severityName(s conflictsv1.Severity) string {
	switch s {
	case conflictsv1.Severity_SEVERITY_BLOCKER:
		return "blocker"
	case conflictsv1.Severity_SEVERITY_ERROR:
		return "error"
	case conflictsv1.Severity_SEVERITY_WARN:
		return "warn"
	case conflictsv1.Severity_SEVERITY_INFO:
		return "info"
	default:
		return "unspecified"
	}
}

// jsonReport renders the proto message into a stable-field-name struct
// for --json consumers (CI pipelines, scripts). Avoids leaking proto
// camelCase field names by hand-mapping the user-visible shape.
type jsonReportT struct {
	Scenario      string           `json:"scenario"`
	Outcome       string           `json:"outcome"`
	Error         string           `json:"error,omitempty"`
	TotalFindings int32            `json:"total_findings"`
	BySeverity    map[string]int32 `json:"by_severity,omitempty"`
	ByType        map[string]int32 `json:"by_type,omitempty"`
	Findings      []jsonFinding    `json:"findings,omitempty"`
	Domains       map[string]any   `json:"domains"`
	Graph         map[string]any   `json:"graph"`
	DurationMS    int64            `json:"duration_ms"`
}

type jsonFinding struct {
	ID        string   `json:"id"`
	Detector  string   `json:"detector"`
	Type      string   `json:"type"`
	Subtype   string   `json:"subtype,omitempty"`
	Severity  string   `json:"severity"`
	Locations []string `json:"locations,omitempty"`
	Domains   []string `json:"domains,omitempty"`
	Headline  string   `json:"headline"`
}

func jsonReport(msg *auditv1.AuditRunResponse) jsonReportT {
	out := jsonReportT{
		Scenario:      msg.GetScenario(),
		Outcome:       outcomeName(msg.GetOutcome()),
		Error:         msg.GetError(),
		TotalFindings: msg.GetTotalFindings(),
		BySeverity:    msg.GetBySeverity(),
		ByType:        msg.GetByType(),
		DurationMS:    msg.GetDuration().AsDuration().Milliseconds(),
		Domains: map[string]any{
			"authority":    msg.GetDomains().GetAuthority(),
			"confidence":   msg.GetDomains().GetConfidence(),
			"domain_count": msg.GetDomains().GetDomainCount(),
		},
		Graph: map[string]any{
			"snapshot_id":       msg.GetGraph().GetSnapshotId(),
			"file_count":        msg.GetGraph().GetFileCount(),
			"package_count":     msg.GetGraph().GetPackageCount(),
			"import_edge_count": msg.GetGraph().GetImportEdgeCount(),
		},
	}
	for _, f := range msg.GetFindings() {
		out.Findings = append(out.Findings, jsonFinding{
			ID:        f.GetId(),
			Detector:  f.GetDetector(),
			Type:      f.GetType(),
			Subtype:   f.GetSubtype(),
			Severity:  severityName(f.GetSeverity()),
			Locations: f.GetLocations(),
			Domains:   f.GetDomains(),
			Headline:  f.GetHeadline(),
		})
	}
	return out
}
