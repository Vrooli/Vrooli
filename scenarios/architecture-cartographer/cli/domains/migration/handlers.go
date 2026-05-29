package migration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"connectrpc.com/connect"

	migrationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/migration"
	migrationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/migration/migration_v1connect"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client migrationconnect.MigrationServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: migrationconnect.NewMigrationServiceClient(httpClient, baseURL),
	}
}

// loadAuditFindings reads a test-genie --json SuiteExecutionResult from a
// path (or stdin when "-") and flattens every phase's findings into the
// shared ArchitectureFinding slice the tracker ingests. The findings
// serialize with proto-int enums; encoding/json round-trips them back into
// the generated type since both sides share this contract.
func loadAuditFindings(path string) ([]*architecturev1.ArchitectureFinding, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil
	}
	var (
		data []byte
		err  error
	)
	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, fmt.Errorf("read audit report %q: %w", path, err)
	}
	var report struct {
		Phases []struct {
			Findings []*architecturev1.ArchitectureFinding `json:"findings"`
		} `json:"phases"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("parse audit report %q (expected a test-genie --json SuiteExecutionResult): %w", path, err)
	}
	var out []*architecturev1.ArchitectureFinding
	for _, p := range report.Phases {
		for _, f := range p.Findings {
			if f != nil {
				out = append(out, f)
			}
		}
	}
	return out, nil
}

func (h *handlers) create(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	findings, err := loadAuditFindings(ctx.Flag("from-audit"))
	if err != nil {
		return err
	}
	resp, err := h.client.CreateMigration(context.Background(), connect.NewRequest(&migrationv1.CreateMigrationRequest{
		Scenario: scenario,
		Name:     ctx.Flag("name"),
		Findings: findings,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("create migration for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetStatus() == nil {
		return fmt.Errorf("server returned no migration")
	}
	st := resp.Msg.GetStatus()
	hint := fmt.Sprintf("`migration next %s` to get the prioritized worklist.", st.GetMigration().GetId())
	return h.renderStatus(ctx, st,
		fmt.Sprintf("Migration %s created for %q — %d finding(s) ingested.", st.GetMigration().GetId(), scenario, st.GetTotal()),
		hint)
}

func (h *handlers) status(ctx cliapp.RunContext) error {
	id := ctx.Positional("migration-id")
	resp, err := h.client.GetMigrationStatus(context.Background(), connect.NewRequest(&migrationv1.GetMigrationStatusRequest{MigrationId: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get migration %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetStatus() == nil {
		return fmt.Errorf("server returned no migration")
	}
	st := resp.Msg.GetStatus()
	return h.renderStatus(ctx, st,
		fmt.Sprintf("Migration %s (%s): %d/%d resolved, %d validated, %d regression(s).",
			st.GetMigration().GetId(), lifecycleName(st.GetMigration().GetStatus()),
			st.GetResolved(), st.GetTotal(), st.GetValidated(), st.GetRegressions()),
		fmt.Sprintf("`migration next %s` for the next worklist chunk.", id))
}

func (h *handlers) next(ctx cliapp.RunContext) error {
	id := ctx.Positional("migration-id")
	resp, err := h.client.NextMigrationStep(context.Background(), connect.NewRequest(&migrationv1.NextMigrationStepRequest{MigrationId: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("next step for migration %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no worklist")
	}
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
	}
	findings := resp.Msg.GetFindings()
	results := make([]string, 0, len(findings))
	for _, f := range findings {
		results = append(results, findingLine(f))
	}
	summary := fmt.Sprintf("%d open finding(s) — work top-down (cycles block dependent moves).", len(findings))
	if len(findings) == 0 {
		summary = "No open findings. Re-audit to confirm, then close the migration."
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{summary},
		ResultsHeading: "Worklist",
		Results:        results,
	})
}

func (h *handlers) resolve(ctx cliapp.RunContext) error {
	id := ctx.Positional("migration-id")
	resp, err := h.client.ResolveFinding(context.Background(), connect.NewRequest(&migrationv1.ResolveFindingRequest{
		MigrationId: id,
		StableId:    ctx.Flag("finding"),
		Note:        ctx.Flag("note"),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("resolve finding in migration %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetFinding() == nil {
		return fmt.Errorf("server returned no finding")
	}
	f := resp.Msg.GetFinding()
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Resolved %s (%s).", f.GetStableId(), statusName(f.GetStatus()))},
		Changes:     []string{findingLine(f)},
		NextCommand: []string{fmt.Sprintf("`migration reaudit %s --from-audit <report.json>` to confirm the fix held.", id)},
	})
}

func (h *handlers) apply(ctx cliapp.RunContext) error {
	id := ctx.Positional("migration-id")
	resp, err := h.client.ApplyFinding(context.Background(), connect.NewRequest(&migrationv1.ApplyFindingRequest{
		MigrationId: id,
		StableId:    ctx.Flag("finding"),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("apply finding in migration %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetFinding() == nil {
		return fmt.Errorf("server returned no finding")
	}
	f := resp.Msg.GetFinding()
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Applied %s (status-only, %s).", f.GetStableId(), statusName(f.GetStatus()))},
		Changes:     []string{findingLine(f)},
		NextCommand: []string{fmt.Sprintf("`migration reaudit %s --from-audit <report.json>` to confirm.", id)},
	})
}

func (h *handlers) reaudit(ctx cliapp.RunContext) error {
	id := ctx.Positional("migration-id")
	findings, err := loadAuditFindings(ctx.Flag("from-audit"))
	if err != nil {
		return err
	}
	resp, err := h.client.ReauditMigration(context.Background(), connect.NewRequest(&migrationv1.ReauditMigrationRequest{
		MigrationId: id,
		Findings:    findings,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("reaudit migration %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no reaudit result")
	}
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
	}
	st := resp.Msg.GetStatus()
	status := fmt.Sprintf("Reaudit: %d validated, %d still open, %d regression(s). Progress: %d/%d resolved.",
		len(resp.Msg.GetValidated()), len(resp.Msg.GetStillOpen()), len(resp.Msg.GetRegressions()),
		st.GetResolved()+st.GetValidated(), st.GetTotal())
	var triage []cliapp.TriageGroup
	if regs := resp.Msg.GetRegressions(); len(regs) > 0 {
		items := make([]string, 0, len(regs))
		for _, f := range regs {
			items = append(items, findingLine(f))
		}
		triage = append(triage, cliapp.TriageGroup{Heading: "⚠️  Regressions (introduced or reappeared)", Items: items})
	}
	return ctx.RenderOperational(cliapp.OperationalReport{
		Status: []string{status},
		Triage: triage,
		NextSteps: []string{
			fmt.Sprintf("`migration next %s` for the remaining worklist.", id),
			fmt.Sprintf("`migration close %s` once all findings are validated.", id),
		},
	})
}

func (h *handlers) close(ctx cliapp.RunContext) error {
	id := ctx.Positional("migration-id")
	resp, err := h.client.CloseMigration(context.Background(), connect.NewRequest(&migrationv1.CloseMigrationRequest{MigrationId: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("close migration %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetStatus() == nil {
		return fmt.Errorf("server returned no migration")
	}
	st := resp.Msg.GetStatus()
	return h.renderStatus(ctx, st,
		fmt.Sprintf("Migration %s closed.", st.GetMigration().GetId()),
		"")
}

// renderStatus renders a MigrationStatus as human output (or JSON when
// --json). Open findings are grouped into a triage block.
func (h *handlers) renderStatus(ctx cliapp.RunContext, st *migrationv1.MigrationStatus, summary, hint string) error {
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), st)
	}
	var triage []cliapp.TriageGroup
	var open, regressed []string
	for _, f := range st.GetFindings() {
		if f.GetRegressed() {
			regressed = append(regressed, findingLine(f))
		}
		if isOpenStatus(f.GetStatus()) {
			open = append(open, findingLine(f))
		}
	}
	if len(regressed) > 0 {
		triage = append(triage, cliapp.TriageGroup{Heading: "⚠️  Regressions", Items: regressed})
	}
	if len(open) > 0 {
		triage = append(triage, cliapp.TriageGroup{Heading: "Open findings", Items: open})
	}
	next := []string{}
	if hint != "" {
		next = append(next, hint)
	}
	return ctx.RenderOperational(cliapp.OperationalReport{
		Status:    []string{summary},
		Triage:    triage,
		NextSteps: next,
	})
}

// findingLine renders one tracked finding for human output.
func findingLine(f *migrationv1.TrackedFinding) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s  %s/%s  [%s]", f.GetStableId(), f.GetSource(), f.GetCode(), f.GetSeverity())
	if locs := f.GetLocations(); len(locs) > 0 {
		fmt.Fprintf(&b, " %s", strings.Join(locs, ", "))
	}
	b.WriteString("  → " + statusName(f.GetStatus()))
	if f.GetRegressed() {
		b.WriteString(" (REGRESSED)")
	}
	return b.String()
}

func statusName(s migrationv1.TrackedFindingStatus) string {
	return strings.ToLower(strings.TrimPrefix(s.String(), "TRACKED_FINDING_STATUS_"))
}

func lifecycleName(s migrationv1.MigrationLifecycle) string {
	return strings.ToLower(strings.TrimPrefix(s.String(), "MIGRATION_LIFECYCLE_"))
}

func isOpenStatus(s migrationv1.TrackedFindingStatus) bool {
	switch s {
	case migrationv1.TrackedFindingStatus_TRACKED_FINDING_STATUS_RESOLVED,
		migrationv1.TrackedFindingStatus_TRACKED_FINDING_STATUS_VALIDATED,
		migrationv1.TrackedFindingStatus_TRACKED_FINDING_STATUS_COMMITTED,
		migrationv1.TrackedFindingStatus_TRACKED_FINDING_STATUS_FORCE_RESOLVED:
		return false
	default:
		return true
	}
}
