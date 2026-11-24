package storage

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
)

type objectReadBackend struct {
	StorageInterface
	label           string
	artifactReads   int
	screenshotReads int
	artifactErr     error
	screenshotErr   error
}

func (b *objectReadBackend) GetArtifact(context.Context, string) (io.ReadCloser, *minio.ObjectInfo, error) {
	b.artifactReads++
	if b.artifactErr != nil {
		return nil, nil, b.artifactErr
	}
	return io.NopCloser(strings.NewReader(b.label)), &minio.ObjectInfo{ContentType: "application/octet-stream", Size: int64(len(b.label))}, nil
}

func (b *objectReadBackend) GetScreenshot(context.Context, string) (io.ReadCloser, *minio.ObjectInfo, error) {
	b.screenshotReads++
	if b.screenshotErr != nil {
		return nil, nil, b.screenshotErr
	}
	return io.NopCloser(strings.NewReader(b.label)), &minio.ObjectInfo{ContentType: "image/png", Size: int64(len(b.label))}, nil
}

func TestResilientStoredObjectReadsShareFallbackPolicy(t *testing.T) {
	primaryFailure := errors.New("primary read failed")
	fallbackFailure := errors.New("fallback read failed")

	for _, op := range []struct {
		name            string
		read            func(*ResilientStorage) (io.ReadCloser, *minio.ObjectInfo, error)
		wantContentType string
	}{
		{
			name: "artifact",
			read: func(r *ResilientStorage) (io.ReadCloser, *minio.ObjectInfo, error) {
				return r.GetArtifact(context.Background(), "object")
			},
			wantContentType: "application/octet-stream",
		},
		{
			name: "screenshot",
			read: func(r *ResilientStorage) (io.ReadCloser, *minio.ObjectInfo, error) {
				return r.GetScreenshot(context.Background(), "object")
			},
			wantContentType: "image/png",
		},
	} {
		for _, scenario := range []struct {
			name              string
			useFallback       bool
			primaryErr        error
			fallbackErr       error
			wantPayload       string
			wantErr           error
			wantPrimaryReads  int
			wantFallbackReads int
		}{
			{name: "primary success", wantPayload: "primary", wantPrimaryReads: 1},
			{name: "primary failure uses fallback", primaryErr: primaryFailure, wantPayload: "fallback", wantPrimaryReads: 1, wantFallbackReads: 1},
			{name: "fallback preference", useFallback: true, wantPayload: "fallback", wantFallbackReads: 1},
			{name: "fallback preference recovers through primary", useFallback: true, fallbackErr: fallbackFailure, wantPayload: "primary", wantPrimaryReads: 1, wantFallbackReads: 1},
			{name: "fallback failure is returned", primaryErr: primaryFailure, fallbackErr: fallbackFailure, wantErr: fallbackFailure, wantPrimaryReads: 1, wantFallbackReads: 1},
		} {
			t.Run(op.name+"/"+scenario.name, func(t *testing.T) {
				primary := &objectReadBackend{label: "primary", artifactErr: scenario.primaryErr, screenshotErr: scenario.primaryErr}
				fallback := &objectReadBackend{label: "fallback", artifactErr: scenario.fallbackErr, screenshotErr: scenario.fallbackErr}
				resilient := NewResilientStorage(ResilientStorageConfig{Primary: primary, Fallback: fallback, Logger: logrus.New()})
				resilient.useFallback.Store(scenario.useFallback)

				reader, info, err := op.read(resilient)
				if op.name == "artifact" {
					if primary.artifactReads != scenario.wantPrimaryReads || fallback.artifactReads != scenario.wantFallbackReads || primary.screenshotReads != 0 || fallback.screenshotReads != 0 {
						t.Fatalf("artifact/screenshot getter calls = %d/%d and %d/%d, want artifact %d/%d and screenshot 0/0", primary.artifactReads, fallback.artifactReads, primary.screenshotReads, fallback.screenshotReads, scenario.wantPrimaryReads, scenario.wantFallbackReads)
					}
				} else if primary.screenshotReads != scenario.wantPrimaryReads || fallback.screenshotReads != scenario.wantFallbackReads || primary.artifactReads != 0 || fallback.artifactReads != 0 {
					t.Fatalf("screenshot/artifact getter calls = %d/%d and %d/%d, want screenshot %d/%d and artifact 0/0", primary.screenshotReads, fallback.screenshotReads, primary.artifactReads, fallback.artifactReads, scenario.wantPrimaryReads, scenario.wantFallbackReads)
				}
				if scenario.wantErr != nil {
					if !errors.Is(err, scenario.wantErr) {
						t.Fatalf("error = %v, want %v", err, scenario.wantErr)
					}
					return
				}
				if err != nil {
					t.Fatalf("read: %v", err)
				}
				defer reader.Close()
				got, readErr := io.ReadAll(reader)
				if readErr != nil {
					t.Fatalf("read body: %v", readErr)
				}
				if string(got) != scenario.wantPayload {
					t.Fatalf("body = %q, want %q", got, scenario.wantPayload)
				}
				if info == nil || info.ContentType != op.wantContentType {
					t.Fatalf("info = %#v, want content type %q", info, op.wantContentType)
				}
			})
		}
	}
}
