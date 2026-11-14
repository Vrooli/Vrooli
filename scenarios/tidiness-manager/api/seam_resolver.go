package main

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	repocontract "github.com/vrooli/repo-contract-go"
	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	factsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts/facts_v1connect"
)

const codeFactsSeamResolver = "code-facts"

// scanCodeFactsSeams resolves only seams explicitly assigned to the broker.
// Literal and directive seams remain in ScanSeams because those facts are not
// present in the shared graph contract.
func scanCodeFactsSeams(treeRoot string, seams []compiledSeam) ([]SeamHit, error) {
	var requested []compiledSeam
	for _, seam := range seams {
		if seam.seam.Resolver == codeFactsSeamResolver {
			requested = append(requested, seam)
		}
	}
	if len(requested) == 0 {
		return nil, nil
	}

	target, err := codeFactsTarget(treeRoot)
	if err != nil {
		return nil, err
	}
	baseURL, err := discovery.NewResolver(discovery.ResolverConfig{}).ResolveScenarioURLDefault(context.Background(), "code-facts")
	if err != nil {
		return nil, fmt.Errorf("resolve code-facts: %w", err)
	}
	client := factsconnect.NewCodeFactsServiceClient(http.DefaultClient, baseURL)
	hits := make([]SeamHit, 0)
	// Keep calls/references and symbols in separate reports. Besides making
	// the resolver's family contract explicit, this avoids turning one seam
	// scan into a large mixed-family cache write when another analyzer is
	// refreshing the same target.
	for _, familyRequest := range []struct {
		include []factsv1.FactFamily
		kind    string
	}{
		{include: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_CALLS, factsv1.FactFamily_FACT_FAMILY_REFERENCES}, kind: "call"},
		{include: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_SYMBOLS}, kind: "declaration"},
	} {
		candidates := make([]compiledSeam, 0, len(requested))
		for _, candidate := range requested {
			if candidate.seam.Bypass.Kind == familyRequest.kind {
				candidates = append(candidates, candidate)
			}
		}
		if len(candidates) == 0 {
			continue
		}
		request := &factsv1.DescribeCodeFactsRequest{Target: target, Include: familyRequest.include, UseCache: true, MaxDepth: 1, PageSize: 1000}
		for page := 0; page < 100; page++ {
			response, requestErr := describeCodeFactsWithRetry(client, request)
			if requestErr != nil {
				return nil, fmt.Errorf("describe code facts for %s seams: %w", familyRequest.kind, requestErr)
			}
			for _, fact := range response.Msg.GetFacts() {
				for _, candidate := range candidates {
					if !brokerFactKindMatches(candidate.seam.Bypass.Kind, fact.GetFamily()) || !brokerFactMatches(candidate, fact) {
						continue
					}
					for _, evidence := range fact.GetEvidence() {
						path := strings.TrimSpace(evidence.GetRange().GetFile())
						if path == "" || !seamPathIncluded(filepath.ToSlash(path), candidate.seam.Scope) {
							continue
						}
						line := int(evidence.GetRange().GetStartLine())
						if line == 0 {
							line = 1
						}
						symbol := firstNonEmptyFact(evidence.GetSymbol(), fact.GetSubject(), fact.GetAttributes()["callee"])
						hits = append(hits, SeamHit{SeamID: candidate.seam.ID, Canonical: candidate.seam.Canonical, Why: candidate.seam.Why, Remediation: candidate.seam.Remediation, Severity: candidate.seam.Severity, Budget: candidate.seam.Budget, Path: path, Symbol: symbol, Line: line, Analyzer: evidence.GetAnalyzer()})
					}
				}
			}
			request.PageToken = response.Msg.GetNextPageToken()
			if request.PageToken == "" {
				break
			}
		}
	}
	return hits, nil
}

func describeCodeFactsWithRetry(client factsconnect.CodeFactsServiceClient, request *factsv1.DescribeCodeFactsRequest) (*connect.Response[factsv1.CodeFactsReport], error) {
	var response *connect.Response[factsv1.CodeFactsReport]
	var err error
	for attempt := 0; attempt < 5; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		response, err = client.DescribeCodeFacts(ctx, connect.NewRequest(request))
		cancel()
		if err == nil || (!strings.Contains(err.Error(), "SQLITE_BUSY") && !strings.Contains(err.Error(), "database is locked")) {
			return response, err
		}
		time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond)
	}
	return nil, err
}

func codeFactsTarget(treeRoot string) (*factsv1.CodeTarget, error) {
	repoRoot, err := repocontract.FindRepoRootFromPath(treeRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve seam repository root: %w", err)
	}
	repoRoot = filepath.Clean(repoRoot)
	if rel, relErr := filepath.Rel(filepath.Join(repoRoot, "scenarios"), treeRoot); relErr == nil && rel != "." && !strings.HasPrefix(rel, "..") {
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) > 0 && parts[0] != "" {
			return &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: parts[0], RepoRoot: repoRoot}, nil
		}
	}
	return &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_CONTROL_PLANE, RepoRoot: repoRoot}, nil
}

func brokerFactKindMatches(kind string, family factsv1.FactFamily) bool {
	switch kind {
	case "call":
		return family == factsv1.FactFamily_FACT_FAMILY_CALLS || family == factsv1.FactFamily_FACT_FAMILY_REFERENCES
	case "declaration":
		return family == factsv1.FactFamily_FACT_FAMILY_SYMBOLS
	default:
		return false
	}
}

func brokerFactMatches(candidate compiledSeam, fact *factsv1.GenericFact) bool {
	attrs := fact.GetAttributes()
	values := []string{fact.GetSubject(), attrs["callee"], attrs["callee_symbol"], attrs["name"], attrs["referenced_symbol"], attrs["symbol"]}
	if subject := fact.GetSubject(); subject != "" {
		if packageID := strings.TrimPrefix(attrs["package_id"], "package:"); packageID != "" {
			if slash := strings.LastIndex(packageID, "/"); slash >= 0 {
				packageID = packageID[slash+1:]
			}
			values = append(values, packageID+"."+subject)
		}
	}
	for _, value := range values {
		if value != "" && candidate.pattern.MatchString(value) {
			if candidate.seam.Bypass.Kind != "declaration" || declarationFactKindMatches(candidate.seam.Bypass.DeclKind, fact) {
				return true
			}
		}
	}
	return false
}

func declarationFactKindMatches(want string, fact *factsv1.GenericFact) bool {
	if want == "" {
		return true
	}
	kind := strings.ToLower(firstNonEmptyFact(fact.GetAttributes()["kind"], fact.GetKind()))
	switch want {
	case "func":
		return strings.Contains(kind, "func") || strings.Contains(kind, "method") || strings.Contains(kind, "hook")
	case "method":
		return strings.Contains(kind, "method")
	case "type":
		return strings.Contains(kind, "type") || strings.Contains(kind, "class") || strings.Contains(kind, "interface")
	case "interface":
		return strings.Contains(kind, "interface")
	case "const":
		return strings.Contains(kind, "const")
	case "var":
		return strings.Contains(kind, "var")
	default:
		return false
	}
}

func firstNonEmptyFact(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
