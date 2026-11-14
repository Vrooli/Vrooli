//go:build darwin

package platform

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/sys/unix"
)

// Pin every directory from the filesystem root. Publication uses the retained
// parent descriptor, never a second pathname traversal into a mutable parent.
func darwinImmutableDirectories(ctx context.Context, path string, create bool) ([]*os.File, error) {
	if os.Getuid() != 0 || os.Geteuid() != 0 {
		return nil, ErrOperationIdentity
	}
	if path == "/var" || strings.HasPrefix(path, "/var/") {
		path = "/private" + path // the sole OS alias accepted by the native verifier
	}
	var held []*os.File
	fail := func(err error) ([]*os.File, error) {
		for _, file := range held {
			_ = file.Close()
		}
		return nil, err
	}
	fd, err := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	held = append(held, os.NewFile(uintptr(fd), "/"))
	current := "/"
	if err = darwinSystemBrokerCheck(current, true); err != nil {
		return fail(err)
	}
	for _, name := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
		if name == "" {
			continue
		}
		if err = ctx.Err(); err != nil {
			return fail(err)
		}
		parent := held[len(held)-1]
		created := false
		child, openErr := unix.Openat(int(parent.Fd()), name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
		if errors.Is(openErr, unix.ENOENT) && create {
			err = unix.Mkdirat(int(parent.Fd()), name, 0755)
			created = err == nil
			if err != nil && !errors.Is(err, unix.EEXIST) {
				return fail(err)
			}
			if err = parent.Sync(); err != nil {
				return fail(err)
			}
			child, openErr = unix.Openat(int(parent.Fd()), name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
		}
		if openErr != nil {
			return fail(openErr)
		}
		current = filepath.Join(current, name)
		file := os.NewFile(uintptr(child), current)
		held = append(held, file)
		// mkdirat honors the process umask. Only our newly created inode may
		// receive the public code-store traversal mode; existing directories
		// retain their actual access policy.
		if created {
			if err = file.Chmod(0755); err != nil {
				return fail(err)
			}
			if err = file.Sync(); err != nil {
				return fail(err)
			}
		}
		if err = darwinSystemBrokerCheck(current, true); err != nil {
			return fail(err)
		}
		a, statErr := file.Stat()
		b, nameErr := os.Lstat(current)
		if statErr != nil {
			return fail(statErr)
		}
		if nameErr != nil {
			return fail(nameErr)
		}
		if !os.SameFile(a, b) {
			return fail(ErrOperationIdentity)
		}
	}
	return held, nil
}

func prepareImmutableNativeDirectory(ctx context.Context, path string) (func() error, error) {
	held, err := darwinImmutableDirectories(ctx, path, true)
	if err != nil {
		return nil, err
	}
	var once sync.Once
	var result error
	return func() error {
		once.Do(func() {
			for n := len(held) - 1; n >= 0; n-- {
				result = errors.Join(result, held[n].Close())
			}
		})
		return result
	}, nil
}

func writeImmutableNativeFile(ctx context.Context, path string, write func(io.Writer) error) error {
	held, err := darwinImmutableDirectories(ctx, filepath.Dir(path), false)
	if err != nil {
		return err
	}
	defer func() {
		for _, file := range held {
			_ = file.Close()
		}
	}()
	parent := held[len(held)-1]
	var nonce [16]byte
	if _, err = rand.Read(nonce[:]); err != nil {
		return err
	}
	name := ".native-publish-" + hex.EncodeToString(nonce[:])
	fd, err := unix.Openat(int(parent.Fd()), name, unix.O_CREAT|unix.O_EXCL|unix.O_WRONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0600)
	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(fd), filepath.Join(filepath.Dir(path), name))
	defer file.Close()
	defer unix.Unlinkat(int(parent.Fd()), name, 0)
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = write(file); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = file.Chmod(0555); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return err
	}
	if err = darwinSystemBrokerCheck(file.Name(), false); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	// A single no-replace rename preserves the single-link invariant even if
	// the process dies at commit. A link/unlink pair leaves a rejected alias
	// after a crash between those calls.
	if err = unix.RenameatxNp(int(parent.Fd()), name, int(parent.Fd()), filepath.Base(path), unix.RENAME_EXCL); err != nil {
		return err
	}
	return parent.Sync()
}

func grantImmutableNativeCodeReaders(ctx context.Context, path string, readers []string) error {
	// This is the Windows selected-SID metadata ACL provisioning capability.
	// Darwin publication uses public root-owned 0755 code directories instead;
	// do not pretend a read-only pathname check applied a principal-specific ACL.
	return ErrUnsupported
}
