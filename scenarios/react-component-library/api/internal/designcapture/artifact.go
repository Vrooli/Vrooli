package designcapture

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"path"
	"strings"

	apiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	executionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	"google.golang.org/protobuf/encoding/protojson"
)

// Screenshot describes bytes served by the browser owner, without copying them
// or exposing a producer filesystem location through the capture receipt.
type Screenshot struct {
	Reference     string
	URL           string
	Width, Height int
	ContentType   string
}
type ScreenshotResolver interface {
	ResolveScreenshot(context.Context, Operation, Artifact) (Screenshot, error)
}

func (s Service) Screenshot(ctx context.Context, id, reference string, resolver ScreenshotResolver) (Screenshot, error) {
	if s.Repository == nil || resolver == nil {
		return Screenshot{}, fmt.Errorf("capture repository and screenshot resolver are required")
	}
	op, err := s.Repository.Get(ctx, id)
	if err != nil {
		return Screenshot{}, err
	}
	if op.State != Completed {
		return Screenshot{}, fmt.Errorf("verified completed capture is required")
	}
	for _, artifact := range op.Artifacts {
		if artifact.Reference == reference && artifact.Kind == "screenshot" {
			return resolver.ResolveScreenshot(ctx, op, artifact)
		}
	}
	return Screenshot{}, fmt.Errorf("screenshot is not part of this capture receipt")
}

func (d BASDispatcher) ResolveScreenshot(ctx context.Context, op Operation, artifact Artifact) (Screenshot, error) {
	parts := strings.Split(artifact.Reference, ":")
	if len(parts) != 3 || parts[0] != "bas" || parts[1] != op.ProducerID || parts[2] == "" || artifact.Kind != "screenshot" {
		return Screenshot{}, fmt.Errorf("screenshot producer identity mismatch")
	}
	raw, err := protojson.Marshal(&apiv1.GetExecutionScreenshotsRequest{ExecutionId: op.ProducerID})
	if err != nil {
		return Screenshot{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(d.BASBaseURL, "/")+"/browser_automation_studio.v1.ExecutionsService/GetExecutionScreenshots", strings.NewReader(string(raw)))
	if err != nil {
		return Screenshot{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := d.Client
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(req)
	if err != nil {
		return Screenshot{}, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1024*1024+1))
	if err != nil {
		return Screenshot{}, err
	}
	if len(body) > 1024*1024 {
		return Screenshot{}, fmt.Errorf("BAS screenshot metadata exceeds limit")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Screenshot{}, fmt.Errorf("BAS screenshots returned HTTP %d", response.StatusCode)
	}
	var listing executionv1.GetScreenshotsResponse
	if err := protojson.Unmarshal(body, &listing); err != nil {
		return Screenshot{}, err
	}
	if listing.GetExecutionId() != op.ProducerID {
		return Screenshot{}, fmt.Errorf("screenshot listing belongs to another producer")
	}
	var found *executionv1.ExecutionScreenshot
	for _, shot := range listing.GetScreenshots() {
		if shot.GetScreenshot().GetArtifactId() == parts[2] {
			if found != nil {
				return Screenshot{}, fmt.Errorf("ambiguous screenshot identity")
			}
			found = shot
		}
	}
	if found == nil {
		return Screenshot{}, fmt.Errorf("captured screenshot is unavailable from its owner")
	}
	shot := found.GetScreenshot()
	if found.GetNodeId() != "target" || !screenshotScaleMatches(int(shot.GetWidth()), int(shot.GetHeight()), op.Request.Width, op.Request.Height) || (shot.GetContentType() != "image/png" && shot.GetContentType() != "image/jpeg") {
		return Screenshot{}, fmt.Errorf("screenshot differs from verified target")
	}
	owner, err := url.Parse(d.BASBaseURL)
	if err != nil || (owner.Scheme != "http" && owner.Scheme != "https") || owner.Host == "" || owner.User != nil {
		return Screenshot{}, fmt.Errorf("invalid browser owner URL")
	}
	location, err := url.Parse(shot.GetUrl())
	// Only the owner's declared, execution-scoped image route is permitted.
	// Reject alternate origins, query redirects, and ambiguous path encodings.
	if err != nil || location.IsAbs() || location.Host != "" || location.RawQuery != "" || location.Fragment != "" || location.RawPath != "" || path.Clean(location.Path) != location.Path || !strings.HasPrefix(location.Path, "/api/v1/screenshots/"+op.ProducerID+"/") {
		return Screenshot{}, fmt.Errorf("screenshot location is outside its producer route")
	}
	return Screenshot{Reference: artifact.Reference, URL: owner.ResolveReference(location).String(), Width: int(shot.GetWidth()), Height: int(shot.GetHeight()), ContentType: shot.GetContentType()}, nil
}

// Screenshot pixels can exceed CSS viewport dimensions on high-density devices.
// Region geometry and viewport identity are verified separately by Attach.
func screenshotScaleMatches(width, height, viewportWidth, viewportHeight int) bool {
	if viewportWidth <= 0 || viewportHeight <= 0 || width <= 0 || height <= 0 {
		return false
	}
	scaleX, scaleY := float64(width)/float64(viewportWidth), float64(height)/float64(viewportHeight)
	return scaleX >= 1 && scaleX <= 4 && scaleY >= 1 && scaleY <= 4 && math.Abs(float64(width)*float64(viewportHeight)-float64(height)*float64(viewportWidth)) <= float64(max(viewportWidth, viewportHeight))
}
