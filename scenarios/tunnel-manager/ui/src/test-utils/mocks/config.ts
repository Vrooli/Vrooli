import { create, type MessageInitShape } from "@bufbuild/protobuf";
import { vi } from "vitest";
import {
  ConfigReadinessSchema,
  CredentialStatusSchema,
  GetConfigResponseSchema,
  Mode,
  SyncResponseSchema,
  TunnelConfigSchema,
  type ConfigReadiness,
  type CredentialStatus,
  type GetConfigResponse,
  type SyncResponse,
  type TunnelConfig,
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

export const makeConfigMocks = () => ({
  configClient: {
    getConfig: vi.fn().mockResolvedValue(makeConfigResponse()),
    getCredentialStatus: vi.fn().mockResolvedValue({ status: makeCredentialStatus() }),
    setCloudflareCredentials: vi.fn().mockResolvedValue({ status: makeCredentialStatus({ ready: true, source: "file:scenario", missingFields: [] }) }),
    clearCloudflareCredentials: vi.fn().mockResolvedValue({ status: makeCredentialStatus() }),
    sync: vi.fn().mockResolvedValue(makeSyncResponse()),
    switchMode: vi.fn().mockResolvedValue({
      previousMode: Mode.LOCAL,
      currentMode: Mode.REMOTE,
    }),
  },
  getConfigState: vi.fn().mockResolvedValue(makeConfigResponse()),
  getConfig: vi.fn().mockResolvedValue(makeTunnelConfig()),
  sync: vi.fn().mockResolvedValue(makeSyncResponse()),
});
