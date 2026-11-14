package runtimeapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/resources"
)

// The only path input is the trusted platform install result. Actual source
// bytes come from the currently mapped executable, never a caller nomination.
type nativeLightingLauncherEffects struct {
	sealedDigest func(context.Context, string) (string, error)
	stage        func(context.Context) (string, error)
}

func prepareNativeLightingLauncher(ctx context.Context, c *resources.Controller, installed string) error {
	return prepareNativeLightingLauncherWith(ctx, installed, nativeLightingLauncherEffects{sealedDigest: func(ctx context.Context, path string) (string, error) {
		if err := platform.VerifyImmutableNativePath(ctx, path, false); err != nil {
			return "", err
		}
		file, err := os.Open(path)
		if err != nil {
			return "", err
		}
		defer file.Close()
		return nativeLightingCLIImageDigest(ctx, file)
	}, stage: c.PrepareOpenRGBNativeLauncher})
}

func prepareNativeLightingLauncherWith(ctx context.Context, installed string, e nativeLightingLauncherEffects) error {
	if installed == "" {
		return errors.New("native setup returned no sealed service host")
	}
	expected, err := e.sealedDigest(ctx, installed)
	if err != nil {
		return err
	}
	path, err := e.stage(ctx)
	if err != nil {
		return err
	}
	digest, err := e.sealedDigest(ctx, path)
	if err != nil {
		return err
	}
	if digest != expected {
		return errors.New("native launcher differs from the exact installed service host")
	}
	return nil
}

func nativeLightingCLIImageDigest(ctx context.Context, file *os.File) (string, error) {
	if ctx == nil || file == nil {
		return "", errors.New("native image digest requires context and owned file")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 512<<20 {
		return "", errors.New("native service host image is not a bounded regular file")
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	hash := sha256.New()
	reader := &nativeLightingContextReader{ctx: ctx, reader: io.LimitReader(file, 512<<20+1)}
	n, err := io.Copy(hash, reader)
	if err != nil {
		return "", err
	}
	if n <= 0 || n > 512<<20 || n != info.Size() {
		return "", fmt.Errorf("native service host exceeds image bound")
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

type nativeLightingContextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *nativeLightingContextReader) Read(data []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(data)
}
