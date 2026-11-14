package quality

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	qualityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/quality"
	"google.golang.org/protobuf/proto"
)

type judgeFunc func(context.Context, JudgeRequest) (*qualityv1.JudgeResult, error)

func (f judgeFunc) Score(c context.Context, r JudgeRequest) (*qualityv1.JudgeResult, error) {
	return f(c, r)
}
func number(v float64) *float64 { return &v }
func qualifiedJudge(model string, fidelity float64) *qualityv1.JudgeResult {
	return &qualityv1.JudgeResult{Provider: "ollama", Model: model, Fidelity: fidelity, Instruction: 1, Artifacts: 1, Rationale: "Visible kettle matches reference.", Validated: true, RubricSha256: strings.Repeat("a", 64), RouteEventId: "route-event", CapacityClaimId: "capacity-claim", CapacityOwnership: "resource:ollama"}
}

func manifest() *qualityv1.EvaluationManifest {
	return &qualityv1.EvaluationManifest{SchemaVersion: 1, Operation: "caption", Profile: "text", DatasetId: "owned-products-v1", CandidateModel: "candidate", CandidateRevision: "sha256:weights-a", Judge: "disabled", Cases: []*qualityv1.EvaluationCase{{Id: "kettle", ReferenceText: "A blue kettle on a wooden table.", OutputText: "A blue kettle on a wooden table.", Instruction: "Describe the visible object."}}, Budgets: []*qualityv1.MetricBudget{{Metric: "text.similarity", Minimum: number(.95)}}}
}

func score(t *testing.T, e *Evaluator, m *qualityv1.EvaluationManifest, base *qualityv1.EvaluationReport) *qualityv1.EvaluationReport {
	t.Helper()
	r, err := e.Evaluate(context.Background(), m, nil, base, nil)
	require.NoError(t, err)
	return r
}

func TestQualityGateRejectsMissingAndIncorrectStructuredFacts(t *testing.T) {
	m := manifest()
	m.Profile = "json"
	m.Operation = "object_detection"
	m.Cases[0].ReferenceText = `{"objects":[{"label":"kettle","count":1}],"complete":true}`
	m.Cases[0].OutputText = `{"complete":true,"objects":[{"count":1,"label":"kettle"}]}`
	m.Budgets = []*qualityv1.MetricBudget{{Metric: "json.field_accuracy", Minimum: number(1)}}
	good := score(t, &Evaluator{}, m, nil)
	require.True(t, good.Passed)
	require.Equal(t, 1., good.Means["json.exact"])
	m.Cases[0].OutputText = `{"objects":[{"label":"car","count":2}]}`
	bad := score(t, &Evaluator{}, m, nil)
	require.False(t, bad.Passed)
	require.True(t, bad.Complete)
	require.Equal(t, 0., bad.Means["json.field_accuracy"])
	m.Cases[0].OutputText = `{"objects":`
	malformed := score(t, &Evaluator{}, m, nil)
	require.False(t, malformed.Passed)
	require.False(t, malformed.Complete)
	require.Len(t, malformed.Cases, 1)
}

func TestBaselinePairingUsesInputsAndConfigurationNotCaseOrderOrModel(t *testing.T) {
	m := manifest()
	m.Cases = append(m.Cases, &qualityv1.EvaluationCase{Id: "vase", ReferenceText: "A green vase.", OutputText: "A green vase."})
	base := score(t, &Evaluator{}, m, nil)
	m.CandidateModel = "candidate-b"
	m.CandidateRevision = "sha256:weights-b"
	m.Cases[0], m.Cases[1] = m.Cases[1], m.Cases[0]
	m.Budgets = append(m.Budgets, &qualityv1.MetricBudget{Metric: "text.similarity", MaxRegression: number(0)})
	r := score(t, &Evaluator{}, m, base)
	require.True(t, r.Passed)
	require.Equal(t, base.SuiteSha256, r.SuiteSha256)
	require.False(t, r.PromotionEligible, "a metrics-only report must not qualify a model promotion")
	m.Cases[0].ReferenceText = "A red vase."
	r = score(t, &Evaluator{}, m, base)
	require.False(t, r.Passed)
	require.False(t, r.Complete)
	require.NotEqual(t, base.SuiteSha256, r.SuiteSha256)
}

func TestBudgetRegressionAndMissingEvidenceFailClosed(t *testing.T) {
	m := manifest()
	base := score(t, &Evaluator{}, m, nil)
	m.Budgets = []*qualityv1.MetricBudget{{Metric: "text.similarity", Scope: "mean", MaxRegression: number(.01)}}
	r := score(t, &Evaluator{}, m, nil)
	require.False(t, r.Passed)
	require.Contains(t, r.Violations[0].Reason, "baseline")
	m.Cases[0].OutputText = "A red car."
	r = score(t, &Evaluator{}, m, base)
	require.False(t, r.Passed)
	require.Contains(t, r.Violations[0].Reason, "regression")
	m.Profile = "json"
	m.Cases[0].ReferenceText = `{"count":1}`
	m.Cases[0].OutputText = `invalid`
	m.Budgets = []*qualityv1.MetricBudget{{Metric: "json.exact", Scope: "mean", Minimum: number(1)}}
	r = score(t, &Evaluator{}, m, nil)
	require.False(t, r.Passed)
	require.Contains(t, r.Violations[len(r.Violations)-1].Reason, "missing")
}

func TestReportIntegrityRejectsChangedScore(t *testing.T) {
	r := score(t, &Evaluator{}, manifest(), nil)
	require.NoError(t, VerifyReport(r))
	r.Cases[0].Metrics["text.similarity"] = 0
	require.ErrorContains(t, VerifyReport(r), "SHA-256 mismatch")
	_, err := (&Evaluator{}).Evaluate(context.Background(), manifest(), nil, r, nil)
	require.ErrorContains(t, err, "baseline")
}

func TestJudgeFailureAndWrongIdentityCannotPass(t *testing.T) {
	for _, name := range []string{"missing", "failed", "wrong-model", "missing-rationale", "missing-route-evidence", "out-of-range"} {
		t.Run(name, func(t *testing.T) {
			m := manifest()
			m.Judge = "required"
			m.JudgeModel = "qwen3-vl:4b"
			e := &Evaluator{}
			if name != "missing" {
				e.Judge = judgeFunc(func(context.Context, JudgeRequest) (*qualityv1.JudgeResult, error) {
					if name == "failed" {
						return nil, errors.New("capacity refused")
					}
					j := qualifiedJudge(m.JudgeModel, 1)
					if name == "wrong-model" {
						j.Model = "unreviewed-model"
					}
					if name == "missing-rationale" {
						j.Rationale = ""
					}
					if name == "missing-route-evidence" {
						j.CapacityClaimId = ""
					}
					if name == "out-of-range" {
						j.Fidelity = 1.2
					}
					return j, nil
				})
			}
			r := score(t, e, m, nil)
			require.False(t, r.Passed)
			require.False(t, r.Complete)
			require.False(t, r.PromotionEligible)
		})
	}
}

func TestPromotionEvidenceRequiresPassingPairedLocalJudge(t *testing.T) {
	m := manifest()
	m.Judge = "required"
	m.JudgeModel = "qwen3-vl:4b"
	fidelity := .8
	e := &Evaluator{Judge: judgeFunc(func(_ context.Context, r JudgeRequest) (*qualityv1.JudgeResult, error) {
		return qualifiedJudge(r.ExpectedModel, fidelity), nil
	})}
	base := score(t, e, m, nil)
	require.True(t, base.Passed)
	require.False(t, base.PromotionEligible)
	fidelity = 1
	m.Budgets = append(m.Budgets, &qualityv1.MetricBudget{Metric: "judge.fidelity", Scope: "mean", MinImprovement: number(.1)})
	m.CandidateRevision = "sha256:weights-b"
	r := score(t, e, m, base)
	require.True(t, r.PromotionEligible)
	m.Rubric = "Use a different criterion"
	r = score(t, e, m, base)
	require.False(t, r.PromotionEligible)
	require.False(t, r.Complete)
}

func TestAlphaMetricsExposeMissingThinStructuresAndBackgroundLeak(t *testing.T) {
	ref := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	candidate := image.NewNRGBA(ref.Bounds())
	for y := 4; y < 28; y++ {
		for x := 10; x < 22; x++ {
			ref.SetNRGBA(x, y, color.NRGBA{80, 120, 200, 255})
			candidate.SetNRGBA(x, y, color.NRGBA{80, 120, 200, 255})
		}
	}
	for x := 2; x < 10; x++ {
		ref.SetNRGBA(x, 12, color.NRGBA{80, 120, 200, 255})
	}
	encode := func(im image.Image) []byte {
		var b bytes.Buffer
		require.NoError(t, png.Encode(&b, im))
		return b.Bytes()
	}
	m, _, err := imageMetrics("matte", encode(ref), encode(ref))
	require.NoError(t, err)
	require.Equal(t, 1., m["alpha.iou"])
	require.Equal(t, 1., m["alpha.boundary_f1"])
	require.Zero(t, m["alpha.mae"])
	m, _, err = imageMetrics("matte", encode(ref), encode(candidate))
	require.NoError(t, err)
	require.Less(t, m["alpha.recall"], 1.)
	require.Equal(t, 1., m["alpha.precision"])
	require.Less(t, m["alpha.boundary_f1"], 1.)
	candidate.SetNRGBA(0, 0, color.NRGBA{255, 255, 255, 255})
	m, _, err = imageMetrics("matte", encode(ref), encode(candidate))
	require.NoError(t, err)
	require.Less(t, m["alpha.precision"], 1.)
	_, _, err = imageMetrics("matte", encode(ref), encode(image.NewNRGBA(image.Rect(0, 0, 16, 16))))
	require.ErrorContains(t, err, "dimensions differ")
}

func TestImageHashAssertionAndCancellation(t *testing.T) {
	m := manifest()
	m.Cases[0].Input = "owned-input"
	m.Cases[0].InputSha256 = "wrong"
	r, err := (&Evaluator{}).Evaluate(context.Background(), m, func(context.Context, string) ([]byte, error) { return []byte("input"), nil }, nil, nil)
	require.NoError(t, err)
	require.False(t, r.Passed)
	require.Contains(t, r.Cases[0].Errors[0], "SHA-256")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = (&Evaluator{}).Evaluate(ctx, m, nil, nil, nil)
	require.ErrorIs(t, err, context.Canceled)
}

func TestTextToImageUsesJudgeWithoutMisleadingPixelReferenceMetrics(t *testing.T) {
	var output bytes.Buffer
	require.NoError(t, png.Encode(&output, image.NewNRGBA(image.Rect(0, 0, 16, 16))))
	m := &qualityv1.EvaluationManifest{
		SchemaVersion: 1, Operation: "text_to_image", Profile: "image", DatasetId: "generated-prompts-v1",
		CandidateModel: "sd-1.5", CandidateRevision: "sha256:sd15-host-weights", Judge: "required", JudgeModel: "qwen3-vl:4b",
		Rubric:  "No canonical reference exists; judge prompt adherence, composition and visible artifacts.",
		Cases:   []*qualityv1.EvaluationCase{{Id: "mountain-lake", Output: "generated.png", Instruction: "a serene mountain lake at golden hour"}},
		Budgets: []*qualityv1.MetricBudget{{Metric: "judge.instruction", Minimum: number(.7)}, {Metric: "judge.artifacts", Minimum: number(.7)}},
	}
	var got JudgeRequest
	e := &Evaluator{Judge: judgeFunc(func(_ context.Context, req JudgeRequest) (*qualityv1.JudgeResult, error) {
		got = req
		return qualifiedJudge("qwen3-vl:4b", .9), nil
	})}
	r, err := e.Evaluate(context.Background(), m, func(_ context.Context, key string) ([]byte, error) {
		require.Equal(t, "generated.png", key)
		return output.Bytes(), nil
	}, nil, nil)
	require.NoError(t, err)
	require.True(t, r.Passed)
	require.Equal(t, .9, r.Means["judge.fidelity"])
	require.NotContains(t, r.Means, "reference.ssim")
	require.Contains(t, r.Cases[0].Warnings, "reference-image metrics omitted: text-to-image has no canonical pixel reference")
	require.Empty(t, got.Reference)
	require.Equal(t, "a serene mountain lake at golden hour", got.Instruction)
	require.Len(t, got.Output, output.Len())

	m.Judge = "disabled"
	require.ErrorContains(t, Validate(m), "requires the local judge")
}

func TestInstructionEditUsesJudgeAgainstSourceWithoutPixelReference(t *testing.T) {
	var source, output bytes.Buffer
	require.NoError(t, png.Encode(&source, image.NewNRGBA(image.Rect(0, 0, 16, 16))))
	require.NoError(t, png.Encode(&output, image.NewNRGBA(image.Rect(0, 0, 16, 16))))
	m := &qualityv1.EvaluationManifest{
		SchemaVersion: 1, Operation: "edit_instruct", Profile: "image", DatasetId: "instruction-edit-source-v1",
		CandidateModel: "instruct-pix2pix", CandidateRevision: "sha256:instruct-pix2pix-host-weights", Judge: "required", JudgeModel: "qwen3-vl:4b",
		Rubric:  "Change the blue star accents to warm coral while preserving the eagle linework and dark background.",
		Cases:   []*qualityv1.EvaluationCase{{Id: "aquila-stars", Input: "aquila-source.png", Output: "aquila-edited.png", Instruction: "Change the bright blue stars to warm coral."}},
		Budgets: []*qualityv1.MetricBudget{{Metric: "judge.fidelity", Minimum: number(.7)}, {Metric: "judge.instruction", Minimum: number(.75)}, {Metric: "judge.artifacts", Minimum: number(.7)}},
	}
	var got JudgeRequest
	e := &Evaluator{Judge: judgeFunc(func(_ context.Context, req JudgeRequest) (*qualityv1.JudgeResult, error) {
		got = req
		return qualifiedJudge("qwen3-vl:4b", .9), nil
	})}
	r, err := e.Evaluate(context.Background(), m, func(_ context.Context, key string) ([]byte, error) {
		switch key {
		case "aquila-source.png":
			return source.Bytes(), nil
		case "aquila-edited.png":
			return output.Bytes(), nil
		default:
			return nil, fmt.Errorf("unexpected asset %q", key)
		}
	}, nil, nil)
	require.NoError(t, err)
	require.True(t, r.Passed)
	require.NotContains(t, r.Means, "reference.ssim")
	require.Contains(t, r.Cases[0].Warnings, "reference-image metrics omitted: no canonical pixel reference; the local judge uses the supplied source image and instruction")
	require.Len(t, got.Input, source.Len())
	require.Empty(t, got.Reference)
	require.Len(t, got.Output, output.Len())
	require.Equal(t, "Change the bright blue stars to warm coral.", got.Instruction)

	m.Judge = "disabled"
	require.ErrorContains(t, Validate(m), "requires the local judge")
	m.Judge = "required"
	m.Cases[0].Input = ""
	require.ErrorContains(t, Validate(m), "source image is required")
}

func TestManifestRejectsUnscorableOrVacuousGate(t *testing.T) {
	for _, mutate := range []func(*qualityv1.EvaluationManifest){func(m *qualityv1.EvaluationManifest) { m.Cases = nil }, func(m *qualityv1.EvaluationManifest) {
		m.Cases = append(m.Cases, proto.Clone(m.Cases[0]).(*qualityv1.EvaluationCase))
	}, func(m *qualityv1.EvaluationManifest) { m.Budgets = nil }, func(m *qualityv1.EvaluationManifest) { m.Budgets[0].Metric = "judge.fidelity" }, func(m *qualityv1.EvaluationManifest) { m.CandidateRevision = "" }, func(m *qualityv1.EvaluationManifest) { m.Cases[0].ReferenceText = "" }} {
		m := manifest()
		mutate(m)
		require.Error(t, Validate(m))
	}
}

var benchmarkReport *qualityv1.EvaluationReport

func BenchmarkEvaluateText64(b *testing.B) {
	m := manifest()
	m.Cases = make([]*qualityv1.EvaluationCase, 64)
	for i := range m.Cases {
		m.Cases[i] = &qualityv1.EvaluationCase{Id: fmt.Sprintf("case-%02d", i), ReferenceText: "A blue kettle on a wooden table.", OutputText: "A blue kettle on a wooden table."}
	}
	e := &Evaluator{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r, err := e.Evaluate(context.Background(), m, nil, nil, nil)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkReport = r
	}
}
