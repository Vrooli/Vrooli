package models

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"testing"
)

// TestArchitectureTechniquesMirrorPython asserts the Go architecture→technique
// derivation table (architecture.go) and the Python mirror (_diffusers.ARCHITECTURES)
// agree on the {op, technique, ready} triple per architecture — so capability
// derivation cannot drift between the Go selector and the Python runtime. Caveat
// prose is Go-only and excluded from the contract.
func TestArchitectureTechniquesMirrorPython(t *testing.T) {
	py := python3(t)
	root := sidecarPYPath(t)
	cmd := exec.Command(py, "-c", "import json,image_tools_sidecar._diffusers as d; print(json.dumps(d.ARCHITECTURES))")
	cmd.Env = append(os.Environ(), "PYTHONPATH="+root)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("dump python ARCHITECTURES: %v", err)
	}
	type row struct {
		Op        string `json:"op"`
		Technique string `json:"technique"`
		Ready     bool   `json:"ready"`
	}
	var pyArch map[string][]row
	if err := json.Unmarshal(out, &pyArch); err != nil {
		t.Fatalf("decode python ARCHITECTURES %q: %v", out, err)
	}

	// Build the comparable Go view: only architectures that derive something.
	goArch := map[string][]row{}
	for arch := range architectures {
		ds := DerivableTechniques(arch)
		if len(ds) == 0 {
			continue
		}
		rows := make([]row, 0, len(ds))
		for _, d := range ds {
			rows = append(rows, row{Op: d.Op, Technique: d.Technique, Ready: d.Ready})
		}
		goArch[string(arch)] = rows
	}

	norm := func(m map[string][]row) string {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var b strings.Builder
		for _, k := range keys {
			rs := append([]row(nil), m[k]...)
			sort.Slice(rs, func(i, j int) bool { return rs[i].Op < rs[j].Op })
			fmt.Fprintf(&b, "%s:%v\n", k, rs)
		}
		return b.String()
	}
	if g, p := norm(goArch), norm(pyArch); g != p {
		t.Fatalf("architecture→technique table drift:\n--- go ---\n%s--- python ---\n%s", g, p)
	}
}

// TestEffectiveOpsGolden snapshots the derived capability matrix for every seed
// model: its native ops, its derived ops, and each derived op's offerable state.
// A new checkpoint or a derivation Ready-flip is a reviewed diff here, with the
// capability coverage computed (not hand-listed). The golden is asserted by
// structural invariants + the motivating cases rather than a brittle full-string
// dump, so unrelated seed edits don't churn it.
func TestEffectiveOpsGolden(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	for _, m := range r.Models() {
		eos := m.EffectiveOps()
		// Invariant 1: every declared op appears exactly once as native.
		nativeCount := map[string]int{}
		for _, eo := range eos {
			if eo.Support == "native" {
				nativeCount[eo.Op]++
			}
		}
		for _, op := range m.Operations {
			if nativeCount[op] != 1 {
				t.Errorf("model %s: declared op %q should appear once as native, got %d", m.ID, op, nativeCount[op])
			}
		}
		// Invariant 2: derived ops are never also native, carry a technique + caveat,
		// and are offerable IFF their derivation is proven (Ready). Offerable() must
		// track Ready exactly — the no-vaporware gate (an unproven derivation is
		// reported for honest surfacing but never selected).
		for _, eo := range eos {
			if eo.Support != "derived" {
				continue
			}
			if nativeCount[eo.Op] > 0 {
				t.Errorf("model %s: op %q is both native and derived", m.ID, eo.Op)
			}
			if eo.Technique == "" || eo.Caveat == "" {
				t.Errorf("model %s: derived op %q must name a technique + caveat, got %q/%q", m.ID, eo.Op, eo.Technique, eo.Caveat)
			}
			if eo.Offerable() != eo.Ready {
				t.Errorf("model %s: derived op %q offerable=%v but ready=%v (must match)", m.ID, eo.Op, eo.Offerable(), eo.Ready)
			}
		}
	}

	// Motivating case: a base SDXL txt2img checkpoint derives the image-conditioned
	// ops (image_to_image / inpaint / outpaint / edit_instruct), each derived+unproven.
	sdxl, ok := r.ByID("sdxl-1.0")
	if !ok {
		t.Fatal("seed missing sdxl-1.0")
	}
	want := map[string]string{ // op -> support
		"text_to_image":  "native",
		"image_to_image": "native", // sdxl-1.0 declares img2img natively
		"inpaint":        "derived",
		"outpaint":       "derived",
		"edit_instruct":  "derived",
	}
	got := map[string]string{}
	for _, eo := range sdxl.EffectiveOps() {
		got[eo.Op] = eo.Support
	}
	for op, sup := range want {
		if got[op] != sup {
			t.Errorf("sdxl-1.0 op %q: support=%q want %q (full=%v)", op, got[op], sup, got)
		}
	}
	// sdxl-1.0 now SERVES inpaint via the proven sdxl diffusers-inpaint derivation
	// (StableDiffusionXLInpaintPipeline). outpaint / edit_instruct stay unproven and
	// unserved (offerable gate honest).
	if !sdxl.ServesOperation("inpaint") {
		t.Error("sdxl-1.0 must serve inpaint via the proven sdxl diffusers-inpaint derivation")
	}
	for _, op := range []string{"outpaint", "edit_instruct"} {
		if sdxl.ServesOperation(op) {
			t.Errorf("sdxl-1.0 must not yet serve derived op %q (unproven)", op)
		}
	}

	// Proven derivation: sd-1.5 (architecture sd15) now SERVES inpaint through the
	// proven diffusers-inpaint technique — the motivating "a base checkpoint gains a
	// derived capability" outcome. Its declared image_to_image stays native; the
	// still-unproven outpaint / edit_instruct derivations remain unserved.
	sd15m, ok := r.ByID("sd-1.5")
	if !ok {
		t.Fatal("seed missing sd-1.5")
	}
	if !sd15m.ServesOperation("inpaint") {
		t.Error("sd-1.5 must serve inpaint via the proven sd15 diffusers-inpaint derivation")
	}
	for _, eo := range sd15m.EffectiveOps() {
		if eo.Op == "inpaint" && (eo.Support != "derived" || !eo.Ready) {
			t.Errorf("sd-1.5 inpaint: support=%q ready=%v, want derived+ready", eo.Support, eo.Ready)
		}
	}
	for _, op := range []string{"outpaint", "edit_instruct"} {
		if sd15m.ServesOperation(op) {
			t.Errorf("sd-1.5 must not yet serve unproven derived op %q", op)
		}
	}
}
