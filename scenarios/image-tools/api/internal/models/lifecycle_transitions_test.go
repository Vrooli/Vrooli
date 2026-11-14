package models

import "testing"

func TestLifecycleTransitionMatrix(t *testing.T) {
	states := []LifecycleState{"", ModelAbsent, ModelInstalling, ModelInstalled, ModelBuilding, ModelSmoking, ModelReady, ModelStale, ModelFailed, "unknown"}
	allowed := map[LifecycleState]map[LifecycleState]bool{
		"":              {ModelInstalling: true, ModelAbsent: true},
		ModelAbsent:     {ModelInstalling: true, ModelAbsent: true},
		ModelInstalling: {ModelInstalled: true, ModelFailed: true},
		ModelInstalled:  {ModelInstalling: true, ModelBuilding: true, ModelSmoking: true, ModelReady: true, ModelStale: true, ModelFailed: true, ModelAbsent: true},
		ModelBuilding:   {ModelInstalled: true, ModelSmoking: true, ModelReady: true, ModelFailed: true},
		ModelSmoking:    {ModelInstalled: true, ModelReady: true, ModelStale: true, ModelFailed: true},
		ModelReady:      {ModelInstalling: true, ModelStale: true, ModelAbsent: true},
		ModelStale:      {ModelInstalling: true, ModelAbsent: true},
		ModelFailed:     {ModelInstalling: true, ModelAbsent: true},
	}

	for _, from := range states {
		for _, to := range states {
			t.Run(string(from)+"_to_"+string(to), func(t *testing.T) {
				_, err := (ModelLifecycle{ModelID: "test", State: from}).Transition(to)
				want := allowed[from][to]
				if (err == nil) != want {
					t.Fatalf("transition %q -> %q error = %v, allowed = %t", from, to, err, want)
				}
			})
		}
	}
}
