package quint

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"react-vite-temporal-model/internal/model"
)

type ArtifactTrace struct {
	Name    string              `json:"name"`
	Initial string              `json:"initial"`
	Steps   []ArtifactTraceStep `json:"steps"`
}

type ArtifactTraceStep struct {
	Event     string `json:"event"`
	Want      string `json:"want"`
	WantError bool   `json:"wantError"`
}

func NormalizeTraces(flow model.Flow, dir string) ([]ArtifactTrace, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	files := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)
	stateByQuint := map[string]string{}
	for _, state := range flow.States {
		stateByQuint[state.Quint] = state.ID
	}
	eventByQuint := map[string]string{}
	for _, event := range flow.Events {
		eventByQuint[event.Quint] = event.ID
	}
	traces := make([]ArtifactTrace, 0, len(files))
	for i, file := range files {
		data, err := os.ReadFile(filepath.Join(dir, file))
		if err != nil {
			return nil, err
		}
		var raw struct {
			States []struct {
				Status   tagged `json:"status"`
				Event    tagged `json:"event"`
				Rejected bool   `json:"rejected"`
			} `json:"states"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("parse %s: %w", file, err)
		}
		if len(raw.States) == 0 {
			return nil, fmt.Errorf("%s contains no ITF states", file)
		}
		initial, ok := stateByQuint[raw.States[0].Status.Tag]
		if !ok {
			return nil, fmt.Errorf("%s contains unknown status tag %q", file, raw.States[0].Status.Tag)
		}
		steps := make([]ArtifactTraceStep, 0, len(raw.States)-1)
		for _, state := range raw.States[1:] {
			event, ok := eventByQuint[state.Event.Tag]
			if !ok {
				return nil, fmt.Errorf("%s contains unknown event tag %q", file, state.Event.Tag)
			}
			want, ok := stateByQuint[state.Status.Tag]
			if !ok {
				return nil, fmt.Errorf("%s contains unknown status tag %q", file, state.Status.Tag)
			}
			steps = append(steps, ArtifactTraceStep{Event: event, Want: want, WantError: state.Rejected})
		}
		traces = append(traces, ArtifactTrace{
			Name:    fmt.Sprintf("generated_model_%03d", i+1),
			Initial: initial,
			Steps:   steps,
		})
	}
	return traces, nil
}

type tagged struct {
	Tag string `json:"tag"`
}
