package capacity

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	capacitypb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/capacity"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	"system-monitor/cli/internal/support"
)

// Register exposes the capacity governance commands. They are read-only over the
// platform claim ledger (overview/claims/reconcile/policy get) plus a single
// policy mutation (policy set). Claim mutation flows through `vrooli capacity`,
// never this scenario CLI.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "capacity",
		Description: "Inspect the host capacity claim ledger and tune policy levers",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "overview", Description: "Show per-GPU contention and the active claim table", Run: func(args []string) error { return runOverview(core, args) }},
			{Name: "claims", Description: "List capacity claims (--owner, --active)", Run: func(args []string) error { return runClaims(core, args) }},
			{Name: "reconcile", Description: "Classify observed GPU consumers against the ledger", Run: func(args []string) error { return runReconcile(core, args) }},
			{Name: "policy", Description: "Show or set capacity policy levers (get|set)", Run: func(args []string) error { return runPolicy(core, args) }},
		},
	}
}

func runOverview(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("capacity overview")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/capacity/overview", nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		return support.PrettyPrintJSON(body)
	}

	var resp capacitypb.GetCapacityOverviewResponse
	if err := support.DecodeProto(body, &resp); err != nil {
		return err
	}

	gpuGroups := make([]cliapp.TriageGroup, 0, len(resp.GetGpus()))
	for _, g := range resp.GetGpus() {
		gpuGroups = append(gpuGroups, cliapp.TriageGroup{
			Heading: fmt.Sprintf("GPU %d (%s)", g.GetIndex(), support.FormatMaybeString(g.GetName(), "unknown")),
			Items: []string{
				fmt.Sprintf("Total: %s", formatBytes(g.GetTotalBytes())),
				fmt.Sprintf("Used (observed): %s", formatBytes(g.GetUsedBytes())),
				fmt.Sprintf("Free: %s", formatBytes(g.GetFreeBytes())),
				fmt.Sprintf("Claimed: %s", formatBytes(g.GetClaimedBytes())),
				fmt.Sprintf("Mem util: %.1f%%", g.GetMemoryUtilizationPercent()),
			},
		})
	}
	if len(gpuGroups) == 0 {
		gpuGroups = append(gpuGroups, cliapp.TriageGroup{Heading: "GPUs", Items: []string{"(no GPUs detected)"}})
	}
	gpuGroups = append(gpuGroups, cliapp.TriageGroup{Heading: "Active claims", Items: claimLines(resp.GetClaims())})

	status := []string{
		fmt.Sprintf("%d active claim(s) across %d GPU(s).", len(resp.GetClaims()), len(resp.GetGpus())),
		fmt.Sprintf("Capacity sensing: %s", support.BoolString(resp.GetSensingAvailable(), "available", "unavailable")),
	}
	for _, warn := range resp.GetWarnings() {
		status = append(status, "⚠ "+warn)
	}

	return cliapp.RenderOperationalReport(os.Stdout, cliapp.OperationalReport{
		Status: status,
		Triage: gpuGroups,
		NextSteps: []string{
			"system-monitor capacity reconcile",
			"system-monitor capacity claims --active",
			"system-monitor capacity policy get",
		},
	})
}

func runClaims(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("capacity claims")
	owner := fs.String("owner", "", "Filter to a single owner id")
	active := fs.Bool("active", false, "Only show active (reserved/granted/degraded) claims")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	if *owner != "" {
		query.Set("owner_id", *owner)
	}
	if *active {
		query.Set("active_only", "true")
	}

	body, err := core.Get("/capacity/claims", query)
	if err != nil {
		return err
	}
	if *jsonOutput {
		return support.PrettyPrintJSON(body)
	}

	var resp capacitypb.ListCapacityClaimsResponse
	if err := support.DecodeProto(body, &resp); err != nil {
		return err
	}
	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d claim(s).", len(resp.GetClaims()))},
		ResultsHeading: "Claims",
		Results:        claimLines(resp.GetClaims()),
		RetrievalHints: []string{"system-monitor capacity overview", "system-monitor capacity reconcile"},
	})
}

func runReconcile(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("capacity reconcile")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/capacity/reconcile", nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		return support.PrettyPrintJSON(body)
	}

	var resp capacitypb.ReconcileCapacityResponse
	if err := support.DecodeProto(body, &resp); err != nil {
		return err
	}

	lines := make([]string, 0, len(resp.GetFindings()))
	warnCount := 0
	for _, f := range resp.GetFindings() {
		marker := " "
		if f.GetSeverity() == "warn" {
			marker = "⚠"
			warnCount++
		}
		lines = append(lines, fmt.Sprintf("%s [%s] %s", marker, strings.ToUpper(f.GetClass()), f.GetMessage()))
	}
	if len(lines) == 0 {
		lines = append(lines, "No GPU consumers above the tracking threshold.")
	}
	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d finding(s), %d warning(s).", len(resp.GetFindings()), warnCount)},
		ResultsHeading: "Reconciliation",
		Results:        lines,
		RetrievalHints: []string{"system-monitor capacity claims --active", "system-monitor capacity policy get"},
	})
}

func runPolicy(core *cliapp.ScenarioApp, args []string) error {
	action, rest := splitAction(args)
	switch action {
	case "get", "":
		return runPolicyGet(core, rest)
	case "set":
		return runPolicySet(core, rest)
	default:
		return fmt.Errorf("usage: system-monitor capacity policy <get|set>")
	}
}

func runPolicyGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("capacity policy get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/capacity/policy", nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		return support.PrettyPrintJSON(body)
	}

	var resp capacitypb.GetCapacityPolicyResponse
	if err := support.DecodeProto(body, &resp); err != nil {
		return err
	}
	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d policy lever(s).", len(resp.GetLevers()))},
		ResultsHeading: "Policy",
		Results:        leverLines(resp.GetLevers()),
		RetrievalHints: []string{"system-monitor capacity policy set <key> <value>"},
	})
}

func runPolicySet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("capacity policy set")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	positional := fs.Args()
	if len(positional) < 2 {
		return fmt.Errorf("usage: system-monitor capacity policy set <key> <value>")
	}
	key, value := positional[0], positional[1]

	body, err := core.Request("POST", "/capacity/policy", nil, &capacitypb.SetCapacityPolicyRequest{Key: key, Value: value})
	if err != nil {
		return err
	}
	if *jsonOutput {
		return support.PrettyPrintJSON(body)
	}

	var resp capacitypb.SetCapacityPolicyResponse
	if err := support.DecodeProto(body, &resp); err != nil {
		return err
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Set %s = %s.", key, value)},
		Changes:     leverLines(resp.GetLevers()),
		NextCommand: []string{"system-monitor capacity policy get", "system-monitor capacity overview"},
	})
}

func claimLines(claims []*capacitypb.CapacityClaim) []string {
	if len(claims) == 0 {
		return []string{"(no claims)"}
	}
	lines := make([]string, 0, len(claims))
	for _, c := range claims {
		protectedMark := ""
		if c.GetProtected() {
			protectedMark = " 🛡"
		}
		lines = append(lines, fmt.Sprintf("%s/%s  %s  %s  prio=%s  activity=%s  %s%s",
			c.GetOwnerKind(), c.GetOwnerId(), c.GetResourceKind(), c.GetStatus(),
			c.GetPriorityTier(), c.GetActivityState(), formatBytes(c.GetAmountBytes()), protectedMark))
	}
	return lines
}

func leverLines(levers []*capacitypb.PolicyLever) []string {
	lines := make([]string, 0, len(levers))
	for _, l := range levers {
		lines = append(lines, fmt.Sprintf("%s = %s", l.GetKey(), l.GetValue()))
	}
	if len(lines) == 0 {
		return []string{"(no policy levers)"}
	}
	return lines
}

func splitAction(args []string) (string, []string) {
	if len(args) == 0 {
		return "", nil
	}
	return args[0], args[1:]
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
