package ai

import "testing"

func TestCatalog_BreadthOpsRegistered(t *testing.T) {
	cases := []struct {
		op       string
		family   string
		reqImage bool
		reqMask  bool
		prompt   bool
	}{
		{"outpaint", "edit", true, true, true},
		{"background_replace", "edit", true, true, true},
		{"colorize", "enhance", true, false, false},
		{"depth_map", "extract", true, false, false},
		{"deblur", "restore", true, false, false},
		{"face_restore", "restore", true, false, false},
		{"old_photo_restore", "restore", true, false, false},
		{"caption", "extract", true, false, false},
	}
	for _, c := range cases {
		op, ok := Get(c.op)
		if !ok {
			t.Errorf("op %q not in catalog", c.op)
			continue
		}
		if op.Family != c.family {
			t.Errorf("%q family = %q, want %q", c.op, op.Family, c.family)
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
	for _, op := range []string{"outpaint", "background_replace", "colorize", "depth_map", "deblur", "face_restore", "old_photo_restore", "caption"} {
		if _, ok := runners[op]; !ok {
			t.Errorf("BuildRunners missing a runner for %q", op)
		}
	}
}
