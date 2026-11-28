// Package localserviceadmission reports and converges the local-service policy
// installed by project setup. It never installs root artifacts or invokes sudo.
package localserviceadmission

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/privilegebroker"
)

type brokerClient interface {
	Do(context.Context, privilegebroker.Request) (privilegebroker.Result, error)
}
type handler struct {
	manifest hostreqkit.SafeguardManifest
	client   brokerClient
}

func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return handler{manifest: manifest, client: privilegebroker.NewClient()}
}
func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindSafeguard }

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	status.SupportClass = hostreqkit.SupportSupported
	if host.OS != "linux" {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		return status
	}
	if requirement.Manual {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status
	}
	bindings, err := privilegebroker.LocalAdmissionBindings(requirement.Config)
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status
	}
	if len(bindings) == 0 {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "No local-service bindings selected.")
		return status
	}
	status.Applied = true
	for _, binding := range bindings {
		result, err := h.call(privilegebroker.ActionLocalAdmissionInspect, binding.ID)
		if err != nil || result.LocalAdmission == nil || result.LocalAdmission.BindingDigest != binding.Digest() {
			status.Applied = false
			status.BlockingReason = hostreqkit.BlockingNeedsSudo
			status.Notes = append(status.Notes, "Install the current binding through `vrooli setup broker --sudo-mode=ask`.")
			continue
		}
		if result.Status != "verified" || !result.LocalAdmission.Enabled {
			status.Applied = false
			status.Notes = append(status.Notes, binding.ID+": "+result.LocalAdmission.Reason)
		}
	}
	if status.Applied {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
	}
	return status
}

func (h handler) Apply(_ hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	if status.SupportClass != hostreqkit.SupportSupported {
		return status, nil
	}
	if status.Applied {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		return status, nil
	}
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		return status, nil
	}
	bindings, err := privilegebroker.LocalAdmissionBindings(status.Config)
	if err != nil {
		return status, err
	}
	// Recheck installed policy for every effect: a stale inspection must never
	// apply a different installed address/interface under the same binding ID.
	for _, binding := range bindings {
		result, err := h.call(privilegebroker.ActionLocalAdmissionInspect, binding.ID)
		if err != nil || result.LocalAdmission == nil || result.LocalAdmission.BindingDigest != binding.Digest() {
			status.ExecutionState = hostreqkit.ExecutionManualActionRequired
			return status, nil
		}
		result, err = h.call(privilegebroker.ActionLocalAdmissionEnsure, binding.ID)
		if err != nil || result.Status != "verified" && result.Status != "changed" {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, "Local-service admission is not verified for "+binding.ID)
			if err != nil {
				status.Notes = append(status.Notes, err.Error())
			} else {
				if result.Code != "" {
					status.Notes = append(status.Notes, result.Code)
				}
				if result.Evidence.Detail != "" {
					status.Notes = append(status.Notes, result.Evidence.Detail)
				}
			}
			return status, nil
		}
	}
	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionApplied
	status.BlockingReason = hostreqkit.BlockingNone
	return status, nil
}

func (h handler) call(action, id string) (privilegebroker.Result, error) {
	if h.client == nil {
		return privilegebroker.Result{}, fmt.Errorf("broker unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return h.client.Do(ctx, privilegebroker.Request{Version: privilegebroker.ProtocolVersion, RequestID: "local-admission-" + strconv.FormatInt(time.Now().UnixNano(), 10), Action: action, LocalAdmission: &privilegebroker.LocalAdmissionSubject{Binding: id}})
}
