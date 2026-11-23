package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"image-tools/internal/backends"
	"image-tools/internal/capabilities"
	"image-tools/internal/capacity"
	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/models"
	"image-tools/internal/storage"

	"github.com/vrooli/api-core/blobstore"
)

// extentStore remembers which artifacts the engine stored and deleted, so a
// test can prove the stages leave nothing behind but the result.
type extentStore struct {
	*storage.Store
	mu      sync.Mutex
	stored  []string
	deleted map[string]bool
}

func (r *extentStore) Put(ctx context.Context, key string, rd io.Reader, mime string) error {
	r.mu.Lock()
	r.stored = append(r.stored, key)
	r.mu.Unlock()
	return r.Store.Put(ctx, key, rd, mime)
}

func (r *extentStore) Delete(ctx context.Context, key string) error {
	r.mu.Lock()
	r.deleted[key] = true
	r.mu.Unlock()
	return r.Store.Delete(ctx, key)
}

func solidPNG(t *testing.T, w, h int, c color.NRGBA) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, 255
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// extentEngine serves face_swap and segment with stand-in providers (a fixed
// swapped image, an all-selected mask) and color_transfer with the real
// in-process implementation. Extra providers serve further stages (the general
// editor of a restyle).
func extentEngine(t *testing.T, swapped, mask []byte, extra ...backends.Provider) (*Engine, *extentStore, *fakeProvider, *fakeProvider) {
	t.Helper()
	registry, err := models.Load()
	if err != nil {
		t.Fatal(err)
	}
	swapModel, _ := registry.DefaultFor("face_swap")
	maskModel, ok := registry.ByID("segformer-b2-clothes")
	if !ok {
		t.Fatal("segformer-b2-clothes missing from the registry")
	}
	swap := &fakeProvider{name: swapModel.Backend, ops: []string{"face_swap"}, out: swapped}
	segment := &fakeProvider{name: maskModel.Backend, ops: []string{"segment"}, out: mask}
	reg := backends.New()
	for _, p := range append([]backends.Provider{swap, segment, builtinProviderSpecs()[0]}, extra...) {
		if err := reg.Register(p); err != nil {
			t.Fatal(err)
		}
	}
	store := &extentStore{Store: storage.NewWithBlobStore(blobstore.NewMemoryBlobStore(), t.TempDir()), deleted: map[string]bool{}}
	eng, err := NewEngineWithRightsEvaluator(Deps{
		Registry: registry, Backends: reg, Probe: capabilities.FakeProbe{Host: capabilities.Host{OS: "linux", Arch: "amd64", Cores: 8, GPUs: []capabilities.GPU{{Index: 0, UUID: "GPU-test", VRAMBytes: 16 << 30}}}},
		Store: store, ModelInstalled: func(string) bool { return true }, ModelsRoot: t.TempDir(), Disk: &fakeDiskAdmission{},
	}, allowRightsForTest)
	if err != nil {
		t.Fatal(err)
	}
	return eng, store, swap, segment
}

func runExtentJob(t *testing.T, eng *Engine, store *extentStore, params map[string]string) (internaljobs.Result, error) {
	t.Helper()
	ctx := context.Background()
	for key, data := range map[string][]byte{
		"in/target.png":    solidPNG(t, 64, 64, color.NRGBA{R: 230, G: 190, B: 170}),
		"in/reference.png": solidPNG(t, 64, 64, color.NRGBA{R: 90, G: 60, B: 50}),
	} {
		if err := store.Put(ctx, key, bytes.NewReader(data), "image/png"); err != nil {
			t.Fatal(err)
		}
	}
	store.stored = nil
	registry, _ := models.Load()
	model, _ := registry.DefaultFor("face_swap")
	raw, err := marshalPinnedPayload(t, eng, "face_swap", Payload{
		Operation: "face_swap", InputKey: "in/target.png", ReferenceKey: "in/reference.png", ModelID: model.ID, GPU: false, Params: params,
	})
	if err != nil {
		t.Fatal(err)
	}
	return eng.BuildRunners()["face_swap"](ctx, internaljobs.Job{ID: "job-extent", Operation: "face_swap", Payload: raw}, func(int, string) {})
}

func resultColour(t *testing.T, store *extentStore, ref string) color.NRGBA {
	t.Helper()
	rc, _, err := store.Get(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rc.Close() }()
	img, err := png.Decode(rc)
	if err != nil {
		t.Fatal(err)
	}
	return color.NRGBAModel.Convert(img.At(32, 32)).(color.NRGBA)
}

func TestExtentRunsTheDeclaredStagesAndLeavesOnlyTheResult(t *testing.T) {
	swapped := solidPNG(t, 64, 64, color.NRGBA{R: 230, G: 190, B: 170})
	white := solidPNG(t, 64, 64, color.NRGBA{R: 255, G: 255, B: 255})
	eng, store, swap, segment := extentEngine(t, swapped, white)

	result, err := runExtentJob(t, eng, store, map[string]string{"extent": "full_look"})
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Meta[MetaExtentStages]; got != "face,skin,hair_colour" {
		t.Fatalf("stages that ran = %q", got)
	}
	if swap.execN != 1 || segment.execN != 4 { // a region and a reference mask for each of the two recolour stages
		t.Fatalf("stage runs: swap %d, segment %d", swap.execN, segment.execN)
	}
	for _, stage := range []string{"skin", "hair_colour"} {
		if result.Meta["extent."+stage+".mask_model"] != "segformer-b2-clothes" {
			t.Fatalf("stage %s not recorded: %v", stage, result.Meta)
		}
	}
	if c := resultColour(t, store, result.Ref); c.R > 150 {
		t.Fatalf("the recolour stages did not move the swapped face toward the reference: %v", c)
	}
	var left []string
	for _, key := range store.stored {
		if !store.deleted[key] {
			left = append(left, key)
		}
	}
	if len(left) != 1 || left[0] != result.Ref {
		t.Fatalf("intermediate images were left behind: %v (result %s)", left, result.Ref)
	}
}

func TestExtentStagesAreIndependentToggles(t *testing.T) {
	swapped := solidPNG(t, 64, 64, color.NRGBA{R: 230, G: 190, B: 170})
	eng, store, _, segment := extentEngine(t, swapped, solidPNG(t, 64, 64, color.NRGBA{R: 255, G: 255, B: 255}))
	result, err := runExtentJob(t, eng, store, map[string]string{"match": `["hair-colour"]`})
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Meta[MetaExtentStages]; got != "face,hair_colour" || segment.execN != 2 {
		t.Fatalf("stages %q, segment runs %d", got, segment.execN)
	}
}

func TestExtentWithoutMatchOrPresetIsThePlainSwap(t *testing.T) {
	swapped := solidPNG(t, 64, 64, color.NRGBA{R: 1, G: 2, B: 3})
	eng, store, swap, segment := extentEngine(t, swapped, nil)
	result, err := runExtentJob(t, eng, store, nil)
	if err != nil {
		t.Fatal(err)
	}
	if segment.execN != 0 || swap.execN != 1 || result.Meta[MetaExtentStages] != "" {
		t.Fatalf("a plain swap ran extra stages: segment %d, meta %v", segment.execN, result.Meta)
	}
}

func TestExtentStageFailureFailsTheJobNamingTheStageAndCleansUp(t *testing.T) {
	swapped := solidPNG(t, 64, 64, color.NRGBA{R: 230, G: 190, B: 170})
	eng, store, _, segment := extentEngine(t, swapped, solidPNG(t, 64, 64, color.NRGBA{R: 255, G: 255, B: 255}))
	segment.failWith = context.DeadlineExceeded
	_, err := runExtentJob(t, eng, store, map[string]string{"match": `["skin"]`})
	if err == nil || !strings.Contains(err.Error(), `stage "skin"`) {
		t.Fatalf("error = %v", err)
	}
	for _, key := range store.stored {
		if !store.deleted[key] {
			t.Fatalf("%s was left behind by a failed job", key)
		}
	}
}

func TestExtentRefusesStagesTheOperationDoesNotDeclare(t *testing.T) {
	eng, store, _, _ := extentEngine(t, solidPNG(t, 8, 8, color.NRGBA{A: 255}), nil)
	if _, err := runExtentJob(t, eng, store, map[string]string{"match": `["freckles"]`}); err == nil || !strings.Contains(err.Error(), "freckles") {
		t.Fatalf("error = %v", err)
	}
}

func TestExtentWholeHeadPinsTheGeneralEditorAndItsInstruction(t *testing.T) {
	params := map[string]string{"extent": "whole_head"}
	model, err := ExtentModel("face_swap", params)
	if err != nil || model != "flux-2-klein-4b" {
		t.Fatalf("model = %q, %v", model, err)
	}
	prompt, err := ExtentPrompt("face_swap", params)
	if err != nil || !strings.Contains(prompt, "head") {
		t.Fatalf("prompt = %q, %v", prompt, err)
	}
	if model, _ := ExtentModel("face_swap", map[string]string{"extent": "face_skin"}); model != "" {
		t.Fatalf("the dedicated swapper's extent must not pin a model: %q", model)
	}
	if _, err := ExtentModel("upscale", params); err != nil {
		t.Fatalf("an operation without an extent ignores it: %v", err)
	}
}

func TestExtentMatchParamIsJSONOnTheWire(t *testing.T) {
	raw, _ := json.Marshal([]string{"skin"})
	stages, extended, err := extentRequest("face_swap", map[string]string{"match": string(raw)})
	if err != nil || !extended || len(stages) != 2 || stages[1].ID != "skin" {
		t.Fatalf("stages %v extended %v err %v", stages, extended, err)
	}
}

// A stage runs inside a running job, where the job manager would not requeue it,
// so a capacity refusal has to be waited out in place.
func TestExtentStageWaitsOutACapacityRefusalInsteadOfFailingTheJob(t *testing.T) {
	oldWait, oldBackoff := stageBusyWait, stageBusyBackoff
	stageBusyWait, stageBusyBackoff = time.Minute, []time.Duration{time.Millisecond}
	t.Cleanup(func() { stageBusyWait, stageBusyBackoff = oldWait, oldBackoff })
	swapped := solidPNG(t, 64, 64, color.NRGBA{R: 230, G: 190, B: 170})
	eng, store, _, segment := extentEngine(t, swapped, solidPNG(t, 64, 64, color.NRGBA{R: 255, G: 255, B: 255}))
	refusals := 0
	segment.onExecute = func() {
		if refusals < 2 {
			refusals++
			segment.failWith = internaljobs.ErrCapacityBusy
			return
		}
		segment.failWith = nil
	}
	result, err := runExtentJob(t, eng, store, map[string]string{"match": `["skin"]`})
	if err != nil || result.Meta[MetaExtentStages] != "face,skin" || refusals != 2 {
		t.Fatalf("a stage refused capacity twice must still finish: %v, refusals %d, meta %v", err, refusals, result.Meta)
	}
}

func TestExtentStageGivesUpWhenTheCapacityNeverFrees(t *testing.T) {
	oldWait, oldBackoff := stageBusyWait, stageBusyBackoff
	stageBusyWait, stageBusyBackoff = 20*time.Millisecond, []time.Duration{5 * time.Millisecond}
	t.Cleanup(func() { stageBusyWait, stageBusyBackoff = oldWait, oldBackoff })
	eng, store, _, segment := extentEngine(t, solidPNG(t, 64, 64, color.NRGBA{R: 230, G: 190, B: 170}), solidPNG(t, 64, 64, color.NRGBA{R: 255, G: 255, B: 255}))
	segment.failWith = internaljobs.ErrCapacityBusy
	_, err := runExtentJob(t, eng, store, map[string]string{"match": `["skin"]`})
	if !errors.Is(err, internaljobs.ErrCapacityBusy) || !strings.Contains(err.Error(), `stage "skin"`) {
		t.Fatalf("error = %v", err)
	}
}

// The hair style stage has a general editor redo the picture, but only the hair
// region of that result may reach the output: the swapped face is not re-rendered.
func TestExtentHairStyleTakesOnlyTheHairRegionFromTheEditor(t *testing.T) {
	registry, err := models.Load()
	if err != nil {
		t.Fatal(err)
	}
	kleinModel, ok := registry.ByID("flux-2-klein-4b")
	if !ok {
		t.Fatal("flux-2-klein-4b missing from the registry")
	}
	edited := solidPNG(t, 32, 32, color.NRGBA{G: 200}) // a smaller render, as an editor may return
	klein := &fakeProvider{name: kleinModel.Backend, ops: []string{"face_swap"}, out: edited}
	// The hair mask selects the left half of the picture.
	left := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			v := uint8(0)
			if x < 32 {
				v = 255
			}
			left.SetNRGBA(x, y, color.NRGBA{R: v, G: v, B: v, A: 255})
		}
	}
	var mask bytes.Buffer
	if err := png.Encode(&mask, left); err != nil {
		t.Fatal(err)
	}
	swapped := solidPNG(t, 64, 64, color.NRGBA{R: 230, G: 190, B: 170})
	eng, store, swap, segment := extentEngine(t, swapped, mask.Bytes(), klein)

	result, err := runExtentJob(t, eng, store, map[string]string{"match": `["hair_style"]`})
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Meta[MetaExtentStages]; got != "face,hair_style" {
		t.Fatalf("stages that ran = %q", got)
	}
	if swap.execN != 1 || klein.execN != 1 || segment.execN != 2 { // the working image and the edited one
		t.Fatalf("runs: swap %d, editor %d, segment %d", swap.execN, klein.execN, segment.execN)
	}
	if got := klein.lastReq.Params["prompt"]; !strings.Contains(got, "hairstyle") {
		t.Fatalf("the editor was not given the stage's instruction: %q", got)
	}
	rc, _, err := store.Get(context.Background(), result.Ref)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rc.Close() }()
	img, err := png.Decode(rc)
	if err != nil {
		t.Fatal(err)
	}
	if c := color.NRGBAModel.Convert(img.At(5, 32)).(color.NRGBA); c.G < 190 || c.R > 20 {
		t.Fatalf("the hair region did not take the editor's result: %v", c)
	}
	if c := color.NRGBAModel.Convert(img.At(60, 32)).(color.NRGBA); c.R != 230 || c.G != 190 {
		t.Fatalf("outside the hair region the swapped image must survive: %v", c)
	}
	var left2 []string
	for _, key := range store.stored {
		if !store.deleted[key] {
			left2 = append(left2, key)
		}
	}
	if len(left2) != 1 || left2[0] != result.Ref {
		t.Fatalf("intermediate images were left behind: %v (result %s)", left2, result.Ref)
	}
}

// A stage that waits for capacity must not be waiting while the job itself holds
// capacity: two extent jobs could each hold what the other's stage needs. The
// swap stage's claims are released before the stage chain starts.
func TestExtentReleasesTheSwapClaimsBeforeTheStagesRun(t *testing.T) {
	white := solidPNG(t, 64, 64, color.NRGBA{R: 255, G: 255, B: 255})
	eng, store, _, segment := extentEngine(t, solidPNG(t, 64, 64, color.NRGBA{R: 230, G: 190, B: 170}), white)
	// Claims are told apart by owner: the job's own is "image-tools:<job>:<kind>"; a
	// stage's job id is "<job>:<stage>", which adds a segment.
	var parentClaims, parentHeld int
	var heldDuringStage int
	claim := func(_ context.Context, owner string, _ int64) (capacity.MemoryLease, error) {
		if strings.Count(owner, ":") != 2 { // a stage's owner has its label as a further segment
			return &memoryTestLease{}, nil
		}
		parentClaims++
		parentHeld++
		return &memoryTestLease{onRelease: func(context.Context) { parentHeld-- }}, nil
	}
	eng.deps.CPU, eng.deps.Memory = claim, claim
	segment.onExecute = func() { heldDuringStage = parentHeld }

	ctx := context.Background()
	for key, data := range map[string][]byte{
		"in/target.png":    solidPNG(t, 64, 64, color.NRGBA{R: 230, G: 190, B: 170}),
		"in/reference.png": solidPNG(t, 64, 64, color.NRGBA{R: 90, G: 60, B: 50}),
	} {
		if err := store.Put(ctx, key, bytes.NewReader(data), "image/png"); err != nil {
			t.Fatal(err)
		}
	}
	registry, _ := models.Load()
	model, _ := registry.DefaultFor("face_swap")
	raw, err := marshalPinnedPayload(t, eng, "face_swap", Payload{
		Operation: "face_swap", InputKey: "in/target.png", ReferenceKey: "in/reference.png", ModelID: model.ID,
		Params: map[string]string{"match": `["skin"]`},
	})
	if err != nil {
		t.Fatal(err)
	}
	job := internaljobs.Job{ID: "job-claims", Operation: "face_swap", Payload: raw}
	admission, err := eng.Admit(ctx, job)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := eng.BuildRunners()["face_swap"](admission.Context, job, func(int, string) {}); err != nil {
		t.Fatal(err)
	}
	if parentClaims == 0 {
		t.Fatal("the swap admission took no CPU or RAM claim, so this test proves nothing")
	}
	if heldDuringStage != 0 {
		t.Fatalf("the job still held %d claims while its skin stage ran", heldDuringStage)
	}
	if err := admission.Release(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestExtentSkinTargetBodyTakesTheColourFromTheImagesOwnBody(t *testing.T) {
	white := solidPNG(t, 64, 64, color.NRGBA{R: 255, G: 255, B: 255})
	eng, store, _, segment := extentEngine(t, solidPNG(t, 64, 64, color.NRGBA{R: 230, G: 190, B: 170}), white)
	var phrases []string
	segment.onExecute = func() {
		phrases = append(phrases, segment.lastReq.Params["segmentation"])
	}
	result, err := runExtentJob(t, eng, store, map[string]string{"match": `["skin"]`, "skin_target": "body"})
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Meta["extent.skin.skin_target"]; got != "body" {
		t.Fatalf("the variant that ran was not recorded: %v", result.Meta)
	}
	if len(phrases) != 2 || !strings.Contains(phrases[0], "face") || strings.Contains(phrases[0], "arms") || !strings.Contains(phrases[1], "arms") {
		t.Fatalf("body must segment the face to recolour and the arms to take colour from: %v", phrases)
	}
	// The default takes the colour from the reference person, face and arms together.
	phrases = nil
	result, err = runExtentJob(t, eng, store, map[string]string{"match": `["skin"]`})
	if err != nil || result.Meta["extent.skin.skin_target"] != "source" || len(phrases) != 2 || !strings.Contains(phrases[1], "face arms") {
		t.Fatalf("default skin target: %v, %v, %v", err, result.Meta, phrases)
	}
}

func TestExtentRefusesAnUnknownSkinTargetBeforeRunning(t *testing.T) {
	eng, store, swap, _ := extentEngine(t, solidPNG(t, 8, 8, color.NRGBA{A: 255}), nil)
	if _, err := runExtentJob(t, eng, store, map[string]string{"match": `["skin"]`, "skin_target": "moon"}); err == nil || !strings.Contains(err.Error(), "moon") {
		t.Fatalf("error = %v", err)
	}
	if swap.execN != 0 {
		t.Fatal("a refused request must not run the swap")
	}
}
