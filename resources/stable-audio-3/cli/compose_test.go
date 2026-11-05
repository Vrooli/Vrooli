package main

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
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

	"golang.org/x/sys/unix"
)

func testWAV() []byte {
	b := make([]byte, 44+44100*6)
	copy(b, "RIFF")
	binary.LittleEndian.PutUint32(b[4:8], uint32(len(b)-8))
	copy(b[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(b[16:20], 16)
	binary.LittleEndian.PutUint16(b[20:22], 1)
	binary.LittleEndian.PutUint16(b[22:24], 2)
	binary.LittleEndian.PutUint32(b[24:28], 44100)
	binary.LittleEndian.PutUint32(b[28:32], 264600)
	binary.LittleEndian.PutUint16(b[32:34], 6)
	binary.LittleEndian.PutUint16(b[34:36], 24)
	copy(b[36:], "data")
	binary.LittleEndian.PutUint32(b[40:44], 44100*6)
	return b
}

func fakeReceipt(request, wav []byte) map[string]any {
	return map[string]any{
		"schema_version": 1, "status": "completed", "operation_id": strings.Repeat("a", 32),
		"request_sha256": fmt.Sprintf("%x", sha256.Sum256(request)),
		"model":          map[string]any{"model_id": modelID, "model_revision": modelRevision, "source_revision": sourceRevision},
		"sampling":       map[string]any{"seed": 0, "steps": 8, "cfg_scale": 1},
		"runtime":        map[string]any{"torch": "2.7.1+cu126", "cuda": "12.6", "transformers": "5.10.4", "flash_attn": "2.8.3.post2", "cxx11_abi": true, "declared_composed_sha256": strings.Repeat("d", 64), "source_files": map[string]string{"server.py": strings.Repeat("e", 64), "loader.py": strings.Repeat("f", 64), "requirements.lock": strings.Repeat("a", 64)}, "torch_allocator_ceiling_bytes": 8 << 30, "broker_claim_bytes": 9 << 30},
		"output":         map[string]any{"name": "audio.wav", "sha256": fmt.Sprintf("%x", sha256.Sum256(wav)), "bytes": len(wav), "frames": 44100, "sample_rate": 44100, "channels": 2, "format": "WAV", "subtype": "PCM_24"},
	}
}

func TestComposePublishesReceiptBoundWAVAndPreservesExistingOutput(t *testing.T) {
	dir := t.TempDir()
	request := filepath.Join(dir, "request.json")
	raw := []byte(`{"prompt":"original instrumental club cue","duration":1,"seed":0}`)
	if err := os.WriteFile(request, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	wav := testWAV()
	var calls atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method == "POST" {
			body, _ := io.ReadAll(r.Body)
			if string(body) != string(raw) {
				t.Error("request bytes were changed")
			}
			json.NewEncoder(w).Encode(fakeReceipt(raw, wav))
		} else {
			w.Write(wav)
		}
	}))
	defer api.Close()
	out := filepath.Join(dir, "take")
	result, err := composeToDirectory(context.Background(), api.Client(), api.URL, request, out)
	if err != nil || result["status"] != "completed" {
		t.Fatalf("compose = %v %v", result, err)
	}
	retained, err := os.ReadFile(filepath.Join(out, "audio.wav"))
	if err != nil || string(retained) != string(wav) {
		t.Fatal("source WAV changed during export", err)
	}
	if _, err := os.Stat(filepath.Join(out, "generation-receipt.json")); err != nil {
		t.Fatal(err)
	}
	_, err = composeToDirectory(context.Background(), api.Client(), api.URL, request, out)
	if err == nil || calls.Load() != 2 {
		t.Fatal("existing output was not refused before network work", err, calls.Load())
	}
}

func TestComposeRejectsWrongReceiptOrAudioWithoutPublishing(t *testing.T) {
	for _, test := range []string{"request-hash", "model", "seed", "missing-zero-seed", "missing-cfg", "runtime", "missing-lock-identity", "audio-hash", "encoded-format", "destination-race"} {
		t.Run(test, func(t *testing.T) {
			dir := t.TempDir()
			raw := []byte(`{"prompt":"original cue","duration":1,"seed":0}`)
			path, out := filepath.Join(dir, "request.json"), filepath.Join(dir, "out")
			os.WriteFile(path, raw, 0o600)
			wav := testWAV()
			if test == "encoded-format" {
				binary.LittleEndian.PutUint32(wav[24:28], 48000)
			}
			receipt := fakeReceipt(raw, wav)
			switch test {
			case "request-hash":
				receipt["request_sha256"] = strings.Repeat("b", 64)
			case "model":
				receipt["model"].(map[string]any)["model_id"] = "ACE-Step"
			case "seed":
				receipt["sampling"].(map[string]any)["seed"] = 1
			case "missing-zero-seed":
				delete(receipt["sampling"].(map[string]any), "seed")
			case "missing-cfg":
				delete(receipt["sampling"].(map[string]any), "cfg_scale")
			case "audio-hash":
				receipt["output"].(map[string]any)["sha256"] = strings.Repeat("c", 64)
			case "runtime":
				delete(receipt, "runtime")
			case "missing-lock-identity":
				delete(receipt["runtime"].(map[string]any)["source_files"].(map[string]string), "requirements.lock")
			}
			api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "POST" {
					if test == "destination-race" {
						os.Mkdir(out, 0o700)
					}
					json.NewEncoder(w).Encode(receipt)
				} else {
					w.Write(wav)
				}
			}))
			defer api.Close()
			if _, err := composeToDirectory(context.Background(), api.Client(), api.URL, path, out); err == nil {
				t.Fatal("invalid result was accepted")
			}
			if _, err := os.Stat(filepath.Join(out, "generation-receipt.json")); !os.IsNotExist(err) {
				t.Fatal("invalid result published a receipt", err)
			}
			pending, _ := filepath.Glob(filepath.Join(dir, ".sa3-export-*"))
			if len(pending) != 0 {
				t.Fatal("failed export left staging", pending)
			}
		})
	}
}

func TestComposeCanceledDownloadAndInvalidInputsDoNotPublish(t *testing.T) {
	dir := t.TempDir()
	raw := []byte(`{"prompt":"cue","duration":1,"seed":0}`)
	path, out := filepath.Join(dir, "request.json"), filepath.Join(dir, "out")
	os.WriteFile(path, raw, 0o600)
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			json.NewEncoder(w).Encode(fakeReceipt(raw, testWAV()))
			return
		}
		w.WriteHeader(200)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer api.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := composeToDirectory(ctx, api.Client(), api.URL, path, out); err == nil {
		t.Fatal("canceled download accepted")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("canceled download published", err)
	}
	for _, raw := range []string{`{"prompt":"cue","seed":-1}`, `{"prompt":"cue","seed":4294967296}`, `{"prompt":"cue","seed":0,"unexpected":1}`, `{"prompt":"cue","seed":0} {}`, `{"prompt":"cue"}`} {
		if _, err := decodeRequest([]byte(raw)); err == nil {
			t.Error("invalid request accepted", raw)
		}
	}
	pipe := filepath.Join(dir, "pipe")
	if err := unix.Mkfifo(pipe, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readRegularContext(context.Background(), pipe, 65536); err == nil {
		t.Fatal("FIFO accepted")
	}
}

func TestComposeRefusesUnusableOutputParentsBeforeAnyHTTP(t *testing.T) {
	for _, scenario := range []string{"missing", "not-directory", "unwritable"} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			raw := []byte(`{"prompt":"cue","duration":1,"seed":0}`)
			request := filepath.Join(dir, "request.json")
			if err := os.WriteFile(request, raw, 0o600); err != nil {
				t.Fatal(err)
			}
			parent := filepath.Join(dir, "parent")
			switch scenario {
			case "not-directory":
				if err := os.WriteFile(parent, []byte("preserve"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "unwritable":
				if os.Geteuid() == 0 {
					// Root bypasses ordinary mode bits; procfs still refuses new entries.
					parent = "/proc/self"
					if info, err := os.Stat(parent); err != nil || !info.IsDir() {
						t.Fatal("Linux procfs fixture unavailable", err)
					}
				} else {
					if err := os.Mkdir(parent, 0o500); err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() { _ = os.Chmod(parent, 0o700) })
				}
			}
			var calls atomic.Int32
			api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method == http.MethodPost {
					_ = json.NewEncoder(w).Encode(fakeReceipt(raw, testWAV()))
				} else {
					_, _ = w.Write(testWAV())
				}
			}))
			defer api.Close()
			out := filepath.Join(parent, "take")
			if _, err := composeToDirectory(context.Background(), api.Client(), api.URL, request, out); err == nil {
				t.Fatal("unusable output parent was accepted")
			}
			if calls.Load() != 0 {
				t.Fatalf("local preflight refusal made %d HTTP calls; want zero", calls.Load())
			}
			if _, err := os.Stat(out); err == nil {
				t.Fatal("refused request published output")
			}
		})
	}
}
