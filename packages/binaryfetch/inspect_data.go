package binaryfetch

import (
	"context"
	"crypto/sha1" // Git blob identity authenticates discovery; execution still requires SHA-256.
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const MaxInspectionBytes int64 = 1 << 20

type DataInspection struct {
	URL           string
	GitBlobSHA1   string
	MaxBytes      int64
	Authorization *BearerAuthorization `json:"-"`
}
type DataInspectionReceipt struct {
	SHA256      string `json:"sha256"`
	GitBlobSHA1 string `json:"git_blob_sha1"`
	Bytes       int64  `json:"bytes"`
	Origin      string `json:"origin"`
}

// InspectData verifies a bounded Git blob, publishes new non-executable bytes,
// and reports their SHA-256 for subsequent declaration. It never admits execution.
func InspectData(ctx context.Context, spec DataInspection, destination string) (DataInspectionReceipt, error) {
	var receipt DataInspectionReceipt
	pin, err := hex.DecodeString(spec.GitBlobSHA1)
	if err != nil || len(pin) != sha1.Size {
		return receipt, errors.New("inspection requires a pinned 40-hex Git blob SHA-1")
	}
	if spec.MaxBytes < 1 || spec.MaxBytes > MaxInspectionBytes {
		return receipt, errors.New("inspection max_bytes must be 1–1048576")
	}
	u, err := url.Parse(spec.URL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" {
		return receipt, errors.New("inspection requires an HTTPS URL without userinfo or fragment")
	}
	if err := ctx.Err(); err != nil {
		return receipt, err
	}
	if _, err := os.Lstat(destination); err == nil {
		return receipt, errors.New("inspection destination already exists")
	} else if !os.IsNotExist(err) {
		return receipt, err
	}
	response, err := downloadResponse(ctx, spec.URL, spec.Authorization)
	if err != nil {
		return receipt, err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return receipt, fmt.Errorf("inspection refused with HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, spec.MaxBytes+1))
	if err != nil {
		return receipt, safeDownloadError(spec.Authorization, err)
	}
	if len(body) == 0 || int64(len(body)) > spec.MaxBytes {
		return receipt, errors.New("inspection response empty or over byte limit")
	}
	if _, html := IsHTML(body); html {
		return receipt, ErrLooksLikeHTML
	}
	git := sha1.New()
	fmt.Fprintf(git, "blob %d%c", len(body), 0)
	_, _ = git.Write(body)
	if !strings.EqualFold(hex.EncodeToString(git.Sum(nil)), spec.GitBlobSHA1) {
		return receipt, errors.New("inspection Git blob identity mismatch")
	}
	if err := ctx.Err(); err != nil {
		return receipt, err
	}
	temp, err := os.CreateTemp(filepath.Dir(destination), ".inspect-data-*")
	if err != nil {
		return receipt, err
	}
	defer os.Remove(temp.Name())
	if _, err = temp.Write(body); err != nil {
		temp.Close()
		return receipt, err
	}
	if err = temp.Chmod(0600); err != nil {
		temp.Close()
		return receipt, err
	}
	if err = temp.Close(); err != nil {
		return receipt, err
	}
	if err := ctx.Err(); err != nil {
		return receipt, err
	}
	// Hard-link publication is atomic and refuses a competing destination.
	if err = os.Link(temp.Name(), destination); err != nil {
		return receipt, fmt.Errorf("publish inspection without overwrite: %w", err)
	}
	sum := sha256.Sum256(body)
	return DataInspectionReceipt{SHA256: hex.EncodeToString(sum[:]), GitBlobSHA1: strings.ToLower(spec.GitBlobSHA1), Bytes: int64(len(body)), Origin: u.Scheme + "://" + u.Host}, nil
}
