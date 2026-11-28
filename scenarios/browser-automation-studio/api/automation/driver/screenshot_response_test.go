package driver

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestCaptureScreenshotResponseNormalizesDataURIAndDimensions(t *testing.T) {
	var encoded bytes.Buffer
	frame := image.NewRGBA(image.Rect(0, 0, 390, 900))
	frame.Set(0, 0, color.White)
	if err := png.Encode(&encoded, frame); err != nil {
		t.Fatal(err)
	}
	raw := base64.StdEncoding.EncodeToString(encoded.Bytes())
	var response CaptureScreenshotResponse
	if err := json.Unmarshal([]byte(`{"screenshot":"data:image/png;base64,`+raw+`"}`), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data != raw || response.MediaType != "image/png" || response.Width != 390 || response.Height != 900 {
		t.Fatalf("screenshot response = %+v", response)
	}
}

func TestCaptureScreenshotResponseRejectsMalformedDataURI(t *testing.T) {
	var response CaptureScreenshotResponse
	if err := json.Unmarshal([]byte(`{"screenshot":"data:image/png,not-base64"}`), &response); err == nil {
		t.Fatal("malformed screenshot data URI was accepted")
	}
}
