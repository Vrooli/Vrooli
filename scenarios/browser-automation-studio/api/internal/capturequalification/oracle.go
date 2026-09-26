package capturequalification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/png"
	"math"
	"os"
)

type artifact struct {
	Type      string `json:"type"`
	Path      string `json:"path"`
	Reference string `json:"reference"`
	Primary   bool   `json:"primary"`
	SHA256    string `json:"sha256,omitempty"`
	Bytes     int    `json:"bytes,omitempty"`
}

type captureResponse struct {
	ExecutionID string     `json:"execution_id"`
	DurationMS  float64    `json:"duration_ms"`
	DOMTreeJSON string     `json:"dom_tree_json"`
	Artifacts   []artifact `json:"artifacts"`
	Readiness   struct {
		Outcome string `json:"outcome"`
	} `json:"readiness"`
}

type treeNode struct {
	ID       string     `json:"id"`
	Tag      string     `json:"tagName"`
	Text     string     `json:"text"`
	Children []treeNode `json:"children"`
	Rect     struct {
		Width  float64 `json:"width"`
		Height float64 `json:"height"`
	} `json:"rect"`
	Computed map[string]string `json:"computed"`
}

func verifyCapture(c captureResponse, observed []observation, nonce, trial, url string) ([]artifact, error) {
	if c.ExecutionID == "" || c.Readiness.Outcome != "ready" || c.DurationMS <= 0 || math.IsInf(c.DurationMS, 0) || math.IsNaN(c.DurationMS) {
		return nil, fmt.Errorf("missing successful owner admission/readiness/duration")
	}
	if len(observed) != 1 {
		return nil, fmt.Errorf("expected one independent fixture observation, got %d", len(observed))
	}
	want := observation{URL: url, Width: 1280, Height: 720, DPR: 1, Heading: "Capture fixture " + nonce, Value: "fixture-value"}
	if observed[0] != want {
		return nil, fmt.Errorf("fixture viewport/DPR/identity/input observation mismatch")
	}
	if err := verifyTree(c.DOMTreeJSON, nonce); err != nil {
		return nil, err
	}
	var images, primary, trees int
	verified := make([]artifact, 0, len(c.Artifacts))
	for _, a := range c.Artifacts {
		if a.Type != "screenshot" && a.Type != "dom-tree" {
			continue
		}
		b, err := os.ReadFile(a.Path)
		if err != nil {
			return verified, fmt.Errorf("read %s artifact: %w", a.Type, err)
		}
		a.SHA256, a.Bytes = digest(b), len(b)
		verified = append(verified, a)
		if a.Type == "screenshot" {
			images++
			if a.Primary {
				primary++
			}
			err = verifyPNG(b, paintColor(nonce, trial))
		} else {
			trees++
			if a.Reference != "bas-capture://"+c.ExecutionID+"/dom_tree" || !equalJSON(b, []byte(c.DOMTreeJSON)) {
				err = fmt.Errorf("stored snapshot differs from returned snapshot or owner reference")
			}
		}
		if err != nil {
			return verified, err
		}
	}
	if images == 0 || primary != 1 || trees != 1 {
		return verified, fmt.Errorf("incomplete artifact set: %d images, %d primary, %d trees", images, primary, trees)
	}
	return verified, nil
}

func verifyPNG(b []byte, paint [3]byte) error {
	config, err := png.DecodeConfig(bytes.NewReader(b))
	if err != nil || config.Width != 1280 || config.Height != 720 {
		return fmt.Errorf("PNG viewport/DPR mismatch or invalid header: %dx%d: %v", config.Width, config.Height, err)
	}
	im, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("decode screenshot: %w", err)
	}
	if im.Bounds().Dx() != 1280 || im.Bounds().Dy() != 720 {
		return fmt.Errorf("PNG viewport/DPR mismatch: %v", im.Bounds())
	}
	points := []struct {
		x, y int
		rgb  [3]byte
	}{
		{320, 180, [3]byte{210, 30, 50}}, {960, 180, [3]byte{20, 180, 70}},
		{320, 540, [3]byte{20, 60, 210}}, {960, 540, [3]byte{230, 185, 20}},
		{24, 416, paint},
	}
	for _, p := range points {
		r, g, b, a := im.At(p.x, p.y).RGBA()
		if a != 65535 || [3]byte{byte(r >> 8), byte(g >> 8), byte(b >> 8)} != p.rgb {
			return fmt.Errorf("screenshot pixel mismatch at (%d,%d)", p.x, p.y)
		}
	}
	return nil
}

func verifyTree(raw, nonce string) error {
	var root treeNode
	if err := json.Unmarshal([]byte(raw), &root); err != nil {
		return fmt.Errorf("decode computed snapshot: %w", err)
	}
	if root.Rect.Width != 1280 || root.Rect.Height != 720 {
		return fmt.Errorf("computed snapshot viewport mismatch")
	}
	var sections int
	var heading, color bool
	var walk func(treeNode)
	walk = func(n treeNode) {
		if n.ID == "ready" {
			heading = n.Text == "Capture fixture "+nonce
		}
		if n.Tag == "SECTION" {
			if sections == 0 {
				color = n.Computed["backgroundColor"] == "rgb(210, 30, 50)"
			}
			sections++
		}
		for _, child := range n.Children {
			walk(child)
		}
	}
	walk(root)
	if !heading || !color || sections != 4 {
		return fmt.Errorf("computed snapshot missing fixture identity, structure or styles")
	}
	return nil
}

func equalJSON(a, b []byte) bool {
	var x, y any
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
		return false
	}
	xb, _ := json.Marshal(x)
	yb, _ := json.Marshal(y)
	return bytes.Equal(xb, yb)
}
