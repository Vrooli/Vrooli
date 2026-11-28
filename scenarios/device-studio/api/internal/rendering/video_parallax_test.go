package rendering

import (
	"context"
	"image"
	"image/color"
	"testing"
)

func TestLayeredParallaxUsesMaskedDepthOffsetAndKeepsFrameZeroExact(t *testing.T) {
	background := image.NewNRGBA(image.Rect(0, 0, 4, 1))
	foreground := image.NewNRGBA(background.Bounds())
	mask := image.NewGray(background.Bounds())
	activeMask := image.NewGray(background.Bounds())
	for x := 0; x < 4; x++ {
		background.SetNRGBA(x, 0, color.NRGBA{B: 10, A: 255})
	}
	foreground.SetNRGBA(1, 0, color.NRGBA{R: 250, A: 255})
	mask.SetGray(1, 0, color.Gray{Y: 255})
	inputs := []videoLoadedImage{{image: background}, {image: activeMask}, {image: foreground}, {image: mask}}
	scene := VideoScene{Layers: []VideoParallaxLayer{{DepthBasis: 1000}}}
	motion := &LayeredParallax{ProfileVersion: 1, DurationOutputFrames: 2, OverscanPixels: 1, TravelXPixels: 1}
	layers, err := prepareParallaxLayers(context.Background(), inputs, scene, background.Bounds(), motion)
	if err != nil {
		t.Fatal(err)
	}
	first, err := composeParallaxPlate(context.Background(), background, layers, motion, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := color.NRGBAModel.Convert(first.At(1, 0)).(color.NRGBA); got != (color.NRGBA{R: 250, A: 255}) {
		t.Fatalf("frame zero differs from authored layer composite: %+v", got)
	}
	last, err := composeParallaxPlate(context.Background(), background, layers, motion, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got := color.NRGBAModel.Convert(last.At(2, 0)).(color.NRGBA); got != (color.NRGBA{R: 250, A: 255}) {
		t.Fatalf("depth layer did not reach its bounded endpoint: %+v", got)
	}
	if got := color.NRGBAModel.Convert(last.At(1, 0)).(color.NRGBA); got != (color.NRGBA{B: 10, A: 255}) {
		t.Fatalf("mask leaked the shifted layer at its old position: %+v", got)
	}
	activeMask.SetGray(2, 0, color.Gray{Y: 255})
	inputs[1] = videoLoadedImage{image: activeMask}
	if _, err := prepareParallaxLayers(context.Background(), inputs, scene, background.Bounds(), motion); err == nil {
		t.Fatal("camera sweep that may enter the active screen was accepted")
	}
}
