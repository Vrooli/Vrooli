package target

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type exitedElectronProcess struct{ waitErr error }

func (p exitedElectronProcess) PID() int               { return 73 }
func (p exitedElectronProcess) Signal(os.Signal) error { return nil }
func (p exitedElectronProcess) Kill() error            { return nil }
func (p exitedElectronProcess) Wait() error            { return p.waitErr }

func TestStartElectronSessionPreservesEarlyExitCauseAndRemovesOwnedProfile(t *testing.T) {
	artifact := filepath.Join(t.TempDir(), "electron-app")
	if err := os.WriteFile(artifact, []byte("fixture"), 0o755); err != nil {
		t.Fatal(err)
	}

	var profile string
	_, err := StartElectronSession(context.Background(), ElectronSessionOptions{
		ArtifactPath: artifact,
		Launch:       sessionLaunchOptions(),
		StartProcess: func(_ context.Context, _ string, args []string, _ []string) (ElectronProcess, error) {
			for _, arg := range args {
				if strings.HasPrefix(arg, "--user-data-dir=") {
					profile = strings.TrimPrefix(arg, "--user-data-dir=")
				}
			}
			return exitedElectronProcess{waitErr: errors.New("exit status 17")}, nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "Electron artifact exited before CDP became ready") || !strings.Contains(err.Error(), "exit status 17") {
		t.Fatalf("startup failure = %v, want early-exit diagnostic with process status", err)
	}
	if profile == "" {
		t.Fatal("launch did not receive an isolated profile directory")
	}
	if _, err := os.Stat(profile); !os.IsNotExist(err) {
		t.Fatalf("owned profile remains after early process exit: %s, stat err=%v", profile, err)
	}
}
