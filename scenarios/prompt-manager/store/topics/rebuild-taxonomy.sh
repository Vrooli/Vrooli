#!/usr/bin/env bash
# Rebuild the steer-skill topic taxonomy — the locked tree from the
# discover-ranking-refactor plan (§6). Topics own their skill lists (SSOT);
# leaves inherit their folder's and the root's skills via upward accumulation.
#
# Re-runnable: deletes every taxonomy topic first, then recreates the tree
# parent-before-child, so a second run yields an identical tree. Requires the
# prompt-manager API to be running:
#
#   vrooli scenario start prompt-manager        # or: make start
#   bash store/topics/rebuild-taxonomy.sh
#
# After running, restart the scenario (or trigger a reconcile) so the rebuilt
# topics are re-embedded and their descriptions drive topic discoverability:
#
#   vrooli scenario restart prompt-manager
set -euo pipefail

pm() { prompt-manager "$@"; }

# Every ID this script manages (new tree + legacy seed IDs), deleted first so the
# rebuild is idempotent.
ALL_IDS=(
  development architecture building-features refactoring-cleanup api-cli-interfaces
  testing debugging-investigation reliability-resilience frontend-ui
  configuration-observability platform-storage-packaging performance security
  finishing-spec-hygiene
  topic-1 topic-2 topic-3 topic-4 topic-5 topic-6
)

echo "==> Deleting existing taxonomy topics (idempotent)…"
for id in "${ALL_IDS[@]}"; do
  pm topic delete "$id" >/dev/null 2>&1 || true
done

echo "==> Creating root + folders + leaves (parent before child)…"

# Root development practices — always in view for any dev task.
pm topic create --id development --name "Development" \
  --description "General software development on a Vrooli scenario: building, maintaining, refactoring, and shipping code. Root practices every task should keep in view." \
  --skills "documentation-health,test,intent-clarification"

# Architecture & code structure (folder — its own pack plus the three leaves below).
pm topic create --id architecture --name "Architecture & code structure" --parent development \
  --description "Architecture and code structure: where responsibilities live, the seams between components, the invariants the code relies on, and the vocabulary that names them." \
  --skills "seam-discovery-and-enforcement,boundary-of-responsibility-enforcement,invariant-discovery-and-enforcement,screaming-architecture-audit,concept-vocabulary-unification"

pm topic create --id building-features --name "Building features" --parent architecture \
  --description "Building a new feature or capability: extracting decision boundaries, designing error semantics and recovery paths, and compressing the domain model." \
  --skills "decision-boundary-extraction,error-semantics-recovery-path-design,domain-compression"

pm topic create --id refactoring-cleanup --name "Refactoring & cleanup" --parent architecture \
  --description "Refactoring and cleaning up existing code: restructuring, removing dead code and duplication, unifying utilities, and reducing cognitive load." \
  --skills "refactor,code-cleanup,utils-unification,cognitive-load-reduction"

pm topic create --id api-cli-interfaces --name "API & CLI interfaces" --parent architecture \
  --description "Designing and evolving API, CLI, and cross-component interfaces: HTTP/Connect APIs, CLI command surfaces, interoperability contracts, and bundle integration." \
  --skills "api-steer,cli-steer,interoperability-steer,bundle-integration-steer"

# Top-level leaves under Development.
pm topic create --id testing --name "Testing" --parent development \
  --description "Writing and structuring tests: unit test architecture and end-to-end testing." \
  --skills "unit-testing-architecture-steer,e2e-testing"

pm topic create --id debugging-investigation --name "Debugging & investigation" --parent development \
  --description "Debugging a non-obvious problem or investigating unfamiliar code: scientific debugging and systematic exploration." \
  --skills "scientific-debugging,explore"

pm topic create --id reliability-resilience --name "Reliability & resilience" --parent development \
  --description "Reliability and resilience: graceful degradation under failure, idempotency and replay safety, interruption and progress continuity, evolution-axis resilience, and temporal ordering." \
  --skills "failure-topography-and-graceful-degradation,idempotency-replay-safety-hardening,progress-continuity-interruption-resilience,change-axis-and-evolution-resilience-audit,temporal-flow-audit"

pm topic create --id frontend-ui --name "Frontend / UI" --parent development \
  --description "Frontend and UI work: React coherence and stability, Vrooli UI interop, UX and experience architecture, navigation integrity, design-system migration, and internationalization." \
  --skills "react-coherence,react-stability,vrooli-ui-interop,ux,experience-architecture-audit,navigation-integrity-audit,ui-design-system-migration,ui-i18n-adoption"

pm topic create --id configuration-observability --name "Configuration & observability" --parent development \
  --description "Configuration and observability: designing tunable control surfaces and levers, and the signal and feedback surfaces that make behavior visible and steerable." \
  --skills "control-surface-tunable-levers-design,signal-and-feedback-surface-design"

pm topic create --id platform-storage-packaging --name "Platform, storage & packaging" --parent development \
  --description "Platform, storage, and packaging: storage-layer design and cross-platform readiness for deployment and distribution." \
  --skills "storage-steer,cross-platform-readiness"

pm topic create --id performance --name "Performance" --parent development \
  --description "Performance work: profiling, optimizing throughput and latency, and reducing resource use." \
  --skills "performance"

pm topic create --id security --name "Security" --parent development \
  --description "Security work: hardening, threat modeling, and closing vulnerabilities." \
  --skills "security"

pm topic create --id finishing-spec-hygiene --name "Finishing & spec hygiene" --parent development \
  --description "Finishing and spec hygiene: syncing spec to implementation, final polish, progress tracking, and reference-pattern fitness." \
  --skills "spec-sync,polish,progress,reference-pattern-fitness"

echo "==> Done. Topic tree:"
pm topic tree || true
echo
echo "Now re-embed: vrooli scenario restart prompt-manager"
