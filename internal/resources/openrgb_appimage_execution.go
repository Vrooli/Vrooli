package resources

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
)

// prepareOpenRGBAppImageExecution runs inside the existing native launcher,
// after native self-ownership checks and before consuming the device witness.
func (c *Controller) prepareOpenRGBAppImageExecution(ctx context.Context, artifact ManagedServiceArtifact, base managedUpdateInvocation, port int) (openRGBAppImageStage, managedUpdateInvocation, *os.File, error) {
	verified, err := c.verifyOpenRGBRoleArtifact(ctx, artifact)
	if err != nil {
		return openRGBAppImageStage{}, managedUpdateInvocation{}, nil, err
	}
	stage := *verified
	invocation, err := stage.servingInvocation(base, port)
	if err != nil {
		return openRGBAppImageStage{}, managedUpdateInvocation{}, nil, err
	}
	file, err := stage.openPayload()
	if err != nil {
		return openRGBAppImageStage{}, managedUpdateInvocation{}, nil, err
	}
	return stage, invocation, file, nil
}

func (c *Controller) verifyOpenRGBRoleArtifact(ctx context.Context, artifact ManagedServiceArtifact) (*openRGBAppImageStage, error) {
	profile, profileErr := openRGBReleaseProfileFor(runtime.GOOS, runtime.GOARCH)
	if profileErr != nil || !profile.acceptsArtifactSHA256(artifact.SHA256) || artifact.Version != openRGBAppImageVersion {
		return nil, errOpenRGBReleasePayloadUnqualified
	}
	appDir, err := c.OpenRGBAppImageStagePath(artifact)
	if err != nil {
		return nil, err
	}
	retained := artifact
	retained.Path = filepath.Join(filepath.Dir(appDir), openRGBSealedEnvelope)
	stage, err := verifyOpenRGBAppImageStage(ctx, retained, appDir)
	stage.envelope = artifact
	return &stage, err
}

func (c *Controller) openOpenRGBRollbackEnvelope(ctx context.Context, artifact ManagedServiceArtifact) (*os.File, error) {
	if _, err := c.verifyOpenRGBRoleArtifact(ctx, artifact); err != nil {
		return nil, err
	}
	appDir, err := c.OpenRGBAppImageStagePath(artifact)
	if err != nil {
		return nil, err
	}
	return os.Open(filepath.Join(filepath.Dir(appDir), openRGBSealedEnvelope))
}
