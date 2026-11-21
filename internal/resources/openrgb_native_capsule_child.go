package resources

import (
	"context"
	"errors"
	platform "github.com/vrooli/platform-go"
	"os"
	"path/filepath"
)

func prepareOpenRGBNativeCapsuleImage(ctx context.Context, m openRGBNativeMaterial) (managedUpdateInvocation, *os.File, error) {
	retained := m.Contract.Artifact
	retained.Path = filepath.Join(filepath.Dir(m.StagePath), openRGBSealedEnvelope)
	stage, err := verifyOpenRGBAppImageStage(ctx, retained, m.StagePath)
	if err != nil {
		return managedUpdateInvocation{}, nil, err
	}
	if err = verifyOpenRGBReleaseRuntimePrerequisites(ctx, stage.appDir); err != nil {
		return managedUpdateInvocation{}, nil, err
	}
	stage.envelope = m.Contract.Artifact
	invocation, err := stage.servingInvocation(m.Invocation, m.Runtime.Port)
	if err != nil {
		return managedUpdateInvocation{}, nil, err
	}
	file, err := stage.openPayload()
	if err != nil {
		return managedUpdateInvocation{}, nil, err
	}
	return invocation, file, nil
}

// The owner reopens and verifies its frozen payload independently of request
// data, then validates the held kernel child identity and exact native Job.
func verifyOpenRGBNativeExecutionChild(ctx context.Context, m openRGBNativeMaterial, ref platform.ServiceRef, w openRGBExecWitness) error {
	if ref.Native == nil {
		return errors.New("native execution child has no native owner")
	}
	if ref.Native.Platform != "windows" {
		if w.ExecutionChild != nil {
			return errors.New("non-Windows execution has an unadmitted child")
		}
		return nil
	}
	if w.ExecutionChild == nil {
		return errors.New("Windows execution child was not persisted")
	}
	c := *w.ExecutionChild
	if err := c.validateFor(ref); err != nil {
		return err
	}
	_, image, err := prepareOpenRGBNativeCapsuleImage(ctx, m)
	if err != nil {
		return err
	}
	defer image.Close()
	return platform.VerifyOwnedImageChild(ctx, ref, platform.OwnedImageChildIdentity{PID: c.PID, Incarnation: c.Incarnation, ImageID: c.ImageID, ImageHandle: c.ImageHandle, SecurityProof: c.SecurityProof, NativeAccount: c.NativeAccount, ContainmentID: c.ContainmentID}, image)
}
