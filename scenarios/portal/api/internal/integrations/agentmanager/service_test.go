package agentmanager

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

const ownedRunID = "9b5a0610-88ec-4e4a-9964-626a8b67bdf5"

func TestStopRequiresMatchingTerminalOwnerState(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		code       int
		confirmed  bool
	}{
		{"cancelled", fmt.Sprintf(`{"status":"stopped","run":{"id":%q,"status":"RUN_STATUS_CANCELLED"}}`, ownedRunID), 200, true},
		{"completed-before-stop", fmt.Sprintf(`{"run":{"id":%q,"status":"RUN_STATUS_COMPLETE"}}`, ownedRunID), 200, true},
		{"still-running", fmt.Sprintf(`{"status":"stopped","run":{"id":%q,"status":"RUN_STATUS_RUNNING"}}`, ownedRunID), 200, false},
		{"wrong-run", `{"run":{"id":"other","status":"RUN_STATUS_CANCELLED"}}`, 200, false},
		{"missing-run", `{"status":"stopped"}`, 200, false},
		{"malformed", `{`, 200, false},
		{"owner-refusal", `{"error":"denied"}`, 403, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				require.Equal(t, http.MethodPost, r.Method)
				require.Equal(t, "/api/v1/runs/"+ownedRunID+"/stop", r.URL.Path)
				w.WriteHeader(tc.code)
				_, _ = io.WriteString(w, tc.body)
			}))
			defer server.Close()
			state, err := NewHTTPClient(server.URL, server.Client()).Stop(context.Background(), ownedRunID)
			if tc.confirmed {
				require.NoError(t, err)
				require.True(t, state.Terminal)
				require.Equal(t, ownedRunID, state.RunID)
			} else {
				require.ErrorIs(t, err, ErrStopUnconfirmed)
			}
			require.Equal(t, 1, calls)
		})
	}
}

func TestLostStopReplyReconcilesWithReadWithoutRepeatingStop(t *testing.T) {
	stops, reads := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			stops++
			_, _ = io.WriteString(w, `{"run":`)
		case http.MethodGet:
			reads++
			require.Equal(t, "/api/v1/runs/"+ownedRunID, r.URL.Path)
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.Empty(t, body)
			_, _ = fmt.Fprintf(w, `{"run":{"id":%q,"status":"RUN_STATUS_CANCELLED"}}`, ownedRunID)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()
	client := NewHTTPClient(server.URL, server.Client())
	_, err := client.Stop(context.Background(), ownedRunID)
	require.True(t, errors.Is(err, ErrStopUnconfirmed))
	state, err := client.Run(context.Background(), ownedRunID)
	require.NoError(t, err)
	require.True(t, state.Terminal)
	require.Equal(t, 1, stops)
	require.Equal(t, 1, reads)
}

func TestRunOperationsRefuseNonCanonicalIdentityBeforeNetwork(t *testing.T) {
	client := NewHTTPClient("http://127.0.0.1:1", nil)
	for _, id := range []string{"", "../other/stop", "00000000-0000-0000-0000-000000000000"} {
		_, err := client.Run(context.Background(), id)
		require.ErrorContains(t, err, "UUID required")
	}
}

func TestAdmissionLookupRequiresUniqueCompleteExactMatch(t *testing.T) {
	tag := "portal-admission-" + ownedRunID
	for _, tc := range []struct {
		name, body string
		ok         bool
	}{
		{"unique", fmt.Sprintf(`{"runs":[{"id":%q,"task_id":%q,"tag":%q}],"total":1}`, ownedRunID, ownedRunID, tag), true},
		{"absent", `{"runs":[],"total":0}`, false},
		{"ambiguous", fmt.Sprintf(`{"runs":[{"id":%q,"task_id":%q,"tag":%q}],"total":2}`, ownedRunID, ownedRunID, tag), false},
		{"incomplete", fmt.Sprintf(`{"runs":[{"id":%q,"task_id":%q,"tag":%q}],"total":1,"has_more":true}`, ownedRunID, ownedRunID, tag), false},
		{"prefix-only", fmt.Sprintf(`{"runs":[{"id":%q,"task_id":%q,"tag":%q}],"total":1}`, ownedRunID, ownedRunID, tag+"-other"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodGet, r.Method)
				require.Equal(t, tag, r.URL.Query().Get("tag_prefix"))
				_, _ = io.WriteString(w, tc.body)
			}))
			defer server.Close()
			session, err := NewHTTPClient(server.URL, server.Client()).FindAdmission(context.Background(), ownedRunID)
			if tc.ok {
				require.NoError(t, err)
				require.Equal(t, ownedRunID, session.RunID)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestUploadAttachmentUsesBoundedMultipartPayload(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/v1/attachments/upload", r.URL.Path)
		require.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")
		require.NoError(t, r.ParseMultipartForm(1024))
		file, header, err := r.FormFile("file")
		require.NoError(t, err)
		defer file.Close()
		require.Equal(t, "portal-context.png", header.Filename)
		body, err := io.ReadAll(file)
		require.NoError(t, err)
		require.Equal(t, []byte("rendered"), body)
		_, _ = io.WriteString(w, `{"id":"attachment-1"}`)
	}))
	defer server.Close()
	id, err := NewHTTPClient(server.URL, server.Client()).UploadAttachment(context.Background(), []byte("rendered"), "portal-context.png", "image/png")
	require.NoError(t, err)
	require.Equal(t, "attachment-1", id)
	require.Equal(t, 1, calls)
}
