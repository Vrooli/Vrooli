package resources

import (
	"context"
	"fmt"

	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
)

type ManagedServiceUpdateInspection struct {
	Resource            string                  `json:"resource"`
	Eligibility         string                  `json:"eligibility"`
	DeclarationEligible bool                    `json:"declaration_eligible"`
	Reasons             []string                `json:"reasons"`
	Unknown             []string                `json:"unknown"`
	Desired             *ManagedServiceArtifact `json:"desired,omitempty"`
	Running             *ManagedServiceState    `json:"running,omitempty"`
	Operation           *ManagedServiceUpdate   `json:"operation,omitempty"`
}

// Inspection observes current artifact/process/ledger state; it neither adopts
// a legacy launch nor resolves credentials, retains bytes or renews a lease.
// Eligibility here is declaration eligibility, not update admission or a fresh
// native/remote safety proof.
func (c *Controller) InspectManagedServiceUpdate(ctx context.Context, name string) (ManagedServiceUpdateInspection, error) {
	r := ManagedServiceUpdateInspection{Resource: name, Eligibility: "unknown", Reasons: []string{}, Unknown: []string{"native update admission and authenticated remote continuity have not been evaluated by this inspection"}}
	m, err := c.managedServiceArtifactManifest(name)
	if err != nil {
		return r, err
	}
	if err := manifestpkg.ValidateManagedServiceUpdate(m); err != nil {
		r.Reasons = append(r.Reasons, err.Error())
	} else {
		r.DeclarationEligible = true
	}
	artifact, err := c.inspectManagedServiceArtifact(ctx, m)
	if artifact.Version != "" {
		r.Desired = &artifact
	}
	if err != nil {
		r.Unknown = append(r.Unknown, fmt.Sprintf("staged artifact: %v", err))
		r.Reasons = append(r.Reasons, "declared staged artifact is not verified")
	}
	sup, _, err := managedServiceSupervisorFor(name)
	if err != nil {
		return r, err
	}
	state, running, err := sup.Status()
	if err != nil {
		r.Unknown = append(r.Unknown, fmt.Sprintf("serving owner: %v", err))
	} else if running {
		r.Running = &state
	} else {
		r.Reasons = append(r.Reasons, "no running managed-service owner")
	}
	store, err := newManagedUpdateStore(name)
	if err != nil {
		return r, err
	}
	ledger, err := store.read()
	if err != nil {
		r.Unknown = append(r.Unknown, fmt.Sprintf("update ledger: %v", err))
	} else if len(ledger.Records) > 0 {
		latest := ledger.Records[len(ledger.Records)-1].State
		r.Operation = &latest
		if !latest.Phase.Terminal() {
			r.Eligibility = "operation_in_progress"
		}
	}
	if len(r.Reasons) > 0 {
		r.Eligibility = "ineligible"
	}
	return r, nil
}
