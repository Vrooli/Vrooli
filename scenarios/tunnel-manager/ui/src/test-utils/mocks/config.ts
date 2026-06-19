import { create, type MessageInitShape } from "@bufbuild/protobuf";
import { vi } from "vitest";
import {
  ConfigReadinessSchema,
  GetConfigResponseSchema,
  Mode,
  SyncResponseSchema,
  TunnelConfigSchema,
  type ConfigReadiness,
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
