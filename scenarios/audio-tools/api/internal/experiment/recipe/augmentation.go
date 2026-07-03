package recipe

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"strings"

	audiomix "audio-tools/internal/audio/mix"
	inteval "audio-tools/internal/eval"
)

const defaultCompetingText = "This is a competing speaker in the background during the dictation experiment."

// knownNoiseTypes is the set of noise beds GenerateNoise renders distinctly
// (canonical user-facing names plus documented aliases). Anything else silently
// falls through to the white-noise default branch, so the recipe validator
// rejects unknown types up front rather than running a mislabeled condition.
var knownNoiseTypes = map[string]struct{}{
	"white":        {},
	"fan":          {},
	"constant_fan": {},
	"percussive":   {},
	"dynamic":      {},
	"music":        {},
	"music_like":   {},
}

// KnownNoiseTypes returns the canonical, user-facing noise bed names in a stable
// order for help text and validation messages.
func KnownNoiseTypes() []string {
	return []string{"white", "fan", "percussive", "music"}
}

// IsKnownNoiseType reports whether kind names a supported noise bed (canonical
// name or documented alias), case- and whitespace-insensitive.
func IsKnownNoiseType(kind string) bool {
	_, ok := knownNoiseTypes[strings.ToLower(strings.TrimSpace(kind))]
	return ok
}

// AugmentationSpec describes deterministic audio overlays for experiment
// inputs. Noise is generated locally; competing voices are supplied as
// canonical PCM by the caller so this package stays pure/testable.
type AugmentationSpec struct {
	Seed            int64
	NoiseTypes      []string
	SNRDB           []float64
	CompetingVoices []string
	CompetingText   string
	SynthesizeVoice func(context.Context, string, string) ([]byte, error)
}

// Condition records one realized augmentation cell.
type Condition struct {
	ID      string
	Kind    string
	Source  string
	SNRDB   float64
	Skipped bool
	Note    string
}

// ConditionGroup is the set of eval clips that belong to one augmentation
// condition row. ID is "clean", "noise:<type>/<snr>db", or
// "voice:<voice>/<snr>db".
type ConditionGroup struct {
	ID    string
	Clips []inteval.Clip
}

// ApplyAugmentation returns clean input clips plus every realized
// noise/competing-voice condition. The same seed/spec/base audio produces
// byte-identical generated noise and condition ordering.
func ApplyAugmentation(ctx context.Context, base []inteval.Clip, spec AugmentationSpec) ([]inteval.Clip, []Condition, error) {
	if !spec.enabled() {
		return base, nil, nil
	}
	if len(base) == 0 {
		return nil, nil, fmt.Errorf("experiment augmentation: no base clips")
	}
	snrs := spec.SNRDB
	if len(snrs) == 0 {
		snrs = []float64{12}
	}
	seed := spec.Seed
	if seed == 0 {
		seed = defaultSeed
	}

	out := append([]inteval.Clip(nil), base...)
	conditions := []Condition{{ID: "clean", Kind: "clean", Source: "none", Note: "unmodified input"}}
	for _, noiseType := range normalizeStrings(spec.NoiseTypes) {
		for _, snr := range snrs {
			for _, clip := range base {
				noise := GenerateNoise(noiseType, len(clip.PCM), seed, clip.ID)
				mixed, stats, err := audiomix.OverlayAtSNR(clip.PCM, noise, snr)
				id := fmt.Sprintf("%s/noise:%s/%gdb", clip.ID, noiseType, snr)
				cond := Condition{ID: id, Kind: "noise", Source: noiseType, SNRDB: snr}
				if err != nil {
					cond.Skipped = true
					cond.Note = err.Error()
					conditions = append(conditions, cond)
					continue
				}
				cond.Note = fmt.Sprintf("actual_snr_db=%.2f clipped=%d", stats.ActualSNRDB, stats.ClippedCount)
				conditions = append(conditions, cond)
				aug := clip
				aug.ID = id
				aug.PCM = mixed
				out = append(out, aug)
			}
		}
	}

	text := strings.TrimSpace(spec.CompetingText)
	if text == "" {
		text = defaultCompetingText
	}
	for _, voice := range normalizeStrings(spec.CompetingVoices) {
		if spec.SynthesizeVoice == nil {
			conditions = append(conditions, Condition{ID: "voice:" + voice, Kind: "competing_voice", Source: voice, Skipped: true, Note: "TTS synthesis is not configured"})
			continue
		}
		pcm, err := spec.SynthesizeVoice(ctx, voice, text)
		if err != nil {
			conditions = append(conditions, Condition{ID: "voice:" + voice, Kind: "competing_voice", Source: voice, Skipped: true, Note: "augmentation skipped: " + err.Error()})
			continue
		}
		for _, snr := range snrs {
			for _, clip := range base {
				mixed, stats, err := audiomix.OverlayAtSNR(clip.PCM, pcm, snr)
				id := fmt.Sprintf("%s/voice:%s/%gdb", clip.ID, voice, snr)
				cond := Condition{ID: id, Kind: "competing_voice", Source: voice, SNRDB: snr}
				if err != nil {
					cond.Skipped = true
					cond.Note = err.Error()
					conditions = append(conditions, cond)
					continue
				}
				cond.Note = fmt.Sprintf("actual_snr_db=%.2f clipped=%d text=%q", stats.ActualSNRDB, stats.ClippedCount, text)
				conditions = append(conditions, cond)
				aug := clip
				aug.ID = id
				aug.PCM = mixed
				out = append(out, aug)
			}
		}
	}
	return out, conditions, nil
}

// GroupClipsByAugCondition folds ApplyAugmentation output back into report
// rows. ApplyAugmentation stamps augmented clip IDs as
// "<base-id>/noise:<type>/<snr>db" or "<base-id>/voice:<id>/<snr>db"; the
// suffix is the cross-clip condition key.
func GroupClipsByAugCondition(clips []inteval.Clip) []ConditionGroup {
	groups := make([]ConditionGroup, 0)
	index := make(map[string]int)
	for _, clip := range clips {
		id := augmentationConditionID(clip.ID)
		pos, ok := index[id]
		if !ok {
			index[id] = len(groups)
			groups = append(groups, ConditionGroup{ID: id})
			pos = len(groups) - 1
		}
		groups[pos].Clips = append(groups[pos].Clips, clip)
	}
	return groups
}

func augmentationConditionID(clipID string) string {
	for _, marker := range []string{"/noise:", "/voice:"} {
		if idx := strings.Index(clipID, marker); idx >= 0 {
			return clipID[idx+1:]
		}
	}
	return "clean"
}

func (s AugmentationSpec) enabled() bool {
	return len(s.NoiseTypes) > 0 || len(s.CompetingVoices) > 0
}

// GenerateNoise creates a deterministic canonical PCM noise bed.
func GenerateNoise(kind string, length int, seed int64, salt string) []byte {
	if length <= 0 {
		return nil
	}
	if length%2 != 0 {
		length--
	}
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind == "" {
		kind = "white"
	}
	rng := rand.New(rand.NewSource(seed + int64(hashString(kind+"|"+salt))))
	out := make([]byte, length)
	samples := length / 2
	for i := 0; i < samples; i++ {
		var v int
		switch kind {
		case "fan", "constant_fan":
			v = int(9000*math.Sin(2*math.Pi*float64(i)/160)) + int(2500*math.Sin(2*math.Pi*float64(i)/53))
		case "percussive", "dynamic":
			if i%3200 < 160 {
				v = rng.Intn(24000) - 12000
			} else {
				v = rng.Intn(2000) - 1000
			}
		case "music", "music_like":
			v = int(7000*math.Sin(2*math.Pi*float64(i)/145)) + int(5000*math.Sin(2*math.Pi*float64(i)/193)) + int(2500*math.Sin(2*math.Pi*float64(i)/241))
		default:
			v = rng.Intn(24000) - 12000
		}
		binary.LittleEndian.PutUint16(out[i*2:], uint16(int16(v)))
	}
	return out
}

func normalizeStrings(in []string) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		if s := strings.TrimSpace(v); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func hashString(s string) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}
