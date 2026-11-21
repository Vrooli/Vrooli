package ops

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"sort"

	"github.com/disintegration/imaging"
)

// icnsTypeForSize maps an ICNS OSType to its pixel size. The types in use cover
// the standard and @2x apple icon variants electron-builder expects.
var icnsTypes = []struct {
	Type string
	Size int
}{
	{"ic11", 32},
	{"ic12", 64},
	{"ic07", 128},
	{"ic08", 256},
	{"ic13", 256},
	{"ic09", 512},
	{"ic14", 512},
	{"ic10", 1024},
}

// IconContainer packs PNG renders of one source into an ICO or ICNS container.
// Input may be SVG (rendered at each size) or a raster (resized). The sizes are
// the requested pixel sizes; for ICNS they are matched to the OSType table.
func IconContainer(in RunInput) (RunResult, error) {
	format := normalizeFormat(in.Params.ContainerFormat)
	sizes := in.Params.Sizes
	if len(sizes) == 0 {
		return RunResult{}, fmt.Errorf("ops: icon_container requires at least one size")
	}
	isSVG := looksLikeSVG(in.Bytes)

	render := func(size int) (image.Image, error) {
		if size <= 0 || size > MaxSVGRasterDimension {
			return nil, fmt.Errorf("ops: icon_container size %d out of range", size)
		}
		if isSVG {
			return rasterizeSVG(in.Bytes, size, size)
		}
		img := imaging.Fit(in.Img, size, size, imaging.Lanczos)
		return img, nil
	}

	switch format {
	case FormatICO:
		imgs := make([]image.Image, 0, len(sizes))
		for _, s := range sizes {
			img, err := render(s)
			if err != nil {
				return RunResult{}, err
			}
			imgs = append(imgs, img)
		}
		data, err := encodeICO(imgs)
		if err != nil {
			return RunResult{}, err
		}
		return RunResult{Bytes: data, Format: FormatICO, Mime: MIMEFor(FormatICO)}, nil
	case FormatICNS:
		bySize := map[int]image.Image{}
		for _, s := range sizes {
			img, err := render(s)
			if err != nil {
				return RunResult{}, err
			}
			bySize[s] = img
		}
		data, err := encodeICNS(bySize)
		if err != nil {
			return RunResult{}, err
		}
		return RunResult{Bytes: data, Format: FormatICNS, Mime: MIMEFor(FormatICNS)}, nil
	default:
		return RunResult{}, fmt.Errorf("ops: icon_container format must be ico or icns, got %q", in.Params.ContainerFormat)
	}
}

// encodeICO writes an ICONDIR with one PNG-payload entry per image. A 256 px
// entry uses width/height byte 0, per the ICO spec.
func encodeICO(imgs []image.Image) ([]byte, error) {
	type entry struct {
		img    image.Image
		width  int
		height int
		data   []byte
	}
	entries := make([]entry, 0, len(imgs))
	for _, img := range imgs {
		b := img.Bounds()
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return nil, err
		}
		entries = append(entries, entry{img: img, width: b.Dx(), height: b.Dy(), data: buf.Bytes()})
	}
	var out bytes.Buffer
	writeU16(&out, 0) // reserved
	writeU16(&out, 1) // type icon
	writeU16(&out, uint16(len(entries)))
	offset := 6 + 16*len(entries)
	for _, e := range entries {
		out.WriteByte(icoDimByte(e.width))
		out.WriteByte(icoDimByte(e.height))
		out.WriteByte(0)  // color count
		out.WriteByte(0)  // reserved
		writeU16(&out, 1) // planes
		writeU16(&out, 32)
		writeU32(&out, uint32(len(e.data)))
		writeU32(&out, uint32(offset))
		offset += len(e.data)
	}
	for _, e := range entries {
		out.Write(e.data)
	}
	return out.Bytes(), nil
}

func icoDimByte(v int) byte {
	if v >= 256 {
		return 0
	}
	return byte(v)
}

// encodeICNS writes the 'icns' body with one typed PNG entry per size that maps
// to a known OSType. Only the canonical type per size is emitted, so a size
// requested once produces one entry even though two types share it.
func encodeICNS(bySize map[int]image.Image) ([]byte, error) {
	sizes := make([]int, 0, len(bySize))
	for s := range bySize {
		sizes = append(sizes, s)
	}
	sort.Ints(sizes)
	for _, s := range sizes {
		if _, ok := bySize[s]; !ok {
			return nil, fmt.Errorf("ops: no ICNS payload for size %d (allowed: %v)", s, icnsSizes())
		}
	}
	var body bytes.Buffer
	// Emit every OSType whose size was requested, in table order. Sizes 256 and
	// 512 have two types each (ic08/ic13 and ic09/ic14); electron-builder
	// expects both the standard and @2x entries.
	for _, t := range icnsTypes {
		img, ok := bySize[t.Size]
		if !ok {
			continue
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return nil, err
		}
		body.WriteString(t.Type)
		writeU32BE(&body, uint32(8+buf.Len()))
		body.Write(buf.Bytes())
	}
	var out bytes.Buffer
	out.WriteString("icns")
	writeU32BE(&out, uint32(8+body.Len()))
	out.Write(body.Bytes())
	return out.Bytes(), nil
}

func icnsSizes() []int {
	seen := map[int]bool{}
	var out []int
	for _, t := range icnsTypes {
		if !seen[t.Size] {
			seen[t.Size] = true
			out = append(out, t.Size)
		}
	}
	sort.Ints(out)
	return out
}

func writeU16(w *bytes.Buffer, v uint16) {
	var b [2]byte
	binary.LittleEndian.PutUint16(b[:], v)
	w.Write(b[:])
}

func writeU32(w *bytes.Buffer, v uint32) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	w.Write(b[:])
}

func writeU32BE(w *bytes.Buffer, v uint32) {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], v)
	w.Write(b[:])
}
