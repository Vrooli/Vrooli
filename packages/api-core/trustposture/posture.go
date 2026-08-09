// Package trustposture contains the per-install trust stance shared by the
// control plane and scenarios. A posture selects operational defaults; it
// never selects whether an identity is verified.
package trustposture

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const stateRelativePath = ".vrooli/operator-state.json"

// Posture is the operator-selected trust stance for one installation.
type Posture string

const (
	Personal Posture = "personal"
	Shared   Posture = "shared"
	Hosted   Posture = "hosted"
)

// Defaults are the values selected by a posture. Verification remains
// mandatory for every row.
type Defaults struct {
	AccessTokenTTL      time.Duration
	BreakGlassAvailable bool
	BreakGlassTTL       time.Duration
	NodeExecutionScopes []string
	JWKSCacheGrace      time.Duration
}

var postureDefaults = map[Posture]Defaults{
	Personal: {
		AccessTokenTTL:      60 * time.Minute,
		BreakGlassAvailable: true,
		BreakGlassTTL:       15 * time.Minute,
		NodeExecutionScopes: []string{"vrooli-bridge:read", "vrooli-bridge:write"},
		JWKSCacheGrace:      24 * time.Hour,
	},
	Shared: {
		AccessTokenTTL:      15 * time.Minute,
		BreakGlassAvailable: true,
		BreakGlassTTL:       10 * time.Minute,
		NodeExecutionScopes: []string{},
		JWKSCacheGrace:      4 * time.Hour,
	},
	Hosted: {
		AccessTokenTTL:      10 * time.Minute,
		BreakGlassAvailable: false,
		BreakGlassTTL:       0,
		NodeExecutionScopes: []string{},
		JWKSCacheGrace:      time.Hour,
	},
}

// DefaultsFor returns a defensive copy of the defaults for p.
func DefaultsFor(p Posture) (Defaults, error) {
	d, ok := postureDefaults[p]
	if !ok {
		return Defaults{}, fmt.Errorf("trust posture %q: %w", p, ErrInvalidPosture)
	}
	d.NodeExecutionScopes = append([]string(nil), d.NodeExecutionScopes...)
	return d, nil
}

// Table returns the complete policy table in deterministic order for docs and
// diagnostics without exposing mutable package state.
func Table() map[Posture]Defaults {
	out := make(map[Posture]Defaults, len(postureDefaults))
	for p := range postureDefaults {
		out[p], _ = DefaultsFor(p)
	}
	return out
}

var ErrInvalidPosture = errors.New("invalid trust posture")

// State is the typed reader result. Source is either the state file path or
// "default" when no operator-state file exists.
type State struct {
	Posture Posture
	Source  string
}

// TransitionEvent is the typed audit payload an operator workflow records
// when changing posture. Runtime readers expose no write operation, so agents
// cannot manufacture or widen this event.
type TransitionEvent struct {
	Action string
	Actor  string
	From   Posture
	To     Posture
	At     time.Time
}

// Transition validates an operator posture change and returns its typed audit
// payload. Persistence and audit transport remain with the operator workflow.
func Transition(from State, to Posture, actor string, at time.Time) (TransitionEvent, error) {
	if _, err := DefaultsFor(to); err != nil {
		return TransitionEvent{}, err
	}
	if strings.TrimSpace(actor) == "" {
		return TransitionEvent{}, errors.New("trust posture transition: operator actor is required")
	}
	if from.Posture == to {
		return TransitionEvent{}, errors.New("trust posture transition: posture is unchanged")
	}
	if at.IsZero() {
		return TransitionEvent{}, errors.New("trust posture transition: timestamp is required")
	}
	return TransitionEvent{Action: "trust_posture.transition", Actor: actor, From: from.Posture, To: to, At: at.UTC()}, nil
}

// Parse reads only the posture field from an operator-state document. The
// complete document is validated by the operator-state schema; this parser is
// the small runtime seam used by agents and scenarios.
func Parse(data []byte, source string) (State, error) {
	var doc struct {
		TrustPosture string `json:"trust_posture"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return State{}, fmt.Errorf("decode operator state: %w", err)
	}
	p := Posture(strings.TrimSpace(doc.TrustPosture))
	if p == "" {
		p = Personal
	}
	if _, err := DefaultsFor(p); err != nil {
		return State{}, fmt.Errorf("operator state %s: %w", source, err)
	}
	if strings.TrimSpace(source) == "" {
		source = "default"
	}
	return State{Posture: p, Source: source}, nil
}

// Load loads the state rooted at root. A missing file is the documented
// personal default; malformed or invalid state is an error and never silently
// changes the security stance.
func Load(root string) (State, error) {
	path := filepath.Join(root, stateRelativePath)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return State{Posture: Personal, Source: "default"}, nil
	}
	if err != nil {
		return State{}, fmt.Errorf("read operator state %s: %w", path, err)
	}
	return Parse(data, path)
}

// LoadWorkingTree walks upward from the current directory to find the
// repository's .vrooli directory. This keeps scenario processes independent
// of the lifecycle-selected working directory.
func LoadWorkingTree() (State, error) {
	working, err := os.Getwd()
	if err != nil {
		return State{}, fmt.Errorf("get working directory: %w", err)
	}
	for dir := working; ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, stateRelativePath)); err == nil {
			return Load(dir)
		} else if !errors.Is(err, os.ErrNotExist) {
			return State{}, err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return State{Posture: Personal, Source: "default"}, nil
		}
	}
}
