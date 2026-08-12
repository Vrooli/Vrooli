//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"backdrop-studio/integration"
	"backdrop-studio/internal/perceptual"

	"github.com/stretchr/testify/require"
)

// corpusPath is the recorded perceptual state of the catalog. It is a
// regression corpus, not a golden: it records where every style *sits*
// relative to its bar, so a change that quietly moves a style toward the floor
// is visible before it falls through.
const corpusPath = "../../docs/evidence/perceptual/corpus.json"

// corpusTolerance is how far a metric may move before the lane calls it a
// regression. Renders are deterministic, so any real movement is a change in
// the code or the catalog; the band exists for the arithmetic of a resolved
// screen landing on a different whole pixel, not for noise.
const corpusTolerance = 0.05

type corpusEntry struct {
	Style   string             `json:"style"`
	Surface string             `json:"surface"`
	Seed    int64              `json:"seed"`
	Metrics map[string]float64 `json:"metrics"`
	Bar     map[string]float64 `json:"bar"`
}

type corpusFile struct {
	Note    string        `json:"note"`
	Entries []corpusEntry `json:"entries"`
}

// TestPerceptualCorpus is both the producer and the regression check.
//
// With BACKDROP_STUDIO_WRITE_EVIDENCE set it records the catalog's current
// perceptual state. Without it — the ordinary lane run — it compares against
// the recorded state and fails on any metric that moved outside the band,
// naming the style, the metric, and the direction.
//
// The direction matters more than the magnitude. A style drifting *down* on
// subject survival is on its way to becoming the next `engraved-colonnade`,
// and the point of a corpus is to see that coming rather than to discover it
// in a released asset.
func TestPerceptualCorpus(t *testing.T) {
	env, caps := newEnvironment(t)
	ctx := context.Background()

	styles, err := env.Styles(ctx)
	require.NoError(t, err)
	surfaces, err := env.Surfaces(ctx)
	require.NoError(t, err)
	sort.Slice(styles, func(i, j int) bool { return styles[i].ID < styles[j].ID })

	const seed = 7
	observed := make([]corpusEntry, 0, len(styles))
	skipped := 0

	for _, style := range styles {
		if style.ModelBacked() && !caps.GenerationOK {
			t.Logf("SKIP(no-image-model) %s", style.ID)
			skipped++
			continue
		}
		permitted := integration.PermittedSurfaces(style, surfaces)
		if len(permitted) == 0 {
			t.Logf("SKIP(no-permitted-surface) %s", style.ID)
			skipped++
			continue
		}
		surface := permitted[0]
		job, submitErr := env.Submit(ctx, integration.SubmitOptions{
			StyleID: style.ID, Placement: style.Placements[0], Seed: seed, SurfaceID: surface.ID,
		})
		if submitErr != nil {
			if integration.IsGPUCapacityFailure(submitErr) {
				t.Logf("SKIP(gpu-capacity) %s: %v", style.ID, submitErr)
				skipped++
				continue
			}
			// A style the gate refuses is a real finding, not a lane failure to
			// route around — the message names the metric and the value.
			require.NoErrorf(t, submitErr, "%s did not render", style.ID)
		}
		require.NotEmptyf(t, job.Candidates, "%s produced no candidate", style.ID)

		var verdict perceptual.Verdict
		require.NoErrorf(t, json.Unmarshal([]byte(job.Candidates[0].QualityJSON), &verdict),
			"%s recorded no perceptual verdict; the gate did not run", style.ID)
		require.Truef(t, verdict.Passed, "%s recorded a failing verdict on a candidate that was returned", style.ID)

		entry := corpusEntry{
			Style: style.ID, Surface: surface.ID, Seed: seed,
			Metrics: map[string]float64{}, Bar: map[string]float64{},
		}
		for _, m := range verdict.Metrics {
			entry.Metrics[m.Name] = round4(m.Value)
			switch {
			case m.Min > 0:
				entry.Bar[m.Name] = round4(m.Min)
			case m.Max > 0:
				entry.Bar[m.Name] = round4(m.Max)
			}
		}
		observed = append(observed, entry)
		t.Logf("%-24s %s %v", style.ID, surface.ID, entry.Metrics)
	}

	require.NotEmpty(t, observed, "the corpus needs at least one measured style to mean anything")
	t.Logf("measured %d styles, skipped %d", len(observed), skipped)

	if os.Getenv(writeEvidenceEnv) != "" {
		writeCorpus(t, observed, skipped)
		return
	}
	compareCorpus(t, observed)
}

func writeCorpus(t *testing.T, entries []corpusEntry, skipped int) {
	t.Helper()
	file := corpusFile{
		Note: fmt.Sprintf(
			"Perceptual state of the seeded catalog at seed 7, one entry per style at its first permitted surface. "+
				"`metrics` is what each style scored; `bar` is the threshold it had to clear. Regenerate with "+
				"`make integration-evidence`. The lane fails when a metric moves more than %.2f from its recorded "+
				"value — deterministic renders do not drift, so movement is a change in the code or the catalog. "+
				"%d style(s) were unmeasurable on the recording host and carry no entry.",
			corpusTolerance, skipped),
		Entries: entries,
	}
	raw, err := json.MarshalIndent(file, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(corpusPath), 0o755))
	require.NoError(t, os.WriteFile(corpusPath, append(raw, '\n'), 0o644))
	t.Logf("wrote %s with %d entries", corpusPath, len(entries))
}

func compareCorpus(t *testing.T, observed []corpusEntry) {
	t.Helper()
	raw, err := os.ReadFile(corpusPath)
	require.NoErrorf(t, err, "the corpus is missing; record it with `make integration-evidence`")
	var recorded corpusFile
	require.NoError(t, json.Unmarshal(raw, &recorded))

	byStyle := make(map[string]corpusEntry, len(recorded.Entries))
	for _, e := range recorded.Entries {
		byStyle[e.Style] = e
	}
	for _, got := range observed {
		want, known := byStyle[got.Style]
		if !known {
			t.Logf("NEW %s is not in the corpus yet; re-record with `make integration-evidence`", got.Style)
			continue
		}
		for name, value := range got.Metrics {
			prior, had := want.Metrics[name]
			if !had {
				continue
			}
			direction := "up"
			if value < prior {
				direction = "DOWN toward its floor"
			}
			require.InDeltaf(t, prior, value, corpusTolerance,
				"%s: %s moved %s, %.4f -> %.4f (bar %.4f). A deterministic render does not drift; something changed.",
				got.Style, name, direction, prior, value, want.Bar[name])
		}
	}
}

func round4(v float64) float64 {
	return float64(int(v*10000+0.5)) / 10000
}
