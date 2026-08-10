<!-- GENERATED FILE: structure-health rules docs. DO NOT EDIT. -->

# Structural Rule Catalog

This page is generated from the Structure Health rule catalog. The executable catalog is the source; this page is a reviewable artifact.

| Code | Target kind | Severity | Enforcement | Claim |
|---|---|---|---|---|
| CONTROL_PLANE_LAYOUT_MISSING | control-plane | warning | enforced | control-plane.go-native |
| DOCS_LAYOUT_MISSING | docs | warning | enforced | docs.layout |
| DOCS_MANIFEST_INVALID | docs | error | enforced | docs.manifest |
| PACKAGE_GO_REPLACE_MISSING | package | error | enforced | package.go-replaces |
| PACKAGE_INTERNAL_IMPORT | package | error | enforced | package.no-root-internal |
| PACKAGE_LAYOUT_MISSING | package | error | enforced | package.layout |
| PACKAGE_MANIFEST_INVALID | package | error | enforced | package.manifest-shape |
| PACKAGE_MANIFEST_MISSING | package | error | enforced | package.manifest |
| PACKAGE_MODULE_PATH_MISMATCH | package | error | enforced | package.module-identifiers |
| PACKAGE_NAME_MISMATCH | package | error | enforced | package.identity |
| PACKAGE_OWN_MODULE_MISSING | package | error | enforced | package.own-module |
| PROFILE_CONFORMANCE_VIOLATION | scenario | warning | advisory | scenario.profile-advisory |
| PROFILE_DEVELOP_STEPS | scenario | warning | enforced | scenario.develop-steps |
| PROFILE_ENV_VALIDATION | scenario | warning | enforced | scenario.environment-validation |
| PROFILE_HARDCODED_VALUES | scenario | warning | enforced | scenario.no-hardcoded-values |
| PROFILE_HEALTH_LIFECYCLE | scenario | error | enforced | scenario.health-lifecycle |
| PROFILE_PORTS | scenario | error | enforced | scenario.ports |
| PROFILE_REQUIRED_STRUCTURE | scenario | error | enforced | scenario.required-structure |
| PROFILE_RUNTIME_STORAGE | scenario | error | enforced | scenario.runtime-storage |
| PROFILE_SETUP_CONDITIONS | scenario | error | enforced | scenario.setup-conditions |
| PROFILE_SETUP_STEPS | scenario | warning | enforced | scenario.setup-steps |
| PROFILE_TEST_COVERAGE | scenario | warning | enforced | scenario.test-coverage |
| PROFILE_TEST_STEPS | scenario | error | enforced | scenario.test-steps |
| PROFILE_UI_LIFECYCLE_LAUNCH | scenario | error | enforced | scenario.ui-lifecycle-launch |
| PROFILE_UI_STRUCTURE | scenario | error | enforced | scenario.ui-structure |
| PROJECT_CLAIM_UNRESOLVED | project | error | enforced | project.claim-resolution |
| PROJECT_BUNDLE_PROFILE | project | error | enforced | project.bundle-profile |
| PROJECT_CANONICAL_LAYOUT | project | error | enforced | project.canonical-layout |
| PROJECT_CONFIG_SURFACE | project | error | enforced | project.config-surface |
| PROJECT_CONTRACT_INVALID | project | error | enforced | project.contract |
| PROJECT_EXCLUDED_LEGACY | project | error | enforced | project.excluded-legacy |
| PROJECT_LIVE_STRUCTURE | project | error | enforced | project.live-structure |
| PROJECT_ORPHAN_GO_WORK_SUM | project | error | enforced | project.go-work-pair |
| PROJECT_PHASE1_SEMANTICS | project | error | enforced | project.phase1-semantics |
| PROJECT_PNPM_WORKSPACE_INVALID | project | error | enforced | project.pnpm-workspace |
| PROJECT_PROFILE_ROOTS | project | error | enforced | project.profile-roots |
| PROJECT_RESOURCE_ARTIFACTS | project | error | enforced | project.resource-artifacts |
| PROJECT_ROOT_NPMRC | project | error | enforced | project.no-root-npmrc |
| PROJECT_ROOT_PNPM_LOCK | project | error | enforced | project.no-root-pnpm-lock |
| PROJECT_RUNTIME_HOME | project | error | enforced | project.runtime-home |
| RESOURCE_HEALTH_KIND_MISSING | resource | error | enforced | resource.health-checks |
| RESOURCE_IMAGE_UNPINNED | resource | error | enforced | resource.image-pinning |
| RESOURCE_MANIFEST_INVALID | resource | error | enforced | resource.manifest |
| RESOURCE_SHELL_FORBIDDEN | resource | error | enforced | resource.go-native-lifecycle |
| SAFEGUARD_HANDLER_MISSING | safeguard | error | enforced | safeguard.handler |
| SAFEGUARD_MANIFEST_INVALID | safeguard | error | enforced | safeguard.manifest |
| SAFEGUARD_NAME_MISMATCH | safeguard | error | enforced | safeguard.identity |
| SCENARIO_UI_BOUNDARY_MISSING | scenario | error | enforced | scenario.ui-boundary |
| SCENARIO_UI_LOCKFILE_MISSING | scenario | error | enforced | scenario.ui-lockfile |
| SCENARIO_WORKSPACE_DEPENDENCY | scenario | error | enforced | scenario.no-workspace-star |
| TEAM_LAYOUT_MISSING | team | error | enforced | team.layout |
| TEAM_MANIFEST_INVALID | team | error | enforced | team.manifest |
| TEAM_OWNER_MISMATCH | team | error | enforced | team.identity |
| TEAM_OWNER_MISSING | team | error | enforced | team.owner |
| TOOL_HANDLER_MISSING | tool | error | enforced | tool.handler |
| TOOL_MANIFEST_INVALID | tool | error | enforced | tool.manifest |
| TOOL_NAME_MISMATCH | tool | error | enforced | tool.identity |

See [rule coverage](structure-rule-coverage.md) for the generated reachability matrix.
