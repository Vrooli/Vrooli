package artifacts

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/coreset"
	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/setup/v1"
	selectionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/selection"
	selectionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/selection/selectionv1connect"
	"vrooli-bridge/internal/artifacts"
)

type onboardingSelectionVerifier struct {
	resolve func(context.Context) (string, error)
	client  connect.HTTPClient
}

func (v onboardingSelectionVerifier) Authorize(ctx context.Context, req artifacts.ClosureShipmentRequest) error {
	if strings.TrimSpace(req.SelectionAuthorization) == "" {
		return errors.New("authenticated onboarding owner credential is required")
	}
	if v.resolve == nil {
		return errors.New("onboarding selection authority is not configured")
	}
	base, err := v.resolve(ctx)
	if err != nil {
		return err
	}
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return errors.New("onboarding selection authority URL is empty")
	}
	client := v.client
	if client == nil {
		client = http.DefaultClient
	}
	api := selectionconnect.NewSelectionServiceClient(client, base)
	request := connect.NewRequest(&selectionv1.GetHandoffRequest{
		Reference: req.SelectionReceiptRef, DeploymentId: req.DeploymentID, Target: req.TargetID,
		EnrollmentGeneration: req.EnrollmentGeneration, DesiredRevision: req.DesiredRevision,
		SelectionDigest: req.SelectionDigest,
	})
	request.Header().Set("Authorization", req.SelectionAuthorization)
	response, err := api.GetHandoff(ctx, request)
	if err != nil {
		return errors.New("onboarding rejected selection receipt: " + err.Error())
	}
	if response.Msg.GetHandoff() == nil || response.Msg.GetSelection() == nil {
		return errors.New("onboarding returned an incomplete selection receipt")
	}
	handoff := response.Msg.GetHandoff()
	if handoff.GetReference() != req.SelectionReceiptRef || handoff.GetState() != "pending" || handoff.GetDeploymentId() != req.DeploymentID || handoff.GetTarget() != req.TargetID || handoff.GetMachineId() != req.MachineID || handoff.GetNodeId() != req.NodeID || handoff.GetEnrollmentGeneration() != req.EnrollmentGeneration || handoff.GetDesiredRevision() != req.DesiredRevision || handoff.GetSelectionDigest() != req.SelectionDigest {
		return errors.New("onboarding selection receipt binding mismatch")
	}
	if handoff.GetClosureDigest() == "" || handoff.GetClosureDigest() != req.Closure.Digest {
		return errors.New("canonical closure membership differs from the onboarding selection receipt")
	}
	return matchSelectionRoots(response.Msg.GetSelection(), req.Closure)
}

func matchSelectionRoots(selection *setupv1.Selection, closure coreset.DependencyClosure) error {
	wantScenarios := normalizeSet(selection.GetScenarios())
	wantResources := normalizeSet(selection.GetOptionalResources())
	gotScenarios, gotResources, seenResources := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, member := range closure.Members {
		name := strings.ToLower(strings.TrimSpace(member.Component.Name))
		if member.Component.Kind == "scenario" && member.Selected {
			gotScenarios[name] = true
		}
		if member.Component.Kind == "resource" {
			seenResources[name] = true
			if member.Selected {
				gotResources[name] = true
			}
		}
	}
	if !equalSets(wantScenarios, gotScenarios) {
		return errors.New("selected scenario roots differ from the onboarding selection receipt")
	}
	for resource := range wantResources {
		if !seenResources[resource] {
			return errors.New("selected resource is absent from the canonical closure")
		}
	}
	for resource := range gotResources {
		if !wantResources[resource] {
			return errors.New("canonical closure selects a resource outside the onboarding selection receipt")
		}
	}
	return nil
}

func normalizeSet(values []string) map[string]bool {
	set := map[string]bool{}
	for _, value := range values {
		if value = strings.ToLower(strings.TrimSpace(value)); value != "" {
			set[value] = true
		}
	}
	return set
}

func equalSets(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for value := range a {
		if !b[value] {
			return false
		}
	}
	return true
}
