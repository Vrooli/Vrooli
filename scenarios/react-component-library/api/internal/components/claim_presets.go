package components

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExperienceKind is the contract vocabulary used by new library assets.
type ExperienceKind string

const (
	ExperienceKindControl ExperienceKind = "control"
	ExperienceKindInput   ExperienceKind = "input"
	ExperienceKindSurface ExperienceKind = "surface"
	ExperienceKindOverlay ExperienceKind = "overlay"
	ExperienceKindShell   ExperienceKind = "shell"
)

var experienceClaimPresets = map[ExperienceKind][]string{
	ExperienceKindControl: {"tap-target-size", "content-not-clipped", "state-contrast", "padding", "keyboard-reachable", "size-parity"},
	ExperienceKindInput:   {"accessible-name", "error-association", "state-contrast", "keyboard-reachable", "font-size"},
	ExperienceKindSurface: {"no-document-horizontal-overflow", "spacing", "heading-hierarchy"},
	ExperienceKindOverlay: {"focus-contained", "layered-dismissal", "focus-restored", "reading-order"},
	ExperienceKindShell:   {"chrome-pinned", "viewport-fill", "safe-area-tap-targets", "no-document-horizontal-overflow"},
}

// ExperienceClaimPreset returns a copy so callers cannot mutate the registry.
func ExperienceClaimPreset(kind string) []string {
	claims := experienceClaimPresets[ExperienceKind(strings.TrimSpace(kind))]
	return append([]string(nil), claims...)
}

func experienceKindOrDefault(kind string) string {
	kind = strings.TrimSpace(kind)
	if len(ExperienceClaimPreset(kind)) > 0 {
		return kind
	}
	return string(ExperienceKindControl)
}

type scaffoldExperienceContract struct {
	Contract  scaffoldContractHeader    `json:"contract"`
	Component scaffoldContractComponent `json:"component"`
	States    []scaffoldContractState   `json:"states"`
	Elements  []scaffoldContractElement `json:"elements"`
	Claims    []scaffoldContractClaim   `json:"claims"`
	Bindings  scaffoldContractBindings  `json:"bindings"`
}

type scaffoldContractHeader struct {
	Kind       string `json:"kind"`
	Schema     string `json:"schema"`
	Provenance string `json:"provenance"`
}

type scaffoldContractComponent struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Purpose string `json:"purpose"`
}

type scaffoldContractState struct {
	ID      string `json:"id"`
	Example string `json:"example"`
}

type scaffoldContractElement struct {
	ID   string `json:"id"`
	Role string `json:"role"`
}

type scaffoldContractClaim struct {
	ID        string   `json:"id"`
	Type      string   `json:"type"`
	Statement string   `json:"statement"`
	Tier      string   `json:"tier"`
	Elements  []string `json:"elements,omitempty"`
}

type scaffoldContractBindings struct {
	Elements map[string]map[string]string `json:"elements"`
}

func scaffoldExperienceContractFile(versionDir, kind, libraryID, displayName, version string) error {
	path := filepath.Join(versionDir, "experience-contract.json")
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat experience contract scaffold %q: %w", path, err)
	}
	claims := ExperienceClaimPreset(kind)
	if len(claims) == 0 {
		return fmt.Errorf("unknown experience contract kind %q", kind)
	}
	contractClaims := make([]scaffoldContractClaim, 0, len(claims))
	for _, claimType := range claims {
		contractClaims = append(contractClaims, scaffoldContractClaim{
			ID: claimType, Type: claimType,
			Statement: fmt.Sprintf("The %s example satisfies the %s contract claim.", displayName, claimType),
			Tier:      "machine", Elements: []string{"control"},
		})
	}
	document := scaffoldExperienceContract{
		Contract:  scaffoldContractHeader{Kind: "rcl-component-experience-contract", Schema: "scenario-experience-spec/v1", Provenance: fmt.Sprintf("%s@%s", libraryID, version)},
		Component: scaffoldContractComponent{ID: strings.ToLower(strings.ReplaceAll(libraryID, "react-component-library:", "")), Title: displayName, Purpose: "Kind-based experience contract scaffold."},
		States:    []scaffoldContractState{{ID: "default", Example: "default"}},
		Elements:  []scaffoldContractElement{{ID: "control", Role: "generic"}},
		Claims:    contractClaims,
		Bindings:  scaffoldContractBindings{Elements: map[string]map[string]string{"control": {"selector": "[data-testid]"}}},
	}
	raw, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return fmt.Errorf("encode experience contract scaffold: %w", err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
		return fmt.Errorf("write experience contract scaffold %q: %w", path, err)
	}
	return nil
}
