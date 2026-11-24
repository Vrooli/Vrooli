package componenttests

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
	captureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture/captureconnect"
)

type CaptureTerminalError struct{ Err error }

func (e CaptureTerminalError) Error() string { return e.Err.Error() }
func (e CaptureTerminalError) Unwrap() error { return e.Err }

type CaptureUnresolvedError struct{ Err error }

func (e CaptureUnresolvedError) Error() string { return e.Err.Error() }
func (e CaptureUnresolvedError) Unwrap() error { return e.Err }

// CaptureSurface submits one tokenized BAS page capture and reconciles a lost
// response through BAS's request-token lookup before deciding whether retry is
// safe. It requires both screenshot and accessibility evidence references.
func (e BASCaptureExecutor) CaptureSurface(ctx context.Context, request *capturev1.CaptureRequest) (*capturev1.CaptureResponse, error) {
	if request == nil || strings.TrimSpace(request.GetRequestToken()) == "" {
		return nil, errors.New("BAS surface capture requires a stable request token")
	}
	baseURL, err := e.basBaseURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve BAS capture service: %w", err)
	}
	client := captureconnect.NewCaptureServiceClient(e.client(), baseURL)
	response, captureErr := client.Capture(ctx, connect.NewRequest(request))
	if captureErr == nil {
		return requireSurfaceArtifacts(response.Msg)
	}
	lookup, lookupErr := client.GetCaptureByRequestToken(ctx, connect.NewRequest(&capturev1.GetCaptureByRequestTokenRequest{RequestToken: request.GetRequestToken()}))
	if lookupErr == nil {
		switch lookup.Msg.GetState() {
		case capturev1.GetCaptureByRequestTokenResponse_STATE_COMPLETED:
			return requireSurfaceArtifacts(lookup.Msg.GetCapture())
		case capturev1.GetCaptureByRequestTokenResponse_STATE_FAILED:
			return nil, CaptureTerminalError{Err: fmt.Errorf("BAS capture failed: %s", lookup.Msg.GetError())}
		case capturev1.GetCaptureByRequestTokenResponse_STATE_IN_PROGRESS:
			// Same token and request resumes or joins the preallocated execution.
			response, retryErr := client.Capture(ctx, connect.NewRequest(request))
			if retryErr == nil {
				return requireSurfaceArtifacts(response.Msg)
			}
			return nil, CaptureUnresolvedError{Err: fmt.Errorf("BAS capture remains unresolved after retry (execution %s): %w", lookup.Msg.GetExecutionId(), retryErr)}
		}
	}
	return nil, CaptureUnresolvedError{Err: fmt.Errorf("BAS capture failed and request-token lookup was unavailable: %w", captureErr)}
}

func requireSurfaceArtifacts(response *capturev1.CaptureResponse) (*capturev1.CaptureResponse, error) {
	if response == nil || strings.TrimSpace(response.GetExecutionId()) == "" {
		return nil, CaptureTerminalError{Err: errors.New("BAS completed capture without an execution identity")}
	}
	screenshot, accessibility := false, false
	for _, artifact := range response.GetArtifacts() {
		if artifact == nil || strings.TrimSpace(artifact.GetReference()) == "" {
			continue
		}
		switch artifact.GetType() {
		case capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT:
			screenshot = true
		case capturev1.CaptureType_CAPTURE_TYPE_ACCESSIBILITY:
			accessibility = true
		}
	}
	if !screenshot || !accessibility {
		return nil, CaptureTerminalError{Err: errors.New("BAS capture completed without both screenshot and accessibility artifact references")}
	}
	return response, nil
}
