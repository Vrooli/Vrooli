package storage

import (
	"io"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewScreenshotStorage_DefaultsToLocal(t *testing.T) {
	t.Setenv("BAS_SCREENSHOT_STORAGE", "")
	root := t.TempDir()
	log := logrus.New()
	log.SetOutput(io.Discard)

	store := NewScreenshotStorage(log, root)

	assert.IsType(t, &FileStorage{}, store)
}

func TestNewScreenshotStorage_FallsBackWhenMinIOFails(t *testing.T) {
	t.Setenv("BAS_SCREENSHOT_STORAGE", "minio")
	t.Setenv("MINIO_ENDPOINT", "")
	t.Setenv("MINIO_ACCESS_KEY", "")
	t.Setenv("MINIO_SECRET_KEY", "")
	t.Setenv("MINIO_BUCKET_NAME", "")
	root := t.TempDir()
	log := logrus.New()
	log.SetOutput(io.Discard)

	store := NewScreenshotStorage(log, root)

	assert.IsType(t, &FileStorage{}, store)
}
