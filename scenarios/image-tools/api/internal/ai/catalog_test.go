package ai

import "testing"

func TestCatalog_BreadthOpsRegistered(t *testing.T) {
	cases := []struct {
		op       string
		category Category
		reqImage bool
		reqMask  bool
		prompt   bool
	}{
		{"outpaint", CategoryGeneration, true, true, true},
		{"background_replace", CategoryGeneration, true, true, true},
		{"colorize", CategoryEnhancement, true, false, false},
		{"depth_map", CategoryEnhancement, true, false, false},
	}
	for _, c := range cases {
		op, ok := Get(c.op)
		if !ok {
			t.Errorf("op %q not in catalog", c.op)
			continue
		}
		if op.Category != c.category {
			t.Errorf("%q category = %q, want %q", c.op, op.Category, c.category)
		}
		if op.RequiresImage != c.reqImage || op.RequiresMask != c.reqMask || op.PromptDriven != c.prompt {
			t.Errorf("%q flags = img:%t mask:%t prompt:%t, want img:%t mask:%t prompt:%t",
				c.op, op.RequiresImage, op.RequiresMask, op.PromptDriven, c.reqImage, c.reqMask, c.prompt)
		}
	}
}

func TestBuildRunners_CoversBreadthOps(t *testing.T) {
	// Every catalog op (incl. the new breadth ops) must get a runner so the
	// dispatcher can execute it.
	eng := &Engine{}
	runners := eng.BuildRunners()
	for _, op := range []string{"outpaint", "background_replace", "colorize", "depth_map"} {
		if _, ok := runners[op]; !ok {
			t.Errorf("BuildRunners missing a runner for %q", op)
		}
	}
}
