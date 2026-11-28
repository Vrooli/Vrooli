package facts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

// Composition exposes typed composition-root evidence through Code Facts.
func (s *Service) Composition(_ context.Context, req *factsv1.DescribeCompositionRequest) (*factsv1.CompositionReport, error) {
	if req == nil || req.GetTarget() == nil {
		return nil, fmt.Errorf("target is required")
	}
	if err := validateTarget(req.GetTarget()); err != nil {
		return nil, err
	}
	target, err := resolveTarget(req.GetTarget())
	if err != nil {
		return nil, err
	}
	var evidence *CompositionReport
	if req.GetTarget().GetKind() == factsv1.TargetKind_TARGET_KIND_CONTROL_PLANE {
		evidence, err = DescribeControlPlaneCLIComposition(target.GetRootPath())
	} else {
		evidence, err = DescribeComposition(target.GetRootPath())
	}
	if err != nil {
		return nil, err
	}
	report := &factsv1.CompositionReport{Target: target.GetRequested(), AnalysisScope: evidence.AnalysisScope}
	for _, binding := range evidence.Bindings {
		report.Bindings = append(report.Bindings, &factsv1.CompositionBinding{Environment: binding.Environment, Consumer: binding.Consumer, Dependency: binding.Dependency, InterfaceType: binding.Interface, Provider: binding.Provider, ProviderExpression: binding.ProviderExpression, Status: binding.Status, SourcePath: binding.SourcePath, Line: int32(binding.Line), SourceHash: binding.SourceHash, Provenance: binding.Provenance})
	}
	return report, nil
}

type rankedCompositionBinding struct {
	hit   *factsv1.SearchHit
	score float64
}

var compositionTokenPattern = regexp.MustCompile(`[a-zA-Z0-9]+`)

// SearchComposition is a query-ranked owner adapter for one scenario. It
// emits the ordinary Code Facts SearchResponse shape for Search Hub, while
// keeping evidence generation and provenance owned by Code Facts.
func (s *Service) SearchComposition(ctx context.Context, req *factsv1.SearchCompositionRequest) (*factsv1.SearchResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	query := strings.TrimSpace(req.GetQuery())
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	scenario := strings.TrimSpace(req.GetScenario())
	controlPlane := scenario == "control-plane"
	if scenario == "" || (!controlPlane && strings.ContainsAny(scenario, `/\\:`)) || scenario == "." || scenario == ".." {
		return nil, fmt.Errorf("scenario must be one scenario slug")
	}
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	target := &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: scenario}
	if controlPlane {
		target = &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_CONTROL_PLANE}
	}
	report, err := s.Composition(ctx, &factsv1.DescribeCompositionRequest{Target: target})
	if err != nil {
		return nil, err
	}
	terms := compositionTokens(query)
	if len(terms) == 0 {
		return nil, fmt.Errorf("query must contain searchable terms")
	}
	ranked := make([]rankedCompositionBinding, 0, len(report.GetBindings()))
	for _, binding := range report.GetBindings() {
		hit, fields := compositionHit(binding)
		if controlPlane {
			hit.Scope = "global"
		} else {
			hit.Scope = "scenario:" + scenario
		}
		score := compositionScore(terms, fields)
		if score <= 0 {
			continue
		}
		hit.Score = score
		ranked = append(ranked, rankedCompositionBinding{hit: hit, score: score})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].hit.GetId() < ranked[j].hit.GetId()
	})
	response := &factsv1.SearchResponse{RetrievalRegime: "owner_composition_lexical_v1"}
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	for _, item := range ranked {
		response.Results = append(response.Results, item.hit)
	}
	return response, nil
}

func compositionHit(binding *factsv1.CompositionBinding) (*factsv1.SearchHit, map[string]string) {
	text := fmt.Sprintf("%s composition binding %s.%s interface %s provider %s expression %s status %s source %s:%d source_hash %s provenance %s", binding.GetEnvironment(), binding.GetConsumer(), binding.GetDependency(), binding.GetInterfaceType(), binding.GetProvider(), binding.GetProviderExpression(), binding.GetStatus(), binding.GetSourcePath(), binding.GetLine(), binding.GetSourceHash(), binding.GetProvenance())
	identity := strings.Join([]string{binding.GetEnvironment(), binding.GetConsumer(), binding.GetDependency(), binding.GetSourcePath(), fmt.Sprint(binding.GetLine()), binding.GetProviderExpression()}, "\x00")
	sum := sha256.Sum256([]byte(identity))
	status := factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN
	if binding.GetStatus() == "concrete" {
		status = factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN
	}
	id, analyzer, factKind := "code-facts:composition:"+hex.EncodeToString(sum[:12]), "code-facts.composition", "composition_binding"
	if binding.GetEnvironment() == "control-plane" {
		id, analyzer, factKind = "code-facts:cli-composition:"+binding.GetDependency(), "code-facts.cli-composition", "cli_command_composition"
	}
	hit := &factsv1.SearchHit{Id: id, Title: binding.GetConsumer() + "." + binding.GetDependency() + " — " + binding.GetEnvironment() + " " + binding.GetStatus(), Text: text, Path: binding.GetSourcePath(), Analyzer: analyzer, EvidenceStatus: status, FactKind: factKind, SourceHash: binding.GetSourceHash(), StartLine: binding.GetLine(), EndLine: binding.GetLine()}
	fields := map[string]string{"identity": binding.GetConsumer() + " " + binding.GetDependency(), "provider": binding.GetProvider() + " " + binding.GetProviderExpression(), "interface": binding.GetInterfaceType(), "state": binding.GetEnvironment() + " " + binding.GetStatus(), "path": binding.GetSourcePath(), "text": text}
	return hit, fields
}

var compositionStopWords = map[string]bool{"a": true, "an": true, "and": true, "are": true, "for": true, "how": true, "in": true, "is": true, "of": true, "the": true, "to": true, "what": true, "which": true, "with": true, "does": true, "show": true, "find": true}

func compositionTokens(value string) []string {
	seen := map[string]bool{}
	var out []string
	for _, token := range compositionTokenPattern.FindAllString(strings.ToLower(value), -1) {
		if len(token) < 2 || compositionStopWords[token] || seen[token] {
			continue
		}
		seen[token] = true
		out = append(out, token)
	}
	return out
}
func compositionScore(terms []string, fields map[string]string) float64 {
	weights := map[string]float64{"identity": 3, "provider": 3, "interface": 2.5, "state": 1.5, "path": 0.8, "text": 0.4}
	fieldTerms := map[string]map[string]bool{}
	for field, value := range fields {
		fieldTerms[field] = map[string]bool{}
		for _, token := range compositionTokens(value) {
			fieldTerms[field][token] = true
		}
	}
	// Explicit state/environment terms are filters as well as rank features;
	// a request for concrete production wiring must not return unknown test rows.
	for _, term := range terms {
		if term == "concrete" || term == "unknown" || term == "ambiguous" || term == "production" || term == "test" {
			if !fieldTerms["state"][term] {
				return 0
			}
		}
	}
	score := 0.0
	for _, term := range terms {
		best := 0.0
		for field, tokens := range fieldTerms {
			if tokens[term] && weights[field] > best {
				best = weights[field]
			}
		}
		if best > 0 {
			score += best / 3
		}
	}
	return score / float64(len(terms))
}
