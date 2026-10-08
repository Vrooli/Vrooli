# Environment Management

This page describes the current project-level environment and secrets posture.

## Current Truth

Environment and secrets handling in Vrooli is a mix of:

- project-level configuration under `.vrooli/`
- project and scenario manifests
- host-tool and setup requirements declared in manifests and setup flows
- deployment-tier-specific secrets behavior described in the Deployment Hub

This is not a stable “one environment model for everything” story yet, so avoid treating older environment docs as universally canonical.

## What To Use Today

### The one command that configures or heals a host

```bash
vrooli setup --include-optional --maintenance-window --sudo-mode=ask --onboarding=auto
```

Setup performs the bootstrap steps and hands off to `vrooli-onboarding`, which
asks for every value the operator has to supply: credentials, host-tool and
safeguard consent, and the operating mode. Anything an operator has to do
outside this command is a defect, not a workflow.

What each flag contributes:

| Flag | Effect |
|---|---|
| `--include-optional` | Applies optional safeguards as well as required ones. Without it, optional items are listed but not installed. |
| `--maintenance-window` | Acknowledges that a safeguard may interrupt a graphical or remote session. |
| `--sudo-mode=ask` | Lets the in-process `sudo` wrapper prompt when a host item needs privilege, on an interactive run. The default is `skip`, which lists privileged items instead of installing them. An already-elevated invocation also installs them, but it runs the whole of setup as root and is outside this flow. |
| `--onboarding=auto` | Opens a browser when the invoking session can show one, and otherwise prints the URL and an exact resume command. |

Setup verifies before it reports. Its last line states a verified readiness
verdict rather than an unconditional success, and `--result-file` writes that
verdict into the machine-readable result document under `readiness`. A present
configuration marker no longer implies success on its own: when the verdict
contradicts it, setup reports `configuration_pending` and names this command.

### Inspection

```bash
vrooli setup --help
vrooli setup status
vrooli develop --help
vrooli status
vrooli doctor
```

`vrooli setup status` prints the readiness verdict beside the configuration
marker line and applies no host change.

These commands are the best current entrypoints for understanding environment expectations, tool availability, and setup posture.

### Resource enablement precedence

The effective resource catalog combines the project `.vrooli/service.json`
with operator state. An explicit project or operator `enabled: false` wins over
a scenario dependency declaration, including `startup_policy: "try_start"`.
The scenario remains free to declare the dependency, but the control plane
does not start a resource the operator disabled. `vrooli resource status --json`
reports `disabled_dependency_consumers` with each declaring scenario and its
startup policy so the decision is observable.

Setup progress is intentionally safe in non-interactive environments. It emits
plain newline-delimited diagnostics by default, with optional structured events
via `VROOLI_SETUP_PROGRESS_FORMAT=json` and suppression via
`VROOLI_SETUP_PROGRESS_FORMAT=quiet`. This is separate from the terminal setup
result written by `--result-file`.

### Control-plane timing overrides

Every control-plane timing lever can be raised or lowered without rebuilding.
Set `VROOLI_TUNING_<LEVER_NAME>` to a Go duration such as `750ms`, `9s`, or
`3m`. The lever name is the upper-snake-case form of the accessor in
`internal/tuning/timing.go`. For example:

```bash
VROOLI_TUNING_HEALTH_CHECK_TIMEOUT=9s vrooli scenario status test-genie --json
```

The environment is read once per lever in each process. An absent value uses
the compiled default. A malformed value emits one warning and also uses the
compiled default; it never prevents setup or another control-plane command from
starting. Runtime- or manifest-provided bounds remain the fallback when their
operator override is absent.

The complete timing override surface is generated from `vrooli tuning list --json`:

<!-- BEGIN GENERATED TUNING LEVERS -->
| Lever | Kind | Environment variable | Compiled default | Description |
| --- | --- | --- | --- | --- |
| `ActivityDebounce` | duration | `VROOLI_TUNING_ACTIVITY_DEBOUNCE` | `5s` | Controls activity debounce. |
| `ActivityReadHeaderTimeout` | duration | `VROOLI_TUNING_ACTIVITY_READ_HEADER_TIMEOUT` | `15s` | Controls activity read header timeout. |
| `AgentInstallDownloadTimeout` | duration | `VROOLI_TUNING_AGENT_INSTALL_DOWNLOAD_TIMEOUT` | `10m0s` | Controls agent install download timeout. |
| `ArtifactRetentionWindow` | duration | `VROOLI_TUNING_ARTIFACT_RETENTION_WINDOW` | `720h0m0s` | Controls artifact retention window. |
| `AttestationValidityWindow` | duration | `VROOLI_TUNING_ATTESTATION_VALIDITY_WINDOW` | `744h0m0s` | Controls attestation validity window. |
| `BackgroundLaunchPollInterval` | duration | `VROOLI_TUNING_BACKGROUND_LAUNCH_POLL_INTERVAL` | `50ms` | Controls background launch poll interval. |
| `BuildWidth` | count | `VROOLI_TUNING_BUILD_WIDTH` | `4 processes` | Controls build width. |
| `CapabilityRequestTimeout` | duration | `VROOLI_TUNING_CAPABILITY_REQUEST_TIMEOUT` | `30s` | Controls capability request timeout. |
| `CapacityDegradeDebounce` | duration | `VROOLI_TUNING_CAPACITY_DEGRADE_DEBOUNCE` | `30s` | Controls capacity degrade debounce. |
| `CapacityHeartbeatTTL` | duration | `VROOLI_TUNING_CAPACITY_HEARTBEAT_TTL` | `30s` | Controls capacity heartbeat ttl. |
| `CapacityObservedPeakHalflife` | duration | `VROOLI_TUNING_CAPACITY_OBSERVED_PEAK_HALFLIFE` | `10m0s` | Controls capacity observed peak halflife. |
| `CompanionCapacitySyncInterval` | duration | `VROOLI_TUNING_COMPANION_CAPACITY_SYNC_INTERVAL` | `15s` | Controls companion capacity sync interval. |
| `CompanionCrashWindow` | duration | `VROOLI_TUNING_COMPANION_CRASH_WINDOW` | `10m0s` | Controls companion crash window. |
| `CompanionHeartbeatInterval` | duration | `VROOLI_TUNING_COMPANION_HEARTBEAT_INTERVAL` | `1m0s` | Controls companion heartbeat interval. |
| `ControlPlaneClientTimeout` | duration | `VROOLI_TUNING_CONTROL_PLANE_CLIENT_TIMEOUT` | `10s` | Controls control plane client timeout. |
| `CopyRetentionWindow` | duration | `VROOLI_TUNING_COPY_RETENTION_WINDOW` | `15m0s` | Controls copy retention window. |
| `CredentialEscrowRetention` | duration | `VROOLI_TUNING_CREDENTIAL_ESCROW_RETENTION` | `10m0s` | Controls credential escrow retention. |
| `CredentialReloadFallbackDelay` | duration | `VROOLI_TUNING_CREDENTIAL_RELOAD_FALLBACK_DELAY` | `2m0s` | Controls credential reload fallback delay. |
| `CredentialReloadOperationTimeout` | duration | `VROOLI_TUNING_CREDENTIAL_RELOAD_OPERATION_TIMEOUT` | `caller-provided` | Controls credential reload operation timeout. |
| `CredentialRepairTimeout` | duration | `VROOLI_TUNING_CREDENTIAL_REPAIR_TIMEOUT` | `1m30s` | Controls credential repair timeout. |
| `CredentialServiceTimeout` | duration | `VROOLI_TUNING_CREDENTIAL_SERVICE_TIMEOUT` | `15s` | Controls credential service timeout. |
| `CredentialStoreCommandTimeout` | duration | `VROOLI_TUNING_CREDENTIAL_STORE_COMMAND_TIMEOUT` | `caller-provided` | Controls credential store command timeout. |
| `CredentialStoreProbeTimeout` | duration | `VROOLI_TUNING_CREDENTIAL_STORE_PROBE_TIMEOUT` | `1s` | Controls credential store probe timeout. |
| `CredentialUnitProbeTimeout` | duration | `VROOLI_TUNING_CREDENTIAL_UNIT_PROBE_TIMEOUT` | `3s` | Controls credential unit probe timeout. |
| `DailyRetentionWindow` | duration | `VROOLI_TUNING_DAILY_RETENTION_WINDOW` | `24h0m0s` | Controls daily retention window. |
| `DependencyBestEffortStartTimeout` | duration | `VROOLI_TUNING_DEPENDENCY_BEST_EFFORT_START_TIMEOUT` | `15s` | Controls dependency best effort start timeout. |
| `DockerRuntimeEnvironmentTimeout` | duration | `VROOLI_TUNING_DOCKER_RUNTIME_ENVIRONMENT_TIMEOUT` | `caller-provided` | Controls docker runtime environment timeout. |
| `DockerRuntimeOperationTimeout` | duration | `VROOLI_TUNING_DOCKER_RUNTIME_OPERATION_TIMEOUT` | `2m0s` | Controls docker runtime operation timeout. |
| `EagerScenarioWaitWindow` | duration | `VROOLI_TUNING_EAGER_SCENARIO_WAIT_WINDOW` | `30s` | Controls eager scenario wait window. |
| `EmergencyWatchdogInterval` | duration | `VROOLI_TUNING_EMERGENCY_WATCHDOG_INTERVAL` | `5m0s` | Controls emergency watchdog interval. |
| `EphemeralPortProbeTimeout` | duration | `VROOLI_TUNING_EPHEMERAL_PORT_PROBE_TIMEOUT` | `2s` | Controls ephemeral port probe timeout. |
| `FastHealthPollInterval` | duration | `VROOLI_TUNING_FAST_HEALTH_POLL_INTERVAL` | `250ms` | Controls fast health poll interval. |
| `FastPersistenceRetryInterval` | duration | `VROOLI_TUNING_FAST_PERSISTENCE_RETRY_INTERVAL` | `25ms` | Controls fast persistence retry interval. |
| `FreshnessCheckBudget` | duration | `VROOLI_TUNING_FRESHNESS_CHECK_BUDGET` | `5s` | Controls freshness check budget. |
| `HealthCheckTimeout` | duration | `VROOLI_TUNING_HEALTH_CHECK_TIMEOUT` | `3s` | Controls health check timeout. |
| `HealthProbeInterval` | duration | `VROOLI_TUNING_HEALTH_PROBE_INTERVAL` | `500ms` | Controls health probe interval. |
| `HostFactsRetryInterval` | duration | `VROOLI_TUNING_HOST_FACTS_RETRY_INTERVAL` | `10ms` | Controls host facts retry interval. |
| `HostGPUInventoryTTL` | duration | `VROOLI_TUNING_HOST_GPU_INVENTORY_TTL` | `2m0s` | Controls host gpu inventory ttl. |
| `HostInventoryTTL` | duration | `VROOLI_TUNING_HOST_INVENTORY_TTL` | `30s` | Controls host inventory ttl. |
| `HostPlatformInventoryTTL` | duration | `VROOLI_TUNING_HOST_PLATFORM_INVENTORY_TTL` | `5m0s` | Controls host platform inventory ttl. |
| `HostPresentationCommandTimeout` | duration | `VROOLI_TUNING_HOST_PRESENTATION_COMMAND_TIMEOUT` | `2s` | Controls host presentation command timeout. |
| `HostPresentationProbeTimeout` | duration | `VROOLI_TUNING_HOST_PRESENTATION_PROBE_TIMEOUT` | `750ms` | Controls host presentation probe timeout. |
| `HostRequirementCommandTimeout` | duration | `VROOLI_TUNING_HOST_REQUIREMENT_COMMAND_TIMEOUT` | `30s` | Controls host requirement command timeout. |
| `HostWorkloadInventoryTTL` | duration | `VROOLI_TUNING_HOST_WORKLOAD_INVENTORY_TTL` | `5m0s` | Controls host workload inventory ttl. |
| `HygieneProviderExecutionBudget` | duration | `VROOLI_TUNING_HYGIENE_PROVIDER_EXECUTION_BUDGET` | `caller-provided` | Controls hygiene provider execution budget. |
| `IntegrityCollectionTimeout` | duration | `VROOLI_TUNING_INTEGRITY_COLLECTION_TIMEOUT` | `8s` | Controls integrity collection timeout. |
| `LifecycleExtendedOperationTimeout` | duration | `VROOLI_TUNING_LIFECYCLE_EXTENDED_OPERATION_TIMEOUT` | `2m0s` | Controls lifecycle extended operation timeout. |
| `LifecycleHealthPollInterval` | duration | `VROOLI_TUNING_LIFECYCLE_HEALTH_POLL_INTERVAL` | `1s` | Controls lifecycle health poll interval. |
| `LifecycleOperationTimeout` | duration | `VROOLI_TUNING_LIFECYCLE_OPERATION_TIMEOUT` | `30s` | Controls lifecycle operation timeout. |
| `LifecyclePollInterval` | duration | `VROOLI_TUNING_LIFECYCLE_POLL_INTERVAL` | `100ms` | Controls lifecycle poll interval. |
| `LifecyclePollMaxInterval` | duration | `VROOLI_TUNING_LIFECYCLE_POLL_MAX_INTERVAL` | `2s` | Controls lifecycle poll max interval. |
| `LifecycleTransitionTimeout` | duration | `VROOLI_TUNING_LIFECYCLE_TRANSITION_TIMEOUT` | `2s` | Controls lifecycle transition timeout. |
| `ListenerEnrichmentTimeout` | duration | `VROOLI_TUNING_LISTENER_ENRICHMENT_TIMEOUT` | `3s` | Controls listener enrichment timeout. |
| `MaintenanceSettleDelay` | duration | `VROOLI_TUNING_MAINTENANCE_SETTLE_DELAY` | `150ms` | Controls maintenance settle delay. |
| `ManagedServiceConfiguredTimeout` | duration | `VROOLI_TUNING_MANAGED_SERVICE_CONFIGURED_TIMEOUT` | `caller-provided` | Controls managed service configured timeout. |
| `ManagedServiceForceStopTimeout` | duration | `VROOLI_TUNING_MANAGED_SERVICE_FORCE_STOP_TIMEOUT` | `2s` | Controls managed service force stop timeout. |
| `PlatformSupportRequestTimeout` | duration | `VROOLI_TUNING_PLATFORM_SUPPORT_REQUEST_TIMEOUT` | `30s` | Controls platform support request timeout. |
| `PrivilegeBrokerOperationTimeout` | duration | `VROOLI_TUNING_PRIVILEGE_BROKER_OPERATION_TIMEOUT` | `2m0s` | Controls privilege broker operation timeout. |
| `PrivilegeBrokerRequestTimeout` | duration | `VROOLI_TUNING_PRIVILEGE_BROKER_REQUEST_TIMEOUT` | `caller-provided` | Controls privilege broker request timeout. |
| `PrivilegeBrokerUnlockTimeout` | duration | `VROOLI_TUNING_PRIVILEGE_BROKER_UNLOCK_TIMEOUT` | `2s` | Controls privilege broker unlock timeout. |
| `ProcessHealthCheckTimeout` | duration | `VROOLI_TUNING_PROCESS_HEALTH_CHECK_TIMEOUT` | `caller-provided` | Controls process health check timeout. |
| `ProgressDisplayResolution` | duration | `VROOLI_TUNING_PROGRESS_DISPLAY_RESOLUTION` | `100ms` | Controls progress display resolution. |
| `ProviderBudget` | duration | `VROOLI_TUNING_PROVIDER_BUDGET` | `3m0s` | Controls provider budget. |
| `ReachabilityTimeout` | duration | `VROOLI_TUNING_REACHABILITY_TIMEOUT` | `1m0s` | Controls reachability timeout. |
| `ReloadFallbackGracePeriod` | duration | `VROOLI_TUNING_RELOAD_FALLBACK_GRACE_PERIOD` | `20s` | Controls reload fallback grace period. |
| `RemoteDesktopProbeTimeout` | duration | `VROOLI_TUNING_REMOTE_DESKTOP_PROBE_TIMEOUT` | `5s` | Controls remote desktop probe timeout. |
| `RepairDeadline` | duration | `VROOLI_TUNING_REPAIR_DEADLINE` | `30m0s` | Controls repair deadline. |
| `ResourceCommandTimeout` | duration | `VROOLI_TUNING_RESOURCE_COMMAND_TIMEOUT` | `10m0s` | Controls resource command timeout. |
| `ResourceControlExtendedTimeout` | duration | `VROOLI_TUNING_RESOURCE_CONTROL_EXTENDED_TIMEOUT` | `2m0s` | Controls resource control extended timeout. |
| `ResourceControlTimeout` | duration | `VROOLI_TUNING_RESOURCE_CONTROL_TIMEOUT` | `30s` | Controls resource control timeout. |
| `ResourceHTTPTimeout` | duration | `VROOLI_TUNING_RESOURCE_HTTP_TIMEOUT` | `1m0s` | Controls resource http timeout. |
| `ResourceHealthCheckTimeout` | duration | `VROOLI_TUNING_RESOURCE_HEALTH_CHECK_TIMEOUT` | `caller-provided` | Controls resource health check timeout. |
| `ResourceLongHTTPTimeout` | duration | `VROOLI_TUNING_RESOURCE_LONG_HTTP_TIMEOUT` | `15m0s` | Controls resource long http timeout. |
| `ResourceMediumHTTPTimeout` | duration | `VROOLI_TUNING_RESOURCE_MEDIUM_HTTP_TIMEOUT` | `30s` | Controls resource medium http timeout. |
| `ResourceOperationTimeout` | duration | `VROOLI_TUNING_RESOURCE_OPERATION_TIMEOUT` | `caller-provided` | Controls resource operation timeout. |
| `ResourceShortHTTPTimeout` | duration | `VROOLI_TUNING_RESOURCE_SHORT_HTTP_TIMEOUT` | `10s` | Controls resource short http timeout. |
| `ScenarioActionTimeout` | duration | `VROOLI_TUNING_SCENARIO_ACTION_TIMEOUT` | `caller-provided` | Controls scenario action timeout. |
| `ScenarioHeartbeatTTL` | duration | `VROOLI_TUNING_SCENARIO_HEARTBEAT_TTL` | `30s` | Controls scenario heartbeat ttl. |
| `ScenarioRequirementsSnapshotTimeout` | duration | `VROOLI_TUNING_SCENARIO_REQUIREMENTS_SNAPSHOT_TIMEOUT` | `5s` | Controls scenario requirements snapshot timeout. |
| `ScenarioReservedClaimTTL` | duration | `VROOLI_TUNING_SCENARIO_RESERVED_CLAIM_TTL` | `5m0s` | Controls scenario reserved claim ttl. |
| `ScenarioWaitTimeout` | duration | `VROOLI_TUNING_SCENARIO_WAIT_TIMEOUT` | `10m0s` | Controls scenario wait timeout. |
| `SecretToolTimeout` | duration | `VROOLI_TUNING_SECRET_TOOL_TIMEOUT` | `15s` | Controls secret tool timeout. |
| `ServiceHealthTimeout` | duration | `VROOLI_TUNING_SERVICE_HEALTH_TIMEOUT` | `5s` | Controls service health timeout. |
| `SetupExtendedOperationTimeout` | duration | `VROOLI_TUNING_SETUP_EXTENDED_OPERATION_TIMEOUT` | `2m0s` | Controls setup extended operation timeout. |
| `SetupFilesystemSettleDelay` | duration | `VROOLI_TUNING_SETUP_FILESYSTEM_SETTLE_DELAY` | `200ms` | Controls setup filesystem settle delay. |
| `SetupHTTPProbeTimeout` | duration | `VROOLI_TUNING_SETUP_HTTP_PROBE_TIMEOUT` | `2s` | Controls setup http probe timeout. |
| `SetupOperationTimeout` | duration | `VROOLI_TUNING_SETUP_OPERATION_TIMEOUT` | `30s` | Controls setup operation timeout. |
| `SetupProgressObservationInterval` | duration | `VROOLI_TUNING_SETUP_PROGRESS_OBSERVATION_INTERVAL` | `30s` | Controls setup progress observation interval. |
| `SetupProgressPollInterval` | duration | `VROOLI_TUNING_SETUP_PROGRESS_POLL_INTERVAL` | `1s` | Controls setup progress poll interval. |
| `SetupProgressStaleThreshold` | duration | `VROOLI_TUNING_SETUP_PROGRESS_STALE_THRESHOLD` | `5m0s` | Controls setup progress stale threshold. |
| `StructureProviderBudget` | duration | `VROOLI_TUNING_STRUCTURE_PROVIDER_BUDGET` | `30s` | Controls structure provider budget. |
| `StructureProviderCallTimeout` | duration | `VROOLI_TUNING_STRUCTURE_PROVIDER_CALL_TIMEOUT` | `caller-provided` | Controls structure provider call timeout. |
| `StructureProviderExtendedBudget` | duration | `VROOLI_TUNING_STRUCTURE_PROVIDER_EXTENDED_BUDGET` | `2m0s` | Controls structure provider extended budget. |
| `SupervisorHealthInterval` | duration | `VROOLI_TUNING_SUPERVISOR_HEALTH_INTERVAL` | `45s` | Controls supervisor health interval. |
| `SupervisorRecoveryCooldown` | duration | `VROOLI_TUNING_SUPERVISOR_RECOVERY_COOLDOWN` | `5m0s` | Controls supervisor recovery cooldown. |
| `SupervisorRecoveryQuietPeriod` | duration | `VROOLI_TUNING_SUPERVISOR_RECOVERY_QUIET_PERIOD` | `2m0s` | Controls supervisor recovery quiet period. |
| `TerminalClaimRetention` | duration | `VROOLI_TUNING_TERMINAL_CLAIM_RETENTION` | `336h0m0s` | Controls terminal claim retention. |
| `TidinessProviderBudget` | duration | `VROOLI_TUNING_TIDINESS_PROVIDER_BUDGET` | `2m0s` | Controls tidiness provider budget. |
| `TidinessProviderCallTimeout` | duration | `VROOLI_TUNING_TIDINESS_PROVIDER_CALL_TIMEOUT` | `caller-provided` | Controls tidiness provider call timeout. |
| `VaultBootstrapLease` | duration | `VROOLI_TUNING_VAULT_BOOTSTRAP_LEASE` | `5m0s` | Controls vault bootstrap lease. |
<!-- END GENERATED TUNING LEVERS -->

For deployment-tier-specific secrets thinking:

- [../deployment/README.md](../deployment/README.md)

For scenario-specific runtime expectations:

- the scenario's own `.vrooli/service.json`
- scenario-local docs when the behavior is specific to that scenario

## Guidance

- Treat project configuration under `.vrooli/` as part of the project-level operational truth.
- Treat scenario-local `.vrooli/service.json` files as scenario-specific operational truth.
- Avoid documenting environment behavior in old shell-script terms unless that specific path still exists and is intentionally supported.
- Be careful with secrets claims: current behavior varies by setup path and deployment tier.
- Keep host requirements, setup behavior, and secret handling distinct. They overlap, but they are not one single unified concern.

## Status

This page is intentionally conservative until a fuller environment-and-secrets rewrite happens, but it is still the canonical project-level reference for the current posture.

## Platform evidence

Platform-specific setup and lifecycle claims use the canonical
[platform support matrix](platform-support.md). In particular, macOS is not
hardware-qualified merely because a setup or cross-build command is available.

## Host disk floor (operator steps)

> Added after the 2026-07-31 disk-exhaustion incident, in which the host root
> filesystem reached 100 percent, the runtime supervisor crash-looped about 110
> times in nine minutes, rsyslog lost log data, and the emergency watchdog
> itself failed to write.

The Go safeguards now detect and remediate disk pressure on their own. The
steps below are the host-level floor underneath them, and they need root, so
**an operator applies them deliberately — no Vrooli process does this silently.**

### 1. Age Vrooli build staging in `/tmp` at three days

The default policy cannot work. `systemd-tmpfiles-clean.timer` fires daily, but
`/usr/lib/tmpfiles.d/tmp.conf` ages `/tmp` at 30 days. At incident time exactly
one entry in `/tmp` was older than thirty days while 116 GB across 43,647
entries was younger: build-staging directories of about 2.4 GB each are created
far faster than a 30-day TTL can ever retire them.

This override shortens **only** the Vrooli staging patterns and leaves every
other `/tmp` path on the default 30-day rule.

```bash
sudo tee /etc/tmpfiles.d/vrooli-tmp.conf >/dev/null <<'EOF'
# Vrooli build-staging debris ages at 3 days, not the /tmp default of 30.
# These directories are ~2.4 GB each and are recreated on every release build,
# so a 30-day TTL can never retire them faster than they accumulate.
# Everything else in /tmp keeps the default rule from /usr/lib/tmpfiles.d/tmp.conf.
d /tmp/vrooli-release          1777 root root 3d
e /tmp/vrooli-.*-release       -    -    -    3d
e /tmp/vrooli-resource-release -    -    -    3d
e /tmp/tmp.*                   -    -    -    3d
EOF

# Read back what the new rule would clean, without cleaning anything:
sudo systemd-tmpfiles --dry-run --clean 2>&1 | grep -E 'vrooli|tmp\.' | head -20

# Apply when the dry run looks right:
sudo systemd-tmpfiles --clean
```

**Never shorten `systemd-private-*`.** Those directories carry the current boot
id despite old mtimes, and deleting them breaks running services.

### 2. Reserve a floor on the root filesystem

Filling the data areas must not make the root filesystem unwritable for the
supervisor and the watchdog. ext4 reserves 5 percent for root by default; on a
1.8 TB filesystem that is about 93 GB, which is what kept `df` reporting 93
percent while a `Bfree`-based check reported 87.

Confirm the reserve is intact:

```bash
sudo tune2fs -l /dev/nvme0n1p2 | grep -i 'reserved block count'
```

If the reserve has been reduced to zero (a common space-saving change), restore
it to 1 percent — enough for root-owned writes without giving up much capacity:

```bash
sudo tune2fs -m 1 /dev/nvme0n1p2
```

The watchdog enforces a second, unprivileged floor above this one. It is
configurable and defaults to 10 GiB:

```bash
EMERGENCY_WATCHDOG_DISK_FLOOR_MB=10240   # request cleanup below this
EMERGENCY_WATCHDOG_DISK_THRESHOLD=120    # seconds below the floor before escalating
EMERGENCY_WATCHDOG_MOUNT=/               # root and home share one filesystem here
```

Below the floor the watchdog requests a `high`-band cleanup; below half the
floor it requests `critical`, which permits unattended safe-tier reclamation.

### 3. Bound the journal

The journal was never vacuumed during the incident because it needs root:

```bash
sudo journalctl --vacuum-size=200M
sudo sed -i 's/^#\?SystemMaxUse=.*/SystemMaxUse=500M/' /etc/systemd/journald.conf
sudo systemctl restart systemd-journald
```

### 4. Remove the dead cron entry

`*/5 * * * * ./automation/scheduled-monitor.sh main` has never executed: the
path is relative and cron runs from `$HOME`, while the script lived under
`scenarios/system-monitor/`. Repairing it would be wrong even so — it targeted
port 8080 (the live port is 16914) and checked only CPU and memory, never disk.
The system-monitor threshold loop now owns that job.

```bash
crontab -l | grep -v 'automation/scheduled-monitor.sh' | crontab -
crontab -l   # confirm no entry references a relative path
```

### Rehearsal

Never rehearse against the live root filesystem. Drive the watchdog's disk
logic against a synthetic `df` and a stubbed `storage-manager` instead, or a
loopback mount if one can be created:

```bash
truncate -s 200M /tmp/vrooli-rehearsal.img
mkfs.ext4 -q /tmp/vrooli-rehearsal.img
mkdir -p /tmp/vrooli-rehearsal-mnt
sudo mount -o loop /tmp/vrooli-rehearsal.img /tmp/vrooli-rehearsal-mnt

# Fill past each band and observe the watchdog's decisions:
EMERGENCY_WATCHDOG_MOUNT=/tmp/vrooli-rehearsal-mnt \
EMERGENCY_WATCHDOG_DISK_FLOOR_MB=100 \
EMERGENCY_WATCHDOG_DISK_THRESHOLD=0 \
  sh ~/.vrooli/libexec/emergency-watchdog.sh

Rehearse memory and fork-rate findings without filling the live filesystem by
running the installed binary against captured inputs:

```bash
vrooli-watchdog --report-only \
  --fixtures internal/hostpressure/testdata/host-2026-08-22
```

The fixture path exercises CPU, stranded-memory, fork-rate, workload ownership,
and crash-loop evidence while leaving the host, swap policy, and disk untouched.

sudo umount /tmp/vrooli-rehearsal-mnt && rm -f /tmp/vrooli-rehearsal.img
```

### Reading current disk pressure

One command reports usage, the active band, the last violation, and the last
remediation result:

```bash
curl -s http://localhost:16914/api/v1/disk-pressure | jq
```
