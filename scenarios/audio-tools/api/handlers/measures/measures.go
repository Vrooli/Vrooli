package measures

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"connectrpc.com/connect"
	gomeasures "github.com/vrooli/measures-go"
	sharedmeasuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
)

const (
	byokMeasure             = "byok_credentials.byok-credentials-created"
	corpusMeasure           = "corpus_clips.corpus-clips-created"
	runsMeasure             = "experiment_runs.experiment-runs-created"
	experimentsMeasure      = "experiments.experiments-created"
	speakerMeasure          = "speaker_profiles.speaker-profiles-created"
	wakewordMeasure         = "wakeword_templates.wakeword-templates-created"
	duplexAccountingMeasure = "duplex_accounting_operations.duplex-accounting-operations-created"
)

type countQuery func(context.Context, time.Time, time.Time) (int64, error)

func declaration(name, domain, intent, unit, serviceMethod string, questions ...string) gomeasures.MeasureDeclaration {
	return gomeasures.MeasureDeclaration{
		Name: name, Domain: domain, Intent: intent, Questions: questions,
		Params: map[string]gomeasures.Param{
			"window": {Name: "window", Type: gomeasures.ParamTypeTimeWindow, Default: string(gomeasures.TokenThisWeek)},
		},
		Result: gomeasures.Result{Kind: gomeasures.ResultScalar, ValueField: "count", Unit: unit, SummaryTemplate: "{count} " + unit + " created ({window})"},
		Effect: gomeasures.EffectRead, RunEligible: true,
		Service: "MeasuresService", Method: serviceMethod,
	}
}

func safeCounter(dependency any, name string, count countQuery) countQuery {
	return func(ctx context.Context, from, to time.Time) (int64, error) {
		if dependency == nil {
			return 0, fmt.Errorf("%s counter is not configured", name)
		}
		value := reflect.ValueOf(dependency)
		switch value.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
			if value.IsNil() {
				return 0, fmt.Errorf("%s counter is not configured", name)
			}
		}
		return count(ctx, from, to)
	}
}

func registerMeasures(reg *gomeasures.Registry, d Deps) error {
	items := []struct {
		decl  gomeasures.MeasureDeclaration
		count countQuery
	}{
		{declaration(byokMeasure, "byok_credentials", "Count currently retained BYOK credential capabilities created in a time window.", "credential capabilities", "CountBYOKCredentials",
			"how many BYOK credential capabilities were added this week", "retained API credentials created last month"), safeCounter(d.BYOK, "BYOK", func(ctx context.Context, from, to time.Time) (int64, error) {
			return d.BYOK.CountCreatedBetween(ctx, from, to)
		})},
		{declaration(corpusMeasure, "corpus_clips", "Count corpus clip metadata rows created in a time window.", "corpus clips", "CountCorpusClips",
			"how many corpus clips were added this week", "clips added to the evaluation corpus last month"), safeCounter(d.Corpus, "corpus", func(ctx context.Context, from, to time.Time) (int64, error) {
			return d.Corpus.CountClipsCreatedInWindow(ctx, from, to)
		})},
		{declaration(runsMeasure, "experiment_runs", "Count persisted experiment run rows created in a time window.", "experiment runs", "CountExperimentRuns",
			"how many experiment runs were recorded this week", "experiment runs created in the last 30 days"), safeCounter(d.Experiments, "experiment", func(ctx context.Context, from, to time.Time) (int64, error) {
			counts, err := d.Experiments.CountCreatedInRange(ctx, from, to)
			return counts.Runs, err
		})},
		{declaration(experimentsMeasure, "experiments", "Count persisted experiments created in a time window.", "experiments", "CountExperiments",
			"how many experiments were created this week", "experiments created last month"), safeCounter(d.Experiments, "experiment", func(ctx context.Context, from, to time.Time) (int64, error) {
			counts, err := d.Experiments.CountCreatedInRange(ctx, from, to)
			return counts.Experiments, err
		})},
		{declaration(speakerMeasure, "speaker_profiles", "Count speaker profiles created in a time window; profile embeddings and identities are not returned.", "speaker profiles", "CountSpeakerProfiles",
			"how many speaker profiles were enrolled this week", "speaker profiles created last month"), safeCounter(d.Speakers, "speaker", func(ctx context.Context, from, to time.Time) (int64, error) {
			return d.Speakers.CountCreatedBetween(ctx, from, to)
		})},
		{declaration(wakewordMeasure, "wakeword_templates", "Count wakeword templates created in a time window; phrases and embeddings are not returned.", "wakeword templates", "CountWakewordTemplates",
			"how many wakeword templates were created this week", "wakeword templates added last month"), safeCounter(d.Wakewords, "wakeword", func(ctx context.Context, from, to time.Time) (int64, error) {
			return d.Wakewords.CountCreatedBetween(ctx, from, to)
		})},
		{declaration(duplexAccountingMeasure, "duplex_accounting_operations", "Count persisted duplex accounting operation metadata rows created in a time window; no operation details or identifiers are returned.", "duplex accounting operations", "CountDuplexAccountingOperations",
			"how many duplex accounting operations were created this week", "duplex accounting operations created last month"), safeCounter(d.DuplexAccounting, "duplex accounting", func(ctx context.Context, from, to time.Time) (int64, error) {
			return d.DuplexAccounting.CountCreatedBetween(ctx, from, to)
		})},
	}
	for _, item := range items {
		item := item
		if err := reg.Register(item.decl, func(ctx context.Context, req gomeasures.MeasureRequest) (gomeasures.MeasureResult, error) {
			rng, err := resolveToken(req.Params["window"], d.Clock.Now())
			if err != nil {
				return gomeasures.MeasureResult{}, err
			}
			count, err := item.count(ctx, rng.From, rng.To)
			if err != nil {
				return gomeasures.MeasureResult{}, err
			}
			return gomeasures.MeasureResult{
				Value:      strconv.FormatInt(count, 10),
				Provenance: gomeasures.Provenance{ExecutedQuery: fmt.Sprintf("created_at in [%s,%s)", rng.From.UTC().Format(time.RFC3339Nano), rng.To.UTC().Format(time.RFC3339Nano))},
			}, nil
		}); err != nil {
			return fmt.Errorf("register measure %q: %w", item.decl.Name, err)
		}
	}
	return nil
}

func resolveToken(token string, now time.Time) (gomeasures.Range, error) {
	window := gomeasures.TimeWindowToken(token)
	if window == "" {
		window = gomeasures.TokenThisWeek
	}
	return gomeasures.ResolveToken(window, now.UTC(), time.UTC)
}

func resolveProtoWindow(window *sharedmeasuresv1.TimeWindow, now time.Time) (gomeasures.Range, error) {
	if window == nil || window.GetWindow() == nil {
		return resolveToken("", now)
	}
	return gomeasures.ResolveTimeWindow(window, now.UTC(), time.UTC)
}

func countInWindow(ctx context.Context, window *sharedmeasuresv1.TimeWindow, now time.Time, count countQuery) (int64, error) {
	rng, err := resolveProtoWindow(window, now)
	if err != nil {
		return 0, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return count(ctx, rng.From, rng.To)
}
