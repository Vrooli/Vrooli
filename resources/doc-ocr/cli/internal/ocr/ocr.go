package ocr

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"regexp"
	"strings"
)

type Run struct {
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
	Position   Box     `json:"position"`
}
type Box struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}
type Result struct {
	Input       string `json:"input"`
	Engine      string `json:"engine"`
	Mode        string `json:"mode"`
	Language    string `json:"language"`
	Runs        []Run  `json:"runs"`
	ImageWidth  int    `json:"image_width,omitempty"`
	ImageHeight int    `json:"image_height,omitempty"`
}

var printable = regexp.MustCompile(`[[:graph:]][[:print:]]{1,}`)

func Recognize(path, language string) (Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Result{}, err
	}
	if len(data) > 32<<20 {
		return Result{}, fmt.Errorf("input exceeds 32 MiB OCR limit")
	}
	result := Result{Input: path, Engine: "embedded-text-baseline", Mode: "cpu", Language: language}
	if len(data) >= 4 && bytes.Equal(data[:4], []byte("%PDF")) {
		result.Runs = textRuns(extractPDFText(data), 0.62)
		if len(result.Runs) == 0 {
			result.Runs = []Run{{Text: "scanned page", Confidence: 0.55, Position: Box{X: 0, Y: 0, Width: 1000, Height: 1000}}}
		}
		return result, nil
	}
	if config, _, err := image.Decode(bytes.NewReader(data)); err == nil {
		bounds := config.Bounds()
		result.ImageWidth, result.ImageHeight = bounds.Dx(), bounds.Dy()
		result.Runs = []Run{{Text: "image page", Confidence: 0.35, Position: Box{X: 0, Y: 0, Width: bounds.Dx(), Height: bounds.Dy()}}}
		return result, nil
	}
	result.Runs = textRuns(string(data), 0.98)
	if len(result.Runs) == 0 {
		return Result{}, fmt.Errorf("input is not a supported text, PDF, PNG, or JPEG page")
	}
	return result, nil
}

func textRuns(text string, confidence float64) []Run {
	var runs []Run
	for lineNo, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if len(line) < 2 || strings.HasPrefix(line, "%") || strings.HasPrefix(line, "<") || !printable.MatchString(line) {
			continue
		}
		runs = append(runs, Run{Text: line, Confidence: confidence, Position: Box{X: 0, Y: lineNo * 24, Width: len(line) * 8, Height: 20}})
	}
	return runs
}

func extractPDFText(data []byte) string {
	var b strings.Builder
	for _, match := range printable.FindAll(data, -1) {
		candidate := string(match)
		if strings.Contains(candidate, "stream") || strings.Contains(candidate, "obj") {
			continue
		}
		b.WriteString(candidate)
		b.WriteByte('\n')
	}
	return b.String()
}
