package facts

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/audit"
	auditconnect "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/audit/audit_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/shared"
	signalsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/signals"
	signalsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/signals/signals_v1connect"
	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

const cartographerScenario = "architecture-cartographer"

type FileDomainProvider interface {
	DescribeFileDomains(ctx context.Context, target *factsv1.TargetContext) ([]*factsv1.GenericFact, []*factsv1.Evidence, []*factsv1.Warning, error)
}

type cartographerFileDomainProvider struct {
	resolver   URLResolver
	httpClient connect.HTTPClient
}

func NewCartographerFileDomainProvider(resolver URLResolver, httpClient connect.HTTPClient) FileDomainProvider {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &cartographerFileDomainProvider{resolver: resolver, httpClient: httpClient}
}

func (p *cartographerFileDomainProvider) DescribeFileDomains(ctx context.Context, target *factsv1.TargetContext) ([]*factsv1.GenericFact, []*factsv1.Evidence, []*factsv1.Warning, error) {
	if target == nil || !target.GetScenarioAware() || strings.TrimSpace(target.GetScenario()) == "" {
		return nil, nil, []*factsv1.Warning{providerWarning("architecture-cartographer", "scenario_required", "FILE_DOMAIN facts require a scenario-aware target.", factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED)}, nil
	}
	if p.resolver == nil {
		return nil, nil, nil, ProviderUnavailableError{Analyzer: cartographerScenario, Err: errors.New("missing URL resolver")}
	}
	baseURL, err := p.resolver.ResolveScenarioURLDefault(ctx, cartographerScenario)
	if err != nil {
		return nil, nil, nil, ProviderUnavailableError{Analyzer: cartographerScenario, Err: err}
	}

	auditResp, err := auditconnect.NewAuditServiceClient(p.httpClient, baseURL).Run(ctx, connect.NewRequest(&auditv1.AuditRunRequest{
		Scenario:          target.GetScenario(),
		AllowLowAuthority: true,
	}))
	if err != nil {
		return nil, nil, nil, classifyProviderError(cartographerScenario, err)
	}
	authorityConfidence := auditResp.Msg.GetDomains().GetConfidence()
	if authorityConfidence == "" {
		authorityConfidence = strings.ToLower(strings.TrimPrefix(auditResp.Msg.GetAuthorityConfidence().String(), "AUTHORITY_CONFIDENCE_"))
	}

	files, err := fileDomainSourceFiles(target.GetRootPath())
	if err != nil {
		return nil, nil, nil, err
	}
	client := signalsconnect.NewSignalsServiceClient(p.httpClient, baseURL)
	facts := make([]*factsv1.GenericFact, 0, len(files))
	var warnings []*factsv1.Warning
	for _, file := range files {
		resp, scoreErr := client.ScoreChunk(ctx, connect.NewRequest(&signalsv1.ScoreChunkRequest{
			Scenario: target.GetScenario(),
			RepoPath: file,
		}))
		if scoreErr != nil {
			warnings = append(warnings, providerWarning("architecture-cartographer.signals", "file_unscored", fmt.Sprintf("%s: %v", file, scoreErr), factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN))
			continue
		}
		facts = append(facts, fileDomainFact(file, authorityConfidence, resp.Msg.GetVerdict()))
	}
	evidence := []*factsv1.Evidence{{
		Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN,
		Confidence: 1,
		Analyzer:   "architecture-cartographer",
		Message:    fmt.Sprintf("Delegated FILE_DOMAIN ownership verdicts for %d source file(s).", len(facts)),
	}}
	sort.SliceStable(facts, func(i, j int) bool { return facts[i].GetSubject() < facts[j].GetSubject() })
	return facts, evidence, warnings, nil
}

func fileDomainFact(path string, authorityConfidence string, verdict *sharedv1.Verdict) *factsv1.GenericFact {
	attrs := map[string]string{
		"path":                 path,
		"authority_confidence": authorityConfidence,
		"analyzer":             "architecture-cartographer",
	}
	confidence := 0.0
	if verdict != nil {
		attrs["top_domain"] = verdict.GetTopDomain()
		attrs["top_value"] = strconv.FormatFloat(verdict.GetTopValue(), 'f', -1, 64)
		attrs["tier"] = tierString(verdict.GetTier())
		attrs["runner_up_domain"] = verdict.GetRunnerUpDomain()
		attrs["runner_up_value"] = strconv.FormatFloat(verdict.GetRunnerUpValue(), 'f', -1, 64)
		attrs["tied"] = strconv.FormatBool(verdict.GetTied())
		confidence = verdict.GetTopValue()
	}
	return &factsv1.GenericFact{
		Id:         "architecture-cartographer:file_domain:" + stablePathID(path),
		Family:     factsv1.FactFamily_FACT_FAMILY_FILE_DOMAIN,
		Kind:       "file_domain",
		Subject:    path,
		Attributes: attrs,
		Evidence: []*factsv1.Evidence{{
			Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN,
			Confidence: confidence,
			Analyzer:   "architecture-cartographer.signals",
			Message:    "Architecture Cartographer produced the file-domain verdict.",
			Range:      &factsv1.SourceRange{File: path},
		}},
	}
}

func fileDomainSourceFiles(root string) ([]string, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "" || root == "." {
		return nil, errors.New("target root path is required for FILE_DOMAIN")
	}
	var out []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := entry.Name()
		if entry.IsDir() {
			switch name {
			case ".git", "node_modules", "vendor", "dist", "build", ".next", ".turbo":
				return filepath.SkipDir
			default:
				return nil
			}
		}
		if !isFileDomainSourceFile(name) {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

func isFileDomainSourceFile(name string) bool {
	switch filepath.Ext(name) {
	case ".go", ".ts", ".tsx":
		return true
	default:
		return false
	}
}

func tierString(t sharedv1.Tier) string {
	switch t {
	case sharedv1.Tier_TIER_AUTO_PLACE:
		return "auto_place"
	case sharedv1.Tier_TIER_SUGGEST:
		return "suggest"
	case sharedv1.Tier_TIER_CONFLICT:
		return "conflict"
	default:
		return ""
	}
}

func stablePathID(path string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", " ", "_")
	return replacer.Replace(path)
}
