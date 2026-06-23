package privacy

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/vrooli/cli-core/cliapp"
	privacyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/privacy"
	privacyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/privacy/privacy_v1connect"
	"google.golang.org/protobuf/proto"
)

type handlers struct{ core *cliapp.ScenarioApp }

func (h handlers) retention(ctx cliapp.RunContext) error {
	resp, err := cliapp.Call[*privacyv1.GetRetentionSettingsRequest, *privacyv1.GetRetentionSettingsResponse](h.core, http.MethodPost, privacyconnect.PrivacyServiceGetRetentionSettingsProcedure, &privacyv1.GetRetentionSettingsRequest{})
	if err != nil {
		return cliapp.WrapAPIError("get retention settings", err, nil)
	}
	return renderRetention(ctx, resp, resp.GetSettings())
}

func (h handlers) retentionSet(ctx cliapp.RunContext) error {
	settings := &privacyv1.RetentionSettings{
		QueryLogDays:   parseInt32(ctx.Flag("query-log-days")),
		SnapshotDays:   parseInt32(ctx.Flag("snapshot-days")),
		ExperimentDays: parseInt32(ctx.Flag("experiment-days")),
		Profile:        ctx.Flag("profile"),
	}
	resp, err := cliapp.Call[*privacyv1.UpdateRetentionSettingsRequest, *privacyv1.UpdateRetentionSettingsResponse](h.core, http.MethodPost, privacyconnect.PrivacyServiceUpdateRetentionSettingsProcedure, &privacyv1.UpdateRetentionSettingsRequest{Settings: settings})
	if err != nil {
		return cliapp.WrapAPIError("update retention settings", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp, cliapp.MutationReport{Result: []string{"Updated retention settings."}, Changes: []string{formatRetention(resp.GetSettings())}})
}

func (h handlers) visibility(ctx cliapp.RunContext) error {
	resp, err := cliapp.Call[*privacyv1.GetVisibilitySettingsRequest, *privacyv1.GetVisibilitySettingsResponse](h.core, http.MethodPost, privacyconnect.PrivacyServiceGetVisibilitySettingsProcedure, &privacyv1.GetVisibilitySettingsRequest{})
	if err != nil {
		return cliapp.WrapAPIError("get visibility settings", err, nil)
	}
	s := resp.GetSettings()
	return cliapp.RenderProtoList(ctx, resp, cliapp.ListReport{Summary: []string{fmt.Sprintf("query_domains=%t device_history=%t household_mode=%t", s.GetShowQueryDomains(), s.GetShowDeviceHistory(), s.GetHouseholdMode())}, ResultsHeading: "Notes", Results: s.GetNotes()})
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
