package quality

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	qualityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/quality"
)

type perceptualFunc func(ctx context.Context, ref, out []byte) (float64, error)

func (f perceptualFunc) Distance(ctx context.Context, ref, out []byte) (float64, error) {
	return f(ctx, ref, out)
}

func testPNG(t *testing.T, w, h int, shade uint8) []byte {
	t.Helper()
	im := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			im.Set(x, y, color.NRGBA{R: uint8(x*7) + shade, G: uint8(y * 5), B: shade, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, im))
	return buf.Bytes()
}

func lpipsManifest(maximum float64) *qualityv1.EvaluationManifest {
	return &qualityv1.EvaluationManifest{SchemaVersion: 1, Operation: "upscale", Profile: "image", DatasetId: "lpips", CandidateModel: "c", CandidateRevision: "r", Judge: "disabled",
		Cases:   []*qualityv1.EvaluationCase{{Id: "pair", Reference: "ref", Output: "out"}},
		Budgets: []*qualityv1.MetricBudget{{Metric: MetricLPIPS, Maximum: number(maximum)}}}
}

func loaderFor(ref, out []byte) Loader {
	return func(_ context.Context, key string) ([]byte, error) {
		if key == "ref" {
			return ref, nil
		}
		return out, nil
	}
}

func TestLPIPSIsScoredOnlyWhenBudgetedAndGatesLowerIsBetter(t *testing.T) {
	ref, out := testPNG(t, 96, 80, 0), testPNG(t, 96, 80, 9)
	calls := 0
	e := &Evaluator{Perceptual: perceptualFunc(func(context.Context, []byte, []byte) (float64, error) { calls++; return .31, nil })}

	r, err := e.Evaluate(context.Background(), lpipsManifest(.4), loaderFor(ref, out), nil, nil)
	require.NoError(t, err)
	require.True(t, r.Passed, "%v", r.Violations)
	require.InDelta(t, .31, r.Cases[0].Metrics[MetricLPIPS], 1e-9)
	require.Equal(t, 1, calls)
	require.Contains(t, strings.Join(r.Limits, " "), "LPIPS")

	r, err = e.Evaluate(context.Background(), lpipsManifest(.2), loaderFor(ref, out), nil, nil)
	require.NoError(t, err)
	require.False(t, r.Passed)
	require.Equal(t, "above maximum", r.Violations[0].Reason)

	// A manifest that does not budget LPIPS never pays for it.
	m := lpipsManifest(.4)
	m.Budgets = []*qualityv1.MetricBudget{{Metric: "reference.ssim", Minimum: number(.1)}}
	calls = 0
	_, err = e.Evaluate(context.Background(), m, loaderFor(ref, out), nil, nil)
	require.NoError(t, err)
	require.Zero(t, calls)
}

func TestLPIPSFailsClosedWithoutScorerOrOnSizeMismatchOrScorerError(t *testing.T) {
	ref, out := testPNG(t, 96, 80, 0), testPNG(t, 96, 80, 9)
	r, err := (&Evaluator{}).Evaluate(context.Background(), lpipsManifest(.4), loaderFor(ref, out), nil, nil)
	require.NoError(t, err)
	require.False(t, r.Passed)
	require.Contains(t, r.Cases[0].Errors[0], "no perceptual scorer")

	called := false
	scorer := perceptualFunc(func(context.Context, []byte, []byte) (float64, error) { called = true; return 0, nil })
	r, err = (&Evaluator{Perceptual: scorer}).Evaluate(context.Background(), lpipsManifest(.4), loaderFor(ref, testPNG(t, 192, 160, 9)), nil, nil)
	require.NoError(t, err)
	require.False(t, r.Passed)
	require.False(t, called, "mismatched sizes must not be resized to fit")
	require.Equal(t, "required metric is missing", r.Violations[0].Reason)
	require.Contains(t, r.Cases[0].Warnings[len(r.Cases[0].Warnings)-1], "sizes differ")

	r, err = (&Evaluator{Perceptual: perceptualFunc(func(context.Context, []byte, []byte) (float64, error) { return 0, errors.New("sidecar down") })}).Evaluate(context.Background(), lpipsManifest(.4), loaderFor(ref, out), nil, nil)
	require.NoError(t, err)
	require.False(t, r.Complete)
	require.Contains(t, r.Cases[0].Errors[0], "sidecar down")
}

func TestSidecarLPIPSRefusesAMissingBackboneAndParsesTheSidecarContract(t *testing.T) {
	_, err := (&SidecarLPIPS{}).Distance(context.Background(), []byte("a"), []byte("b"))
	require.ErrorContains(t, err, "not configured")
	_, err = (&SidecarLPIPS{Python: "python3", Backbone: filepath.Join(t.TempDir(), "absent.pth")}).Distance(context.Background(), []byte("a"), []byte("b"))
	require.ErrorContains(t, err, "lpips-alexnet")

	dir := t.TempDir()
	backbone := filepath.Join(dir, "alex.pth")
	require.NoError(t, os.WriteFile(backbone, []byte("x"), 0o600))
	fake := filepath.Join(dir, "python")
	require.NoError(t, os.WriteFile(fake, []byte("#!/bin/sh\necho '{\"lpips\": 0.25, \"net\": \"alex\"}'\n"), 0o700))
	d, err := (&SidecarLPIPS{Python: fake, Backbone: backbone}).Distance(context.Background(), []byte("a"), []byte("b"))
	require.NoError(t, err)
	require.InDelta(t, .25, d, 1e-9)
	require.NoError(t, os.WriteFile(fake, []byte("#!/bin/sh\necho '{\"lpips\": 0.25, \"net\": \"vgg\"}'\n"), 0o700))
	_, err = (&SidecarLPIPS{Python: fake, Backbone: backbone}).Distance(context.Background(), []byte("a"), []byte("b"))
	require.ErrorContains(t, err, "unusable")
}
