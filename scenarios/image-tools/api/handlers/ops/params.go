package ops

import (
	"fmt"

	internalops "image-tools/internal/ops"

	opsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ops"
)

// translateParams converts the proto OpParams oneof into the flat internal
// params for the named operation. It enforces that the set oneof case matches
// the {operation} path segment, so a request can't claim "resize" while sending
// crop params. A nil OpParams (no params part) yields an empty, op-defaulted
// params set — valid for ops whose defaults are meaningful (e.g. metadata read).
func translateParams(operation string, pb *opsv1.OpParams) (*internalops.Params, error) {
	p := &internalops.Params{}
	if pb == nil {
		return p, nil
	}
	switch operation {
	case "resize":
		if v := pb.GetResize(); v != nil {
			p.Width, p.Height, p.Fit, p.Gravity = int(v.GetWidth()), int(v.GetHeight()), v.GetFit(), v.GetGravity()
		}
	case "crop":
		if v := pb.GetCrop(); v != nil {
			p.X, p.Y, p.Width, p.Height, p.Gravity = int(v.GetX()), int(v.GetY()), int(v.GetWidth()), int(v.GetHeight()), v.GetGravity()
		}
	case "rotate":
		if v := pb.GetRotate(); v != nil {
			p.Angle, p.Expand, p.Background = v.GetAngle(), v.GetExpand(), v.GetBackground()
		}
	case "flip":
		if v := pb.GetFlip(); v != nil {
			p.Axis = v.GetAxis()
		}
	case "deskew":
		if v := pb.GetDeskew(); v != nil {
			p.Background = v.GetBackground()
		}
	case "thumbnail":
		if v := pb.GetThumbnail(); v != nil {
			p.Width, p.Height = int(v.GetWidth()), int(v.GetHeight())
		}
	case "canvas":
		if v := pb.GetCanvas(); v != nil {
			p.Width, p.Height, p.Background, p.Gravity = int(v.GetWidth()), int(v.GetHeight()), v.GetBackground(), v.GetGravity()
		}
	case "adjust":
		if v := pb.GetAdjust(); v != nil {
			p.Brightness, p.Contrast, p.Gamma, p.Saturation, p.Hue = v.GetBrightness(), v.GetContrast(), v.GetGamma(), v.GetSaturation(), v.GetHue()
		}
	case "filter":
		if v := pb.GetFilter(); v != nil {
			p.Filter, p.Amount = v.GetFilter(), v.GetAmount()
		}
	case "convert":
		if v := pb.GetConvert(); v != nil {
			p.Format, p.Quality, p.Lossless = v.GetFormat(), int(v.GetQuality()), v.GetLossless()
		}
	case "compress":
		if v := pb.GetCompress(); v != nil {
			p.Format, p.Quality, p.Lossless, p.TargetBytes = v.GetFormat(), int(v.GetQuality()), v.GetLossless(), v.GetTargetBytes()
		}
	case "overlay":
		if v := pb.GetOverlay(); v != nil {
			p.Text, p.Position, p.Opacity, p.Color, p.FontSize = v.GetText(), v.GetPosition(), v.GetOpacity(), v.GetColor(), v.GetFontSize()
		}
	case "metadata":
		if v := pb.GetMetadata(); v != nil {
			p.StripAll, p.StripGPS, p.AutoOrient = v.GetStripAll(), v.GetStripGps(), v.GetAutoOrient()
		}
	case "duotone":
		if v := pb.GetDuotone(); v != nil {
			p.Dark, p.Light, p.Mid, p.MidLow, p.MidHigh = v.GetDark(), v.GetLight(), v.GetMid(), v.GetMidLow(), v.GetMidHigh()
			p.Normalize = v.GetNormalize()
		}
	case "posterize":
		if v := pb.GetPosterize(); v != nil {
			p.Levels, p.Dark, p.Light = int(v.GetLevels()), v.GetDark(), v.GetLight()
			p.Normalize = v.GetNormalize()
		}
	case "halftone":
		if v := pb.GetHalftone(); v != nil {
			p.LPI, p.Angle, p.Dot, p.Dark, p.Light = int(v.GetLpi()), v.GetAngle(), v.GetDot(), v.GetDark(), v.GetLight()
			p.Normalize = v.GetNormalize()
		}
	case "dither_ordered":
		if v := pb.GetDitherOrdered(); v != nil {
			p.Dark, p.Light, p.Normalize = v.GetDark(), v.GetLight(), v.GetNormalize()
		}
	case "dither_diffusion":
		if v := pb.GetDitherDiffusion(); v != nil {
			p.Dark, p.Light, p.Normalize = v.GetDark(), v.GetLight(), v.GetNormalize()
		}
	case "grain":
		if v := pb.GetGrain(); v != nil {
			p.Seed, p.Amount, p.ContrastMultiplier = v.GetSeed(), v.GetAmount(), v.GetContrastMultiplier()
		}
	case "scrim":
		if v := pb.GetScrim(); v != nil {
			p.ScrimColor, p.Opacity, p.Direction = v.GetColor(), v.GetOpacity(), v.GetDirection()
		}
	case "line_screen":
		if v := pb.GetLineScreen(); v != nil {
			p.Spacing, p.Angle, p.Dark, p.Light, p.Normalize = v.GetSpacing(), v.GetAngle(), v.GetDark(), v.GetLight(), v.GetNormalize()
			p.SpacingRel = v.GetSpacingRel()
		}
	case "stipple":
		if v := pb.GetStipple(); v != nil {
			p.Spacing, p.Seed, p.Dark, p.Light, p.Normalize = v.GetSpacing(), v.GetSeed(), v.GetDark(), v.GetLight(), v.GetNormalize()
			p.SpacingRel = v.GetSpacingRel()
		}
	case "engraving":
		if v := pb.GetEngraving(); v != nil {
			p.Spacing, p.Dark, p.Light, p.Normalize = v.GetSpacing(), v.GetDark(), v.GetLight(), v.GetNormalize()
			p.SpacingRel = v.GetSpacingRel()
		}
	case "aberration":
		if v := pb.GetAberration(); v != nil {
			// The implementation consumes Distance (radial px). Amplitude is
			// accepted as the older wire name and folded onto the same knob.
			p.Amplitude, p.Distance = v.GetAmplitude(), int(v.GetDistance())
			if p.Distance == 0 && v.GetAmplitude() > 0 {
				p.Distance = int(v.GetAmplitude())
			}
			p.DistanceRel = v.GetDistanceRel()
		}
	case "bloom":
		if v := pb.GetBloom(); v != nil {
			p.Radius, p.Threshold = int(v.GetRadius()), v.GetThreshold()
			p.RadiusRel = v.GetRadiusRel()
		}
	case "curve":
		if v := pb.GetCurve(); v != nil {
			p.Curve = v.GetExponent()
		}
	case "defocus":
		if v := pb.GetDefocus(); v != nil {
			p.Radius, p.BladeCount = int(v.GetRadius()), int(v.GetBladeCount())
			p.RadiusRel = v.GetRadiusRel()
		}
	case "motion_blur":
		if v := pb.GetMotionBlur(); v != nil {
			p.Distance, p.Angle = int(v.GetDistance()), v.GetAngle()
			p.DistanceRel = v.GetDistanceRel()
		}
	case "ascii_mosaic":
		if v := pb.GetAsciiMosaic(); v != nil {
			p.BlockSize, p.Dark, p.Light, p.Normalize = int(v.GetBlockSize()), v.GetDark(), v.GetLight(), v.GetNormalize()
			p.BlockSizeRel = v.GetBlockSizeRel()
		}
	case "pixel_sort":
		if v := pb.GetPixelSort(); v != nil {
			p.Threshold, p.Axis = v.GetThreshold(), v.GetAxis()
		}
	case "displacement":
		if v := pb.GetDisplacement(); v != nil {
			p.Amplitude, p.Seed, p.Spacing = v.GetAmplitude(), v.GetSeed(), v.GetSpacing()
			p.SpacingRel, p.AmplitudeRel = v.GetSpacingRel(), v.GetAmplitudeRel()
		}
	default:
		return nil, fmt.Errorf("unknown operation %q", operation)
	}
	return p, nil
}
