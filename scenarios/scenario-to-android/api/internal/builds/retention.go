package builds

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

// RetainArtifact publishes verified APK/AAB bytes under the request's owned
// data root. Build workspaces are disposable; returned release paths are not.
// Existing content is verified, never overwritten. No signing material is copied.
func RetainArtifact(ctx context.Context, dataRoot string, artifact deliveryramp.Artifact) (deliveryramp.Artifact, error) {
	const maxBytes int64 = 1 << 30
	digest := strings.TrimPrefix(artifact.Checksum, "sha256:")
	decoded, err := hex.DecodeString(digest)
	if err != nil || len(decoded) != sha256.Size || artifact.Checksum != "sha256:"+digest || strings.ToLower(digest) != digest || artifact.SizeBytes <= 0 || artifact.SizeBytes > maxBytes {
		return deliveryramp.Artifact{}, fmt.Errorf("retain Android artifact: invalid checksum or size")
	}
	if dataRoot == "" || artifact.Metadata["apk_path"] == "" || artifact.Metadata["aab_path"] == "" {
		return deliveryramp.Artifact{}, fmt.Errorf("retain Android artifact: data root and APK/AAB paths required")
	}
	if err := ctx.Err(); err != nil {
		return deliveryramp.Artifact{}, err
	}
	parent := filepath.Join(dataRoot, "builds", "artifacts")
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return deliveryramp.Artifact{}, err
	}
	target := filepath.Join(parent, digest)
	retained := artifact
	retained.Metadata = make(map[string]string, len(artifact.Metadata)+1)
	for key, value := range artifact.Metadata {
		retained.Metadata[key] = value
	}
	retained.LocalPath = filepath.Join(target, "app.apk")
	retained.Metadata["apk_path"] = retained.LocalPath
	retained.Metadata["aab_path"] = filepath.Join(target, "app.aab")
	retained.Metadata["retention"] = "scenario-data"
	verify := func(dir string) error {
		hash := sha256.New()
		var size int64
		for index, name := range []string{"app.apk", "app.aab"} {
			if index > 0 {
				_, _ = hash.Write([]byte{0})
			}
			n, err := copyArtifactFile(ctx, hash, filepath.Join(dir, name), maxBytes-size)
			if err != nil {
				return err
			}
			size += n
		}
		if size != artifact.SizeBytes || hex.EncodeToString(hash.Sum(nil)) != digest {
			return fmt.Errorf("retained Android artifact checksum/size mismatch")
		}
		return nil
	}
	if _, err := os.Stat(target); err == nil {
		if err := verify(target); err != nil {
			return deliveryramp.Artifact{}, err
		}
		return retained, nil
	} else if !os.IsNotExist(err) {
		return deliveryramp.Artifact{}, err
	}
	stage, err := os.MkdirTemp(parent, ".retain-")
	if err != nil {
		return deliveryramp.Artifact{}, err
	}
	defer os.RemoveAll(stage)
	var copiedBytes int64
	for index, source := range []string{artifact.Metadata["apk_path"], artifact.Metadata["aab_path"]} {
		name := []string{"app.apk", "app.aab"}[index]
		out, err := os.OpenFile(filepath.Join(stage, name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return deliveryramp.Artifact{}, err
		}
		n, copyErr := copyArtifactFile(ctx, out, source, artifact.SizeBytes-copiedBytes)
		copiedBytes += n
		closeErr := out.Close()
		if copyErr != nil {
			return deliveryramp.Artifact{}, copyErr
		}
		if closeErr != nil {
			return deliveryramp.Artifact{}, closeErr
		}
	}
	if err := verify(stage); err != nil {
		return deliveryramp.Artifact{}, err
	}
	manifest, err := json.MarshalIndent(retained, "", "  ")
	if err != nil {
		return deliveryramp.Artifact{}, err
	}
	if err := os.WriteFile(filepath.Join(stage, "artifact.json"), manifest, 0o600); err != nil {
		return deliveryramp.Artifact{}, err
	}
	if err := ctx.Err(); err != nil {
		return deliveryramp.Artifact{}, err
	}
	if err := os.Rename(stage, target); err != nil {
		// A concurrent identical build may have published first.
		if verifyErr := verify(target); verifyErr != nil {
			return deliveryramp.Artifact{}, fmt.Errorf("publish retained Android artifact: %w", err)
		}
	}
	return retained, nil
}

func copyArtifactFile(ctx context.Context, out io.Writer, path string, limit int64) (int64, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return 0, err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > limit {
		return 0, fmt.Errorf("Android artifact must be a bounded regular file")
	}
	in, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer in.Close()
	buffer := make([]byte, 64*1024)
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		n, readErr := in.Read(buffer)
		if n > 0 {
			total += int64(n)
			if total > limit {
				return total, fmt.Errorf("Android artifact exceeds retention limit")
			}
			written, err := out.Write(buffer[:n])
			if err != nil {
				return total, err
			}
			if written != n {
				return total, io.ErrShortWrite
			}
		}
		if readErr == io.EOF {
			return total, nil
		}
		if readErr != nil {
			return total, readErr
		}
	}
}
