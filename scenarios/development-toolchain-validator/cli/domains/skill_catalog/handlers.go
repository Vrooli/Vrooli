package skill_catalog

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	skillcatalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/skill_catalog"
	skillcatalogconnect "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/skill_catalog/skill_catalog_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client skillcatalogconnect.SkillCatalogServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: skillcatalogconnect.NewSkillCatalogServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) sync(ctx cliapp.RunContext) error {
	resp, err := h.client.Sync(context.Background(), connect.NewRequest(&skillcatalogv1.SyncRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("sync skill catalog", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no sync response")
	}
	changes := []string{
		fmt.Sprintf("added=%d updated=%d removed=%d", resp.Msg.Added, resp.Msg.Updated, resp.Msg.Removed),
	}
	for _, s := range resp.Msg.Skills {
		changes = append(changes, formatSkill(s))
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Synced skill catalog (%d skills total).", len(resp.Msg.Skills))},
		Changes: changes,
		NextCommand: []string{
			"`skill-catalog list` — show mirrored skills",
		},
	})
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListSkills(context.Background(), connect.NewRequest(&skillcatalogv1.ListSkillsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list skill catalog", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no skills response")
	}
	results := make([]string, 0, len(resp.Msg.Skills))
	for _, s := range resp.Msg.Skills {
		results = append(results, formatSkill(s))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Mirroring %d skill(s).", len(resp.Msg.Skills))},
		ResultsHeading: "Skills",
		Results:        results,
		RetrievalHints: []string{
			"`skill-catalog get <id>` — show a single mirrored skill",
			"`skill-catalog sync` — refresh from prompt-manager",
		},
	})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetSkill(context.Background(), connect.NewRequest(&skillcatalogv1.GetSkillRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get skill %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Skill == nil {
		return fmt.Errorf("server returned no skill")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched skill %s.", resp.Msg.Skill.Id)},
		ResultsHeading: "Skill",
		Results:        []string{formatSkill(resp.Msg.Skill)},
	})
}

func formatSkill(s *skillcatalogv1.Skill) string {
	if s == nil {
		return "(nil)"
	}
	synced := ""
	if s.SyncedAt != nil {
		synced = s.SyncedAt.AsTime().Format(time.RFC3339)
	}
	hash := s.ContentHash
	if len(hash) > 12 {
		hash = hash[:12] + "…"
	}
	return fmt.Sprintf("%s — version=%s content_hash=%s synced=%s",
		s.Id, s.Version, hash, synced)
}
