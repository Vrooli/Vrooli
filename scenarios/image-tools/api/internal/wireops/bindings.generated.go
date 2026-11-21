// AUTO-GENERATED — do not edit by hand.
//
// Source : packages/proto/schemas/image-tools/v1/ops/ops.proto (OpParams)
// Codegen: api/cmd/gen-capability (invoked by `make capability-gen`)
// Drift  : api/internal/wireops/render_test.go fails the build if this file is stale.
//
// One case per operation whose parameters are flat scalars. The layered views
// (composite, parallax) are bound by hand in translate.go.

package wireops

import (
	opsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ops"

	"image-tools/internal/ops"
)

// bindParams sets the scalar parameters of one operation. It reports whether
// the operation is a generated one.
func bindParams(operation string, pb *opsv1.OpParams, p *ops.Params) bool {
	switch operation {
	case "aberration":
		if v := pb.GetAberration(); v != nil {
			p.Amplitude = v.GetAmplitude()
			p.Distance = int(v.GetDistance())
			p.DistanceRel = v.GetDistanceRel()
		}
		return true
	case "adjust":
		if v := pb.GetAdjust(); v != nil {
			p.Brightness = v.GetBrightness()
			p.Contrast = v.GetContrast()
			p.Gamma = v.GetGamma()
			p.Saturation = v.GetSaturation()
			p.Hue = v.GetHue()
		}
		return true
	case "ascii_mosaic":
		if v := pb.GetAsciiMosaic(); v != nil {
			p.BlockSize = int(v.GetBlockSize())
			p.Dark = v.GetDark()
			p.Light = v.GetLight()
			p.Normalize = v.GetNormalize()
			p.BlockSizeRel = v.GetBlockSizeRel()
		}
		return true
	case "bloom":
		if v := pb.GetBloom(); v != nil {
			p.Radius = int(v.GetRadius())
			p.Threshold = v.GetThreshold()
			p.RadiusRel = v.GetRadiusRel()
		}
		return true
	case "canny":
		if v := pb.GetCanny(); v != nil {
			p.Amount = v.GetAmount()
			p.LowThreshold = v.GetLowThreshold()
			p.HighThreshold = v.GetHighThreshold()
		}
		return true
	case "canvas":
		if v := pb.GetCanvas(); v != nil {
			p.Width = int(v.GetWidth())
			p.Height = int(v.GetHeight())
			p.Background = v.GetBackground()
			p.Gravity = v.GetGravity()
		}
		return true
	case "compress":
		if v := pb.GetCompress(); v != nil {
			p.Format = v.GetFormat()
			p.Quality = int(v.GetQuality())
			p.Lossless = v.GetLossless()
			p.TargetBytes = v.GetTargetBytes()
		}
		return true
	case "convert":
		if v := pb.GetConvert(); v != nil {
			p.Format = v.GetFormat()
			p.Quality = int(v.GetQuality())
			p.Lossless = v.GetLossless()
		}
		return true
	case "crop":
		if v := pb.GetCrop(); v != nil {
			p.X = int(v.GetX())
			p.Y = int(v.GetY())
			p.Width = int(v.GetWidth())
			p.Height = int(v.GetHeight())
			p.Gravity = v.GetGravity()
		}
		return true
	case "curve":
		if v := pb.GetCurve(); v != nil {
			p.Curve = v.GetExponent()
		}
		return true
	case "defocus":
		if v := pb.GetDefocus(); v != nil {
			p.Radius = int(v.GetRadius())
			p.BladeCount = int(v.GetBladeCount())
			p.RadiusRel = v.GetRadiusRel()
		}
		return true
	case "deskew":
		if v := pb.GetDeskew(); v != nil {
			p.Background = v.GetBackground()
		}
		return true
	case "displacement":
		if v := pb.GetDisplacement(); v != nil {
			p.Amplitude = v.GetAmplitude()
			p.Seed = v.GetSeed()
			p.Spacing = v.GetSpacing()
			p.SpacingRel = v.GetSpacingRel()
			p.AmplitudeRel = v.GetAmplitudeRel()
		}
		return true
	case "dither_diffusion":
		if v := pb.GetDitherDiffusion(); v != nil {
			p.Dark = v.GetDark()
			p.Light = v.GetLight()
			p.Normalize = v.GetNormalize()
		}
		return true
	case "dither_ordered":
		if v := pb.GetDitherOrdered(); v != nil {
			p.Dark = v.GetDark()
			p.Light = v.GetLight()
			p.Normalize = v.GetNormalize()
		}
		return true
	case "duotone":
		if v := pb.GetDuotone(); v != nil {
			p.Dark = v.GetDark()
			p.Light = v.GetLight()
			p.Mid = v.GetMid()
			p.MidLow = v.GetMidLow()
			p.MidHigh = v.GetMidHigh()
			p.Normalize = v.GetNormalize()
		}
		return true
	case "engraving":
		if v := pb.GetEngraving(); v != nil {
			p.Spacing = v.GetSpacing()
			p.Dark = v.GetDark()
			p.Light = v.GetLight()
			p.Normalize = v.GetNormalize()
			p.SpacingRel = v.GetSpacingRel()
		}
		return true
	case "filter":
		if v := pb.GetFilter(); v != nil {
			p.Filter = v.GetFilter()
			p.Amount = v.GetAmount()
		}
		return true
	case "flip":
		if v := pb.GetFlip(); v != nil {
			p.Axis = v.GetAxis()
		}
		return true
	case "grain":
		if v := pb.GetGrain(); v != nil {
			p.Seed = v.GetSeed()
			p.Amount = v.GetAmount()
			p.ContrastMultiplier = v.GetContrastMultiplier()
		}
		return true
	case "halftone":
		if v := pb.GetHalftone(); v != nil {
			p.LPI = int(v.GetLpi())
			p.Angle = v.GetAngle()
			p.Dot = v.GetDot()
			p.Dark = v.GetDark()
			p.Light = v.GetLight()
			p.Normalize = v.GetNormalize()
		}
		return true
	case "icon_container":
		if v := pb.GetIconContainer(); v != nil {
			p.ContainerFormat = v.GetFormat()
			for _, item := range v.GetSizes() {
				p.Sizes = append(p.Sizes, int(item))
			}
		}
		return true
	case "line_screen":
		if v := pb.GetLineScreen(); v != nil {
			p.Spacing = v.GetSpacing()
			p.Angle = v.GetAngle()
			p.Dark = v.GetDark()
			p.Light = v.GetLight()
			p.Normalize = v.GetNormalize()
			p.SpacingRel = v.GetSpacingRel()
		}
		return true
	case "metadata":
		if v := pb.GetMetadata(); v != nil {
			p.StripAll = v.GetStripAll()
			p.StripGPS = v.GetStripGps()
			p.AutoOrient = v.GetAutoOrient()
		}
		return true
	case "motion_blur":
		if v := pb.GetMotionBlur(); v != nil {
			p.Distance = int(v.GetDistance())
			p.Angle = v.GetAngle()
			p.DistanceRel = v.GetDistanceRel()
		}
		return true
	case "overlay":
		if v := pb.GetOverlay(); v != nil {
			p.Text = v.GetText()
			p.Position = v.GetPosition()
			p.Opacity = v.GetOpacity()
			p.Color = v.GetColor()
			p.FontSize = v.GetFontSize()
		}
		return true
	case "pixel_sort":
		if v := pb.GetPixelSort(); v != nil {
			p.Threshold = v.GetThreshold()
			p.Axis = v.GetAxis()
		}
		return true
	case "posterize":
		if v := pb.GetPosterize(); v != nil {
			p.Levels = int(v.GetLevels())
			p.Dark = v.GetDark()
			p.Light = v.GetLight()
			p.Normalize = v.GetNormalize()
		}
		return true
	case "rasterize":
		if v := pb.GetRasterize(); v != nil {
			p.Width = int(v.GetWidth())
			p.Height = int(v.GetHeight())
			p.Background = v.GetBackground()
		}
		return true
	case "resize":
		if v := pb.GetResize(); v != nil {
			p.Width = int(v.GetWidth())
			p.Height = int(v.GetHeight())
			p.Fit = v.GetFit()
			p.Gravity = v.GetGravity()
		}
		return true
	case "rotate":
		if v := pb.GetRotate(); v != nil {
			p.Angle = v.GetAngle()
			p.Expand = v.GetExpand()
			p.Background = v.GetBackground()
		}
		return true
	case "screenshot_cleanup":
		if v := pb.GetScreenshotCleanup(); v != nil {
			p.Tolerance = v.GetTolerance()
			p.MaxTrimPercent = v.GetMaxTrimPercent()
			p.PaddingPx = int(v.GetPaddingPx())
		}
		return true
	case "scrim":
		if v := pb.GetScrim(); v != nil {
			p.ScrimColor = v.GetColor()
			p.Opacity = v.GetOpacity()
			p.Direction = v.GetDirection()
			p.RegionX = v.GetRegionX()
			p.RegionY = v.GetRegionY()
			p.RegionWidth = v.GetRegionWidth()
			p.RegionHeight = v.GetRegionHeight()
			p.RegionFeather = v.GetRegionFeather()
		}
		return true
	case "stipple":
		if v := pb.GetStipple(); v != nil {
			p.Spacing = v.GetSpacing()
			p.Seed = v.GetSeed()
			p.Dark = v.GetDark()
			p.Light = v.GetLight()
			p.Normalize = v.GetNormalize()
			p.SpacingRel = v.GetSpacingRel()
		}
		return true
	case "thumbnail":
		if v := pb.GetThumbnail(); v != nil {
			p.Width = int(v.GetWidth())
			p.Height = int(v.GetHeight())
		}
		return true
	case "vectorize":
		if v := pb.GetVectorize(); v != nil {
			p.Colors = int(v.GetColors())
			p.KeepColors = append(p.KeepColors, v.GetKeepColors()...)
			p.DropBackgroundLayers = v.GetDropBackgroundLayers()
			p.ClipToLargestRoundedRegion = v.GetClipToLargestRoundedRegion()
			p.InsetPx = v.GetInsetPx()
			p.TolerancePx = v.GetTolerancePx()
			p.Smoothing = v.GetSmoothing()
			p.MinAreaPx = v.GetMinAreaPx()
		}
		return true
	}
	return false
}
