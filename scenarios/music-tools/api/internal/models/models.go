// Package models is the small music model registry. The resource owns weight
// acquisition; this registry owns the caller-visible identity and licence lane.
package models

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"runtime"
	"slices"
	"sort"
	"strings"
)

type LicenceLane string

const (
	LicencePermissive LicenceLane = "permissive"
	LicenceRestricted LicenceLane = "restricted"
)

// Capability describes supported behavior, not guaranteed audible output.
type Capability struct {
	Supported bool   `json:"supported"`
	Reason    string `json:"reason"`
	Source    string `json:"source"`
}

type Model struct {
	Runtime               *CompositionRuntime `json:"runtime,omitempty"`
	Engine                string              `json:"engine"`
	CompositionPreference int                 `json:"composition_preference"`
	SungLyrics            Capability          `json:"sung_lyrics"`
	VoiceReference        Capability          `json:"voice_reference"`
	ID                    string              `json:"id"`
	DisplayName           string              `json:"display_name"`
	Variant               string              `json:"variant"`
	Revision              string              `json:"revision"`
	Licence               LicenceLane         `json:"licence"`
	Installed             bool                `json:"installed"`
	Bytes                 uint64              `json:"bytes"`
	Operations            []string            `json:"operations"`
	Hardware              Hardware            `json:"hardware"`
	License               string              `json:"license"`
	CommercialUse         string              `json:"commercial_use"`
	CommercialUseNotes    string              `json:"commercial_use_notes"`
	BaseModelLineage      string              `json:"base_model_lineage"`
	KnownRisks            string              `json:"known_risks"`
	Checksum              string              `json:"checksum"`
	DiskFloorBytes        uint64              `json:"disk_floor_bytes"`
}

// CompositionRuntime describes the local song protocol, independently of a
// model name. Controls and identity are declarations, not provider-name guesses.
type CompositionRuntime struct {
	Protocol        string            `json:"protocol"`
	Resource        string            `json:"resource"`
	Profile         string            `json:"profile"`
	URLVariable     string            `json:"url_variable"`
	DefaultURL      string            `json:"default_url"`
	CatalogSHA256   string            `json:"catalog_sha256"`
	ArtifactSHA256  string            `json:"artifact_sha256"`
	SourceRevision  string            `json:"source_revision"`
	WeightFiles     int               `json:"weight_files"`
	Versions        map[string]string `json:"versions"`
	MinDuration     float64           `json:"min_duration"`
	MaxDuration     float64           `json:"max_duration"`
	DefaultSteps    int               `json:"default_steps"`
	MinSteps        int               `json:"min_steps"`
	MaxSteps        int               `json:"max_steps"`
	DefaultGuidance float64           `json:"default_guidance"`
	MinGuidance     float64           `json:"min_guidance"`
	MaxGuidance     float64           `json:"max_guidance"`
	TimeoutSeconds  int               `json:"timeout_seconds"`
}

type Hardware struct {
	MinVRAMBytes uint64 `json:"min_vram_bytes"`
	MinRAMBytes  uint64 `json:"min_ram_bytes"`
	CPUCapable   bool   `json:"cpu_capable"`
}

type Host struct {
	VRAMBytes     uint64 `json:"vram_bytes"`
	FreeVRAMBytes uint64 `json:"free_vram_bytes"`
	RAMBytes      uint64 `json:"ram_bytes"`
	FreeRAMBytes  uint64 `json:"free_ram_bytes"`
	OS            string `json:"os"`
	Arch          string `json:"arch"`
}

//go:embed registry.seed.json
var seedBytes []byte

func LoadSeed(permissiveBuild bool) (*Registry, error) {
	var seed []Model
	if err := json.Unmarshal(seedBytes, &seed); err != nil {
		return nil, fmt.Errorf("models: parse seed: %w", err)
	}
	r := New(permissiveBuild, seed...)
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return r, nil
}

var ErrRestrictedModel = errors.New("models: restricted model is not allowed in permissive lane")

type Registry struct {
	models          map[string]Model
	permissiveBuild bool
}

func New(permissiveBuild bool, seed ...Model) *Registry {
	r := &Registry{models: map[string]Model{}, permissiveBuild: permissiveBuild}
	for _, m := range seed {
		r.Add(m)
	}
	return r
}

func (r *Registry) Add(m Model) {
	if m.Licence == "" {
		m.Licence = LicenceRestricted
	}
	r.models[m.ID] = m
}

func (r *Registry) Resolve(id string) (Model, error) {
	m, ok := r.models[id]
	if !ok {
		return Model{}, fmt.Errorf("models: unknown model %q", id)
	}
	if r.permissiveBuild && m.Licence != LicencePermissive {
		return Model{}, fmt.Errorf("%w: %s", ErrRestrictedModel, id)
	}
	return m, nil
}

func (r *Registry) List() []Model {
	out := make([]Model, 0, len(r.models))
	for _, m := range r.models {
		if !r.permissiveBuild || m.Licence == LicencePermissive {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// compositionModels is the existing registry's declared preference order. A
// resource/variant name never implies lyric support, installation or readiness.
func (r *Registry) compositionModels() []Model {
	out := []Model{}
	for _, m := range r.models {
		if m.Engine != "" && m.CompositionPreference > 0 && slices.Contains(m.Operations, "compose") {
			out = append(out, m)
		}
	}
	slices.SortFunc(out, func(a, b Model) int {
		if a.CompositionPreference < b.CompositionPreference {
			return -1
		}
		if a.CompositionPreference > b.CompositionPreference {
			return 1
		}
		return strings.Compare(a.ID, b.ID)
	})
	return out
}

func (r *Registry) requireLyrics(m Model, needsLyrics bool) (Model, error) {
	if needsLyrics && !m.SungLyrics.Supported {
		reason, source := m.SungLyrics.Reason, m.SungLyrics.Source
		if reason == "" {
			reason = "sung-lyrics capability is not declared"
		}
		if source == "" {
			source = "no capability source declared"
		}
		return Model{}, fmt.Errorf("models: %s cannot satisfy authored sung lyrics: %s (source: %s)", m.ID, reason, source)
	}
	return r.Resolve(m.ID)
}

// SelectComposition preserves explicit choices. Otherwise capability filters the
// declared preference order. Licence refusal never triggers a silent fallback.
func (r *Registry) SelectComposition(variant string, needsLyrics bool) (Model, error) {
	for _, m := range r.compositionModels() {
		if variant != "" {
			if m.Variant == variant {
				return r.requireLyrics(m, needsLyrics)
			}
		} else if !needsLyrics || m.SungLyrics.Supported {
			return r.requireLyrics(m, needsLyrics)
		}
	}
	if variant != "" {
		return Model{}, fmt.Errorf("models: unsupported explicit model variant %q", variant)
	}
	return Model{}, fmt.Errorf("models: no declared composition model supports the requested capability")
}

// ValidateComposition checks recorded ownership without resolving a new default
// for recovered jobs. An absent old provider is mapped to ACE by the caller.
func (r *Registry) ValidateComposition(engine, variant string, needsLyrics bool) error {
	for _, m := range r.compositionModels() {
		if m.Engine == engine && (variant == "" || m.Variant == variant) {
			_, err := r.requireLyrics(m, needsLyrics)
			return err
		}
	}
	return fmt.Errorf("models: no declared composition model matches recorded engine %q and variant %q", engine, variant)
}

func (r *Registry) Validate() error {
	for _, model := range r.models {
		if model.ID == "" || model.DisplayName == "" {
			return fmt.Errorf("models: incomplete entry %q", model.ID)
		}
		if model.Licence != LicencePermissive && model.Licence != LicenceRestricted {
			return fmt.Errorf("models: %s has invalid licence lane %q", model.ID, model.Licence)
		}
		if model.License == "" || model.CommercialUse == "" || model.BaseModelLineage == "" || model.KnownRisks == "" {
			return fmt.Errorf("models: %s is missing capability labels", model.ID)
		}
		if runtime := model.Runtime; runtime != nil {
			if runtime.Protocol != "song-v1" || runtime.Resource == "" || runtime.Profile == "" || runtime.URLVariable == "" || runtime.DefaultURL == "" || runtime.SourceRevision == "" || runtime.WeightFiles <= 0 || len(runtime.Versions) == 0 {
				return fmt.Errorf("models: %s has an incomplete local song runtime", model.ID)
			}
			for _, digest := range []string{runtime.CatalogSHA256, runtime.ArtifactSHA256} {
				if decoded, err := hex.DecodeString(digest); err != nil || len(decoded) != sha256.Size { return fmt.Errorf("models: %s has an invalid runtime digest", model.ID) }
			}
			for _, value := range []float64{runtime.MinDuration,runtime.MaxDuration,runtime.DefaultGuidance,runtime.MinGuidance,runtime.MaxGuidance} {
				if math.IsNaN(value) || math.IsInf(value,0) { return fmt.Errorf("models: %s has a nonfinite runtime control",model.ID) }
			}
			if runtime.MinDuration < 1 || runtime.MaxDuration < runtime.MinDuration || runtime.MaxDuration > 240 || runtime.MinSteps < 0 || runtime.DefaultSteps < runtime.MinSteps || runtime.DefaultSteps > runtime.MaxSteps || runtime.MaxSteps > 100 || runtime.DefaultGuidance < runtime.MinGuidance || runtime.DefaultGuidance > runtime.MaxGuidance || runtime.MinGuidance <= 0 || runtime.MaxGuidance > 20 || runtime.TimeoutSeconds < 1 || runtime.TimeoutSeconds > 86400 {
				return fmt.Errorf("models: %s has inconsistent runtime controls",model.ID)
			}
		}
	}
	return nil
}

func (r *Registry) ResolveFor(id string, host Host) (Model, error) {
	m, err := r.Resolve(id)
	if err != nil {
		return Model{}, err
	}
	if m.Hardware.MinVRAMBytes > 0 && host.FreeVRAMBytes < m.Hardware.MinVRAMBytes {
		return Model{}, fmt.Errorf("models: %s needs %d free VRAM bytes, host has %d", id, m.Hardware.MinVRAMBytes, host.FreeVRAMBytes)
	}
	if m.Hardware.MinRAMBytes > 0 && host.FreeRAMBytes < m.Hardware.MinRAMBytes {
		return Model{}, fmt.Errorf("models: %s needs %d free RAM bytes, host has %d", id, m.Hardware.MinRAMBytes, host.FreeRAMBytes)
	}
	if host.OS != "" && host.OS != runtime.GOOS {
		return Model{}, fmt.Errorf("models: %s is not supported on %s", id, host.OS)
	}
	return m, nil
}

func VerifyChecksum(r io.Reader, expected string) (uint64, error) {
	h := sha256.New()
	n, err := io.Copy(h, r)
	if err != nil {
		return uint64(n), err
	}
	actual := hex.EncodeToString(h.Sum(nil))
	if expected == "" || actual != expected {
		return uint64(n), fmt.Errorf("models: checksum mismatch: got %s want %s", actual, expected)
	}
	return uint64(n), nil
}

func CheckDisk(free, download, floor uint64) error {
	if free < download+floor {
		return fmt.Errorf("models: insufficient free disk: need %d bytes, have %d", download+floor, free)
	}
	return nil
}
