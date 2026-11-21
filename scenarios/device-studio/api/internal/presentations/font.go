package presentations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"math"
	"strconv"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

func fontDigest() string {
	sum := sha256.Sum256(goregular.TTF)
	return hex.EncodeToString(sum[:])
}

func newPinnedFace(size float64) (font.Face, error) {
	f, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingNone})
}

func validateGlyphs(c Config) error {
	texts := []string{}
	if c.Phone != nil {
		texts = append(texts, c.Phone.Clock, c.Phone.Carrier)
		if c.Phone.ShowBatteryPercent {
			texts = append(texts, percentString(c.Phone.BatteryPercent))
		}
	}
	if c.Browser != nil {
		texts = append(texts, c.Browser.Title, c.Browser.Address)
	}
	face, err := newPinnedFace(14)
	if err != nil {
		return err
	}
	defer face.Close()
	for _, text := range texts {
		for _, r := range text {
			if _, ok := face.GlyphAdvance(r); !ok {
				return ErrUnsupportedGlyph
			}
		}
	}
	return nil
}

func validatePhoneCarrierFit(text string, target image.Point) error {
	if text == "" {
		return nil
	}
	sx, sy := float64(target.X)/phoneRefWidth, float64(target.Y)/phoneRefHeight
	size := 11 * math.Min(sx, sy)
	face, err := newPinnedFace(size)
	if err != nil {
		return err
	}
	defer face.Close()
	for _, r := range text {
		if _, ok := face.GlyphAdvance(r); !ok {
			return ErrUnsupportedGlyph
		}
	}
	drawer := font.Drawer{Face: face}
	bounds, _ := drawer.BoundString(text)
	maxWidth := int(math.Round(118 * sx))
	baseline := int(math.Round(67 * sy))
	safeTop, safeBottom := int(math.Round(52*sy)), int(math.Round(72*sy))
	if maxWidth <= 0 || drawer.MeasureString(text).Ceil() > maxWidth || bounds.Min.X.Floor() < 0 || bounds.Max.X.Ceil() > maxWidth || baseline+bounds.Min.Y.Floor() < safeTop || baseline+bounds.Max.Y.Ceil() > safeBottom {
		return ErrCarrierDoesNotFit
	}
	return nil
}

func drawPinnedText(ctx context.Context, dst *image.NRGBA, text string, x, baseline, maxWidth int, size float64, ink color.NRGBA) error {
	if text == "" {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	face, err := newPinnedFace(size)
	if err != nil {
		return err
	}
	defer face.Close()
	if maxWidth <= 0 {
		return nil
	}
	d := font.Drawer{Dst: dst, Src: image.NewUniform(ink), Face: face}
	if d.MeasureString(text).Ceil() > maxWidth {
		const ellipsis = "..."
		for len(text) > 0 && d.MeasureString(text+ellipsis).Ceil() > maxWidth {
			_, n := utf8.DecodeLastRuneInString(text)
			text = text[:len(text)-n]
		}
		if text == "" {
			return nil
		}
		text += ellipsis
	}
	d.Dot = fixed.P(x, baseline)
	d.DrawString(text)
	return ctx.Err()
}

func percentString(v uint8) string { return strconv.Itoa(int(v)) + "%" }
