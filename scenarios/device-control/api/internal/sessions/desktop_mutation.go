package sessions

import "context"

type desktopMutationKey struct{}

// CheckDesktopMutation rechecks the controller's current act authority at a
// late native mutation boundary. Public request metadata cannot supply it.
// Native adapters must call synchronously within Apply, never retain it.
func CheckDesktopMutation(ctx context.Context) error {
	check, ok := ctx.Value(desktopMutationKey{}).(func(context.Context) error)
	if !ok || ctx.Err() != nil {
		return ErrDesktopAdmission
	}
	return check(ctx)
}
