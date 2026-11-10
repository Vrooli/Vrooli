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
	for _, parsed := range report.GetGraphs() {
		if parsed.GetGraph() == nil {
			continue
		}
		unit := parsed.GetParseUnit()
		if !filepath.IsAbs(report.GetTarget().GetRootPath()) || !filepath.IsAbs(unit.GetRootPath()) {
			return graph.RawGraph{}, fmt.Errorf("Code Facts graph requires absolute target and parse-unit roots")
		}
		rel, err := filepath.Rel(report.GetTarget().GetRootPath(), unit.GetRootPath())
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
			return graph.RawGraph{}, fmt.Errorf("Code Facts parse unit %q is outside target root", unit.GetRootPath())
		}
		rel = filepath.ToSlash(rel)
		// File IDs are local to a parse unit. Qualify all identity references so
		// api/main.go and cli/main.go cannot collapse into one file or symbol.
		prefix := unit.GetLanguage() + ":" + rel + ":"
		qualify := func(id string) string {
			if id == "" {
				return ""
			}
			return prefix + id
		}
		out.Languages = append(out.Languages, language)
		out.ExtractionMS += parsed.GetExtractionMs()
		if profile := parsed.GetExtractionProfile(); profile != "" {
			out.ExtractionProfiles = append(out.ExtractionProfiles, profile)
		}
		for _, omission := range parsed.GetOmittedInformation() {
			out.OmittedInformation = append(out.OmittedInformation, graph.InformationOmission{Capability: omission.GetCapability(), Reason: omission.GetReason()})
		}
		for _, node := range parsed.GetGraph().GetNodes() {
			if node == nil {
				continue
			}
			attrs, kind := node.GetAttributes(), node.GetAttributes()["kind"]
			switch {
			case node.GetKind() == commonv1.NodeKind_NODE_KIND_FILE:
				file := graph.FileNodeFromProto(node, language)
				file.ID, file.PackageID = qualify(file.ID), qualify(file.PackageID)
				file.Path = filepath.ToSlash(filepath.Join(rel, file.Path))
				out.Files = append(out.Files, file)
			case node.GetKind() == commonv1.NodeKind_NODE_KIND_MODULE || (node.GetKind() == commonv1.NodeKind_NODE_KIND_PACKAGE && (kind == "" || kind == "package")):
				name := node.GetName()
				if language == graph.LanguageGo && node.GetKind() == commonv1.NodeKind_NODE_KIND_PACKAGE {
					name = attrs["import_path"]
					if name == "" {
						name = node.GetPath()
					}
				}
				out.Packages = append(out.Packages, graph.PackageNode{ID: qualify(node.GetId()), ImportPath: name, Language: language})
			default:
				if kind == "" {
					kind = node.GetKind().String()
				}
				out.Symbols = append(out.Symbols, graph.SymbolNode{ID: qualify(node.GetId()), Name: node.GetName(), PackageID: qualify(attrs["package_id"]), FileID: qualify(attrs["file_id"]), Kind: kind, Exported: attrs["exported"] == "true"})
			}
		}
		for _, edge := range parsed.GetGraph().GetEdges() {
			if edge == nil || (language == graph.LanguageGo && edge.GetKind() != commonv1.EdgeKind_EDGE_KIND_IMPORT) {
				continue
			}
			item := graph.ImportEdgeFromProto(edge)
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
