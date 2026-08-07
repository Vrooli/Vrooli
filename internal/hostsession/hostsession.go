package hostsession

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	linuxBootIDPath = "/proc/sys/kernel/random/boot_id"
	sessionFile     = "host_session_id"
)

// Snapshot identifies the current host execution session closely enough to
// decide whether durable runtime records were written before the latest boot.
type Snapshot struct {
	BootID    string
	SessionID string
	Source    string
}

type Provider interface {
	Current(ctx context.Context, home string) (Snapshot, error)
}

type DefaultProvider struct{}

func (DefaultProvider) Current(ctx context.Context, home string) (Snapshot, error) {
	select {
	case <-ctx.Done():
		return Snapshot{}, ctx.Err()
	default:
	}
	if hostreqspec.CurrentPlatform() == "linux" {
		bootID, err := readTextFile(linuxBootIDPath)
		if err == nil && bootID != "" {
			return Snapshot{BootID: bootID, SessionID: bootID, Source: "linux_boot_id"}, nil
		}
	}
	token, err := persistentFallbackSession(home)
	if err != nil {
		return Snapshot{}, err
	}
	return Snapshot{BootID: token, SessionID: token, Source: "persistent_fallback"}, nil
}

func readTextFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func persistentFallbackSession(home string) (string, error) {
	if strings.TrimSpace(home) == "" {
		resolved, err := config.HomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		home = resolved
	}
	stateDir, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return "", err
	}
	path := filepath.Join(stateDir, sessionFile)
	if token, err := readTextFile(path); err == nil && token != "" {
		return token, nil
	} else if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("read host session fallback token: %w", err)
	}
	if _, err := config.EnsureOwnedDir(filepath.Dir(path)); err != nil {
		return "", fmt.Errorf("prepare host session directory: %w", err)
	}
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate host session fallback token: %w", err)
	}
	token := hex.EncodeToString(raw[:])
	if err := config.WriteOwnedFile(path, []byte(token+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("write host session fallback token: %w", err)
	}
	return token, nil
}
