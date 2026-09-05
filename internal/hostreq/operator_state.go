package hostreq

import (
	"context"
	"fmt"
	"strings"

	"github.com/vrooli/api-core/trustposture"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/operatorstate"
)

const operatorStateFileName = ".vrooli/operator-state.json"

type operatorStateEntry struct {
	OptedIn *bool          `json:"opted_in"`
	Config  map[string]any `json:"config"`
}

// OperatorState is the validated, read-only view of the mutable operator
// choice document. A missing file is a valid empty state; malformed state is
// an error so setup cannot silently fall back to a different policy.
type OperatorState struct {
	hostTools      map[string]operatorStateEntry
	hostSafeguards map[string]operatorStateEntry
	trustPosture   trustposture.State
}

// LoadOperatorState loads the project's durable operator choices. The file is
// intentionally read at the resolver boundary so every caller sees the same
// typed distinction between opted in, declined, and not recorded.
func LoadOperatorState(root string) (OperatorState, error) {
	document, err := operatorstate.New(operatorstate.Config{RepoRoot: root}).Load(context.Background())
	if err != nil {
		return OperatorState{}, fmt.Errorf("load operator state: %w", err)
	}
	posture := trustposture.State{Posture: trustposture.Personal, Source: "default"}
	if strings.TrimSpace(document.TrustPosture) != "" {
		posture.Posture = trustposture.Posture(document.TrustPosture)
		posture.Source = "operator-state"
	}
	tools := make(map[string]operatorStateEntry, len(document.HostTools))
	for name, choice := range document.HostTools {
		tools[name] = operatorStateEntry{OptedIn: choice.OptedIn, Config: choice.Config}
	}
	safeguards := make(map[string]operatorStateEntry, len(document.HostSafeguards))
	for name, choice := range document.HostSafeguards {
		safeguards[name] = operatorStateEntry{OptedIn: choice.OptedIn, Config: choice.Config}
	}
	return OperatorState{
		hostTools:      tools,
		hostSafeguards: safeguards,
		trustPosture:   posture,
	}, nil
}

// TrustPosture returns the immutable, typed posture and the source selected by
// the operator-state reader. Callers cannot mutate the durable state through
// this value.
func (s OperatorState) TrustPosture() trustposture.State {
	return s.trustPosture
}

func (s OperatorState) choice(kind hostreqspec.Kind, name string) hostreqspec.OperatorChoice {
	var entries map[string]operatorStateEntry
	if kind == hostreqspec.KindSafeguard {
		entries = s.hostSafeguards
	} else {
		entries = s.hostTools
	}
	entry, ok := entries[strings.TrimSpace(name)]
	if !ok || entry.OptedIn == nil {
		return hostreqspec.OperatorChoiceNotRecorded
	}
	if *entry.OptedIn {
		return hostreqspec.OperatorChoiceOptedIn
	}
	return hostreqspec.OperatorChoiceDeclined
}

func (s OperatorState) config(kind hostreqspec.Kind, name string) map[string]any {
	var entries map[string]operatorStateEntry
	if kind == hostreqspec.KindSafeguard {
		entries = s.hostSafeguards
	} else {
		entries = s.hostTools
	}
	entry, ok := entries[strings.TrimSpace(name)]
	if !ok || len(entry.Config) == 0 {
		return nil
	}
	return cloneConfig(entry.Config)
}

func cloneConfig(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
