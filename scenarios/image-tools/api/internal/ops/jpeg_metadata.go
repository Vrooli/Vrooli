package ops

import (
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

var errMalformedJPEGMetadata = errors.New("ops: malformed JPEG metadata")

// stripJPEGGPSMetadata removes every JPEG APP/COM segment except a valid JFIF
// header and rebuilt, explicitly whitelisted authorship fields. Entropy-coded
// scan data is copied without decoding or re-encoding, including restart
// markers and byte-stuffed values.
func stripJPEGGPSMetadata(src []byte) ([]byte, error) {
	if len(src) < 4 || src[0] != 0xff || src[1] != 0xd8 {
		return nil, fmt.Errorf("%w: missing SOI", errMalformedJPEGMetadata)
	}
	out := append([]byte(nil), src[:2]...)
	pos := 2
	inScan := false
	sawSOS := false
	for pos < len(src) {
		if inScan {
			markerStart, err := nextJPEGMarker(src, pos)
			if err != nil {
				return nil, err
			}
			out = append(out, src[pos:markerStart]...)
			pos = markerStart
			inScan = false
		}
		start := pos
		if src[pos] != 0xff {
			return nil, fmt.Errorf("%w: expected marker at byte %d", errMalformedJPEGMetadata, pos)
		}
		for pos < len(src) && src[pos] == 0xff {
			pos++
		}
		if pos >= len(src) {
			return nil, fmt.Errorf("%w: truncated marker", errMalformedJPEGMetadata)
		}
		marker := src[pos]
		pos++
		if marker == 0x00 {
			return nil, fmt.Errorf("%w: stuffed byte before scan", errMalformedJPEGMetadata)
		}
		if marker == 0xd9 {
			if !sawSOS {
				return nil, fmt.Errorf("%w: EOI before SOS", errMalformedJPEGMetadata)
			}
			if pos != len(src) {
				return nil, fmt.Errorf("%w: unexamined bytes after EOI", errMalformedJPEGMetadata)
			}
			return append(out, src[start:pos]...), nil
		}
		if marker == 0xda {
			if pos+2 > len(src) {
				return nil, fmt.Errorf("%w: truncated SOS length", errMalformedJPEGMetadata)
			}
			length := int(binary.BigEndian.Uint16(src[pos : pos+2]))
			if length < 8 || pos+length > len(src) {
				return nil, fmt.Errorf("%w: invalid SOS length", errMalformedJPEGMetadata)
			}
			components := int(src[pos+2])
			if components == 0 || length != 6+2*components {
				return nil, fmt.Errorf("%w: invalid SOS component list", errMalformedJPEGMetadata)
			}
			out = append(out, src[start:pos+length]...)
			pos += length
			sawSOS = true
			inScan = true
			continue
		}
		if marker == 0xd8 {
			return nil, fmt.Errorf("%w: nested SOI marker", errMalformedJPEGMetadata)
		}
		if marker == 0x01 || marker >= 0xd0 && marker <= 0xd7 {
			out = append(out, src[start:pos]...)
			continue
		}
		if pos+2 > len(src) {
			return nil, fmt.Errorf("%w: truncated marker length", errMalformedJPEGMetadata)
		}
		length := int(binary.BigEndian.Uint16(src[pos : pos+2]))
		if length < 2 || pos+length > len(src) {
			return nil, fmt.Errorf("%w: invalid marker length for ff%02x", errMalformedJPEGMetadata, marker)
		}
		payload := src[pos+2 : pos+length]
		segmentEnd := pos + length
		keep := true
		var replacement []byte
		var err error
		switch {
		case marker == 0xe0:
			keep = validJFIF(payload)
		case marker == 0xe1 && bytes.HasPrefix(payload, []byte("Exif\x00\x00")):
			keep = false
			replacement, err = stripExifGPS(payload)
		case marker == 0xe1 && bytes.HasPrefix(payload, []byte("http://ns.adobe.com/xap/1.0/\x00")):
			keep = false
			replacement, err = stripXMP(payload)
		case marker == 0xed && bytes.HasPrefix(payload, []byte("Photoshop 3.0\x00")):
			keep = false
			replacement, err = stripPhotoshopIPTC(payload)
		case marker >= 0xe0 && marker <= 0xef || marker == 0xfe:
			keep = false
		}
		if err != nil {
			return nil, err
		}
		if replacement != nil {
			if len(replacement) > 65533 {
				return nil, fmt.Errorf("%w: retained metadata exceeds JPEG segment limit", errMalformedJPEGMetadata)
			}
			if len(replacement) > 0 {
				out = appendJPEGSegment(out, marker, replacement)
			}
		} else if keep {
			out = append(out, src[start:segmentEnd]...)
		}
		pos = segmentEnd
	}
	return nil, fmt.Errorf("%w: missing SOS", errMalformedJPEGMetadata)
}

// nextJPEGMarker finds the next marker that terminates entropy-coded scan data.
// A stuffed FF00 byte and restart markers are part of the scan and are skipped.
func nextJPEGMarker(src []byte, pos int) (int, error) {
	for pos < len(src) {
		if src[pos] != 0xff {
			pos++
			continue
		}
		start := pos
		for pos < len(src) && src[pos] == 0xff {
			pos++
		}
		if pos >= len(src) {
			return 0, fmt.Errorf("%w: truncated marker after scan", errMalformedJPEGMetadata)
		}
		marker := src[pos]
		pos++
		if marker == 0x00 || marker >= 0xd0 && marker <= 0xd7 {
			continue
		}
		return start, nil
	}
	return 0, fmt.Errorf("%w: scan has no terminating marker", errMalformedJPEGMetadata)
}

func appendJPEGSegment(dst []byte, marker byte, payload []byte) []byte {
	dst = append(dst, 0xff, marker, 0, 0)
	binary.BigEndian.PutUint16(dst[len(dst)-2:], uint16(len(payload)+2))
	return append(dst, payload...)
}

func validJFIF(payload []byte) bool {
	return len(payload) >= 14 && bytes.HasPrefix(payload, []byte("JFIF\x00")) &&
		payload[5] == 1 && payload[6] <= 2 && payload[7] <= 2 &&
		binary.BigEndian.Uint16(payload[8:10]) > 0 && binary.BigEndian.Uint16(payload[10:12]) > 0 &&
		len(payload) == 14+3*int(payload[12])*int(payload[13])
}

type tiffEntry struct {
	tag, typ uint16
	count    uint32
	value    [4]byte
}

func stripExifGPS(payload []byte) ([]byte, error) {
	if len(payload) < 14 {
		return nil, fmt.Errorf("%w: short EXIF/TIFF header", errMalformedJPEGMetadata)
	}
	t := payload[6:]
	order := binary.ByteOrder(binary.LittleEndian)
	switch string(t[:2]) {
	case "II":
	case "MM":
		order = binary.BigEndian
	default:
		return nil, fmt.Errorf("%w: bad TIFF byte order", errMalformedJPEGMetadata)
	}
	if order.Uint16(t[2:4]) != 42 {
		return nil, fmt.Errorf("%w: bad TIFF magic", errMalformedJPEGMetadata)
	}
	ifd := order.Uint32(t[4:8])
	entries, err := validateTIFFTree(t, order, ifd, make(map[uint32]bool), 0)
	if err != nil {
		return nil, fmt.Errorf("%w: EXIF: %v", errMalformedJPEGMetadata, err)
	}
	var retained []struct {
		tag   uint16
		value []byte
	}
	for _, e := range entries {
		if (e.tag != 0x013b && e.tag != 0x8298) || e.typ != 2 || e.count == 0 {
			continue
		}
		v, err := tiffValue(t, order, e)
		if err != nil {
			return nil, fmt.Errorf("%w: EXIF whitelist value: %v", errMalformedJPEGMetadata, err)
		}
		retained = append(retained, struct {
			tag   uint16
			value []byte
		}{e.tag, bytes.TrimRight(v, "\x00")})
	}
	if len(retained) == 0 {
		return nil, nil
	}
	// Build a fresh minimal TIFF. No original unknown/private entry or pointer
	// can survive this allowlist reconstruction.
	data := make([]byte, 8+2+12*len(retained)+4)
	copy(data, []byte{'I', 'I', 42, 0, 8, 0, 0, 0})
	order.PutUint16(data[8:], uint16(len(retained)))
	for i, item := range retained {
		off := 10 + i*12
		order.PutUint16(data[off:], item.tag)
		order.PutUint16(data[off+2:], 2)
		order.PutUint32(data[off+4:], uint32(len(item.value)+1))
		v := append(append([]byte(nil), item.value...), 0)
		if len(v) <= 4 {
			copy(data[off+8:off+12], v)
		} else {
			order.PutUint32(data[off+8:], uint32(len(data)))
			data = append(data, v...)
		}
	}
	return append([]byte("Exif\x00\x00"), data...), nil
}

func parseTIFFIFD(t []byte, order binary.ByteOrder, offset uint32) ([]tiffEntry, error) {
	if uint64(offset)+2 > uint64(len(t)) {
		return nil, fmt.Errorf("IFD offset outside TIFF")
	}
	count := int(order.Uint16(t[offset:]))
	end := uint64(offset) + 2 + uint64(count)*12 + 4
	if end > uint64(len(t)) {
		return nil, fmt.Errorf("truncated IFD entries")
	}
	entries := make([]tiffEntry, 0, count)
	for i := 0; i < count; i++ {
		b := t[int(offset)+2+i*12:]
		e := tiffEntry{tag: order.Uint16(b), typ: order.Uint16(b[2:]), count: order.Uint32(b[4:])}
		copy(e.value[:], b[8:12])
		if _, err := tiffValue(t, order, e); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func validateTIFFTree(t []byte, order binary.ByteOrder, offset uint32, seen map[uint32]bool, depth int) ([]tiffEntry, error) {
	if depth > 32 {
		return nil, fmt.Errorf("TIFF IFD nesting exceeds limit")
	}
	if seen[offset] {
		return nil, fmt.Errorf("cyclic TIFF IFD pointer")
	}
	seen[offset] = true
	entries, err := parseTIFFIFD(t, order, offset)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.tag != 0x8769 && e.tag != 0x8825 && e.tag != 0xa005 && e.tag != 0x014a {
			continue
		}
		if e.typ != 4 && e.typ != 13 {
			return nil, fmt.Errorf("TIFF IFD pointer has type %d", e.typ)
		}
		v, err := tiffValue(t, order, e)
		if err != nil {
			return nil, err
		}
		for i := uint32(0); i < e.count; i++ {
			child := order.Uint32(v[i*4 : i*4+4])
			if child == 0 {
				continue
			}
			if _, err := validateTIFFTree(t, order, child, seen, depth+1); err != nil {
				return nil, err
			}
		}
	}
	nextOffset := uint64(offset) + 2 + uint64(len(entries))*12
	if nextOffset+4 > uint64(len(t)) {
		return nil, fmt.Errorf("truncated next-IFD pointer")
	}
	if next := order.Uint32(t[nextOffset : nextOffset+4]); next != 0 {
		if _, err := validateTIFFTree(t, order, next, seen, depth+1); err != nil {
			return nil, err
		}
	}
	return entries, nil
}

func tiffValue(t []byte, order binary.ByteOrder, e tiffEntry) ([]byte, error) {
	sizes := map[uint16]uint64{1: 1, 2: 1, 3: 2, 4: 4, 5: 8, 6: 1, 7: 1, 8: 2, 9: 4, 10: 8, 11: 4, 12: 8, 13: 4}
	size, ok := sizes[e.typ]
	if !ok {
		return nil, fmt.Errorf("unknown TIFF type %d", e.typ)
	}
	n := size * uint64(e.count)
	if n > uint64(len(t)) {
		return nil, fmt.Errorf("TIFF value too large")
	}
	if n <= 4 {
		return e.value[:n], nil
	}
	off := uint64(order.Uint32(e.value[:]))
	if off+n > uint64(len(t)) {
		return nil, fmt.Errorf("TIFF value outside container")
	}
	return t[off : off+n], nil
}

const (
	dcNS        = "http://purl.org/dc/elements/1.1/"
	xmpRightsNS = "http://ns.adobe.com/xap/1.0/rights/"
	photoshopNS = "http://ns.adobe.com/photoshop/1.0/"
	rdfNS       = "http://www.w3.org/1999/02/22-rdf-syntax-ns#"
	xmlNS       = "http://www.w3.org/XML/1998/namespace"
)

type xmpValue struct {
	text string
	lang string
}

func stripXMP(payload []byte) ([]byte, error) {
	const sig = "http://ns.adobe.com/xap/1.0/\x00"
	raw := payload[len(sig):]
	type prop struct {
		name, ns string
		values   []xmpValue
	}
	props := []prop{{"creator", dcNS, nil}, {"rights", dcNS, nil}, {"UsageTerms", xmpRightsNS, nil}, {"Credit", photoshopNS, nil}}
	dec := xml.NewDecoder(bytes.NewReader(raw))
	active := -1
	depth := 0
	propertyDepth := -1
	listDepth := -1
	containerSeen := false
	propertyLang := ""
	var simpleText strings.Builder
	var listText strings.Builder
	listLang := ""
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: malformed XMP XML: %v", errMalformedJPEGMetadata, err)
		}
		switch v := tok.(type) {
		case xml.StartElement:
			depth++
			if active < 0 {
				for i, p := range props {
					if v.Name.Space == p.ns && v.Name.Local == p.name {
						active = i
						propertyDepth = depth
						containerSeen = false
						propertyLang = xmlLang(v.Attr)
						simpleText.Reset()
						break
					}
				}
			} else if v.Name.Space == rdfNS && (v.Name.Local == "Alt" || v.Name.Local == "Seq") {
				containerSeen = true
			} else if v.Name.Space == rdfNS && v.Name.Local == "li" && containerSeen {
				listDepth = depth
				listLang = xmlLang(v.Attr)
				listText.Reset()
			}
		case xml.CharData:
			if active >= 0 {
				if listDepth >= 0 {
					listText.Write([]byte(v))
				} else if !containerSeen {
					simpleText.Write([]byte(v))
				}
			}
		case xml.EndElement:
			if active >= 0 && listDepth == depth && v.Name.Space == rdfNS && v.Name.Local == "li" {
				value := strings.TrimSpace(listText.String())
				if value != "" {
					props[active].values = append(props[active].values, xmpValue{text: value, lang: listLang})
				}
				listDepth = -1
				listLang = ""
				listText.Reset()
			}
			if active >= 0 && propertyDepth == depth && v.Name.Space == props[active].ns && v.Name.Local == props[active].name {
				if !containerSeen {
					value := strings.TrimSpace(simpleText.String())
					if value != "" {
						props[active].values = append(props[active].values, xmpValue{text: value, lang: propertyLang})
					}
				}
				active = -1
				propertyDepth = -1
				propertyLang = ""
				containerSeen = false
				simpleText.Reset()
			}
			depth--
		}
	}
	var b bytes.Buffer
	b.WriteString("http://ns.adobe.com/xap/1.0/\x00<x:xmpmeta xmlns:x=\"adobe:ns:meta/\" xmlns:rdf=\"http://www.w3.org/1999/02/22-rdf-syntax-ns#\" xmlns:dc=\"" + dcNS + "\" xmlns:xmpRights=\"" + xmpRightsNS + "\" xmlns:photoshop=\"" + photoshopNS + "\"><rdf:RDF><rdf:Description>")
	for _, p := range props {
		if len(p.values) == 0 {
			continue
		}
		prefix := map[string]string{dcNS: "dc", xmpRightsNS: "xmpRights", photoshopNS: "photoshop"}[p.ns]
		if p.name == "creator" || p.name == "rights" || p.name == "UsageTerms" {
			container := "Seq"
			values := p.values
			if p.name != "creator" {
				container = "Alt"
				values = normalizedAltValues(values)
			}
			b.WriteString("<" + prefix + ":" + p.name + "><rdf:" + container + ">")
			for _, value := range values {
				b.WriteString("<rdf:li")
				if value.lang != "" {
					b.WriteString(" xml:lang=\"")
					_ = xml.EscapeText(&b, []byte(value.lang))
					b.WriteString("\"")
				}
				b.WriteString(">")
				_ = xml.EscapeText(&b, []byte(value.text))
				b.WriteString("</rdf:li>")
			}
			b.WriteString("</rdf:" + container + "></" + prefix + ":" + p.name + ">")
			continue
		}
		for _, value := range p.values {
			b.WriteString("<" + prefix + ":" + p.name + ">")
			_ = xml.EscapeText(&b, []byte(value.text))
			b.WriteString("</" + prefix + ":" + p.name + ">")
		}
	}
	b.WriteString("</rdf:Description></rdf:RDF></x:xmpmeta>")
	if len(props[0].values)+len(props[1].values)+len(props[2].values)+len(props[3].values) == 0 {
		return nil, nil
	}
	return b.Bytes(), nil
}

func xmlLang(attrs []xml.Attr) string {
	for _, attr := range attrs {
		if attr.Name.Space == xmlNS && attr.Name.Local == "lang" {
			return strings.TrimSpace(attr.Value)
		}
	}
	return ""
}

// normalizedAltValues preserves the first value for each language and emits
// x-default first as required by XMP. Untagged values use x-default only when
// no explicit default is present; duplicate language alternatives are dropped.
// If several untagged values have to be reduced to an Alt, the first is kept
// because no language distinction can be recovered from the input.
func normalizedAltValues(values []xmpValue) []xmpValue {
	var out []xmpValue
	var untagged []xmpValue
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		lang := strings.TrimSpace(value.lang)
		if lang == "" {
			untagged = append(untagged, value)
			continue
		}
		key := strings.ToLower(lang)
		if seen[key] {
			continue
		}
		seen[key] = true
		if key == "x-default" {
			lang = "x-default"
		}
		out = append(out, xmpValue{text: value.text, lang: lang})
	}
	if !seen["x-default"] && len(untagged) > 0 {
		out = append(out, xmpValue{text: untagged[0].text, lang: "x-default"})
	}
	for i, value := range out {
		if strings.EqualFold(value.lang, "x-default") {
			copy(out[1:i+1], out[0:i])
			out[0] = value
			break
		}
	}
	return out
}

func stripPhotoshopIPTC(payload []byte) ([]byte, error) {
	const header = "Photoshop 3.0\x00"
	data := payload[len(header):]
	var retained []byte
	for pos := 0; pos < len(data); {
		if pos+7 > len(data) || string(data[pos:pos+4]) != "8BIM" {
			return nil, fmt.Errorf("%w: malformed Photoshop resource header", errMalformedJPEGMetadata)
		}
		id := binary.BigEndian.Uint16(data[pos+4:])
		pos += 6
		nameLen := int(data[pos])
		pos++
		if pos+nameLen > len(data) {
			return nil, fmt.Errorf("%w: truncated Photoshop resource name", errMalformedJPEGMetadata)
		}
		pos += nameLen
		if (1+nameLen)%2 != 0 {
			if pos >= len(data) {
				return nil, fmt.Errorf("%w: missing Photoshop name padding", errMalformedJPEGMetadata)
			}
			pos++
		}
		if pos+4 > len(data) {
			return nil, fmt.Errorf("%w: missing Photoshop resource size", errMalformedJPEGMetadata)
		}
		n := uint64(binary.BigEndian.Uint32(data[pos:]))
		pos += 4
		if n > uint64(len(data)-pos) {
			return nil, fmt.Errorf("%w: truncated Photoshop resource", errMalformedJPEGMetadata)
		}
		body := data[pos : pos+int(n)]
		pos += int(n)
		if n%2 != 0 {
			if pos >= len(data) {
				return nil, fmt.Errorf("%w: missing Photoshop data padding", errMalformedJPEGMetadata)
			}
			pos++
		}
		if id != 0x0404 {
			continue
		}
		kept, err := filterIPTCIIM(body)
		if err != nil {
			return nil, err
		}
		if len(kept) > 0 {
			retained = append(retained, []byte("8BIM")...)
			retained = append(retained, 0x04, 0x04, 0, 0)
			retained = append(retained, 0, 0, 0, 0)
			binary.BigEndian.PutUint32(retained[len(retained)-4:], uint32(len(kept)))
			retained = append(retained, kept...)
			if len(kept)%2 != 0 {
				retained = append(retained, 0)
			}
		}
	}
	if len(retained) == 0 {
		return nil, nil
	}
	return append([]byte(header), retained...), nil
}

func filterIPTCIIM(data []byte) ([]byte, error) {
	var out []byte
	for pos := 0; pos < len(data); {
		start := pos
		if pos+5 > len(data) || data[pos] != 0x1c {
			return nil, fmt.Errorf("%w: malformed IPTC IIM dataset", errMalformedJPEGMetadata)
		}
		record, dataset := data[pos+1], data[pos+2]
		pos += 3
		length := int(binary.BigEndian.Uint16(data[pos:]))
		pos += 2
		if length&0x8000 != 0 {
			n := length & 0x7fff
			if n == 0 || n > 4 || pos+n > len(data) {
				return nil, fmt.Errorf("%w: invalid IPTC extended length", errMalformedJPEGMetadata)
			}
			length = 0
			for _, b := range data[pos : pos+n] {
				length = length<<8 | int(b)
			}
			pos += n
		}
		if length > len(data)-pos {
			return nil, fmt.Errorf("%w: truncated IPTC value", errMalformedJPEGMetadata)
		}
		pos += length
		if record == 2 && (dataset == 80 || dataset == 90 || dataset == 116) {
			out = append(out, data[start:pos]...)
		}
	}
	return out, nil
}
