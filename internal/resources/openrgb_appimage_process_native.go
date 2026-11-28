//go:build darwin || windows

package resources

import (
	"context"
	"fmt"
	platform "github.com/vrooli/platform-go"
	"os"
	"sort"
)

func (o openRGBServingObservation) observeAppImageStage(ctx context.Context, stage openRGBAppImageStage, image os.FileInfo, imageSHA256 string) error {
	pid, err := o.servingPID()
	if err != nil {
		return err
	}
	args, err := platform.ProcessArguments(pid)
	if err != nil {
		return err
	}
	values, err := platform.ReadProcessEnvironment(pid)
	if err != nil {
		return err
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	environment := make([]string, 0, len(keys))
	for _, k := range keys {
		environment = append(environment, k+"="+values[k])
	}
	expected, err := stage.servingInvocation(o.base, int(o.endpoint.Port()))
	if err != nil {
		return err
	}
	matches, err := platform.ProcessWorkingDirectoryMatches(pid, expected.Dir)
	if err != nil || !matches {
		return fmt.Errorf("OpenRGB serving working directory does not match sealed artifact: %w", err)
	}
	directory := expected.Dir
	settings, err := readManagedUpdatePrivateFile(o.base.ConfigPath, managedUpdateConfigMaxBytes)
	if err != nil {
		return err
	}
	live := openRGBAppImageServingProcess{image: image, imageSHA256: imageSHA256, args: args, env: environment, dir: directory, config: settings}
	if err := o.verifyAppImageStage(ctx, stage, live, openRGBServingContext, observeOpenRGBHeadlessListener); err != nil {
		return err
	}
	if child := o.witness.ExecutionChild; child != nil {
		payload, err := stage.openPayload()
		if err != nil {
			return err
		}
		defer payload.Close()
		if err := platform.VerifyOwnedImageChild(ctx, o.ref, platform.OwnedImageChildIdentity{PID: child.PID, Incarnation: child.Incarnation, ImageID: child.ImageID, ImageHandle: child.ImageHandle, SecurityProof: child.SecurityProof, NativeAccount: child.NativeAccount, ContainmentID: child.ContainmentID}, payload); err != nil {
			return fmt.Errorf("OpenRGB serving child left its admitted Job/image/account: %w", err)
		}
	}
	_, err = o.servingPID()
	return err
}
