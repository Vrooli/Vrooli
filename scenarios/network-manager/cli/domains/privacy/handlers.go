package privacy

import (
	"context"
	"fmt"
	"strconv"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	privacyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/privacy"
	privacyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/privacy/privacy_v1connect"
	"google.golang.org/protobuf/proto"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client privacyconnect.PrivacyServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return handlers{
		core:   core,
		client: privacyconnect.NewPrivacyServiceClient(httpClient, baseURL),
	}
}

func (h handlers) retention(ctx cliapp.RunContext) error {
	resp, err := h.client.GetRetentionSettings(context.Background(), connect.NewRequest(&privacyv1.GetRetentionSettingsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get retention settings", err, nil)
	}
	return renderRetention(ctx, resp.Msg, resp.Msg.GetSettings())
}

func (h handlers) retentionSet(ctx cliapp.RunContext) error {
	current, err := h.client.GetRetentionSettings(context.Background(), connect.NewRequest(&privacyv1.GetRetentionSettingsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get current retention settings", err, nil)
	}
	settings := current.Msg.GetSettings()
	if settings == nil {
		settings = &privacyv1.RetentionSettings{}
	}
	settings = &privacyv1.RetentionSettings{
		QueryLogDays:   settings.GetQueryLogDays(),
		SnapshotDays:   settings.GetSnapshotDays(),
		ExperimentDays: settings.GetExperimentDays(),
		Profile:        settings.GetProfile(),
	}
	if raw := ctx.Flag("query-log-days"); raw != "" {
		settings.QueryLogDays = parseInt32(raw)
	}
	if raw := ctx.Flag("snapshot-days"); raw != "" {
		settings.SnapshotDays = parseInt32(raw)
	}
	if raw := ctx.Flag("experiment-days"); raw != "" {
		settings.ExperimentDays = parseInt32(raw)
	}
	if raw := ctx.Flag("profile"); raw != "" {
		settings.Profile = raw
	}
	resp, err := h.client.UpdateRetentionSettings(context.Background(), connect.NewRequest(&privacyv1.UpdateRetentionSettingsRequest{Settings: settings}))
	if err != nil {
		return cliapp.WrapAPIError("update retention settings", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{"Updated retention settings."}, Changes: []string{formatRetention(resp.Msg.GetSettings())}})
}

func (h handlers) visibility(ctx cliapp.RunContext) error {
	resp, err := h.client.GetVisibilitySettings(context.Background(), connect.NewRequest(&privacyv1.GetVisibilitySettingsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get visibility settings", err, nil)
	}
	s := resp.Msg.GetSettings()
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("query_domains=%t device_history=%t household_mode=%t", s.GetShowQueryDomains(), s.GetShowDeviceHistory(), s.GetHouseholdMode())}, ResultsHeading: "Notes", Results: s.GetNotes()})
}

func renderRetention(ctx cliapp.RunContext, payload proto.Message, s *privacyv1.RetentionSettings) error {
	return cliapp.RenderProtoList(ctx, payload, cliapp.ListReport{Summary: []string{formatRetention(s)}, ResultsHeading: "Defaults", Results: []string{"Minimal retention is the default P0 privacy posture."}})
}

func formatRetention(s *privacyv1.RetentionSettings) string {
	if s == nil {
		return "Retention settings unavailable."
	}
	return fmt.Sprintf("profile=%s query_log_days=%d snapshot_days=%d experiment_days=%d", s.GetProfile(), s.GetQueryLogDays(), s.GetSnapshotDays(), s.GetExperimentDays())
}

func parseInt32(v string) int32 {
	if v == "" {
		return 0
	}
	n, _ := strconv.ParseInt(v, 10, 32)
	return int32(n)
}
