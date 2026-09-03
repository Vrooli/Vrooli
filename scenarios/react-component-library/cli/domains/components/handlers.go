package components

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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
	"github.com/vrooli/cli-core/cliutil"
	sharedlibspec "github.com/vrooli/vrooli/packages/react-component-library/libspec"
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

func (h *handlers) testRun(ctx cliapp.RunContext) error {
	resp, err := h.testClient.RunComponentTest(context.Background(), connect.NewRequest(&componenttestsv1.RunComponentTestRequest{ComponentId: ctx.Positional("component-id"), Version: ctx.Flag("version"), IncludeClosure: ctx.Flag("closure") == "true"}))
	if err != nil {
		return cliapp.WrapAPIError("run component test", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Report == nil {
		return fmt.Errorf("server returned no component test report")
	}
	label := "Component test completed."
	if resp.Msg.Reused {
		label = fmt.Sprintf("Component test reused revision %s since report %s.", shortRevision(resp.Msg.SourceRevision), resp.Msg.Report.Id)
	}
	return renderTestReport(ctx, resp.Msg.Report, label)
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

func (h *handlers) testShow(ctx cliapp.RunContext) error {
	resp, err := h.testClient.GetComponentTestReport(context.Background(), connect.NewRequest(&componenttestsv1.GetComponentTestReportRequest{Id: ctx.Positional("report-id")}))
	if err != nil {
		return cliapp.WrapAPIError("get component test report", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Report == nil {
		return fmt.Errorf("server returned no component test report")
	}
	return renderTestReport(ctx, resp.Msg.Report, "Component test report.")
}

func (h *handlers) testRerun(ctx cliapp.RunContext) error {
	resp, err := h.testClient.RerunComponentTest(context.Background(), connect.NewRequest(&componenttestsv1.RerunComponentTestRequest{ReportId: ctx.Positional("report-id")}))
	if err != nil {
		return cliapp.WrapAPIError("rerun component test", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Report == nil {
		return fmt.Errorf("server returned no component test report")
	}
	return renderTestReport(ctx, resp.Msg.Report, "Component test rerun completed.")
}

func (h *handlers) testList(ctx cliapp.RunContext) error {
	limit := int32(0)
	if raw := ctx.Flag("limit"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			return fmt.Errorf("--limit must be an integer (got %q)", raw)
		}
		limit = int32(value)
	}
	resp, err := h.testClient.ListComponentTestReports(context.Background(), connect.NewRequest(&componenttestsv1.ListComponentTestReportsRequest{ComponentId: ctx.Positional("component-id"), Limit: limit}))
	if err != nil {
		return cliapp.WrapAPIError("list component test reports", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no component test reports")
	}
	results := make([]string, 0, len(resp.Msg.Reports))
	for _, report := range resp.Msg.Reports {
		results = append(results, fmt.Sprintf("%s\t%s\t%s\t%s", report.Id, report.Verdict, report.RootVersion, report.CreatedAt.AsTime().Format(time.RFC3339)))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d component test report(s).", len(results))}, ResultsHeading: "Reports", Results: results, RetrievalHints: []string{"`components test-show <report-id>` — inspect stages and remediation"}})
}

func (h *handlers) sweep(ctx cliapp.RunContext) error {
	resp, err := h.testClient.SweepComponentTests(context.Background(), connect.NewRequest(&componenttestsv1.SweepComponentTestsRequest{
		Resume: ctx.BoolFlag("resume"), IncludeClosure: ctx.BoolFlag("closure"), ComponentId: ctx.Flag("component-id"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("sweep component tests", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no component test sweep response")
	}
	rows := make([]string, 0, len(resp.Msg.Reports))
	for _, report := range resp.Msg.Reports {
		if report == nil {
			continue
		}
		rows = append(rows, fmt.Sprintf("%s\t%s@%s\t%s", report.Id, report.RootLibraryId, report.RootVersion, report.Verdict))
	}
	mode := "fresh"
	if resp.Msg.Skipped > 0 || ctx.BoolFlag("resume") {
		mode = "resume"
	}
	summary := fmt.Sprintf("Component sweep (%s): planned=%d started=%d skipped=%d passed=%d failed=%d blocked=%d complete=%t.", mode, resp.Msg.Planned, resp.Msg.Started, resp.Msg.Skipped, resp.Msg.Passed, resp.Msg.Failed, resp.Msg.Blocked, resp.Msg.Complete)
	if len(resp.Msg.Errors) > 0 {
		rows = append(rows, resp.Msg.Errors...)
	}
	if err := cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{summary}, ResultsHeading: "Version reports", Results: rows, RetrievalHints: []string{"`components sweep --resume` — retry only untested or blocked versions"}}); err != nil {
		return err
	}
	if resp.Msg.Blocked > 0 || !resp.Msg.Complete {
		return fmt.Errorf("component sweep is not complete: %d blocked result(s), %d error(s)", resp.Msg.Blocked, len(resp.Msg.Errors))
	}
	return nil
}

func renderTestReport(ctx cliapp.RunContext, report *componenttestsv1.ComponentTestReport, summary string) error {
	results := make([]string, 0, len(report.Results))
	for _, result := range report.Results {
		line := fmt.Sprintf("%s\t%s@%s\t%s", result.Stage, result.AssetLibraryId, result.Version, result.Verdict)
		if result.Message != "" {
			line += "\t" + result.Message
		}
		if result.Remediation != "" {
			line += "\t" + result.Remediation
		}
		results = append(results, line)
	}
	return cliapp.RenderProtoList(ctx, report, cliapp.ListReport{Summary: []string{summary, fmt.Sprintf("Report %s: %s.", report.Id, report.Verdict)}, ResultsHeading: "Stages", Results: results, RetrievalHints: []string{"`components test-show " + report.Id + "` — retrieve this durable report", "`components test-rerun " + report.Id + "` — rerun the same pinned closure"}})
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
			"`components get-by-library-id <libraryId>` — inspect a single entry",
		},
	})
}

// list calls ComponentsService.ListComponents with optional filter flags.
func (h *handlers) list(ctx cliapp.RunContext) error {
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
			return fmt.Errorf("--asset-kind must be component or hook (got %q)", rawKind)
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
			return fmt.Errorf("--limit must be an integer (got %q)", raw)
		}
		req.Limit = int32(n)
	}
	resp, err := h.client.ListComponents(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("list components", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no list response")
	}
	results := make([]string, 0, len(resp.Msg.Components))
	for _, c := range resp.Msg.Components {
		results = append(results, formatComponent(c))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d component(s).", len(resp.Msg.Components))},
		ResultsHeading: "Components",
		Results:        results,
		RetrievalHints: []string{
			"`components get <id>` — show a single component",
			"`components index` — refresh from disk",
		},
	})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetComponent(context.Background(), connect.NewRequest(&componentsv1.GetComponentRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get component %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Component == nil {
		return fmt.Errorf("server returned no component")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched component %s.", resp.Msg.Component.LibraryId)},
		ResultsHeading: "Component",
		Results:        []string{formatComponent(resp.Msg.Component)},
	})
}

func (h *handlers) getByLibraryID(ctx cliapp.RunContext) error {
	libid := ctx.Positional("library-id")
	resp, err := h.client.GetComponentByLibraryId(context.Background(), connect.NewRequest(&componentsv1.GetComponentByLibraryIdRequest{LibraryId: libid}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get component %q", libid), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Component == nil {
		return fmt.Errorf("server returned no component")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched component %s.", resp.Msg.Component.LibraryId)},
		ResultsHeading: "Component",
		Results:        []string{formatComponent(resp.Msg.Component)},
	})
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

func (h *handlers) ingest(ctx cliapp.RunContext) error {
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
			return fmt.Errorf("%w\nRe-run with --accept-behavior-loss to record and accept these losses on the draft's parity report", err)
		}
		return cliapp.WrapAPIError("ingest component", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Component == nil {
		return fmt.Errorf("server returned no ingest response")
	}
	results := []string{resp.Msg.ManifestPath, resp.Msg.SourcePath}
	for _, finding := range resp.Msg.Findings {
		results = append(results, fmt.Sprintf("%s: %s", finding.Code, finding.Message))
	}
	summary := []string{fmt.Sprintf("Ingested %s as draft %s.", resp.Msg.Component.LibraryId, resp.Msg.DraftVersion)}
	if report := resp.Msg.ParityReport; report != nil && report.Acknowledged && len(report.Findings) > 0 {
		summary = append(summary, fmt.Sprintf("Accepted %d behavior-loss finding(s); recorded as an acknowledged parity report.", len(report.Findings)))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Files and findings",
		Results:        results,
		RetrievalHints: []string{fmt.Sprintf("Review %s before promoting the draft.", resp.Msg.ChecklistPath)},
	})
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

func (h *handlers) versionBegin(ctx cliapp.RunContext) error {
	resp, err := h.client.BeginComponentVersion(context.Background(), connect.NewRequest(&componentsv1.BeginComponentVersionRequest{
		Component: ctx.Positional("component"),
		Bump:      ctx.Flag("bump"),
		Version:   ctx.Flag("version"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("begin component version", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Version == nil || resp.Msg.Component == nil {
		return fmt.Errorf("server returned no component draft")
	}
	results := append([]string(nil), resp.Msg.ArtifactPaths...)
	results = append(results, "Preview: "+resp.Msg.PreviewPath)
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Began %s draft %s.", resp.Msg.Component.LibraryId, resp.Msg.Version.Version)},
		ResultsHeading: "Draft artifacts",
		Results:        results,
		RetrievalHints: []string{fmt.Sprintf("`components check %s --version %s` — run the focused preflight", resp.Msg.Component.LibraryId, resp.Msg.Version.Version)},
	})
}

func (h *handlers) versionCheck(ctx cliapp.RunContext) error {
	resp, err := h.client.CheckComponentVersion(context.Background(), connect.NewRequest(&componentsv1.CheckComponentVersionRequest{
		Component: ctx.Positional("component"),
		Version:   ctx.Flag("version"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("check component version", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Component == nil {
		return fmt.Errorf("server returned no component check")
	}
	results := make([]string, 0, len(resp.Msg.Checks)+1)
	for _, check := range resp.Msg.Checks {
		line := fmt.Sprintf("%s\t%s\t%s", check.Stage, check.Verdict, check.Message)
		if check.Remediation != "" {
			line += "\t" + check.Remediation
		}
		results = append(results, line)
	}
	results = append(results, "Preview: "+resp.Msg.PreviewPath)
	summary := "Component preflight passed."
	if !resp.Msg.Passed {
		summary = "Component preflight failed."
	}
	if err := cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{summary}, ResultsHeading: "Checks", Results: results}); err != nil {
		return err
	}
	if !resp.Msg.Passed {
		return fmt.Errorf("component preflight failed for %s@%s", resp.Msg.Component.LibraryId, resp.Msg.Version)
	}
	return nil
}

func (h *handlers) versionPublish(ctx cliapp.RunContext) error {
	resp, err := h.client.PublishComponentVersion(context.Background(), connect.NewRequest(&componentsv1.PublishComponentVersionRequest{
		Component:               ctx.Positional("component"),
		DraftVersion:            ctx.Flag("draft-version"),
		Version:                 ctx.Flag("version"),
		ChangelogMd:             ctx.Flag("changelog"),
		AcknowledgeParityWaiver: ctx.Flag("acknowledge-parity-waiver") != "",
	}))
	if err != nil {
		return cliapp.WrapAPIError("publish component version", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Version == nil || resp.Msg.Component == nil {
		return fmt.Errorf("server returned no published component version")
	}
	results := append([]string(nil), resp.Msg.ArtifactPaths...)
	results = append(results, "Preview: "+resp.Msg.PreviewPath)
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Published %s@%s.", resp.Msg.Component.LibraryId, resp.Msg.Version.Version)},
		ResultsHeading: "Published artifacts",
		Results:        results,
	})
}

func (h *handlers) republishDependents(ctx cliapp.RunContext) error {
	asset := strings.TrimSpace(ctx.Positional("asset"))
	if !strings.Contains(asset, ":") {
		asset = "react-component-library:" + asset
	}
	targetVersion := strings.TrimSpace(ctx.Positional("version"))
	listed, err := h.client.ListComponents(context.Background(), connect.NewRequest(&componentsv1.ListComponentsRequest{Limit: 1000}))
	if err != nil {
		return cliapp.WrapAPIError("list dependent components", err, nil)
	}
	var dependents []*componentsv1.Component
	for _, component := range listed.Msg.Components {
		// The indexed dependency version is the resolved lock, not the
		// authored selector. Inspect the latest authored source as well so a
		// one-time migration can republish exact pins even when the selected
		// release is already the dependency's latest release.
		if hasExactLibrarySpecifier(component, asset) {
			dependents = append(dependents, component)
		}
	}
	sort.Slice(dependents, func(i, j int) bool {
		if dependents[i].AssetKind == dependents[j].AssetKind {
			return dependents[i].LibraryId < dependents[j].LibraryId
		}
		return dependents[i].AssetKind < dependents[j].AssetKind
	})
	results := make([]string, 0, len(dependents))
	apply := ctx.Flag("apply") != ""
	for _, component := range dependents {
		line := fmt.Sprintf("%s current=%s", component.LibraryId, component.LatestVersion)
		if apply {
			draft, beginErr := h.client.BeginComponentVersion(context.Background(), connect.NewRequest(&componentsv1.BeginComponentVersionRequest{Component: component.LibraryId, Bump: "patch"}))
			if beginErr != nil {
				return cliapp.WrapAPIError("begin dependent draft", beginErr, nil)
			}
			if normalizeErr := h.normalizeDraftLibrarySpecifiers(component, draft.Msg.Version.Version, listed.Msg.Components); normalizeErr != nil {
				return normalizeErr
			}
			published, publishErr := h.client.PublishComponentVersion(context.Background(), connect.NewRequest(&componentsv1.PublishComponentVersionRequest{Component: component.LibraryId, DraftVersion: draft.Msg.Version.Version}))
			if publishErr != nil {
				return cliapp.WrapAPIError("publish dependent", publishErr, nil)
			}
			line += " published=" + published.Msg.Version.Version
		}
		results = append(results, line)
	}
	mode := "dry-run"
	if apply {
		mode = "applied"
	}
	return cliapp.RenderProtoList(ctx, listed.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Dependent republish %s: %d asset(s) contain exact selectors for %s (requested target %s).", mode, len(dependents), asset, targetVersion)},
		ResultsHeading: "Dependents in dependency-rank order", Results: results,
	})
}

func (h *handlers) migrateSpecifiers(ctx cliapp.RunContext) error {
	listed, err := h.client.ListComponents(context.Background(), connect.NewRequest(&componentsv1.ListComponentsRequest{Limit: 1000}))
	if err != nil {
		return cliapp.WrapAPIError("list components for specifier migration", err, nil)
	}
	dependents := make([]*componentsv1.Component, 0)
	for _, component := range listed.Msg.Components {
		if hasAnyExactLibrarySpecifier(component) {
			dependents = append(dependents, component)
		}
	}
	sort.Slice(dependents, func(i, j int) bool { return dependents[i].LibraryId < dependents[j].LibraryId })
	apply := ctx.Flag("apply") != ""
	results := make([]string, 0, len(dependents))
	for _, component := range dependents {
		line := fmt.Sprintf("%s current=%s", component.LibraryId, component.LatestVersion)
		if apply {
			draft, beginErr := h.client.BeginComponentVersion(context.Background(), connect.NewRequest(&componentsv1.BeginComponentVersionRequest{Component: component.LibraryId, Bump: "patch"}))
			if beginErr != nil {
				return cliapp.WrapAPIError("begin specifier migration draft", beginErr, nil)
			}
			if normalizeErr := h.normalizeDraftLibrarySpecifiers(component, draft.Msg.Version.Version, listed.Msg.Components); normalizeErr != nil {
				return normalizeErr
			}
			published, publishErr := h.client.PublishComponentVersion(context.Background(), connect.NewRequest(&componentsv1.PublishComponentVersionRequest{Component: component.LibraryId, DraftVersion: draft.Msg.Version.Version}))
			if publishErr != nil {
				return cliapp.WrapAPIError("publish specifier migration", publishErr, nil)
			}
			line += " published=" + published.Msg.Version.Version
		}
		results = append(results, line)
	}
	mode := "dry-run"
	if apply {
		mode = "applied"
	}
	return cliapp.RenderProtoList(ctx, listed.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Specifier migration %s: %d latest asset(s) contain exact selectors.", mode, len(dependents))},
		ResultsHeading: "Assets requiring major-line migration", Results: results,
	})
}

type specifierMigrationPlanItem struct {
	Order          int    `json:"order"`
	LibraryID      string `json:"library_id"`
	Current        string `json:"current_version"`
	Next           string `json:"next_version"`
	DependencyRank int    `json:"dependency_rank"`
}

// republishPlan is deliberately read-only. It records the deterministic
// migration order before any draft is opened, so an interrupted migration can
// be resumed from an explicit plan rather than from a partially observed tree.
func (h *handlers) republishPlan(ctx cliapp.RunContext) error {
	listed, err := h.client.ListComponents(context.Background(), connect.NewRequest(&componentsv1.ListComponentsRequest{Limit: 1000}))
	if err != nil {
		return cliapp.WrapAPIError("list components for republish plan", err, nil)
	}
	items := make([]specifierMigrationPlanItem, 0)
	for _, component := range listed.Msg.Components {
		if !hasAnyExactLibrarySpecifier(component) {
			continue
		}
		rank := migrationDependencyRank(component)
		items = append(items, specifierMigrationPlanItem{
			LibraryID: component.LibraryId, Current: component.LatestVersion,
			Next: patchVersion(component.LatestVersion), DependencyRank: rank,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].DependencyRank != items[j].DependencyRank {
			return items[i].DependencyRank < items[j].DependencyRank
		}
		return items[i].LibraryID < items[j].LibraryID
	})
	for i := range items {
		items[i].Order = i + 1
	}
	payload := struct {
		SchemaVersion int                          `json:"schema_version"`
		Count         int                          `json:"count"`
		Items         []specifierMigrationPlanItem `json:"items"`
	}{SchemaVersion: 1, Count: len(items), Items: items}
	if ctx.JSON() {
		return cliapp.PrintJSON(ctx.Stdout(), payload)
	}
	results := make([]string, 0, len(items))
	for _, item := range items {
		results = append(results, fmt.Sprintf("%03d rank=%d %s current=%s next=%s", item.Order, item.DependencyRank, item.LibraryID, item.Current, item.Next))
	}
	return ctx.RenderList(cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Specifier republish plan: %d asset(s).", len(items))},
		ResultsHeading: "Dependency-rank-ordered migration", Results: results,
	})
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

func hasExactLibrarySpecifier(component *componentsv1.Component, asset string) bool {
	if component == nil || component.SourcePath == "" {
		return false
	}
	root := cliutil.ResolveRepoRoot()
	assetRoot := filepath.Join(root, "scenarios/react-component-library/library", filepath.FromSlash(component.SourcePath))
	marker := string(filepath.Separator) + "versions" + string(filepath.Separator)
	if index := strings.Index(assetRoot, marker); index >= 0 {
		assetRoot = assetRoot[:index]
	}
	manifestPath := filepath.Join(assetRoot, "component.json")
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		return false
	}
	var metadata struct {
		Latest string `json:"latest"`
	}
	if err := json.Unmarshal(manifest, &metadata); err != nil || metadata.Latest == "" {
		return false
	}
	versionRoot := filepath.Join(assetRoot, "versions", metadata.Latest)
	entries, err := os.ReadDir(versionRoot)
	if err != nil {
		return false
	}
	name := strings.TrimPrefix(asset, "react-component-library:")
	exact := regexp.MustCompile(`@vrooli/react-component-library/` + regexp.QuoteMeta(name) + `/\d+\.\d+\.\d+`)
	for _, entry := range entries {
		if entry.IsDir() || (filepath.Ext(entry.Name()) != ".ts" && filepath.Ext(entry.Name()) != ".tsx") {
			continue
		}
		body, readErr := os.ReadFile(filepath.Join(versionRoot, entry.Name()))
		if readErr == nil && exact.Match(body) {
			return true
		}
	}
	return false
}

func hasAnyExactLibrarySpecifier(component *componentsv1.Component) bool {
	if component == nil || component.SourcePath == "" {
		return false
	}
	root := cliutil.ResolveRepoRoot()
	assetRoot := filepath.Join(root, "scenarios/react-component-library/library", filepath.FromSlash(component.SourcePath))
	marker := string(filepath.Separator) + "versions" + string(filepath.Separator)
	if index := strings.Index(assetRoot, marker); index >= 0 {
		assetRoot = assetRoot[:index]
	}
	manifest, err := os.ReadFile(filepath.Join(assetRoot, "component.json"))
	if err != nil {
		return false
	}
	var metadata struct {
		Latest string `json:"latest"`
	}
	if json.Unmarshal(manifest, &metadata) != nil || metadata.Latest == "" {
		return false
	}
	versionRoot := filepath.Join(assetRoot, "versions", metadata.Latest)
	exact := regexp.MustCompile(`@vrooli/react-component-library/[A-Za-z][A-Za-z0-9-]*/\d+\.\d+\.\d+`)
	var found bool
	_ = filepath.WalkDir(versionRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || found || (filepath.Ext(entry.Name()) != ".ts" && filepath.Ext(entry.Name()) != ".tsx") {
			return nil
		}
		body, readErr := os.ReadFile(path)
		if readErr == nil && exact.Match(body) {
			found = true
		}
		return nil
	})
	return found
}

// normalizeDraftLibrarySpecifiers changes only the newly-created draft. A
// republish must migrate copied historical source whose exact dependency has
// since been retired; the release publisher then freezes the major-line
// imports into the new release and regenerates its exact-resolution lock.
func (h *handlers) normalizeDraftLibrarySpecifiers(component *componentsv1.Component, draftVersion string, candidates []*componentsv1.Component) error {
	latest := make(map[string]string, len(candidates))
	for _, candidate := range candidates {
		latest[candidate.LibraryId] = candidate.LatestVersion
	}
	repoRoot := cliutil.ResolveRepoRoot()
	versionDir := filepath.Dir(filepath.Join(repoRoot, "scenarios/react-component-library/library", filepath.FromSlash(component.SourcePath)))
	draftDir := filepath.Join(filepath.Dir(versionDir), draftVersion)
	entries, err := os.ReadDir(draftDir)
	if err != nil {
		return fmt.Errorf("read dependent draft %s: %w", component.LibraryId, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || (filepath.Ext(entry.Name()) != ".ts" && filepath.Ext(entry.Name()) != ".tsx") {
			continue
		}
		path := filepath.Join(draftDir, entry.Name())
		body, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read dependent draft file %s: %w", path, err)
		}
		normalized := sharedlibspec.Rewrite(string(body), func(specifier sharedlibspec.Specifier) string {
			active := latest["react-component-library:"+specifier.Name]
			if active == "" {
				return ""
			}
			return sharedlibspec.Prefix + specifier.Name + "/" + strings.Split(active, ".")[0]
		})
		if normalized == string(body) {
			continue
		}
		if _, err := h.client.UpdateComponentContent(context.Background(), connect.NewRequest(&componentsv1.UpdateComponentContentRequest{Id: component.LibraryId, Path: entry.Name(), Content: normalized})); err != nil {
			return cliapp.WrapAPIError("normalize dependent draft", err, nil)
		}
	}
	contractPath := filepath.Join(draftDir, "experience-contract.json")
	if _, err := os.Stat(contractPath); err == nil {
		catalogSlug := strings.ToLower(strings.ReplaceAll(component.CatalogId, ".", "-"))
		contract := fmt.Sprintf("{\n  \"kind\": \"experience-reference\",\n  \"component\": \"%s\",\n  \"ref\": \"../../../../../experience/components/%s.json\"\n}\n", component.CatalogId, catalogSlug)
		if _, err := h.client.UpdateComponentContent(context.Background(), connect.NewRequest(&componentsv1.UpdateComponentContentRequest{Id: component.LibraryId, Path: "experience-contract.json", Content: contract})); err != nil {
			return cliapp.WrapAPIError("normalize dependent experience reference", err, nil)
		}
	}
	return nil
}

func (h *handlers) manifestUpdate(ctx cliapp.RunContext) error {
	req := &componentsv1.UpdateComponentManifestRequest{
		ComponentId:                    ctx.Positional("component-id"),
		DisplayName:                    ctx.Flag("display-name"),
		Description:                    ctx.Flag("description"),
		LatestVersion:                  ctx.Flag("latest-version"),
		DraftVersion:                   ctx.Flag("draft-version"),
		CatalogId:                      ctx.Flag("catalog-id"),
		ClearSupplementalJustification: ctx.Flag("clear-supplemental-justification") != "",
		ClearCatalogId:                 ctx.Flag("clear-catalog-id") != "",
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
		return cliapp.WrapAPIError("update component manifest", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Component == nil {
		return fmt.Errorf("server returned no manifest update response")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Updated %s.", resp.Msg.Component.LibraryId)},
		ResultsHeading: "Component",
		Results:        []string{formatComponent(resp.Msg.Component)},
	})
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

func (h *handlers) versions(ctx cliapp.RunContext) error {
	componentID := ctx.Positional("component-id")
	req := &componentsv1.ListComponentVersionsRequest{ComponentId: componentID}
	if raw := ctx.Flag("limit"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			return fmt.Errorf("--limit must be an integer (got %q)", raw)
		}
		req.Limit = int32(n)
	}
	resp, err := h.client.ListComponentVersions(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("list versions for component %q", componentID), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no versions response")
	}
	results := make([]string, 0, len(resp.Msg.Versions))
	for _, v := range resp.Msg.Versions {
		results = append(results, formatVersion(v))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d version(s).", len(resp.Msg.Versions))},
		ResultsHeading: "Versions",
		Results:        results,
	})
}

func (h *handlers) showVersion(ctx cliapp.RunContext) error {
	componentID := ctx.Positional("component-id")
	version := ctx.Positional("version")
	resp, err := h.client.GetComponentVersionContent(context.Background(), connect.NewRequest(&componentsv1.GetComponentVersionContentRequest{
		ComponentId: componentID,
		Version:     version,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("show version %q for component %q", version, componentID), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no version response")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Read %s.", resp.Msg.Version.SourcePath)},
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
func (h *handlers) contentSet(ctx cliapp.RunContext) error {
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
		return fmt.Errorf("read source %q: %w", src, err)
	}
	req := &componentsv1.UpdateComponentContentRequest{
		Id:             id,
		Path:           ctx.Flag("path"),
		Content:        string(body),
		ExpectedSha256: ctx.Flag("expected-sha256"),
	}
	resp, err := h.client.UpdateComponentContent(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("set content for component %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no update response")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary: []string{fmt.Sprintf("Wrote %s (sha256=%s).", resp.Msg.SourcePath, resp.Msg.Sha256)},
		RetrievalHints: []string{
			"`components index` — re-walk if the @libraryId header changed",
		},
	})
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

func formatVersion(v *componentsv1.ComponentVersion) string {
	if v == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s — %s %s sha=%s @ %s", v.Id, v.Version, v.Status.String(), v.ContentSha256, v.SourcePath)
}

func formatStory(story *componentsv1.ComponentStory) string {
	if story == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s — %s %s schema=%d @ %s", story.LibraryId, story.Version, story.Kind, story.SchemaVersion, story.SourcePath)
}
