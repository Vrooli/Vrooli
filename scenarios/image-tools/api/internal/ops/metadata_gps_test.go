package ops

import (
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"strings"
	"testing"
)

func gpsTestSegment(marker byte, payload []byte) []byte {
	out := []byte{0xff, marker, 0, 0}
	binary.BigEndian.PutUint16(out[2:], uint16(len(payload)+2))
	return append(out, payload...)
}

func gpsTestSOS(ss, se byte) []byte {
	// One-component scan: selector, table selector, spectral start/end, Ah/Al.
	return gpsTestSegment(0xda, []byte{1, 1, 0, ss, se, 0})
}

func gpsTestProgressiveMarkerStream() (source, expected []byte) {
	// This minimal marker stream exercises the scanner's progressive/multi-scan
	// framing. It is not a JPEG decoder fixture; frame and scan semantics are
	// represented by SOF2 plus distinct DC and AC scan headers.
	sof2 := gpsTestSegment(0xc2, []byte{8, 0, 8, 0, 12, 1, 1, 0x11, 0})
	dcScan := gpsTestSOS(0, 0)
	acScan := gpsTestSOS(1, 63)
	scanOne := []byte{0x11, 0xff, 0x00, 0x22, 0xff, 0xd0, 0x33}
	scanTwo := []byte{0x44, 0xff, 0x00, 0x55}
	location := gpsTestSegment(0xe1, gpsTestGPSOnlyEXIF())
	comment := gpsTestSegment(0xfe, []byte("between progressive scans private location"))
	trailerLocation := gpsTestSegment(0xe2, []byte("after scan private camera data"))
	end := []byte{0xff, 0xd9}

	source = append([]byte{0xff, 0xd8}, sof2...)
	source = append(source, dcScan...)
	source = append(source, scanOne...)
	source = append(source, location...)
	source = append(source, comment...)
	source = append(source, acScan...)
	source = append(source, scanTwo...)
	source = append(source, trailerLocation...)
	source = append(source, end...)

	expected = append([]byte{0xff, 0xd8}, sof2...)
	expected = append(expected, dcScan...)
	expected = append(expected, scanOne...)
	expected = append(expected, acScan...)
	expected = append(expected, scanTwo...)
	expected = append(expected, end...)
	return source, expected
}

func gpsTestLocalizedXMP() []byte {
	body := `<x:xmpmeta xmlns:x="adobe:ns:meta/" xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:xmpRights="http://ns.adobe.com/xap/1.0/rights/" xmlns:exif="http://ns.adobe.com/exif/1.0/"><rdf:RDF><rdf:Description><dc:rights><rdf:Alt><rdf:li xml:lang="fr-FR">Droits</rdf:li><rdf:li xml:lang="x-default">Rights default</rdf:li><rdf:li xml:lang="en-US">Rights</rdf:li><rdf:li xml:lang="x-default">Duplicate default</rdf:li></rdf:Alt></dc:rights><xmpRights:UsageTerms><rdf:Alt><rdf:li xml:lang="de-DE">Nutzungsbedingungen</rdf:li><rdf:li xml:lang="x-default">Terms default</rdf:li><rdf:li xml:lang="ja-JP">利用条件</rdf:li></rdf:Alt></xmpRights:UsageTerms><exif:GPSLatitude>51,30N</exif:GPSLatitude></rdf:Description></rdf:RDF></x:xmpmeta>`
	return append([]byte("http://ns.adobe.com/xap/1.0/\x00"), []byte(body)...)
}

type gpsTestAltValue struct {
	lang string
	text string
}

func gpsTestXMPAltValues(t *testing.T, jpegBytes []byte, namespace, property string) []gpsTestAltValue {
	t.Helper()
	const sig = "http://ns.adobe.com/xap/1.0/\x00"
	start := bytes.Index(jpegBytes, []byte(sig))
	if start < 0 {
		t.Fatalf("retained XMP packet missing for {%s}%s", namespace, property)
	}
	bodyStart := start + len(sig)
	endTag := bytes.Index(jpegBytes[bodyStart:], []byte("</x:xmpmeta>"))
	if endTag < 0 {
		t.Fatal("retained XMP packet has no closing xmpmeta element")
	}
	dec := xml.NewDecoder(bytes.NewReader(jpegBytes[bodyStart : bodyStart+endTag+len("</x:xmpmeta>")]))
	const rdfNS = "http://www.w3.org/1999/02/22-rdf-syntax-ns#"
	const xmlNS = "http://www.w3.org/XML/1998/namespace"
	active, inList := false, false
	var current gpsTestAltValue
	var values []gpsTestAltValue
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return values
		}
		if err != nil {
			t.Fatalf("decode retained XMP: %v", err)
		}
		switch token := tok.(type) {
		case xml.StartElement:
			if !active && token.Name.Space == namespace && token.Name.Local == property {
				active = true
				continue
			}
			if active && token.Name.Space == rdfNS && token.Name.Local == "li" {
				inList = true
				current = gpsTestAltValue{}
				for _, attr := range token.Attr {
					if attr.Name.Space == xmlNS && attr.Name.Local == "lang" {
						current.lang = attr.Value
					}
				}
			}
		case xml.CharData:
			if inList {
				current.text += string(token)
			}
		case xml.EndElement:
			if inList && token.Name.Space == rdfNS && token.Name.Local == "li" {
				current.text = strings.TrimSpace(current.text)
				values = append(values, current)
				inList = false
				continue
			}
			if active && token.Name.Space == namespace && token.Name.Local == property {
				active = false
			}
		}
	}
}

func gpsTestJPEG(t *testing.T, markers ...[]byte) []byte {
	t.Helper()
	var encoded bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 12, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 12; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 17), G: uint8(y * 29), B: 90, A: 255})
		}
	}
	if err := jpeg.Encode(&encoded, img, &jpeg.Options{Quality: 83}); err != nil {
		t.Fatal(err)
	}
	base := encoded.Bytes()
	sos := bytes.Index(base, []byte{0xff, 0xda})
	if sos < 0 {
		t.Fatal("image/jpeg output has no SOS")
	}
	out := append([]byte(nil), base[:2]...)
	for _, marker := range markers {
		out = append(out, marker...)
	}
	out = append(out, base[2:sos]...)
	return append(out, base[sos:]...)
}

func gpsTestJFIF() []byte {
	return gpsTestSegment(0xe0, []byte{'J', 'F', 'I', 'F', 0, 1, 2, 0, 0, 1, 0, 1, 0, 0})
}

func TestValidJFIFAcceptsFullWidthPositiveDensity(t *testing.T) {
	segment := gpsTestJFIF()
	payload := append([]byte(nil), segment[4:]...)
	binary.BigEndian.PutUint16(payload[8:10], 300)
	binary.BigEndian.PutUint16(payload[10:12], 65535)
	if !validJFIF(payload) {
		t.Fatal("valid 16-bit JFIF densities were rejected")
	}

	binary.BigEndian.PutUint16(payload[8:10], 0)
	if validJFIF(payload) {
		t.Fatal("zero JFIF X density was accepted")
	}
	binary.BigEndian.PutUint16(payload[8:10], 300)
	binary.BigEndian.PutUint16(payload[10:12], 0)
	if validJFIF(payload) {
		t.Fatal("zero JFIF Y density was accepted")
	}
}

func gpsTestEXIF() []byte {
	// Little endian TIFF: Artist, Copyright and GPS pointer in IFD0; real
	// GPSLatitudeRef and GPSLatitude rational entries in the GPS child IFD.
	t := make([]byte, 8+2+4*12+4)
	copy(t, []byte{'I', 'I', 42, 0, 8, 0, 0, 0})
	binary.LittleEndian.PutUint16(t[8:], 4)
	entry := func(off int, tag, typ uint16, count, value uint32) {
		binary.LittleEndian.PutUint16(t[off:], tag)
		binary.LittleEndian.PutUint16(t[off+2:], typ)
		binary.LittleEndian.PutUint32(t[off+4:], count)
		binary.LittleEndian.PutUint32(t[off+8:], value)
	}
	entry(10, 0x013b, 2, 6, uint32(len(t)))
	t = append(t, []byte("Alice\x00")...)
	entry(22, 0x8298, 2, uint32(len("Copyright 2026\x00")), uint32(len(t)))
	t = append(t, []byte("Copyright 2026\x00")...)
	entry(34, 0xc7a1, 2, uint32(len("PrivateTiff\x00")), uint32(len(t)))
	t = append(t, []byte("PrivateTiff\x00")...)
	gpsOffset := uint32(len(t))
	entry(46, 0x8825, 4, 1, gpsOffset)
	// The next-IFD pointer immediately follows the three IFD0 entries.
	binary.LittleEndian.PutUint32(t[58:], 0)
	// GPS IFD includes latitude reference and three rational coordinates.
	t = append(t, make([]byte, 2+2*12+4)...)
	binary.LittleEndian.PutUint16(t[gpsOffset:], 2)
	entry(int(gpsOffset)+2, 1, 2, 2, 0x004e)
	rationalOffset := uint32(len(t))
	entry(int(gpsOffset)+14, 2, 5, 3, rationalOffset)
	binary.LittleEndian.PutUint32(t[int(gpsOffset)+26:], 0)
	for _, n := range []uint32{51, 1, 1, 30, 1, 1, 0, 1, 1} {
		var b [4]byte
		binary.LittleEndian.PutUint32(b[:], n)
		t = append(t, b[:]...)
	}
	return append([]byte("Exif\x00\x00"), t...)
}

func gpsTestGPSOnlyEXIF() []byte {
	t := make([]byte, 8+2+12+4)
	copy(t, []byte{'I', 'I', 42, 0, 8, 0, 0, 0})
	binary.LittleEndian.PutUint16(t[8:], 1)
	binary.LittleEndian.PutUint16(t[10:], 0x8825)
	binary.LittleEndian.PutUint16(t[12:], 4)
	binary.LittleEndian.PutUint32(t[14:], 1)
	gpsOffset := uint32(len(t))
	binary.LittleEndian.PutUint32(t[18:], gpsOffset)
	t = append(t, make([]byte, 2+2*12+4)...)
	binary.LittleEndian.PutUint16(t[gpsOffset:], 2)
	binary.LittleEndian.PutUint16(t[int(gpsOffset)+2:], 1)
	binary.LittleEndian.PutUint16(t[int(gpsOffset)+4:], 2)
	binary.LittleEndian.PutUint32(t[int(gpsOffset)+6:], 2)
	binary.LittleEndian.PutUint16(t[int(gpsOffset)+14:], 2)
	binary.LittleEndian.PutUint16(t[int(gpsOffset)+16:], 5)
	binary.LittleEndian.PutUint32(t[int(gpsOffset)+18:], 3)
	rationalOffset := uint32(len(t))
	binary.LittleEndian.PutUint32(t[int(gpsOffset)+22:], rationalOffset)
	for _, value := range []uint32{51, 1, 30, 1, 0, 1} {
		var b [4]byte
		binary.LittleEndian.PutUint32(b[:], value)
		t = append(t, b[:]...)
	}
	return append([]byte("Exif\x00\x00"), t...)
}

func gpsTestXMP(withCredit bool) []byte {
	credit := ""
	if withCredit {
		credit = "<photoshop:Credit>Press desk</photoshop:Credit>"
	}
	body := `<x:xmpmeta xmlns:x="adobe:ns:meta/" xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:exif="http://ns.adobe.com/exif/1.0/" xmlns:Iptc4xmpCore="http://iptc.org/std/Iptc4xmpCore/1.0/xmlns/" xmlns:photoshop="http://ns.adobe.com/photoshop/1.0/"><rdf:RDF><rdf:Description><dc:creator><rdf:Seq><rdf:li>Alice</rdf:li><rdf:li>Bob</rdf:li></rdf:Seq></dc:creator><dc:rights><rdf:Alt><rdf:li>Rights</rdf:li></rdf:Alt></dc:rights><xmpRights:UsageTerms xmlns:xmpRights="http://ns.adobe.com/xap/1.0/rights/"><rdf:Alt><rdf:li>Editorial use</rdf:li></rdf:Alt></xmpRights:UsageTerms>` + credit + `<exif:GPSLatitude>51,30N</exif:GPSLatitude><Iptc4xmpCore:Location>Secret Valley</Iptc4xmpCore:Location><photoshop:City>Hidden City</photoshop:City><private:Location xmlns:private="urn:private">Private Place</private:Location></rdf:Description></rdf:RDF></x:xmpmeta>`
	return append([]byte("http://ns.adobe.com/xap/1.0/\x00"), []byte(body)...)
}

func gpsTestIIM(dataset uint16, value string) []byte {
	// IPTC IIM encodes the dataset in one byte; the following two bytes are length.
	out := []byte{0x1c, 2, byte(dataset), 0, 0}
	binary.BigEndian.PutUint16(out[3:], uint16(len(value)))
	return append(out, []byte(value)...)
}

func gpsTestPhotoshop() []byte {
	iim := append(gpsTestIIM(80, "Byline Person"), gpsTestIIM(90, "Credit desk")...)
	iim = append(iim, gpsTestIIM(116, "Copyright notice")...)
	iim = append(iim, gpsTestIIM(92, "Secret Sublocation")...)
	resource := append([]byte("8BIM"), 0x04, 0x04, 0, 0, 0, 0, 0, 0)
	binary.BigEndian.PutUint32(resource[8:], uint32(len(iim)))
	resource = append(resource, iim...)
	if len(iim)%2 != 0 {
		resource = append(resource, 0)
	}
	return append([]byte("Photoshop 3.0\x00"), resource...)
}

func gpsTestPhotoshopLocationOnly() []byte {
	iim := gpsTestIIM(92, "Secret Sublocation")
	resource := append([]byte("8BIM"), 0x04, 0x04, 0, 0, 0, 0, 0, 0)
	binary.BigEndian.PutUint32(resource[8:], uint32(len(iim)))
	resource = append(resource, iim...)
	if len(iim)%2 != 0 {
		resource = append(resource, 0)
	}
	return append([]byte("Photoshop 3.0\x00"), resource...)
}

func gpsTestLocationOnlyXMP() []byte {
	body := `<x:xmpmeta xmlns:x="adobe:ns:meta/" xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:exif="http://ns.adobe.com/exif/1.0/" xmlns:iptc="http://iptc.org/std/Iptc4xmpCore/1.0/xmlns/"><rdf:RDF><rdf:Description><exif:GPSLatitude>51,30N</exif:GPSLatitude><iptc:Location>Secret Valley</iptc:Location></rdf:Description></rdf:RDF></x:xmpmeta>`
	return append([]byte("http://ns.adobe.com/xap/1.0/\x00"), []byte(body)...)
}

func assertGPSXMPRDFContainers(t *testing.T, jpegBytes []byte) {
	t.Helper()
	const sig = "http://ns.adobe.com/xap/1.0/\x00"
	start := bytes.Index(jpegBytes, []byte(sig))
	if start < 0 {
		t.Fatal("retained XMP packet missing")
	}
	bodyStart := start + len(sig)
	endTag := bytes.Index(jpegBytes[bodyStart:], []byte("</x:xmpmeta>"))
	if endTag < 0 {
		t.Fatal("retained XMP packet has no closing xmpmeta element")
	}
	xmlBytes := jpegBytes[bodyStart : bodyStart+endTag+len("</x:xmpmeta>")]
	dec := xml.NewDecoder(bytes.NewReader(xmlBytes))
	var stack []xml.Name
	seen := map[string]bool{}
	for {
		tok, err := dec.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("serialized XMP is not well-formed RDF/XML: %v", err)
		}
		switch token := tok.(type) {
		case xml.StartElement:
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				switch {
				case parent.Space == dcNS && parent.Local == "creator":
					if token.Name.Space != "http://www.w3.org/1999/02/22-rdf-syntax-ns#" || token.Name.Local != "Seq" {
						t.Fatalf("dc:creator child is {%s}%s, want rdf:Seq", token.Name.Space, token.Name.Local)
					}
				case parent.Space == dcNS && parent.Local == "rights", parent.Space == xmpRightsNS && parent.Local == "UsageTerms":
					if token.Name.Space != "http://www.w3.org/1999/02/22-rdf-syntax-ns#" || token.Name.Local != "Alt" {
						t.Fatalf("rights property child is {%s}%s, want rdf:Alt", token.Name.Space, token.Name.Local)
					}
				case token.Name.Space == "http://www.w3.org/1999/02/22-rdf-syntax-ns#" && token.Name.Local == "li":
					if parent.Space != "http://www.w3.org/1999/02/22-rdf-syntax-ns#" || parent.Local != "Seq" && parent.Local != "Alt" {
						t.Fatalf("rdf:li parent is {%s}%s, want rdf:Seq or rdf:Alt", parent.Space, parent.Local)
					}
				}
			}
			if token.Name.Space == dcNS && (token.Name.Local == "creator" || token.Name.Local == "rights") || token.Name.Space == xmpRightsNS && token.Name.Local == "UsageTerms" {
				seen[token.Name.Space+"#"+token.Name.Local] = true
			}
			stack = append(stack, token.Name)
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	for _, key := range []string{dcNS + "#creator", dcNS + "#rights", xmpRightsNS + "#UsageTerms"} {
		if !seen[key] {
			t.Errorf("serialized XMP missing whitelisted property %q", key)
		}
	}
}

func gpsTestSOSOffset(data []byte) int { return bytes.Index(data, []byte{0xff, 0xda}) }

func gpsTestStrip(t *testing.T, src []byte) []byte {
	t.Helper()
	res, err := Execute("metadata", src, &Params{StripGPS: true})
	if err != nil {
		t.Fatalf("StripGPS: %v", err)
	}
	if res.Format != FormatJPEG {
		t.Fatalf("format %q, want jpeg", res.Format)
	}
	return res.Bytes
}

func TestStripGPSJPEGRetainsOnlyWhitelistedMetadataAndExactScan(t *testing.T) {
	jfif := gpsTestJFIF()
	unknownAPP := gpsTestSegment(0xe2, []byte("private camera serial 123"))
	comment := gpsTestSegment(0xfe, []byte("private location comment"))
	xmp := gpsTestSegment(0xe1, gpsTestXMP(true))
	exif := gpsTestSegment(0xe1, gpsTestEXIF())
	iptc := gpsTestSegment(0xed, gpsTestPhotoshop())
	src := gpsTestJPEG(t, jfif, exif, xmp, iptc, unknownAPP, comment)
	out := gpsTestStrip(t, src)
	for _, want := range [][]byte{jfif, []byte("Alice"), []byte("Bob"), []byte("Rights"), []byte("Editorial use"), []byte("Press desk"), []byte("Byline Person"), []byte("Credit desk"), []byte("Copyright notice")} {
		if !bytes.Contains(out, want) {
			t.Errorf("retained metadata missing %q", want)
		}
	}
	for _, leak := range [][]byte{[]byte("51,30N"), []byte("Secret Valley"), []byte("Hidden City"), []byte("Secret Sublocation"), []byte("Private Place"), []byte("PrivateTiff"), []byte("private camera serial 123"), []byte("private location comment")} {
		if bytes.Contains(out, leak) {
			t.Errorf("metadata leak %q", leak)
		}
	}
	for _, segment := range [][]byte{unknownAPP, comment} {
		if bytes.Contains(out, segment) {
			t.Errorf("unknown APP/COM segment survived: %q", segment[:2])
		}
	}
	srcSOS, outSOS := gpsTestSOSOffset(src), gpsTestSOSOffset(out)
	if srcSOS < 0 || outSOS < 0 || !bytes.Equal(src[srcSOS:], out[outSOS:]) {
		t.Fatal("SOS and entropy-coded scan bytes changed")
	}
	const xmpSig = "http://ns.adobe.com/xap/1.0/\x00"
	xmpStart := bytes.Index(out, []byte(xmpSig))
	if xmpStart < 0 {
		t.Fatal("retained XMP packet missing while checking creator order")
	}
	xmpEnd := bytes.Index(out[xmpStart+len(xmpSig):], []byte("</x:xmpmeta>"))
	if xmpEnd < 0 {
		t.Fatal("retained XMP packet has no closing element while checking creator order")
	}
	xmpPacket := out[xmpStart : xmpStart+len(xmpSig)+xmpEnd]
	if bytes.Count(xmpPacket, []byte("Alice")) != 1 || bytes.Index(xmpPacket, []byte("Alice")) > bytes.Index(xmpPacket, []byte("Bob")) {
		t.Fatal("XMP multi-value order was not deterministic/source ordered")
	}
	assertGPSXMPRDFContainers(t, out)
}

func TestStripGPSWalksAllProgressiveScansAndPreservesScanBytes(t *testing.T) {
	source, expected := gpsTestProgressiveMarkerStream()
	got, err := stripJPEGGPSMetadata(source)
	if err != nil {
		t.Fatalf("strip progressive marker stream: %v", err)
	}
	if !bytes.Equal(got, expected) {
		t.Fatal("marker walk must remove APP/COM between and after scans while preserving every scan byte")
	}
	for _, private := range [][]byte{[]byte("Exif\x00\x00"), []byte("between progressive scans private location"), []byte("after scan private camera data")} {
		if bytes.Contains(got, private) {
			t.Errorf("private marker payload survived progressive scan walk: %q", private)
		}
	}
}

func TestStripGPSPreservesLocalizedXMPAltLanguagesAndDefaultOrder(t *testing.T) {
	source := gpsTestJPEG(t, gpsTestSegment(0xe1, gpsTestLocalizedXMP()))
	got := gpsTestStrip(t, source)
	if bytes.Contains(got, []byte("51,30N")) {
		t.Fatal("location-bearing XMP value survived selective retention")
	}
	assertValues := func(propertyNS, property string, want []gpsTestAltValue) {
		t.Helper()
		values := gpsTestXMPAltValues(t, got, propertyNS, property)
		if len(values) != len(want) {
			t.Fatalf("{%s}%s has %d language alternatives, want %d: %+v", propertyNS, property, len(values), len(want), values)
		}
		seen := make(map[string]bool, len(values))
		for i, value := range values {
			if value.lang != want[i].lang || value.text != want[i].text {
				t.Errorf("{%s}%s alternative %d = (%q, %q), want (%q, %q)", propertyNS, property, i, value.lang, value.text, want[i].lang, want[i].text)
			}
			key := strings.ToLower(value.lang)
			if seen[key] {
				t.Errorf("{%s}%s emitted duplicate language %q", propertyNS, property, value.lang)
			}
			seen[key] = true
		}
		if len(values) > 0 && values[0].lang != "x-default" {
			t.Errorf("{%s}%s x-default must be emitted first, got %q", propertyNS, property, values[0].lang)
		}
	}
	assertValues(dcNS, "rights", []gpsTestAltValue{
		{lang: "x-default", text: "Rights default"},
		{lang: "fr-FR", text: "Droits"},
		{lang: "en-US", text: "Rights"},
	})
	assertValues(xmpRightsNS, "UsageTerms", []gpsTestAltValue{
		{lang: "x-default", text: "Terms default"},
		{lang: "de-DE", text: "Nutzungsbedingungen"},
		{lang: "ja-JP", text: "利用条件"},
	})
}

func TestStripGPSRejectsUnexaminedBytesAfterEOI(t *testing.T) {
	source := gpsTestJPEG(t)
	source = append(source, gpsTestSegment(0xe1, gpsTestEXIF())...)
	got, err := stripJPEGGPSMetadata(source)
	if err == nil {
		t.Fatal("metadata after EOI must not be returned as a successful strip")
	}
	if len(got) != 0 {
		t.Fatal("failed strip returned output bytes")
	}
}

func TestStripGPSRemovesRecognizedContainersWithNoRetainedFields(t *testing.T) {
	src := gpsTestJPEG(t,
		gpsTestSegment(0xe1, gpsTestGPSOnlyEXIF()),
		gpsTestSegment(0xe1, gpsTestLocationOnlyXMP()),
		gpsTestSegment(0xed, gpsTestPhotoshopLocationOnly()),
	)
	out := gpsTestStrip(t, src)
	for _, marker := range [][]byte{[]byte("Exif\x00\x00"), []byte("http://ns.adobe.com/xap/1.0/\x00"), []byte("Photoshop 3.0\x00")} {
		if bytes.Contains(out, marker) {
			t.Errorf("recognized metadata with no allowlisted values survived: %q", marker)
		}
	}
	for _, location := range [][]byte{[]byte("51,30N"), []byte("Secret Valley"), []byte("Secret Sublocation")} {
		if bytes.Contains(out, location) {
			t.Errorf("location leaked from empty recognized container: %q", location)
		}
	}
	if !bytes.Equal(src[gpsTestSOSOffset(src):], out[gpsTestSOSOffset(out):]) {
		t.Fatal("empty recognized-container strip changed scan bytes")
	}
}

func TestStripGPSJPEGLocationOnlyXMPAndIPTC(t *testing.T) {
	xmp := gpsTestSegment(0xe1, gpsTestXMP(false))
	iptc := gpsTestSegment(0xed, gpsTestPhotoshopLocationOnly())
	src := gpsTestJPEG(t, xmp, iptc)
	out := gpsTestStrip(t, src)
	for _, leak := range []string{"51,30N", "Secret Valley", "Hidden City", "Secret Sublocation", "Private Place", "Byline Person", "Credit desk", "Copyright notice"} {
		if strings.Contains(string(out), leak) {
			t.Errorf("location-only metadata retained %q", leak)
		}
	}
	if !bytes.Equal(src[gpsTestSOSOffset(src):], out[gpsTestSOSOffset(out):]) {
		t.Fatal("location-only StripGPS changed scan bytes")
	}
}

func TestStripGPSJPEGMalformedMetadataFailsClosed(t *testing.T) {
	validExif := gpsTestEXIF()
	badExif := append([]byte(nil), validExif...)
	binary.LittleEndian.PutUint32(badExif[6+4:6+8], 0xfffffff0)
	badXMP := append([]byte("http://ns.adobe.com/xap/1.0/\x00"), []byte("<rdf:RDF><broken>")...)
	badAPP13 := append([]byte("Photoshop 3.0\x00"), []byte("not-8BIM")...)
	good := gpsTestJPEG(t)
	badLength := append([]byte(nil), good...)
	badLength = append(badLength[:2], append([]byte{0xff, 0xe2, 0, 1}, badLength[2:]...)...)
	cases := map[string][]byte{
		"segment length":   badLength,
		"EXIF TIFF offset": gpsTestJPEG(t, gpsTestSegment(0xe1, badExif)),
		"XMP RDF XML":      gpsTestJPEG(t, gpsTestSegment(0xe1, badXMP)),
		"Photoshop IPTC":   gpsTestJPEG(t, gpsTestSegment(0xed, badAPP13)),
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			res, err := Execute("metadata", src, &Params{StripGPS: true})
			if err == nil {
				t.Fatal("malformed recognized metadata succeeded")
			}
			if len(res.Bytes) != 0 {
				t.Fatal("failed strip returned output bytes")
			}
		})
	}
}

func TestStripGPSNonJPEGRemainsFullMetadataStrip(t *testing.T) {
	src := encodePNG(t, solid(9, 7, color.RGBA{R: 30, G: 60, B: 90, A: 255}))
	res, err := Execute("metadata", src, &Params{StripGPS: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Format != FormatPNG || res.Mime != "image/png" {
		t.Fatalf("non-JPEG format/mime = %q/%q", res.Format, res.Mime)
	}
	if bytes.Contains(res.Bytes, []byte("Exif")) || bytes.Contains(res.Bytes, []byte("xap/1.0")) {
		t.Fatal("non-JPEG StripGPS did not use full-strip compatibility path")
	}
}
