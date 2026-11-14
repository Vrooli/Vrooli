package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func songFixture(t *testing.T) (string, []byte, map[string]any, []byte) {
	t.Helper()
	seed := uint32(0)
	score := "X:1\nM:4/4\nK:C\nC4 D4 E4 F4|\n"
	request := songComposeRequest{
		Profile: "yue2-q8", Prompt: "Original groove", Lyrics: "[Chorus]\nKeep the receipt",
		Duration: 1, Seed: &seed, Steps: 32, Guidance: 1, Revision: yueRevision, Timeout: 10, Score: &score,
	}
	raw, _ := json.Marshal(request)
	path := filepath.Join(t.TempDir(), "request.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	wav := make([]byte, 44+4800*4)
	copy(wav, "RIFF")
	binary.LittleEndian.PutUint32(wav[4:], uint32(len(wav)-8))
	copy(wav[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(wav[16:], 16)
	binary.LittleEndian.PutUint16(wav[20:], 1)
	binary.LittleEndian.PutUint16(wav[22:], 2)
	binary.LittleEndian.PutUint32(wav[24:], 48000)
	binary.LittleEndian.PutUint32(wav[28:], 192000)
	binary.LittleEndian.PutUint16(wav[32:], 4)
	binary.LittleEndian.PutUint16(wav[34:], 16)
	copy(wav[36:], "data")
	binary.LittleEndian.PutUint32(wav[40:], 4800*4)
	artifact := func(text, source string) map[string]any {
		return map[string]any{"text": text, "bytes": len(text), "sha256": fmt.Sprintf("%x", sha256.Sum256([]byte(text))), "source": source, "truncated": false}
	}
	receipt := map[string]any{
		"schema_version": 1, "status": "completed", "operation_id": strings.Repeat("1", 32),
		"request_sha256": fmt.Sprintf("%x", sha256.Sum256(raw)), "effective_request": request,
		"artifact_sha256": strings.Repeat("a", 64), "catalog_sha256": strings.Repeat("b", 64),
		"model": map[string]any{"id": yueModel, "revision": yueRevision, "source_revision": yueSource},
		"worker": map[string]any{
			"seed": seed, "model_id": yueModel, "revision": yueRevision, "weight_files_verified": 8,
			"native_child_reaped": true, "native_runtime": map[string]any{"backend": "cuda", "version": "0.9.3"},
			"sampling":              map[string]any{"steps": 32, "guidance": 1, "score_source": "provided"},
			"composition_artifacts": map[string]any{"score": artifact(score, "provided"), "semantic": artifact("[1,2,3]", "generated")},
		},
		"output": map[string]any{
			"sha256": fmt.Sprintf("%x", sha256.Sum256(wav)), "bytes": len(wav), "frames": 4800,
			"sample_rate": 48000, "channels": 2, "duration_seconds": 0.1,
		},
	}
	return path, raw, receipt, wav
}

func fixtureHealth() map[string]any {
	return map[string]any{
		"status": "ready", "protocol": "song-v1", "artifact_sha256": strings.Repeat("a", 64),
		"catalog_sha256": strings.Repeat("b", 64), "profiles": map[string]any{"yue2-q8": map[string]any{
			"available": true, "model_id": yueModel, "revision": yueRevision,
		}},
	}
}

func fixtureAttribution(req *http.Request) error {
	req.Header.Set("X-Fixture-Attribution", "verified")
	return nil
}

func TestSongComposeExportsExactScoreReceiptAndAudioThroughAttributedOwner(t *testing.T) {
	path, raw, receipt, wav := songFixture(t)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("X-Fixture-Attribution") != "verified" {
			t.Error("request lost attribution")
		}
		switch r.URL.Path {
		case "/health":
			json.NewEncoder(w).Encode(fixtureHealth())
		case "/v1/compose":
			var got bytes.Buffer
			got.ReadFrom(r.Body)
			if !bytes.Equal(got.Bytes(), raw) {
				t.Error("authored input bytes changed before owner")
			}
			json.NewEncoder(w).Encode(receipt)
		case "/v1/operations/" + strings.Repeat("1", 32) + "/audio":
			w.Write(wav)
		default:
			t.Errorf("unexpected owner path %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	out := filepath.Join(t.TempDir(), "new")
	result, err := songComposeToDirectory(t.Context(), server.Client(), fixtureAttribution, server.URL, path, out, strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	if result["status"] != "completed" || calls.Load() != 3 {
		t.Fatalf("result=%v calls=%d", result, calls.Load())
	}
	actual, _ := os.ReadFile(filepath.Join(out, "audio.wav"))
	if !bytes.Equal(actual, wav) {
		t.Fatal("exported audio changed")
	}
	actual, _ = os.ReadFile(filepath.Join(out, "request.json"))
	if !bytes.Equal(actual, raw) {
		t.Fatal("exported request changed")
	}
	for _, name := range []string{"receipt.json", "score.abc", "semantic.json"} {
		if info, err := os.Stat(filepath.Join(out, name)); err != nil || info.Size() == 0 {
			t.Fatalf("missing retained witness %s: %v", name, err)
		}
	}
}

func TestSongComposeRejectsForgedRuntimeSamplingScoreAndPCM(t *testing.T) {
	for _, change := range []string{"seed", "artifact", "score-source", "score-text", "frames", "pcm"} {
		t.Run(change, func(t *testing.T) {
			path, _, receipt, wav := songFixture(t)
			worker := receipt["worker"].(map[string]any)
			switch change {
			case "seed":
				worker["seed"] = 1
			case "artifact":
				receipt["artifact_sha256"] = strings.Repeat("c", 64)
			case "score-source":
				worker["sampling"].(map[string]any)["score_source"] = "generated"
			case "score-text":
				worker["composition_artifacts"].(map[string]any)["score"].(map[string]any)["text"] = "different original"
			case "frames":
				receipt["output"].(map[string]any)["frames"] = 4801
			case "pcm":
				binary.LittleEndian.PutUint16(wav[34:], 24)
				receipt["output"].(map[string]any)["sha256"] = fmt.Sprintf("%x", sha256.Sum256(wav))
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/health" {
					json.NewEncoder(w).Encode(fixtureHealth())
				} else if r.Method == http.MethodPost {
					json.NewEncoder(w).Encode(receipt)
				} else {
					w.Write(wav)
				}
			}))
			defer server.Close()
			out := filepath.Join(t.TempDir(), "new")
			if _, err := songComposeToDirectory(t.Context(), server.Client(), fixtureAttribution, server.URL, path, out, strings.Repeat("a", 64)); err == nil {
				t.Fatal("forged result accepted")
			}
			if _, err := os.Lstat(out); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("failed export published an output")
			}
		})
	}
}

func TestSongComposeRefusesBeforeGenerationAndNeverReplacesExistingOutput(t *testing.T) {
	path, _, receipt, wav := songFixture(t)
	var posts atomic.Int32
	out := filepath.Join(t.TempDir(), "new")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/health":
			json.NewEncoder(w).Encode(fixtureHealth())
		case r.Method == http.MethodPost:
			posts.Add(1)
			os.Mkdir(out, 0o700) // concurrent creator wins the final name
			os.WriteFile(filepath.Join(out, "keep"), []byte("prior"), 0o600)
			json.NewEncoder(w).Encode(receipt)
		default:
			w.Write(wav)
		}
	}))
	defer server.Close()
	if _, err := songComposeToDirectory(t.Context(), server.Client(), fixtureAttribution, server.URL, path, out, strings.Repeat("c", 64)); err == nil || posts.Load() != 0 {
		t.Fatal("unadopted artifact reached generation")
	}
	refused := func(*http.Request) error { return errors.New("canonical authority refused") }
	if _, err := songComposeToDirectory(t.Context(), server.Client(), refused, server.URL, path, out, strings.Repeat("a", 64)); err == nil || posts.Load() != 0 {
		t.Fatal("authority error reached generation")
	}
	if _, err := songComposeToDirectory(t.Context(), server.Client(), fixtureAttribution, server.URL, path, out, strings.Repeat("a", 64)); err == nil {
		t.Fatal("concurrent output replaced")
	}
	keep, _ := os.ReadFile(filepath.Join(out, "keep"))
	if string(keep) != "prior" || posts.Load() != 1 {
		t.Fatal("existing output damaged")
	}
	if _, err := songComposeToDirectory(t.Context(), server.Client(), fixtureAttribution, server.URL, path, out, strings.Repeat("a", 64)); err == nil || posts.Load() != 1 {
		t.Fatal("existing output caused another generation")
	}
}

func TestSongComposeCancellationReachesOwnerWithoutRetry(t *testing.T) {
	path, _, _, _ := songFixture(t)
	entered, drained := make(chan struct{}), make(chan struct{})
	var posts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			json.NewEncoder(w).Encode(fixtureHealth())
			return
		}
		posts.Add(1)
		io.Copy(io.Discard, r.Body)
		close(entered)
		<-r.Context().Done()
		close(drained)
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	out := filepath.Join(t.TempDir(), "new")
	done := make(chan error, 1)
	go func() {
		_, err := songComposeToDirectory(ctx, server.Client(), fixtureAttribution, server.URL, path,
			out, strings.Repeat("a", 64))
		done <- err
	}()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("fixture owner was not reached")
	}
	cancel()
	select {
	case err := <-done:
		if err == nil || !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation lost: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("caller did not cancel")
	}
	select {
	case <-drained:
	case <-time.After(3 * time.Second):
		t.Fatal("owner transport context remained live")
	}
	if posts.Load() != 1 {
		t.Fatal("cancellation retried generation")
	}
}

func TestSongComposeRejectsUnboundedOrPathConditioning(t *testing.T) {
	_, raw, _, _ := songFixture(t)
	var value map[string]any
	json.Unmarshal(raw, &value)
	for _, invalid := range []any{"/etc/passwd", "https://example.invalid/score.abc", "X:1\nK:C\n" + strings.Repeat("\U0001f3b5", 4096), nil} {
		value["score"] = invalid
		data, _ := json.Marshal(value)
		if _, err := decodeSongCompose(data); err == nil {
			t.Fatal("invalid score accepted")
		}
	}
	delete(value, "score")
	value["score_file"] = "/etc/passwd"
	data, _ := json.Marshal(value)
	if _, err := decodeSongCompose(data); err == nil {
		t.Fatal("native file path option accepted")
	}
	path := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink("/etc/passwd", path); err != nil {
		t.Fatal(err)
	}
	if file, err := openSongRegular(path); err == nil {
		file.Close()
		t.Fatal("symlink input accepted")
	}
}
