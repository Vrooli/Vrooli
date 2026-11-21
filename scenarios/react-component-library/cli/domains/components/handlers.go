package components

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"connectrpc.com/connect"

	componentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components"
	componentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components/components_v1connect"
	componenttestsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/componenttests"
	componenttestsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/componenttests/componenttests_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// handlers bundles the closure over *cliapp.ScenarioApp + the generated
// Connect-Go client, mirroring the cli/domains/notes/ shape.
type handlers struct {
	core       *cliapp.ScenarioApp
	client     componentsconnect.ComponentsServiceClient
	testClient componenttestsconnect.ComponentTestsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	// A closure test can render several pinned story contracts sequentially.
	// Keep ordinary component-registry calls on the scenario default timeout,
	// but do not turn a valid long-running evidence RPC into a client-side
	// cancellation before the server can persist its report.
	testHTTPClient, _ := cliapp.NewConnectHTTPClientWithTimeout(core, 20*time.Minute)
	return &handlers{
		core:       core,
		client:     componentsconnect.NewComponentsServiceClient(httpClient, baseURL),
		testClient: componenttestsconnect.NewComponentTestsServiceClient(testHTTPClient, baseURL),
	}
}

func (h *handlers) testCall(ctx cliapp.OperationContext) (*componenttestsv1.RunComponentTestResponse, error) {
	resp, err := h.testClient.RunComponentTest(context.Background(), connect.NewRequest(&componenttestsv1.RunComponentTestRequest{ComponentId: ctx.Positional("component-id"), Version: ctx.Flag("version"), IncludeClosure: ctx.Flag("closure") == "true"}))
	if err != nil {
		return nil, cliapp.WrapAPIError("run component test", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no component test response")
	}
	return resp.Msg, nil
}

func (h *handlers) testReport(_ cliapp.OperationContext, msg *componenttestsv1.RunComponentTestResponse) cliapp.MutationReport {
	if msg == nil || msg.GetReport() == nil {
		return cliapp.MutationReport{Result: []string{"Component test response contained no report."}}
	}
	report := msg.GetReport()
	label := "Component test " + report.GetVerdict() + "."
	if msg.Reused {
		label = fmt.Sprintf("Component test reused revision %s since report %s (%s).", shortRevision(msg.SourceRevision), report.GetId(), report.GetVerdict())
	}
	changes := make([]string, 0, len(report.Results))
	for _, result := range report.Results {
		changes = append(changes, fmt.Sprintf("%s\t%s@%s\t%s", result.Stage, result.AssetLibraryId, result.Version, result.Verdict))
	}
	reportView := cliapp.MutationReport{Result: []string{label}, Changes: changes}
	if report.GetId() != "" {
		reportView.NextCommand = []string{"`components test-show " + report.GetId() + "` — retrieve this durable report"}
	}
	return reportView
}

func testOutcome(msg *componenttestsv1.RunComponentTestResponse) error {
	if msg == nil || msg.GetReport() == nil {
		return fmt.Errorf("server returned no component test report")
	}
	report := msg.GetReport()
	switch report.GetVerdict() {
	case "passed":
		return nil
	case "failed", "blocked":
		return fmt.Errorf("component test %s: report %s", report.GetVerdict(), report.GetId())
	default:
		return fmt.Errorf("component test report %s has unrecognized verdict %q", report.GetId(), report.GetVerdict())
	}
}

func shortRevision(revision string) string {
	revision = strings.TrimSpace(revision)
	if len(revision) > 8 {
		return revision[:8]
	}
	if revision == "" {
		return "unknown"
	}
	return revision
}

func (h *handlers) sweepCall(ctx cliapp.OperationContext) (*componenttestsv1.SweepComponentTestsResponse, error) {
	resp, err := h.testClient.SweepComponentTests(context.Background(), connect.NewRequest(&componenttestsv1.SweepComponentTestsRequest{
		Resume: ctx.BoolFlag("resume"), IncludeClosure: ctx.BoolFlag("closure"), ComponentId: ctx.Flag("component-id"),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("sweep component tests", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no component test sweep response")
	}
	return resp.Msg, nil
}

func (h *handlers) sweepReport(ctx cliapp.OperationContext, msg *componenttestsv1.SweepComponentTestsResponse) cliapp.ListReport {
	rows := make([]string, 0, len(msg.Reports))
	for _, report := range msg.Reports {
		if report == nil {
			continue
		}
		rows = append(rows, fmt.Sprintf("%s\t%s@%s\t%s", report.Id, report.RootLibraryId, report.RootVersion, report.Verdict))
	}
	mode := "fresh"
	if msg.Skipped > 0 || ctx.BoolFlag("resume") {
		mode = "resume"
	}
	summary := fmt.Sprintf("Component sweep (%s): planned=%d started=%d skipped=%d passed=%d failed=%d blocked=%d complete=%t.", mode, msg.Planned, msg.Started, msg.Skipped, msg.Passed, msg.Failed, msg.Blocked, msg.Complete)
	if len(msg.Errors) > 0 {
		rows = append(rows, msg.Errors...)
	}
	return cliapp.ListReport{Summary: []string{summary}, ResultsHeading: "Version reports", Results: rows, RetrievalHints: []string{"`components sweep --resume` — retry only untested or blocked versions"}}
}

// index calls ComponentsService.IndexComponents. The walk runs server-side;
// the response carries the summary.
func (h *handlers) index(ctx cliapp.RunContext) error {
	resp, err := h.client.IndexComponents(context.Background(), connect.NewRequest(&componentsv1.IndexComponentsRequest{NoReconcile: ctx.BoolFlag("no-reconcile")}))
	if err != nil {
		return cliapp.WrapAPIError("index components", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no index response")
	}
	msg := resp.Msg
	summary := []string{fmt.Sprintf(
		"Scanned %d file(s); indexed %d, skipped %d, deleted %d.",
		msg.Scanned, msg.Indexed, msg.Skipped, msg.Deleted)}
	results := append([]string{}, msg.LibraryIds...)
	if len(msg.Errors) > 0 {
		summary = append(summary, fmt.Sprintf("%d error(s) reported.", len(msg.Errors)))
		results = append(results, msg.Errors...)
	}
	if len(msg.Warnings) > 0 {
		summary = append(summary, fmt.Sprintf("%d warning(s) reported; warnings do not block indexing.", len(msg.Warnings)))
		results = append(results, msg.Warnings...)
	}
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Library IDs",
		Results:        results,
		RetrievalHints: []string{
			"`components list` — show indexed components",
			"`components get <libraryId>` — inspect a single entry",
		},
	})
}

// list calls ComponentsService.ListComponents with optional filter flags.
func (h *handlers) listCall(ctx cliapp.OperationContext) (*componentsv1.ListComponentsResponse, error) {
	req := &componentsv1.ListComponentsRequest{
		Match:    ctx.Flag("match"),
		Tag:      ctx.Flag("tag"),
		Category: ctx.Flag("category"),
		StyleId:  ctx.Flag("style"),
		Affinity: ctx.Flag("affinity"),
	}
	if rawKind := strings.TrimSpace(ctx.Flag("asset-kind")); rawKind != "" {
		switch strings.ToLower(rawKind) {
		case "component":
			req.AssetKind = componentsv1.AssetKind_ASSET_KIND_COMPONENT
		case "hook":
			req.AssetKind = componentsv1.AssetKind_ASSET_KIND_HOOK
		default:
			return nil, fmt.Errorf("--asset-kind must be component or hook (got %q)", rawKind)
		}
	}
	if rawTags := ctx.Flag("tags"); rawTags != "" {
		// Comma-separated multi-tag OR. Trim entries silently — the
		// repository drops blanks too, so `--tags ,form,` is fine.
		for _, t := range strings.Split(rawTags, ",") {
			if trimmed := strings.TrimSpace(t); trimmed != "" {
				req.Tags = append(req.Tags, trimmed)
			}
		}
	}
	if raw := ctx.Flag("limit"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("--limit must be an integer (got %q)", raw)
		}
		req.Limit = int32(n)
	}
	resp, err := h.client.ListComponents(context.Background(), connect.NewRequest(req))
	if err != nil {
		return nil, cliapp.WrapAPIError("list components", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no list response")
	}
	return resp.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, msg *componentsv1.ListComponentsResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.Components))
	for _, c := range msg.Components {
		results = append(results, formatComponent(c))
	}
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d component(s).", len(msg.Components))},
		ResultsHeading: "Components",
		Results:        results,
		RetrievalHints: []string{
			"`components get <id>` — show a single component",
			"`components index` — refresh from disk",
		},
	}
}

func (h *handlers) getCall(ctx cliapp.OperationContext) (*componentsv1.GetComponentResponse, error) {
	id := ctx.Positional("id")
	resp, err := h.client.GetComponent(context.Background(), connect.NewRequest(&componentsv1.GetComponentRequest{Id: id}))
	if err != nil {
		return nil, cliapp.WrapAPIError(fmt.Sprintf("get component %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Component == nil {
		return nil, fmt.Errorf("server returned no component")
	}
	return resp.Msg, nil
}

func (h *handlers) getReport(_ cliapp.OperationContext, msg *componentsv1.GetComponentResponse) cliapp.ListReport {
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched component %s.", msg.Component.LibraryId)},
		ResultsHeading: "Component",
		Results:        []string{formatComponent(msg.Component)},
	}
}

func (h *handlers) styles(ctx cliapp.RunContext) error {
	resp, err := h.client.ListDesignStyles(context.Background(), connect.NewRequest(&componentsv1.ListDesignStylesRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list design styles", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no styles response")
	}
	results := make([]string, 0, len(resp.Msg.Styles))
	for _, style := range resp.Msg.Styles {
		results = append(results, fmt.Sprintf("%s\t%s\t%s", style.Id, style.Name, strings.Join(style.Supports, ",")))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d design style(s).", len(resp.Msg.Styles))},
		ResultsHeading: "Design styles",
		Results:        results,
	})
}

func (h *handlers) validateStyleFit(ctx cliapp.RunContext) error {
	componentID := ctx.Positional("component-id")
	scenario := ctx.Positional("scenario")
	resp, err := h.client.ValidateStyleFit(context.Background(), connect.NewRequest(&componentsv1.ValidateStyleFitRequest{
		ComponentId: componentID,
		Scenario:    scenario,
		Version:     ctx.Flag("version"),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("validate style fit for component %q in scenario %q", componentID, scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no style-fit response")
	}
	results := []string{formatStyleFitVerdict(resp.Msg)}
	if strings.TrimSpace(resp.Msg.Detail) != "" {
		results = append(results, resp.Msg.Detail)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Style fit is %s for scenario %s.", formatStyleFitKind(resp.Msg.Kind), resp.Msg.Scenario)},
		ResultsHeading: "Style fit",
		Results:        results,
	})
}

func (h *handlers) init(ctx cliapp.RunContext) error {
	req := &componentsv1.InitializeComponentRequest{
		Slug:           ctx.Positional("slug"),
		LibraryId:      ctx.Flag("library-id"),
		DisplayName:    ctx.Flag("display-name"),
		Description:    ctx.Flag("description"),
		InitialVersion: ctx.Flag("version"),
		FileName:       ctx.Flag("file-name"),
	}
	if rawTags := ctx.Flag("tags"); rawTags != "" {
		req.Tags = splitCSV(rawTags)
	}
	if src := ctx.Flag("source-file"); src != "" {
		body, err := readSourceArg(src)
		if err != nil {
			return err
		}
		req.InitialSource = string(body)
	}
	resp, err := h.client.InitializeComponent(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("initialize component", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Component == nil {
		return fmt.Errorf("server returned no initialize response")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Initialized %s.", resp.Msg.Component.LibraryId)},
		ResultsHeading: "Created files",
		Results:        []string{resp.Msg.ManifestPath, resp.Msg.SourcePath},
		RetrievalHints: []string{
			"`components content-get " + resp.Msg.Component.Id + "` — inspect the generated source",
		},
	})
}

func (h *handlers) ingestCall(ctx cliapp.OperationContext) (*componentsv1.IngestComponentResponse, error) {
	req := &componentsv1.IngestComponentRequest{
		Scenario:               ctx.Positional("scenario"),
		SourceFile:             ctx.Positional("source-file"),
		Slug:                   ctx.Positional("slug"),
		DisplayName:            ctx.Flag("display-name"),
		Description:            ctx.Flag("description"),
		Slot:                   ctx.Flag("slot"),
		Version:                ctx.Flag("version"),
		AcceptBehaviorLoss:     ctx.Flag("accept-behavior-loss") == "true",
		ExperienceContractPath: ctx.Flag("experience-contract"),
	}
	if rawTags := ctx.Flag("tags"); rawTags != "" {
		req.Tags = splitCSV(rawTags)
	}
	if companions := ctx.Flag("companion-files"); companions != "" {
		req.SourceFiles = splitCSV(companions)
	}
	resp, err := h.client.IngestComponent(context.Background(), connect.NewRequest(req))
	if err != nil {
		// A blocked harvest (FailedPrecondition) already names the dropped
		// behaviors in its message; point the operator at the override.
		if connect.CodeOf(err) == connect.CodeFailedPrecondition {
			return nil, fmt.Errorf("%w\nRe-run with --accept-behavior-loss to record and accept these losses on the draft's parity report", err)
		}
		return nil, cliapp.WrapAPIError("ingest component", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Component == nil {
		return nil, fmt.Errorf("server returned no ingest response")
	}
	return resp.Msg, nil
}

func (h *handlers) ingestReport(_ cliapp.OperationContext, msg *componentsv1.IngestComponentResponse) cliapp.MutationReport {
	results := []string{msg.ManifestPath, msg.SourcePath}
	for _, finding := range msg.Findings {
		results = append(results, fmt.Sprintf("%s: %s", finding.Code, finding.Message))
	}
	result := []string{fmt.Sprintf("Ingested %s as draft %s.", msg.Component.LibraryId, msg.DraftVersion)}
	if report := msg.ParityReport; report != nil && report.Acknowledged && len(report.Findings) > 0 {
		result = append(result, fmt.Sprintf("Accepted %d behavior-loss finding(s); recorded as an acknowledged parity report.", len(report.Findings)))
	}
	return cliapp.MutationReport{Result: result, Changes: results, NextCommand: []string{fmt.Sprintf("Review %s before promoting the draft.", msg.ChecklistPath)}}
}

func (h *handlers) versionCreate(ctx cliapp.RunContext) error {
	req := &componentsv1.CreateComponentVersionRequest{
		ComponentId:             ctx.Positional("component-id"),
		Version:                 ctx.Positional("version"),
		FromVersion:             ctx.Flag("from-version"),
		FileName:                ctx.Flag("file-name"),
		ChangelogMd:             ctx.Flag("changelog"),
		AcknowledgeParityWaiver: ctx.Flag("acknowledge-parity-waiver") == "true",
	}
	switch {
	case ctx.Flag("draft") == "true":
		req.Intent = componentsv1.ComponentVersionIntent_COMPONENT_VERSION_INTENT_DRAFT
	}
	if src := ctx.Flag("source-file"); src != "" {
		body, err := readSourceArg(src)
		if err != nil {
			return err
		}
		req.Source = string(body)
	}
	if reportPath := ctx.Flag("parity-report"); reportPath != "" {
		body, err := readSourceArg(reportPath)
		if err != nil {
			return err
		}
		report := &componentsv1.IngestParityReport{}
		if err := protojson.Unmarshal(body, report); err != nil {
			return fmt.Errorf("decode --parity-report: %w", err)
		}
		req.ParityReport = report
	}
	resp, err := h.client.CreateComponentVersion(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("create component version", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Version == nil {
		return fmt.Errorf("server returned no version create response")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Created version %s.", resp.Msg.Version.Version)},
		ResultsHeading: "Source",
		Results:        []string{resp.Msg.SourcePath},
	})
}

func (h *handlers) versionBeginCall(ctx cliapp.OperationContext) (*componentsv1.BeginComponentVersionResponse, error) {
	resp, err := h.client.BeginComponentVersion(context.Background(), connect.NewRequest(&componentsv1.BeginComponentVersionRequest{
		Component: ctx.Positional("component"),
		Bump:      ctx.Flag("bump"),
		Version:   ctx.Flag("version"),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("begin component version", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Version == nil || resp.Msg.Component == nil {
		return nil, fmt.Errorf("server returned no component draft")
	}
	return resp.Msg, nil
}

func (h *handlers) versionBeginReport(_ cliapp.OperationContext, msg *componentsv1.BeginComponentVersionResponse) cliapp.MutationReport {
	results := append([]string(nil), msg.ArtifactPaths...)
	results = append(results, "Preview: "+msg.PreviewPath)
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Began %s draft %s.", msg.Component.LibraryId, msg.Version.Version)}, Changes: results, NextCommand: []string{fmt.Sprintf("`components check %s --version %s` — run the focused preflight", msg.Component.LibraryId, msg.Version.Version)}}
}

func (h *handlers) versionCheckCall(ctx cliapp.OperationContext) (*componentsv1.CheckComponentVersionResponse, error) {
	resp, err := h.client.CheckComponentVersion(context.Background(), connect.NewRequest(&componentsv1.CheckComponentVersionRequest{
		Component: ctx.Positional("component"),
		Version:   ctx.Flag("version"),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("check component version", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Component == nil {
		return nil, fmt.Errorf("server returned no component check")
	}
	return resp.Msg, nil
}

func (h *handlers) versionCheckReport(_ cliapp.OperationContext, msg *componentsv1.CheckComponentVersionResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.Checks)+1)
	for _, check := range msg.Checks {
		line := fmt.Sprintf("%s\t%s\t%s", check.Stage, check.Verdict, check.Message)
		if check.Remediation != "" {
			line += "\t" + check.Remediation
		}
		results = append(results, line)
	}
	results = append(results, "Preview: "+msg.PreviewPath)
	summary := "Component preflight passed."
	if !msg.Passed {
		summary = "Component preflight failed."
	}
	return cliapp.ListReport{Summary: []string{summary}, ResultsHeading: "Checks", Results: results}
}

func versionCheckOutcome(msg *componentsv1.CheckComponentVersionResponse) error {
	if msg == nil || msg.Component == nil {
		return fmt.Errorf("server returned no component check")
	}
	if !msg.Passed {
		return fmt.Errorf("component preflight failed for %s@%s", msg.Component.LibraryId, msg.Version)
	}
	return nil
}

func (h *handlers) versionPublishCall(ctx cliapp.OperationContext) (*componentsv1.PublishComponentVersionResponse, error) {
	resp, err := h.client.PublishComponentVersion(context.Background(), connect.NewRequest(&componentsv1.PublishComponentVersionRequest{
		Component:               ctx.Positional("component"),
		DraftVersion:            ctx.Flag("draft-version"),
		Version:                 ctx.Flag("version"),
		ChangelogMd:             ctx.Flag("changelog"),
		AcknowledgeParityWaiver: ctx.Flag("acknowledge-parity-waiver") != "",
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("publish component version", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Version == nil || resp.Msg.Component == nil {
		return nil, fmt.Errorf("server returned no published component version")
	}
	return resp.Msg, nil
}

func (h *handlers) versionPublishReport(_ cliapp.OperationContext, msg *componentsv1.PublishComponentVersionResponse) cliapp.MutationReport {
	results := append([]string(nil), msg.ArtifactPaths...)
	results = append(results, "Preview: "+msg.PreviewPath)
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Published %s@%s.", msg.Component.LibraryId, msg.Version.Version)}, Changes: results}
}

func patchVersion(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return version
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return version
	}
	return fmt.Sprintf("%s.%s.%d", parts[0], parts[1], patch+1)
}

func migrationDependencyRank(component *componentsv1.Component) int {
	if component == nil {
		return 0
	}
	// Lower-level assets must be republished before their consumers. The
	// SourcePath carries the catalog's stable dependency-layer ordering. The
	// generated Component message intentionally exposes only component versus
	// hook, which is not enough to order foundations, services, and primitives.
	parts := strings.Split(filepath.ToSlash(component.SourcePath), "/")
	if len(parts) == 0 {
		return 5
	}
	switch parts[0] {
	case "foundations":
		return 1
	case "hooks":
		return 2
	case "services", "adapters":
		return 3
	case "primitives":
		return 4
	default:
		return 5
	}
}

func (h *handlers) manifestUpdateCall(ctx cliapp.OperationContext) (*componentsv1.UpdateComponentManifestResponse, error) {
	req := &componentsv1.UpdateComponentManifestRequest{
		ComponentId:                    ctx.Positional("component"),
		DisplayName:                    ctx.Flag("display-name"),
		Description:                    ctx.Flag("description"),
		LatestVersion:                  ctx.Flag("latest-version"),
		DraftVersion:                   ctx.Flag("draft-version"),
		CatalogId:                      ctx.Flag("catalog-id"),
		Entry:                          ctx.Flag("entry"),
		ClearSupplementalJustification: ctx.BoolFlag("clear-supplemental-justification"),
		ClearCatalogId:                 ctx.BoolFlag("clear-catalog-id"),
	}
	req.ClearRetiredMajorAliases = ctx.Flag("clear-retired-major-aliases") != ""
	if raw := ctx.Flag("retired-major-aliases"); raw != "" {
		req.RetiredMajorAliases = splitCSV(raw)
	}
	if rawTags := ctx.Flag("tags"); rawTags != "" {
		req.Tags = splitCSV(rawTags)
	}
	if raw := ctx.Flag("deprecated-versions"); raw != "" {
		req.DeprecatedVersions = splitCSV(raw)
	}
	if raw := ctx.Flag("replaced-by"); raw != "" {
		req.ReplacedBy = splitCSV(raw)
	}
	resp, err := h.client.UpdateComponentManifest(context.Background(), connect.NewRequest(req))
	if err != nil {
		return nil, cliapp.WrapAPIError("update component manifest", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Component == nil {
		return nil, fmt.Errorf("server returned no manifest update response")
	}
	return resp.Msg, nil
}

func (h *handlers) manifestUpdateReport(_ cliapp.OperationContext, msg *componentsv1.UpdateComponentManifestResponse) cliapp.MutationReport {
	return cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Updated %s.", msg.Component.LibraryId)},
		Changes: []string{formatComponent(msg.Component)},
	}
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func readSourceArg(src string) ([]byte, error) {
	if src == "-" {
		return readAllStdin()
	}
	body, err := os.ReadFile(src)
	if err != nil {
		return nil, fmt.Errorf("read source %q: %w", src, err)
	}
	return body, nil
}

// contentGet calls ComponentsService.GetComponentContent and prints the
// source body to stdout. Human output writes the body as-is so it can
// be piped (e.g. `… content get <id> > Button.tsx`); --json emits the
// proto wire shape with body, source_path, and sha256.
func (h *handlers) contentGet(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetComponentContent(context.Background(), connect.NewRequest(&componentsv1.GetComponentContentRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get content for component %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no content response")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Read %s (sha256=%s).", resp.Msg.SourcePath, resp.Msg.Sha256)},
		ResultsHeading: "Content",
		Results:        []string{resp.Msg.Content},
	})
}

func (h *handlers) stories(ctx cliapp.RunContext) error {
	componentID := ctx.Positional("component-id")
	req := &componentsv1.ListComponentStoriesRequest{ComponentId: componentID, Version: ctx.Flag("version")}
	if raw := ctx.Flag("limit"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			return fmt.Errorf("--limit must be an integer (got %q)", raw)
		}
		req.Limit = int32(n)
	}
	resp, err := h.client.ListComponentStories(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("list stories for component %q", componentID), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no stories response")
	}
	results := make([]string, 0, len(resp.Msg.Stories))
	for _, story := range resp.Msg.Stories {
		results = append(results, formatStory(story))
	}
	summary := []string{fmt.Sprintf("Found %d story contract(s).", len(resp.Msg.Stories))}
	if len(resp.Msg.Warnings) > 0 {
		summary = append(summary, fmt.Sprintf("%d warning(s) reported; warnings do not block indexing.", len(resp.Msg.Warnings)))
		results = append(results, resp.Msg.Warnings...)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: summary, ResultsHeading: "Stories", Results: results})
}

// contentSet reads <file> from disk (or "-" for stdin) and calls
// ComponentsService.UpdateComponentContent. --expected-sha256 forwards
// the optimistic-concurrency guard.
func (h *handlers) contentSetCall(ctx cliapp.OperationContext) (*componentsv1.UpdateComponentContentResponse, error) {
	id := ctx.Positional("id")
	src := ctx.Positional("file")
	var body []byte
	var err error
	if src == "-" {
		body, err = readAllStdin()
	} else {
		body, err = os.ReadFile(src)
	}
	if err != nil {
		return nil, fmt.Errorf("read source %q: %w", src, err)
	}
	req := &componentsv1.UpdateComponentContentRequest{
		Id:             id,
		Path:           ctx.Flag("path"),
		Content:        string(body),
		ExpectedSha256: ctx.Flag("expected-sha256"),
	}
	resp, err := h.client.UpdateComponentContent(context.Background(), connect.NewRequest(req))
	if err != nil {
		return nil, cliapp.WrapAPIError(fmt.Sprintf("set content for component %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no update response")
	}
	return resp.Msg, nil
}

func (h *handlers) contentSetReport(_ cliapp.OperationContext, msg *componentsv1.UpdateComponentContentResponse) cliapp.MutationReport {
	return cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Wrote %s (sha256=%s).", msg.SourcePath, msg.Sha256)},
		NextCommand: []string{"`components index` — re-walk if the @libraryId header changed"},
	}
}

func readAllStdin() ([]byte, error) {
	return io.ReadAll(os.Stdin)
}

// formatComponent produces a one-line representation for the result block.
func formatComponent(c *componentsv1.Component) string {
	if c == nil {
		return "(nil)"
	}
	indexed := ""
	if c.IndexedAt != nil {
		indexed = c.IndexedAt.AsTime().Format(time.RFC3339)
	}
	tagsPart := ""
	if len(c.Tags) > 0 {
		tagsPart = " tags=[" + strings.Join(c.Tags, ",") + "]"
	}
	versionPart := ""
	if c.Version != "" {
		versionPart = " v" + c.Version
	}
	slotPart := ""
	if c.Slot != "" {
		slotPart = " slot=" + c.Slot
	}
	stylePart := ""
	if len(c.DesignStyles) > 0 {
		parts := make([]string, 0, len(c.DesignStyles))
		for _, style := range c.DesignStyles {
			parts = append(parts, fmt.Sprintf("%s:%s", style.StyleId, formatDesignAffinity(style.Affinity)))
		}
		stylePart = " styles=[" + strings.Join(parts, ",") + "]"
	}
	return fmt.Sprintf("%s — %s%s%s%s%s @ %s [indexed=%s]", c.LibraryId, c.DisplayName, versionPart, slotPart, stylePart, tagsPart, c.SourcePath, indexed)
}

func formatDesignAffinity(affinity componentsv1.DesignAffinity) string {
	switch affinity {
	case componentsv1.DesignAffinity_DESIGN_AFFINITY_NATIVE:
		return "native"
	case componentsv1.DesignAffinity_DESIGN_AFFINITY_COMPATIBLE:
		return "compatible"
	case componentsv1.DesignAffinity_DESIGN_AFFINITY_DISCOURAGED:
		return "discouraged"
	default:
		return "unspecified"
	}
}

func formatStyleFitKind(kind componentsv1.StyleFitVerdictKind) string {
	switch kind {
	case componentsv1.StyleFitVerdictKind_STYLE_FIT_VERDICT_KIND_OK:
		return "ok"
	case componentsv1.StyleFitVerdictKind_STYLE_FIT_VERDICT_KIND_INFO:
		return "info"
	case componentsv1.StyleFitVerdictKind_STYLE_FIT_VERDICT_KIND_WARN:
		return "warn"
	default:
		return "unspecified"
	}
}

func formatStyleFitVerdict(v *componentsv1.ValidateStyleFitResponse) string {
	if v == nil {
		return "(nil)"
	}
	version := ""
	if v.Version != "" {
		version = " version=" + v.Version
	}
	affinity := formatDesignAffinity(v.Affinity)
	if affinity == "unspecified" {
		affinity = "none"
	}
	return fmt.Sprintf("%s scenario=%s style=%s component=%s%s affinity=%s",
		formatStyleFitKind(v.Kind), v.Scenario, v.ScenarioStyle, v.ComponentId, version, affinity)
}

func formatStory(story *componentsv1.ComponentStory) string {
	if story == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s — %s %s schema=%d @ %s", story.LibraryId, story.Version, story.Kind, story.SchemaVersion, story.SourcePath)
}

func (h *handlers) retireCall(ctx cliapp.OperationContext) (*componentsv1.RetireComponentResponse, error) {
	resp, err := h.client.RetireComponent(context.Background(), connect.NewRequest(&componentsv1.RetireComponentRequest{LibraryId: ctx.Positional("component"), Apply: ctx.BoolFlag("apply")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("retire component", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no retirement response")
	}
	return resp.Msg, nil
}

func (h *handlers) retireReport(_ cliapp.OperationContext, msg *componentsv1.RetireComponentResponse) cliapp.MutationReport {
	report := cliapp.MutationReport{Result: []string{fmt.Sprintf("%s: ready=%t retired=%t", msg.LibraryId, msg.Ready, msg.Retired)}}
	if msg.Error != "" {
		report.Result = append(report.Result, msg.Error)
	}
	for _, path := range []string{msg.SnapshotPath, msg.SourceArchivePath, msg.CatalogArchivePath} {
		if path != "" {
			report.Changes = append(report.Changes, path)
		}
	}
	for _, blocker := range append(append(append([]string{}, msg.RequiredBy...), msg.SuggestedBy...), msg.SourceReferences...) {
		report.Result = append(report.Result, "Consumer: "+blocker)
	}
	return report
}

func retireOutcome(ctx cliapp.OperationContext, msg *componentsv1.RetireComponentResponse) error {
	if msg.Error != "" {
		return fmt.Errorf("retirement: %s", msg.Error)
	}
	if ctx.BoolFlag("apply") && !msg.Retired {
		return fmt.Errorf("component was not retired")
	}
	if !ctx.BoolFlag("apply") && !msg.Ready {
		return fmt.Errorf("component has retirement blockers")
	}
	return nil
}
