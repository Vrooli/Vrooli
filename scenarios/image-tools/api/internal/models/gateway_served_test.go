package models

import (
	"strings"
	"testing"

	"image-tools/internal/capabilities"
)

func gatewayRow(mut func(*Model)) Model {
	m := Model{
		ID: "vlm", Name: "VLM", Operations: []string{"caption", "object_detection", "tagging"}, Tier: TierQuality,
		Backend: BackendGatewayVision, Hardware: Hardware{CPUCapable: true},
		Runtime:          Runtime{GatewayRole: "vision.default", BoxOrder: BoxOrderXYXY, BoxScale: 1000},
		CapabilityLabels: CapabilityLabels{CommercialUse: CommercialUseYes},
		Enabled:          true,
	}
	mut(&m)
	return m
}

// A general vision model is one registry row. The contract that keeps it that
// way: the row names its gateway role, serves only vision operations, and says
// how it writes boxes.
func TestGatewayServedRowContract(t *testing.T) {
	reg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for name, tc := range map[string]struct {
		mut  func(*Model)
		want string
	}{
		"valid local row":               {func(*Model) {}, ""},
		"valid hosted row":              {func(m *Model) { m.Backend = BackendOpenRouter; m.Runtime.BoxOrder = BoxOrderYXYX }, ""},
		"local row without a role":      {func(m *Model) { m.Runtime.GatewayRole = "" }, "gateway_role"},
		"hosted vision row, no role":    {func(m *Model) { m.Backend = BackendOpenRouter; m.Runtime.GatewayRole = "" }, "gateway_role"},
		"local row serving an image op": {func(m *Model) { m.Operations = []string{"caption", "colorize"} }, "serves only"},
		"detection without box order":   {func(m *Model) { m.Runtime.BoxOrder = "" }, "box_order"},
		"detection with a bad order":    {func(m *Model) { m.Runtime.BoxOrder = "xxyy" }, "box_order"},
		"caption-only needs no boxes":   {func(m *Model) { m.Operations = []string{"caption"}; m.Runtime.BoxOrder = "" }, ""},
		"image-only openrouter row": {func(m *Model) {
			m.Backend = BackendOpenRouter
			m.Operations = []string{"edit_instruct"}
			m.Runtime = Runtime{}
		}, ""},
	} {
		err := reg.validateModel(gatewayRow(tc.mut))
		switch {
		case tc.want == "" && err != nil:
			t.Errorf("%s: unexpected error %v", name, err)
		case tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)):
			t.Errorf("%s: error %v, want it to mention %q", name, err, tc.want)
		}
	}
}

func TestGatewayServedModelsNeedNoDownload(t *testing.T) {
	local := gatewayRow(func(*Model) {})
	if local.RequiresWeights() {
		t.Error("a gateway-served model's weights belong to the gateway, not to image-tools acquisition")
	}
	if record, weightless := WeightlessInstallRecord(local); !weightless || !record.Installed {
		t.Errorf("a gateway-served model reads installed from the start, got %+v", record)
	}
	if !local.GatewayServed() {
		t.Error("backend gateway-vision is gateway-served")
	}
	image := gatewayRow(func(m *Model) {
		m.Backend = BackendOpenRouter
		m.Runtime = Runtime{}
		m.Operations = []string{"edit_instruct"}
	})
	if image.GatewayServed() {
		t.Error("an openrouter image row with no gateway role is not a vision row")
	}
}

// The seeded vision rows load under the same rules, and Florence-2 stays the
// fast default.
func TestSeededGatewayVisionRows(t *testing.T) {
	reg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"qwen3-vl-4b", "gemini-flash-lite-vision"} {
		m, ok := reg.ByID(id)
		if !ok {
			t.Fatalf("seed has no %s row", id)
		}
		if !m.GatewayServed() || m.Runtime.GatewayRole == "" || (m.ID != "qwen3-vl-4b" && len(m.DefaultFor) != 0) {
			t.Errorf("%s: %+v", id, m)
		}
	}
	if def, ok := reg.DefaultFor("caption"); !ok || def.ID != "florence-2" {
		t.Errorf("caption default = %v, want florence-2", def.ID)
	}
	// WD14 is not installed and names its tags tag_<index>; the general vision
	// model is the tagger that works.
	if def, ok := reg.DefaultFor("tagging"); !ok || def.ID != "qwen3-vl-4b" {
		t.Errorf("tagging default = %v, want qwen3-vl-4b", def.ID)
	}
}

// The gateway places a gateway-served model on this machine's hardware, so a CPU
// or GPU fit warning from this scenario's own probe would describe a placement it
// does not make.
func TestGatewayServedSelectionCarriesNoHardwareFitWarning(t *testing.T) {
	reg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, host := range []capabilities.Host{cpuHost(), busyGPUHost(16, 15)} {
		sel, err := reg.Select(SelectRequest{RightsEvaluator: allowRightsForTest, Operation: "tagging", Host: host}, nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, w := range sel.Warnings {
			if strings.Contains(w, "CPU") || strings.Contains(w, "GPU") || strings.Contains(w, "VRAM") {
				t.Errorf("host %+v: hardware warning on a gateway-served model: %q", host, w)
			}
		}
		if !strings.Contains(sel.Reason, "AI Gateway") {
			t.Errorf("selection reason should name the gateway: %q", sel.Reason)
		}
	}
}
