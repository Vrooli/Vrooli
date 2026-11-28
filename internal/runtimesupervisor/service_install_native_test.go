package runtimesupervisor

import (
	"context"
	"errors"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	platform "github.com/vrooli/platform-go"
)

func TestNativeSystemInstallPinsSealedOperatorSupervisorAndCanonicalBootMetadata(t *testing.T) {
	for _, target := range []string{"darwin", "windows"} {
		t.Run(target, func(t *testing.T) {
			home := t.TempDir()
			exe := filepath.Join(t.TempDir(), "vrooli")
			sealed := filepath.Join(t.TempDir(), "sealed", "vrooli")
			state := t.TempDir()
			storage := t.TempDir()
			e := supervisorServiceInstallEffects{target: target, executable: func(h, requested string) (string, bool, error) {
				if h != home {
					t.Fatal(h)
				}
				return exe, false, nil
			}, operator: func(os, h, image string) (platform.NativeSystemServiceBrokerOptions, error) {
				if os != target || h != home || image != exe {
					t.Fatal("wrong operator source")
				}
				return platform.NativeSystemServiceBrokerOptions{OperatorHome: home, ExecutableSHA256: strings.Repeat("a", 64), GlobalResourceStorageRoot: storage}, nil
			}, install: func(o platform.ServiceInstallOptions) (platform.ServiceInstallResult, error) {
				if o.SystemBroker == nil || o.SystemBroker.OperatorHome != home || o.SystemBroker.GlobalResourceStorageRoot != storage || o.Executable != exe || o.User || o.SourceRoot != "" || o.HomeDir != "" || o.LogPath != "" {
					t.Fatalf("operator source leaked into privileged unit: %+v", o)
				}
				return platform.ServiceInstallResult{Scope: "system", Active: true, SealedExecutablePath: sealed, StateRoot: state, LogPath: filepath.Join(state, "runtime-supervisor.log")}, nil
			}}
			got, err := installSupervisorServiceWith(context.Background(), ServiceInstallOptions{HomeDir: home, Executable: exe, SourceRoot: filepath.Join(home, "checkout")}, e)
			if err != nil {
				t.Fatal(err)
			}
			if got.Executable != sealed || !got.ExecutableIsCanonical || got.StateRoot != state || got.LogPath != filepath.Join(state, "runtime-supervisor.log") {
				t.Fatalf("lost sealed outcome %+v", got)
			}
		})
	}
}

func TestNativeSystemInstallOperatorFailurePrecedesPlatformEffects(t *testing.T) {
	denied := errors.New("operator identity unavailable")
	installed := false
	e := supervisorServiceInstallEffects{target: "darwin", executable: func(string, string) (string, bool, error) { return "/sealed/vrooli", true, nil }, operator: func(string, string, string) (platform.NativeSystemServiceBrokerOptions, error) {
		return platform.NativeSystemServiceBrokerOptions{}, denied
	}, install: func(platform.ServiceInstallOptions) (platform.ServiceInstallResult, error) {
		installed = true
		return platform.ServiceInstallResult{}, nil
	}}
	if _, err := installSupervisorServiceWith(context.Background(), ServiceInstallOptions{}, e); !errors.Is(err, denied) || installed {
		t.Fatalf("identity denial effects: %v installed=%v", err, installed)
	}
}

func TestNativeSystemOperatorMetadataRejectsRootAndHomeSubstitution(t *testing.T) {
	home := t.TempDir()
	u := &user.User{Uid: "501", Gid: "20", Username: "operator", HomeDir: home}
	o, err := nativeSupervisorOperatorMetadata("darwin", home, strings.Repeat("a", 64), u)
	if err != nil || o.OperatorPrincipal != "unix:501" || o.OperatorUID != 501 || o.OperatorGID != 20 {
		t.Fatalf("wrong identity %+v %v", o, err)
	}
	for _, bad := range []*user.User{{Uid: "0", Gid: "20", Username: "root", HomeDir: home}, {Uid: "501", Gid: "0", Username: "operator", HomeDir: home}, {Uid: "501", Gid: "20", Username: "operator", HomeDir: home + "other"}} {
		if _, err := nativeSupervisorOperatorMetadata("darwin", home, "digest", bad); err == nil {
			t.Fatalf("accepted wrong account %+v", bad)
		}
	}
	if _, err := nativeSupervisorOperatorMetadata("windows", home, "digest", &user.User{Uid: "S-1-5-18", Username: "SYSTEM", HomeDir: home}); err == nil {
		t.Fatal("accepted LocalSystem as operator")
	}
}
