package filepreview

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif" // registers the GIF decoder for image.Decode
	"image/jpeg"
	"image/png"
	"io"
	"os"
)

// Thumbnail bounds. A thumbnail is a glance, not a preview: the source is
// capped so one request cannot pin a large decode on the host, and the decoded
// pixel count is checked from the header before any pixels are allocated.
const (
	ThumbnailMinEdge     = 32
	ThumbnailMaxEdge     = 512
	ThumbnailDefaultEdge = 160
	thumbnailMaxSource   = 48 << 20
	thumbnailMaxPixels   = 64_000_000
	thumbnailJPEGQuality = 80
)

// ErrThumbnailUnsupported means the file cannot be thumbnailed here (an
// unsupported format such as WebP/HEIC, or one too large to decode safely).
// Callers fall back to a generic icon; it is not a server fault.
var ErrThumbnailUnsupported = errors.New("thumbnail unsupported")

// Thumbnail decodes a JPEG, PNG or GIF (first frame) at path and returns it
// scaled to fit within edge×edge, never enlarged, with JPEG EXIF orientation
// applied so phone photos are upright. Opaque sources encode as JPEG; sources
// that can carry transparency encode as PNG.
func Thumbnail(path string, edge int) ([]byte, string, error) {
	edge = max(ThumbnailMinEdge, min(ThumbnailMaxEdge, edge))
	f, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, "", err
	}
	if !info.Mode().IsRegular() || info.Size() > thumbnailMaxSource {
		return nil, "", ErrThumbnailUnsupported
	}
	data, err := io.ReadAll(io.LimitReader(f, thumbnailMaxSource+1))
	if err != nil {
		return nil, "", err
	}

	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png" && format != "gif") {
		return nil, "", ErrThumbnailUnsupported
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > thumbnailMaxPixels {
		return nil, "", ErrThumbnailUnsupported
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", ErrThumbnailUnsupported
	}
	// Orient after scaling: the fit is symmetric, and rotating the small
	// result is far cheaper than rotating the full-size photo.
	out := scaleToFit(src, edge)
	if format == "jpeg" {
		out = orient(out, jpegOrientation(data))
	}
	var buf bytes.Buffer
	if format == "jpeg" {
		err = jpeg.Encode(&buf, out, &jpeg.Options{Quality: thumbnailJPEGQuality})
		return buf.Bytes(), "image/jpeg", err
	}
	err = (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(&buf, out)
	return buf.Bytes(), "image/png", err
}

// scaleToFit box-filters src down so its longer side is at most edge. Each
// destination pixel averages an evenly spread grid of up to 4×4 of the source
// pixels it covers: enough to avoid nearest-neighbour aliasing, while bounding
// the work by the thumbnail's size rather than the photo's.
func scaleToFit(src image.Image, edge int) *image.NRGBA {
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	dw, dh := sw, sh
	if sw > edge || sh > edge {
		if sw >= sh {
			dw, dh = edge, max(1, sh*edge/sw)
		} else {
			dw, dh = max(1, sw*edge/sh), edge
		}
	}
	dst := image.NewNRGBA(image.Rect(0, 0, dw, dh))
	if dw == sw && dh == sh {
		draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
		return dst
	}
	for y := 0; y < dh; y++ {
		y0, y1 := b.Min.Y+y*sh/dh, b.Min.Y+max((y+1)*sh/dh, y*sh/dh+1)
		for x := 0; x < dw; x++ {
			x0, x1 := b.Min.X+x*sw/dw, b.Min.X+max((x+1)*sw/dw, x*sw/dw+1)
			var r, g, bl, a, n uint64
			for _, sy := range spread(y0, y1) {
				for _, sx := range spread(x0, x1) {
					c := color.NRGBA64Model.Convert(src.At(sx, sy)).(color.NRGBA64)
					r, g, bl, a = r+uint64(c.R), g+uint64(c.G), bl+uint64(c.B), a+uint64(c.A)
					n++
				}
			}
			dst.SetNRGBA(x, y, color.NRGBA{R: uint8(r / n >> 8), G: uint8(g / n >> 8), B: uint8(bl / n >> 8), A: uint8(a / n >> 8)})
		}
	}
	return dst
}

// spread returns up to four evenly spaced sample positions in [lo, hi).
func spread(lo, hi int) []int {
	n := min(4, hi-lo)
	out := make([]int, n)
	for i := range out {
		out[i] = lo + (2*i+1)*(hi-lo)/(2*n)
	}
	return out
}

// orient applies an EXIF orientation (1–8) to img. 1 and unknown values are
// returned unchanged.
func orient(img *image.NRGBA, o int) *image.NRGBA {
	if o < 2 || o > 8 {
		return img
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	swap := o >= 5
	dw, dh := w, h
	if swap {
		dw, dh = h, w
	}
	dst := image.NewNRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var dx, dy int
			switch o {
			case 2:
				dx, dy = w-1-x, y
			case 3:
				dx, dy = w-1-x, h-1-y
			case 4:
				dx, dy = x, h-1-y
			case 5:
				dx, dy = y, x
			case 6:
				dx, dy = h-1-y, x
			case 7:
				dx, dy = h-1-y, w-1-x
			case 8:
				dx, dy = y, w-1-x
			}
			dst.SetNRGBA(dx, dy, img.NRGBAAt(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

// jpegOrientation reads the EXIF Orientation tag from a JPEG's APP1 segment,
// returning 1 (upright) when there is none or it cannot be parsed.
func jpegOrientation(data []byte) int {
	r := bufio.NewReader(bytes.NewReader(data))
	var soi [2]byte
	if _, err := io.ReadFull(r, soi[:]); err != nil || soi != [2]byte{0xFF, 0xD8} {
		return 1
	}
	for {
		var marker [2]byte
		if _, err := io.ReadFull(r, marker[:]); err != nil || marker[0] != 0xFF {
			return 1
		}
		// Start of scan or end of image: no EXIF before the pixels.
		if marker[1] == 0xDA || marker[1] == 0xD9 {
			return 1
		}
		var size [2]byte
		if _, err := io.ReadFull(r, size[:]); err != nil {
			return 1
		}
		n := int(binary.BigEndian.Uint16(size[:])) - 2
		if n < 0 {
			return 1
		}
		seg := make([]byte, n)
		if _, err := io.ReadFull(r, seg); err != nil {
			return 1
		}
		if marker[1] == 0xE1 && len(seg) > 6 && string(seg[:6]) == "Exif\x00\x00" {
			return exifOrientation(seg[6:])
		}
	}
}

func exifOrientation(tiff []byte) int {
	if len(tiff) < 8 {
		return 1
	}
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 1
	}
	ifd := int(order.Uint32(tiff[4:8]))
	if ifd < 8 || ifd+2 > len(tiff) {
		return 1
	}
	count := int(order.Uint16(tiff[ifd : ifd+2]))
	for i := 0; i < count; i++ {
		off := ifd + 2 + i*12
		if off+12 > len(tiff) {
			return 1
		}
		if order.Uint16(tiff[off:off+2]) == 0x0112 {
			return int(order.Uint16(tiff[off+8 : off+10]))
		}
	}
	return 1
}
