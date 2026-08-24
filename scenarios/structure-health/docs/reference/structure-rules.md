<!-- GENERATED FILE: structure-health rules docs. DO NOT EDIT. -->

# Structural Rule Catalog

This page is generated from the Structure Health rule catalog.

| Code | Target kind | Severity | Enforcement | Claim | What it checks | Remediation |
|---|---|---|---|---|---|---|
| CONTROL_PLANE_LAYOUT_MISSING | control-plane | warning | enforced | control-plane.go-native | Control-plane targets are backed by Go source. | Keep control-plane cmd/internal targets backed by Go source files. |
| DOCS_LAYOUT_MISSING | docs | warning | enforced | docs.layout | The documentation target has a README hub. | Add the documentation hub README.md. |
| DOCS_MANIFEST_INVALID | docs | error | enforced | docs.manifest | Documentation manifests are valid and complete. | Repair manifest.json with version, title, and sections. |
| PACKAGE_BUILD_OUTPUTS_COMMITTED | package | error | enforced | package.no-committed-build-outputs | Generated package build outputs are not committed to the repository. | Remove generated outputs from version control and ignore the declared output paths. |
| PACKAGE_BUILD_OUTPUTS_UNDECLARED | package | error | enforced | package.build-output-declarations | Every package generate or build lifecycle command declares the files it produces. | Add non-empty repository-relative outputs globs to each generate or build command in .vrooli/package.json. |
| PACKAGE_CONSUMER_CLASS_SCAN_FAILED | package | error | enforced | package.consumer-discovery | Package consumer discovery is available to evaluate declared adoption boundaries. | Restore the control-plane package registry and re-run the package structure gate. |
| PACKAGE_CONSUMER_CLASS_VIOLATION | package | error | enforced | package.allowed-consumers | Discovered package consumers belong to classes explicitly allowed by the package manifest. | Declare the consumer class in package.adoption.allowed_consumers or move the consumer behind an appropriate package boundary. |
| PACKAGE_RESOURCE_ENV_OWNER_INVALID | package | error | enforced | package.resource-environment-ownership | Package resource-environment ownership claims name existing resource manifests. | Declare an existing resources/<name>/resource.json or remove the ownership claim. |
| PACKAGE_GO_REPLACE_MISSING | package | error | enforced | package.go-replaces | Modules depending on local Vrooli modules repeat the required local replace directives. | Use Scenario Dependency Analyzer to reconcile the module's local replaces. |
| PACKAGE_INTERNAL_IMPORT | package | error | enforced | package.no-root-internal | Packages do not import the control plane's private internal packages. | Promote or duplicate the shared capability and remove the root internal import. |
| PACKAGE_LAYOUT_MISSING | package | error | enforced | package.layout | A package has README.md and a language configuration at its root. | Provide README.md and a go.mod or package.json at the package root. |
| PACKAGE_MANIFEST_INVALID | package | error | enforced | package.manifest-shape | Package governance manifests have the required shape. | Repair .vrooli/package.json against the package schema. |
| PACKAGE_MANIFEST_MISSING | package | error | enforced | package.manifest | Every package has a .vrooli/package.json manifest. | Add a valid .vrooli/package.json package governance manifest. |
| PACKAGE_MODULE_PATH_MISMATCH | package | error | enforced | package.module-identifiers | Declared package module identifiers match discovered module paths. | Add the discovered module path to package.module_identifiers. |
| PACKAGE_NAME_MISMATCH | package | error | enforced | package.identity | The package manifest name matches the target identifier. | Set package.name to the canonical package id. |
| PACKAGE_OWN_MODULE_MISSING | package | error | enforced | package.own-module | A module-language parse unit rooted at a package has its own module configuration. | Add a module configuration at the package root or record an intentional exception. |
| PACKAGE_SOURCE_ENTRYPOINT | package | error | enforced | package.compiled-entrypoints | JavaScript package entrypoints resolve to compiled output rather than source files. | Point package metadata exports at dist/ or another declared generated output directory. |
| PROFILE_CONFORMANCE_VIOLATION | scenario | warning | advisory | scenario.profile-advisory | Unrecognized scenario profiles report profile-specific conventions without blocking. | Review the profile-specific finding and either satisfy the convention or use a recognized profile. |
| PROFILE_ENV_VALIDATION | scenario | warning | enforced | scenario.environment-validation | Scenario environment validation follows the profile convention. | Validate environment variables at the scenario boundary. |
| PROFILE_HARDCODED_VALUES | scenario | warning | enforced | scenario.no-hardcoded-values | Scenario source does not hardcode runtime configuration values. | Move runtime configuration to validated environment or service configuration. |
| PROFILE_HEALTH_LIFECYCLE | scenario | error | enforced | scenario.health-lifecycle | Scenario health lifecycle wiring is present and valid. | Repair the scenario health lifecycle configuration. |
| PROFILE_PORTS | scenario | error | enforced | scenario.ports | Scenario ports follow the declared lifecycle contract. | Repair scenario port declarations and references. |
| PROFILE_REQUIRED_STRUCTURE | scenario | error | enforced | scenario.required-structure | Scenario required structure is present. | Restore the required scenario layout. |
| PROFILE_RUNTIME_STORAGE | scenario | error | enforced | scenario.runtime-storage | Scenario runtime storage uses governed locations. | Move runtime state to the governed runtime-home storage seam. |
| PROFILE_TEST_COVERAGE | scenario | warning | enforced | scenario.test-coverage | Scenario source has the required test coverage shape. | Add or repair tests for the scenario surface. |
| PROFILE_UI_LIFECYCLE_LAUNCH | scenario | error | enforced | scenario.ui-lifecycle-launch | Scenario UI launch wiring follows the lifecycle contract. | Repair UI launch wiring and lifecycle ownership. |
| PROFILE_UI_STRUCTURE | scenario | error | enforced | scenario.ui-structure | Scenario UI structure follows the profile convention. | Repair the scenario UI structure. |
| PROJECT_BUNDLE_PROFILE | project | error | enforced | project.bundle-profile | The repository bundle profile includes and excludes canonical roots. | Restore the mini_vrooli_bundle include, exclude, and parameter policy. |
| PROJECT_CANONICAL_LAYOUT | project | error | enforced | project.canonical-layout | Canonical repository layout markers match the contract. | Restore the canonical repository markers and paths. |
| PROJECT_CLAIM_UNRESOLVED | project | error | enforced | project.claim-resolution | Every marked enforcement claim resolves to a catalog rule. | Add the referenced rule to the catalog or remove the enforcement claim. |
| PROJECT_CONFIG_SURFACE | project | error | enforced | project.config-surface | The project configuration surface matches the repository contract. | Remove unapproved entries from .vrooli. |
| PROJECT_CONTRACT_INVALID | project | error | enforced | project.contract | The repository contract is valid and readable. | Repair .vrooli/repo-contract.json. |
| PROJECT_CREDENTIAL_DESCRIPTOR_DUPLICATE | project | error | enforced | project.credential-descriptor-uniqueness | A single manifest does not declare one logical credential field more than once. | Remove the duplicate logical_id/field declaration from the manifest. |
| PROJECT_EXCLUDED_LEGACY | project | error | enforced | project.excluded-legacy | Retired repository paths and contract entries remain excluded. | Remove retired paths and legacy contract entries. |
| PROJECT_LIVE_STRUCTURE | project | error | enforced | project.live-structure | Required repository directories, files, and manifests exist. | Restore the required repository directories, files, and manifests. |
| PROJECT_ORPHAN_GO_WORK_SUM | project | error | enforced | project.go-work-pair | go.work.sum is present only with its go.work owner. | Remove the orphaned go.work.sum or restore its intentional go.work owner. |
| PROJECT_PHASE1_SEMANTICS | project | error | enforced | project.phase1-semantics | Phase-one repository contract semantics are canonical. | Restore the phase-one repository contract semantics. |
| PROJECT_PNPM_WORKSPACE_INVALID | project | error | enforced | project.pnpm-workspace | The root pnpm workspace owns packages only and keeps isolated workspace settings. | Keep pnpm-workspace.yaml scoped to packages/* with the required isolation settings. |
| PROJECT_PROFILE_ROOTS | project | error | enforced | project.profile-roots | Repository profile includes stay inside canonical roots. | Keep profile includes inside canonical repository roots. |
| PROJECT_RESOURCE_ARTIFACTS | project | error | enforced | project.resource-artifacts | Generated resource schema artifacts are present and valid. | Regenerate resource schema artifacts and repair missing resource references. |
| PROJECT_ROOT_NPMRC | project | error | enforced | project.no-root-npmrc | The repository root has no npm configuration that leaks across scenario boundaries. | Remove the root .npmrc or move configuration to its owning boundary. |
| PROJECT_ROOT_PNPM_LOCK | project | error | enforced | project.no-root-pnpm-lock | The repository root has no stray pnpm lockfile. | Remove the root pnpm-lock.yaml; scenario UIs own their lockfiles. |
| PROJECT_RUNTIME_HOME | project | error | enforced | project.runtime-home | The runtime-home contract is canonical. | Restore the runtime-home structural authority. |
| REPO_SCHEMA_ID_UNIQUE | project | error | enforced | project.schema-id-uniqueness | Every repository JSON Schema declares a unique $id. | Delete schema forks or assign a distinct authoritative $id to each genuinely independent schema. |
| RESOURCE_HEALTH_KIND_MISSING | resource | error | enforced | resource.health-checks | Resources declare valid readiness or liveness health checks. | Declare at least one readiness or liveness health check. |
| RESOURCE_IMAGE_UNPINNED | resource | error | enforced | resource.image-pinning | Resource container images are digest pinned. | Pin container images with a sha256 digest. |
| RESOURCE_MANIFEST_INVALID | resource | error | enforced | resource.manifest | Resource manifests are valid and complete. | Repair resource.json so it is valid and complete. |
| RESOURCE_SHELL_FORBIDDEN | resource | error | enforced | resource.go-native-lifecycle | Resource lifecycle is not owned by shell scripts. | Remove shell-owned resource lifecycle files. |
| SAFEGUARD_HANDLER_MISSING | safeguard | error | enforced | safeguard.handler | Declared safeguard handlers exist. | Add the Go handler declared by safeguard.json. |
| SAFEGUARD_MANIFEST_INVALID | safeguard | error | enforced | safeguard.manifest | Safeguard manifests are valid and complete. | Repair safeguard.json so it is valid and complete. |
| SAFEGUARD_NAME_MISMATCH | safeguard | error | enforced | safeguard.identity | Safeguard identity matches its target identifier. | Set safeguard.json.name to the canonical safeguard id. |
| SCENARIO_BUILD_KIND_UNKNOWN | scenario | warning | advisory | scenario.build-kind | Every component build kind is registered by the lifecycle component builder contract. | Use a registered component build kind or add a complete builder-registry implementation before adopting a new kind. |
| SCENARIO_COMPONENT_INVALID | scenario | warning | enforced | scenario.component-contract | Component builds, ports, supervisors, dependencies, reuse edges, and graph cycles resolve within the scenario manifest. | Repair components so every build and process reference resolves without a cycle. |
| SCENARIO_HARDCODED_PEER_ADDRESS | scenario | error | enforced | scenario.no-hardcoded-peer-address | Scenarios with peer dependencies do not hardcode loopback peer ports in component environment values. | Declare a dependencies.scenarios binding for the peer port. |
| SCENARIO_MANIFEST_INVALID | scenario | error | enforced | scenario.manifest | Scenario manifests validate against the canonical service schema. | Repair .vrooli/service.json against .vrooli/schemas/service.schema.json, or correct the schema if it no longer describes the implemented contract. |
| SCENARIO_PEER_BINDING_INVALID | scenario | warning | enforced | scenario.peer-bindings | Scenario manifests do not declare the retired peer-binding environment projection. | Remove dependencies.scenarios[].bindings and resolve peer addresses through discovery. |
| SCENARIO_PORT_ENV_CONVENTION | scenario | error | enforced | scenario.port-env-convention | Every declared port key NAME publishes the environment variable NAME_PORT used by component port ownership. | Rename ports.<name>.env_var to the uppercase port key followed by _PORT and update its consumers. |
| SCENARIO_REDECLARES_RESOURCE_ENV | scenario | error | enforced | scenario.resource-env-authority | Component environment values do not redeclare variables exported by enabled resources. | Delete the duplicate component environment value and consume the resource export. |
| SCENARIO_SECRET_LITERAL | scenario | error | enforced | scenario.no-secret-literals | Component environment declarations contain neither secret-bearing keys nor high-entropy literals. | Declare secret inputs through the credential authority. |
| SCENARIO_SHARED_PACKAGE_BYPASS | scenario | error | enforced | scenario.shared-package-boundary | Scenario UIs consume shared packages through governed compiled outputs instead of package source trees. | Remove packages/*/src aliases and consume the package through its declared file dependency and compiled exports. |
| SCENARIO_SHELL_FORBIDDEN | scenario | error | enforced | scenario.no-shell-lifecycle | Declared scenario lifecycle and component processes never invoke a shell interpreter or shell file. | Replace the declared shell invocation with component metadata, data_dirs, an argv-native command, or a Go CLI subcommand. |
| SCENARIO_UI_BOUNDARY_MISSING | scenario | error | enforced | scenario.ui-boundary | Scenario UIs carry a pnpm workspace boundary file. | Add ui/pnpm-workspace.yaml to stop package-manager discovery at the scenario boundary. |
| SCENARIO_UI_LOCKFILE_MISSING | scenario | error | enforced | scenario.ui-lockfile | Scenario UIs carry a committed lockfile. | Generate and commit the UI lockfile through Scenario Dependency Analyzer. |
| SCENARIO_UI_SERVES_BUILD | scenario | error | enforced | scenario.ui-production-serving | Every declared UI component serves a built bundle rather than a development or preview server. | Build the UI during setup and run a production static server from the component declaration. |
| SCENARIO_WORKSPACE_DEPENDENCY | scenario | error | enforced | scenario.no-workspace-star | Scenario UI dependencies do not use unsupported workspace protocol declarations. | Use an explicit package version or a governed local dependency declaration. |
| TEAM_LAYOUT_MISSING | team | error | enforced | team.layout | Team records have a README and declared sections. | Provide README.md and at least one declared manifest section. |
| TEAM_MANIFEST_INVALID | team | error | enforced | team.manifest | Team manifests are valid and complete. | Add a valid team plan-of-record manifest. |
| TEAM_OWNER_MISMATCH | team | error | enforced | team.identity | Team ownership matches the target identifier. | Align contract.team with the enumerated team target id. |
| TEAM_OWNER_MISSING | team | error | enforced | team.owner | Team manifests declare a stable owner identifier. | Declare contract.team as the stable team target id. |
| TOOL_HANDLER_MISSING | tool | error | enforced | tool.handler | Declared tool handlers exist. | Add the Go handler declared by tool.json. |
| TOOL_MANIFEST_INVALID | tool | error | enforced | tool.manifest | Tool manifests are valid and complete. | Repair tool.json so it is valid and complete. |
| TOOL_NAME_MISMATCH | tool | error | enforced | tool.identity | Tool identity matches its target identifier. | Set tool.json.name to the canonical tool id. |

## Coverage Matrix

| Target kind | Rules | Enforced | Advisory | None | Reachable | Callers |
|---|---:|---:|---:|---:|---|---:|
| scenario | 24 | 22 | 2 | 0 | yes | 2 |
| resource | 4 | 4 | 0 | 0 | yes | 2 |
| tool | 3 | 3 | 0 | 0 | yes | 2 |
| safeguard | 3 | 3 | 0 | 0 | yes | 2 |
| team | 4 | 4 | 0 | 0 | yes | 2 |
| package | 13 | 13 | 0 | 0 | yes | 2 |
| control-plane | 1 | 1 | 0 | 0 | yes | 2 |
| docs | 2 | 2 | 0 | 0 | yes | 2 |
| project | 17 | 17 | 0 | 0 | yes | 2 |
