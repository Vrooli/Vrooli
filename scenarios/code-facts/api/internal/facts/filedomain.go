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
	"sync"

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
	if target == nil {
		return nil, nil, []*factsv1.Warning{providerWarning("architecture-cartographer", "scenario_required", "FILE_DOMAIN facts require a scenario-aware target.", factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED)}, nil
	}
	if !target.GetScenarioAware() && (target.GetResolvedKind() == factsv1.TargetKind_TARGET_KIND_PROJECT || target.GetResolvedKind() == factsv1.TargetKind_TARGET_KIND_REPO) {
		return p.describeProjectFileDomains(ctx, target)
	}
	if !target.GetScenarioAware() || strings.TrimSpace(target.GetScenario()) == "" {
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
	facts, warnings := p.scoreFiles(ctx, client, target.GetScenario(), files, authorityConfidence)
	evidence := []*factsv1.Evidence{{
		Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN,
		Confidence: 1,
		Analyzer:   "architecture-cartographer",
		Message:    fmt.Sprintf("Delegated FILE_DOMAIN ownership verdicts for %d source file(s).", len(facts)),
	}}
	sort.SliceStable(facts, func(i, j int) bool { return facts[i].GetSubject() < facts[j].GetSubject() })
	return facts, evidence, warnings, nil
}

func (p *cartographerFileDomainProvider) describeProjectFileDomains(ctx context.Context, target *factsv1.TargetContext) ([]*factsv1.GenericFact, []*factsv1.Evidence, []*factsv1.Warning, error) {
	root := target.GetRootPath()
	if len(target.GetRootPaths()) > 0 {
		for _, candidate := range target.GetRootPaths() {
			if filepath.Base(candidate) == "scenarios" {
				root = filepath.Dir(candidate)
				break
			}
		}
	}
	scenariosRoot := filepath.Join(root, "scenarios")
	entries, err := os.ReadDir(scenariosRoot)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("list project scenarios for FILE_DOMAIN: %w", err)
	}
	type result struct {
		facts    []*factsv1.GenericFact
		evidence []*factsv1.Evidence
		warnings []*factsv1.Warning
	}
	results := make([]result, len(entries))
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for i, entry := range entries {
		if !entry.IsDir() || !hasServiceManifest(filepath.Join(scenariosRoot, entry.Name())) {
			continue
		}
		i, name := i, entry.Name()
		select {
		case <-ctx.Done():
			return nil, nil, nil, ctx.Err()
		case sem <- struct{}{}:
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			child := &factsv1.TargetContext{RootPath: filepath.Join(scenariosRoot, name), Scenario: name, ScenarioAware: true, ResolvedKind: factsv1.TargetKind_TARGET_KIND_SCENARIO}
			facts, evidence, warnings, err := p.DescribeFileDomains(ctx, child)
			if err != nil {
				results[i].warnings = []*factsv1.Warning{providerWarning("architecture-cartographer", "scenario_failed", fmt.Sprintf("%s: %v", name, err), factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN)}
				return
			}
			for _, fact := range facts {
				fact.Id = name + ":" + fact.GetId()
				if fact.Attributes == nil {
					fact.Attributes = map[string]string{}
				}
				fact.Attributes["scenario"] = name
			}
			results[i] = result{facts: facts, evidence: evidence, warnings: warnings}
		}()
	}
	wg.Wait()
	var facts []*factsv1.GenericFact
	var evidence []*factsv1.Evidence
	var warnings []*factsv1.Warning
	for _, item := range results {
		facts = append(facts, item.facts...)
		evidence = append(evidence, item.evidence...)
		warnings = append(warnings, item.warnings...)
	}
	sort.SliceStable(facts, func(i, j int) bool { return facts[i].GetId() < facts[j].GetId() })
	return facts, evidence, warnings, nil
}

func (p *cartographerFileDomainProvider) scoreFiles(ctx context.Context, client signalsconnect.SignalsServiceClient, scenario string, files []string, authorityConfidence string) ([]*factsv1.GenericFact, []*factsv1.Warning) {
	facts := make([]*factsv1.GenericFact, len(files))
	warnings := make([]*factsv1.Warning, 0)
	sem := make(chan struct{}, 16)
	var wg sync.WaitGroup
	var warningMu sync.Mutex
	addWarning := func(warning *factsv1.Warning) {
		warningMu.Lock()
		defer warningMu.Unlock()
		warnings = append(warnings, warning)
	}
	for i, file := range files {
		i, file := i, file
		select {
		case <-ctx.Done():
			addWarning(providerWarning("architecture-cartographer.signals", "context_cancelled", ctx.Err().Error(), factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN))
			continue
		case sem <- struct{}{}:
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			resp, err := client.ScoreChunk(ctx, connect.NewRequest(&signalsv1.ScoreChunkRequest{Scenario: scenario, RepoPath: file}))
			if err != nil {
				addWarning(providerWarning("architecture-cartographer.signals", "file_unscored", fmt.Sprintf("%s: %v", file, err), factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN))
				return
			}
			facts[i] = fileDomainFact(file, authorityConfidence, resp.Msg.GetVerdict())
		}()
	}
	wg.Wait()
	compact := facts[:0]
	for _, fact := range facts {
		if fact != nil {
			compact = append(compact, fact)
		}
	}
	sort.SliceStable(compact, func(i, j int) bool { return compact[i].GetSubject() < compact[j].GetSubject() })
	sort.SliceStable(warnings, func(i, j int) bool {
		if warnings[i].GetCode() != warnings[j].GetCode() {
			return warnings[i].GetCode() < warnings[j].GetCode()
		}
		return warnings[i].GetMessage() < warnings[j].GetMessage()
	})
	return compact, warnings
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
