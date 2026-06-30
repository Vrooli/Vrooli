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
	ID         string
	PCM        []byte
	SampleRate int
	Reference  string
	Format     string
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

	rng := rand.New(rand.NewSource(seed))
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
	ids := make([]string, 0, len(selected))
	for i, clip := range selected {
		if i > 0 {
			pcm = append(pcm, gap...)
		}
		pcm = append(pcm, clip.PCM...)
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
