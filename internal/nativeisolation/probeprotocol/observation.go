// Package probeprotocol contains fixed nonsecret observation data only.
package probeprotocol

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"time"
)

const Helper = "/usr/bin/vrooli-native-isolation-probe"
const DeniedMarker = "/vrooli-qualification/denied.marker"
const WorkspaceDirectory = ".vrooli-isolation-canary"
const MaxReportBytes = 4096

type Observation struct {
	Repository      string    `json:"repository"`
	CanaryID        string    `json:"canaryId"`
	WorkspaceSHA256 string    `json:"workspaceSha256"`
	DeniedSHA256    string    `json:"deniedSha256"`
	Deadline        time.Time `json:"deadline"`
}

func Hex(s string, n int) bool {
	if len(s) != n {
		return false
	}
	for _, c := range s {
		if !strings.ContainsRune("0123456789abcdef", c) {
			return false
		}
	}
	return true
}
func CleanPath(s string) bool {
	return filepath.IsAbs(s) && filepath.Clean(s) == s && s != "/" && !strings.ContainsAny(s, "\x00\r\n\t :%\\")
}
func (o Observation) Validate(now time.Time) error {
	if !CleanPath(o.Repository) || !Hex(o.CanaryID, 32) || !Hex(o.WorkspaceSHA256, 64) || !Hex(o.DeniedSHA256, 64) || !now.Before(o.Deadline) || o.Deadline.Sub(now) > 5*time.Minute || o.WorkspaceSHA256 != MarkerDigest(o.CanaryID, "workspace") || o.DeniedSHA256 != MarkerDigest(o.CanaryID, "denied") {
		return errors.New("fixed observation refused")
	}
	return nil
}
func (o Observation) WorkspaceMarker() string {
	return filepath.Join(o.Repository, WorkspaceDirectory, o.CanaryID, "workspace.marker")
}
func (o Observation) Encode(now time.Time) (string, error) {
	if e := o.Validate(now); e != nil {
		return "", e
	}
	raw, e := json.Marshal(o)
	if e != nil || len(raw) > 2048 {
		return "", errors.New("fixed observation refused")
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
func Decode(raw string, now time.Time) (Observation, error) {
	var o Observation
	if len(raw) > 2800 {
		return o, errors.New("fixed observation refused")
	}
	data, e := base64.RawURLEncoding.DecodeString(raw)
	if e != nil || len(data) > 2048 {
		return o, errors.New("fixed observation refused")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&o) != nil || o.Validate(now) != nil {
		return Observation{}, errors.New("fixed observation refused")
	}
	canonical, e := o.Encode(now)
	if e != nil || canonical != raw {
		return Observation{}, errors.New("fixed observation refused")
	}
	return o, nil
}

// Report is observation data, never identity, a grant, or a qualification
// boolean. An external owner must independently observe both processes and
// terminal cgroup cleanup; this bounded report cannot replace those facts.
type Report struct {
	Event              string    `json:"event"`
	Deadline           time.Time `json:"deadline"`
	ParentPID          int       `json:"parentPid"`
	ChildPID           int       `json:"childPid"`
	WorkspaceReadable  bool      `json:"workspaceReadable"`
	MarkerReadDenied   bool      `json:"markerReadDenied"`
	NetworkObservation string    `json:"networkObservation"`
}

// Canary contents are fixed public data, never a credential or arbitrary file
// fingerprint supplied by a caller. Owners create these exact disposable bytes.
func MarkerContents(id, role string) []byte {
	if !Hex(id, 32) || (role != "workspace" && role != "denied") {
		return nil
	}
	return []byte("vrooli-native-isolation-canary:v1:" + id + ":" + role + "\n")
}
func MarkerDigest(id, role string) string {
	raw := MarkerContents(id, role)
	if raw == nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
