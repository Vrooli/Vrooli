package manifest

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	manifestv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/manifest"
	manifestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/manifest/manifest_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client manifestconnect.ManifestServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: manifestconnect.NewManifestServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListManifests(context.Background(), connect.NewRequest(&manifestv1.ListManifestsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list manifests", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no manifests response")
	}
	results := make([]string, 0, len(resp.Msg.Manifests))
	for _, m := range resp.Msg.Manifests {
		results = append(results, formatManifest(m))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d manifest(s).", len(resp.Msg.Manifests))},
		ResultsHeading: "Manifests",
		Results:        results,
		RetrievalHints: []string{
			"`manifest get <skill_id> <golden_slug>` — show a single manifest",
			"`manifest upsert --skill <id> --golden <slug> --wildcard-allowed` — create or replace a manifest",
		},
	})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	skillID := ctx.Positional("skill_id")
	goldenSlug := ctx.Positional("golden_slug")
	resp, err := h.client.GetManifest(context.Background(), connect.NewRequest(&manifestv1.GetManifestRequest{
		SkillId:    skillID,
		GoldenSlug: goldenSlug,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get manifest %q/%q", skillID, goldenSlug), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Manifest == nil {
		return fmt.Errorf("server returned no manifest")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched manifest %s/%s.", resp.Msg.Manifest.SkillId, resp.Msg.Manifest.GoldenSlug)},
		ResultsHeading: "Manifest",
		Results:        []string{formatManifest(resp.Msg.Manifest)},
	})
}

func (h *handlers) upsert(ctx cliapp.RunContext) error {
	skill := ctx.Flag("skill_id")
	golden := ctx.Flag("golden_slug")
	allowedPaths := splitNonEmpty(ctx.Flag("allowed_paths"))
	convergence := parseConvergence(ctx.Flag("convergence_target"))
	wildcard := ctx.BoolFlag("wildcard-allowed")

	m := &manifestv1.Manifest{
		SkillId:               skill,
		GoldenSlug:            golden,
		AllowedPaths:          allowedPaths,
		WildcardAllowed:       wildcard,
		ConvergenceTarget:     convergence,
		TemplateVersionPinned: ctx.Flag("template_version_pinned"),
		SkillVersionPinned:    ctx.Flag("skill_version_pinned"),
	}
	resp, err := h.client.UpsertManifest(context.Background(), connect.NewRequest(&manifestv1.UpsertManifestRequest{Manifest: m}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("upsert manifest %q/%q", skill, golden), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Manifest == nil {
		return fmt.Errorf("server returned no manifest")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Upserted manifest %s/%s.", resp.Msg.Manifest.SkillId, resp.Msg.Manifest.GoldenSlug)},
		Changes: []string{formatManifest(resp.Msg.Manifest)},
		NextCommand: []string{
			fmt.Sprintf("`manifest get %s %s` — show this manifest", resp.Msg.Manifest.SkillId, resp.Msg.Manifest.GoldenSlug),
		},
	})
}

func (h *handlers) clearStale(ctx cliapp.RunContext) error {
	skill := ctx.Flag("skill")
	golden := ctx.Flag("golden")
	resp, err := h.client.ClearStale(context.Background(), connect.NewRequest(&manifestv1.ClearStaleRequest{
		SkillId:    skill,
		GoldenSlug: golden,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("clear-stale %q/%q", skill, golden), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no clear-stale response")
	}
	cleared := ""
	if resp.Msg.ClearedAt != nil {
		cleared = resp.Msg.ClearedAt.AsTime().Format(time.RFC3339)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Cleared staleness for %s/%s at %s.", skill, golden, cleared)},
	})
}

func splitNonEmpty(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func parseConvergence(s string) manifestv1.ConvergenceTarget {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "none", "unspecified":
		return manifestv1.ConvergenceTarget_CONVERGENCE_TARGET_NONE
	case "empty-diff", "empty_diff", "emptydiff":
		return manifestv1.ConvergenceTarget_CONVERGENCE_TARGET_EMPTY_DIFF
	default:
		return manifestv1.ConvergenceTarget_CONVERGENCE_TARGET_UNSPECIFIED
	}
}

func formatManifest(m *manifestv1.Manifest) string {
	if m == nil {
		return "(nil)"
	}
	wildcard := "no"
	if m.WildcardAllowed {
		wildcard = "yes"
	}
	updated := ""
	if m.UpdatedAt != nil {
		updated = m.UpdatedAt.AsTime().Format(time.RFC3339)
	}
	return fmt.Sprintf("%s/%s — wildcard=%s allowed=%d rules=%d converge=%s pinned(template=%s,skill=%s) updated=%s",
		m.SkillId, m.GoldenSlug, wildcard,
		len(m.AllowedPaths), len(m.ContentRules),
		convergenceLabel(m.ConvergenceTarget),
		m.TemplateVersionPinned, m.SkillVersionPinned, updated)
}

func convergenceLabel(c manifestv1.ConvergenceTarget) string {
	switch c {
	case manifestv1.ConvergenceTarget_CONVERGENCE_TARGET_NONE:
		return "none"
	case manifestv1.ConvergenceTarget_CONVERGENCE_TARGET_EMPTY_DIFF:
		return "empty-diff"
	default:
		return "unspecified"
	}
}
