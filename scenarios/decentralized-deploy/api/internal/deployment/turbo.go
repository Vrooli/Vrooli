package deployment

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	TurboSandboxMaxItemBytes = 5 * 1024 * 1024
	turboSandboxUploadHost   = "upload.services.ar-io.dev"
	turboSandboxGatewayHost  = "ar-io.dev"
)

// TurboSandboxAdapter sends exactly one signed ANS-104 item to ar.io's free
// public testnet sandbox. It has no payment path and rejects any receipt that
// does not explicitly say winc="0". The generated Ed25519 key stays in memory
// only and is cleared on return; it is never persisted in a deployment record.
type TurboSandboxAdapter struct {
	Gate       Gate
	Policy     PathRecord
	Client     *http.Client
	UploadURL  string
	GatewayURL string
	Random     io.Reader
}

func (t TurboSandboxAdapter) Name() string { return "turbo-sandbox" }

func (t TurboSandboxAdapter) validatePolicy() error {
	if t.Policy.Provider != "turbo" || t.Policy.Path != "ar-io-free-testnet-sandbox" || t.Policy.Category != "decentralized-storage-free-sandbox" {
		return errors.New("Turbo sandbox adapter requires the recorded free-testnet catalogue path")
	}
	for _, item := range []struct {
		label string
		raw   string
		host  string
	}{{"upload", t.Policy.UploadURL, turboSandboxUploadHost}, {"gateway", t.Policy.GatewayURL, turboSandboxGatewayHost}} {
		parsed, err := url.Parse(item.raw)
		if err != nil || parsed.Scheme != "https" || parsed.Hostname() != item.host || parsed.Port() != "" || parsed.User != nil || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
			return fmt.Errorf("Turbo sandbox catalogue %s URL must use the approved HTTPS endpoint %s", item.label, item.host)
		}
	}
	if t.Policy.MaxItemBytes <= 0 || t.Policy.MaxItemBytes > TurboSandboxMaxItemBytes {
		return errors.New("Turbo sandbox catalogue item limit is outside the hard safety bound")
	}
	return nil
}

func (t TurboSandboxAdapter) client() *http.Client {
	if t.Client != nil {
		return t.Client
	}
	return &http.Client{
		Timeout:       30 * time.Second,
		Transport:     &http.Transport{Proxy: nil, TLSHandshakeTimeout: 10 * time.Second},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

func (t TurboSandboxAdapter) uploadURL() string {
	if t.UploadURL != "" {
		return strings.TrimRight(t.UploadURL, "/")
	}
	return strings.TrimRight(t.Policy.UploadURL, "/")
}

func (t TurboSandboxAdapter) gatewayURL() string {
	if t.GatewayURL != "" {
		return strings.TrimRight(t.GatewayURL, "/")
	}
	return strings.TrimRight(t.Policy.GatewayURL, "/")
}

func (t TurboSandboxAdapter) entryBytes(artifact Artifact) ([]byte, error) {
	if artifact.EntryPath == "" {
		return nil, errors.New("Turbo sandbox upload requires an explicit artifact entry path")
	}
	clean := filepath.Clean(artifact.EntryPath)
	if filepath.IsAbs(clean) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return nil, errors.New("Turbo sandbox entry path must be relative to the artifact root")
	}
	rootPath := filepath.Clean(artifact.Path)
	rootInfo, err := os.Lstat(rootPath)
	if err != nil {
		return nil, fmt.Errorf("inspect Turbo artifact root: %w", err)
	}
	if !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("Turbo artifact root must be a real directory")
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, fmt.Errorf("open Turbo artifact root: %w", err)
	}
	defer root.Close()
	current := ""
	parts := strings.Split(filepath.ToSlash(clean), "/")
	for index, part := range parts {
		if part == "" || part == "." || part == ".." {
			return nil, errors.New("invalid Turbo artifact entry path")
		}
		if current == "" {
			current = part
		} else {
			current = filepath.Join(current, part)
		}
		info, err := root.Lstat(current)
		if err != nil {
			return nil, fmt.Errorf("inspect Turbo artifact entry: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("Turbo artifact entry contains a symlink")
		}
		if index < len(parts)-1 && !info.IsDir() {
			return nil, errors.New("Turbo artifact entry parent is not a directory")
		}
		if index == len(parts)-1 && !info.Mode().IsRegular() {
			return nil, errors.New("Turbo artifact entry must be a regular file")
		}
	}
	f, err := root.Open(clean)
	if err != nil {
		return nil, fmt.Errorf("read Turbo artifact entry: %w", err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, TurboSandboxMaxItemBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read Turbo artifact entry: %w", err)
	}
	if len(data) == 0 || len(data) > TurboSandboxMaxItemBytes {
		return nil, fmt.Errorf("Turbo sandbox item size %d is outside the 1..%d byte free-tier bound", len(data), TurboSandboxMaxItemBytes)
	}
	return data, nil
}

type turboInfo struct {
	FreeUploadLimitBytes int `json:"freeUploadLimitBytes"`
	FreeTier             struct {
		LifetimeBytes int `json:"lifetimeBytes"`
		IPBytes       int `json:"ipBytes"`
		MaxItemBytes  int `json:"maxItemBytes"`
	} `json:"freeTier"`
}

func (t TurboSandboxAdapter) getInfo(ctx context.Context) (turboInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.uploadURL()+"/v1/info", nil)
	if err != nil {
		return turboInfo{}, err
	}
	resp, err := t.client().Do(req)
	if err != nil {
		return turboInfo{}, fmt.Errorf("read Turbo sandbox limits: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return turboInfo{}, fmt.Errorf("read Turbo sandbox limits: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32*1024+1))
	if err != nil || len(body) > 32*1024 {
		return turboInfo{}, errors.New("Turbo sandbox info exceeded the response bound")
	}
	var info turboInfo
	if err := decodeSingleJSON(body, &info); err != nil {
		return turboInfo{}, fmt.Errorf("decode Turbo sandbox limits: %w", err)
	}
	if info.FreeUploadLimitBytes <= 0 || info.FreeTier.MaxItemBytes <= 0 || info.FreeTier.LifetimeBytes <= 0 || info.FreeTier.IPBytes <= 0 {
		return turboInfo{}, errors.New("Turbo sandbox did not report a positive free-tier allowance")
	}
	return info, nil
}

// decodeSingleJSON rejects concatenated JSON values and trailing non-whitespace.
// Provider responses that merely start with an acceptable object are not
// sufficient evidence for a free-tier decision.
func decodeSingleJSON(body []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("unexpected trailing JSON value")
		}
		return fmt.Errorf("unexpected trailing response data: %w", err)
	}
	return nil
}

func avroLong(value int64) []byte {
	encoded := uint64((value << 1) ^ (value >> 63))
	var result []byte
	for encoded >= 0x80 {
		result = append(result, byte(encoded)|0x80)
		encoded >>= 7
	}
	return append(result, byte(encoded))
}

func turboTags() []byte {
	name := []byte("Content-Type")
	value := []byte("text/html; charset=utf-8")
	result := avroLong(1) // one tag
	result = append(result, avroLong(int64(len(name)))...)
	result = append(result, name...)
	result = append(result, avroLong(int64(len(value)))...)
	result = append(result, value...)
	return append(result, 0) // end of AVRO array
}

func sha384(data []byte) []byte { sum := sha512.Sum384(data); return sum[:] }

func deepHashBlob(data []byte) []byte {
	length := []byte(fmt.Sprintf("blob%d", len(data)))
	return sha384(append(sha384(length), sha384(data)...))
}

func deepHashList(fields ...[]byte) []byte {
	acc := sha384([]byte(fmt.Sprintf("list%d", len(fields))))
	for _, field := range fields {
		acc = sha384(append(acc, deepHashBlob(field)...))
	}
	return acc
}

func signedTurboItem(data []byte, source io.Reader) ([]byte, string, error) {
	if source == nil {
		source = rand.Reader
	}
	publicKey, privateKey, err := ed25519.GenerateKey(source)
	if err != nil {
		return nil, "", fmt.Errorf("generate ephemeral Turbo signer: %w", err)
	}
	defer clear(privateKey)
	tags := turboTags()
	digest := deepHashList([]byte("dataitem"), []byte("1"), []byte("4"), publicKey, nil, nil, tags, data)
	signature := ed25519.Sign(privateKey, []byte(hex.EncodeToString(digest)))
	item := make([]byte, 0, 2+64+32+2+8+8+len(tags)+len(data))
	var short [2]byte
	binary.LittleEndian.PutUint16(short[:], 4)
	item = append(item, short[:]...)
	item = append(item, signature...)
	item = append(item, publicKey...)
	item = append(item, 0, 0) // target and anchor lengths
	var word [8]byte
	binary.LittleEndian.PutUint64(word[:], 1)
	item = append(item, word[:]...)
	binary.LittleEndian.PutUint64(word[:], uint64(len(tags)))
	item = append(item, word[:]...)
	item = append(item, tags...)
	item = append(item, data...)
	idHash := sha256.Sum256(signature)
	id := base64.RawURLEncoding.EncodeToString(idHash[:])
	return item, id, nil
}

type turboReceipt struct {
	ID                  string   `json:"id"`
	Owner               string   `json:"owner"`
	Winc                string   `json:"winc"`
	Timestamp           int64    `json:"timestamp"`
	DataCaches          []string `json:"dataCaches"`
	FastFinalityIndexes []string `json:"fastFinalityIndexes"`
	Version             string   `json:"version"`
	DeadlineHeight      int64    `json:"deadlineHeight"`
	Public              string   `json:"public"`
	Signature           string   `json:"signature"`
}

func (t TurboSandboxAdapter) postItem(ctx context.Context, item []byte, expectedID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.uploadURL()+"/v1/tx", bytes.NewReader(item))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := t.client().Do(req)
	if err != nil {
		return fmt.Errorf("Turbo sandbox upload outcome is ambiguous; no retry permitted: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32*1024+1))
	if err != nil || len(body) > 32*1024 {
		return errors.New("Turbo sandbox receipt exceeded the response bound; no retry permitted")
	}
	if resp.StatusCode != http.StatusOK {
		diagnostic := strings.TrimSpace(string(body))
		if len(diagnostic) > 1024 {
			diagnostic = diagnostic[:1024] + "…"
		}
		return fmt.Errorf("Turbo sandbox upload refused with HTTP %d for item %s (%q); no payment or retry permitted", resp.StatusCode, expectedID, diagnostic)
	}
	var receipt turboReceipt
	if err := decodeSingleJSON(body, &receipt); err != nil {
		return fmt.Errorf("decode Turbo sandbox receipt; no retry permitted: %w", err)
	}
	if receipt.ID != expectedID || receipt.Winc != "0" {
		return fmt.Errorf("Turbo sandbox receipt failed free-tier identity/cost gate (expected_id=%s, receipt_id=%s, winc=%q); no retry permitted", expectedID, receipt.ID, receipt.Winc)
	}
	return nil
}

func (t TurboSandboxAdapter) fetch(ctx context.Context, id string) ([]byte, error) {
	if id == "" || strings.ContainsAny(id, "/?#") {
		return nil, errors.New("invalid Turbo sandbox item id")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.gatewayURL()+"/raw/"+id, nil)
	if err != nil {
		return nil, err
	}
	resp, err := t.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("retrieve Turbo sandbox item: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("retrieve Turbo sandbox item: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, TurboSandboxMaxItemBytes+1))
	if err != nil || len(data) > TurboSandboxMaxItemBytes {
		return nil, errors.New("Turbo sandbox retrieval exceeded the item bound")
	}
	return data, nil
}

func (t TurboSandboxAdapter) Deploy(ctx context.Context, artifact Artifact) (StoredContent, error) {
	if err := t.validatePolicy(); err != nil {
		return StoredContent{}, err
	}
	if err := t.Gate.Check(ctx, EffectSandbox, t.Name()); err != nil {
		return StoredContent{}, err
	}
	data, err := t.entryBytes(artifact)
	if err != nil {
		return StoredContent{}, err
	}
	info, err := t.getInfo(ctx)
	if err != nil {
		return StoredContent{}, err
	}
	if len(data) > info.FreeUploadLimitBytes || len(data) > info.FreeTier.MaxItemBytes || len(data) > t.Policy.MaxItemBytes {
		return StoredContent{}, fmt.Errorf("Turbo item %d bytes exceeds the current free tier", len(data))
	}
	item, id, err := signedTurboItem(data, t.Random)
	if err != nil {
		return StoredContent{}, err
	}
	if err := t.postItem(ctx, item, id); err != nil {
		return StoredContent{}, err
	}
	got, err := t.fetch(ctx, id)
	if err != nil {
		return StoredContent{}, fmt.Errorf("Turbo sandbox upload receipt accepted for item %s, but exact retrieval failed: %w; do not retry the POST", id, err)
	}
	wantHash, gotHash := sha256.Sum256(data), sha256.Sum256(got)
	if !bytes.Equal(data, got) || wantHash != gotHash {
		return StoredContent{}, fmt.Errorf("Turbo sandbox retrieval bytes for item %s differ from the submitted synthetic content", id)
	}
	return StoredContent{Provider: t.Name(), CID: id, URI: t.gatewayURL() + "/raw/" + id, EntryPath: artifact.EntryPath}, nil
}

func (t TurboSandboxAdapter) Renew(ctx context.Context, content StoredContent) error {
	if err := t.validatePolicy(); err != nil {
		return err
	}
	if err := t.Gate.Check(ctx, EffectSandbox, t.Name()); err != nil {
		return err
	}
	return errors.New("Turbo sandbox items are ephemeral and have no renewal API")
}

func (t TurboSandboxAdapter) Teardown(ctx context.Context, content StoredContent) error {
	if err := t.validatePolicy(); err != nil {
		return err
	}
	if err := t.Gate.Check(ctx, EffectSandbox, t.Name()); err != nil {
		return err
	}
	// The sandbox exposes no deletion guarantee. This removes local scenario
	// state only; public gateway data may remain until the sandbox purges it.
	return nil
}

func (t TurboSandboxAdapter) Resolve(ctx context.Context, content StoredContent) ([]byte, error) {
	if err := t.validatePolicy(); err != nil {
		return nil, err
	}
	if err := t.Gate.Check(ctx, EffectSandbox, t.Name()); err != nil {
		return nil, err
	}
	data, err := t.fetch(ctx, content.CID)
	if err != nil {
		return nil, err
	}
	return data, nil
}
