package setup

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/operatorcapability"
	"github.com/vrooli/vrooli/internal/operatorinput"
)

// discoverAndQueueCapabilities is the generic setup handoff. Providers return
// typed descriptors and name only the inputs that still require operator
// authority. Setup persists descriptors and candidate metadata, never answer
// values, and remains safe to rerun because Enqueue replaces an existing
// request by stable ID.
func discoverAndQueueCapabilities(
	ctx context.Context,
	discover func(context.Context, string, string) ([]operatorcapability.Status, error),
	root, home string,
	stdout io.Writer,
) error {
	statuses, err := discover(ctx, root, home)
	if err != nil {
		return fmt.Errorf("discover operator capabilities: %w", err)
	}
	pending, err := operatorinput.Load()
	if err != nil {
		return fmt.Errorf("load existing operator inputs: %w", err)
	}
	existing := make(map[string]struct{}, len(pending.Requests))
	legacyStorePassphrasePending := false
	for _, request := range pending.Requests {
		if request.CapabilityID != "" && request.InputID != "" {
			existing[request.CapabilityID+":"+request.InputID] = struct{}{}
		}
		// Preserve the pre-contract queue ID while older setup producers are
		// still present. It maps to the same generic store-access action and
		// must not cause a duplicate prompt.
		if request.ID == "credential-store-passphrase" {
			existing["credential-store-access:passphrase"] = struct{}{}
			legacyStorePassphrasePending = true
		}
	}
	for _, status := range statuses {
		requests, requestErr := status.Descriptor.OperatorInputs()
		if requestErr != nil {
			return fmt.Errorf("prepare operator inputs for capability %q: %w", status.Descriptor.ID, requestErr)
		}
		missing := make(map[string]struct{}, len(status.MissingInputs))
		for _, id := range status.MissingInputs {
			missing[strings.TrimSpace(id)] = struct{}{}
		}
		for _, request := range requests {
			_, inputID, ok := strings.Cut(request.ID, ":")
			if !ok {
				return fmt.Errorf("capability %q produced malformed operator input ID %q", status.Descriptor.ID, request.ID)
			}
			if _, needed := missing[inputID]; !needed {
				continue
			}
			if legacyStorePassphrasePending && status.Descriptor.ID == "credential-store-access" && inputID == "confirm" {
				continue
			}
			if _, alreadyPending := existing[status.Descriptor.ID+":"+inputID]; alreadyPending {
				continue
			}
			if request.Kind == operatorinput.KindSecret && request.Default != "" {
				return fmt.Errorf("capability %q secret input %q has a persisted default", status.Descriptor.ID, inputID)
			}
			if err := operatorinput.Enqueue(request); err != nil {
				return fmt.Errorf("queue operator input %q: %w", request.ID, err)
			}
			existing[status.Descriptor.ID+":"+inputID] = struct{}{}
			_, _ = fmt.Fprintf(stdout, "[PENDING] %s requires operator input %s.\n", status.Descriptor.Title, request.ID)
		}
	}
	return nil
}
