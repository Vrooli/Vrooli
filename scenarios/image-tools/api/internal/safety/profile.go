package safety

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

// ReviewProfile is the declarative description of the content reviewer: which
// classifiers run, at which bars, which face/age models escalate a sexual image,
// and what the host does with a refusal. It is data, not code: profiles.json is
// the shipped default and IMAGE_TOOLS_SAFETY_PROFILE can point at another file.
//
// The profile is resolved once at boot from the deployment tier and carried on
// Policy. The bundled Python reviewer receives the same object in its request, so
// the host and the reviewer cannot disagree about a threshold or a model id.
type ReviewProfile struct {
	SchemaVersion int    `json:"schema_version"`
	Name          string `json:"name"`
	// Mode is "enforce" (refuse) or "shadow" (record the verdict and continue).
	// Shadow mode is for local experimentation; a public profile should enforce.
	Mode ReviewMode `json:"mode"`
	// OnRefusal says what the engine does when one variation is refused:
	// "fail_job" deletes every variation (public default) or "drop_variation"
	// deletes only the refused bytes and returns the passing variations.
	OnRefusal RefusalAction `json:"on_refusal"`
	// Classifiers are the sexual-content classifiers, in score-reporting order.
	// The first is the "primary" score recorded on the job.
	Classifiers []ClassifierProfile `json:"classifiers"`
	// Aggregate is how classifier verdicts combine. "any" (a single classifier
	// crossing its own bar is enough) is the only supported value today.
	Aggregate string      `json:"aggregate"`
	Face      FaceProfile `json:"face"`
	Age       AgeProfile  `json:"age"`
	// SecondOpinion lets a face-independent age reading clear only the
	// "sexual image whose subject's age could not be established" refusal.
	SecondOpinion SecondOpinionProfile `json:"second_opinion"`
	// OnMinorFace is the non-negotiable class: a minor-appearing face in sexual
	// content. Only "refuse" is accepted.
	OnMinorFace string `json:"on_minor_face"`
	// OnUnassessable is "refuse" or "second_opinion".
	OnUnassessable UnassessableAction `json:"on_unassessable"`

	MaxImagePixels int64 `json:"max_image_pixels"`
	// QuarantineHours keeps refused outputs in a private quarantine for admin
	// review before purging them. 0 deletes a refused output immediately.
	QuarantineHours int    `json:"quarantine_hours"`
	ReviewerVersion string `json:"reviewer_version"`
}

// ReviewMode controls whether a refusal is enforced or only recorded.
type ReviewMode string

const (
	ReviewModeEnforce ReviewMode = "enforce"
	ReviewModeShadow  ReviewMode = "shadow"
)

// RefusalAction controls the blast radius of a refused variation.
type RefusalAction string

const (
	// RefusalFailJob preserves the public contract: any refusal fails the job and
	// the refused bytes never remain addressable.
	RefusalFailJob RefusalAction = "fail_job"
	// RefusalDropVariation deletes only the refused variation and returns the
	// others. It never publishes refused bytes; it avoids throwing away good
	// variations because one of several was flagged.
	RefusalDropVariation RefusalAction = "drop_variation"
)

// UnassessableAction decides what to do when a sexual image's subject age could
// not be established by the face/age stage.
type UnassessableAction string

const (
	UnassessableRefuse        UnassessableAction = "refuse"
	UnassessableSecondOpinion UnassessableAction = "second_opinion"
)

// ClassifierProfile is one sexual-content classifier and its two bars. The
// strict bars apply to output from a model labelled nsfw_capable, whose synthetic
// images the classifiers read less reliably.
type ClassifierProfile struct {
	ID              string   `json:"id"`
	Threshold       float64  `json:"threshold"`
	StrictThreshold *float64 `json:"strict_threshold,omitempty"`
	// InputHW is the model's resize target. Omitted keeps the historical default
	// (224 for the precise classifier, 384 for the high-recall one).
	InputHW []int `json:"input_hw,omitempty"`
}

// EffectiveThreshold is the bar this classifier uses for the request.
func (c ClassifierProfile) EffectiveThreshold(strict bool) float64 {
	if strict && c.StrictThreshold != nil {
		return *c.StrictThreshold
	}
	return c.Threshold
}

// FaceProfile locates and configures the face detector.
type FaceProfile struct {
	ID             string  `json:"id"`
	ScoreThreshold float64 `json:"score_threshold"`
	MinPx          int     `json:"min_px"`
	CropMargin     float64 `json:"crop_margin"`
	MaxFaces       int     `json:"max_faces"`
}

// AgeProfile locates the age head and sets the minor threshold.
type AgeProfile struct {
	ID        string  `json:"id"`
	Threshold float64 `json:"threshold"`
}

// SecondOpinionProfile configures the face-independent age judge. The model
// choice is data too: a deployment can point the judge at a different local
// vision role or model without changing safety code.
type SecondOpinionProfile struct {
	Enabled       bool     `json:"enabled"`
	MinAge        int      `json:"min_age"`
	MinConfidence string   `json:"min_confidence"`
	JudgeRole     string   `json:"judge_role,omitempty"`
	Models        []string `json:"models,omitempty"`
	OllamaURL     string   `json:"ollama_url,omitempty"`
}

// Shipped profile names. A deployment tier selects one; a JSON override can add
// more names and IMAGE_TOOLS_SAFETY_PROFILE_NAME selects among them.
const (
	ProfileLocal  = "local"
	ProfilePublic = "public"
)

const (
	// ReviewProfileEnv points at a JSON file with the same shape as
	// profiles.json. It replaces the shipped defaults for this process.
	ReviewProfileEnv = "IMAGE_TOOLS_SAFETY_PROFILE"
	// ReviewProfileNameEnv selects a profile by name from that file.
	ReviewProfileNameEnv = "IMAGE_TOOLS_SAFETY_PROFILE_NAME"
	// ReviewModeEnv overrides the selected profile's mode.
	ReviewModeEnv = "IMAGE_TOOLS_SAFETY_MODE"
	// ReviewOnRefusalEnv overrides the selected profile's refusal action.
	ReviewOnRefusalEnv = "IMAGE_TOOLS_SAFETY_ON_REFUSAL"
	// ReviewQuarantineHoursEnv overrides how long a refused output is retained
	// in the private quarantine for admin review. 0 deletes it immediately.
	ReviewQuarantineHoursEnv = "IMAGE_TOOLS_SAFETY_QUARANTINE_HOURS"
	// ReviewQuarantineTokenEnv is the explicit admin token a public/cloud
	// deployment must configure to expose the quarantine review surface.
	// Local deployments do not require it.
	ReviewQuarantineTokenEnv = "IMAGE_TOOLS_SAFETY_QUARANTINE_TOKEN"
)

//go:embed profiles.json
var profilesJSON []byte

type profileSet struct {
	SchemaVersion int                      `json:"schema_version"`
	Profiles      map[string]ReviewProfile `json:"profiles"`
}

var (
	defaultProfilesOnce sync.Once
	defaultProfiles     map[string]ReviewProfile
	defaultProfilesErr  error
)

// LoadReviewProfiles parses and validates a profile set. An empty path uses the
// embedded profiles.json.
func LoadReviewProfiles(path string) (map[string]ReviewProfile, error) {
	raw := profilesJSON
	if strings.TrimSpace(path) != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("safety: read review profile %q: %w", path, err)
		}
		raw = data
	}
	var set profileSet
	if err := json.Unmarshal(raw, &set); err != nil {
		return nil, fmt.Errorf("safety: parse review profiles: %w", err)
	}
	if set.SchemaVersion != 1 {
		return nil, fmt.Errorf("safety: review profile schema_version = %d, want 1", set.SchemaVersion)
	}
	if len(set.Profiles) == 0 {
		return nil, fmt.Errorf("safety: review profile set is empty")
	}
	for name, profile := range set.Profiles {
		profile.SchemaVersion = set.SchemaVersion
		if err := profile.Validate(); err != nil {
			return nil, fmt.Errorf("safety: profile %q: %w", name, err)
		}
		set.Profiles[name] = profile
	}
	return set.Profiles, nil
}

// EmbeddedReviewProfiles returns the shipped profiles. It is intended for tests
// and callers that need the baseline without environment overrides.
func EmbeddedReviewProfiles() (map[string]ReviewProfile, error) {
	defaultProfilesOnce.Do(func() {
		defaultProfiles, defaultProfilesErr = LoadReviewProfiles("")
	})
	if defaultProfilesErr != nil {
		return nil, defaultProfilesErr
	}
	out := make(map[string]ReviewProfile, len(defaultProfiles))
	for name, profile := range defaultProfiles {
		out[name] = profile.Clone()
	}
	return out, nil
}

// DefaultReviewProfile returns the shipped profile for a tier. The embedded set
// is validated at load time; a broken build panics rather than silently running
// a different reviewer.
func DefaultReviewProfile(t Tier) ReviewProfile {
	profiles, err := EmbeddedReviewProfiles()
	if err != nil {
		panic("safety: invalid embedded review profiles: " + err.Error())
	}
	name := ProfileLocal
	if t == TierPublic {
		name = ProfilePublic
	}
	profile, ok := profiles[name]
	if !ok {
		panic("safety: embedded review profiles lack " + name)
	}
	return profile.Clone()
}

// ResolveReviewProfile applies the operator override path/name and the mode and
// refusal overrides to the tier default. A malformed override is an error rather
// than a silent fallback: a deployment must know its reviewer configuration.
func ResolveReviewProfile(t Tier, lookup func(string) (string, bool)) (ReviewProfile, error) {
	if lookup == nil {
		lookup = lookupEnv
	}
	path := ""
	if v, ok := lookup(ReviewProfileEnv); ok {
		path = strings.TrimSpace(v)
	}
	profiles, err := LoadReviewProfiles(path)
	if err != nil {
		return ReviewProfile{}, err
	}
	name := ProfileLocal
	if t == TierPublic {
		name = ProfilePublic
	}
	if v, ok := lookup(ReviewProfileNameEnv); ok && strings.TrimSpace(v) != "" {
		name = strings.TrimSpace(v)
	}
	profile, ok := profiles[name]
	if !ok {
		return ReviewProfile{}, fmt.Errorf("safety: review profile %q is not defined", name)
	}
	if v, ok := lookup(ReviewModeEnv); ok && strings.TrimSpace(v) != "" {
		mode := ReviewMode(strings.ToLower(strings.TrimSpace(v)))
		if mode != ReviewModeEnforce && mode != ReviewModeShadow {
			return ReviewProfile{}, fmt.Errorf("safety: %s = %q, want enforce or shadow", ReviewModeEnv, v)
		}
		profile.Mode = mode
	}
	if v, ok := lookup(ReviewOnRefusalEnv); ok && strings.TrimSpace(v) != "" {
		action := RefusalAction(strings.ToLower(strings.TrimSpace(v)))
		if action != RefusalFailJob && action != RefusalDropVariation {
			return ReviewProfile{}, fmt.Errorf("safety: %s = %q, want fail_job or drop_variation", ReviewOnRefusalEnv, v)
		}
		profile.OnRefusal = action
	}
	if v, ok := lookup(ReviewQuarantineHoursEnv); ok && strings.TrimSpace(v) != "" {
		hours, parseErr := strconv.Atoi(strings.TrimSpace(v))
		if parseErr != nil || hours < 0 {
			return ReviewProfile{}, fmt.Errorf("safety: %s = %q, want a non-negative integer", ReviewQuarantineHoursEnv, v)
		}
		profile.QuarantineHours = hours
	}
	if err := profile.Validate(); err != nil {
		return ReviewProfile{}, err
	}
	return profile.Clone(), nil
}

// Validate checks the invariants every consumer relies on.
func (p ReviewProfile) Validate() error {
	if p.SchemaVersion != 0 && p.SchemaVersion != 1 {
		return fmt.Errorf("schema_version = %d, want 1", p.SchemaVersion)
	}
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("name is required")
	}
	switch p.Mode {
	case ReviewModeEnforce, ReviewModeShadow:
	default:
		return fmt.Errorf("mode = %q, want enforce or shadow", p.Mode)
	}
	switch p.OnRefusal {
	case RefusalFailJob, RefusalDropVariation:
	default:
		return fmt.Errorf("on_refusal = %q, want fail_job or drop_variation", p.OnRefusal)
	}
	if len(p.Classifiers) == 0 {
		return fmt.Errorf("at least one classifier is required")
	}
	seen := map[string]bool{}
	for i, c := range p.Classifiers {
		if strings.TrimSpace(c.ID) == "" {
			return fmt.Errorf("classifiers[%d].id is required", i)
		}
		if seen[c.ID] {
			return fmt.Errorf("classifier %q is listed twice", c.ID)
		}
		seen[c.ID] = true
		if err := unitInterval("classifiers["+strconv.Itoa(i)+"].threshold", c.Threshold); err != nil {
			return err
		}
		if c.StrictThreshold != nil {
			if err := unitInterval("classifiers["+strconv.Itoa(i)+"].strict_threshold", *c.StrictThreshold); err != nil {
				return err
			}
			if *c.StrictThreshold > c.Threshold {
				return fmt.Errorf("classifier %q strict threshold %.3f is above its threshold %.3f", c.ID, *c.StrictThreshold, c.Threshold)
			}
		}
		if len(c.InputHW) != 0 && (len(c.InputHW) != 2 || c.InputHW[0] <= 0 || c.InputHW[1] <= 0) {
			return fmt.Errorf("classifier %q input_hw must be two positive integers", c.ID)
		}
	}
	if p.Aggregate != "any" {
		return fmt.Errorf("aggregate = %q, want any", p.Aggregate)
	}
	if strings.TrimSpace(p.Face.ID) == "" {
		return fmt.Errorf("face.id is required")
	}
	if err := unitInterval("face.score_threshold", p.Face.ScoreThreshold); err != nil {
		return err
	}
	if p.Face.MinPx <= 0 {
		return fmt.Errorf("face.min_px must be positive")
	}
	if p.Face.CropMargin < 1 {
		return fmt.Errorf("face.crop_margin must be at least 1")
	}
	if p.Face.MaxFaces <= 0 {
		return fmt.Errorf("face.max_faces must be positive")
	}
	if strings.TrimSpace(p.Age.ID) == "" {
		return fmt.Errorf("age.id is required")
	}
	if err := unitInterval("age.threshold", p.Age.Threshold); err != nil {
		return err
	}
	if p.OnMinorFace != "refuse" {
		return fmt.Errorf("on_minor_face = %q, only refuse is supported", p.OnMinorFace)
	}
	switch p.OnUnassessable {
	case UnassessableRefuse:
	case UnassessableSecondOpinion:
		if !p.SecondOpinion.Enabled {
			return fmt.Errorf("on_unassessable = second_opinion but second_opinion.enabled is false")
		}
	default:
		return fmt.Errorf("on_unassessable = %q, want refuse or second_opinion", p.OnUnassessable)
	}
	if p.SecondOpinion.Enabled {
		if p.SecondOpinion.MinAge < 18 {
			return fmt.Errorf("second_opinion.min_age must be at least 18")
		}
		confidence := strings.ToLower(strings.TrimSpace(p.SecondOpinion.MinConfidence))
		if confidence != "medium" && confidence != "high" {
			return fmt.Errorf("second_opinion.min_confidence = %q, want medium or high (low never clears)", p.SecondOpinion.MinConfidence)
		}
		for i, model := range p.SecondOpinion.Models {
			if strings.TrimSpace(model) == "" {
				return fmt.Errorf("second_opinion.models[%d] is empty", i)
			}
		}
	}
	if p.MaxImagePixels <= 0 {
		return fmt.Errorf("max_image_pixels must be positive")
	}
	if p.QuarantineHours < 0 || p.QuarantineHours > 8760 {
		return fmt.Errorf("quarantine_hours = %d, want 0..8760", p.QuarantineHours)
	}
	if strings.TrimSpace(p.ReviewerVersion) == "" {
		return fmt.Errorf("reviewer_version is required")
	}
	return nil
}

func unitInterval(field string, v float64) error {
	if v < 0 || v > 1 {
		return fmt.Errorf("%s = %v, want 0..1", field, v)
	}
	return nil
}

// Clone returns a deep copy safe for a caller to mutate (e.g. a per-service
// threshold override) without changing the embedded profile.
func (p ReviewProfile) Clone() ReviewProfile {
	out := p
	out.Classifiers = append([]ClassifierProfile(nil), p.Classifiers...)
	for i := range out.Classifiers {
		if p.Classifiers[i].StrictThreshold != nil {
			value := *p.Classifiers[i].StrictThreshold
			out.Classifiers[i].StrictThreshold = &value
		}
		out.Classifiers[i].InputHW = append([]int(nil), p.Classifiers[i].InputHW...)
	}
	out.SecondOpinion.Models = append([]string(nil), p.SecondOpinion.Models...)
	return out
}

// ClassifierIDs returns the classifier model ids in order.
func (p ReviewProfile) ClassifierIDs() []string {
	out := make([]string, 0, len(p.Classifiers))
	for _, c := range p.Classifiers {
		out = append(out, c.ID)
	}
	return out
}

// EscalationIDs returns the face and age model ids read only for a sexual image.
func (p ReviewProfile) EscalationIDs() []string {
	out := make([]string, 0, 2)
	seen := map[string]bool{}
	for _, id := range []string{p.Face.ID, p.Age.ID} {
		if strings.TrimSpace(id) != "" && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// Summary is a compact human description for discovery and logs.
func (p ReviewProfile) Summary() string {
	parts := make([]string, 0, len(p.Classifiers)+4)
	for _, c := range p.Classifiers {
		bar := fmt.Sprintf("%s %.2f", c.ID, c.Threshold)
		if c.StrictThreshold != nil {
			bar += fmt.Sprintf("/%.2f strict", *c.StrictThreshold)
		}
		parts = append(parts, bar)
	}
	out := fmt.Sprintf("review profile %s (%s, %s): sexual %s; age<20 %.2f (%s)",
		p.Name, p.Mode, p.OnRefusal, strings.Join(parts, ", "), p.Age.Threshold, p.Age.ID)
	if p.OnUnassessable == UnassessableSecondOpinion {
		out += fmt.Sprintf("; unassessable -> second opinion (adult >= %d, %s confidence)", p.SecondOpinion.MinAge, p.SecondOpinion.MinConfidence)
	} else {
		out += "; unassessable -> refuse"
	}
	if p.QuarantineHours > 0 {
		out += fmt.Sprintf("; refusals quarantine %dh for admin review", p.QuarantineHours)
	} else {
		out += "; refusals delete immediately"
	}
	return out
}
