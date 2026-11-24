package journeys

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHTTPDeviceClientLeaseLifecycleKeepsDeviceControlIdentity(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.EscapedPath() {
		case "/api/v1/sessions/acquire":
			require.Equal(t, http.MethodPost, request.Method)
			var payload map[string]any
			require.NoError(t, json.NewDecoder(request.Body).Decode(&payload))
			require.Equal(t, "android-phone", payload["device_id"])
			require.Equal(t, "journey-owner", payload["actor"])
			require.Equal(t, float64(90), payload["ttl_seconds"])
			_ = json.NewEncoder(writer).Encode(map[string]any{"session": map[string]any{
				"id": "lease/one", "lease_token": "lease-secret", "expires_at": time.Now().Add(time.Minute),
			}})
		case "/api/v1/sessions/lease/one/validate":
			t.Fatal("lease ID containing a slash must remain one escaped path segment")
		case "/api/v1/sessions/lease%2Fone/validate":
			require.Equal(t, "/api/v1/sessions/lease%2Fone/validate", request.URL.EscapedPath())
			var payload map[string]string
			require.NoError(t, json.NewDecoder(request.Body).Decode(&payload))
			require.Equal(t, map[string]string{"device_id": "android-phone", "lease_token": "lease-secret"}, payload)
			writer.WriteHeader(http.StatusNoContent)
		case "/api/v1/sessions/lease%2Fone/release":
			require.Equal(t, int64(0), request.ContentLength)
			writer.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected Device Control request: %s %s", request.Method, request.URL.RequestURI())
		}
	}))
	defer server.Close()

	client := &HTTPDeviceClient{BaseURL: server.URL, Actor: "journey-owner"}
	lease, err := client.Acquire(context.Background(), "android-phone", "", 90*time.Second)
	require.NoError(t, err)
	require.Equal(t, Lease{ID: "lease/one", DeviceID: "android-phone", Token: "lease-secret"}, Lease{ID: lease.ID, DeviceID: lease.DeviceID, Token: lease.Token})
	require.NoError(t, client.ValidateLease(context.Background(), lease))
	require.NoError(t, client.Release(context.Background(), lease))
	require.Equal(t, 3, requests)
}

func TestHTTPDeviceClientDoesNotReplayUncertainRecordingStopAndCanInspectOwner(t *testing.T) {
	stopRequests, inspectRequests := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/api/v1/recordings/start":
			var payload map[string]string
			require.NoError(t, json.NewDecoder(request.Body).Decode(&payload))
			require.Equal(t, "android-phone", payload["device_id"])
			require.Equal(t, "journey-owner", payload["actor"])
			require.Equal(t, "lease-secret", payload["lease_token"])
			require.Equal(t, "usb", payload["transport"], "offline-transition recordings must stay reachable after Wi-Fi is disabled")
			_, _ = fmt.Fprint(writer, `{"handle":{"id":"recording-17"}}`)
		case "/api/v1/recordings/stop":
			stopRequests++
			var payload map[string]string
			require.NoError(t, json.NewDecoder(request.Body).Decode(&payload))
			require.Equal(t, map[string]string{"device_id": "android-phone", "actor": "journey-owner", "lease_token": "lease-secret", "handle_id": "recording-17"}, payload)
			writer.WriteHeader(http.StatusConflict)
			_, _ = fmt.Fprint(writer, `{"status":"failed","code":"recording_stop_uncertain","message":"response lost after native stop","recovered_recording":{"reference":{"id":"evidence-recovered"}}}`)
		case "/api/v1/recordings/inspect":
			inspectRequests++
			var payload map[string]string
			require.NoError(t, json.NewDecoder(request.Body).Decode(&payload))
			require.Equal(t, "recording-17", payload["handle_id"])
			require.Equal(t, "lease-secret", payload["lease_token"])
			_ = json.NewEncoder(writer).Encode(RecordingInspection{
				HandleID: "recording-17", OwnerState: "uncertain", ProcessState: "exited",
				ArtifactAvailable: true, RecoveredEvidenceID: "evidence-recovered",
				Reason: "native stop completed but restoration failed",
			})
		default:
			t.Fatalf("unexpected Device Control request: %s %s", request.Method, request.URL.RequestURI())
		}
	}))
	defer server.Close()

	client := &HTTPDeviceClient{BaseURL: server.URL, Actor: "journey-owner", DeviceTransport: "wireless"}
	lease := Lease{ID: "lease-one", DeviceID: "android-phone", Token: "lease-secret"}
	require.NoError(t, client.StartChapterRecording(context.Background(), lease, "offline_transition"))
	_, err := client.StopChapterRecording(context.Background(), lease, "offline_transition")
	require.ErrorContains(t, err, "recordings/stop returned 409 Conflict")
	_, err = client.StopChapterRecording(context.Background(), lease, "offline_transition")
	require.ErrorContains(t, err, "outcome is uncertain")
	require.Equal(t, 1, stopRequests, "an uncertain stop must not be replayed")

	inspection, err := client.InspectChapterRecording(context.Background(), lease, "offline_transition")
	require.NoError(t, err)
	require.Equal(t, "uncertain", inspection.OwnerState)
	require.Equal(t, "exited", inspection.ProcessState)
	require.True(t, inspection.ArtifactAvailable)
	require.Equal(t, "evidence-recovered", inspection.RecoveredEvidenceID)
	require.Equal(t, 1, inspectRequests)
}

func TestHTTPDeviceClientSuccessfulRecordingStopRetiresOnlyItsHandle(t *testing.T) {
	var stopHandles []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/api/v1/recordings/start":
			var payload map[string]string
			require.NoError(t, json.NewDecoder(request.Body).Decode(&payload))
			if payload["device_id"] == "device-a" {
				_, _ = fmt.Fprint(writer, `{"handle":{"id":"handle-a"}}`)
			} else {
				_, _ = fmt.Fprint(writer, `{"handle":{"id":"handle-b"}}`)
			}
		case "/api/v1/recordings/stop":
			var payload map[string]string
			require.NoError(t, json.NewDecoder(request.Body).Decode(&payload))
			stopHandles = append(stopHandles, payload["handle_id"])
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"reference":       map[string]any{"id": "evidence-" + payload["handle_id"], "kind": "video", "checksum": "abc", "redaction_verified": true},
				"start_offset_ms": 12, "end_offset_ms": 34,
			})
		default:
			t.Fatalf("unexpected Device Control request: %s %s", request.Method, request.URL.RequestURI())
		}
	}))
	defer server.Close()

	client := &HTTPDeviceClient{BaseURL: server.URL}
	leaseA := Lease{ID: "lease-a", DeviceID: "device-a", Token: "token-a"}
	leaseB := Lease{ID: "lease-b", DeviceID: "device-b", Token: "token-b"}
	require.NoError(t, client.StartChapterRecording(context.Background(), leaseA, "launch"))
	require.NoError(t, client.StartChapterRecording(context.Background(), leaseB, "launch"))
	artifact, err := client.StopChapterRecording(context.Background(), leaseA, "launch")
	require.NoError(t, err)
	require.Equal(t, "evidence-handle-a", artifact.Reference.ID)
	require.Equal(t, "sha256:abc", artifact.Reference.Checksum)
	require.True(t, artifact.Reference.Redacted)
	require.True(t, artifact.HasOffsets)
	require.EqualValues(t, 12, artifact.StartMs)
	require.EqualValues(t, 34, artifact.EndMs)
	_, err = client.StopChapterRecording(context.Background(), leaseA, "launch")
	require.ErrorContains(t, err, "no active recording")
	artifactB, err := client.StopChapterRecording(context.Background(), leaseB, "launch")
	require.NoError(t, err)
	require.Equal(t, "evidence-handle-b", artifactB.Reference.ID)
	require.Equal(t, []string{"handle-a", "handle-b"}, stopHandles)
}

func TestHTTPDeviceClientSerializesConcurrentStopsForOneChapter(t *testing.T) {
	stopEntered := make(chan struct{})
	finishStop := make(chan struct{})
	var stopRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/api/v1/recordings/start":
			_, _ = fmt.Fprint(writer, `{"handle":{"id":"shared-handle"}}`)
		case "/api/v1/recordings/stop":
			stopRequests.Add(1)
			close(stopEntered)
			<-finishStop
			_, _ = fmt.Fprint(writer, `{"reference":{"id":"shared-evidence","kind":"video","checksum":"abc","redaction_verified":true},"start_offset_ms":0,"end_offset_ms":10}`)
		default:
			t.Fatalf("unexpected Device Control request: %s %s", request.Method, request.URL.RequestURI())
		}
	}))
	defer server.Close()

	client := &HTTPDeviceClient{BaseURL: server.URL}
	lease := Lease{ID: "lease-one", DeviceID: "android-phone", Token: "lease-secret"}
	require.NoError(t, client.StartChapterRecording(context.Background(), lease, "launch"))

	firstResult := make(chan error, 1)
	go func() {
		_, err := client.StopChapterRecording(context.Background(), lease, "launch")
		firstResult <- err
	}()
	<-stopEntered
	_, secondErr := client.StopChapterRecording(context.Background(), lease, "launch")
	require.ErrorContains(t, secondErr, "stop is already in progress")
	close(finishStop)
	require.NoError(t, <-firstResult)
	require.EqualValues(t, 1, stopRequests.Load(), "one owner handle must have one concurrent stop request")
}
