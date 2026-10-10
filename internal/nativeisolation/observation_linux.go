//go:build linux

package nativeisolation

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/vrooli/vrooli/internal/nativeisolation/probeprotocol"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

func checkObservationMarkers(s ProbeSpec) error {
	o, e := s.observation(s.Deadline.Add(-1))
	if e != nil {
		return ErrRefused
	}
	if trustedPath(filepath.Dir(s.FirstObservation.deniedSource()), s.Manifest.NativeUID) != nil {
		return ErrRefused
	}
	if checkWorkspace(s.Manifest.WorkspaceSource, s.Manifest.Isolation.ProtectedPaths) != nil {
		return ErrRefused
	}
	for _, marker := range []struct {
		path, hash string
		denied     bool
	}{{s.FirstObservation.deniedSource(), o.DeniedSHA256, true}, {filepath.Join(s.Manifest.WorkspaceSource, probeprotocol.WorkspaceDirectory, o.CanaryID, "workspace.marker"), o.WorkspaceSHA256, false}} {
		info, e := os.Lstat(marker.path)
		if e != nil || !info.Mode().IsRegular() || info.Size() > 1024 {
			return ErrRefused
		}
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return ErrRefused
		}
		if marker.denied && (st.Uid == s.Manifest.NativeUID || info.Mode().Perm() != 0600) {
			return ErrRefused
		}
		// Refuse aliasing through marker directory components before any read.
		real, e := filepath.EvalSymlinks(marker.path)
		if e != nil || real != marker.path {
			return ErrRefused
		}
		fd, e := syscall.Open(marker.path, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
		if e != nil {
			return ErrRefused
		}
		f := os.NewFile(uintptr(fd), marker.path)
		raw, re := io.ReadAll(io.LimitReader(f, 1025))
		f.Close()
		sum := sha256.Sum256(raw)
		if re != nil || len(raw) > 1024 || hex.EncodeToString(sum[:]) != marker.hash {
			return ErrRefused
		}
	}
	return nil
}
