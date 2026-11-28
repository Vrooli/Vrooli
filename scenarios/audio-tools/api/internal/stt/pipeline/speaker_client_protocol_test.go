package pipeline

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSpeakerClientReadAndDeleteOperations(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "GET /ready":
			_, _ = io.WriteString(w, `{"status":"ready","model_loaded":true,"profile_store_ok":true,"temp_dir_ok":true}`)
		case "GET /v1/info":
			_, _ = io.WriteString(w, `{"backend":"sherpa","model":"embedder","device":"cpu","sample_rate":16000,"version":"1","embedding_dim":256}`)
		case "GET /v1/profiles":
			_, _ = io.WriteString(w, `{"count":1,"profiles":[{"id":"alice","display_name":"Alice","clip_count":1}]}`)
		case "GET /v1/profiles/alice":
			_, _ = io.WriteString(w, `{"id":"alice","display_name":"Alice","clips":[{"clip_id":"clip-1","label":"intro","embedding_dim":256}]}`)
		case "GET /v1/profiles/alice/clips":
			_, _ = io.WriteString(w, `{"profile_id":"alice","count":1,"clips":[{"clip_id":"clip-1","label":"intro"}]}`)
		case "DELETE /v1/profiles/alice/clips/clip-1":
			_, _ = io.WriteString(w, `{"profile_id":"alice","clip_id":"clip-1","deleted_profile":true,"clip_count":0}`)
		case "DELETE /v1/profiles/alice":
			_, _ = io.WriteString(w, `{}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := &SpeakerClient{BaseURL: server.URL + "/", Doer: server.Client()}

	ready, err := client.Ready(t.Context())
	if err != nil || ready.Status != "ready" || !ready.ModelLoaded || !ready.ProfileStoreOK || !ready.TempDirOK {
		t.Fatalf("Ready() = %+v, %v", ready, err)
	}
	info, err := client.Info(t.Context())
	if err != nil || info.Backend != "sherpa" || info.EmbeddingDim != 256 || info.SampleRate != 16000 {
		t.Fatalf("Info() = %+v, %v", info, err)
	}
	profiles, err := client.ListProfiles(t.Context())
	if err != nil || profiles.Count != 1 || profiles.Profiles[0].ID != "alice" {
		t.Fatalf("ListProfiles() = %+v, %v", profiles, err)
	}
	detail, err := client.GetProfile(t.Context(), "alice")
	if err != nil || detail.ID != "alice" || len(detail.Clips) != 1 || detail.Clips[0].ClipID != "clip-1" {
		t.Fatalf("GetProfile() = %+v, %v", detail, err)
	}
	clips, err := client.ListClips(t.Context(), "alice")
	if err != nil || clips.ProfileID != "alice" || clips.Count != 1 || clips.Clips[0].Label != "intro" {
		t.Fatalf("ListClips() = %+v, %v", clips, err)
	}
	deleted, err := client.DeleteClip(t.Context(), "alice", "clip-1")
	if err != nil || deleted.ProfileID != "alice" || deleted.ClipID != "clip-1" || !deleted.DeletedProfile {
		t.Fatalf("DeleteClip() = %+v, %v", deleted, err)
	}
	if err := client.DeleteProfile(t.Context(), "alice"); err != nil {
		t.Fatalf("DeleteProfile() error = %v", err)
	}
	want := []string{"GET /ready", "GET /v1/info", "GET /v1/profiles", "GET /v1/profiles/alice", "GET /v1/profiles/alice/clips", "DELETE /v1/profiles/alice/clips/clip-1", "DELETE /v1/profiles/alice"}
	if strings.Join(requests, "|") != strings.Join(want, "|") {
		t.Fatalf("requests = %v, want %v", requests, want)
	}
}

func TestSpeakerClientEnrollMultipartProtocol(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/profiles" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		mediaType, params, err := mimeParse(r.Header.Get("Content-Type"))
		if err != nil || mediaType != "multipart/form-data" {
			t.Errorf("Content-Type = %q, %v", r.Header.Get("Content-Type"), err)
		}
		mr := multipart.NewReader(r.Body, params["boundary"])
		fields := map[string]string{}
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Errorf("NextPart: %v", err)
				return
			}
			data, _ := io.ReadAll(part)
			if part.FormName() == "audio" {
				if part.FileName() != "take.wav" || string(data) != "wav bytes" {
					t.Errorf("audio part filename=%q body=%q", part.FileName(), data)
				}
			} else {
				fields[part.FormName()] = string(data)
			}
		}
		want := map[string]string{"profile_id": "alice", "display_name": "Alice A", "notes": "notes", "label": "greeting"}
		if len(fields) != len(want) {
			t.Errorf("fields = %#v", fields)
		}
		for key, value := range want {
			if fields[key] != value {
				t.Errorf("field %s = %q, want %q", key, fields[key], value)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"profile_id":"alice","clip_id":"c1","label":"greeting","clip_count":1,"embedding_dim":256,"self_consistency_warning":true}`)
	}))
	defer server.Close()
	client := &SpeakerClient{BaseURL: server.URL, Doer: server.Client()}
	out, err := client.Enroll(t.Context(), []byte("wav bytes"), "alice", "Alice A", "notes", "greeting", "take.wav")
	if err != nil || out.ProfileID != "alice" || out.ClipID != "c1" || !out.SelfConsistencyWarning || out.EmbeddingDim != 256 {
		t.Fatalf("Enroll() = %+v, %v", out, err)
	}
}

func mimeParse(value string) (string, map[string]string, error) {
	return mime.ParseMediaType(value)
}

func TestSpeakerClientExtractProtocolAndMismatch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/v1/extract" {
				t.Errorf("request = %s %s", r.Method, r.URL.Path)
			}
			mr, err := r.MultipartReader()
			if err != nil {
				t.Fatalf("MultipartReader: %v", err)
			}
			fields := map[string]string{}
			for {
				part, err := mr.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("NextPart: %v", err)
				}
				data, _ := io.ReadAll(part)
				if part.FormName() == "audio" {
					if part.FileName() != "segment.webm" || string(data) != "audio bytes" {
						t.Errorf("audio part filename=%q body=%q", part.FileName(), data)
					}
				} else {
					fields[part.FormName()] = string(data)
				}
			}
			if fields["profile_id"] != "alice" || fields["verify"] != "false" {
				t.Errorf("fields = %#v", fields)
			}
			w.Header().Set("X-Speaker-Score", "0.875")
			w.Header().Set("X-Speaker-Matched", "TRUE")
			w.Header().Set("X-Duration-Ms", "123.5")
			w.Header().Set("X-Audio-Seconds", "2.25")
			_, _ = w.Write([]byte("extracted wav"))
		}))
		defer server.Close()
		client := &SpeakerClient{BaseURL: server.URL, Doer: server.Client()}
		out, err := client.Extract(t.Context(), []byte("audio bytes"), "alice", false)
		if err != nil || string(out.Audio) != "extracted wav" || out.Score != .875 || !out.Matched || out.DurationMs != 123.5 || out.AudioSeconds != 2.25 {
			t.Fatalf("Extract() = %+v, %v", out, err)
		}
	})
	t.Run("model mismatch", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusConflict)
			_, _ = io.WriteString(w, `{"code":"speaker_model_mismatch","error":"re-enroll profile"}`)
		}))
		defer server.Close()
		client := &SpeakerClient{BaseURL: server.URL, Doer: server.Client()}
		_, err := client.Extract(t.Context(), []byte("audio"), "alice", true)
		var mismatch *SpeakerModelMismatchError
		if !errors.As(err, &mismatch) || mismatch.Message != "re-enroll profile" {
			t.Fatalf("error = %#v, want typed mismatch", err)
		}
	})
}

func TestSpeakerClientMapsJSONModelMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"code": "speaker_model_mismatch", "error": "wrong model"})
	}))
	defer server.Close()
	client := &SpeakerClient{BaseURL: server.URL, Doer: server.Client()}
	_, err := client.DeleteClip(t.Context(), "alice", "clip")
	var mismatch *SpeakerModelMismatchError
	if !errors.As(err, &mismatch) || mismatch.Message != "wrong model" {
		t.Fatalf("error = %#v, want typed mismatch", err)
	}
}

func TestSpeakerClientBoundsSuccessfulResponses(t *testing.T) {
	t.Run("metadata JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"status":"`+strings.Repeat("x", maxSpeakerJSONResponseBytes)+`"}`)
		}))
		defer server.Close()
		client := &SpeakerClient{BaseURL: server.URL, Doer: server.Client()}
		if _, err := client.Ready(t.Context()); err == nil || !strings.Contains(err.Error(), "exceeds") {
			t.Fatalf("Ready() error = %v, want bounded-response error", err)
		}
	})
	t.Run("extracted audio", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, strings.Repeat("x", maxSpeakerAudioResponseBytes+1))
		}))
		defer server.Close()
		client := &SpeakerClient{BaseURL: server.URL, Doer: server.Client()}
		if _, err := client.Extract(t.Context(), []byte("audio"), "alice", false); err == nil || !strings.Contains(err.Error(), "exceeds") {
			t.Fatalf("Extract() error = %v, want bounded-response error", err)
		}
	})
}
