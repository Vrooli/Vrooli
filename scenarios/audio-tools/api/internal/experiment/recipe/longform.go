//go:build audio_lab

// Package recipe builds deterministic experiment inputs from corpus clips.
package recipe

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	inteval "audio-tools/internal/eval"
)

const (
	CanonicalSampleRate = 16000
	DefaultGapMs        = 5000
	defaultSeed         = int64(1)
)

// Clip is the subset of corpus/eval clip data needed to assemble an input.
type Clip struct {
	ID               string
	PCM              []byte
	SampleRate       int
	Reference        string
	Format           string
	TimedReference   []inteval.TimedWord
	TimingProvenance *inteval.TimingReferenceProvenance
}

// Spec describes one seeded long-form concatenation.
type Spec struct {
	Seed                  int64
	TargetDurationSeconds int
	GapMs                 int
}

// Realization records the exact generated input recipe.
type Realization struct {
	ClipIDs    []string
	Reference  string
	DurationMs int64
}

// Build concatenates corpus clips with zero-byte silence gaps. The same Spec
// and same corpus snapshot produce byte-identical PCM and reference text.
func Build(spec Spec, clips []Clip) (inteval.Clip, Realization, error) {
	if len(clips) == 0 {
		return inteval.Clip{}, Realization{}, fmt.Errorf("experiment recipe: no clips to concatenate")
	}
	normalized, err := normalizeClips(clips)
	if err != nil {
		return inteval.Clip{}, Realization{}, err
	}
	seed := spec.Seed
	if seed == 0 {
		seed = defaultSeed
	}
	gapMs := spec.GapMs
	if gapMs <= 0 {
		gapMs = DefaultGapMs
	}

	rng := rand.New(rand.NewSource(seed)) // #nosec G404 -- seeded math/rand selects deterministic long-form corpus samples.
	var selected []Clip
	target := time.Duration(spec.TargetDurationSeconds) * time.Second
	total := time.Duration(0)
	for len(selected) == 0 || (target > 0 && total < target) {
		perm := rng.Perm(len(normalized))
		for _, idx := range perm {
			clip := normalized[idx]
			if len(selected) > 0 {
				total += time.Duration(gapMs) * time.Millisecond
			}
			total += duration(clip)
			selected = append(selected, clip)
			if target > 0 && total >= target {
				break
			}
		}
		if target <= 0 {
			break
		}
	}

	gap := silence(gapMs)
	var pcm []byte
	refs := make([]string, 0, len(selected))
	var timedReference []inteval.TimedWord
	allReviewed := true
	var provenanceSources, provenanceReviewers, provenanceMethods, provenanceTools, provenanceModels, provenanceDates []string
	ids := make([]string, 0, len(selected))
	var offsetSeconds float64
	for i, clip := range selected {
		if i > 0 {
			pcm = append(pcm, gap...)
			offsetSeconds += float64(gapMs) / 1000
		}
		pcm = append(pcm, clip.PCM...)
		if clip.TimingProvenance == nil || !clip.TimingProvenance.ReviewReady() || len(clip.TimedReference) == 0 {
			allReviewed = false
		} else {
			p := clip.TimingProvenance
			provenanceSources = append(provenanceSources, p.SourceReference)
			provenanceReviewers = append(provenanceReviewers, p.Reviewer)
			provenanceMethods = append(provenanceMethods, p.ReviewMethod)
			provenanceTools = append(provenanceTools, p.AlignmentTool)
			provenanceModels = append(provenanceModels, p.AlignmentModel)
			provenanceDates = append(provenanceDates, p.ReviewedAt)
			for _, word := range clip.TimedReference {
				timedReference = append(timedReference, inteval.TimedWord{Word: word.Word, Start: word.Start + offsetSeconds, End: word.End + offsetSeconds})
			}
		}
		offsetSeconds += float64(len(clip.PCM)) / float64(CanonicalSampleRate*2)
		ids = append(ids, clip.ID)
		if ref := strings.TrimSpace(clip.Reference); ref != "" {
			refs = append(refs, ref)
		}
	}
	reference := strings.Join(refs, " ")
	out := inteval.Clip{
		ID:         "long-form",
		PCM:        pcm,
		SampleRate: CanonicalSampleRate,
		Reference:  reference,
		Format:     "pcm_s16le",
	}
	if allReviewed && len(timedReference) > 0 {
		out.TimedReference = timedReference
		out.TimingProvenance = &inteval.TimingReferenceProvenance{
			Status: "human_reviewed", SourceReference: strings.Join(provenanceSources, ";"),
			Reviewer: strings.Join(provenanceReviewers, ";"), ReviewMethod: strings.Join(provenanceMethods, ";"),
			AlignmentTool: strings.Join(provenanceTools, ";"), AlignmentModel: strings.Join(provenanceModels, ";"),
			ReviewedAt: strings.Join(provenanceDates, ";"),
		}
	}
	return out, Realization{ClipIDs: ids, Reference: reference, DurationMs: int64(out.Duration() / time.Millisecond)}, nil
}

func normalizeClips(clips []Clip) ([]Clip, error) {
	out := make([]Clip, 0, len(clips))
	for _, c := range clips {
		if c.ID == "" {
			return nil, fmt.Errorf("experiment recipe: clip id is required")
		}
		sr := c.SampleRate
		if sr <= 0 {
			sr = CanonicalSampleRate
		}
		if sr != CanonicalSampleRate {
			return nil, fmt.Errorf("experiment recipe: clip %q sample rate %d is not canonical %d", c.ID, sr, CanonicalSampleRate)
		}
		cp := c
		cp.SampleRate = sr
		if cp.Format == "" {
			cp.Format = "pcm_s16le"
		}
		out = append(out, cp)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func duration(c Clip) time.Duration {
	if c.SampleRate <= 0 {
		return 0
	}
	return time.Duration(float64(len(c.PCM)) / float64(c.SampleRate*2) * float64(time.Second))
}

func silence(ms int) []byte {
	if ms <= 0 {
		return nil
	}
	return make([]byte, CanonicalSampleRate*2*ms/1000)
}
