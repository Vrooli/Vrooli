package measures

import (
	"context"
	"fmt"
	"os"

	"connectrpc.com/connect"
	"vrooli-autoheal/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/measures"
	measuresconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/measures/measures_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name: "measure", Description: "Read typed reliability measures", NeedsAPI: true,
		Subcommands: []cliapp.Command{
			{Name: "uptime", Description: "Read uptime for one check", Run: func(args []string) error { return uptime(core, args) }},
			{Name: "outages", Description: "Read interval-weighted downtime for one supervised member", Run: func(args []string) error { return outages(core, args) }},
			{Name: "restarts", Description: "Count recorded restarts in a window", Run: func(args []string) error { return restarts(core, args) }},
			{Name: "heal-outcomes", Description: "Count recovery outcomes in a window", Run: func(args []string) error { return healOutcomes(core, args) }},
			{Name: "critical", Description: "Count critical observations in a window", Run: func(args []string) error { return critical(core, args) }},
		},
	}
}

func outages(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("measure outages")
	memberID := fs.String("member-id", "", "Supervised member identifier, such as resource-qdrant")
	window := fs.Int("window-hours", 24, "Measurement window in hours")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if *memberID == "" {
		return fmt.Errorf("--member-id is required")
	}
	response, err := client(core).GetOutageSummary(context.Background(), connect.NewRequest(&measuresv1.GetOutageSummaryRequest{MemberId: *memberID, WindowHours: int32(*window)}))
	if err != nil {
		return cliapp.WrapAPIError("read outage measure", err, nil)
	}
	return renderMessage(response.Msg, *jsonOutput, fmt.Sprintf("%s unavailable: %.3fs across %d outage(s)", response.Msg.Outage.MemberId, response.Msg.Outage.TotalUnavailableSeconds, response.Msg.Outage.DistinctOutageCount))
}

func client(core *cliapp.ScenarioApp) measuresconnect.MeasuresServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return measuresconnect.NewMeasuresServiceClient(httpClient, baseURL)
}

func uptime(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("measure uptime")
	checkID := fs.String("check-id", "", "Check identifier")
	window := fs.Int("window-hours", 24, "Measurement window in hours")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if *checkID == "" {
		return fmt.Errorf("--check-id is required")
	}
	response, err := client(core).GetUptimeByCheck(context.Background(), connect.NewRequest(&measuresv1.GetUptimeByCheckRequest{CheckId: *checkID, WindowHours: int32(*window)}))
	if err != nil {
		return cliapp.WrapAPIError("read uptime measure", err, nil)
	}
	return renderMessage(response.Msg, *jsonOutput, fmt.Sprintf("%s uptime: %.2f%%", response.Msg.Uptime.CheckId, response.Msg.Uptime.UptimePercent))
}

func restarts(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("measure restarts")
	window := fs.Int("window-hours", 24, "Measurement window in hours")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	response, err := client(core).GetRestartCount(context.Background(), connect.NewRequest(&measuresv1.GetRestartCountRequest{WindowHours: int32(*window)}))
	if err != nil {
		return cliapp.WrapAPIError("read restart measure", err, nil)
	}
	return renderMessage(response.Msg, *jsonOutput, fmt.Sprintf("restarts: %d", response.Msg.Restarts.Count))
}

func healOutcomes(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("measure heal-outcomes")
	window := fs.Int("window-hours", 24, "Measurement window in hours")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	response, err := client(core).GetHealOutcomes(context.Background(), connect.NewRequest(&measuresv1.GetHealOutcomesRequest{WindowHours: int32(*window)}))
	if err != nil {
		return cliapp.WrapAPIError("read healing outcome measure", err, nil)
	}
	return renderMessage(response.Msg, *jsonOutput, fmt.Sprintf("healing outcomes: %d buckets", len(response.Msg.Outcomes)))
}

func critical(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("measure critical")
	window := fs.Int("window-hours", 24, "Measurement window in hours")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	response, err := client(core).GetCriticalCount(context.Background(), connect.NewRequest(&measuresv1.GetCriticalCountRequest{WindowHours: int32(*window)}))
	if err != nil {
		return cliapp.WrapAPIError("read critical measure", err, nil)
	}
	return renderMessage(response.Msg, *jsonOutput, fmt.Sprintf("critical observations: %d", response.Msg.Critical.Count))
}

func renderMessage(message proto.Message, jsonOutput bool, summary string) error {
	if jsonOutput {
		encoded, err := protojson.MarshalOptions{Indent: "  "}.Marshal(message)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(os.Stdout, string(encoded))
		return err
	}
	return cliapp.RenderOperationalReport(os.Stdout, cliapp.OperationalReport{Status: []string{summary}})
}
