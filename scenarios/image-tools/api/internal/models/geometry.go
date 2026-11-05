package models

import (
	"strconv"
	"strings"
)

// Geometry is the canvas a model can actually draw well.
//
// It exists because callers were guessing it. Backdrop Studio carried three
// constants named for Stable Diffusion 1.5 — a 512px native edge, a 64px latent
// stride, a 768px cap — and applied them to whatever model image-tools happened
// to select. On an SDXL or FLUX host that is simply wrong in both directions: it
// throws away half the model's trained resolution, and it quantises to a stride
// the model does not use. Worse, the constants lived in a scenario that by
// charter (OT-P0-004) owns no model configuration at all, so nothing could
// correct them when the installed model set changed.
//
// The facts are properties of the weights, so they are answered here, beside the
// registry that knows which weights exist.
type Geometry struct {
	// NativeWidth and NativeHeight are the resolution the model was trained at.
	// Zero means the model declares none — the cloud/BYOK entries, where the
	// provider owns geometry and the caller may ask for the size it wants.
	NativeWidth, NativeHeight int
	// SizeQuantum is the stride both axes must be a multiple of, imposed by the
	// latent downsampling factor. 1 means "no constraint".
	SizeQuantum int
	// MaxEdge caps the long axis. Zero means uncapped.
	//
	// Aspect is not free on a diffusion model: pushed far past its training
	// resolution on one axis, it stops extending the composition and starts
	// repeating it — two horizons, two suns, a second colonnade — and no
	// downstream treatment repairs that. So a wide delivery surface is drawn at
	// a moderate aspect and cover-cropped, which loses some framing but never
	// produces a duplicated subject.
	MaxEdge int
}

// Declared reports whether the model states a native geometry. A caller that
// gets false must send the geometry it wants rather than quantising against
// zeroes.
func (g Geometry) Declared() bool { return g.NativeWidth > 0 && g.NativeHeight > 0 }

// maxEdgeRatio is how far past the native long edge a diffusion model is asked
// to draw before the composition starts repeating.
//
// One conservative ratio for every architecture rather than a per-model tuning:
// 1.5x is the bound that holds for the weakest architecture in the registry
// (SD-1.5), and a model that could hold more only loses framing, while a model
// that could hold less produces a duplicated subject. Given the asymmetry,
// under-reaching is the correct default until a per-architecture measurement
// exists to raise it.
const maxEdgeRatio = 1.5

// latentQuantum is the pixel stride each architecture's latent space imposes.
// It is a hard property of the VAE downsampling factor, not a tuning: an axis
// off the stride is rounded by the pipeline, and the caller then receives a
// different geometry than it asked for.
var latentQuantum = map[Architecture]int{
	// 8x VAE downsample, UNet requires the latent divisible by 8 → 64px.
	ArchSD15: 64,
	ArchSDXL: 64,
	// FLUX packs latents 2x2 on top of the 8x VAE → 16px.
	ArchFlux: 16,
	// SD-1.5 lineage.
	ArchInstructPix2Pix: 64,
	// Qwen-Image and LongCat edit pipelines align on the 8x VAE alone.
	ArchQwenImageEdit:    8,
	ArchLongCatImageEdit: 8,
}

// Geometry returns the canvas this model draws well, derived from its declared
// native resolution and the latent stride of its architecture.
func (m Model) Geometry() Geometry {
	g := Geometry{SizeQuantum: 1}
	if q, ok := latentQuantum[m.Architecture]; ok {
		g.SizeQuantum = q
	}
	w, h, ok := parseResolution(m.IO.NativeResolution)
	if !ok {
		// "model-dependent" — the cloud entries. Geometry belongs to the
		// provider, so nothing is asserted here and Declared() stays false.
		return g
	}
	g.NativeWidth, g.NativeHeight = w, h
	long := w
	if h > long {
		long = h
	}
	g.MaxEdge = int(float64(long) * maxEdgeRatio)
	return g
}

// Fit maps a delivery size onto a canvas this model can draw: the delivery
// aspect preserved as far as the model can hold it, the short edge at native
// resolution, both axes on the latent stride.
//
// A model with no declared native geometry gets the delivery size back, snapped
// to its stride — the honest answer when the provider owns geometry.
func (g Geometry) Fit(deliveryW, deliveryH int) (int, int) {
	if deliveryW <= 0 || deliveryH <= 0 {
		if g.Declared() {
			return g.NativeWidth, g.NativeHeight
		}
		return 0, 0
	}
	if !g.Declared() {
		return g.snap(deliveryW), g.snap(deliveryH)
	}
	aspect := float64(deliveryW) / float64(deliveryH)
	width, height := g.NativeWidth, g.NativeHeight
	if aspect >= 1 {
		width = int(float64(g.NativeHeight) * aspect)
		if g.MaxEdge > 0 && width > g.MaxEdge {
			width = g.MaxEdge
		}
	} else {
		height = int(float64(g.NativeWidth) / aspect)
		if g.MaxEdge > 0 && height > g.MaxEdge {
			height = g.MaxEdge
		}
	}
	return g.snap(width), g.snap(height)
}

func (g Geometry) snap(v int) int {
	q := g.SizeQuantum
	if q <= 1 {
		return v
	}
	if v < q {
		return q
	}
	return (v / q) * q
}

// parseResolution reads the "WxH" form the registry declares. Anything else —
// "model-dependent", an empty string — is not a resolution and is reported as
// absent rather than guessed at.
func parseResolution(raw string) (int, int, bool) {
	parts := strings.SplitN(strings.TrimSpace(strings.ToLower(raw)), "x", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	w, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || w <= 0 {
		return 0, 0, false
	}
	h, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || h <= 0 {
		return 0, 0, false
	}
	return w, h, true
}
