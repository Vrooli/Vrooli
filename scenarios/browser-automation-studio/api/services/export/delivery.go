package export

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

// PublishedFile identifies complete bytes published without replacing another export.
type PublishedFile struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

func ValidateFilename(name string) error {
	if strings.TrimSpace(name) == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\\x00\r\n") || filepath.Base(name) != name {
		return fmt.Errorf("export filename must be one nonempty path component")
	}
	return nil
}

// PublishFile stages inside the destination filesystem and links complete bytes
// atomically. Link refuses existing files and symlinks rather than replacing them.
func PublishFile(ctx context.Context, directory, name string, produce func(string) error) (PublishedFile, error) {
	var result PublishedFile
	if err := ValidateFilename(name); err != nil {
		return result, err
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return result, err
	}
	if err = os.MkdirAll(absolute, 0755); err != nil {
		return result, err
	}
	root, err := os.OpenRoot(absolute)
	if err != nil {
		return result, err
	}
	defer root.Close()
	stage := ".bas-export-" + uuid.NewString() + filepath.Ext(name)
	file, err := root.OpenFile(stage, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		return result, err
	}
	if err = file.Close(); err != nil {
		_ = root.Remove(stage)
		return result, err
	}
	defer root.Remove(stage)
	if err = produce(filepath.Join(absolute, stage)); err != nil {
		return result, err
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	stagedInfo, err := root.Lstat(stage)
	if err != nil {
		return result, err
	}
	if !stagedInfo.Mode().IsRegular() {
		return result, errors.New("export producer replaced staging file with a non-regular file")
	}
	file, err = root.OpenFile(stage, os.O_RDWR, 0)
	if err != nil {
		return result, err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return result, err
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		file.Close()
		return result, errors.New("export producer must supply a nonempty regular media file")
	}
	hash := sha256.New()
	_, err = io.Copy(hash, &contextReader{ctx: ctx, reader: file})
	if err == nil {
		err = file.Sync()
	}
	err = errors.Join(err, file.Close())
	if err != nil {
		return result, err
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	if err = root.Link(stage, name); err != nil {
		return result, fmt.Errorf("publish export without replacing existing output: %w", err)
	}
	return PublishedFile{Path: filepath.Join(absolute, name), Size: info.Size(), SHA256: hex.EncodeToString(hash.Sum(nil))}, nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

// CopyMedia checks cancellation while copying into an owned staging path.
func CopyMedia(ctx context.Context, source, destination string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	_, err = io.Copy(output, &contextReader{ctx: ctx, reader: input})
	return errors.Join(err, output.Close())
}

type CompletionRepository interface {
	UpdateExportComplete(context.Context, uuid.UUID, string, int64, database.JSONMap) error
}

// CommitExport never claims success when persistence fails. Verified published
// bytes survive with a recoverable receipt; callers emit completion only on nil.
func CommitExport(ctx context.Context, repo CompletionRepository, row *database.ExportIndex, media PublishedFile) error {
	if row.Settings == nil {
		row.Settings = database.JSONMap{}
	}
	row.Settings["output_sha256"] = media.SHA256
	err := ctx.Err()
	if err == nil {
		err = repo.UpdateExportComplete(ctx, row.ID, media.Path, media.Size, row.Settings)
	}
	if err == nil {
		return nil
	}
	receipt := struct {
		ExportID uuid.UUID        `json:"export_id"`
		Media    PublishedFile    `json:"media"`
		Settings database.JSONMap `json:"settings"`
		Error    string           `json:"error"`
	}{row.ID, media, row.Settings, err.Error()}
	data, marshalErr := json.MarshalIndent(receipt, "", "  ")
	if marshalErr != nil {
		return errors.Join(err, fmt.Errorf("marshal export recovery receipt: %w", marshalErr))
	}
	// Recovery is also published atomically and never replaces prior evidence.
	recoveryCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	_, writeErr := PublishFile(recoveryCtx, filepath.Dir(media.Path), filepath.Base(media.Path)+".recovery.json", func(path string) error { return os.WriteFile(path, data, 0600) })
	if writeErr != nil {
		return errors.Join(err, fmt.Errorf("retain export recovery receipt: %w", writeErr))
	}

	return fmt.Errorf("export file retained at %s; persistence failed: %w", media.Path, err)
}
