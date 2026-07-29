package main

import (
	"image"
	"image/color"
	"math"
)

// resizeContain centers a nearest-neighbor resize without enlarging the source.
func resizeContain(src image.Image, targetW, targetH int) *image.RGBA {
	if targetW <= 0 || targetH <= 0 {
		return nil
	}

	srcBounds := src.Bounds()
	srcW, srcH := srcBounds.Dx(), srcBounds.Dy()
	if srcW == 0 || srcH == 0 {
		return nil
	}

	scale := math.Min(float64(targetW)/float64(srcW), float64(targetH)/float64(srcH))
	if scale > 1 {
		scale = 1
	}
	dstW, dstH := int(float64(srcW)*scale), int(float64(srcH)*scale)
	if dstW == 0 || dstH == 0 {
		return nil
	}

	dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	for y := 0; y < targetH; y++ {
		for x := 0; x < targetW; x++ {
			dst.Set(x, y, color.RGBA{})
		}
	}
	xOffset, yOffset := (targetW-dstW)/2, (targetH-dstH)/2
	for y := 0; y < dstH; y++ {
		for x := 0; x < dstW; x++ {
			srcX := int(float64(x)/scale + 0.5)
			srcY := int(float64(y)/scale + 0.5)
			dst.Set(x+xOffset, y+yOffset, src.At(srcBounds.Min.X+srcX, srcBounds.Min.Y+srcY))
		}
	}
	return dst
}
