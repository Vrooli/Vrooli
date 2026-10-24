package domains

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	domainsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/domains"
	domainsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/domains/domains_v1connect"

	"architecture-cartographer/cli/internal/attestrender"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client domainsconnect.DomainsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: domainsconnect.NewDomainsServiceClient(httpClient, baseURL),
	}
}

// extract derives the domain map fresh from the scenario's on-disk sources.
func (h *handlers) extract(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.ExtractDomains(context.Background(), connect.NewRequest(&domainsv1.ExtractDomainsRequest{Scenario: scenario}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("extract domains for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetDomainMap() == nil {
		return fmt.Errorf("server returned no domain map")
	}
	return renderMap(ctx, resp.Msg.GetDomainMap())
}

// show returns the derived domain map for a scenario.
func (h *handlers) show(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.GetDomainMap(context.Background(), connect.NewRequest(&domainsv1.GetDomainMapRequest{Scenario: scenario}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get domain map for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetDomainMap() == nil {
		return fmt.Errorf("server returned no domain map")
	}
	return renderMap(ctx, resp.Msg.GetDomainMap())
}

// convergence reports where the scenario's surfaces disagree on the domain set.
func (h *handlers) convergence(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.ConvergenceReport(context.Background(), connect.NewRequest(&domainsv1.ConvergenceReportRequest{Scenario: scenario}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("convergence report for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no convergence report")
	}
	authorityLine := fmt.Sprintf("Authority source: %s", sourceName(resp.Msg.GetAuthority()))
	confidenceLine := fmt.Sprintf("Authority confidence: %s", confidenceName(resp.Msg.GetAuthorityConfidence()))
	findings := resp.Msg.GetFindings()
	if len(findings) == 0 {
		return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
			Summary:        []string{fmt.Sprintf("Surfaces converge for %q — no disagreements.", scenario), authorityLine, confidenceLine},
			ResultsHeading: "Findings",
			Results:        nil,
		})
	}
	results := make([]string, 0, len(findings))
	for _, f := range findings {
		label := f.GetDomain()
		if rolled := f.GetRolledUpDomains(); len(rolled) > 0 {
			label = fmt.Sprintf("domains=%s", strings.Join(rolled, ","))
		} else if label == "" {
			label = "—"
		}
		results = append(results, fmt.Sprintf("[%s] %s — %s (%s)",
			convergenceSeverityName(f.GetSeverity()), label, f.GetMessage(), f.GetKind()))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d convergence finding(s) for %q.", len(findings), scenario), authorityLine, confidenceLine},
		ResultsHeading: "Findings",
		Results:        results,
		RetrievalHints: []string{"Findings are advisory; warn = real disagreement, info = coverage signal. Reconcile DOMAINS.md, api folders, and cli groups."},
	})
}

// draft proposes a DOMAINS.md inventory for human ratification.
func (h *handlers) draft(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.DraftDomains(context.Background(), connect.NewRequest(&domainsv1.DraftDomainsRequest{Scenario: scenario}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("draft domains for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no domains draft")
	}
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
	}
	fmt.Fprint(ctx.Stdout(), resp.Msg.GetMarkdown())
	if !strings.HasSuffix(resp.Msg.GetMarkdown(), "\n") {
		fmt.Fprintln(ctx.Stdout())
	}
	return nil
}

func confidenceName(c domainsv1.AuthorityConfidence) string {
	switch c {
	case domainsv1.AuthorityConfidence_AUTHORITY_CONFIDENCE_HIGH:
		return "high"
	case domainsv1.AuthorityConfidence_AUTHORITY_CONFIDENCE_LOW:
		return "low (fallback — no curated DOMAINS.md or api manifest)"
	default:
		return "unspecified"
	}
}

func convergenceSeverityName(s domainsv1.ConvergenceSeverity) string {
	switch s {
	case domainsv1.ConvergenceSeverity_CONVERGENCE_SEVERITY_WARN:
		return "warn"
	case domainsv1.ConvergenceSeverity_CONVERGENCE_SEVERITY_INFO:
		return "info"
	default:
		return "unspecified"
	}
}

func renderMap(ctx cliapp.RunContext, m *domainsv1.DerivedDomainMap) error {
	results := make([]string, 0, len(m.GetDomains()))
	for _, d := range m.GetDomains() {
		line := fmt.Sprintf("%s — paths: %s", d.GetName(), strings.Join(d.GetPaths(), ", "))
		if arche := primaryArchetypeName(d.GetArchetypes()); arche != "" {
			line += fmt.Sprintf(" — archetype: %s", arche)
		}
		line += fmt.Sprintf(" — declared by: %s", strings.Join(sourceNames(d.GetProvenance()), ", "))
		if basis := attestrender.Basis(d.GetAttestation().GetBasis()); basis != "" {
			line += fmt.Sprintf(" — basis: %s", basis)
		}
		results = append(results, line)
	}
	summary := []string{
		fmt.Sprintf("Derived %d domain(s) for %q.", len(m.GetDomains()), m.GetScenario()),
		fmt.Sprintf("Authority source: %s", sourceName(m.GetAuthority())),
		fmt.Sprintf("Authority confidence: %s", confidenceName(m.GetAuthorityConfidence())),
	}
	if att := m.GetAttestation(); att != nil {
		summary = append(summary, fmt.Sprintf("Map basis: %s (%s)", attestrender.Basis(att.GetBasis()), attestrender.Sufficiency(att.GetSufficiency())))
	}
	if shared := m.GetSharedSubstrate(); len(shared) > 0 {
		summary = append(summary, fmt.Sprintf("Shared substrate: %s", strings.Join(shared, ", ")))
	}
	return cliapp.RenderProtoList(ctx, m, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Domains",
		Results:        results,
		RetrievalHints: []string{"Provenance shows which ladder rungs agreed on each domain; the authority rung defines the expected set."},
	})
}

func primaryArchetypeName(archetypes []*domainsv1.DomainArchetype) string {
	for _, archetype := range archetypes {
		if archetype.GetSource() == domainsv1.ArchetypeSource_ARCHETYPE_SOURCE_DECLARED {
			if label := archetypeLabel(archetype); label != "" {
				return label
			}
		}
	}
	if len(archetypes) > 0 {
		return archetypeLabel(archetypes[0])
	}
	return ""
}

// archetypeLabel renders a DomainArchetype as its canonical lowercase name,
// falling back to the preserved declared label when the declared value did not
// map onto the canonical vocabulary.
func archetypeLabel(a *domainsv1.DomainArchetype) string {
	switch a.GetArchetype() {
	case domainsv1.Archetype_ARCHETYPE_REPORTING:
		return "reporting"
	case domainsv1.Archetype_ARCHETYPE_SERVICE:
		return "service"
	case domainsv1.Archetype_ARCHETYPE_MUTATION:
		return "mutation"
	case domainsv1.Archetype_ARCHETYPE_CLASSIFICATION:
		return "classification"
	case domainsv1.Archetype_ARCHETYPE_ORCHESTRATION:
		return "orchestration"
	case domainsv1.Archetype_ARCHETYPE_SCORING:
		return "scoring"
	case domainsv1.Archetype_ARCHETYPE_QUERY:
		return "query"
	default:
		return a.GetDeclaredLabel()
	}
}

func sourceNames(sources []domainsv1.DomainSource) []string {
	out := make([]string, 0, len(sources))
	for _, s := range sources {
		out = append(out, sourceName(s))
	}
	return out
}

func sourceName(s domainsv1.DomainSource) string {
	switch s {
	case domainsv1.DomainSource_DOMAIN_SOURCE_API_MANIFEST:
		return "api-manifest"
	case domainsv1.DomainSource_DOMAIN_SOURCE_DOMAINS_DOC:
		return "domains-doc"
	case domainsv1.DomainSource_DOMAIN_SOURCE_API_FOLDERS:
		return "api-folders"
	case domainsv1.DomainSource_DOMAIN_SOURCE_CLI_GROUPS:
		return "cli-groups"
	case domainsv1.DomainSource_DOMAIN_SOURCE_UI_FEATURES:
		return "ui-features"
	default:
		return "unspecified"
	}
}
