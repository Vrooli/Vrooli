package validation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"unit-health/internal/adapterregistry"
	"unit-health/internal/adapters"
	"unit-health/internal/testquality"
)

type qualityCollection struct {
	collector adapters.QualityCollector
	input     adapters.QualityInput
}

// prepareNativeQuality binds evidence to this invocation, not a timestamp or
// an appendable historic report. Typecheck commands do not assert behavior.
func prepareNativeQuality(plan *ExecutionPlan, workspaces []Workspace) ([]qualityCollection, error) {
	var collections []qualityCollection
	registry := adapterregistry.Default()
	for i := range plan.Commands {
		command := &plan.Commands[i]
		if command.TestKind == "typecheck" {
			continue
		}
		for _, ws := range workspaces {
			if ws.ID != command.WorkspaceID {
				continue
			}
			analyzer, ok := registry.Resolve(ws.AdapterID, ws.Language, ws.Framework)
			if !ok {
				continue
			}
			collector, ok := analyzer.(adapters.QualityCollector)
			if !ok {
				continue
			}
			var nonce [16]byte
			if _, err := rand.Read(nonce[:]); err != nil {
				return nil, fmt.Errorf("quality run identity: %w", err)
			}
			token := hex.EncodeToString(nonce[:])
			artifact := collector.QualityArtifact(ws.RootPath, token)
			env := make(map[string]string, len(command.Env)+2)
			for key, value := range command.Env {
				env[key] = value
			}
			env["VROOLI_TEST_RUN_ID"], env["VROOLI_TEST_QUALITY_OUTPUT"] = token, artifact.Path
			command.Env = env
			command.Artifacts = append(command.Artifacts, Artifact{Label: artifact.Label, Kind: artifact.Kind, Reference: artifact.Path})
			collections = append(collections, qualityCollection{collector, adapters.QualityInput{Workspace: ws.ID, Root: ws.RootPath, RunID: token, Artifact: artifact, TestKind: command.TestKind}})
			break
		}
	}
	return collections, nil
}

func collectNativeQuality(collections []qualityCollection, executed bool) *testquality.Report {
	return collectNativeQualityWithContext(context.Background(), collections, executed)
}

func collectNativeQualityWithContext(ctx context.Context, collections []qualityCollection, executed bool, sourceRows ...testquality.Result) *testquality.Report {
	report, _, _ := collectNativeQualityEvidence(ctx, collections, executed, sourceRows)
	return report
}

func collectNativeQualityEvidence(ctx context.Context, collections []qualityCollection, executed bool, sourceRows []testquality.Result) (*testquality.Report, []testquality.TestLinks, testquality.Reason) {
	var links []testquality.TestLinks
	linkReason := testquality.ReasonNone
	if len(collections) == 0 {
		linkReason = testquality.UnsupportedAdapter
	}
	catalog, err := testquality.LoadCatalog()
	if err != nil {
		return &testquality.Report{SchemaVersion: testquality.SchemaVersion, UnavailableReason: testquality.MissingInput}, nil, testquality.MissingInput
	}
	results := append([]testquality.Result(nil), sourceRows...)
	var unavailable []testquality.Reason
	if len(collections) == 0 && len(sourceRows) == 0 {
		unavailable = append(unavailable, testquality.UnsupportedAdapter)
	}
	for _, collection := range collections {
		collection.input.Executed = executed
		var rows []testquality.Result
		var reason testquality.Reason
		if collector, ok := collection.collector.(adapters.QualityEvidenceCollector); ok {
			var observed []testquality.TestLinks
			rows, observed, reason = collector.CollectQualityEvidence(collection.input)
			links = append(links, observed...)
			if reason != testquality.ReasonNone {
				linkReason = reason
			}
			if reason == testquality.ReasonNone && len(observed) == 0 {
				linkReason = testquality.MissingInput
			}
		} else {
			rows, reason = collection.collector.CollectQuality(collection.input)
			linkReason = testquality.UnsupportedAdapter
		}
		results = append(results, rows...)
		if syntax, ok := collection.collector.(adapters.SyntaxCollector); ok {
			syntaxRows, syntaxReason := syntax.CollectSyntax(ctx, collection.input)
			results = append(results, syntaxRows...)
			if syntaxReason != testquality.ReasonNone {
				unavailable = append(unavailable, syntaxReason)
			}
		}
		if reason != testquality.ReasonNone {
			unavailable = append(unavailable, reason)
		}
	}
	results = catalog.ApplyCatalogEnforcement(results)
	// Never silently truncate or invent discovered tests for unavailable artifacts.
	report, err := testquality.BuildReport(catalog.Version, results, 10000, "")
	if err != nil {
		reason := testquality.MissingInput
		if len(results) > 10000 {
			reason = testquality.ResolutionLimit
		}
		return &testquality.Report{SchemaVersion: testquality.SchemaVersion, CatalogVersion: catalog.Version, UnavailableReason: reason}, links, linkReason
	}
	for _, reason := range unavailable {
		report.MarkIncomplete(reason)
	}
	return &report, links, linkReason
}

func analyzeSourceQuality(ctx context.Context, workspaces []Workspace) ([]testquality.Result, testquality.Reason) {
	rows, _, reasons := analyzeSourceEvidence(ctx, workspaces)
	if len(reasons) > 0 {
		return rows, reasons[0]
	}
	return rows, testquality.ReasonNone
}

func analyzeSourceEvidence(ctx context.Context, workspaces []Workspace) ([]testquality.Result, []testquality.TestLinks, []testquality.Reason) {
	registry := adapterregistry.Default()
	var rows []testquality.Result
	var links []testquality.TestLinks
	var unavailable []testquality.Reason
	for _, ws := range workspaces {
		analyzer, ok := registry.ResolveSource(adapters.Identity{ID: ws.AdapterID, Version: ws.AdapterVersion}, adapters.Match{Language: ws.Language, Framework: ws.Framework})
		if !ok {
			continue
		}
		input := adapters.QualityInput{Root: ws.RootPath, Workspace: ws.ID, TestKind: ws.TestKind}
		var results []testquality.Result
		var reason testquality.Reason
		if combined, ok := analyzer.(adapters.SourceEvidenceAnalyzer); ok {
			var observed []testquality.TestLinks
			results, observed, reason = combined.AnalyzeSourceEvidence(ctx, input)
			links = append(links, observed...)
		} else {
			results, reason = analyzer.AnalyzeSource(ctx, input)
		}
		rows = append(rows, results...)
		if reason != testquality.ReasonNone {
			unavailable = append(unavailable, reason)
		}
	}
	return rows, links, unavailable
}
