package validation

import (
	"context"
	"net/http"
	"path/filepath"
	"sort"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	factsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts/facts_v1connect"
)

const codeFactsScenarioID = "code-facts"

// CodeFactsDetector resolves a scenario's API language + domains by calling the
// code-facts service, and degrades to the filesystem detector on any failure so
// detection is never silently empty. Code-facts is the primary source for
// domains (FACT_FAMILY_FILE_DOMAIN); the API-surface language is taken from the
// Go-rooted parse unit, with the filesystem manifest as the authoritative
// fallback.
type CodeFactsDetector struct {
	Resolver interface {
		ResolveScenarioURLDefault(context.Context, string) (string, error)
	}
	HTTPClient connect.HTTPClient
	// Fallback is the detector used when code-facts is unavailable. Defaults to
	// FilesystemDetector.
	Fallback Detector
}

var _ Detector = CodeFactsDetector{}

// Detect asks code-facts for surfaces + file-domains, then fills any gaps from
// the filesystem fallback. A code-facts failure is non-fatal: the whole result
// degrades to the filesystem detector.
func (d CodeFactsDetector) Detect(ctx context.Context, scenario, scenarioDir string) Detection {
	fallback := d.Fallback
	if fallback == nil {
		fallback = FilesystemDetector{}
	}
	fs := fallback.Detect(ctx, scenario, scenarioDir)

	resolver := d.Resolver
	if resolver == nil {
		resolver = discovery.NewResolver(discovery.ResolverConfig{})
	}
	httpClient := d.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	baseURL, err := resolver.ResolveScenarioURLDefault(ctx, codeFactsScenarioID)
	if err != nil {
		return fs
	}
	client := factsconnect.NewCodeFactsServiceClient(httpClient, baseURL)
	resp, err := client.DescribeCodeFacts(ctx, connect.NewRequest(&factsv1.DescribeCodeFactsRequest{
		Target: &factsv1.CodeTarget{
			Kind:     factsv1.TargetKind_TARGET_KIND_SCENARIO,
			Scenario: scenario,
			Path:     scenarioDir,
		},
		Include: []factsv1.FactFamily{
			factsv1.FactFamily_FACT_FAMILY_PARSE_UNITS,
			factsv1.FactFamily_FACT_FAMILY_FILE_DOMAIN,
		},
	}))
	if err != nil {
		return fs
	}

	out := Detection{
		Language: apiLanguageFromParseUnits(resp.Msg.GetParseUnits(), scenarioDir),
		Domains:  domainsFromFacts(resp.Msg.GetFacts()),
	}
	// Never regress below the filesystem signal: code-facts may legitimately
	// return nothing for an unparsed surface.
	if out.Language == "" {
		out.Language = fs.Language
	}
	if len(out.Domains) == 0 {
		out.Domains = fs.Domains
	}
	return out
}

// apiLanguageFromParseUnits picks the language of the parse unit rooted under
// the scenario's api/ directory — the API-surface language. Returns "" when no
// api-rooted parse unit is present.
func apiLanguageFromParseUnits(units []*factsv1.ParseUnit, scenarioDir string) string {
	apiDir := filepath.Clean(filepath.Join(scenarioDir, "api"))
	for _, u := range units {
		root := filepath.Clean(strings.TrimSpace(u.GetRootPath()))
		if root == apiDir || strings.HasPrefix(root, apiDir+string(filepath.Separator)) {
			if lang := strings.ToLower(strings.TrimSpace(u.GetLanguage())); lang != "" {
				return lang
			}
		}
	}
	return ""
}

// domainsFromFacts extracts the distinct domain names from FILE_DOMAIN generic
// facts. The domain is carried either as the fact subject or a "domain"
// attribute, depending on the producer; we read both, preferring the attribute.
func domainsFromFacts(facts []*factsv1.GenericFact) []string {
	seen := map[string]struct{}{}
	for _, f := range facts {
		if f.GetFamily() != factsv1.FactFamily_FACT_FAMILY_FILE_DOMAIN {
			continue
		}
		domain := strings.TrimSpace(f.GetAttributes()["domain"])
		if domain == "" {
			domain = strings.TrimSpace(f.GetKind())
		}
		if domain == "" {
			continue
		}
		seen[domain] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for d := range seen {
		out = append(out, d)
	}
	sort.Strings(out)
	return out
}
