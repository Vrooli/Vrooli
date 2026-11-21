package hostwireless

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/packages/capacity"
)

func publishBluetoothFirmware(ctx context.Context, r BluetoothFirmwareQualification, payload []byte, stage string) (_ BluetoothFirmwareQualification, resultErr error) {
	artifactRoot := filepath.Dir(filepath.Dir(stage))
	parent := filepath.Join(artifactRoot, "host-wireless-firmware", "asus-mt6639")
	final := filepath.Join(parent, r.ArchiveSHA256)
	published := r
	published.State, published.ArtifactDirectory = localFirmwareArtifactState, final
	receipt, err := json.MarshalIndent(published, "", "  ")
	if err != nil {
		return r, err
	}
	broker, err := capacity.NewBroker(ctx, "")
	if err != nil {
		return r, err
	}
	defer broker.Close()
	lease, err := broker.ReserveDisk(ctx, "host-wireless:asus-mt6639", artifactRoot, bluetoothPublicationPeak, 0)
	if err != nil {
		return r, err
	}
	clean := true
	defer func() {
		if !clean {
			return
		}
		done, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resultErr = errors.Join(resultErr, lease.Release(done))
	}()
	unlock, err := artifactlock.AcquireWithPathContext(ctx, "asus-mt6639-local-firmware", func(string) (string, error) {
		return filepath.Join(parent, ".qualification.lock"), nil
	})
	if err != nil {
		return r, err
	}
	defer unlock()
	if info, err := os.Lstat(final); err == nil {
		if !info.IsDir() {
			return r, errors.New("local firmware artifact path is not a directory")
		}
		if err := verifyBluetoothArtifact(ctx, final, receipt); err != nil {
			return r, err
		}
		return published, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return r, err
	}
	work, err := os.MkdirTemp(parent, ".qualification-")
	if err != nil {
		return r, err
	}
	defer func() {
		err := os.RemoveAll(work)
		clean = err == nil
		resultErr = errors.Join(resultErr, err)
	}()
	if len(payload) != bluetoothFirmwareSize || digest(payload) != bluetoothFirmwareSHA {
		return r, errors.New("local firmware publication payload differs from its pin")
	}
	if err := os.WriteFile(filepath.Join(work, bluetoothFirmwareName), payload, 0600); err != nil {
		return r, err
	}
	if err := os.WriteFile(filepath.Join(work, "receipt.json"), receipt, 0600); err != nil {
		return r, err
	}
	if err := ctx.Err(); err != nil {
		return r, err
	}
	if err := verifyBluetoothArtifact(ctx, work, receipt); err != nil {
		return r, err
	}
	if err := os.Rename(work, final); err != nil {
		return r, err
	}
	return published, nil
}

func verifyBluetoothArtifact(ctx context.Context, directory string, receipt []byte) error {
	root, err := os.OpenRoot(directory)
	if err != nil {
		return err
	}
	defer root.Close()
	entries, err := fs.ReadDir(root.FS(), ".")
	if err != nil {
		return err
	}
	if len(entries) != 2 {
		return errors.New("local firmware artifact has an unexpected file set")
	}
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			return fmt.Errorf("unexpected local firmware entry %s", entry.Name())
		}
	}
	if err := verifyFile(ctx, bundleFiles{root}, bluetoothFirmwareName, bluetoothFirmwareSHA); err != nil {
		return err
	}
	data, err := readSmallRegular(bundleFiles{root}, "receipt.json", 64<<10)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, receipt) {
		return errors.New("local firmware qualification receipt changed")
	}
	return nil
}
