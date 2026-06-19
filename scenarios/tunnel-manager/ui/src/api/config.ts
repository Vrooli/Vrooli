import { createClient } from "@connectrpc/connect";
import {
  ConfigService,
  Mode,
  type TunnelConfig,
  type SyncResponse,
} from "@vrooli/proto-types/tunnel-manager/v1/config/config_pb";

import { transport } from "./client";

// configClient is the generated Connect-Web client for ConfigService —
// Cloudflare ingress + mode management. Read surface for Overview/Settings;
// Sync/SwitchMode are operator actions.
export const configClient = createClient(ConfigService, transport);

/** getConfig returns the persisted tunnel configuration. */
export async function getConfig(): Promise<TunnelConfig | undefined> {
  const resp = await configClient.getConfig({});
  return resp.config;
}

/** sync reconciles live ingress with the routes manifest (dryRun previews). */
export async function sync(dryRun = false): Promise<SyncResponse> {
  return configClient.sync({ dryRun });
}

export { Mode };
export type { TunnelConfig, SyncResponse };
