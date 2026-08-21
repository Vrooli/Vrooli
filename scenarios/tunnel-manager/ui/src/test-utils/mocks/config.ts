import { create, type MessageInitShape } from "@bufbuild/protobuf";
import { vi } from "vitest";
import { resolveApiBase } from "@vrooli/api-base";
import {
  ConfigReadinessSchema,
  CredentialStatusSchema,
  DriftCountsSchema,
  GetConfigResponseSchema,
  GetDriftResponseSchema,
  IngressEntrySchema,
  Mode,
  OwnershipState,
  IngressSource,
  SyncResponseSchema,
  TunnelConfigSchema,
  AccessStatusSchema,
  GetAccessStatusResponseSchema,
  type ConfigReadiness,
  type CredentialStatus,
  type GetConfigResponse,
  type GetDriftResponse,
  type IngressEntry,
  type SyncResponse,
  type TunnelConfig,
  type AccessStatus,
  type GetAccessStatusResponse,
} from "@vrooli/proto-types/tunnel-manager/v1/config/config_pb";

export const makeTunnelConfig = (
  overrides: MessageInitShape<typeof TunnelConfigSchema> = {},
): TunnelConfig =>
  create(TunnelConfigSchema, {
    mode: Mode.LOCAL,
    promEndpoint: "127.0.0.1:20241",
    ...overrides,
  });

export const makeConfigReadiness = (
  overrides: MessageInitShape<typeof ConfigReadinessSchema> = {},
): ConfigReadiness =>
  create(ConfigReadinessSchema, {
    desiredMode: Mode.LOCAL,
    remoteAvailable: false,
    missingFields: ["CLOUDFLARE_API_TOKEN", "CLOUDFLARE_ACCOUNT_ID", "CLOUDFLARE_TUNNEL_ID"],
    credentialSource: "none",
    localConfigPath: "/home/operator/.cloudflared/config.yml",
    syncReady: true,
    modeReason: "Local mode is active until Cloudflare credentials are available.",
    credentialFields: [
      { name: "CLOUDFLARE_API_TOKEN", source: "missing", writable: true },
      { name: "CLOUDFLARE_ACCOUNT_ID", source: "missing", writable: true },
      { name: "CLOUDFLARE_TUNNEL_ID", source: "missing", writable: true },
    ],
    ...overrides,
  });

export const makeCredentialStatus = (
  overrides: MessageInitShape<typeof CredentialStatusSchema> = {},
): CredentialStatus =>
  create(CredentialStatusSchema, {
    source: "missing",
    missingFields: ["CLOUDFLARE_API_TOKEN", "CLOUDFLARE_ACCOUNT_ID", "CLOUDFLARE_TUNNEL_ID"],
    fields: [
      { name: "CLOUDFLARE_API_TOKEN", source: "missing", writable: true },
      { name: "CLOUDFLARE_ACCOUNT_ID", source: "missing", writable: true },
      { name: "CLOUDFLARE_TUNNEL_ID", source: "missing", writable: true },
    ],
    ...overrides,
  });

export const makeConfigResponse = (
  overrides: MessageInitShape<typeof GetConfigResponseSchema> = {},
): GetConfigResponse =>
  create(GetConfigResponseSchema, {
    config: makeTunnelConfig(),
    readiness: makeConfigReadiness(),
    ...overrides,
  });

export const makeSyncResponse = (
  overrides: MessageInitShape<typeof SyncResponseSchema> = {},
): SyncResponse =>
  create(SyncResponseSchema, {
    mode: Mode.LOCAL,
    noChanges: true,
    message: "Ingress already matches the manifest.",
    ...overrides,
  });

export const makeIngressEntry = (
  overrides: MessageInitShape<typeof IngressEntrySchema> = {},
): IngressEntry =>
  create(IngressEntrySchema, {
    hostname: "agent-manager.itsagitime.com",
    serviceTarget: resolveApiBase({ defaultPort: "21001" }),
    state: OwnershipState.MANAGED,
    source: IngressSource.SCENARIO,
    scenario: "agent-manager",
    ...overrides,
  });

export const makeDriftResponse = (
  overrides: MessageInitShape<typeof GetDriftResponseSchema> = {},
): GetDriftResponse =>
  create(GetDriftResponseSchema, {
    mode: Mode.LOCAL,
    entries: [makeIngressEntry()],
    counts: create(DriftCountsSchema, {
      managed: 1,
      missing: 0,
      externalOk: 0,
      orphaned: 0,
      ignored: 0,
      unmanaged: 0,
    }),
    ...overrides,
  });

export const makeAccessStatus = (
  overrides: MessageInitShape<typeof AccessStatusSchema> = {},
): AccessStatus =>
  create(AccessStatusSchema, {
    enabled: false,
    configured: false,
    hosts: [
      {
        host: "web-console.itsagitime.com",
        override: "inherit",
        effectiveBypass: false,
        managed: false,
        appId: "",
      },
    ],
    toCreate: [],
    toRemove: [],
    ...overrides,
  });

export const makeAccessStatusResponse = (
  overrides: MessageInitShape<typeof GetAccessStatusResponseSchema> = {},
): GetAccessStatusResponse =>
  create(GetAccessStatusResponseSchema, {
    status: makeAccessStatus(),
    ...overrides,
  });

export const makeConfigMocks = () => ({
  configClient: {
    getConfig: vi.fn().mockResolvedValue(makeConfigResponse()),
    getCredentialStatus: vi.fn().mockResolvedValue({ status: makeCredentialStatus() }),
    setCloudflareCredentials: vi.fn().mockResolvedValue({ status: makeCredentialStatus({ ready: true, source: "credential-authority", missingFields: [] }) }),
    clearCloudflareCredentials: vi.fn().mockResolvedValue({ status: makeCredentialStatus() }),
    sync: vi.fn().mockResolvedValue(makeSyncResponse()),
    switchMode: vi.fn().mockResolvedValue({
      previousMode: Mode.LOCAL,
      currentMode: Mode.REMOTE,
    }),
    getDrift: vi.fn().mockResolvedValue(makeDriftResponse()),
    adoptIngress: vi.fn().mockResolvedValue({ entry: makeIngressEntry({ state: OwnershipState.MANAGED }) }),
    ignoreIngress: vi.fn().mockResolvedValue({ entry: makeIngressEntry({ state: OwnershipState.IGNORED }) }),
    pruneIngress: vi.fn().mockResolvedValue({ pruned: true }),
    setPublicExposure: vi.fn().mockResolvedValue({ config: makeTunnelConfig({ publicExposureEnabled: true }) }),
    getAccessStatus: vi.fn().mockResolvedValue(makeAccessStatusResponse()),
    verifyCredentials: vi.fn().mockResolvedValue({ ready: true, checks: [] }),
  },
  getConfigState: vi.fn().mockResolvedValue(makeConfigResponse()),
  getConfig: vi.fn().mockResolvedValue(makeTunnelConfig()),
  sync: vi.fn().mockResolvedValue(makeSyncResponse()),
  getDrift: vi.fn().mockResolvedValue(makeDriftResponse()),
});
