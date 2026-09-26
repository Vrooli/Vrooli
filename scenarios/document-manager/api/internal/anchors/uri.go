// Package anchors owns the canonical citation URI and resolution outcomes.
// The parser rejects non-canonical input rather than normalizing a citation at
// read time; equality is byte equality because consumers persist the string.
package anchors

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Kind string

const (
	KindLogical   Kind = "logical"
	KindGeometric Kind = "geometric"
	KindTabular   Kind = "tabular"
)

type Coordinates struct {
	Page       int
	Box        [4]float64
	Sheet      int
	StartCell  string
	EndCell    string
	StablePath string
	Path       string
	Start      int
	End        int
}

type URI struct {
	DocumentHash string
	Derivation   int
	Kind         Kind
	Coordinates  Coordinates
	Attributes   map[string]string
}

var (
	hashPattern = regexp.MustCompile(`^sha256-[0-9a-f]{64}$`)
	cellPattern = regexp.MustCompile(`^[A-Z]+[1-9][0-9]*$`)
	pathPattern = regexp.MustCompile(`^[0-9]+(?:/[0-9]+)*$`)
)

func (u URI) String() (string, error) {
	if !hashPattern.MatchString(u.DocumentHash) || u.Derivation < 1 {
		return "", fmt.Errorf("invalid document hash or derivation")
	}
	coord, err := formatCoordinates(u.Kind, u.Coordinates)
	if err != nil {
		return "", err
	}
	parts := []string{"vrooli-anchor:1", u.DocumentHash, strconv.Itoa(u.Derivation), string(u.Kind), coord}
	result := strings.Join(parts, "/")
	if len(u.Attributes) > 0 {
		keys := make([]string, 0, len(u.Attributes))
		for key := range u.Attributes {
			keys = append(keys, key)
		}
		for i := 0; i < len(keys); i++ {
			for j := i + 1; j < len(keys); j++ {
				if keys[j] < keys[i] {
					keys[i], keys[j] = keys[j], keys[i]
				}
			}
		}
		attrs := make([]string, 0, len(keys))
		for _, key := range keys {
			if key == "" || strings.ContainsAny(key, "&=? ") || strings.ContainsAny(u.Attributes[key], "&=? ") {
				return "", fmt.Errorf("invalid anchor attribute")
			}
			attrs = append(attrs, key+"="+u.Attributes[key])
		}
		result += "?" + strings.Join(attrs, "&")
	}
	return result, nil
}

func Parse(raw string) (URI, error) {
	if !strings.HasPrefix(raw, "vrooli-anchor:") {
		return URI{}, fmt.Errorf("invalid anchor scheme")
	}
	parts := strings.SplitN(raw, "/", 5)
	if len(parts) != 5 || parts[0] != "vrooli-anchor:1" {
		return URI{}, fmt.Errorf("invalid anchor shape")
	}
	if !hashPattern.MatchString(parts[1]) {
		return URI{}, fmt.Errorf("invalid document address")
	}
	derivation, err := parsePositive(parts[2])
	if err != nil {
		return URI{}, fmt.Errorf("invalid derivation: %w", err)
	}
	kind := Kind(parts[3])
	coordText, attrs, err := parseAttributes(parts[4])
	if err != nil {
		return URI{}, err
	}
	coords, err := parseCoordinates(kind, coordText)
	if err != nil {
		return URI{}, err
	}
	u := URI{DocumentHash: parts[1], Derivation: derivation, Kind: kind, Coordinates: coords, Attributes: attrs}
	canonical, err := u.String()
	if err != nil {
		return URI{}, err
	}
	if canonical != raw {
		return URI{}, fmt.Errorf("anchor is not canonical")
	}
	return u, nil
}

func parsePositive(raw string) (int, error) {
	if raw == "" || (len(raw) > 1 && raw[0] == '0') {
		return 0, fmt.Errorf("must be a positive integer")
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("must be a positive integer")
	}
	return n, nil
}

func parseAttributes(raw string) (string, map[string]string, error) {
	parts := strings.SplitN(raw, "?", 2)
	attrs := map[string]string{}
	if len(parts) == 1 {
		return raw, attrs, nil
	}
	if parts[1] == "" {
		return "", nil, fmt.Errorf("trailing attribute separator")
	}
	last := ""
	for _, item := range strings.Split(parts[1], "&") {
		kv := strings.SplitN(item, "=", 2)
		if len(kv) != 2 || kv[0] == "" || kv[0] <= last || strings.ContainsAny(kv[0]+kv[1], "&=? ") {
			return "", nil, fmt.Errorf("attributes must be sorted and valid")
		}
		last = kv[0]
		attrs[kv[0]] = kv[1]
	}
	return parts[0], attrs, nil
}

func formatCoordinates(kind Kind, c Coordinates) (string, error) {
	switch kind {
	case KindGeometric:
		if c.Page < 1 {
			return "", fmt.Errorf("invalid page")
		}
		for _, value := range c.Box {
			if value < 0 || value > 1 {
				return "", fmt.Errorf("geometric coordinate outside page box")
			}
		}
		return fmt.Sprintf("p%d@%.6f,%.6f,%.6f,%.6f", c.Page, c.Box[0], c.Box[1], c.Box[2], c.Box[3]), nil
	case KindTabular:
		if c.Sheet < 1 || !cellPattern.MatchString(c.StartCell) {
			return "", fmt.Errorf("invalid tabular coordinate")
		}
		if c.EndCell == "" || c.EndCell == c.StartCell {
			return fmt.Sprintf("sheet:%d!%s", c.Sheet, c.StartCell), nil
		}
		if !cellPattern.MatchString(c.EndCell) {
			return "", fmt.Errorf("invalid tabular range")
		}
		return fmt.Sprintf("sheet:%d!%s:%s", c.Sheet, c.StartCell, c.EndCell), nil
	case KindLogical:
		if !pathPattern.MatchString(c.Path) || c.Start < 0 || c.End < c.Start {
			return "", fmt.Errorf("invalid logical coordinate")
		}
		prefix := ""
		if c.StablePath != "" {
			if !pathPattern.MatchString(c.StablePath) {
				return "", fmt.Errorf("invalid stable prefix")
			}
			prefix = c.StablePath + "!"
		}
		return fmt.Sprintf("%s%s@%d-%d", prefix, c.Path, c.Start, c.End), nil
	default:
		return "", fmt.Errorf("unsupported anchor kind %q", kind)
	}
}

func parseCoordinates(kind Kind, raw string) (Coordinates, error) {
	switch kind {
	case KindGeometric:
		at := strings.Split(raw, "@")
		if len(at) != 2 || !strings.HasPrefix(at[0], "p") {
			return Coordinates{}, fmt.Errorf("invalid geometric coordinate")
		}
		page, err := parsePositive(strings.TrimPrefix(at[0], "p"))
		if err != nil {
			return Coordinates{}, err
		}
		values := strings.Split(at[1], ",")
		if len(values) != 4 {
			return Coordinates{}, fmt.Errorf("invalid geometric box")
		}
		var box [4]float64
		for i, value := range values {
			if len(value) != 8 || value[1] != '.' {
				return Coordinates{}, fmt.Errorf("geometric values need six decimals")
			}
			box[i], err = strconv.ParseFloat(value, 64)
			if err != nil || box[i] < 0 || box[i] > 1 {
				return Coordinates{}, fmt.Errorf("invalid geometric value")
			}
		}
		return Coordinates{Page: page, Box: box}, nil
	case KindTabular:
		if !strings.HasPrefix(raw, "sheet:") {
			return Coordinates{}, fmt.Errorf("invalid tabular coordinate")
		}
		parts := strings.SplitN(strings.TrimPrefix(raw, "sheet:"), "!", 2)
		if len(parts) != 2 {
			return Coordinates{}, fmt.Errorf("invalid tabular coordinate")
		}
		sheet, err := parsePositive(parts[0])
		if err != nil {
			return Coordinates{}, err
		}
		cells := strings.Split(parts[1], ":")
		if len(cells) > 2 || !cellPattern.MatchString(cells[0]) {
			return Coordinates{}, fmt.Errorf("invalid cells")
		}
		end := ""
		if len(cells) == 2 {
			end = cells[1]
		}
		return Coordinates{Sheet: sheet, StartCell: cells[0], EndCell: end}, nil
	case KindLogical:
		parts := strings.SplitN(raw, "@", 2)
		if len(parts) != 2 {
			return Coordinates{}, fmt.Errorf("invalid logical coordinate")
		}
		bounds := strings.Split(parts[1], "-")
		if len(bounds) != 2 {
			return Coordinates{}, fmt.Errorf("invalid logical offsets")
		}
		start, err := strconv.Atoi(bounds[0])
		if err != nil || (len(bounds[0]) > 1 && bounds[0][0] == '0') {
			return Coordinates{}, fmt.Errorf("invalid logical start")
		}
		end, err := strconv.Atoi(bounds[1])
		if err != nil || end < start || (len(bounds[1]) > 1 && bounds[1][0] == '0') {
			return Coordinates{}, fmt.Errorf("invalid logical end")
		}
		stable := ""
		path := parts[0]
		if strings.Contains(parts[0], "!") {
			p := strings.SplitN(parts[0], "!", 2)
			stable, path = p[0], p[1]
		}
		if !pathPattern.MatchString(path) || (stable != "" && !pathPattern.MatchString(stable)) {
			return Coordinates{}, fmt.Errorf("invalid logical path")
		}
		return Coordinates{StablePath: stable, Path: path, Start: start, End: end}, nil
	default:
		return Coordinates{}, fmt.Errorf("unsupported anchor kind %q", kind)
	}
}
