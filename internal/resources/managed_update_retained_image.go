package resources

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/tuning"
)

// RetainManagedServiceRollbackArtifact retains the actual serving executable's
// bytes under the resource artifact owner. It neither stops the process nor
// adopts a launch contract or admits an update. The current policy must support
// the operation; an old pathname or a newer desired artifact is not a substitute
// for the running image opened through the native process observer.
func (c *Controller) RetainManagedServiceRollbackArtifact(ctx context.Context, name string) (ManagedServiceArtifact, error) {
	if ctx == nil || c == nil {
		return ManagedServiceArtifact{}, fmt.Errorf("rollback artifact retention requires a controller and context")
	}
	manifest, err := c.managedServiceArtifactManifest(name)
	if err != nil {
		return ManagedServiceArtifact{}, err
	}
	if manifest.ManagedService.UpdatePolicy == nil {
		return ManagedServiceArtifact{}, fmt.Errorf("resource has no explicit managed update policy")
	}
	if err := manifest.ManagedService.ValidateUpdatePolicy(); err != nil {
		return ManagedServiceArtifact{}, err
	}
	if strings.EqualFold(manifest.ManagedService.Artifact.Layout, "dir") {
		return ManagedServiceArtifact{}, fmt.Errorf("live legacy retention requires a single-file executable; directory launch adoption is not qualified")
	}
	release, err := acquireManagedLifecycleLocks(ctx, manifest)
	if err != nil {
		return ManagedServiceArtifact{}, err
	}
	defer release()
	if err := ensureManagedUpdateLifecycleAvailable(name, "rollback artifact retention"); err != nil {
		return ManagedServiceArtifact{}, err
	}
	supervisor, _, err := managedServiceSupervisorFor(name)
	if err != nil {
		return ManagedServiceArtifact{}, err
	}
	state, running, err := supervisor.Status()
	if err != nil {
		return ManagedServiceArtifact{}, fmt.Errorf("read serving ownership for rollback retention: %w", err)
	}
	if !running {
		return ManagedServiceArtifact{}, fmt.Errorf("serving ownership is unavailable for rollback retention")
	}
	incarnation, err := process.Incarnation(state.PID)
	if err != nil {
		return ManagedServiceArtifact{}, fmt.Errorf("serving process incarnation is unavailable: %w", err)
	}
	if incarnation == "" {
		return ManagedServiceArtifact{}, fmt.Errorf("serving process incarnation is unavailable")
	}
	observed := state
	observed.Incarnation = incarnation
	if err := verifyManagedServiceOwnership(observed); err != nil {
		return ManagedServiceArtifact{}, err
	}
	image, err := platform.OpenProcessExecutable(state.PID)
	if err != nil {
		return ManagedServiceArtifact{}, err
	}
	defer image.Close()
	artifact, err := retainManagedUpdateFile(ctx, c, name, state.ArtifactVersion, state.ArtifactSHA256, image)
	if err != nil {
		return ManagedServiceArtifact{}, err
	}
	// A successful copy is insufficient if the serving identity changed while
	// it was read. Keep any verified backup, but do not acknowledge admission.
	after, live, err := supervisor.Status()
	if err != nil || !live || !reflect.DeepEqual(after, state) {
		return ManagedServiceArtifact{}, fmt.Errorf("serving identity changed during rollback retention")
	}
	if err := verifyManagedServiceOwnership(observed); err != nil {
		return ManagedServiceArtifact{}, err
	}
	if err := ctx.Err(); err != nil {
		return ManagedServiceArtifact{}, err
	}
	return artifact, nil
}

// Caller holds the resource lifecycle lock. This then takes the acquisition
// lock, so staging, pruning and retention cannot replace each other's bytes.
func retainManagedUpdateFile(ctx context.Context, c *Controller, name, version, digest string, image *os.File) (ManagedServiceArtifact, error) {
	if !managedUpdateResource.MatchString(name) || !managedUpdateKey.MatchString(version) || !updateDigest(digest) || image == nil {
		return ManagedServiceArtifact{}, fmt.Errorf("retention requires a complete resource release identity and opened image")
	}
	info, err := image.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 {
		return ManagedServiceArtifact{}, fmt.Errorf("retention source is not a nonempty regular executable")
	}
	root, err := managedServiceArtifactStoreRoot(c.Home)
	if err != nil {
		return ManagedServiceArtifact{}, err
	}
	path := filepath.Join(root, name, version, ".rollback", digest)
	release, err := artifactlock.AcquireContext(ctx, "resource:"+name)
	if err != nil {
		return ManagedServiceArtifact{}, err
	}
	defer release()
	if _, err := image.Seek(0, io.SeekStart); err != nil {
		return ManagedServiceArtifact{}, err
	}
	if err := config.WriteOwnedFileDurableStream(path, tuning.PermExecutable, func(dst io.Writer) error {
		hash := sha256.New()
		source := &managedUpdateContextReader{ctx: ctx, reader: image}
		written, err := io.CopyBuffer(io.MultiWriter(dst, hash), &io.LimitedReader{R: source, N: info.Size()}, make([]byte, 64*1024))
		if err != nil {
			return err
		}
		var extra [1]byte
		n, end := source.Read(extra[:])
		if end != nil && end != io.EOF {
			return end
		}
		if written != info.Size() || n != 0 || end != io.EOF || hex.EncodeToString(hash.Sum(nil)) != digest {
			return fmt.Errorf("running image changed or differs from its retained checksum")
		}
		return ctx.Err()
	}); err != nil {
		return ManagedServiceArtifact{}, fmt.Errorf("retain managed rollback image: %w", err)
	}
	return ManagedServiceArtifact{Resource: name, Path: path, Version: version, SHA256: digest}, nil
}

type managedUpdateContextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *managedUpdateContextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := r.reader.Read(p)
	if canceled := r.ctx.Err(); canceled != nil {
		return n, canceled
	}
	return n, err
}
