# Meta-Orchestrator Summary

## Source

Follow-up to `execute/deployment-manager-lpbs-desktop-release-orchestration`. Operators currently learn prerequisites are missing only by calling `releases start` and reading the failure — wasting the advisory lock and leaving a failed row on the unique (profile, commit, channel) index. A preflight check solves that.

## Codebase Inspection Already Done

- `scenarios/deployment-manager/api/releases/handlers.go:216-320` — Start handler: parses body, defaults channel, acquires advisory lock, inserts release row, calls orchestrator, re-reads row. All gates run inside the locked/allocated path.
- `scenarios/deployment-manager/api/deployments/orchestrator_release.go:18-52` — RunDeploy sequences: deployLoadProfile (approvals), deployCheckCloudHealth, deployCheckLPBSReadiness, then build/package/publish/verify. The first three are the preflight-candidate gates.
- `scenarios/deployment-manager/api/deployments/orchestrator.go:197-` — deployLoadProfile includes the approval gate check against the commit. Extract this into a reusable gate-only helper so preflight and start share it.
- `scenarios/deployment-manager/api/deployments/cloud_client.go` — CheckLPBSHealth is already side-effect-free, safe for preflight.
- `scenarios/deployment-manager/api/deployments/lpbs_release_client.go` — CheckDeployReadiness is side-effect-free, safe for preflight.
- `scenarios/deployment-manager/api/profiles/lpbs_config.go` — Get is the natural check for "profile has LPBS config".
- `scenarios/deployment-manager/api/server/routes.go` — existing release routes registered conditionally on ReleasesHandler. Add preflight there.

## Decisions Made

- Preflight is READ-ONLY: no advisory lock, no release row insertion, no S2D invocation, no /verify call.
- Scope of checks: LPBS config present, approvals gated for commit, cloud health, LPBS readiness.
- Response shape keys into the typed error codes from `deployment-manager-release-typed-remediation-errors`.

## Unresolved Questions Deferred To Workshop

- **Should preflight also run required-platforms check?** If the profile declares [linux, windows, macos] but request passes `--platforms linux`, is that a gate failure or accepted? Recommend surface as info, not gate — caller's platform subset is valid if each one is approved.
- **Cacheability**: the same preflight call repeated 10 times in 60 seconds should not hammer LPBS. Should the preflight handler cache results per (profile, commit) for a short TTL, or leave it uncached? Uncached is safer; cache is an optimization.
- **Should preflight be called inline from Start?** E.g., `releases start --preflight-first` that runs preflight before acquiring lock, then start if green. Keep as separate call for now; caller composes.

## Dependency Notes

Depends on `execute/deployment-manager-release-typed-remediation-errors` for the typed error shape that preflight returns. Blocks `landing-page-desktop-upload-guided-remediation-rewrite`.

## Effort Assessment

S because most infrastructure already exists: the orchestrator steps we're pulling out are side-effect-free, typed errors will already be in place after item #1. Main work is: (1) refactor gate logic out of orchestrator into reusable functions, (2) new handler + CLI verb, (3) tests.
