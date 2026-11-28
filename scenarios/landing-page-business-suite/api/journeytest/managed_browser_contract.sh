#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
evidence_log="${MB14_EVIDENCE_LOG:-}"
if [[ -n "$evidence_log" ]]; then
  mkdir -p "$(dirname "$evidence_log")"
  exec > >(tee -a "$evidence_log") 2>&1
fi

printf 'MB14 managed browser cross-owner contract run · %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
run_owner_tests() {
  local owner_dir="$1"
  shift
  printf '\n$ (cd %s && GOWORK=off go test %s)\n' "$owner_dir" "$*"
  (cd "$repo_root/$owner_dir" && GOWORK=off go test "$@")
}

run_owner_tests scenarios/ai-gateway/api ./internal/inference \
  -run 'TestBrowserRunSourcesKeepOneInferenceReceiptContract|TestBrowserNoEligibleRouteOffersNextStepWithoutInferenceOrFunding|TestBrowserRunRolesDoNotFallbackAfterUncertainMeteredOutcome|TestBrowserRunRolesRequireCorrelationFieldsBeforeRepositoryDispatch' \
  -count=1

run_owner_tests scenarios/agent-manager/api ./internal/handlers \
  -run 'TestAttestBrowserSettlementRejectsUntrustedRequestsBeforeConsume|TestRecordBASBrowserEvidenceReturnsTheLPBSReceiptFromTheSettlementOwner|TestRecordBASBrowserEvidenceRequiresExactGrantAndBrokerProof' \
  -count=1

run_owner_tests scenarios/landing-page-business-suite/api ./internal/intelligence ./internal/commerce \
  -run 'TestBrowserEvidenceIdentityReadinessBlocksBeforeReservationAndInference|TestBASProofResolutionRejectsWrongRunOrUnclosedSession|TestClosedBASProofBindsBrowserMeasurementsToTokenSettlement|TestBrowserInferenceRequiresRunGrantProofBoundToReservation|TestGatewayRootOperationReplayRefusedBeforeSecondInferenceDispatch|TestGetBrowserSessionReceiptReturnsSettledJointMeterAndScopesByAccount|TestResolveBrowserSessionSettlementBindingUsesExactRunSessionAndGrant' \
  -count=1

run_owner_tests scenarios/landing-page-business-suite/api . \
  -run '^TestAgentManagerPrincipalSettlesOnlyItsResolvedTenantBinding$' \
  -count=1

printf '\nMB14 cross-owner contract run passed. This exercises production owners with fixture providers/owners; it is not a live customer or hosted-path qualification.\n'
