package designcritique

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"react-component-library/internal/designcapture"
)

type captureFixtureRepository struct{ op designcapture.Operation }

func (r captureFixtureRepository) Get(context.Context, string) (designcapture.Operation, error) {
	return r.op, nil
}
func (captureFixtureRepository) Create(context.Context, string, designcapture.Request) (designcapture.Operation, error) {
	return designcapture.Operation{}, errors.New("unexpected write")
}
func (captureFixtureRepository) Transition(context.Context, string, int64, designcapture.State, string, []designcapture.Artifact, string) (designcapture.Operation, error) {
	return designcapture.Operation{}, errors.New("unexpected write")
}

type imageResolver struct{ url string }

func (r imageResolver) ResolveScreenshot(context.Context, designcapture.Operation, designcapture.Artifact) (designcapture.Screenshot, error) {
	return designcapture.Screenshot{URL: r.url, Width: 390, Height: 844, ContentType: "image/png"}, nil
}
func TestCaptureEvidenceRequiresAvailableMatchingImage(t *testing.T) {
	var encoded bytes.Buffer
	require.NoError(t, png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 390, 844))))
	for _, name := range []string{"valid", "explicit alternate render", "alternate wrong revision", "alternate wrong design", "alternate wrong render", "wrong revision", "wrong render", "wrong viewport", "wrong region", "wrong artifact", "not completed", "image gone", "corrupt image", "redirect"} {
		t.Run(name, func(t *testing.T) {
			review := reviewFixture()
			e := review.Ratings[0].Evidence[0]
			op := designcapture.Operation{State: designcapture.Completed, Request: designcapture.Request{Target: designcapture.Target{Scenario: review.Target.Scenario, DesignID: review.Target.DesignID, Revision: review.Target.Revision, RenderHash: review.Target.RenderHash}, Width: 390, Height: 844}, Artifacts: []designcapture.Artifact{{Kind: "screenshot", Reference: e.Artifact}}}
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				switch name {
				case "image gone":
					w.WriteHeader(404)
				case "corrupt image":
					_, _ = w.Write([]byte("bad PNG"))
				case "redirect":
					http.Redirect(w, r, "/elsewhere", 302)
				default:
					_, _ = w.Write(encoded.Bytes())
				}
			}))
			defer server.Close()

			if strings.HasPrefix(name, "alternate") || name == "explicit alternate render" {
				e.RenderHash = strings.Repeat("c", 64)
				op.Request.Target.RenderHash = e.RenderHash
			}
			switch name {
			case "alternate wrong revision":
				op.Request.Target.Revision = strings.Repeat("d", 64)
			case "alternate wrong design":
				op.Request.Target.DesignID = "other"
			case "alternate wrong render":
				op.Request.Target.RenderHash = strings.Repeat("d", 64)
			case "wrong revision":
				op.Request.Target.Revision = "other"
			case "wrong render":
				op.Request.Target.RenderHash = "other"
			case "wrong viewport":
				e.Width = 1440
			case "wrong region":
				e.Region = "missing"
			case "wrong artifact":
				e.Artifact = "other"
			case "not completed":
				op.State = designcapture.Running
			}
			verifier := CaptureEvidenceVerifier{Captures: captureFixtureRepository{op}, Screenshots: imageResolver{server.URL}}
			err := verifier.Verify(context.Background(), review.Target, e)
			if name == "valid" || name == "explicit alternate render" {
				require.NoError(t, err)
				require.Equal(t, 1, calls)
			} else {
				require.Error(t, err)
			}
			if name == "redirect" {
				require.Equal(t, 1, calls, "redirect target must not be fetched")
			}
		})
	}
}
