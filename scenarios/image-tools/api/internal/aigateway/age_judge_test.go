package aigateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestPrepareVisionBytesBoundsAndReencodes proves the reviewer's in-memory image
// reaches the gateway as a bounded JPEG, the shape the vision route accepts (a raw
// 1.4 MB PNG ended "provider.execution_failed").
func TestPrepareVisionBytesBoundsAndReencodes(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2000, 1000))
	for y := 0; y < 1000; y++ {
		for x := 0; x < 2000; x++ {
			src.Set(x, y, color.RGBA{uint8(x), uint8(y), 120, 255})
		}
	}
	var raw bytes.Buffer
	if err := png.Encode(&raw, src); err != nil {
		t.Fatal(err)
	}
	got, w, h, media, err := prepareVisionBytes(raw.Bytes())
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if w != 2000 || h != 1000 {
		t.Fatalf("reported source size %dx%d, want 2000x1000", w, h)
	}
	if media != "image/jpeg" || !bytes.HasPrefix(got, []byte{0xFF, 0xD8}) {
		t.Fatalf("media %q is not JPEG", media)
	}
	decoded, _, err := image.Decode(bytes.NewReader(got))
	if err != nil {
		t.Fatalf("decode prepared: %v", err)
	}
	if max(decoded.Bounds().Dx(), decoded.Bounds().Dy()) > visionMaxImageEdge {
		t.Fatalf("prepared image is %v, longer than %d", decoded.Bounds().Size(), visionMaxImageEdge)
	}
}

type recordingVision struct{ req VisionRequest }

func (r *recordingVision) Ask(_ context.Context, req VisionRequest) (VisionAnswer, error) {
	r.req = req
	return VisionAnswer{Text: `{"apparent_age_years": 33, "could_be_under_18": false, "confidence": "high"}`}, nil
}
func (r *recordingVision) Reachable(context.Context) error { return nil }

func TestAgeJudgeSendsAPreparedImage(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1800, 1200))
	for i := range src.Pix {
		src.Pix[i] = 200
	}
	var raw bytes.Buffer
	if err := png.Encode(&raw, src); err != nil {
		t.Fatal(err)
	}
	gw := &recordingVision{}
	got, err := NewAgeJudge(gw).Judge(context.Background(), raw.Bytes())
	if err != nil {
		t.Fatalf("judge: %v", err)
	}
	if got.ApparentAge != 33 {
		t.Fatalf("judgement = %+v", got)
	}
	if gw.req.MediaType != "image/jpeg" || len(gw.req.Image) == 0 {
		t.Fatalf("request media=%q bytes=%d, want a JPEG payload", gw.req.MediaType, len(gw.req.Image))
	}
}

type failingVision struct{ err error }

func (f *failingVision) Ask(context.Context, VisionRequest) (VisionAnswer, error) {
	return VisionAnswer{}, f.err
}
func (f *failingVision) Reachable(context.Context) error { return f.err }

func TestAgeJudgeFallsBackToLoopbackOllamaWhenTheGatewayCannotExecute(t *testing.T) {
	var gotModel string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode: %v", err)
		}
		gotModel, _ = body["model"].(string)
		_, _ = w.Write([]byte(`{"message":{"content":"{\"apparent_age_years\": 31, \"could_be_under_18\": false, \"confidence\": \"high\"}"}}`))
	}))
	defer server.Close()
	t.Setenv(ageJudgeOllamaURLEnv, server.URL)

	src := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for i := range src.Pix {
		src.Pix[i] = 180
	}
	var raw bytes.Buffer
	if err := png.Encode(&raw, src); err != nil {
		t.Fatal(err)
	}
	got, err := NewAgeJudge(&failingVision{err: errors.New("provider.execution_failed")}).Judge(context.Background(), raw.Bytes())
	if err != nil {
		t.Fatalf("fallback: %v", err)
	}
	if got.ApparentAge != 31 || got.CouldBeUnder18 {
		t.Fatalf("judgement = %+v", got)
	}
	if gotModel == "" {
		t.Fatal("the fallback did not name a model")
	}
}

func TestParseAgeJudgement(t *testing.T) {
	cases := []struct {
		name, reply string
		wantAge     int
		wantUnder   bool
		wantErr     bool
	}{
		{"plain json", `{"apparent_age_years": 34, "could_be_under_18": false, "confidence": "High"}`, 34, false, false},
		{"fenced and chatty", "Here you go:\n```json\n{\"apparent_age_years\": 12, \"could_be_under_18\": true, \"confidence\": \"high\"}\n```", 12, true, false},
		{"thinking block first", "<think>she looks older</think>{\"apparent_age_years\": 41, \"could_be_under_18\": false, \"confidence\": \"medium\"}", 41, false, false},
		{"missing flag is unusable", `{"apparent_age_years": 30, "confidence": "high"}`, 0, false, true},
		{"missing age is unusable", `{"could_be_under_18": false, "confidence": "high"}`, 0, false, true},
		{"prose is unusable", `I cannot help with that.`, 0, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseAgeJudgement(tc.reply)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if err == nil && (got.ApparentAge != tc.wantAge || got.CouldBeUnder18 != tc.wantUnder) {
				t.Fatalf("got %+v", got)
			}
		})
	}
}
