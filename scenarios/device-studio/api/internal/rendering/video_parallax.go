package rendering

import (
	"context"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"math"
)

type preparedParallaxLayer struct {
	image *image.NRGBA
	depth uint32
}

func prepareParallaxLayers(ctx context.Context, images []videoLoadedImage, scene VideoScene, plateBounds image.Rectangle, motion *LayeredParallax) ([]preparedParallaxLayer, error) {
	if len(scene.Layers) == 0 {
		return nil, nil
	}
	firstLayer := 2
	if scene.HardwareMask != nil {
		firstLayer++
	}
	if len(images) != firstLayer+2*len(scene.Layers) {
		return nil, errors.New("layered-parallax image bindings do not match the loaded inputs")
	}
	prepared := make([]preparedParallaxLayer, 0, len(scene.Layers))
	activeMask := images[1].image
	if !activeMask.Bounds().Eq(plateBounds) {
		return nil, errors.New("parallax active mask dimensions differ from the plate")
	}
	activeBounds, activeVisible, err := nonZeroMaskBounds(ctx, activeMask)
	if err != nil {
		return nil, err
	}
	for i, layer := range scene.Layers {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		foreground, ok := images[firstLayer+2*i].image.(*image.NRGBA)
		if !ok {
			converted := image.NewNRGBA(images[firstLayer+2*i].image.Bounds())
			draw.Draw(converted, converted.Bounds(), images[firstLayer+2*i].image, converted.Bounds().Min, draw.Src)
			foreground = converted
		}
		mask := images[firstLayer+2*i+1].image
		if !foreground.Bounds().Eq(mask.Bounds()) || !foreground.Bounds().Eq(plateBounds) {
			return nil, errors.New("parallax layer and mask dimensions differ")
		}
		layerBounds, layerVisible, err := nonZeroMaskBounds(ctx, mask)
		if err != nil {
			return nil, err
		}
		if !layerVisible {
			return nil, errors.New("parallax layer mask contains no visible pixels")
		}
		maxDX := int(math.Ceil(math.Abs(float64(motion.TravelXPixels) * float64(layer.DepthBasis) / 1000)))
		maxDY := int(math.Ceil(math.Abs(float64(motion.TravelYPixels) * float64(layer.DepthBasis) / 1000)))
		sweptBounds := image.Rect(layerBounds.Min.X-maxDX, layerBounds.Min.Y-maxDY, layerBounds.Max.X+maxDX, layerBounds.Max.Y+maxDY).Intersect(plateBounds)
		if activeVisible && sweptBounds.Overlaps(activeBounds) {
			return nil, errors.New("parallax layer camera sweep may overlap the active screen area")
		}
		masked := image.NewNRGBA(foreground.Bounds())
		for y := foreground.Bounds().Min.Y; y < foreground.Bounds().Max.Y; y++ {
			if y&63 == 0 {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
			}
			for x := foreground.Bounds().Min.X; x < foreground.Bounds().Max.X; x++ {
				pixel := foreground.NRGBAAt(x, y)
				maskPixel := color.NRGBAModel.Convert(mask.At(x, y)).(color.NRGBA)
				coverage := uint8(uint16(maskPixel.R) * uint16(maskPixel.A) / 255)
				pixel.A = uint8(uint16(pixel.A) * uint16(coverage) / 255)
				masked.SetNRGBA(x, y, pixel)
			}
		}
		prepared = append(prepared, preparedParallaxLayer{image: masked, depth: layer.DepthBasis})
	}
	return prepared, nil
}

func nonZeroMaskBounds(ctx context.Context, mask image.Image) (image.Rectangle, bool, error) {
	bounds := image.Rectangle{}
	found := false
	for y := mask.Bounds().Min.Y; y < mask.Bounds().Max.Y; y++ {
		if y&63 == 0 {
			if err := ctx.Err(); err != nil {
				return image.Rectangle{}, false, err
			}
		}
		for x := mask.Bounds().Min.X; x < mask.Bounds().Max.X; x++ {
			p := color.NRGBAModel.Convert(mask.At(x, y)).(color.NRGBA)
			coverage := uint8(uint16(p.R) * uint16(p.A) / 255)
			if coverage == 0 {
				continue
			}
			pixel := image.Rect(x, y, x+1, y+1)
			if !found {
				bounds, found = pixel, true
			} else {
				bounds = bounds.Union(pixel)
			}
		}
	}
	return bounds, found, nil
}

func composeParallaxPlate(ctx context.Context, background image.Image, layers []preparedParallaxLayer, motion *LayeredParallax, outputFrame uint32) (image.Image, error) {
	if motion == nil || len(layers) == 0 {
		return background, nil
	}
	if outputFrame >= motion.DurationOutputFrames {
		outputFrame = motion.DurationOutputFrames - 1
	}
	progress := float64(outputFrame) / float64(motion.DurationOutputFrames-1)
	progress = progress * progress * (3 - 2*progress)
	base := image.NewNRGBA(background.Bounds())
	draw.Draw(base, base.Bounds(), background, background.Bounds().Min, draw.Src)
	for _, layer := range layers {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		depth := float64(layer.depth) / 1000
		dx := int(math.Round(float64(motion.TravelXPixels) * progress * depth))
		dy := int(math.Round(float64(motion.TravelYPixels) * progress * depth))
		destination := layer.image.Bounds().Add(image.Pt(dx, dy)).Intersect(base.Bounds())
		if destination.Empty() {
			continue
		}
		source := destination.Sub(image.Pt(dx, dy))
		draw.Draw(base, destination, layer.image, source.Min, draw.Over)
	}
	return base, nil
}
