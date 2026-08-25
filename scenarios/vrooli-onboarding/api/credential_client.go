package main

import (
	"context"
	"encoding/json"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

// onboardingAuthority is this API's single construction path for the credential
// authority, and the only one. It is a variable so a test can inject a host
// condition that cannot be produced on a real machine — a platform with no
// credential backend at all. Production code never reassigns it.
//
// Owning the seam here rather than reaching for the control-plane package's
// own test hook is what keeps the onboarding API's internal coupling at its
// declared ceiling.
var onboardingAuthority = credentialauthority.Default

// onboardingCredentialClientOptions builds the client the onboarding API uses.
//
// Three fields decide what the client can answer, and each one is load-bearing
// on its own. Root is what makes List and Inventory read the control-plane
// population instead of returning nothing. StateDir is what makes Doctor read
// the recovery receipt instead of skipping its whole recovery block. Descriptors
// is the manifest fallback for a transport that has no repository to walk.
//
// Bundle mode has neither a repository root nor a control-plane runtime home,
// so both paths stay empty there and the client degrades exactly as it does on
// a desktop install rather than reading paths that do not exist.
func onboardingCredentialClientOptions() (credentialclient.ClientOptions, error) {
	authority, err := onboardingAuthority()
	if err != nil {
		return credentialclient.ClientOptions{}, err
	}
	options := credentialclient.ClientOptions{Authority: authority}
	roots, rootsErr := resolveRoots()
	if rootsErr != nil || strings.TrimSpace(roots.RepoRoot) == "" {
		return options, nil
	}
	options.Root = roots.RepoRoot
	if stateDir, stateErr := config.VrooliPath(repocontract.HomeKeyState); stateErr == nil {
		options.StateDir = stateDir
	}
	scopeRoot := options.Root
	options.Descriptors = func() ([]credentialclient.CredentialRef, error) {
		return credentialclient.DescriptorsForScope(scopeRoot, credentialclient.Scope{IncludeProject: true})
	}
	return options, nil
}

func onboardingCredentialClient() (credentialclient.Client, error) {
	options, err := onboardingCredentialClientOptions()
	if err != nil {
		return nil, err
	}
	return credentialclient.NewClient(options)
}

// recheckCredentialAuthority discards the cached backend-availability verdict.
//
// The authority caches that probe for the process lifetime, which is right for
// a short CLI invocation and wrong for the onboarding API: it is started by the
// control plane and can outlive several store state changes, so a wizard opened
// hours later would otherwise report every credential unsupported because of
// the store state at start. Callers invoke this once per request, never once
// per credential.
func recheckCredentialAuthority() {
	if authority, err := onboardingAuthority(); err == nil {
		authority.Recheck()
	}
}

func onboardingDoctorJSON(ctx context.Context) ([]byte, error) {
	client, err := onboardingCredentialClient()
	if err != nil {
		return nil, err
	}
	response, err := client.Doctor(ctx)
	if err != nil {
		return nil, err
	}
	return json.Marshal(response)
}

func onboardingKeyringJSON(ctx context.Context, action string) ([]byte, error) {
	client, err := onboardingCredentialClient()
	if err != nil {
		return nil, err
	}
	var report credentialclient.KeyringReport
	if action == "inspect" {
		report, err = client.KeyringInspect(ctx, "")
	} else {
		report, err = client.KeyringRepair(ctx, "")
	}
	if err != nil {
		return nil, err
	}
	return json.Marshal(report)
}

func onboardingProvision(ctx context.Context, logicalID, field, value string) error {
	client, err := onboardingCredentialClient()
	if err != nil {
		return err
	}
	_, err = client.Provision(ctx, credentialclient.ProvisionRequest{Identity: logicalID, Field: field, Value: value})
	return err
}

func onboardingStatusJSON(ctx context.Context, logicalID, field string) ([]byte, error) {
	client, err := onboardingCredentialClient()
	if err != nil {
		return nil, err
	}
	status, err := client.Status(ctx, logicalID, field)
	if err != nil {
		return nil, err
	}
	return json.Marshal(status)
}
