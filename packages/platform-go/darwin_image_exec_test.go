//go:build !windows

package platform

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type darwinImageExecFixture struct {
	path, root string
	calls      int
	argv, env  []string
	deny       string
}

func newDarwinImageExecFixture(t *testing.T) (darwinImageExecEngine, *darwinImageExecFixture, *os.File) {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "observed-process-image")
	if err := os.WriteFile(path, darwinBrokerMachFixture("/usr/lib/libSystem.B.dylib"), 0555); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { file.Close() })
	f := &darwinImageExecFixture{path: path, root: root}
	check := func(path string, directory bool) error {
		if path == f.deny {
			return ErrOperationIdentity
		}
		st, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if !isDarwinFixtureChild(path, root) {
			return nil
		}
		return validateDarwinBrokerMetadata(path, st.Mode(), 0, 1, directory, "")
	}
	e := darwinImageExecEngine{check: check, path: func(ctx context.Context, path string, dir bool) error {
		return verifyDarwinImagePath(ctx, path, dir, check)
	}, format: func(file *os.File) error { return validateDarwinNativeExecutable(file, "arm64") }, exec: func(path string, argv, env []string) error {
		f.calls++
		if path != f.path {
			return ErrOperationIdentity
		}
		f.argv = argv
		f.env = env
		return nil
	}}
	return e, f, file
}
func isDarwinFixtureChild(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && len(rel) > 0 && (len(rel) < 3 || rel[:3] != "../")
}
func TestDarwinImageExecPreservesExplicitFrozenInvocation(t *testing.T) {
	e, f, file := newDarwinImageExecFixture(t)
	argv := []string{f.path, "--config", "private settings"}
	env := []string{"XDG_CONFIG_HOME=/private/config", "CAPSULE_DESCRIPTOR=/sealed/input"}
	if err := e.replace(file, argv, env); err != nil {
		t.Fatal(err)
	}
	if f.calls != 1 || !reflect.DeepEqual(f.argv, argv) || !reflect.DeepEqual(f.env, env) {
		t.Fatalf("invocation changed: %+v", f)
	}
	argv[1] = "mutated"
	env[0] = "mutated"
	if f.argv[1] == "mutated" || f.env[0] == "mutated" {
		t.Fatal("handoff retained mutable caller slices")
	}
}
func TestDarwinImageExecDoesNotUseObservedFileNameAsPath(t *testing.T) {
	e, f, _ := newDarwinImageExecFixture(t)
	t.Chdir(f.root)
	file, err := os.Open("observed-process-image")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if filepath.IsAbs(file.Name()) {
		t.Fatal("fixture did not retain observation-style filename")
	}
	if err = e.replace(file, []string{f.path}, []string{}); err != nil {
		t.Fatalf("explicit same-vnode path was not used: %v", err)
	}
	if f.calls != 1 {
		t.Fatal("handoff was not reached")
	}
}
func TestDarwinImageExecRefusesReplacedVerifiedPath(t *testing.T) {
	e, f, file := newDarwinImageExecFixture(t)
	replacement := filepath.Join(f.root, "replacement")
	if err := os.WriteFile(replacement, darwinBrokerMachFixture("/usr/lib/libSystem.B.dylib"), 0555); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacement, f.path); err != nil {
		t.Fatal(err)
	}
	if err := e.replace(file, []string{f.path}, []string{}); !errors.Is(err, ErrOperationIdentity) {
		t.Fatalf("replaced vnode was accepted: %v", err)
	}
	if f.calls != 0 {
		t.Fatal("replacement reached exec")
	}
}
func TestDarwinImageExecRefusesUnsealedAncestorAndWritableLeaf(t *testing.T) {
	for _, kind := range []string{"ancestor-acl", "writable-leaf", "symlink-path"} {
		t.Run(kind, func(t *testing.T) {
			e, f, file := newDarwinImageExecFixture(t)
			path := f.path
			switch kind {
			case "ancestor-acl":
				f.deny = f.root
			case "writable-leaf":
				if err := os.Chmod(f.path, 0755); err != nil {
					t.Fatal(err)
				}
			case "symlink-path":
				path = filepath.Join(f.root, "alias")
				if err := os.Symlink(f.path, path); err != nil {
					t.Fatal(err)
				}
			}
			if err := e.replace(file, []string{path}, []string{}); !errors.Is(err, ErrOperationIdentity) {
				t.Fatalf("unsafe native path accepted: %v", err)
			}
			if f.calls != 0 {
				t.Fatal("unsafe path reached exec")
			}
		})
	}
}
func TestDarwinImageExecRefusesScriptsAndAmbientDyld(t *testing.T) {
	for _, kind := range []string{"script", "dyld-insert", "dyld-empty", "nil-env", "duplicate-env", "relative-executable"} {
		t.Run(kind, func(t *testing.T) {
			e, f, file := newDarwinImageExecFixture(t)
			argv := []string{f.path}
			env := []string{}
			switch kind {
			case "script":
				if err := os.Chmod(f.path, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(f.path, []byte("#!/bin/sh\necho unowned"), 0555); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(f.path, 0555); err != nil {
					t.Fatal(err)
				}
			case "dyld-insert":
				env = []string{"DYLD_INSERT_LIBRARIES=/operator/evil.dylib"}
			case "dyld-empty":
				env = []string{"DYLD_LIBRARY_PATH="}
			case "nil-env":
				env = nil
			case "duplicate-env":
				env = []string{"APP=one", "APP=two"}
			case "relative-executable":
				argv[0] = "observed-process-image"
			}
			if err := e.replace(file, argv, env); err == nil {
				t.Fatalf("unsafe native invocation %s accepted", kind)
			}
			if f.calls != 0 {
				t.Fatal("unsafe invocation reached exec")
			}
		})
	}
}
func TestDarwinFrozenEnvironmentIsAuthoritativeClonedAndExact(t *testing.T) {
	b, _, i, s := newDarwinFixture(t)
	ref := startDarwinFixture(t, b, i, s)
	env, err := b.frozenEnvironment(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(env, s.Env) {
		t.Fatalf("frozen environment guessed from native additions: %v", env)
	}
	env = append(env, "MUTATED=caller")
	again, err := b.frozenEnvironment(context.Background(), ref)
	if err != nil || !reflect.DeepEqual(again, s.Env) {
		t.Fatalf("caller modified persisted environment: %v %v", again, err)
	}
	wrong := ref
	native := *ref.Native
	wrong.Native = &native
	wrong.Native.RootIncarnation = "darwin:42:100:124"
	if _, err = b.frozenEnvironment(context.Background(), wrong); !errors.Is(err, ErrOperationIdentity) {
		t.Fatalf("wrong root got capsule environment: %v", err)
	}
}
func TestDarwinFrozenEnvironmentRejectsStoppedRoot(t *testing.T) {
	b, f, i, s := newDarwinFixture(t)
	ref := startDarwinFixture(t, b, i, s)
	f.process = darwinOwnedProcess{}
	f.present = false
	if _, err := b.frozenEnvironment(context.Background(), ref); !errors.Is(err, ErrOperationIdentity) {
		t.Fatalf("stopped/unverified root got clean-environment handoff: %v", err)
	}
}

func TestDarwinFrozenEnvironmentRefusesStaleBoot(t *testing.T) {
	b, f, i, s := newDarwinFixture(t)
	original := startDarwinFixture(t, b, i, s)
	f.boot = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	if _, err := b.frozenEnvironment(context.Background(), original); err == nil {
		t.Fatal("stale boot capsule received execution environment")
	}
	if _, err := b.lookup(context.Background(), i); err == nil {
		t.Fatal("old persisted intent claimed a newboot manager root")
	}
	f.present = false
	f.process = darwinOwnedProcess{}
	if err := b.release(context.Background(), original); err != nil {
		t.Fatal("oldboot extinction prevented exact retirement", err)
	}
}
