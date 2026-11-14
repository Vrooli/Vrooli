// Package geometry validates finite AI-authored geometry recipes and compiles
// them into deterministic, self-contained GLB bytes. It has no I/O authority.
package geometry

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
)

// SchemaJSON is the strict Gateway schema sent with the user's bounded task.
func SchemaJSON() string {
	return `{"type":"object","additionalProperties":false,"required":["version","profile","operation","envelope_mm","parts","screen"],"properties":{"version":{"const":"device_geometry_recipe.v1"},"profile":{"const":"detailed_device_v1"},"operation":{"enum":["construct","correct"]},"envelope_mm":{"type":"array","items":{"type":"number","exclusiveMinimum":0,"maximum":1000},"minItems":3,"maxItems":3},"parts":{"type":"array","minItems":1,"maxItems":32,"items":{"type":"object","additionalProperties":false,"required":["id","kind","role","center_mm","size_mm","rotation_deg","bevel_mm","material"],"properties":{"id":{"type":"string","minLength":1,"maxLength":48,"pattern":"^[a-z0-9_-]+$"},"kind":{"enum":["box","rounded_box","cylinder"]},"role":{"enum":["housing","bezel","display","button","camera","port","speaker","control","detail"]},"center_mm":{"type":"array","items":{"type":"number"},"minItems":3,"maxItems":3},"size_mm":{"type":"array","items":{"type":"number","exclusiveMinimum":0},"minItems":3,"maxItems":3},"rotation_deg":{"type":"array","items":{"enum":[0,90,180,270]},"minItems":3,"maxItems":3},"bevel_mm":{"type":"number","minimum":0,"description":"For rounded_box, must be strictly less than half the smallest size_mm dimension. For box and cylinder, it must be zero."},"material":{"enum":["matte_polymer","gloss_polymer","rubber","glass","anodized_metal","neutral_metal"]}}}},"screen":{"type":"object","additionalProperties":false,"required":["part_id","face","center_uv","size_uv","rotation_deg"],"properties":{"part_id":{"type":"string"},"face":{"enum":["front:+z","back:-z","left:-x","right:+x","top:+y","bottom:-y"]},"center_uv":{"type":"array","items":{"type":"number","minimum":0,"maximum":1},"minItems":2,"maxItems":2},"size_uv":{"type":"array","items":{"type":"number","exclusiveMinimum":0,"maximum":1},"minItems":2,"maxItems":2},"rotation_deg":{"enum":[0,90,180,270]}}}}}`
}

const MaxRecipeBytes = 32 << 10

var ErrInvalidRecipe = errors.New("geometry: invalid recipe")

// SemanticIssue is safe user-facing feedback derived only from finite,
// validated recipe fields. It never contains arbitrary provider text.
type SemanticIssue struct {
	Code                string  `json:"code"`
	PartID              string  `json:"part_id,omitempty"`
	Axis                string  `json:"axis,omitempty"`
	BevelMM             float64 `json:"bevel_mm,omitempty"`
	MinimumDimensionMM  float64 `json:"minimum_dimension_mm,omitempty"`
	MaxBevelExclusiveMM float64 `json:"max_bevel_exclusive_mm,omitempty"`
	Message             string  `json:"message"`
}

type issueError struct {
	issue SemanticIssue
}

func (e issueError) Error() string { return e.issue.Message }
func (e issueError) Unwrap() error { return ErrInvalidRecipe }

func IssueFromError(err error) (SemanticIssue, bool) {
	var typed issueError
	if !errors.As(err, &typed) {
		return SemanticIssue{}, false
	}
	return typed.issue, true
}

type Recipe struct {
	Version   string     `json:"version"`
	Profile   string     `json:"profile"`
	Operation string     `json:"operation"`
	Envelope  Dimensions `json:"envelope_mm"`
	Parts     []Part     `json:"parts"`
	Screen    Screen     `json:"screen"`
}
type Dimensions [3]float64
type Part struct {
	ID       string     `json:"id"`
	Kind     string     `json:"kind"`
	Role     string     `json:"role"`
	Center   [3]float64 `json:"center_mm"`
	Size     [3]float64 `json:"size_mm"`
	Rotation [3]int     `json:"rotation_deg"`
	Bevel    float64    `json:"bevel_mm"`
	Material string     `json:"material"`
}
type Screen struct {
	PartID   string     `json:"part_id"`
	Face     string     `json:"face"`
	Center   [2]float64 `json:"center_uv"`
	Size     [2]float64 `json:"size_uv"`
	Rotation int        `json:"rotation_deg"`
}

// Decode accepts exactly the declared recipe fields and bounds parsing work.
func Decode(data []byte) (Recipe, []byte, error) {
	if len(data) == 0 || len(data) > MaxRecipeBytes {
		return Recipe{}, nil, fmt.Errorf("%w: size outside limit", ErrInvalidRecipe)
	}
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return Recipe{}, nil, fmt.Errorf("%w: %v", ErrInvalidRecipe, err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return Recipe{}, nil, fmt.Errorf("%w: %v", ErrInvalidRecipe, err)
	}
	for _, key := range []string{"version", "profile", "operation", "envelope_mm", "parts", "screen"} {
		if _, ok := raw[key]; !ok {
			return Recipe{}, nil, fmt.Errorf("%w: missing %s", ErrInvalidRecipe, key)
		}
	}
	if err := arrayCardinality(raw["envelope_mm"], 3); err != nil {
		return Recipe{}, nil, err
	}
	var parts []map[string]json.RawMessage
	if err := json.Unmarshal(raw["parts"], &parts); err != nil {
		return Recipe{}, nil, fmt.Errorf("%w: parts must be an array", ErrInvalidRecipe)
	}
	for _, part := range parts {
		for _, key := range []string{"id", "kind", "role", "center_mm", "size_mm", "rotation_deg", "bevel_mm", "material"} {
			if _, ok := part[key]; !ok {
				return Recipe{}, nil, fmt.Errorf("%w: part is missing %s", ErrInvalidRecipe, key)
			}
		}
		for _, key := range []string{"center_mm", "size_mm", "rotation_deg"} {
			if err := arrayCardinality(part[key], 3); err != nil {
				return Recipe{}, nil, err
			}
		}
	}
	var screen map[string]json.RawMessage
	if err := json.Unmarshal(raw["screen"], &screen); err != nil {
		return Recipe{}, nil, fmt.Errorf("%w: screen must be an object", ErrInvalidRecipe)
	}
	for _, key := range []string{"part_id", "face", "center_uv", "size_uv", "rotation_deg"} {
		if _, ok := screen[key]; !ok {
			return Recipe{}, nil, fmt.Errorf("%w: screen is missing %s", ErrInvalidRecipe, key)
		}
	}
	for _, key := range []string{"center_uv", "size_uv"} {
		if err := arrayCardinality(screen[key], 2); err != nil {
			return Recipe{}, nil, err
		}
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	var r Recipe
	if err := d.Decode(&r); err != nil {
		return Recipe{}, nil, fmt.Errorf("%w: %v", ErrInvalidRecipe, err)
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return Recipe{}, nil, fmt.Errorf("%w: trailing JSON", ErrInvalidRecipe)
	}
	if err := Validate(r); err != nil {
		return Recipe{}, nil, err
	}
	b, err := json.Marshal(r)
	if err != nil {
		return Recipe{}, nil, err
	}
	return r, b, nil
}

func arrayCardinality(raw json.RawMessage, want int) error {
	var values []json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil || len(values) != want {
		return fmt.Errorf("%w: expected an array with exactly %d items", ErrInvalidRecipe, want)
	}
	return nil
}

func rejectDuplicateJSONKeys(data []byte) error {
	d := json.NewDecoder(bytes.NewReader(data))
	if err := walkJSONValue(d); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return errors.New("trailing JSON value")
	}
	return nil
}
func walkJSONValue(d *json.Decoder) error {
	token, err := d.Token()
	if err != nil {
		return err
	}
	switch token {
	case json.Delim('{'):
		seen := map[string]bool{}
		for d.More() {
			keyToken, err := d.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("object key is not a string")
			}
			if seen[key] {
				return fmt.Errorf("duplicate JSON field %q", key)
			}
			seen[key] = true
			if err := walkJSONValue(d); err != nil {
				return err
			}
		}
		_, err = d.Token()
		return err
	case json.Delim('['):
		for d.More() {
			if err := walkJSONValue(d); err != nil {
				return err
			}
		}
		_, err = d.Token()
		return err
	default:
		return nil
	}
}

func Validate(r Recipe) error {
	bad := func(reason string) error { return fmt.Errorf("%w: %s", ErrInvalidRecipe, reason) }
	if r.Version != "device_geometry_recipe.v1" || r.Profile != "detailed_device_v1" || (r.Operation != "construct" && r.Operation != "correct") {
		return bad("unsupported version, profile, or operation")
	}
	for _, n := range r.Envelope {
		if !finite(n) || n <= 0 || n > 1000 {
			return bad("envelope dimensions are outside bounds")
		}
	}
	if len(r.Parts) == 0 || len(r.Parts) > 32 {
		return bad("part count is outside 1..32")
	}
	ids := map[string]bool{}
	displays := 0
	roles := set("housing", "bezel", "display", "button", "camera", "port", "speaker", "control", "detail")
	materials := set("matte_polymer", "gloss_polymer", "rubber", "glass", "anodized_metal", "neutral_metal")
	for _, p := range r.Parts {
		if len(p.ID) == 0 || len(p.ID) > 48 || !asciiID(p.ID) || ids[p.ID] {
			return bad("part ids must be unique stable lowercase identifiers")
		}
		ids[p.ID] = true
		if !roles[p.Role] || !materials[p.Material] {
			return bad("unsupported role or material")
		}
		if p.Role == "display" {
			displays++
		}
		if p.Kind != "box" && p.Kind != "rounded_box" && p.Kind != "cylinder" {
			return bad("unsupported primitive")
		}
		if p.Kind == "rounded_box" {
			if !finite(p.Bevel) || p.Bevel <= 0 {
				return bad("rounded boxes need a positive radius")
			}
		} else if p.Bevel != 0 {
			return bad("bevel is only valid on rounded boxes")
		}
		for i := 0; i < 3; i++ {
			if !finite(p.Center[i]) || !finite(p.Size[i]) || p.Size[i] <= 0 || p.Size[i] > r.Envelope[i]+0.1 {
				return bad("part dimensions are invalid")
			}
			if p.Center[i]-p.Size[i]/2 < -r.Envelope[i]/2-0.1 || p.Center[i]+p.Size[i]/2 > r.Envelope[i]/2+0.1 {
				return issueError{issue: SemanticIssue{Code: "part_outside_envelope", PartID: p.ID, Axis: axisName(i), Message: "part exceeds the locked envelope"}}
			}
			if !rightAngle(p.Rotation[i]) {
				return bad("rotation must use right angles")
			}
		}
		rotated := rotatedSize(p.Size, p.Rotation)
		for i := range rotated {
			if math.Abs(p.Center[i])+rotated[i]/2 > r.Envelope[i]/2+0.1 {
				return issueError{issue: SemanticIssue{Code: "rotated_part_outside_envelope", PartID: p.ID, Axis: axisName(i), Message: "rotated part exceeds the locked envelope"}}
			}
		}
		if p.Kind == "rounded_box" {
			minimumDimension := math.Min(p.Size[0], math.Min(p.Size[1], p.Size[2]))
			maxBevelExclusive := minimumDimension / 2
			if p.Bevel >= maxBevelExclusive {
				return issueError{issue: SemanticIssue{
					Code:                "bevel_exceeds_part_dimensions",
					PartID:              p.ID,
					BevelMM:             p.Bevel,
					MinimumDimensionMM:  minimumDimension,
					MaxBevelExclusiveMM: maxBevelExclusive,
					Message:             "bevel must be less than half the smallest part dimension",
				}}
			}
		}
	}
	if displays != 1 || !ids[r.Screen.PartID] {
		return bad("screen must bind the unique display part")
	}
	var display *Part
	for i := range r.Parts {
		if r.Parts[i].ID == r.Screen.PartID {
			display = &r.Parts[i]
		}
	}
	if display == nil || display.Role != "display" {
		return bad("screen binding must reference a display")
	}
	if !set("front:+z", "back:-z", "left:-x", "right:+x", "top:+y", "bottom:-y")[r.Screen.Face] {
		return bad("unsupported display face")
	}
	if !rightAngle(r.Screen.Rotation) {
		return bad("screen rotation must use right angles")
	}
	for _, n := range r.Screen.Center {
		if !finite(n) || n < 0 || n > 1 {
			return bad("screen center outside normalized face")
		}
	}
	for _, n := range r.Screen.Size {
		if !finite(n) || n <= 0 || n > 1 {
			return bad("screen size outside normalized face")
		}
	}
	if !screenFitsDisplayFace(*display, r.Screen) {
		return bad("screen rectangle exceeds the actual flat display face")
	}
	if r.Screen.Center[0]-r.Screen.Size[0]/2 < 0 || r.Screen.Center[0]+r.Screen.Size[0]/2 > 1 || r.Screen.Center[1]-r.Screen.Size[1]/2 < 0 || r.Screen.Center[1]+r.Screen.Size[1]/2 > 1 {
		return bad("screen rectangle exceeds face")
	}
	ua, va := screenAxes(r.Screen.Face)
	if display.Size[ua]*r.Screen.Size[0] <= 0 || display.Size[va]*r.Screen.Size[1] <= 0 {
		return bad("screen dimensions must be positive")
	}
	for _, vertex := range plane(screenPlane(r)) {
		for axis := 0; axis < 3; axis++ {
			if math.Abs(float64(vertex[axis])) > r.Envelope[axis]/2+0.1 {
				return bad("screen surface exceeds locked envelope after stand-off")
			}
		}
	}
	return nil
}

func axisName(axis int) string {
	return [...]string{"x", "y", "z"}[axis]
}

func screenAxes(face string) (int, int) {
	switch face {
	case "left:-x", "right:+x":
		return 2, 1
	case "top:+y", "bottom:-y":
		return 0, 2
	default:
		return 0, 1
	}
}

func screenFitsDisplayFace(display Part, screen Screen) bool {
	if display.Kind == "box" {
		return true
	}
	ua, va := screenAxes(screen.Face)
	width := display.Size[ua] * screen.Size[0]
	height := display.Size[va] * screen.Size[1]
	centerU := math.Abs((screen.Center[0] - .5) * display.Size[ua])
	centerV := math.Abs((screen.Center[1] - .5) * display.Size[va])
	if screen.Rotation == 90 || screen.Rotation == 270 {
		width, height = height, width
	}
	switch display.Kind {
	case "rounded_box":
		return centerU+width/2 <= display.Size[ua]/2-display.Bevel &&
			centerV+height/2 <= display.Size[va]/2-display.Bevel
	case "cylinder":
		// Cylinders are flat only on their local +/-Z caps. The compiler emits
		// a 16-sided cap, so prove all four screen corners lie inside that
		// actual inscribed polygon, not merely inside its bounding box.
		if screen.Face != "front:+z" && screen.Face != "back:-z" {
			return false
		}
		rx, ry := display.Size[0]/2, display.Size[1]/2
		for _, x := range []float64{centerU - width/2, centerU + width/2} {
			for _, y := range []float64{centerV - height/2, centerV + height/2} {
				nx, ny := x/rx, y/ry
				for i := 0; i < 16; i++ {
					a := (float64(i) + .5) * 2 * math.Pi / 16
					if math.Abs(nx*math.Cos(a)+ny*math.Sin(a)) > math.Cos(math.Pi/16)+1e-12 {
						return false
					}
				}
			}
		}
		return true
	default:
		return false
	}
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
func rightAngle(v int) bool { return v == 0 || v == 90 || v == 180 || v == 270 }
func rotatedSize(size [3]float64, rotation [3]int) [3]float64 {
	// A right-angle Euler rotation permutes axis-aligned extents; evaluate the
	// eight corners so the locked envelope check remains exact.
	out := [3]float64{}
	for _, x := range []float64{-1, 1} {
		for _, y := range []float64{-1, 1} {
			for _, z := range []float64{-1, 1} {
				v := [3]float64{x * size[0] / 2, y * size[1] / 2, z * size[2] / 2}
				for axis, deg := range rotation {
					turns := deg / 90 % 4
					for n := 0; n < turns; n++ {
						a, b := (axis+1)%3, (axis+2)%3
						v[a], v[b] = -v[b], v[a]
					}
				}
				for i := range out {
					if math.Abs(v[i]) > out[i] {
						out[i] = math.Abs(v[i])
					}
				}
			}
		}
	}
	return [3]float64{out[0] * 2, out[1] * 2, out[2] * 2}
}
func set(v ...string) map[string]bool {
	m := map[string]bool{}
	for _, s := range v {
		m[s] = true
	}
	return m
}
func asciiID(v string) bool {
	for _, r := range v {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || strings.ContainsRune("-_", r)) {
			return false
		}
	}
	return true
}
