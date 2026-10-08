package depgraph

import (
	"reflect"
	"testing"
)

func satisfiedSet(keys ...string) func(string) bool {
	set := make(map[string]bool, len(keys))
	for _, k := range keys {
		set[k] = true
	}
	return func(key string) bool { return set[key] }
}

func TestWaves(t *testing.T) {
	tests := []struct {
		name      string
		graph     map[string][]string
		satisfied func(string) bool
		wantWaves map[string]int
		wantMax   int
		wantCycle int // number of distinct cycles reported
	}{
		{
			name:      "empty graph",
			graph:     map[string][]string{},
			wantWaves: map[string]int{},
			wantMax:   -1,
		},
		{
			name:      "single node no deps is wave zero",
			graph:     map[string][]string{"a": nil},
			wantWaves: map[string]int{"a": 0},
			wantMax:   0,
		},
		{
			name: "linear chain peels one layer per wave",
			graph: map[string][]string{
				"a": nil,
				"b": {"a"},
				"c": {"b"},
			},
			wantWaves: map[string]int{"a": 0, "b": 1, "c": 2},
			wantMax:   2,
		},
		{
			name: "diamond joins at max dependency wave",
			graph: map[string][]string{
				"root":  nil,
				"left":  {"root"},
				"right": {"root"},
				"join":  {"left", "right"},
			},
			wantWaves: map[string]int{"root": 0, "left": 1, "right": 1, "join": 2},
			wantMax:   2,
		},
		{
			name: "satisfied dep collapses waves",
			graph: map[string][]string{
				"a": nil,
				"b": {"a"},
				"c": {"b"},
			},
			satisfied: satisfiedSet("a", "b"),
			wantWaves: map[string]int{"c": 0},
			wantMax:   0,
		},
		{
			name: "satisfied node excluded from wave assignment",
			graph: map[string][]string{
				"done":    nil,
				"pending": {"done"},
			},
			satisfied: satisfiedSet("done"),
			wantWaves: map[string]int{"pending": 0},
			wantMax:   0,
		},
		{
			name: "unknown dep is fail-open",
			graph: map[string][]string{
				"a": {"ghost"},
			},
			wantWaves: map[string]int{"a": 0},
			wantMax:   0,
		},
		{
			name: "cycle members get CycleWave",
			graph: map[string][]string{
				"a": {"b"},
				"b": {"a"},
				"c": nil,
			},
			wantWaves: map[string]int{"a": CycleWave, "b": CycleWave, "c": 0},
			wantMax:   0,
			wantCycle: 1,
		},
		{
			name: "node downstream of cycle also trapped",
			graph: map[string][]string{
				"a": {"b"},
				"b": {"a"},
				"c": {"a"},
			},
			wantWaves: map[string]int{"a": CycleWave, "b": CycleWave, "c": CycleWave},
			wantMax:   -1,
			wantCycle: 1,
		},
		{
			name: "two independent cycles reported once each",
			graph: map[string][]string{
				"a": {"b"},
				"b": {"a"},
				"x": {"y"},
				"y": {"x"},
			},
			wantWaves: map[string]int{"a": CycleWave, "b": CycleWave, "x": CycleWave, "y": CycleWave},
			wantMax:   -1,
			wantCycle: 2,
		},
		{
			name: "satisfied node breaks its cycle",
			graph: map[string][]string{
				"a": {"b"},
				"b": {"a"},
			},
			satisfied: satisfiedSet("a"),
			wantWaves: map[string]int{"b": 0},
			wantMax:   0,
		},
		{
			name: "wide backlog mixes waves and cycles",
			graph: map[string][]string{
				"done-root": nil,
				"ready":     {"done-root"},
				"later":     {"ready"},
				"deep":      {"later"},
				"loop1":     {"loop2"},
				"loop2":     {"loop1"},
			},
			satisfied: satisfiedSet("done-root"),
			wantWaves: map[string]int{
				"ready": 0,
				"later": 1,
				"deep":  2,
				"loop1": CycleWave,
				"loop2": CycleWave,
			},
			wantMax:   2,
			wantCycle: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Waves(tt.graph, tt.satisfied)
			if !reflect.DeepEqual(got.Waves, tt.wantWaves) {
				t.Errorf("Waves = %v, want %v", got.Waves, tt.wantWaves)
			}
			if got.MaxWave != tt.wantMax {
				t.Errorf("MaxWave = %d, want %d", got.MaxWave, tt.wantMax)
			}
			if len(got.Cycles) != tt.wantCycle {
				t.Errorf("Cycles = %v, want %d distinct", got.Cycles, tt.wantCycle)
			}
		})
	}
}

func TestWaves_NilPredicate(t *testing.T) {
	got := Waves(map[string][]string{"a": nil}, nil)
	if got.Waves["a"] != 0 {
		t.Errorf("nil predicate should treat all nodes as unsatisfied, got %v", got.Waves)
	}
}
