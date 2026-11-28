//go:build linux

package resources

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	platform "github.com/vrooli/platform-go"
)

// Called by the existing native role observer with its already verified image
// handle facts. All process reads target the exact retained native root.
func (o openRGBServingObservation) observeAppImageStage(ctx context.Context, stage openRGBAppImageStage, image os.FileInfo, imageSHA256 string) error {
	procDir := filepath.Join("/proc", strconv.Itoa(o.ref.RootPID))
	args, err := platform.ProcessArguments(o.ref.RootPID)
	if err != nil {
		return err
	}
	environment, err := readOpenRGBProcessFile(filepath.Join(procDir, "environ"), 64<<10)
	if err != nil {
		return err
	}
	if len(environment) == 0 || environment[len(environment)-1] != 0 {
		return errors.New("OpenRGB serving environment is incomplete")
	}
	env := strings.Split(string(environment[:len(environment)-1]), "\x00")
	dir, err := os.Readlink(filepath.Join(procDir, "cwd"))
	if err != nil {
		return err
	}
	// Read through the process's mount view. The preceding claim and following
	// context check bind that view to the witnessed mount/security namespace.
	configPath := filepath.Join(procDir, "root", o.base.ConfigPath)
	config, err := readOpenRGBProcessFile(configPath, managedUpdateConfigMaxBytes)
	if err != nil {
		return err
	}
	live := openRGBAppImageServingProcess{image: image, imageSHA256: imageSHA256, args: args, env: env, dir: dir, config: config}
	return o.verifyAppImageStage(ctx, stage, live, openRGBServingContext, observeOpenRGBHeadlessListener)
}

func readOpenRGBProcessFile(path string, maxBytes int) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, int64(maxBytes)+1))
	if err != nil || len(data) > maxBytes {
		return nil, errors.Join(fmt.Errorf("OpenRGB process evidence is unreadable or exceeds %d bytes", maxBytes), err)
	}
	return data, nil
}
