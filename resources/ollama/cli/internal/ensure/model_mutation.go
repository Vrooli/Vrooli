package ensure

import (
	"context"
	"fmt"
)

// ManagedModelMutation is the owner context for changing the installed model
// set outside a pull: the lifecycle lock held, the verified private client,
// and a re-check that the managed process is still the one inspected.
type ManagedModelMutation struct {
	Client    *Client
	ModelRoot string
	Verify    func(context.Context) error
	Release   func()
}

// PrepareManagedModelMutation holds Ollama's managed lifecycle lock (the same
// key start, restart, stop and missing-model pulls hold) and binds a client to
// the private listener of the exact process Vrooli owns. Release must be
// called when the mutation ends.
func PrepareManagedModelMutation(ctx context.Context) (ManagedModelMutation, error) {
	release, err := acquireManagedOllamaLifecycleLock(ctx)
	if err != nil {
		return ManagedModelMutation{}, fmt.Errorf("acquire managed Ollama lifecycle lock: %w", err)
	}
	root, err := effectiveModelRoot()
	if err != nil {
		release()
		return ManagedModelMutation{}, err
	}
	client, identity, err := resolveManagedOllamaClient(ctx, root)
	if err != nil {
		release()
		return ManagedModelMutation{}, err
	}
	return ManagedModelMutation{
		Client:    client,
		ModelRoot: identity.ModelRoot,
		Verify:    func(ctx context.Context) error { return verifyManagedOllama(ctx, identity) },
		Release:   release,
	}, nil
}

// EffectiveModelRoot is the managed model store path ensure admits pulls
// into, for owner reads such as manifest accounting.
func EffectiveModelRoot() (string, error) { return effectiveModelRoot() }
