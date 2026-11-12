package launchapi

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/schedule"

	"research-desk/internal/home"
	"research-desk/internal/launch"
	"research-desk/internal/peer"
	"research-desk/internal/skillscripts"
	"research-desk/internal/testutil"
	"research-desk/internal/writer"

	activityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/activity"
	launchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/launch"
	launchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/launch/launch_v1connect"
)

// Handlers run over the testutil archive of real research-home excerpts; packet.py and Web
// Console sit behind seams the internal/launch tests cover. These tests pin
// the wire contract: codes, the messages read, and the automatic switch.

const prepared = "local-models-w1-20261111"

type packets struct{}

func (packets) Preview(context.Context, []byte) (*skillscripts.PacketResult, error) {
	return &skillscripts.PacketResult{OK: true}, nil
}

func (packets) Build(context.Context, []byte) (*skillscripts.PacketResult, error) {
	return nil, errors.New("packet.py: no python3 on PATH")
}

func (packets) Check(_ context.Context, id string) (*skillscripts.PacketResult, error) {
	return &skillscripts.PacketResult{OK: true, CommissionID: id}, nil
}

func newHandler(t *testing.T, automatic error) (*handler, string) {
	t.Helper()
	h := testutil.NewHome(t)
	research := h.Research
	clock := schedule.NewFake(time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC))
	reader := home.NewReader(home.Config{ResearchDir: research, CodexDir: h.Codex}, home.Deps{Clock: clock, Procs: home.FixedProcs{}})
	w := writer.New(research, h.State, testutil.EntryChars)
	t.Cleanup(func() { w.Close() })
	limits := &skillscripts.Limits{Path: "limits.json"}
	limits.Defaults.SupervisorModel = "gpt-5.5"
	limits.Models.SupervisorModel = []string{"gpt-5.5", "gpt-5.4"}
	limits.Packet.TopicsMax, limits.Packet.AllowanceHoursMin, limits.Packet.AllowanceHoursMax = 6, 1, 72
	limits.Packet.AllowanceHoursDefault, limits.Packet.OrderedWorkstreamsMax = 24, 4
	service := &launch.Service{
		Reader: reader, Writer: w, Packets: packets{}, Launcher: &launch.Launcher{}, Clock: clock, Logger: log.New(io.Discard, "", 0),
		Limits: func() (*skillscripts.Limits, error) { return limits, nil },
	}
	return NewHandler(Deps{ResearchDir: research, Service: service, Automatic: func(context.Context) (string, error) { return peer.AccessAvailability, automatic }, Logger: log.New(io.Discard, "", 0)}), research
}

// The access used is shown, and a refusal turns automatic launch off
// with the reason.
func TestOptionsShowTheWebConsoleAccessAndARefusal(t *testing.T) {
	h, _ := newHandler(t, nil)
	resp, err := h.GetLaunchOptions(context.Background(), connect.NewRequest(&launchv1.GetLaunchOptionsRequest{}))
	if err != nil || !resp.Msg.AutomaticAvailable || resp.Msg.WebConsoleAccess != "local availability policy (no desk credential)" ||
		resp.Msg.WebConsoleAccessKind != activityv1.WebConsoleAccess_WEB_CONSOLE_ACCESS_AVAILABILITY_POLICY {
		t.Fatalf("options = %+v %v", resp, err)
	}

	h, _ = newHandler(t, errors.New("web-console refused /web_console.v1.sessions.SessionsService/CreateSession: missing web-console:write"))
	resp, err = h.GetLaunchOptions(context.Background(), connect.NewRequest(&launchv1.GetLaunchOptionsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	m := resp.Msg
	if m.AutomaticAvailable || !strings.HasPrefix(m.AutomaticReason, "web-console refused") || !strings.HasSuffix(m.AutomaticReason, "launch by hand with the two messages") || m.DefaultSupervisorModel != "gpt-5.5" || len(m.SupervisorModels) != 2 {
		t.Fatalf("options = %+v", m)
	}
	if p := m.PacketLimits; p.GetTopicsMax() != 6 || p.GetAllowanceHoursMin() != 1 || p.GetAllowanceHoursMax() != 72 ||
		p.GetAllowanceHoursDefault() != 24 || p.GetOrderedWorkstreamsMax() != 4 {
		t.Errorf("the packet limits come from limits.json, so the form holds no copy: %+v", p)
	}
}

// [REQ:RD-P0-001]
func TestTheTwoLaunchMessagesAreARead(t *testing.T) {
	h, research := newHandler(t, nil)
	resp, err := h.GetLaunchMessages(context.Background(), connect.NewRequest(&launchv1.GetLaunchMessagesRequest{CommissionId: prepared}))
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(research, "cohorts", prepared)
	stance, _ := os.ReadFile(filepath.Join(dir, home.StanceFile))
	if resp.Msg.Stance != string(stance) || resp.Msg.Kernel == "" || strings.ContainsAny(resp.Msg.Kernel, "\n") || resp.Msg.CharterPath != filepath.Join(dir, home.CharterFile) {
		t.Fatalf("messages = %+v", resp.Msg)
	}
	if _, err := os.Stat(filepath.Join(dir, home.LaunchFile)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("reading the messages wrote LAUNCH.json: %v", err)
	}
	if slices.Contains(OperatorProcedures, launchconnect.LaunchServiceGetLaunchMessagesProcedure) {
		t.Fatal("reading the messages is a read, open to every verified caller")
	}
}

func TestErrorsCarryCodesAClientCanActOn(t *testing.T) {
	h, _ := newHandler(t, nil)
	ctx := context.Background()
	for name, tc := range map[string]struct {
		call func() error
		want connect.Code
	}{
		"preflight without a target": {func() error {
			_, err := h.Preflight(ctx, connect.NewRequest(&launchv1.PreflightRequest{}))
			return err
		}, connect.CodeInvalidArgument},
		"unknown commission": {func() error {
			_, err := h.GetLaunch(ctx, connect.NewRequest(&launchv1.GetLaunchRequest{CommissionId: "no-such-20261112"}))
			return err
		}, connect.CodeNotFound},
		"unspecified control": {func() error {
			_, err := h.Control(ctx, connect.NewRequest(&launchv1.ControlRequest{CommissionId: prepared}))
			return err
		}, connect.CodeInvalidArgument},
		"discard without a reason": {func() error {
			_, err := h.Control(ctx, connect.NewRequest(&launchv1.ControlRequest{CommissionId: prepared, Action: launchv1.ControlAction_CONTROL_ACTION_DISCARD}))
			return err
		}, connect.CodeFailedPrecondition},
		"pause a prepared commission": {func() error {
			_, err := h.Control(ctx, connect.NewRequest(&launchv1.ControlRequest{CommissionId: prepared, Action: launchv1.ControlAction_CONTROL_ACTION_PAUSE}))
			return err
		}, connect.CodeFailedPrecondition},
		"packet.py unavailable": {func() error {
			_, err := h.Build(ctx, connect.NewRequest(&launchv1.BuildRequest{Source: &launchv1.BuildRequest_SpecJson{SpecJson: `{"topics":["x"]}`}}))
			return err
		}, connect.CodeUnavailable},
	} {
		if got := connect.CodeOf(tc.call()); got != tc.want {
			t.Errorf("%s: code %v, want %v", name, got, tc.want)
		}
	}
}

func TestNoLaunchTestHasRunYet(t *testing.T) {
	h, _ := newHandler(t, nil)
	resp, err := h.GetLaunchTest(context.Background(), connect.NewRequest(&launchv1.GetLaunchTestRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.ReportPath != "" || resp.Msg.Running {
		t.Fatalf("report = %+v", resp.Msg)
	}
}
