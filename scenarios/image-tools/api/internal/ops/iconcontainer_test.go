package ops

import (
	"encoding/binary"
	"testing"
)

func TestIconContainerICO(t *testing.T) {
	sizes := []int{16, 32, 48, 256}
	res, err := IconContainer(RunInput{Bytes: []byte(rasterizeFixtureSVG), Params: &Params{ContainerFormat: FormatICO, Sizes: sizes}})
	if err != nil {
		t.Fatalf("icon_container ico: %v", err)
	}
	if res.Format != FormatICO {
		t.Fatalf("format = %s, want ico", res.Format)
	}
	if len(res.Bytes) < 6 || binary.LittleEndian.Uint16(res.Bytes[2:4]) != 1 {
		t.Fatalf("bad ICONDIR header")
	}
	count := int(binary.LittleEndian.Uint16(res.Bytes[4:6]))
	if count != len(sizes) {
		t.Fatalf("entry count = %d, want %d", count, len(sizes))
	}
	for i := 0; i < count; i++ {
		off := 6 + 16*i
		w := int(res.Bytes[off])
		if sizes[i] >= 256 {
			w = 256
		}
		if w != sizes[i] {
			t.Fatalf("entry %d width byte = %d, want %d", i, w, sizes[i])
		}
		imgLen := int(binary.LittleEndian.Uint32(res.Bytes[off+8 : off+12]))
		imgOff := int(binary.LittleEndian.Uint32(res.Bytes[off+12 : off+16]))
		if imgOff+imgLen > len(res.Bytes) {
			t.Fatalf("entry %d payload out of range", i)
		}
		if !isPNG(res.Bytes[imgOff : imgOff+4]) {
			t.Fatalf("entry %d is not a PNG payload", i)
		}
	}
}

func TestIconContainerICNS(t *testing.T) {
	sizes := []int{32, 64, 128, 256, 512, 1024}
	res, err := IconContainer(RunInput{Bytes: []byte(rasterizeFixtureSVG), Params: &Params{ContainerFormat: FormatICNS, Sizes: sizes}})
	if err != nil {
		t.Fatalf("icon_container icns: %v", err)
	}
	if res.Format != FormatICNS {
		t.Fatalf("format = %s, want icns", res.Format)
	}
	b := res.Bytes
	if string(b[0:4]) != "icns" {
		t.Fatalf("bad ICNS magic")
	}
	total := int(binary.BigEndian.Uint32(b[4:8]))
	if total != len(b) {
		t.Fatalf("ICNS length = %d, want %d", total, len(b))
	}
	want := map[string]int{"ic11": 32, "ic12": 64, "ic07": 128, "ic08": 256, "ic13": 256, "ic09": 512, "ic14": 512, "ic10": 1024}
	got := map[string]bool{}
	pos := 8
	for pos+8 <= len(b) {
		typ := string(b[pos : pos+4])
		length := int(binary.BigEndian.Uint32(b[pos+4 : pos+8]))
		if length < 8 || pos+length > len(b) {
			t.Fatalf("bad ICNS entry at %d", pos)
		}
		if _, ok := want[typ]; !ok {
			t.Fatalf("unexpected ICNS type %q", typ)
		}
		if !isPNG(b[pos+8 : pos+12]) {
			t.Fatalf("ICNS entry %q is not a PNG payload", typ)
		}
		got[typ] = true
		pos += length
	}
	for typ := range want {
		if !got[typ] {
			t.Errorf("missing ICNS entry %q", typ)
		}
	}
}

func TestIconContainerRejectsUnknownFormat(t *testing.T) {
	if _, err := IconContainer(RunInput{Bytes: []byte(rasterizeFixtureSVG), Params: &Params{ContainerFormat: "bmp", Sizes: []int{16}}}); err == nil {
		t.Fatal("expected an unknown-format error")
	}
}

func isPNG(b []byte) bool {
	return len(b) >= 4 && b[0] == 0x89 && b[1] == 'P' && b[2] == 'N' && b[3] == 'G'
}
