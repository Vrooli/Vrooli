package platform

import (
	"context"
	"errors"
	"os"
)

// OwnedImageChild is a prepared, still-suspended image inside an existing
// owned native service tree. Persist all public facts before Resume. Wait
// observes this child only; service-tree quiescence remains the service owner.
// Abort is permitted only before Resume, when no child code has executed.
type OwnedImageChild struct {
	PID           int
	Incarnation   string
	ImageID       string
	ImageHandle   uint64
	NativeAccount string
	ContainmentID string
	SecurityProof string
	resume        func() error
	wait          func() error
	abort         func() error
}

func PrepareOwnedImageChild(file *os.File, argv, env []string, dir string) (*OwnedImageChild, error) {
	if file == nil || len(argv) == 0 || env == nil {
		return nil, errors.New("platform: verified file, argv and explicit environment required")
	}
	return prepareOwnedImageChild(file, argv, env, dir)
}
func (c *OwnedImageChild) Resume() error {
	if c == nil || c.resume == nil {
		return errors.New("platform: invalid prepared image child")
	}
	return c.resume()
}
func (c *OwnedImageChild) Wait() error {
	if c == nil || c.wait == nil {
		return errors.New("platform: invalid prepared image child")
	}
	return c.wait()
}
func (c *OwnedImageChild) Abort() error {
	if c == nil || c.abort == nil {
		return errors.New("platform: invalid prepared image child")
	}
	return c.abort()
}

// OwnedImageChildIdentity is the persisted pre-resume child claim. Verification
// observes kernel facts independently; these values are never PID authority.
type OwnedImageChildIdentity struct {
	PID                                                               int
	ImageHandle                                                       uint64 `json:",omitempty"`
	Incarnation, ImageID, NativeAccount, ContainmentID, SecurityProof string
}

func (c *OwnedImageChild) Identity() OwnedImageChildIdentity {
	if c == nil {
		return OwnedImageChildIdentity{}
	}
	return OwnedImageChildIdentity{PID: c.PID, Incarnation: c.Incarnation, ImageID: c.ImageID, ImageHandle: c.ImageHandle, NativeAccount: c.NativeAccount, ContainmentID: c.ContainmentID, SecurityProof: c.SecurityProof}
}
func VerifyOwnedImageChild(ctx context.Context, ref ServiceRef, id OwnedImageChildIdentity, image *os.File) error {
	if ctx == nil || image == nil || ref.ValidateIdentity() != nil || id.PID <= 0 || id.PID == ref.RootPID {
		return ErrOperationIdentity
	}
	if e := ctx.Err(); e != nil {
		return e
	}
	return verifyOwnedImageChild(ctx, ref, id, image)
}
