package resources

import (
	"context"
	"errors"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"os"

	platform "github.com/vrooli/platform-go"
)

type openRGBNativeCapsuleSession interface {
	Prepare(context.Context) (openRGBNativeMaterial, platform.ServiceRef, error)
	Claim(context.Context, platform.ServiceRef, openRGBExecWitness) error
	Close() error
}
type openRGBNativePreparedChild interface {
	identity() openRGBNativeExecutionChild
	Resume() error
	Wait() error
	Abort() error
}
type openRGBNativeCapsuleExecOwner struct {
	lookup               func(context.Context, platform.ServiceIntent) (platform.ServiceRef, error)
	incarnation          func(int) (string, error)
	selfImage            func(platform.ServiceRef) (*os.File, error)
	normalizeEnvironment func(context.Context, platform.ServiceRef) error
	connect              func(context.Context, openRGBNativeCapsule) (openRGBNativeCapsuleSession, error)
	prepareImage         func(context.Context, openRGBNativeMaterial) (managedUpdateInvocation, *os.File, error)
	observe              func(context.Context, openRGBNativeMaterial, platform.ServiceRef, string) (openRGBExecWitness, error)
	confineTransport     func(context.Context, openRGBNativeMaterial) (string, error)
	prepareChild         func(*os.File, managedUpdateInvocation) (openRGBNativePreparedChild, error)
	verifyProcessOnly    func(int) error
	materialize          func(openRGBNativeMaterial, managedUpdateInvocation) error
	chdir                func(string) error
	replace              func(*os.File, []string, []string) error
}

// executeOpenRGBNativeCapsule never reads an operator ledger. An uncertain
// claim is terminal for this invocation: the owner retains its durable fence,
// and the wrapper aborts any child before that child can execute.
func executeOpenRGBNativeCapsule(ctx context.Context, c openRGBNativeCapsule, o openRGBNativeCapsuleExecOwner) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := c.validate(); err != nil {
		return err
	}
	ref, err := o.lookup(ctx, c.Intent)
	if err != nil {
		return err
	}
	incarnation, err := o.incarnation(os.Getpid())
	if err != nil {
		return err
	}
	if ref.ValidateIdentity() != nil || ref.Native == nil || ref.RootPID != os.Getpid() || ref.Intent != platform.OperationIntent(c.Intent) || ref.Native.Account != c.Account || !ref.MatchesIncarnation(incarnation) {
		return errors.New("native capsule is outside the exact admitted wrapper")
	}
	self, err := o.selfImage(ref)
	if err != nil {
		return err
	}
	digest, err := managedUpdateImageDigest(ctx, self)
	_ = self.Close()
	if err != nil {
		return err
	}
	if digest != c.LauncherSHA256 {
		return errors.New("native capsule launcher image changed")
	}
	if o.normalizeEnvironment != nil {
		if err = o.normalizeEnvironment(ctx, ref); err != nil {
			return err
		}
	}
	session, err := o.connect(ctx, c)
	if err != nil {
		return err
	}
	defer session.Close()
	material, observed, err := session.Prepare(ctx)
	if err != nil {
		return err
	}
	if !observed.Equal(ref) {
		return errors.New("native owner returned a different wrapper")
	}
	if err = material.validate(c); err != nil {
		return err
	}
	derived, image, err := o.prepareImage(ctx, material)
	if err != nil {
		return err
	}
	defer image.Close()
	if o.confineTransport == nil {
		return errors.New("native transport confinement owner is unavailable")
	}
	transportSHA, err := o.confineTransport(ctx, material)
	expectedTransport, recipeErr := hostreqkit.NativeOpenRGBTransportPolicySHA256(material.Policy.Platform)
	if err != nil || recipeErr != nil || transportSHA != expectedTransport {
		return errors.Join(errors.New("native transport confinement differs from the qualified recipe"), err, recipeErr)
	}
	witness, err := o.observe(ctx, material, ref, incarnation)
	if err != nil {
		return err
	}
	witness.TransportSHA256 = transportSHA
	if witness.Policy != material.Policy || witness.Incarnation != incarnation || witness.ClaimSHA256 != material.ClaimSHA256 {
		return errors.New("native capsule witness changed frozen claim")
	}
	var child openRGBNativePreparedChild
	childResumed := false
	if ref.Native.Platform == openRGBPlatformWindows {
		child, err = o.prepareChild(image, derived)
		if err != nil {
			return err
		}
		defer func() {
			if !childResumed {
				_ = child.Abort()
			}
		}()
	}
	if child != nil {
		identity := child.identity()
		if err = identity.validateFor(ref); err != nil {
			return err
		}
		witness.ExecutionChild = &identity
	}
	confirmed, err := o.lookup(ctx, c.Intent)
	if err != nil || !confirmed.Equal(ref) {
		return errors.Join(errors.New("native capsule owner changed during preparation"), err)
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = session.Claim(ctx, ref, witness); err != nil {
		return err
	}
	_ = session.Close()
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = o.materialize(material, derived); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if child == nil {
		if err = o.chdir(derived.Dir); err != nil {
			return err
		}
	}
	if child != nil {
		if err = child.Resume(); err != nil {
			return err
		}
		childResumed = true
		return child.Wait()
	}
	if err = verifyOpenRGBProcessOnlyOwner(ref, o.verifyProcessOnly); err != nil {
		return err
	}
	return o.replace(image, append([]string{derived.Path}, derived.Args...), derived.Env)
}

// A Darwin process-only receipt requires factual inherited no-child policy.
// Other native owners retain their own complete containment contract.
func verifyOpenRGBProcessOnlyOwner(ref platform.ServiceRef, verify func(int) error) error {
	if ref.Native == nil || !ref.Native.ProcessOnly {
		return nil
	}
	if ref.Native.Platform != openRGBPlatformDarwin || ref.RootPID <= 0 || verify == nil {
		return errors.New("native process-only ownership has no kernel verifier")
	}
	return verify(ref.RootPID)
}
