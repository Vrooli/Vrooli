package composition

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strings"
	"unicode/utf8"

	"music-tools/internal/models"
	"music-tools/internal/styles"
)

const (
	providerACE = "ace-step"
	providerSA3 = "stable-audio-3"
)

// Only new admissions resolve defaults. Missing durable provider means the old
// always-ACE contract, even if current defaults/styles have changed.
func (p composeJobPayload) provider() (string, error) {
	switch p.Provider {
	case "", providerACE:
		return providerACE, nil
	case providerSA3:
		return providerSA3, nil
	default:
		if p.ModelSnapshot != nil && p.ModelSnapshot.Engine == p.Provider && p.ModelSnapshot.Runtime != nil && p.ModelSnapshot.Runtime.Protocol == "song-v1" {
			return p.Provider, nil
		}
		return "", fmt.Errorf("unsupported recorded provider %q", p.Provider)
	}
}

func resolveComposition(c styles.Compiled) (models.Model, error) {
	registry, err := models.LoadSeed(false)
	if err != nil {
		return models.Model{}, err
	}
	return registry.SelectComposition(c.Params.Variant, styles.ResolveLyrics(c.Lyrics) != styles.InstrumentalLyrics)
}

func validateProviderInput(provider string, c styles.Compiled, r composeRequest, permissive bool) error {
	registry, err := models.LoadSeed(false)
	if err != nil {
		return err
	}
	if err := registry.ValidateComposition(provider, c.Params.Variant, styles.ResolveLyrics(c.Lyrics) != styles.InstrumentalLyrics); err != nil {
		return err
	}
	if provider == providerSA3 {
		if permissive {
			return fmt.Errorf("Stable Audio 3 requires restricted Community/Gemma eligibility; permissive lane refuses it without fallback")
		}
		if r.Duration < 1 || r.Duration > 120 {
			return fmt.Errorf("Stable Audio 3 duration must be 1..120 seconds")
		}
		if utf8.RuneCountInString(c.Caption) > 12000 || strings.TrimSpace(c.Caption) == "" {
			return fmt.Errorf("Stable Audio 3 prompt must be nonblank and at most 12000 characters")
		}
		encoded, _ := json.Marshal(sa3Request{Prompt: c.Caption, Duration: 120, Seed: math.MaxUint32, Steps: 32, CFG: 20, Timeout: 300, Revision: sa3Revision})
		if len(encoded) > 65536 {
			return fmt.Errorf("Stable Audio 3 request exceeds 64KiB")
		}
		if r.Shift != nil {
			return fmt.Errorf("Stable Audio 3 does not support ACE shift")
		}
		if c.Params.BPM != 0 || strings.TrimSpace(c.Params.KeyScale) != "" {
			return fmt.Errorf("Stable Audio 3 requires bpm 0 and no keyscale; express tempo/key intentions in the caption")
		}
		if c.Params.Steps < 0 || c.Params.Steps > 32 {
			return fmt.Errorf("Stable Audio 3 steps must be 0 (unset)..32")
		}
	} else if provider == providerACE {
		if c.Params.BPM <= 0 {
			return fmt.Errorf("ACE requires positive structured bpm")
		}
	} else {
		model, err := declaredCompositionModel(provider, c.Params.Variant)
		if err != nil {
			return err
		}
		if model.Runtime == nil || model.Runtime.Protocol != "song-v1" {
			return fmt.Errorf("unsupported provider protocol for %q", provider)
		}
		if permissive && model.Licence != models.LicencePermissive {
			return fmt.Errorf("selected model %s requires the restricted licence lane; no fallback", model.ID)
		}
		runtime := model.Runtime
		if float64(r.Duration) < runtime.MinDuration || float64(r.Duration) > runtime.MaxDuration {
			return fmt.Errorf("duration outside declared model limits")
		}
		if r.Shift != nil || c.Params.BPM != 0 || strings.TrimSpace(c.Params.KeyScale) != "" {
			return fmt.Errorf("selected song model uses caption tempo/key and does not support ACE shift")
		}
		if strings.TrimSpace(c.Caption) == "" || utf8.RuneCountInString(c.Caption) > 12000 {
			return fmt.Errorf("song prompt must be nonblank and at most 12000 characters")
		}
		if utf8.RuneCountInString(styles.ResolveLyrics(c.Lyrics)) > 16000 {
			return fmt.Errorf("song lyrics exceed 16000 characters")
		}
		steps := c.Params.Steps
		if steps == 0 {
			steps = runtime.DefaultSteps
		}
		if steps < runtime.MinSteps || steps > runtime.MaxSteps {
			return fmt.Errorf("steps outside supported model controls")
		}
		guidance := c.Params.Guidance
		if guidance == 0 {
			guidance = runtime.DefaultGuidance
		}
		if math.IsNaN(guidance) || math.IsInf(guidance, 0) || guidance < runtime.MinGuidance || guidance > runtime.MaxGuidance {
			return fmt.Errorf("guidance outside supported model controls")
		}
		encoded, err := json.Marshal(songRequestFor(model, c, r, math.MaxUint32))
		if err != nil || len(encoded) > 65536 { return fmt.Errorf("song request exceeds bounded 64KiB JSON") }
	}
	return nil
}

func (s *ModuleState) resolvedInput(p composeJobPayload) (styles.Compiled, error) {
	if p.ResolvedStyle != nil {
		if p.ResolvedStyle.StyleID != p.StyleID {
			return styles.Compiled{}, fmt.Errorf("durable style snapshot identity mismatch")
		}
		return *p.ResolvedStyle, nil
	}
	provider, e := p.provider()
	if e != nil {
		return styles.Compiled{}, e
	}
	if provider != providerACE {
		return styles.Compiled{}, fmt.Errorf("resource job lacks immutable admission input")
	}
	c, e := s.Styles.Compile(p.StyleID)
	c.Lyrics = p.lyrics()
	return c, e
}

// Admission validates readiness; resource lifecycle owns SA3 residency and its
// one generation slot. It must never acquire an ACE offload/job claim.
func (s *ModuleState) sa3Admission(ctx context.Context) (func(), map[string]string, error) {
	if s.PermissiveOnly {
		return nil, nil, fmt.Errorf("Stable Audio 3 requires restricted Community/Gemma eligibility; permissive lane refuses it without fallback")
	}
	if e := s.sa3Ready(ctx); e != nil {
		return nil, nil, e
	}
	return nil, map[string]string{"composition_provider": providerSA3, "capacity_ownership": "resource:stable-audio-3"}, nil
}

func snapshotStyle(provider string, c styles.Compiled) *styles.Compiled {
	if provider == providerSA3 {
		if c.Params.Steps == 0 {
			c.Params.Steps = 8
		}
		if c.Params.Guidance == 0 {
			c.Params.Guidance = 1
		}
	}
	if model, err := declaredCompositionModel(provider, c.Params.Variant); err == nil && model.Runtime != nil {
		if c.Params.Steps == 0 {
			c.Params.Steps = model.Runtime.DefaultSteps
		}
		if c.Params.Guidance == 0 {
			c.Params.Guidance = model.Runtime.DefaultGuidance
		}
	}
	return &c
}

func declaredCompositionModel(engine, variant string) (models.Model, error) {
	registry, err := models.LoadSeed(false)
	if err != nil {
		return models.Model{}, err
	}
	for _, model := range registry.List() {
		if model.Engine == engine && model.Variant == variant && slices.Contains(model.Operations, "compose") {
			return model, nil
		}
	}
	return models.Model{}, fmt.Errorf("no declared composition model for engine %q and variant %q", engine, variant)
}

func snapshotModel(model models.Model) *models.Model {
	if model.Runtime == nil {
		return nil
	}
	return &model
}
