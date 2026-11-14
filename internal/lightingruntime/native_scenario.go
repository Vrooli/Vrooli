package lightingruntime

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/lightingbinding"
	"github.com/vrooli/api-core/runtimediagnostics"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/runtimeprofile"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

var errNoLightingScenario = errors.New("no current Light Desk API launch")

type nativeScenarioOwner struct {
	mu            sync.Mutex
	ensureStarted func(context.Context) error
	resolve       func() (*nativeScenarioTarget, error)
}

type nativeScenarioTarget struct {
	identity lightingbinding.Identity
	socket   string
	verify   func() error
	close    func()
}

// NewNativeScenarioOwner composes the existing scenario lifecycle with the
// operator-private binding transport. ensureStarted must start/ensure only the
// canonical lighting-control instance through the lifecycle owner. No browser
// token or endpoint supplied by a browser participates in this protocol.
func NewNativeScenarioOwner(home string, ensureStarted func(context.Context) error) (ScenarioOwner, error) {
	if ensureStarted == nil {
		return nil, errors.New("Light Desk lifecycle ensure callback is required")
	}
	directory, err := runtimeprofile.Directory(home)
	if err != nil {
		return nil, err
	}
	return &nativeScenarioOwner{ensureStarted: ensureStarted, resolve: func() (*nativeScenarioTarget, error) { return resolveNativeScenario(home, directory) }}, nil
}

func (o *nativeScenarioOwner) EnsureLightingReconciler(ctx context.Context, backend Backend) error {
	binding := lightingbinding.Binding{Address: backend.Endpoint.String(), Incarnation: backend.Incarnation}
	if err := binding.Validate(); err != nil {
		return err
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if err := o.ensureStarted(ctx); err != nil {
		return fmt.Errorf("ensure Light Desk lifecycle: %w", err)
	}
	return o.apply(ctx, &binding)
}

func (o *nativeScenarioOwner) QuiesceLightingReconciler(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	err := o.apply(ctx, nil)
	if errors.Is(err, errNoLightingScenario) {
		return nil
	}
	return err
}

func (o *nativeScenarioOwner) apply(ctx context.Context, binding *lightingbinding.Binding) error {
	target, err := o.resolve()
	if err != nil {
		return err
	}
	defer target.close()
	client := lightingbinding.NewClient(target.socket, target.identity.PID)
	defer client.Close()
	state, err := client.State(ctx)
	if err != nil {
		return fmt.Errorf("read exact Light Desk native owner: %w", err)
	}
	if err := validateNativeState(state, target.identity); err != nil {
		return err
	}
	if err := target.verify(); err != nil {
		return err
	}
	if binding != nil && state.Binding != nil && *binding == *state.Binding && !state.ReconcilePending {
		return nil
	}
	request := lightingbinding.Request{Identity: target.identity, ExpectedRevision: state.Revision, OperationID: uuid.NewString(), Binding: binding}
	// Retry only this exact request after an uncertain transport result. A
	// different writer/revision is never silently adopted during an operation.
	var receipt lightingbinding.Receipt
	for attempt := 0; attempt < 2; attempt++ {
		if err := target.verify(); err != nil {
			return err
		}
		receipt, err = client.Apply(ctx, request)
		if err == nil || errors.Is(err, lightingbinding.ErrConflict) || ctx.Err() != nil {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("apply exact Light Desk native owner: %w", err)
	}
	if err := validateNativeState(receipt.State, target.identity); err != nil {
		return err
	}
	if receipt.OperationID != request.OperationID || receipt.State.Revision != request.ExpectedRevision+1 || !sameNativeBinding(receipt.State.Binding, binding) {
		return errors.New("Light Desk native receipt does not match requested binding")
	}
	return target.verify()
}

func validateNativeState(state lightingbinding.State, identity lightingbinding.Identity) error {
	if state.Version != lightingbinding.Version || state.Identity != identity {
		return errors.New("Light Desk native component identity mismatch")
	}
	if state.Binding != nil {
		return state.Binding.Validate()
	}
	return nil
}
func sameNativeBinding(a, b *lightingbinding.Binding) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func resolveNativeScenario(home, directory string) (*nativeScenarioTarget, error) {
	key := scenarioruntime.InstanceKey{Scenario: "lighting-control", Variant: scenarioruntime.DefaultVariant}
	records, err := process.ReadScenarioRecords(home, key.Slug())
	if err != nil {
		return nil, err
	}
	record, err := selectLightingRecord(records)
	if err != nil {
		return nil, err
	}
	handle, err := process.PinRecord(record)
	if err != nil {
		return nil, fmt.Errorf("pin Light Desk API owner: %w", err)
	}
	fail := func(err error) (*nativeScenarioTarget, error) { _ = handle.Close(); return nil, err }
	env, err := process.ReadEnvironment(record.PID)
	if err != nil {
		return fail(err)
	}
	if record.RuntimeInstanceID == "" || env["VROOLI_RUNTIME_INSTANCE_ID"] != record.RuntimeInstanceID || env["VROOLI_SCENARIO"] != key.Scenario || env["VROOLI_VARIANT"] != key.Variant || env[runtimediagnostics.DirectoryEnv] != directory {
		return fail(errors.New("Light Desk private owner does not match the recorded native launch"))
	}
	identity := lightingbinding.Identity{Scenario: key.Scenario, Variant: key.Variant, RuntimeInstanceID: record.RuntimeInstanceID, Generation: record.Generation, PID: record.PID}
	verify := func() error {
		alive, err := handle.Alive()
		if err != nil || !alive {
			return errors.New("Light Desk pinned owner exited")
		}
		incarnation, err := process.Incarnation(record.PID)
		if err != nil || incarnation != record.Incarnation {
			return errors.New("Light Desk process incarnation changed")
		}
		current, err := process.ReadScenarioRecords(home, key.Slug())
		if err != nil {
			return err
		}
		selected, err := selectLightingRecord(current)
		if err != nil {
			return err
		}
		if selected.Generation != record.Generation || selected.Incarnation != record.Incarnation || selected.PID != record.PID || selected.RuntimeInstanceID != record.RuntimeInstanceID {
			return errors.New("Light Desk native launch replaced during binding")
		}
		return nil
	}
	return &nativeScenarioTarget{identity: identity, socket: lightingbinding.SocketPath(lightingbinding.Directory(directory), identity), verify: verify, close: func() { _ = handle.Close() }}, nil
}

func selectLightingRecord(records []process.Record) (process.Record, error) {
	var matches []process.Record
	for _, record := range records {
		if record.Step == "start-api" && record.Status != process.RecordStatusExitedVerified {
			matches = append(matches, record)
		}
	}
	if len(matches) == 0 {
		return process.Record{}, errNoLightingScenario
	}
	if len(matches) != 1 {
		return process.Record{}, errors.New("Light Desk requires exactly one current native API launch")
	}
	return matches[0], nil
}
