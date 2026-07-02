package experiment

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliapp"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ensureDefaultStrategies(recipe *experimentv1.ExperimentRecipe) {
	if recipe == nil || len(recipe.GetStrategies()) > 0 {
		return
	}
	recipe.Strategies = []*evalv1.EvalStrategy{
		{Kind: "batch", OverlapMaxStallRejects: -1},
		{Kind: "vad_segment", OverlapMaxStallRejects: -1},
		{Kind: "overlap_agree", OverlapMaxStallRejects: -1},
	}
}

func splitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func parseIntFlag(name, value string) (int, error) {
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("--%s must be an integer: %q", name, value)
	}
	return n, nil
}

func parseNonNegativeIntFlag(name, value string) (int, error) {
	n, err := parseIntFlag(name, value)
	if err != nil {
		return 0, err
	}
	if n < 0 {
		return 0, fmt.Errorf("--%s must be non-negative: %d", name, n)
	}
	return n, nil
}

func parseBoolFlag(name, value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("--%s must be true or false: %q", name, value)
	}
}

func parseFloatCSVFlag(name string, value string) ([]float64, error) {
	parts := splitCSV(value)
	out := make([]float64, 0, len(parts))
	for _, part := range parts {
		n, err := strconv.ParseFloat(part, 64)
		if err != nil {
			return nil, fmt.Errorf("--%s must be comma-separated numbers: %q", name, value)
		}
		out = append(out, n)
	}
	return out, nil
}

// parseIntCSVFlag parses a comma-separated list of ints into []int32.
func parseIntCSVFlag(name, value string) ([]int32, error) {
	parts := splitCSV(value)
	out := make([]int32, 0, len(parts))
	for _, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("--%s must be comma-separated integers: %q", name, value)
		}
		out = append(out, int32(n))
	}
	return out, nil
}

// optionalBoolFlag reads a tri-state boolean flag. It returns set=false when the
// flag was not provided. When the flag is present with no/empty value
// (e.g. `--long-form` or `--long-form=`) it means true; an explicit
// `--long-form true|false` is honored. (cli-core's parser cannot express a flag
// that is both bare-able and value-accepting, so these stay valued flags; the
// bare-without-`=` form is rejected by the parser before the handler runs.)
func optionalBoolFlag(ctx cliapp.RunContext, name string) (set bool, value bool, err error) {
	if !ctx.BoolFlag(name) {
		return false, false, nil
	}
	raw := strings.TrimSpace(ctx.Flag(name))
	if raw == "" {
		return true, true, nil
	}
	parsed, err := parseBoolFlag(name, raw)
	if err != nil {
		return false, false, err
	}
	return true, parsed, nil
}

func longFormHasContent(lf *experimentv1.LongFormRecipe) bool {
	return lf.GetEnabled() || lf.GetTargetDurationSeconds() > 0 || lf.GetGapMs() > 0 ||
		lf.GetTagContains() != "" || len(lf.GetSweepDurationsSeconds()) > 0
}

func augmentationHasContent(a *experimentv1.AugmentationRecipe) bool {
	return len(a.GetNoiseTypes()) > 0 || len(a.GetCompetingVoiceIds()) > 0 ||
		len(a.GetSnrDb()) > 0 || a.GetCompetingText() != ""
}

func speakerHasContent(s *experimentv1.SpeakerExperimentRecipe) bool {
	return s.GetTargetProfileId() != "" || s.GetExtractionEnabled() || s.GetVerificationEnabled() ||
		s.GetVerificationMode() != sttv1.SpeakerMode_SPEAKER_MODE_UNSPECIFIED || s.GetThreshold() != 0 ||
		s.GetFallbackWithoutVerification() || s.GetAblationEnabled()
}

func strategyUsesOverlap(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "overlap_agree", "overlap":
		return true
	default:
		return false
	}
}

func strategyUsesVADSilence(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "vad_segment", "vad":
		return true
	default:
		return false
	}
}

// loadBaseRecipe unmarshals a full ExperimentRecipe from --recipe-json (inline)
// or --recipe-file (path) via protojson, to be used as the base that individual
// flags then override. Returns an empty recipe when neither is set.
func loadBaseRecipe(ctx cliapp.RunContext) (*experimentv1.ExperimentRecipe, error) {
	inline := strings.TrimSpace(ctx.Flag("recipe-json"))
	file := strings.TrimSpace(ctx.Flag("recipe-file"))
	if inline != "" && file != "" {
		return nil, fmt.Errorf("--recipe-json and --recipe-file are mutually exclusive; pass only one")
	}
	recipe := &experimentv1.ExperimentRecipe{}
	raw := inline
	source := "--recipe-json"
	if file != "" {
		b, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("read --recipe-file: %w", err)
		}
		raw = string(b)
		source = "--recipe-file"
	}
	if strings.TrimSpace(raw) == "" {
		return recipe, nil
	}
	if err := protojson.Unmarshal([]byte(raw), recipe); err != nil {
		return nil, fmt.Errorf("parse %s as ExperimentRecipe JSON: %w", source, err)
	}
	return recipe, nil
}

// flattenCSV splits each argument on commas and concatenates the results.
func flattenCSV(args []string) []string {
	var out []string
	for _, arg := range args {
		out = append(out, splitCSV(arg)...)
	}
	return out
}

// dedupeStrings drops empties and duplicates while preserving first-seen order.
func dedupeStrings(in []string) []string {
	seen := make(map[string]bool, len(in))
	var out []string
	for _, v := range in {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func int32sCSV(values []int32) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = strconv.Itoa(int(v))
	}
	return strings.Join(parts, ",")
}

func float64sCSV(values []float64) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = strconv.FormatFloat(v, 'f', -1, 64)
	}
	return strings.Join(parts, ",")
}

func speakerModeFromFlag(s string) (sttv1.SpeakerMode, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "off":
		return sttv1.SpeakerMode_SPEAKER_MODE_OFF, nil
	case "filter":
		return sttv1.SpeakerMode_SPEAKER_MODE_FILTER, nil
	case "advisory":
		return sttv1.SpeakerMode_SPEAKER_MODE_ADVISORY, nil
	default:
		return sttv1.SpeakerMode_SPEAKER_MODE_UNSPECIFIED, fmt.Errorf("--speaker-mode must be off|filter|advisory: %q", s)
	}
}

func statusFromFlag(s string) (experimentv1.ExperimentStatus, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "queued":
		return experimentv1.ExperimentStatus_EXPERIMENT_STATUS_QUEUED, nil
	case "running":
		return experimentv1.ExperimentStatus_EXPERIMENT_STATUS_RUNNING, nil
	case "succeeded", "success":
		return experimentv1.ExperimentStatus_EXPERIMENT_STATUS_SUCCEEDED, nil
	case "failed", "failure":
		return experimentv1.ExperimentStatus_EXPERIMENT_STATUS_FAILED, nil
	case "canceled", "cancelled":
		return experimentv1.ExperimentStatus_EXPERIMENT_STATUS_CANCELED, nil
	default:
		return experimentv1.ExperimentStatus_EXPERIMENT_STATUS_UNSPECIFIED, fmt.Errorf("--status must be queued|running|succeeded|failed|canceled: %q", s)
	}
}

func statusLabel(s experimentv1.ExperimentStatus) string {
	switch s {
	case experimentv1.ExperimentStatus_EXPERIMENT_STATUS_QUEUED:
		return "queued"
	case experimentv1.ExperimentStatus_EXPERIMENT_STATUS_RUNNING:
		return "running"
	case experimentv1.ExperimentStatus_EXPERIMENT_STATUS_SUCCEEDED:
		return "succeeded"
	case experimentv1.ExperimentStatus_EXPERIMENT_STATUS_FAILED:
		return "failed"
	case experimentv1.ExperimentStatus_EXPERIMENT_STATUS_CANCELED:
		return "canceled"
	default:
		return "unspecified"
	}
}

func timestampLabel(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return "-"
	}
	return ts.AsTime().Format("2006-01-02T15:04:05Z07:00")
}

func strategyLabels(strategies []*evalv1.EvalStrategy) string {
	if len(strategies) == 0 {
		return "default"
	}
	out := make([]string, 0, len(strategies))
	for _, s := range strategies {
		out = append(out, s.GetKind())
	}
	return strings.Join(out, ",")
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 0 {
		return ""
	}
	if max == 1 {
		return string(runes[:1])
	}
	return string(runes[:max-1]) + "…"
}
