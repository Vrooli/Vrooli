package deployment

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func verifyTurboItemForTest(item []byte) (string, []byte, error) {
	if len(item) < 124 || binary.LittleEndian.Uint16(item[:2]) != 4 {
		return "", nil, fmt.Errorf("invalid ANS-104 header")
	}
	if item[98] != 0 || item[99] != 0 {
		return "", nil, fmt.Errorf("unexpected target or anchor")
	}
	tagCount := binary.LittleEndian.Uint64(item[100:108])
	tagLength := binary.LittleEndian.Uint64(item[108:116])
	if tagCount != 1 || tagLength > uint64(len(item)-124) {
		return "", nil, fmt.Errorf("invalid tag framing")
	}
	tagStart := 116
	dataStart := tagStart + int(tagLength)
	if dataStart > len(item) {
		return "", nil, fmt.Errorf("truncated item")
	}
	tags := item[tagStart:dataStart]
	data := item[dataStart:]
	pk := item[66:98]
	deep := deepHashList([]byte("dataitem"), []byte("1"), []byte("4"), pk, nil, nil, tags, data)
	if !ed25519.Verify(pk, []byte(hex.EncodeToString(deep)), item[2:66]) {
		return "", nil, fmt.Errorf("signature invalid")
	}
	idHash := sha256.Sum256(item[2:66])
	return base64.RawURLEncoding.EncodeToString(idHash[:]), data, nil
}

func TestSignedTurboItemUsesQualifiedType4AndExactPayload(t *testing.T) {
	payload := []byte("<h1>synthetic</h1>")
	item, id, err := signedTurboItem(payload, bytes.NewReader(bytes.Repeat([]byte{7}, ed25519.SeedSize)))
	if err != nil {
		t.Fatal(err)
	}
	gotID, gotPayload, err := verifyTurboItemForTest(item)
	if err != nil {
		t.Fatal(err)
	}
	if gotID != id || !bytes.Equal(gotPayload, payload) {
		t.Fatalf("item mismatch id=%s expected=%s payload=%q", gotID, id, gotPayload)
	}
	if len(turboTags()) == 0 {
		t.Fatal("missing content type tag")
	}
	fixtureSizePayload := bytes.Repeat([]byte("x"), 661)
	fixtureSizeItem, _, err := signedTurboItem(fixtureSizePayload, bytes.NewReader(bytes.Repeat([]byte{8}, ed25519.SeedSize)))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(fixtureSizeItem), 817; got != want {
		t.Fatalf("type-4 fixture framing length=%d want=%d", got, want)
	}
}

func TestTurboItemDecoderVerifiesRetainedPublicSandboxFixture(t *testing.T) {
	item, err := os.ReadFile("testdata/public-sandbox-type4.item")
	if err != nil {
		t.Fatal(err)
	}
	wantPayload, err := os.ReadFile("testdata/index.html")
	if err != nil {
		t.Fatal(err)
	}
	id, payload, err := verifyTurboItemForTest(item)
	if err != nil {
		t.Fatal(err)
	}
	if id != "m8CEFnBzGw6bil57QYP8Hmaz-UpLBOsxmvmu54GVb8Y" {
		t.Fatalf("fixture item id=%s", id)
	}
	if !bytes.Equal(payload, wantPayload) {
		t.Fatalf("fixture payload differs: got %d bytes want %d", len(payload), len(wantPayload))
	}
}

func TestTurboSandboxAdapterUsesFreeReceiptAndExactRetrieval(t *testing.T) {
	payload := []byte("<!doctype html><h1>synthetic free sandbox</h1>")
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "site"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "site", "index.html"), payload, 0o600); err != nil {
		t.Fatal(err)
	}
	var postCount, getCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/info":
			_, _ = w.Write([]byte(`{"freeUploadLimitBytes":5242880,"freeTier":{"lifetimeBytes":104857600,"ipBytes":104857600,"maxItemBytes":5242880}}`))
		case "/v1/tx":
			postCount++
			if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/octet-stream" || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
				t.Errorf("unexpected request headers/method: %s %v", r.Method, r.Header)
			}
			item, err := ioReadBounded(r.Body, TurboSandboxMaxItemBytes+128)
			if err != nil {
				t.Errorf("read item: %v", err)
				http.Error(w, "bad", 400)
				return
			}
			id, gotPayload, err := verifyTurboItemForTest(item)
			if err != nil || !bytes.Equal(gotPayload, payload) {
				t.Errorf("verify posted item: id=%s payload=%q err=%v", id, gotPayload, err)
				http.Error(w, "bad", 400)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"id": id, "owner": "ephemeral", "winc": "0", "timestamp": 1, "dataCaches": []string{}, "fastFinalityIndexes": []string{}, "version": "0.2.0", "deadlineHeight": 1, "public": "public", "signature": "signature"})
		default:
			if len(r.URL.Path) > len("/raw/") && r.URL.Path[:len("/raw/")] == "/raw/" {
				getCount++
				_, _ = w.Write(payload)
				return
			}
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	adapter := TurboSandboxAdapter{Gate: Gate{Mode: GateStub, AllowFreeSandbox: true}, Policy: testTurboSandboxPolicy(t), Client: server.Client(), UploadURL: server.URL, GatewayURL: server.URL}
	stored, err := adapter.Deploy(context.Background(), Artifact{Path: root, EntryPath: "site/index.html", Name: "reference"})
	if err != nil {
		t.Fatal(err)
	}
	if stored.Provider != "turbo-sandbox" || stored.CID == "" || stored.URI != server.URL+"/raw/"+stored.CID {
		t.Fatalf("unexpected storage receipt: %+v", stored)
	}
	resolved, err := adapter.Resolve(context.Background(), stored)
	if err != nil || !bytes.Equal(resolved, payload) {
		t.Fatalf("resolve=%q err=%v", resolved, err)
	}
	if postCount != 1 || getCount != 2 {
		t.Fatalf("request counts post=%d get=%d", postCount, getCount)
	}
	if sha256.Sum256(resolved) != sha256.Sum256(payload) {
		t.Fatal("content hash mismatch")
	}
}

func TestTurboSandboxGateAndNonzeroReceiptFailClosedWithoutRetry(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "site"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "site", "index.html"), []byte("<p>x</p>"), 0o600); err != nil {
		t.Fatal(err)
	}
	var postCount, getCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/info":
			_, _ = w.Write([]byte(`{"freeUploadLimitBytes":5242880,"freeTier":{"lifetimeBytes":104857600,"ipBytes":104857600,"maxItemBytes":5242880}}`))
		case "/v1/tx":
			postCount++
			item, _ := ioReadBounded(r.Body, TurboSandboxMaxItemBytes+128)
			id, _, _ := verifyTurboItemForTest(item)
			_, _ = fmt.Fprintf(w, `{"id":%q,"winc":"1","version":"0.2.0"}`, id)
		case "/raw/":
			getCount++
		default:
			if len(r.URL.Path) > 5 && r.URL.Path[:5] == "/raw/" {
				getCount++
				_, _ = w.Write([]byte("x"))
				return
			}
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	closed := TurboSandboxAdapter{Policy: testTurboSandboxPolicy(t), Client: server.Client(), UploadURL: server.URL, GatewayURL: server.URL}
	if _, err := closed.Deploy(context.Background(), Artifact{Path: root, EntryPath: "site/index.html"}); err == nil {
		t.Fatal("sandbox effect should be closed by default")
	}
	if postCount != 0 {
		t.Fatalf("closed gate sent %d uploads", postCount)
	}
	open := closed
	open.Gate = Gate{AllowFreeSandbox: true}
	if _, err := open.Deploy(context.Background(), Artifact{Path: root, EntryPath: "site/index.html"}); err == nil {
		t.Fatal("nonzero winc receipt must fail")
	}
	if postCount != 1 || getCount != 0 {
		t.Fatalf("nonfree receipt retried or retrieved: post=%d get=%d", postCount, getCount)
	}
}

func TestTurboSandboxHTTPRefusalRetainsBoundedReasonAndDoesNotRetry(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "site"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "site", "index.html"), []byte("<p>synthetic</p>"), 0o600); err != nil {
		t.Fatal(err)
	}
	var posts int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/info":
			_, _ = w.Write([]byte(`{"freeUploadLimitBytes":5242880,"freeTier":{"lifetimeBytes":104857600,"ipBytes":104857600,"maxItemBytes":5242880}}`))
		case "/v1/tx":
			posts++
			http.Error(w, "malformed type-4 payload", http.StatusBadRequest)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	adapter := TurboSandboxAdapter{Gate: Gate{AllowFreeSandbox: true}, Policy: testTurboSandboxPolicy(t), Client: server.Client(), UploadURL: server.URL, GatewayURL: server.URL}
	_, err := adapter.Deploy(context.Background(), Artifact{Path: root, EntryPath: "site/index.html"})
	if err == nil || !strings.Contains(err.Error(), "malformed type-4 payload") {
		t.Fatalf("expected bounded provider refusal detail, got %v", err)
	}
	if posts != 1 {
		t.Fatalf("provider refusal retried %d times", posts)
	}
}

func TestTurboSandboxRefusesWithoutRecordedCataloguePolicy(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		http.NotFound(w, r)
	}))
	defer server.Close()
	adapter := TurboSandboxAdapter{Gate: Gate{AllowFreeSandbox: true}, Client: server.Client(), UploadURL: server.URL, GatewayURL: server.URL}
	if _, err := adapter.Deploy(context.Background(), Artifact{Path: t.TempDir(), EntryPath: "site/index.html"}); err == nil {
		t.Fatal("adapter must require a recorded free-sandbox path policy")
	}
	if requests != 0 {
		t.Fatalf("adapter made %d requests without a catalogue path", requests)
	}
	policy := testTurboSandboxPolicy(t)
	policy.UploadURL = "https://upload.services.ar-io.dev.evil.example"
	adapter.Policy = policy
	if err := adapter.validatePolicy(); err == nil {
		t.Fatal("adapter accepted an unapproved upload host from the catalogue")
	}
	if requests != 0 {
		t.Fatalf("adapter contacted a host after catalogue validation failed: %d requests", requests)
	}
}

func TestTurboSandboxReceiptRejectsTrailingJSONWithoutRetrieval(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "site"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "site", "index.html"), []byte("<p>synthetic</p>"), 0o600); err != nil {
		t.Fatal(err)
	}
	var posts, gets int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/info":
			_, _ = w.Write([]byte(`{"freeUploadLimitBytes":5242880,"freeTier":{"lifetimeBytes":104857600,"ipBytes":104857600,"maxItemBytes":5242880}}`))
		case "/v1/tx":
			posts++
			item, err := ioReadBounded(r.Body, TurboSandboxMaxItemBytes+128)
			if err != nil {
				t.Errorf("read item: %v", err)
				return
			}
			id, _, err := verifyTurboItemForTest(item)
			if err != nil {
				t.Errorf("verify item: %v", err)
				return
			}
			_, _ = fmt.Fprintf(w, `{"id":%q,"winc":"0"}{"unexpected":true}`, id)
		case "/raw/":
			gets++
		default:
			if strings.HasPrefix(r.URL.Path, "/raw/") {
				gets++
			}
		}
	}))
	defer server.Close()
	adapter := TurboSandboxAdapter{Gate: Gate{AllowFreeSandbox: true}, Policy: testTurboSandboxPolicy(t), Client: server.Client(), UploadURL: server.URL, GatewayURL: server.URL}
	if _, err := adapter.Deploy(context.Background(), Artifact{Path: root, EntryPath: "site/index.html"}); err == nil || !strings.Contains(err.Error(), "trailing JSON") {
		t.Fatalf("expected trailing receipt data to fail closed, got %v", err)
	}
	if posts != 1 || gets != 0 {
		t.Fatalf("invalid receipt must not retry or retrieve: posts=%d gets=%d", posts, gets)
	}
}

func testTurboSandboxPolicy(t *testing.T) PathRecord {
	t.Helper()
	file, err := os.Open(filepath.Join("..", "..", "..", "data", "provider-path-catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	catalog, err := LoadCatalog(file)
	if err != nil {
		t.Fatal(err)
	}
	record, err := catalog.Path("turbo", "ar-io-free-testnet-sandbox")
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func ioReadBounded(r io.Reader, limit int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("body exceeds %d bytes", limit)
	}
	return body, nil
}
