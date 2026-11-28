package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strconv"
	"strings"
	"time"

	"image-tools/internal/capability"
	internaljobs "image-tools/internal/jobs"

	"github.com/google/uuid"
)

// Job meta keys the swap extent reports.
const (
	// MetaExtentStages lists, in order, the stages that ran (face,skin,...).
	MetaExtentStages = "extent_stages"
	// metaExtentStagePrefix prefixes the per-stage record: extent.<stage>.<field>.
	metaExtentStagePrefix = "extent."
)

// extentRequest reads the swap extent a payload asks for and resolves it against
// the operation's declaration. ok is false when the operation declares no extent
// or the request asks for nothing beyond the plain swap (that job is the
// operation's ordinary run).
func extentRequest(op string, params map[string]string) (stages []capability.ExtentStage, ok bool, err error) {
	manifest, found := capability.Get(op)
	if !found || manifest.Extent == nil {
		return nil, false, nil
	}
	preset := params["extent"]
	var match []string
	if raw := params["match"]; raw != "" {
		if err := json.Unmarshal([]byte(raw), &match); err != nil {
			return nil, false, fmt.Errorf("ai: %s match: %w", op, err)
		}
	}
	if preset == "" && len(match) == 0 {
		return nil, false, nil
	}
	stages, err = manifest.Extent.Resolve(preset, match)
	if err != nil {
		return nil, false, fmt.Errorf("ai: %s: %w", op, err)
	}
	stages, err = capability.ChooseVariants(stages, params)
	if err != nil {
		return nil, false, fmt.Errorf("ai: %s: %w", op, err)
	}
	return stages, len(stages) > 1, nil
}

// ExtentModel is the model a request's extent pins (the general editor of the
// whole-head stage), or "" when the extent leaves model selection alone.
func ExtentModel(op string, params map[string]string) (string, error) {
	stages, _, err := extentRequest(op, params)
	if err != nil {
		return "", err
	}
	for _, st := range stages {
		if st.Kind == capability.ExtentSwap && st.Model != "" {
			return st.Model, nil
		}
	}
	return "", nil
}

// ExtentPrompt is the instruction a request's extent gives its swap stage, or ""
// when the stage uses the model's default wording.
func ExtentPrompt(op string, params map[string]string) (string, error) {
	stages, _, err := extentRequest(op, params)
	if err != nil {
		return "", err
	}
	for _, st := range stages {
		if st.Kind == capability.ExtentSwap && st.Prompt != "" {
			return st.Prompt, nil
		}
	}
	return "", nil
}

// runExtent is the runner of an operation that declares an extent. The swap
// stage is the operation's ordinary run, under the admission the job queued
// with. Every stage after it is an operation the engine already serves (segment
// to find the region, color_transfer to recolour it), run through the same
// admission, safety review and persistence as any job, one at a time, so a
// stage never holds the capacity the next one needs.
func (e *Engine) runExtent(ctx context.Context, op string, job internaljobs.Job, emit func(progress int, message string)) (internaljobs.Result, error) {
	var pl Payload
	if err := json.Unmarshal(job.Payload, &pl); err != nil {
		return internaljobs.Result{}, fmt.Errorf("ai: read %s payload: %w", op, err)
	}
	stages, extended, err := extentRequest(op, pl.Params)
	if err != nil {
		return internaljobs.Result{}, err
	}
	if !extended {
		result, err := e.run(ctx, op, job, emit)
		if err == nil && len(stages) > 0 {
			result.Meta = withStageMeta(result.Meta, stages)
		}
		return result, err
	}

	// Later stages must outlive the swap's capacity claims, which cancel the
	// context the job was admitted under when they are released.
	stageCtx := ctx
	prepared, _ := ctx.Value(executionAdmissionKey{}).(*preparedExecution)
	if prepared != nil && prepared.jobContext != nil {
		stageCtx = prepared.jobContext
	}
	stageCtx = context.WithValue(stageCtx, executionAdmissionKey{}, nil)

	span := 100 / len(stages)
	band := func(index int) func(int, string) {
		return func(progress int, message string) { emit(index*span+progress*span/100, message) }
	}
	swapped, err := e.run(ctx, op, job, band(0))
	if err != nil {
		return internaljobs.Result{}, err
	}
	// The swap's claims end here, not when the job does. A stage that waits for
	// capacity while its job still held some could deadlock against another job's
	// stage waiting on that capacity; the output review releases them only when it
	// is configured.
	if prepared != nil && prepared.finishInference != nil {
		cleanup, stop := context.WithTimeout(context.WithoutCancel(stageCtx), 10*time.Second)
		err := prepared.finishInference(cleanup)
		stop()
		if err != nil {
			return internaljobs.Result{}, errors.Join(err, e.deleteOutputs(context.WithoutCancel(stageCtx), []string{swapped.Ref}))
		}
	}
	meta := swapped.Meta
	if meta == nil {
		meta = map[string]string{}
	}
	meta = withStageMeta(meta, stages)

	// current is the working image. Every image the stages produce is deleted when
	// the job ends, except the one returned.
	current := swapped.Ref
	temporary := []string{current}
	finish := func(result internaljobs.Result, runErr error) (internaljobs.Result, error) {
		cleanup, stop := context.WithTimeout(context.WithoutCancel(stageCtx), 15*time.Second)
		defer stop()
		var keep []string
		for _, key := range temporary {
			if runErr != nil || key != result.Ref {
				keep = append(keep, key)
			}
		}
		return result, errors.Join(runErr, e.deleteOutputs(cleanup, keep))
	}
	reference := pl.ReferenceKey
	for index, stage := range stages[1:] {
		emit((index+1)*span, "matching "+stage.ID)
		started := time.Now()
		next, produced, err := e.runStage(stageCtx, job, pl, stage, current, reference, band(index+1))
		temporary = append(temporary, produced...)
		if err != nil {
			return finish(internaljobs.Result{}, fmt.Errorf("ai: %s stage %q: %w", op, stage.ID, err))
		}
		meta[metaExtentStagePrefix+stage.ID+".seconds"] = strconv.FormatFloat(time.Since(started).Seconds(), 'f', 1, 64)
		meta[metaExtentStagePrefix+stage.ID+".mask_model"] = stage.MaskModel
		if stage.Option != "" {
			meta[metaExtentStagePrefix+stage.ID+"."+stage.Option] = stage.DefaultVariant
		}
		current = next
		temporary = append(temporary, next)
	}
	return finish(internaljobs.Result{Ref: current, Meta: meta}, nil)
}

func withStageMeta(meta map[string]string, stages []capability.ExtentStage) map[string]string {
	if meta == nil {
		meta = map[string]string{}
	}
	ids := make([]string, 0, len(stages))
	for _, st := range stages {
		ids = append(ids, st.ID)
	}
	meta[MetaExtentStages] = strings.Join(ids, ",")
	return meta
}

// runStage runs one non-swap stage on the working image and returns the new
// working image plus every intermediate it stored.
func (e *Engine) runStage(ctx context.Context, parent internaljobs.Job, pl Payload, stage capability.ExtentStage, current, reference string, emit func(int, string)) (string, []string, error) {
	switch stage.Kind {
	case capability.ExtentRecolour:
		return e.recolourStage(ctx, parent, pl, stage, current, reference, emit)
	case capability.ExtentRestyle:
		return e.restyleStage(ctx, parent, pl, stage, current, reference, emit)
	}
	return "", nil, fmt.Errorf("extent stage kind %q has no executor", stage.Kind)
}

// segmentRegion stores the mask of the phrase's region of an image.
func (e *Engine) segmentRegion(ctx context.Context, parent internaljobs.Job, pl Payload, stage capability.ExtentStage, label, key, phrase string, emit func(int, string)) (string, error) {
	prompt, err := json.Marshal(map[string]string{"text": phrase})
	if err != nil {
		return "", err
	}
	result, err := e.runStageOp(ctx, parent, pl, stage.ID+"-"+label, stageRun{
		op: "segment", model: stage.MaskModel, input: key, params: map[string]string{"segmentation": string(prompt)},
	}, emit)
	return result.Ref, err
}

// restyleStage has the stage's general editor redo the working image with the
// reference's look, then lays in only the region (the hair) of its result: the
// editor re-renders the whole picture, face included, and the face the swap
// produced must survive. The region is the union of the working image's and the
// edited image's, so the old hair is replaced and the new hair appears.
func (e *Engine) restyleStage(ctx context.Context, parent internaljobs.Job, pl Payload, stage capability.ExtentStage, current, reference string, emit func(int, string)) (string, []string, error) {
	var produced []string
	keep := func(key string) {
		if key != "" {
			produced = append(produced, key)
		}
	}
	params := map[string]string{"prompt": stage.Prompt}
	if seed := pl.Params["seed"]; seed != "" {
		params["seed"] = seed
	}
	edited, err := e.runStageOp(ctx, parent, pl, stage.ID+"-edit", stageRun{
		op: "face_swap", model: stage.Model, input: current, reference: reference, params: params,
	}, emit)
	keep(edited.Ref)
	if err != nil {
		return "", produced, err
	}
	before, err := e.segmentRegion(ctx, parent, pl, stage, "region", current, stage.Region, emit)
	keep(before)
	if err != nil {
		return "", produced, err
	}
	after, err := e.segmentRegion(ctx, parent, pl, stage, "edited", edited.Ref, stage.Region, emit)
	keep(after)
	if err != nil {
		return "", produced, err
	}
	union, err := e.unionMasks(ctx, before, after)
	if err != nil {
		return "", produced, err
	}
	keep(union)
	composite := make(map[string]string, len(stage.Composite))
	for name, value := range stage.Composite {
		composite[name] = strconv.FormatFloat(value, 'f', -1, 64)
	}
	result, err := e.runStageOp(ctx, parent, pl, stage.ID+"-composite", stageRun{
		op: "mask_composite", input: current, mask: union, reference: edited.Ref, params: composite,
	}, emit)
	return result.Ref, produced, err
}

// unionMasks stores a mask selecting what either mask selects, at the geometry
// of the first (the second is sampled to it).
func (e *Engine) unionMasks(ctx context.Context, first, second string) (string, error) {
	a, err := e.decodeStored(ctx, first)
	if err != nil {
		return "", err
	}
	b, err := e.decodeStored(ctx, second)
	if err != nil {
		return "", err
	}
	w, h := a.Bounds().Dx(), a.Bounds().Dy()
	wa, wb := sampledWeights(a, w, h), sampledWeights(b, w, h)
	out := image.NewGray(image.Rect(0, 0, w, h))
	for i := range wa {
		if wa[i] > 0 || wb[i] > 0 {
			out.Pix[i] = 255
		}
	}
	data, err := encodePNG(out)
	if err != nil {
		return "", err
	}
	key := "out/" + uuid.NewString() + ".png"
	if err := e.deps.Store.Put(ctx, key, bytes.NewReader(data), "image/png"); err != nil {
		return "", fmt.Errorf("ai: store mask union: %w", err)
	}
	return key, nil
}

func (e *Engine) decodeStored(ctx context.Context, key string) (image.Image, error) {
	rc, _, err := e.deps.Store.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("ai: fetch %q: %w", key, err)
	}
	defer func() { _ = rc.Close() }()
	img, _, err := image.Decode(rc)
	if err != nil {
		return nil, fmt.Errorf("ai: decode %q: %w", key, err)
	}
	return img, nil
}

// recolourStage moves the colour of the stage's region of the reference onto the
// same region of the working image. It returns the new working image and every
// intermediate it stored, so the caller can delete them.
func (e *Engine) recolourStage(ctx context.Context, parent internaljobs.Job, pl Payload, stage capability.ExtentStage, current, reference string, emit func(int, string)) (string, []string, error) {
	var produced []string
	segment := func(label, key, phrase string) (string, error) {
		ref, err := e.segmentRegion(ctx, parent, pl, stage, label, key, phrase, emit)
		if ref != "" {
			produced = append(produced, ref)
		}
		return ref, err
	}
	mask, err := segment("region", current, stage.Region)
	if err != nil {
		return "", produced, err
	}
	// The colour comes from the reference photo, or from the working image itself
	// (the face is matched to the body it sits on).
	colourSource, own := reference, stage.ReferenceFrom == capability.ReferenceWorking
	if own {
		colourSource = current
	}
	referenceMask, err := segment("reference", colourSource, stage.ReferenceRegion)
	if err != nil {
		if own {
			err = fmt.Errorf("take colour from the image's own %q: %w", stage.ReferenceRegion, err)
		}
		return "", produced, err
	}
	selected, err := e.referenceWithAlpha(ctx, colourSource, referenceMask)
	if err != nil {
		return "", produced, err
	}
	produced = append(produced, selected)

	params := make(map[string]string, len(stage.Transfer))
	for name, value := range stage.Transfer {
		params[name] = strconv.FormatFloat(value, 'f', -1, 64)
	}
	result, err := e.runStageOp(ctx, parent, pl, stage.ID+"-transfer", stageRun{
		op: "color_transfer", input: current, mask: mask, reference: selected, params: params,
	}, emit)
	return result.Ref, produced, err
}

// referenceWithAlpha stores the reference image with its alpha channel set to a
// mask, so the colour transfer reads statistics from that region only.
func (e *Engine) referenceWithAlpha(ctx context.Context, referenceKey, maskKey string) (string, error) {
	decode := func(key string) (image.Image, error) { return e.decodeStored(ctx, key) }
	ref, err := decode(referenceKey)
	if err != nil {
		return "", err
	}
	mask, err := decode(maskKey)
	if err != nil {
		return "", err
	}
	rb, mb := ref.Bounds(), mask.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, rb.Dx(), rb.Dy()))
	draw.Draw(out, out.Bounds(), ref, rb.Min, draw.Src)
	for y := 0; y < rb.Dy(); y++ {
		for x := 0; x < rb.Dx(); x++ {
			r, g, b, a := mask.At(mb.Min.X+x*mb.Dx()/rb.Dx(), mb.Min.Y+y*mb.Dy()/rb.Dy()).RGBA()
			c := out.NRGBAAt(x, y)
			if a == 0 || (r+g+b)/3>>8 <= 127 {
				c.A = 0
			}
			out.SetNRGBA(x, y, color.NRGBA{R: c.R, G: c.G, B: c.B, A: c.A})
		}
	}
	data, err := encodePNG(out)
	if err != nil {
		return "", err
	}
	key := "out/" + uuid.NewString() + ".png"
	if err := e.deps.Store.Put(ctx, key, bytes.NewReader(data), "image/png"); err != nil {
		return "", fmt.Errorf("ai: store reference region: %w", err)
	}
	return key, nil
}

// stageRun is one engine operation a stage asks for.
type stageRun struct {
	op, model              string
	input, mask, reference string
	params                 map[string]string
}

// runStageOp plans and runs one stage operation through the engine. A stage is
// always local: it never leaves the machine, whatever route the swap took.
func (e *Engine) runStageOp(ctx context.Context, parent internaljobs.Job, parentPayload Payload, label string, run stageRun, emit func(int, string)) (internaljobs.Result, error) {
	autoAcquire := parentPayload.AutoAcquire
	plan, err := e.Plan(ctx, PlanRequest{
		Operation: run.op, ModelOverride: run.model, FallbackPolicy: "local_only",
		Priority: parentPayload.CapacityPriority, AllowReclaim: parentPayload.AllowReclaim, AutoAcquire: &autoAcquire,
	})
	if err != nil {
		return internaljobs.Result{}, err
	}
	params := make(map[string]string, len(run.params)+len(plan.Params))
	for k, v := range plan.Params {
		params[k] = v
	}
	for k, v := range run.params {
		params[k] = v
	}
	raw, err := json.Marshal(Payload{
		Operation: run.op, InputKey: run.input, MaskKey: run.mask, ReferenceKey: run.reference,
		ModelID: plan.ModelID, Backend: plan.Provider, Tier: plan.Tier, GPU: plan.GPUViable,
		CapacityPriority: parentPayload.CapacityPriority, AllowReclaim: parentPayload.AllowReclaim,
		Params: params, AutoAcquire: plan.AutoAcquire, Route: plan.Route,
	})
	if err != nil {
		return internaljobs.Result{}, err
	}
	stageJob := internaljobs.Job{
		ID: parent.ID + ":" + label, Operation: run.op, Lane: LaneForTier(plan.Tier),
		Priority: parent.Priority, Payload: raw, TestMode: parent.TestMode,
	}
	return retryWhileBusy(ctx, func() (internaljobs.Result, error) { return e.run(ctx, run.op, stageJob, emit) })
}

// A stage runs while its job is already running, past the point where the job
// manager would put a capacity-refused job back in the queue. So a stage that is
// refused capacity (another job holds the warm worker, the host is full) waits
// and asks again, on the same backoff the queue uses, for up to stageBusyWait.
var (
	stageBusyWait    = 20 * time.Minute
	stageBusyBackoff = []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 15 * time.Second}
)

func retryWhileBusy(ctx context.Context, attempt func() (internaljobs.Result, error)) (internaljobs.Result, error) {
	deadline := time.Now().Add(stageBusyWait)
	for tries := 0; ; tries++ {
		result, err := attempt()
		if err == nil || !errors.Is(err, internaljobs.ErrCapacityBusy) || time.Now().After(deadline) {
			return result, err
		}
		wait := stageBusyBackoff[min(tries, len(stageBusyBackoff)-1)]
		select {
		case <-ctx.Done():
			return result, errors.Join(err, context.Cause(ctx))
		case <-time.After(wait):
		}
	}
}
