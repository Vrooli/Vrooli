package vps

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/faultinject"
	"scenario-to-cloud/reach"
)

// This file maps every owner operation onto the target verbs the executor
// runs and interprets their replies. Every effectful verb carries the
// operation identity and fence; the target owner replays receipts it already
// holds, so re-running an action is a replay of the invocation, never of a
// completed effect.

// actionHandler maps one owner operation onto its target verbs. It returns a
// short detail string for the receipt.
type actionHandler func(ctx context.Context, e *executor, action execplan.Action) (string, error)

var actionHandlers = map[string]actionHandler{
	execplan.OpHostPrepare:             runCommandsHandler,
	execplan.OpEdgeFirewallAllow:       runCommandsHandler,
	execplan.OpDataInventory:           runDataInventory,
	execplan.OpReleaseDeliver:          runReleaseDeliver,
	execplan.OpReleaseVerify:           runReleaseVerify,
	execplan.OpReleaseStage:            runCommandsHandler,
	execplan.OpDataBackup:              runDataBackup,
	execplan.OpReleaseActivate:         runReleaseActivate,
	execplan.OpConfigApply:             runConfigApply,
	execplan.OpWorkloadStop:            runCommandsHandler,
	execplan.OpEdgeRouteApply:          runCommandsHandler,
	execplan.OpCredentialsProvision:    runCredentialsProvision,
	execplan.OpRuntimeStartDeps:        runCommandsHandler,
	execplan.OpWorkloadStart:           runWorkloadStart,
	execplan.OpVerifyReadiness:         runVerifyReadiness,
	execplan.OpReleaseRetainPredecesor: runReleaseRetainPredecessor,
	execplan.OpEdgeRouteRetire:         runEdgeRouteRetire,
	execplan.OpGrantsRevoke:            runGrantsRevoke,
	execplan.OpDataRetire:              runDataRetire,
	execplan.OpArtifactsRetire:         runArtifactsRetire,
}

func runCommandsHandler(ctx context.Context, e *executor, action execplan.Action) (string, error) {
	return e.runCommands(ctx, action)
}

func runDataInventory(ctx context.Context, e *executor, action execplan.Action) (string, error) {
	commands, err := ActionCommands(action, e.commandContext())
	if err != nil {
		return "", err
	}
	var details []string
	for _, tc := range commands {
		_, res, err := e.invoke(ctx, tc)
		if err != nil {
			return "", err
		}
		var report struct {
			Scenario       string `json:"scenario"`
			UncoveredCount int    `json:"uncovered_count"`
			Entries        []struct {
				Path    string `json:"path"`
				Covered bool   `json:"covered"`
			} `json:"entries"`
		}
		_ = json.Unmarshal([]byte(res.Stdout), &report)
		var uncovered []string
		for _, entry := range report.Entries {
			if !entry.Covered {
				uncovered = append(uncovered, entry.Path)
			}
		}
		if len(uncovered) == 0 {
			details = append(details, report.Scenario+": no unmapped mutable directories")
		} else {
			details = append(details, report.Scenario+": unmapped "+strings.Join(uncovered, ","))
		}
	}
	return strings.Join(details, "; "), nil
}

func runReleaseVerify(ctx context.Context, e *executor, action execplan.Action) (string, error) {
	commands, err := ActionCommands(action, e.commandContext())
	if err != nil {
		return "", err
	}
	reply, _, err := e.invoke(ctx, commands[0])
	if err != nil {
		return "", err
	}
	var report struct {
		Verified      bool   `json:"verified"`
		ReleaseDigest string `json:"release_digest"`
	}
	_ = json.Unmarshal(reply.Report, &report)
	if !report.Verified {
		return "", apierrors.New(apierrors.CodeReleaseVerificationFailed, "target owner did not verify the delivered release").WithDetail("release_id", action.Inputs["release_id"])
	}
	if report.ReleaseDigest != "" && report.ReleaseDigest != action.Inputs["release_id"] {
		return "", apierrors.New(apierrors.CodeReleaseVerificationFailed, "target verified a different release than the plan names").
			WithDetail("expected", action.Inputs["release_id"]).WithDetail("observed", report.ReleaseDigest)
	}
	return "verified " + action.Inputs["release_id"], nil
}

func runDataBackup(ctx context.Context, e *executor, action execplan.Action) (string, error) {
	if err := faultinject.Hit(ctx, faultinject.DataBeforeBackup); err != nil {
		return "", err
	}
	commands, err := ActionCommands(action, e.commandContext())
	if err != nil {
		return "", err
	}
	reply, _, err := e.invoke(ctx, commands[0])
	if err != nil {
		return "", err
	}
	if len(reply.RecoveryPoint) == 0 && !reply.Replayed {
		return "", apierrors.New(apierrors.CodeBackupProviderUnavailable, "the target owner returned no recovery point manifest")
	}
	detail := "recovery point captured"
	if reply.Replayed {
		detail = "recovery point already captured (replayed)"
	}
	if e.rt.Backups != nil && len(reply.RecoveryPoint) > 0 {
		id, err := e.rt.Backups.Record(ctx, e.deploymentID, e.rt.Identity, reply.RecoveryPoint, action.Inputs["release_digest"])
		if err != nil {
			return "", err
		}
		detail += " and recorded as " + id
	} else if e.rt.Backups == nil {
		detail += " (no cloud-side recorder configured)"
	}
	return detail, nil
}

func runReleaseActivate(ctx context.Context, e *executor, action execplan.Action) (string, error) {
	if err := faultinject.Hit(ctx, faultinject.ActivationBeforeSwitch); err != nil {
		return "", err
	}
	commands, err := ActionCommands(action, e.commandContext())
	if err != nil {
		return "", err
	}
	reply, _, err := e.invoke(ctx, commands[0])
	if err != nil {
		return "", err
	}
	if err := faultinject.Hit(ctx, faultinject.ActivationAfterSwitch); err != nil {
		return "", err
	}
	detail := "activated " + action.Inputs["release_id"]
	if reply.Receipt != nil {
		if reply.Receipt.Outcome == "unchanged" {
			detail = "release " + action.Inputs["release_id"] + " already active"
		}
		if reply.Replayed {
			detail += " (replayed)"
		}
		if unmapped, ok := reply.Receipt.Details["legacy_unmapped"].([]any); ok && len(unmapped) > 0 {
			names := make([]string, 0, len(unmapped))
			for _, u := range unmapped {
				names = append(names, fmt.Sprint(u))
			}
			detail += "; unmapped legacy data left in place: " + strings.Join(names, ",")
		}
	}
	return detail, nil
}

// runConfigApply runs the target's setup and delivers the autoheal scope
// declaration as a file (a small typed document, never a shell write).
func runConfigApply(ctx context.Context, e *executor, action execplan.Action) (string, error) {
	detail, err := e.runCommands(ctx, action)
	if err != nil {
		return "", err
	}
	if action.Inputs["autoheal"] == "true" && action.Inputs["autoheal_scope_path"] != "" {
		path, err := writeAutohealScope(e.manifest)
		if err != nil {
			return "", err
		}
		if _, err := e.reach.Deliver(ctx, e.rt.Target, reach.Delivery{Files: []reach.ArtifactFile{{Role: "autoheal_scope", LocalPath: path, RemotePath: action.Inputs["autoheal_scope_path"]}}}); err != nil {
			return "", err
		}
		detail += "; autoheal scope delivered"
	}
	return detail, nil
}

func runCredentialsProvision(ctx context.Context, e *executor, action execplan.Action) (string, error) {
	if e.manifest.Secrets == nil || len(e.manifest.Secrets.BundleSecrets) == 0 {
		return "no secrets declared", nil
	}
	if e.rt.Credentials == nil {
		return "", apierrors.New(apierrors.CodeUnsupportedCapability, "credential authority is not configured for this executor; credentials.provision cannot run").WithDetail("action", action.ID)
	}
	generated, err := e.rt.SecretsGen.GenerateSecrets(e.manifest.Secrets.BundleSecrets)
	if err != nil {
		return "", fmt.Errorf("generate secrets: %w", err)
	}
	generatedValues := make(map[string]string, len(generated))
	for _, g := range generated {
		generatedValues[g.ID] = g.Value
	}
	operatorValues, err := buildUserSecretMap(e.manifest, e.rt.ProvidedSecrets)
	if err != nil {
		return "", err
	}
	result, err := e.rt.Credentials.Provision(ctx, CredentialProvisionRequest{
		DeploymentID: e.deploymentID, Target: e.rt.Target, Identity: e.rt.Identity, Manifest: e.manifest,
		GeneratedValues: generatedValues, OperatorValues: operatorValues,
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d materialized, %d preserved, %d skipped", len(result.Materialized), len(result.Preserved), len(result.Skipped)), nil
}

// activeRelease is the target's durable pointer as `release list` reports it.
type activeRelease struct {
	Active *struct {
		ActiveRelease   string `json:"active_release"`
		PreviousRelease string `json:"previous_release"`
		Strategy        string `json:"strategy"`
	} `json:"active"`
	Interrupted *struct {
		Candidate string `json:"candidate"`
		Previous  string `json:"previous"`
	} `json:"interrupted_activation"`
	Releases []struct {
		Digest string `json:"digest"`
		Role   string `json:"role"`
		State  string `json:"state"`
		Path   string `json:"path"`
	} `json:"releases"`
}

func (e *executor) listReleases(ctx context.Context, step string) (activeRelease, error) {
	tc := readOnly(step, "cloud-target release list", []string{"--deployment", e.deploymentID}, verbTimeout(execplan.OpReleaseRetainPredecesor))
	_, res, err := e.invoke(ctx, tc)
	if err != nil {
		return activeRelease{}, err
	}
	var listing activeRelease
	if err := json.Unmarshal([]byte(strings.TrimSpace(res.Stdout)), &listing); err != nil {
		return activeRelease{}, fmt.Errorf("release list reply is not JSON: %w", err)
	}
	return listing, nil
}

// runWorkloadStart restarts the recorded active release through the target
// owner. The observed pointer decides which release starts: the plan's
// release id when it is active (or nothing is active yet), otherwise the
// active pointer is the truth and the plan is stale.
func runWorkloadStart(ctx context.Context, e *executor, action execplan.Action) (string, error) {
	listing, err := e.listReleases(ctx, action.ID+".observe")
	if err != nil {
		return "", err
	}
	if listing.Interrupted != nil {
		return "", apierrors.New(apierrors.CodeOperationConflict, "the target reports an interrupted activation; reconcile it before starting").
			WithDetail("candidate", listing.Interrupted.Candidate).WithDetail("previous", listing.Interrupted.Previous)
	}
	releaseID := action.Inputs["release_id"]
	if listing.Active != nil {
		if releaseID != "" && listing.Active.ActiveRelease != releaseID {
			return "", execplan.StaleError([]execplan.StaleReason{{Kind: execplan.PreconditionReleaseDigest, Expected: releaseID, Observed: listing.Active.ActiveRelease, Material: true}})
		}
		releaseID = listing.Active.ActiveRelease
	}
	if releaseID == "" {
		return "", apierrors.New(apierrors.CodeReleaseVerificationFailed, "no release is active on the target and the plan names none; deploy a release before starting").WithDetail("reason", "release_not_staged")
	}
	tc := effectful(action.ID, "cloud-target release activate", activateArgs(action.ID, releaseID, action.Inputs, e.commandContext(), true), verbTimeout(action.ID))
	reply, _, err := e.invoke(ctx, tc)
	if err != nil {
		return "", err
	}
	detail := "started release " + releaseID
	if reply.Replayed {
		detail += " (replayed)"
	}
	return detail, nil
}

var checkPublicHealthFunc = checkPublicHealth

// runVerifyReadiness proves the target runs the plan's release (the runtime
// and the pointer agree, no interrupted switch) and then the public path.
func runVerifyReadiness(ctx context.Context, e *executor, action execplan.Action) (string, error) {
	domainName := action.Inputs["domain"]
	checks := csv(action.Inputs["checks"])
	for _, check := range checks {
		switch check {
		case "local":
			listing, err := e.listReleases(ctx, action.ID+".observe")
			if err != nil {
				return "", fmt.Errorf("local: %w", err)
			}
			if listing.Interrupted != nil {
				return "", apierrors.New(apierrors.CodeHealthUnknown, "local: the target reports an interrupted activation").WithDetail("candidate", listing.Interrupted.Candidate)
			}
			if listing.Active == nil {
				return "", apierrors.New(apierrors.CodeHealthUnknown, "local: no release is active on the target")
			}
			if want := action.Inputs["release_id"]; want != "" && listing.Active.ActiveRelease != want {
				return "", apierrors.New(apierrors.CodeHealthUnknown, "local: the active release is not the one this plan deployed").
					WithDetail("expected", want).WithDetail("observed", listing.Active.ActiveRelease)
			}
		case "https":
			if err := checkPublicHealthFunc(ctx, fmt.Sprintf("https://%s/health", domainName), 10*time.Second); err != nil {
				return "", fmt.Errorf("https: %w", err)
			}
		case "origin":
			if err := checkOriginHealthFunc(ctx, domainName, action.Inputs["host"], 10*time.Second); err != nil {
				return "", fmt.Errorf("origin: %w", err)
			}
		case "public":
			if err := checkPublicHealthFunc(ctx, fmt.Sprintf("https://%s/health", domainName), 10*time.Second); err != nil {
				return "", fmt.Errorf("public: %w", err)
			}
		default:
			return "", fmt.Errorf("unknown readiness check %q", check)
		}
	}
	return "checks passed: " + strings.Join(checks, ","), nil
}

func runReleaseRetainPredecessor(ctx context.Context, e *executor, action execplan.Action) (string, error) {
	listing, err := e.listReleases(ctx, action.ID+".observe")
	if err != nil {
		return "", err
	}
	if listing.Active == nil {
		return "", apierrors.New(apierrors.CodeHealthUnknown, "no active release recorded on the target")
	}
	if want := action.Inputs["release_id"]; want != "" && listing.Active.ActiveRelease != want {
		return "", apierrors.New(apierrors.CodeHealthUnknown, "the active release is not the one this plan deployed").WithDetail("expected", want).WithDetail("observed", listing.Active.ActiveRelease)
	}
	if listing.Active.PreviousRelease == "" {
		return "", apierrors.New(apierrors.CodeReleaseVerificationFailed, "the target retained no predecessor for rollback").WithDetail("reason", "predecessor_not_retained")
	}
	return fmt.Sprintf("predecessor %s retained; rollback eligible: %s (%s)", listing.Active.PreviousRelease, action.Inputs["rollback_eligible"], action.Inputs["rollback_reason"]), nil
}
