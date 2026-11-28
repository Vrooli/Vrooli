// Package codefacts projects the shared fact substrate into architectural
// snapshots. Source discovery, parser admission and freshness belong to Code Facts.
package codefacts

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"architecture-cartographer/internal/graph"
	"connectrpc.com/connect"
	"github.com/vrooli/api-core/factgraph"
	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	"github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts/facts_v1connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

const ScenarioName = "code-facts"

type Client struct {
	urls       graph.ScenarioURLResolver
	repoRoot   string
	language   graph.Language
	httpClient connect.HTTPClient
}

func New(urls graph.ScenarioURLResolver, repoRoot string, language graph.Language, httpClient connect.HTTPClient) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{urls: urls, repoRoot: repoRoot, language: language, httpClient: httpClient}
}

func (c *Client) Name() string {
	if c.language == graph.LanguageTypeScript {
		return "typescript"
	}
	return string(c.language)
}

func (c *Client) SupportedLanguages() []graph.Language { return []graph.Language{c.language} }

func (c *Client) Extract(ctx context.Context, scenario string) (graph.RawGraph, error) {
	if c.urls == nil {
		return graph.RawGraph{}, graph.IntegrationError{Kind: "internal", Scenario: ScenarioName, Cause: fmt.Errorf("missing Code Facts URL resolver")}
	}
	baseURL, err := c.urls.ResolveScenarioURLDefault(ctx, ScenarioName)
	if err != nil {
		return graph.RawGraph{}, graph.ClassifyResolveError(err, ScenarioName)
	}
	target := &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: scenario, RepoRoot: c.repoRoot, LanguageFilter: []string{c.Name()}}
	if scenario == "control-plane" {
		target.Kind = factsv1.TargetKind_TARGET_KIND_CONTROL_PLANE
	}
	response, err := facts_v1connect.NewCodeFactsServiceClient(c.httpClient, baseURL).DescribeCodeFacts(ctx, connect.NewRequest(&factsv1.DescribeCodeFactsRequest{
		Target: target, Include: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_PARSE_UNITS}, IncludeGraphs: true, UseCache: true,
	}))
	if err != nil {
		return graph.RawGraph{}, graph.ClassifyConnectError(err, ScenarioName)
	}
	return project(response.Msg, c.language)
}

func project(report *factsv1.CodeFactsReport, language graph.Language) (graph.RawGraph, error) {
	out := graph.RawGraph{}
	if report.GetCache().GetSourceHash() != "" {
		out.SourceFingerprint = report.GetCache().GetSourceHash() + ":" + report.GetCache().GetConfigHash()
	}
	for _, warning := range report.GetWarnings() {
		if warning.GetStatus() == factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN || warning.GetStatus() == factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED {
			out.SkippedAdapters = append(out.SkippedAdapters, warning.GetCode())
		}
	}
	if (report.GetTarget() == nil || report.GetTarget().GetRootPath() == "") && len(report.GetGraphs()) > 0 {
		return graph.RawGraph{}, fmt.Errorf("Code Facts graph requires target root")
	}
	index := factgraph.NewIndex(report)
	for _, warning := range index.Warnings() {
		if strings.Contains(warning, "outside target") || (warning == "missing target root" && len(report.GetGraphs()) > 0) {
			return graph.RawGraph{}, fmt.Errorf("invalid Code Facts graph projection: %s", warning)
		}
	}
	for _, parsed := range index.Units() {
		rel := parsed.Root()
		// File IDs are local to a parse unit. Qualify all identity references so
		// api/main.go and cli/main.go cannot collapse into one file or symbol.
		prefix := parsed.Language() + ":" + rel + ":"
		qualify := func(id string) string {
			if id == "" {
				return ""
			}
			return prefix + id
		}
		unitLanguage := language
		if parsed.Language() == "typescript" {
			unitLanguage = graph.LanguageTypeScript
		} else if parsed.Language() != "" {
			unitLanguage = graph.Language(parsed.Language())
		}
		out.Languages = append(out.Languages, unitLanguage)
		out.ExtractionMS += parsed.ExtractionMS()
		if profile := parsed.Profile(); profile != "" {
			out.ExtractionProfiles = append(out.ExtractionProfiles, profile)
		}
		for _, omission := range parsed.Omissions() {
			out.OmittedInformation = append(out.OmittedInformation, graph.InformationOmission{Capability: omission.Capability, Reason: omission.Reason})
		}
		for _, node := range parsed.Nodes() {
			attrs, kind := map[string]string{}, node.Attribute("kind")
			for _, key := range []string{"package_id", "file_id", "import_path", "lines", "is_test", "exported"} {
				if value := node.Attribute(key); value != "" {
					attrs[key] = value
				}
			}
			switch {
			case node.Kind() == commonv1.NodeKind_NODE_KIND_FILE.String():
				file := graph.FileNode{ID: node.ID(), Path: node.Path(), PackageID: attrs["package_id"], Language: unitLanguage, Lines: graph.ParseNonNegativeIntAttr(attrs["lines"]), IsTest: attrs["is_test"] == "true"}
				file.ID, file.PackageID = qualify(file.ID), qualify(file.PackageID)
				file.Path = filepath.ToSlash(filepath.Join(rel, file.Path))
				out.Files = append(out.Files, file)
			case node.Kind() == commonv1.NodeKind_NODE_KIND_MODULE.String() || (node.Kind() == commonv1.NodeKind_NODE_KIND_PACKAGE.String() && (kind == "" || kind == "package")):
				name := node.Name()
				if unitLanguage == graph.LanguageGo && node.Kind() == commonv1.NodeKind_NODE_KIND_PACKAGE.String() {
					name = attrs["import_path"]
					if name == "" {
						name = node.Path()
					}
				}
				out.Packages = append(out.Packages, graph.PackageNode{ID: qualify(node.ID()), ImportPath: name, Language: unitLanguage})
			default:
				if kind == "" {
					kind = node.Kind()
				}
				out.Symbols = append(out.Symbols, graph.SymbolNode{ID: qualify(node.ID()), Name: node.Name(), PackageID: qualify(attrs["package_id"]), FileID: qualify(attrs["file_id"]), Kind: kind, Exported: attrs["exported"] == "true"})
			}
		}
		for _, edge := range parsed.Edges() {
			if edge.Kind() != commonv1.EdgeKind_EDGE_KIND_IMPORT.String() {
				continue
			}
			item := graph.ImportEdge{From: edge.From(), ToPackageID: edge.To(), SymbolIDs: graph.SplitCSV(edge.Attribute("symbol_ids")), SymbolKinds: graph.SplitCSV(edge.Attribute("symbol_kinds")), TestOnly: edge.Attribute("test_only") == "true"}
			item.From, item.ToPackageID = qualify(item.From), qualify(item.ToPackageID)
			for i, id := range item.SymbolIDs {
				item.SymbolIDs[i] = qualify(id)
			}
			out.Imports = append(out.Imports, item)
		}
	}
	graph.AssignPackageRepoPaths(out.Packages, out.Files)
	return out, nil
}
