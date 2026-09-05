package experiment

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
	experimentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment/experiment_v1connect"

	testutil "github.com/vrooli/cli-core/cliapptest"
)

type fakeExperimentSvc struct {
	experimentconnect.UnimplementedExperimentServiceHandler
	startFn  func(*experimentv1.StartExperimentRequest) (*experimentv1.StartExperimentResponse, error)
	waitFn   func(*experimentv1.WaitExperimentRequest) (*experimentv1.WaitExperimentResponse, error)
	reportFn func(*experimentv1.GetExperimentReportRequest) (*experimentv1.GetExperimentReportResponse, error)
	deleteFn func(*experimentv1.DeleteExperimentRequest) (*experimentv1.DeleteExperimentResponse, error)
}

func fakeResponse[Request, Response any](fn func(*Request) (*Response, error), req *connect.Request[Request]) (*connect.Response[Response], error) {
	response, err := fn(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(response), nil
}

func (f *fakeExperimentSvc) StartExperiment(_ context.Context, req *connect.Request[experimentv1.StartExperimentRequest]) (*connect.Response[experimentv1.StartExperimentResponse], error) {
	return fakeResponse(f.startFn, req)
}

func (f *fakeExperimentSvc) WaitExperiment(_ context.Context, req *connect.Request[experimentv1.WaitExperimentRequest]) (*connect.Response[experimentv1.WaitExperimentResponse], error) {
	return fakeResponse(f.waitFn, req)
}

func (f *fakeExperimentSvc) GetExperimentReport(_ context.Context, req *connect.Request[experimentv1.GetExperimentReportRequest]) (*connect.Response[experimentv1.GetExperimentReportResponse], error) {
	return fakeResponse(f.reportFn, req)
}

func (f *fakeExperimentSvc) DeleteExperiment(_ context.Context, req *connect.Request[experimentv1.DeleteExperimentRequest]) (*connect.Response[experimentv1.DeleteExperimentResponse], error) {
	return fakeResponse(f.deleteFn, req)
}

func mountExperiment(t *testing.T, svc experimentconnect.ExperimentServiceHandler) *cliapp.ScenarioApp {
	t.Helper()
	path, h := experimentconnect.NewExperimentServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, h)
	return testutil.NewTestApp(t, mux)
}

func experimentStartSchema() cliapp.ArgSchema {
	names := []string{
		"name",
		"strategies",
		"clip-ids",
		"realtime-repeats",
		"latency-tail-seconds",
		"chunk-ms",
		"dropped-span-threshold",
		"overlap-max-window-ms",
		"overlap-max-stall-rejects",
		"overlap-window-ms",
		"overlap-commit-runs",
		"vad-silence-ms",
		"seed",
		"long-form",
		"target-duration-seconds",
		"gap-ms",
		"tag-contains",
		"sweep-durations",
		"noise-types",
		"snr-db",
		"competing-voices",
		"competing-text",
		"target-profile-id",
		"speaker-extraction",
		"speaker-verification",
		"speaker-mode",
		"speaker-threshold",
		"speaker-fallback",
		"speaker-ablation",
		"estimated-seconds",
		"recipe-json",
		"recipe-file",
	}
	flags := make([]cliapp.Flag, 0, len(names)+1)
	for _, name := range names {
		flags = append(flags, cliapp.Flag{Name: name})
	}
	flags = append(flags, cliapp.Flag{Name: "dry-run", Bool: true})
	return cliapp.ArgSchema{Flags: flags}
}

func TestPrintReportTableSurfacesDecisionSignals(t *testing.T) {
	app := testutil.NewTestApp(t, http.NewServeMux())
	ctx, buf := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})

	printReportTable(ctx, &evalv1.EvalReport{
		LatencyMeasured: false,
		Summary: &evalv1.EvalReportSummary{
			Recommendation:  "Prefer batch / clean for this corpus.",
			Confidence:      "low",
			Reasons:         []string{"Lowest WER after deterministic normalization."},
			ConfidenceNotes: []string{"Latency was not measured."},
		},
		Warnings: []*evalv1.ReportWarning{{
			Code:     "latency_not_measured",
			Severity: "info",
			Message:  "Latency columns were not measured because real-time repeats were disabled.",
		}},
		PerStrategy: []*evalv1.StrategyReport{{
			Label:               "batch / clean",
			Wer:                 0.02,
			WhisperCalls:        1,
			Rtf:                 0.4,
			WhisperAudioSeconds: 12,
			Verdict:             "winner",
			Safety:              &evalv1.SafetyGateReport{Passed: true},
			LengthCurves: []*evalv1.LengthBucketCurve{{
				Bucket:              "short",
				ClipCount:           2,
				Wer:                 0.02,
				MaxDroppedSpanWords: 0,
			}},
			Scaling: &evalv1.ScalingAnalysis{
				LatencyClassification: "linear",
				ComputeClassification: "flat",
				Confidence:            "medium",
				Points: []*evalv1.ScalingPoint{
					{ClipId: "clip-30", RealizedDurationMs: 30_000},
					{ClipId: "clip-60", RealizedDurationMs: 60_000},
					{ClipId: "clip-120", RealizedDurationMs: 120_000},
				},
			},
		}},
	})

	out := buf.String()
	require.Contains(t, out, "Recommendation: Prefer batch / clean for this corpus. (confidence: low)")
	require.Contains(t, out, "VERDICT")
	require.Contains(t, out, "winner")
	require.Contains(t, out, "Warnings:")
	require.Contains(t, out, "info/latency_not_measured")
	require.Contains(t, out, "Length curves:")
	require.Contains(t, out, "short")
	require.Contains(t, out, "Scaling analysis:")
	require.Contains(t, out, "linear")
	require.Contains(t, out, "flat")
	require.NotContains(t, out, "metrics=")
}

func TestPrintComparisonRendersWinnerRowsAndKeepsNilReportVisible(t *testing.T) {
	app := testutil.NewTestApp(t, http.NewServeMux())
	ctx, buf := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})

	mkReport := func(winnerWer, winnerRtf float64, passed bool) *evalv1.EvalReport {
		return &evalv1.EvalReport{
			Summary: &evalv1.EvalReportSummary{WinnerStrategy: "batch"},
			PerStrategy: []*evalv1.StrategyReport{
				{Strategy: "batch", Label: "batch / clean", Wer: winnerWer, Rtf: winnerRtf, WhisperCalls: 1, Safety: &evalv1.SafetyGateReport{Passed: passed}, Scaling: &evalv1.ScalingAnalysis{
					LatencyClassification: "linear",
					ComputeClassification: "flat",
					Confidence:            "medium",
					Points:                []*evalv1.ScalingPoint{{ClipId: "clip-30", RealizedDurationMs: 30_000}},
				}},
				{Strategy: "vad_segment", Label: "vad", Wer: winnerWer + 0.05, Rtf: winnerRtf + 1},
			},
		}
	}

	printComparison(ctx, []*experimentv1.ComparedExperiment{
		{
			Experiment: &experimentv1.Experiment{Id: "exp-aaa", Name: "alpha", Status: experimentv1.ExperimentStatus_EXPERIMENT_STATUS_SUCCEEDED},
			Report:     mkReport(0.08, 0.5, false),
		},
		{
			// Best: lowest winner WER.
			Experiment: &experimentv1.Experiment{Id: "exp-bbb", Name: "beta", Status: experimentv1.ExperimentStatus_EXPERIMENT_STATUS_SUCCEEDED},
			Report:     mkReport(0.02, 0.4, true),
		},
		{
			// Still running — nil report must remain visible.
			Experiment: &experimentv1.Experiment{Id: "exp-ccc", Name: "gamma", Status: experimentv1.ExperimentStatus_EXPERIMENT_STATUS_RUNNING, Error: "all speaker experiment conditions were skipped"},
			Report:     nil,
		},
	})

	out := buf.String()
	require.Contains(t, out, "exp-aaa")
	require.Contains(t, out, "exp-bbb")
	require.Contains(t, out, "exp-ccc", "nil-report experiment row must still render")
	require.Contains(t, out, "running", "nil-report experiment must show its status")
	require.Contains(t, out, "SAFE")
	require.Contains(t, out, "SCALE")
	require.Contains(t, out, "linear/flat")
	require.Contains(t, out, "UNSAFE")
	require.Contains(t, out, "* best")
	require.Contains(t, out, "Experiment errors:")
	require.Contains(t, out, "all speaker experiment conditions were skipped")
	// The best (beta) row carries the marker.
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "exp-bbb") {
			require.True(t, strings.HasPrefix(strings.TrimSpace(line), "*"), "best experiment row should be marked: %q", line)
		}
	}
}

func TestPrintComparisonRendersRecipeDiffAndStrategyAlignment(t *testing.T) {
	app := testutil.NewTestApp(t, http.NewServeMux())
	ctx, buf := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})

	mkReport := func(wer float64, p95 float64) *evalv1.EvalReport {
		return &evalv1.EvalReport{
			LatencyMeasured: true,
			Summary:         &evalv1.EvalReportSummary{WinnerStrategy: "overlap_agree"},
			PerStrategy: []*evalv1.StrategyReport{{
				Strategy:                 "overlap_agree",
				Label:                    "overlap agree",
				Wer:                      wer,
				FinalizationLatencyP95Ms: p95,
				FinalizationLatencyP50Ms: p95 / 2,
				WhisperCalls:             2,
				Rtf:                      0.6,
				PartialRevisions:         1,
				WhisperAudioSeconds:      10,
				CommitCount:              1,
				Safety:                   &evalv1.SafetyGateReport{Passed: true},
				Reasons:                  []string{"best overlap row"},
				RefWords:                 100,
				Verdict:                  "winner",
			}},
		}
	}
	mkRecipe := func(stallRejects int32) *experimentv1.ExperimentRecipe {
		return &experimentv1.ExperimentRecipe{Strategies: []*evalv1.EvalStrategy{{
			Kind:                   "overlap_agree",
			OverlapMaxStallRejects: stallRejects,
		}}}
	}

	printComparison(ctx, []*experimentv1.ComparedExperiment{
		{
			Experiment: &experimentv1.Experiment{Id: "exp-aaa", Name: "stall-3", Status: experimentv1.ExperimentStatus_EXPERIMENT_STATUS_SUCCEEDED, Recipe: mkRecipe(3)},
			Report:     mkReport(0.03, 900),
		},
		{
			Experiment: &experimentv1.Experiment{Id: "exp-bbb", Name: "stall-8", Status: experimentv1.ExperimentStatus_EXPERIMENT_STATUS_SUCCEEDED, Recipe: mkRecipe(8)},
			Report:     mkReport(0.02, 700),
		},
	})

	out := buf.String()
	require.Contains(t, out, "Recipe differences:")
	require.Contains(t, out, "strategy.overlap_agree.overlap_max_stall_rejects: 3 -> 8")
	require.Contains(t, out, "By-strategy alignment:")
	require.Contains(t, out, "overlap_agree")
	require.Contains(t, out, "wer 3.0 p95 900ms")
	require.Contains(t, out, "wer 2.0 p95 700ms")
}

func TestFormatRunStatusUsesConditionOnly(t *testing.T) {
	line := formatRunStatus(&experimentv1.ExperimentRun{
		Strategy:      "batch",
		ConditionJson: `{}`,
	})

	require.Equal(t, "batch - completed", strings.TrimSpace(line))
	require.NotContains(t, line, "metrics=")
}

func TestRenderExperimentSurfacesFailureError(t *testing.T) {
	app := testutil.NewTestApp(t, http.NewServeMux())
	ctx, buf := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})

	err := renderExperiment(ctx, &experimentv1.GetExperimentResponse{
		Experiment: &experimentv1.Experiment{
			Id:     "exp-failed",
			Name:   "failed speaker run",
			Status: experimentv1.ExperimentStatus_EXPERIMENT_STATUS_FAILED,
			Error:  "speaker verification requested without a target profile",
		},
	}, "get")

	require.NoError(t, err)
	require.Contains(t, buf.String(), "error=speaker verification requested without a target profile")
}

func TestRenderReportSurfacesFailureError(t *testing.T) {
	app := testutil.NewTestApp(t, http.NewServeMux())
	ctx, buf := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})

	err := renderReport(ctx, &experimentv1.GetExperimentReportResponse{
		Experiment: &experimentv1.Experiment{
			Id:     "exp-failed",
			Name:   "failed speaker run",
			Status: experimentv1.ExperimentStatus_EXPERIMENT_STATUS_FAILED,
			Error:  "speaker verification requested without a target profile",
		},
		Report: &evalv1.EvalReport{},
	})

	require.NoError(t, err)
	require.Contains(t, buf.String(), "Error: speaker verification requested without a target profile")
}

func TestExperimentJSONEmitsZeroValueMetrics(t *testing.T) {
	var out strings.Builder
	err := printExperimentProtoJSON(&out, &experimentv1.GetExperimentReportResponse{
		Experiment: &experimentv1.Experiment{
			Id:     "exp-zero",
			Status: experimentv1.ExperimentStatus_EXPERIMENT_STATUS_SUCCEEDED,
		},
		Report: &evalv1.EvalReport{
			QualityMeasured: true,
			LatencyMeasured: true,
			PerStrategy: []*evalv1.StrategyReport{{
				Strategy: "batch",
				Label:    "batch / clean",
				Safety:   &evalv1.SafetyGateReport{Passed: true},
			}},
		},
	})
	require.NoError(t, err)

	body := out.String()
	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(body), &envelope))
	reportEnvelope, ok := envelope["report"].(map[string]any)
	require.True(t, ok)
	rows, ok := reportEnvelope["per_strategy"].([]any)
	require.True(t, ok)
	require.Len(t, rows, 1)
	row, ok := rows[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(0), row["wer"])
	require.Equal(t, float64(0), row["finalization_latency_p95_ms"])
	require.Equal(t, float64(0), row["partial_revisions"])
	require.Equal(t, float64(0), row["wer_delta_vs_winner"])
	require.NotContains(t, body, `"started_at"`, "unpopulated message fields should not become null")
}

func TestStartHumanOutputShowsEstimatedSeconds(t *testing.T) {
	app := mountExperiment(t, &fakeExperimentSvc{
		startFn: func(req *experimentv1.StartExperimentRequest) (*experimentv1.StartExperimentResponse, error) {
			return &experimentv1.StartExperimentResponse{
				EstimatedSeconds: 125,
				Experiment: &experimentv1.Experiment{
					Id:     "exp-eta",
					Name:   req.GetName(),
					Status: experimentv1.ExperimentStatus_EXPERIMENT_STATUS_QUEUED,
					Recipe: req.GetRecipe(),
				},
			}, nil
		},
	})
	h := newHandlers(app)
	ctx, buf := cliapptest.NewCapturedRunContext(app, experimentStartSchema(), cliapptest.TestRunContextOptions{
		Flags: map[string]string{"name": "eta"},
	})

	require.NoError(t, h.start(ctx))
	require.Contains(t, buf.String(), "estimated_seconds=125 (~2m05s)")
}

func TestWaitJSONReturnsTerminalReportEnvelope(t *testing.T) {
	report := &evalv1.EvalReport{
		QualityMeasured: true,
		PerStrategy: []*evalv1.StrategyReport{{
			Strategy: "batch",
			Label:    "batch / clean",
		}},
	}
	app := mountExperiment(t, &fakeExperimentSvc{
		waitFn: func(req *experimentv1.WaitExperimentRequest) (*experimentv1.WaitExperimentResponse, error) {
			require.Equal(t, "exp-ready", req.GetId())
			return &experimentv1.WaitExperimentResponse{
				Experiment: &experimentv1.Experiment{
					Id:        "exp-ready",
					Status:    experimentv1.ExperimentStatus_EXPERIMENT_STATUS_SUCCEEDED,
					ResultRef: "experiment-blobs/report.json",
				},
				Runs: []*experimentv1.ExperimentRun{{Id: "run-1", Strategy: "batch"}},
			}, nil
		},
		reportFn: func(req *experimentv1.GetExperimentReportRequest) (*experimentv1.GetExperimentReportResponse, error) {
			require.Equal(t, "exp-ready", req.GetId())
			return &experimentv1.GetExperimentReportResponse{
				Experiment: &experimentv1.Experiment{
					Id:        "exp-ready",
					Status:    experimentv1.ExperimentStatus_EXPERIMENT_STATUS_SUCCEEDED,
					ResultRef: "experiment-blobs/report.json",
				},
				Report: report,
				Runs:   []*experimentv1.ExperimentRun{{Id: "run-1", Strategy: "batch"}},
			}, nil
		},
	})
	h := newHandlers(app)
	ctx, buf := cliapptest.NewCapturedRunContext(app,
		cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true}}},
		cliapptest.TestRunContextOptions{JSON: true, Positionals: map[string]string{"id": "exp-ready"}},
	)

	require.NoError(t, h.wait(ctx))
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &envelope))
	require.Contains(t, envelope, "experiment")
	require.Contains(t, envelope, "report")
	require.Contains(t, envelope, "runs")
	reportEnvelope, ok := envelope["report"].(map[string]any)
	require.True(t, ok)
	rows, ok := reportEnvelope["per_strategy"].([]any)
	require.True(t, ok)
	require.Len(t, rows, 1)
	row, ok := rows[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(0), row["wer"])
}

func TestWaitJSONReturnsStableEnvelopeWhenReportMissing(t *testing.T) {
	app := mountExperiment(t, &fakeExperimentSvc{
		waitFn: func(req *experimentv1.WaitExperimentRequest) (*experimentv1.WaitExperimentResponse, error) {
			return &experimentv1.WaitExperimentResponse{
				Experiment: &experimentv1.Experiment{
					Id:     "exp-failed",
					Status: experimentv1.ExperimentStatus_EXPERIMENT_STATUS_FAILED,
					Error:  "boom",
				},
				Runs: []*experimentv1.ExperimentRun{},
			}, nil
		},
	})
	h := newHandlers(app)
	ctx, buf := cliapptest.NewCapturedRunContext(app,
		cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true}}},
		cliapptest.TestRunContextOptions{JSON: true, Positionals: map[string]string{"id": "exp-failed"}},
	)

	require.NoError(t, h.wait(ctx))
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &envelope))
	// Stable shape: experiment + runs + an explicit null report, regardless of
	// whether the run produced one.
	require.Contains(t, envelope, "experiment")
	require.Contains(t, envelope, "runs")
	require.Contains(t, envelope, "report")
	require.Nil(t, envelope["report"], "report must be explicit null when absent")
}

func TestStartDryRunDoesNotSubmitAndEchoesRecipe(t *testing.T) {
	var gotDryRun bool
	app := mountExperiment(t, &fakeExperimentSvc{
		startFn: func(req *experimentv1.StartExperimentRequest) (*experimentv1.StartExperimentResponse, error) {
			gotDryRun = req.GetDryRun()
			// Mirror the server's dry-run contract: echo the recipe, no id.
			return &experimentv1.StartExperimentResponse{
				DryRun:           req.GetDryRun(),
				EstimatedSeconds: 42,
				Experiment: &experimentv1.Experiment{
					Name:   req.GetName(),
					Status: experimentv1.ExperimentStatus_EXPERIMENT_STATUS_UNSPECIFIED,
					Recipe: req.GetRecipe(),
				},
			}, nil
		},
	})
	h := newHandlers(app)
	ctx, buf := cliapptest.NewCapturedRunContext(app, experimentStartSchema(), cliapptest.TestRunContextOptions{
		JSON:      true,
		Flags:     map[string]string{"name": "preview", "snr-db": "-5", "noise-types": "white"},
		BoolFlags: map[string]bool{"dry-run": true},
	})

	require.NoError(t, h.start(ctx))
	require.True(t, gotDryRun, "handler must forward dry_run to the server")
	var resp map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &resp))
	require.Equal(t, true, resp["dry_run"])
	exp, ok := resp["experiment"].(map[string]any)
	require.True(t, ok)
	require.Empty(t, exp["id"], "dry-run preview has no persisted id")
}

func TestStartProviderCellDoesNotInjectLegacyStrategyDefaults(t *testing.T) {
	app := mountExperiment(t, &fakeExperimentSvc{
		startFn: func(req *experimentv1.StartExperimentRequest) (*experimentv1.StartExperimentResponse, error) {
			require.Len(t, req.GetRecipe().GetCells(), 1)
			require.Empty(t, req.GetRecipe().GetStrategies(), "provider-cell recipes must not carry unused legacy defaults")
			return &experimentv1.StartExperimentResponse{Experiment: &experimentv1.Experiment{Recipe: req.GetRecipe()}}, nil
		},
	})
	h := newHandlers(app)
	ctx, _ := cliapptest.NewCapturedRunContext(app, experimentStartSchema(), cliapptest.TestRunContextOptions{
		Flags: map[string]string{"recipe-json": `{"cells":[{"engineId":"kyutai","strategy":"passthrough","replayLane":"REPLAY_LANE_REALTIME"}]}`},
	})
	require.NoError(t, h.start(ctx))
}

func TestDeleteRequiresYes(t *testing.T) {
	app := mountExperiment(t, &fakeExperimentSvc{
		deleteFn: func(*experimentv1.DeleteExperimentRequest) (*experimentv1.DeleteExperimentResponse, error) {
			t.Fatal("DeleteExperiment should not be called without --yes")
			return nil, nil
		},
	})
	h := newHandlers(app)
	ctx, _ := cliapptest.NewCapturedRunContext(app,
		cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true}}, Flags: []cliapp.Flag{{Name: "yes", Bool: true}}},
		cliapptest.TestRunContextOptions{Positionals: map[string]string{"id": "exp-1"}},
	)

	err := h.delete(ctx)

	require.Error(t, err)
	require.Contains(t, err.Error(), "--yes")
}

func TestDeleteJSON(t *testing.T) {
	app := mountExperiment(t, &fakeExperimentSvc{
		deleteFn: func(req *experimentv1.DeleteExperimentRequest) (*experimentv1.DeleteExperimentResponse, error) {
			require.Equal(t, "exp-1", req.GetId())
			return &experimentv1.DeleteExperimentResponse{Id: req.GetId(), DeletedReport: true}, nil
		},
	})
	h := newHandlers(app)
	ctx, buf := cliapptest.NewCapturedRunContext(app,
		cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true}}, Flags: []cliapp.Flag{{Name: "yes", Bool: true}}},
		cliapptest.TestRunContextOptions{JSON: true, Positionals: map[string]string{"id": "exp-1"}, BoolFlags: map[string]bool{"yes": true}},
	)

	require.NoError(t, h.delete(ctx))
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &envelope))
	require.Equal(t, "exp-1", envelope["id"])
	require.Equal(t, true, envelope["deleted_report"])
}

func TestStartAppliesTuningFlagsOnlyToRelevantStrategies(t *testing.T) {
	var captured *experimentv1.StartExperimentRequest
	app := mountExperiment(t, &fakeExperimentSvc{
		startFn: func(req *experimentv1.StartExperimentRequest) (*experimentv1.StartExperimentResponse, error) {
			captured = req
			return &experimentv1.StartExperimentResponse{
				Experiment: &experimentv1.Experiment{
					Id:     "exp-tuned",
					Name:   req.GetName(),
					Status: experimentv1.ExperimentStatus_EXPERIMENT_STATUS_QUEUED,
					Recipe: req.GetRecipe(),
				},
			}, nil
		},
	})
	h := newHandlers(app)
	ctx, _ := cliapptest.NewCapturedRunContext(app, experimentStartSchema(), cliapptest.TestRunContextOptions{
		Flags: map[string]string{
			"name":                      "tuned",
			"strategies":                "batch,vad_segment,overlap_agree",
			"overlap-max-window-ms":     "12000",
			"overlap-max-stall-rejects": "0",
			"overlap-window-ms":         "1800",
			"overlap-commit-runs":       "3",
			"vad-silence-ms":            "900",
		},
	})

	require.NoError(t, h.start(ctx))
	require.NotNil(t, captured)
	byKind := map[string]*evalv1.EvalStrategy{}
	for _, strategy := range captured.GetRecipe().GetStrategies() {
		byKind[strategy.GetKind()] = strategy
	}
	require.Equal(t, int32(-1), byKind["batch"].GetOverlapMaxStallRejects())
	require.Zero(t, byKind["batch"].GetOverlapMaxWindowMs())
	require.Zero(t, byKind["batch"].GetOverlapWindowMs())
	require.Zero(t, byKind["batch"].GetOverlapCommitRuns())
	require.Zero(t, byKind["batch"].GetVadSilenceMs())

	require.Equal(t, int32(-1), byKind["vad_segment"].GetOverlapMaxStallRejects())
	require.Zero(t, byKind["vad_segment"].GetOverlapMaxWindowMs())
	require.Zero(t, byKind["vad_segment"].GetOverlapWindowMs())
	require.Zero(t, byKind["vad_segment"].GetOverlapCommitRuns())
	require.Equal(t, int32(900), byKind["vad_segment"].GetVadSilenceMs())

	require.Equal(t, int32(0), byKind["overlap_agree"].GetOverlapMaxStallRejects())
	require.Equal(t, int32(12000), byKind["overlap_agree"].GetOverlapMaxWindowMs())
	require.Equal(t, int32(1800), byKind["overlap_agree"].GetOverlapWindowMs())
	require.Equal(t, int32(3), byKind["overlap_agree"].GetOverlapCommitRuns())
	require.Zero(t, byKind["overlap_agree"].GetVadSilenceMs())
}

func TestStartRejectsNegativeTuningFlagsBeforeSubmit(t *testing.T) {
	app := mountExperiment(t, &fakeExperimentSvc{
		startFn: func(*experimentv1.StartExperimentRequest) (*experimentv1.StartExperimentResponse, error) {
			t.Fatal("StartExperiment should not be called for invalid local flags")
			return nil, nil
		},
	})
	h := newHandlers(app)
	ctx, _ := cliapptest.NewCapturedRunContext(app, experimentStartSchema(), cliapptest.TestRunContextOptions{
		Flags: map[string]string{
			"strategies":        "overlap_agree",
			"overlap-window-ms": "-1",
		},
	})

	err := h.start(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "--overlap-window-ms must be non-negative")
}
