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
	"runtime"
	"sort"
)

type LicenceLane string

const (
	LicencePermissive LicenceLane = "permissive"
	LicenceRestricted LicenceLane = "restricted"
)

type Model struct {
	ID                 string      `json:"id"`
	DisplayName        string      `json:"display_name"`
	Variant            string      `json:"variant"`
	Revision           string      `json:"revision"`
	Licence            LicenceLane `json:"licence"`
	Installed          bool        `json:"installed"`
	Bytes              uint64      `json:"bytes"`
	Operations         []string    `json:"operations"`
	Hardware           Hardware    `json:"hardware"`
	License            string      `json:"license"`
	CommercialUse      string      `json:"commercial_use"`
	CommercialUseNotes string      `json:"commercial_use_notes"`
	BaseModelLineage   string      `json:"base_model_lineage"`
	KnownRisks         string      `json:"known_risks"`
	Checksum           string      `json:"checksum"`
	DiskFloorBytes     uint64      `json:"disk_floor_bytes"`
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
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
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
