package measures

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/measures"
	sharedmeasuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
)

type canceledMeasuresClient struct {
	contextSeen context.Context
}

func (c *canceledMeasuresClient) capture(ctx context.Context) error {
	c.contextSeen = ctx
	return ctx.Err()
}

func (c *canceledMeasuresClient) CountBYOKCredentials(ctx context.Context, _ *connect.Request[measuresv1.CountBYOKCredentialsRequest]) (*connect.Response[measuresv1.CountBYOKCredentialsResponse], error) {
	return nil, c.capture(ctx)
}

func (c *canceledMeasuresClient) CountCorpusClips(ctx context.Context, _ *connect.Request[measuresv1.CountCorpusClipsRequest]) (*connect.Response[measuresv1.CountCorpusClipsResponse], error) {
	return nil, c.capture(ctx)
}

func (c *canceledMeasuresClient) CountExperimentRuns(ctx context.Context, _ *connect.Request[measuresv1.CountExperimentRunsRequest]) (*connect.Response[measuresv1.CountExperimentRunsResponse], error) {
	return nil, c.capture(ctx)
}

func (c *canceledMeasuresClient) CountExperiments(ctx context.Context, _ *connect.Request[measuresv1.CountExperimentsRequest]) (*connect.Response[measuresv1.CountExperimentsResponse], error) {
	return nil, c.capture(ctx)
}

func (c *canceledMeasuresClient) CountSpeakerProfiles(ctx context.Context, _ *connect.Request[measuresv1.CountSpeakerProfilesRequest]) (*connect.Response[measuresv1.CountSpeakerProfilesResponse], error) {
	return nil, c.capture(ctx)
}

func (c *canceledMeasuresClient) CountWakewordTemplates(ctx context.Context, _ *connect.Request[measuresv1.CountWakewordTemplatesRequest]) (*connect.Response[measuresv1.CountWakewordTemplatesResponse], error) {
	return nil, c.capture(ctx)
}

func (c *canceledMeasuresClient) CountDuplexAccountingOperations(ctx context.Context, _ *connect.Request[measuresv1.CountDuplexAccountingOperationsRequest]) (*connect.Response[measuresv1.CountDuplexAccountingOperationsResponse], error) {
	return nil, c.capture(ctx)
}

func TestCountPropagatesCanceledCommandContextToEveryRPC(t *testing.T) {
	methods := []string{
		"CountBYOKCredentials", "CountCorpusClips", "CountExperimentRuns",
		"CountExperiments", "CountSpeakerProfiles", "CountWakewordTemplates", "CountDuplexAccountingOperations",
	}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			client := &canceledMeasuresClient{}
			commandCtx, cancel := context.WithCancel(context.Background())
			cancel()
			h := &handlers{
				client: client,
				requestContext: func() (context.Context, context.CancelFunc) {
					return commandCtx, func() {}
				},
			}
			output := &bytes.Buffer{}
			runCtx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
				Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "window", Default: "this_week"}}},
				Stdout: output,
			})

			err := h.count(method, "test rows").Run(runCtx)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("count error = %v, want context.Canceled", err)
			}
			if client.contextSeen != commandCtx {
				t.Fatal("RPC did not receive the command context")
			}
			if output.Len() != 0 {
				t.Fatalf("canceled command wrote output %q", output.String())
			}
		})
	}
}

type recordingMeasuresClient struct {
	method      string
	window      *sharedmeasuresv1.TimeWindow
	contextSeen context.Context
	count       int64
	err         error
}

func (c *recordingMeasuresClient) record(ctx context.Context, method string, window *sharedmeasuresv1.TimeWindow) error {
	c.contextSeen, c.method, c.window = ctx, method, window
	return c.err
}

func (c *recordingMeasuresClient) CountBYOKCredentials(ctx context.Context, req *connect.Request[measuresv1.CountBYOKCredentialsRequest]) (*connect.Response[measuresv1.CountBYOKCredentialsResponse], error) {
	if err := c.record(ctx, "CountBYOKCredentials", req.Msg.GetWindow()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&measuresv1.CountBYOKCredentialsResponse{Count: c.count}), nil
}

func (c *recordingMeasuresClient) CountCorpusClips(ctx context.Context, req *connect.Request[measuresv1.CountCorpusClipsRequest]) (*connect.Response[measuresv1.CountCorpusClipsResponse], error) {
	if err := c.record(ctx, "CountCorpusClips", req.Msg.GetWindow()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&measuresv1.CountCorpusClipsResponse{Count: c.count}), nil
}

func (c *recordingMeasuresClient) CountExperimentRuns(ctx context.Context, req *connect.Request[measuresv1.CountExperimentRunsRequest]) (*connect.Response[measuresv1.CountExperimentRunsResponse], error) {
	if err := c.record(ctx, "CountExperimentRuns", req.Msg.GetWindow()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&measuresv1.CountExperimentRunsResponse{Count: c.count}), nil
}

func (c *recordingMeasuresClient) CountExperiments(ctx context.Context, req *connect.Request[measuresv1.CountExperimentsRequest]) (*connect.Response[measuresv1.CountExperimentsResponse], error) {
	if err := c.record(ctx, "CountExperiments", req.Msg.GetWindow()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&measuresv1.CountExperimentsResponse{Count: c.count}), nil
}

func (c *recordingMeasuresClient) CountSpeakerProfiles(ctx context.Context, req *connect.Request[measuresv1.CountSpeakerProfilesRequest]) (*connect.Response[measuresv1.CountSpeakerProfilesResponse], error) {
	if err := c.record(ctx, "CountSpeakerProfiles", req.Msg.GetWindow()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&measuresv1.CountSpeakerProfilesResponse{Count: c.count}), nil
}

func (c *recordingMeasuresClient) CountWakewordTemplates(ctx context.Context, req *connect.Request[measuresv1.CountWakewordTemplatesRequest]) (*connect.Response[measuresv1.CountWakewordTemplatesResponse], error) {
	if err := c.record(ctx, "CountWakewordTemplates", req.Msg.GetWindow()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&measuresv1.CountWakewordTemplatesResponse{Count: c.count}), nil
}

func (c *recordingMeasuresClient) CountDuplexAccountingOperations(ctx context.Context, req *connect.Request[measuresv1.CountDuplexAccountingOperationsRequest]) (*connect.Response[measuresv1.CountDuplexAccountingOperationsResponse], error) {
	if err := c.record(ctx, "CountDuplexAccountingOperations", req.Msg.GetWindow()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&measuresv1.CountDuplexAccountingOperationsResponse{Count: c.count}), nil
}

func TestCountHappyPathCoversAllRPCsAndRenderers(t *testing.T) {
	cases := []struct {
		method string
		unit   string
	}{
		{"CountBYOKCredentials", "credential capabilities"},
		{"CountCorpusClips", "corpus clips"},
		{"CountExperimentRuns", "experiment runs"},
		{"CountExperiments", "experiments"},
		{"CountSpeakerProfiles", "speaker profiles"},
		{"CountWakewordTemplates", "wakeword templates"},
		{"CountDuplexAccountingOperations", "duplex accounting operations"},
	}
	for _, tc := range cases {
		for _, jsonOutput := range []bool{false, true} {
			name := "human"
			if jsonOutput {
				name = "json"
			}
			t.Run(tc.method+"/"+name, func(t *testing.T) {
				client := &recordingMeasuresClient{count: 7}
				requestCtx := context.Background()
				h := &handlers{
					client: client,
					requestContext: func() (context.Context, context.CancelFunc) {
						return requestCtx, func() {}
					},
				}
				output := &bytes.Buffer{}
				runCtx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
					Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "window", Default: "this_week"}}},
					Flags:  map[string]string{"window": "last_7d"},
					JSON:   jsonOutput,
					Stdout: output,
				})
				if err := h.count(tc.method, tc.unit).Run(runCtx); err != nil {
					t.Fatalf("count failed: %v", err)
				}
				if client.method != tc.method {
					t.Fatalf("RPC = %q, want %q", client.method, tc.method)
				}
				if client.contextSeen != requestCtx {
					t.Fatal("RPC did not receive request context")
				}
				if got := client.window.GetToken(); got != sharedmeasuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_7D {
					t.Fatalf("request window = %v, want last_7d", got)
				}
				want := "7 " + tc.unit + " created (last_7d)\n"
				if jsonOutput {
					want = `{"count":7,"unit":"` + tc.unit + `","window":"last_7d"}` + "\n"
				}
				if output.String() != want {
					t.Fatalf("output = %q, want %q", output.String(), want)
				}
			})
		}
	}
}

func TestCountDefaultsWindowAndRejectsUnknownWindow(t *testing.T) {
	t.Run("defaults blank window", func(t *testing.T) {
		client := &recordingMeasuresClient{count: 2}
		h := &handlers{client: client, requestContext: noCancelContext}
		ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
			Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "window", Default: " "}}},
			Stdout: &bytes.Buffer{},
		})
		if err := h.count("CountCorpusClips", "corpus clips").Run(ctx); err != nil {
			t.Fatal(err)
		}
		if got := client.window.GetToken(); got != sharedmeasuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_THIS_WEEK {
			t.Fatalf("default window = %v, want this_week", got)
		}
	})

	t.Run("rejects unknown window before RPC", func(t *testing.T) {
		client := &recordingMeasuresClient{}
		h := &handlers{client: client, requestContext: noCancelContext}
		ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
			Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "window"}}},
			Flags:  map[string]string{"window": "yesterday-ish"},
			Stdout: &bytes.Buffer{},
		})
		err := h.count("CountCorpusClips", "corpus clips").Run(ctx)
		if err == nil || !strings.Contains(err.Error(), `unknown time window "yesterday-ish"`) {
			t.Fatalf("error = %v, want unknown time window", err)
		}
		if client.method != "" {
			t.Fatalf("unexpected RPC call %q", client.method)
		}
	})
}

func TestTimeWindowMapsEverySupportedToken(t *testing.T) {
	windows := map[string]sharedmeasuresv1.TimeWindowToken{
		"this_week":    sharedmeasuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_THIS_WEEK,
		"last_7d":      sharedmeasuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_7D,
		"last_30d":     sharedmeasuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_30D,
		"this_month":   sharedmeasuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_THIS_MONTH,
		"last_month":   sharedmeasuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_MONTH,
		"this_quarter": sharedmeasuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_THIS_QUARTER,
	}
	for token, want := range windows {
		got, err := timeWindow(token)
		if err != nil {
			t.Errorf("timeWindow(%q): %v", token, err)
			continue
		}
		if got.GetToken() != want {
			t.Errorf("timeWindow(%q) = %v, want %v", token, got.GetToken(), want)
		}
	}
	if _, err := timeWindow("unknown"); err == nil {
		t.Fatal("timeWindow(unknown) unexpectedly succeeded")
	}
}

func TestCountReturnsRPCError(t *testing.T) {
	wantErr := errors.New("measure service unavailable")
	client := &recordingMeasuresClient{err: wantErr}
	h := &handlers{client: client, requestContext: noCancelContext}
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "window", Default: "this_week"}}},
		Stdout: &bytes.Buffer{},
	})
	err := h.count("CountExperiments", "experiments").Run(ctx)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want wrapped service error", err)
	}
}

func TestRegisterLoadsAllMeasureCommands(t *testing.T) {
	manifest, err := os.ReadFile("../../manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	group, err := Register(nil, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if group.Name != GroupName || len(group.Subcommands) != 7 {
		t.Fatalf("registered group = %q with %d commands; want %q with 7", group.Name, len(group.Subcommands), GroupName)
	}
	for _, command := range group.Subcommands {
		if command.RunCtx == nil {
			t.Errorf("command %q has no handler", command.Name)
		}
	}
}

func noCancelContext() (context.Context, context.CancelFunc) {
	return context.Background(), func() {}
}
