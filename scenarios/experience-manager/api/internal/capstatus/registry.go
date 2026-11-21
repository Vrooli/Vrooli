// Package capstatus reads the experience capability registry from disk and
// derives each capability's status from what the reconciler can actually do.
//
// The registry files never record status — that is doctrine, not convenience.
// A hand-written `provable: true` goes stale the moment a checker regresses and
// invites an agent to declare success. Everything in this package is computed.
package capstatus

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Registry is the parsed capability registry.
type Registry struct {
	Axes         map[string]Axis
	Evidence     map[string]struct{}
	Capabilities []Capability
}

// Axis is one capture dimension and the values it declares.
type Axis struct {
	ID     string
	Values map[string]struct{}
}

// Capability is one registry entry. Port-only entries carry no Proves block.
type Capability struct {
	ID              string
	Title           string
	Group           string
	Facets          []string
	TierCeiling     string
	Proves          *Proves
	IsPort          bool
	PortSatisfiedBy []string
}

// Proves is what it takes to demonstrate a promise.
type Proves struct {
	Axes       map[string][]string
	Evidence   []string
	ClaimTypes []string
}

type rawDoc struct {
	Kind string `json:"kind"`
	Axes []struct {
		ID     string `json:"id"`
		Values []struct {
			ID string `json:"id"`
		} `json:"values"`
	} `json:"axes"`
	Evidence []struct {
		ID string `json:"id"`
	} `json:"evidence"`
	Group struct {
		ID string `json:"id"`
	} `json:"group"`
	Capabilities []struct {
		ID          string   `json:"id"`
		Title       string   `json:"title"`
		Facets      []string `json:"facets"`
		TierCeiling string   `json:"tierCeiling"`
		Proves      *struct {
			Axes       map[string][]string `json:"axes"`
			Evidence   []string            `json:"evidence"`
			ClaimTypes []string            `json:"claimTypes"`
		} `json:"proves"`
		Port *struct {
			SatisfiedBy []string `json:"satisfiedBy"`
		} `json:"port"`
	} `json:"capabilities"`
}

// Load reads every document under the capabilities directory. Order is stable so
// downstream reports diff cleanly.
func Load(dir string) (Registry, error) {
	reg := Registry{Axes: map[string]Axis{}, Evidence: map[string]struct{}{}}
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return reg, fmt.Errorf("scan %s: %w", dir, err)
	}
	nested, err := filepath.Glob(filepath.Join(dir, "capabilities", "*.json"))
	if err != nil {
		return reg, fmt.Errorf("scan %s/capabilities: %w", dir, err)
	}
	paths = append(paths, nested...)
	sort.Strings(paths)

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return reg, fmt.Errorf("read %s: %w", path, err)
		}
		var doc rawDoc
		if err := json.Unmarshal(data, &doc); err != nil {
			return reg, fmt.Errorf("parse %s: %w", path, err)
		}
		switch doc.Kind {
		case "axis-registry":
			for _, a := range doc.Axes {
				axis := Axis{ID: a.ID, Values: map[string]struct{}{}}
				for _, v := range a.Values {
					axis.Values[v.ID] = struct{}{}
				}
				reg.Axes[a.ID] = axis
			}
		case "evidence-registry":
			for _, e := range doc.Evidence {
				reg.Evidence[e.ID] = struct{}{}
			}
		case "capability-group":
			for _, c := range doc.Capabilities {
				cap := Capability{
					ID: c.ID, Title: c.Title, Group: doc.Group.ID,
					Facets: c.Facets, TierCeiling: c.TierCeiling,
				}
				for _, f := range c.Facets {
					if f == "port" {
						cap.IsPort = true
					}
				}
				if c.Proves != nil {
					cap.Proves = &Proves{
						Axes: c.Proves.Axes, Evidence: c.Proves.Evidence,
						ClaimTypes: c.Proves.ClaimTypes,
					}
				}
				if c.Port != nil {
					cap.PortSatisfiedBy = c.Port.SatisfiedBy
				}
				reg.Capabilities = append(reg.Capabilities, cap)
			}
		}
	}
	sort.Slice(reg.Capabilities, func(i, j int) bool {
		return reg.Capabilities[i].ID < reg.Capabilities[j].ID
	})
	return reg, nil
}
