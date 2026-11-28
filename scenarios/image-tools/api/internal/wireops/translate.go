// Package wireops is the adapter between the typed deterministic wire contract
// (ops.v1.OpParams) and the flat parameter struct the compute layer consumes
// (internal/ops.Params).
//
// The per-operation bindings are generated from the proto descriptor
// (bindings.generated.go, via `make capability-gen`), so adding a parameter to
// ops.proto makes it reachable at the REST edge without a handler edit, and a
// parameter can never be silently dropped between the wire and the pixel code.
//
// This package exists instead of putting the translation in internal/ops because
// the compute layer must not know the transport, and instead of putting it in
// handlers/ops because the binding table needs the proto and the compute struct
// in one place.
package wireops

import (
	"fmt"

	opsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ops"

	"image-tools/internal/ops"
)

// Translate maps a request's typed parameters onto the flat compute parameters
// for one operation. A nil envelope is valid and yields zero parameters, which is
// how a caller runs an operation with all defaults.
func Translate(operation string, pb *opsv1.OpParams) (*ops.Params, error) {
	p := &ops.Params{}
	if pb == nil {
		return p, nil
	}
	if bindParams(operation, pb, p) {
		// scalar operation, nothing layered to read
	} else if ok, err := bindLayered(operation, pb, p); err != nil {
		return nil, err
	} else if !ok {
		return nil, fmt.Errorf("unknown operation %q", operation)
	}
	// Read after the operation-specific binding, not inside it: the reserve is a
	// sibling of the oneof and applies to whichever case was set, so every colour
	// operation gets it without seventeen copies of the same lines. The registry
	// decides what to do with it — colour operations honour it, geometry ones
	// ignore it, because a resize moves the frame out from under the rectangle.
	if k := pb.GetKnockout(); k != nil {
		p.KnockoutX, p.KnockoutY = k.GetX(), k.GetY()
		p.KnockoutWidth, p.KnockoutHeight = k.GetWidth(), k.GetHeight()
		p.KnockoutFeather = k.GetFeather()
		p.KnockoutSolid = k.GetSolid()
	}
	return p, nil
}

// bindLayered handles the operations whose parameters are not a flat set of
// scalars: they carry multipart plate stacks and camera messages, which the
// generator rejects rather than skipping. Reports whether the operation matched.
func bindLayered(operation string, pb *opsv1.OpParams, p *ops.Params) (bool, error) {
	switch operation {
	case "composite", "parallax":
	default:
		return false, nil
	}
	scene := pb.GetComposite()
	if operation == "parallax" {
		q := pb.GetParallax()
		if q == nil {
			return true, nil
		}
		scene = q.GetScene()
		if q.GetStartCamera() == nil || q.GetEndCamera() == nil {
			return false, fmt.Errorf("parallax start_camera and end_camera are required")
		}
		p.Parallax = &ops.ParallaxSequence{
			Start:  layeredCamera(q.GetStartCamera()),
			End:    layeredCamera(q.GetEndCamera()),
			Frames: q.GetFrames(),
			FPS:    q.GetFramesPerSecond(),
		}
	}
	if scene != nil {
		p.Width, p.Height, p.Background = int(scene.GetWidth()), int(scene.GetHeight()), scene.GetBackground()
		if c := scene.GetCamera(); c != nil {
			camera := layeredCamera(c)
			p.Camera = &camera
		}
		for _, plate := range scene.GetPlates() {
			p.Plates = append(p.Plates, ops.PlateSpec{
				Name: plate.GetName(), Depth: int(plate.GetDepth()), Blend: plate.GetBlend(),
				Opacity: plate.GetOpacity(), Distance: plate.Distance,
			})
		}
	}
	return true, nil
}

func layeredCamera(c *opsv1.LayeredCamera) ops.LayeredCamera {
	return ops.LayeredCamera{
		X: c.GetX(), Y: c.GetY(), Z: c.GetZ(),
		YawDegrees: c.GetYawDegrees(), PitchDegrees: c.GetPitchDegrees(),
		FocalLengthPixels: c.GetFocalLengthPixels(), RequireOpaque: c.GetRequireOpaque(),
	}
}
