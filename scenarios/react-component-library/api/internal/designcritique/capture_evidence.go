package designcritique

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"time"

	"react-component-library/internal/designcapture"
)

// CaptureEvidenceVerifier checks current availability through the browser owner.
// It never dispatches, accepts, or alters capture evidence.
type CaptureEvidenceVerifier struct {
	Captures    designcapture.Repository
	Screenshots designcapture.ScreenshotResolver
	Client      *http.Client
}

func (v CaptureEvidenceVerifier) Verify(ctx context.Context, target Target, e Evidence) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if v.Captures == nil || v.Screenshots == nil {
		return fmt.Errorf("capture evidence owner unavailable")
	}
	op, err := v.Captures.Get(ctx, e.CaptureID)
	if err != nil {
		return err
	}
	renderHash := e.RenderHash
	if renderHash == "" {
		renderHash = target.RenderHash
	}
	t := op.Request.Target
	if op.State != designcapture.Completed || t.Scenario != target.Scenario || t.DesignID != target.DesignID || t.Revision != target.Revision || t.RenderHash != renderHash || op.Request.Width != e.Width || op.Request.Height != e.Height {
		return fmt.Errorf("capture does not match reviewed revision, render or viewport")
	}
	regionFound := e.Region == "$page"
	for _, a := range op.Artifacts {
		if a.Evidence != nil {
			for _, region := range a.Evidence.Regions {
				if region.Region == e.Region {
					regionFound = true
				}
			}
		}
	}
	if !regionFound {
		return fmt.Errorf("finding region is not present in captured geometry")
	}
	shot, err := (designcapture.Service{Repository: v.Captures}).Screenshot(ctx, e.CaptureID, e.Artifact, v.Screenshots)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, shot.URL, nil)
	if err != nil {
		return err
	}
	client := http.Client{}
	if v.Client != nil {
		client = *v.Client
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return fmt.Errorf("screenshot owner redirected image evidence")
	}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("screenshot bytes unavailable: HTTP %d", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 32*1024*1024+1))
	if err != nil {
		return err
	}
	if len(data) > 32*1024*1024 {
		return fmt.Errorf("screenshot exceeds bounded review image size")
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("screenshot image metadata invalid: %w", err)
	}
	if (format != "png" && format != "jpeg") || config.Width != shot.Width || config.Height != shot.Height || "image/"+format != shot.ContentType {
		return fmt.Errorf("screenshot bytes differ from owner descriptor")
	}
	// Bound decompression as well as encoded input. A larger image remains
	// unavailable for this review budget rather than claiming it was inspected.
	if int64(config.Width)*int64(config.Height) > 16*1024*1024 {
		return fmt.Errorf("screenshot exceeds review pixel budget")
	}
	if _, _, err := image.Decode(bytes.NewReader(data)); err != nil {
		return fmt.Errorf("screenshot image is incomplete or corrupt: %w", err)
	}
	return nil
}
