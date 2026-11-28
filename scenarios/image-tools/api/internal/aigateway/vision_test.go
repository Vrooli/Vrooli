package aigateway

import (
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"image-tools/internal/backends"
	"image-tools/internal/models"
	"image-tools/internal/storage"
)

type fakeVision struct {
	answer  VisionAnswer
	err     error
	request VisionRequest
	reach   error
}

func (f *fakeVision) Ask(_ context.Context, r VisionRequest) (VisionAnswer, error) {
	f.request = r
	return f.answer, f.err
}
func (f *fakeVision) Reachable(context.Context) error { return f.reach }

func writeImage(t *testing.T, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		img.Set(x, 0, color.RGBA{R: 200, A: 255})
	}
	path := filepath.Join(t.TempDir(), "in.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	return path
}

func visionRequestFor(t *testing.T, op string, rt models.Runtime, w, h int) (backends.Request, string) {
	t.Helper()
	out := filepath.Join(t.TempDir(), "out.json")
	return backends.Request{
		Operation: op,
		Model:     models.Model{ID: "row", Runtime: rt},
		InputKeys: []string{writeImage(t, w, h)},
		Params:    map[string]string{},
		Output:    storage.OutputTarget{LocalPath: out},
	}, out
}

// The two real answer shapes the gateway returns: Ollama's generate object and a
// hosted chat completion.
func TestCompletionTextUnwrapsEveryGatewayShape(t *testing.T) {
	for name, tc := range map[string]struct{ in, want string }{
		"ollama generate":  {`{"done":true,"response":"A guinea pig.","model":"qwen3-vl:4b"}`, "A guinea pig."},
		"chat completion":  {`{"choices":[{"message":{"content":"A tricolour guinea pig.","reasoning":"..."}}]}`, "A tricolour guinea pig."},
		"plain text":       {"  A guinea pig. ", "A guinea pig."},
		"json answer kept": {`[{"label":"a"}]`, `[{"label":"a"}]`},
	} {
		if got := completionText(tc.in); got != tc.want {
			t.Errorf("%s: completionText = %q, want %q", name, got, tc.want)
		}
	}
}

// Models disagree on box order and grid; the row says which, and the parser puts
// every one in source pixels. Numbers are the real answers measured on the
// 768x1024 guinea pig photograph (a hand-drawn box is about 130,176 to 680,684).
func TestParseDetectionsReadsEveryModelConvention(t *testing.T) {
	const w, h = 768, 1024
	cases := []struct {
		name, text, order string
		scale             int
		want              [4]float64
	}{
		{"qwen xyxy on a 1000 grid in a fence", "```json\n[{\"bbox_2d\": [167, 165, 994, 670], \"label\": \"guinea_pig\"}]\n```", models.BoxOrderXYXY, 1000, [4]float64{128.3, 169, 763.4, 686.1}},
		{"gemini yxyx on a 1000 grid", `[{"label":"guinea pig","bbox_2d":[167,163,667,883]}]`, models.BoxOrderYXYX, 1000, [4]float64{125.2, 171, 678.1, 683}},
		{"gemini box_2d key and a wrapper object", `{"objects":[{"box_2d":[166,158,667,896],"label":"guinea pig"}]}`, models.BoxOrderYXYX, 1000, [4]float64{121.3, 170, 688.1, 683}},
		{"pixels need no scale", `[{"label":"cup","box":[10,20,110,220]}]`, models.BoxOrderXYXY, 0, [4]float64{10, 20, 110, 220}},
		{"prose around the list", "Here you go: [{\"label\":\"cup\",\"bbox_2d\":[0,0,500,500]}] done", models.BoxOrderXYXY, 1000, [4]float64{0, 0, 384, 512}},
	}
	for _, tc := range cases {
		got, err := ParseDetections(tc.text, tc.order, tc.scale, w, h)
		if err != nil || len(got) != 1 {
			t.Fatalf("%s: %v %v", tc.name, got, err)
		}
		for i := range tc.want {
			if d := got[0].Box[i] - tc.want[i]; d > 0.2 || d < -0.2 {
				t.Errorf("%s: box %v, want %v", tc.name, got[0].Box, tc.want)
				break
			}
		}
	}
}

func TestParseDetectionsDropsUnusableBoxesAndFailsLoudly(t *testing.T) {
	got, err := ParseDetections(`[
		{"label":"ok","bbox_2d":[100,100,300,300]},
		{"label":"flat","bbox_2d":[100,100,100,300]},
		{"label":"outside","bbox_2d":[2000,2000,3000,3000]},
		{"label":"short","bbox_2d":[1,2,3]},
		{"bbox_2d":[100,100,300,300]},
		{"label":"text","bbox_2d":["a","b","c","d"]}]`, models.BoxOrderXYXY, 1000, 1000, 1000)
	if err != nil || len(got) != 1 || got[0].Label != "ok" {
		t.Fatalf("only the usable box should survive: %v %v", got, err)
	}
	for _, bad := range []string{"I cannot see any objects.", `[{"label":"x"}]`, `{"a":1}`, "[1,2"} {
		if _, err := ParseDetections(bad, models.BoxOrderXYXY, 1000, 100, 100); err == nil {
			t.Errorf("%q should be an error, not an empty success", bad)
		}
	}
}

func TestParseTags(t *testing.T) {
	for name, tc := range map[string]struct {
		in   string
		want string
	}{
		"fenced array":   {"```json\n[\"Guinea Pig\", \"cage\", \"guinea pig\"]\n```", "guinea pig,cage"},
		"objects":        {`[{"tag":"hay"},{"label":"bowl"}]`, "hay,bowl"},
		"comma separate": {"guinea pig, cage, hay", "guinea pig,cage,hay"},
		"after thinking": {"<think>hmm [x]</think>[\"a\",\"b\"]", "a,b"},
	} {
		if got := strings.Join(ParseTags(tc.in), ","); got != tc.want {
			t.Errorf("%s: tags = %q, want %q", name, got, tc.want)
		}
	}
}

func TestExecuteVisionWritesEachOperationsOutputContract(t *testing.T) {
	rt := models.Runtime{GatewayRole: "vision.default", BoxOrder: models.BoxOrderXYXY, BoxScale: 1000}
	for _, tc := range []struct {
		op, reply string
		check     func(t *testing.T, out map[string]any)
	}{
		{"caption", `{"response":"A guinea pig in a cage."}`, func(t *testing.T, out map[string]any) {
			if out["caption"] != "A guinea pig in a cage." {
				t.Errorf("caption output %v", out)
			}
		}},
		{"tagging", `["guinea pig","cage"]`, func(t *testing.T, out map[string]any) {
			if out["count"] != float64(2) {
				t.Errorf("tag output %v", out)
			}
		}},
		{"object_detection", `[{"label":"cup","bbox_2d":[0,0,500,500]}]`, func(t *testing.T, out map[string]any) {
			objects, _ := out["objects"].([]any)
			if len(objects) != 1 || out["count"] != float64(1) {
				t.Fatalf("detection output %v", out)
			}
			if box, _ := objects[0].(map[string]any)["box"].([]any); len(box) != 4 || box[2] != float64(20) || box[3] != float64(10) {
				t.Errorf("box must be in source pixels of the 40x20 image, got %v", box)
			}
		}},
	} {
		gateway := &fakeVision{answer: VisionAnswer{Text: tc.reply, Model: "qwen3-vl:4b", Provider: "ollama", EventID: "ev-1", Status: "succeeded"}}
		req, out := visionRequestFor(t, tc.op, rt, 40, 20)
		result, err := executeVision(context.Background(), gateway, models.BackendGatewayVision, true, req)
		if err != nil {
			t.Fatalf("%s: %v", tc.op, err)
		}
		if gateway.request.Role != "vision.default" || !gateway.request.Local || len(gateway.request.Image) == 0 || gateway.request.Prompt == "" {
			t.Errorf("%s: gateway request %+v", tc.op, gateway.request)
		}
		raw, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		var decoded map[string]any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatalf("%s: output is not JSON: %v", tc.op, err)
		}
		if decoded["model"] != "qwen3-vl:4b" || decoded["backend"] != models.BackendGatewayVision {
			t.Errorf("%s: output must name the model that answered, got %v", tc.op, decoded)
		}
		tc.check(t, decoded)
		if result.Meta["model"] != "qwen3-vl:4b" || result.Meta["role"] != "vision.default" {
			t.Errorf("%s: meta %v", tc.op, result.Meta)
		}
		if _, hasCost := result.Meta[MetaCostUSD]; hasCost {
			t.Errorf("%s: a local route reports no cost, got %v", tc.op, result.Meta)
		}
	}
}

func TestExecuteVisionHostedRouteReportsCostAndAsksRemote(t *testing.T) {
	gateway := &fakeVision{answer: VisionAnswer{Text: `{"choices":[{"message":{"content":"A cup."}}]}`, Model: "google/gemini", EventID: "ev-2", Status: "succeeded", CostUSD: 0.0004, CostKnown: true}}
	req, _ := visionRequestFor(t, "caption", models.Runtime{GatewayRole: "vision.default"}, 40, 20)
	req.Params[ParamMaxCostUSD] = "0.05"
	result, err := executeVision(context.Background(), gateway, models.BackendOpenRouter, false, req)
	if err != nil {
		t.Fatal(err)
	}
	if gateway.request.Local || gateway.request.MaxCostUSD != 0.05 {
		t.Errorf("hosted request %+v", gateway.request)
	}
	if cost, ok := ParseCostUSD(result.Meta); !ok || cost != 0.0004 {
		t.Errorf("the hosted route must report its cost for the spend ledger, meta %v", result.Meta)
	}
	if _, ok := EvidenceVariationFromMeta(0, result.Meta); !ok {
		t.Errorf("hosted route must carry execution evidence: %v", result.Meta)
	}
	if result.Tier != backends.TierBYOK {
		t.Errorf("tier = %v", result.Tier)
	}
}

func TestLocalVisionRouteIsGatewayLocalNotACPURun(t *testing.T) {
	gateway := &fakeVision{answer: VisionAnswer{Text: "A cup.", Model: "qwen3-vl:4b", EventID: "ev-3", Status: "succeeded"}}
	req, _ := visionRequestFor(t, "caption", models.Runtime{GatewayRole: "vision.default"}, 40, 20)
	result, err := executeVision(context.Background(), gateway, models.BackendGatewayVision, true, req)
	if err != nil {
		t.Fatal(err)
	}
	if result.Tier != backends.TierGatewayLocal {
		t.Errorf("tier = %v, want gateway-local", result.Tier)
	}
	declared, ok := any(NewVisionProvider(gateway)).(interface{ LocalTier() backends.Tier })
	if !ok || declared.LocalTier() != backends.TierGatewayLocal {
		t.Error("the provider must declare the tier selection reports")
	}
}

func TestExecuteVisionFailuresAreActionable(t *testing.T) {
	rt := models.Runtime{GatewayRole: "vision.default", BoxOrder: models.BoxOrderXYXY, BoxScale: 1000}
	req, out := visionRequestFor(t, "object_detection", rt, 40, 20)
	if _, err := executeVision(context.Background(), &fakeVision{answer: VisionAnswer{Text: "no boxes here", Status: "succeeded"}}, "b", true, req); err == nil || !strings.Contains(err.Error(), "unusable") {
		t.Errorf("an unparseable answer must fail with its cause, got %v", err)
	}
	if _, err := os.Stat(out); err == nil {
		t.Error("a failed answer must not leave an output file")
	}
	boom := errors.New("gateway down")
	if _, err := executeVision(context.Background(), &fakeVision{err: boom}, "b", true, req); !errors.Is(err, boom) {
		t.Errorf("gateway error must surface, got %v", err)
	}
	noRole, _ := visionRequestFor(t, "caption", models.Runtime{}, 40, 20)
	if _, err := executeVision(context.Background(), &fakeVision{}, "b", true, noRole); err == nil || !strings.Contains(err.Error(), "gateway_role") {
		t.Errorf("a row without a role must say so, got %v", err)
	}
}

func TestPrepareVisionImageBoundsTheSizeAndKeepsOriginalDimensions(t *testing.T) {
	data, w, h, mediaType, err := prepareVisionImage(writeImage(t, 3200, 1600))
	if err != nil || w != 3200 || h != 1600 || mediaType != "image/jpeg" {
		t.Fatalf("dims %dx%d %s %v", w, h, mediaType, err)
	}
	cfg, _, err := image.DecodeConfig(strings.NewReader(string(data)))
	if err != nil || cfg.Width != visionMaxImageEdge || cfg.Height != visionMaxImageEdge/2 {
		t.Errorf("sent image %+v (%v), want long side %d", cfg, err, visionMaxImageEdge)
	}
}

func TestVisionProvidersAreOneExecutorAtTwoLocalities(t *testing.T) {
	local := NewVisionProvider(&fakeVision{})
	if local.Name() != models.BackendGatewayVision || local.IsCloud() || !local.Standalone() {
		t.Errorf("local provider %s cloud=%v", local.Name(), local.IsCloud())
	}
	hosted := NewProvider(&fakeMediaGateway{}, &fakeVision{})
	for _, op := range models.VisionOperations {
		if !contains(local.Operations(), op) || !contains(hosted.Operations(), op) {
			t.Errorf("%s must be served by both vision rungs", op)
		}
	}
	if contains(NewProvider(&fakeMediaGateway{}, nil).Operations(), "caption") {
		t.Error("a hosted provider with no vision client must not claim vision operations")
	}
	if !NewVisionProvider(&fakeVision{}).Available(context.Background()) {
		t.Error("a reachable gateway makes the local rung available")
	}
	if NewVisionProvider(&fakeVision{reach: errors.New("not running")}).Available(context.Background()) {
		t.Error("an unreachable gateway must make it unavailable")
	}
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

// Two local jobs at once must not reach the gateway together: the second waits
// for the first, and a job that is cancelled while waiting leaves cleanly.
func TestLocalVisionCallsRunOneAtATime(t *testing.T) {
	release := make(chan struct{})
	entered := make(chan struct{}, 2)
	active, peak := 0, 0
	var mu sync.Mutex
	gateway := &blockingVision{enter: func() {
		mu.Lock()
		active++
		peak = max(peak, active)
		mu.Unlock()
		entered <- struct{}{}
		<-release
		mu.Lock()
		active--
		mu.Unlock()
	}}
	provider := NewVisionProvider(gateway)
	rt := models.Runtime{GatewayRole: "vision.default"}
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		req, _ := visionRequestFor(t, "caption", rt, 40, 20)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := provider.Execute(context.Background(), req); err != nil {
				t.Error(err)
			}
		}()
	}
	<-entered
	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	if active != 1 {
		t.Errorf("a second local call reached the gateway while one was running (active %d)", active)
	}
	mu.Unlock()
	close(release)
	wg.Wait()
	if peak != 1 {
		t.Errorf("peak concurrent local calls = %d, want 1", peak)
	}

	ctx, cancel := context.WithCancel(context.Background())
	provider.(*visionProvider).slot <- struct{}{}
	cancel()
	req, _ := visionRequestFor(t, "caption", rt, 40, 20)
	if _, err := provider.Execute(ctx, req); !errors.Is(err, context.Canceled) {
		t.Errorf("a cancelled waiter must return the cancellation, got %v", err)
	}
}

type blockingVision struct{ enter func() }

func (b *blockingVision) Ask(context.Context, VisionRequest) (VisionAnswer, error) {
	b.enter()
	return VisionAnswer{Text: "A cup.", Model: "m", Status: "succeeded"}, nil
}
func (b *blockingVision) Reachable(context.Context) error { return nil }
