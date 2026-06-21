import { createClient } from "@connectrpc/connect";
import {
  ConfigService,
  Mode,
  type ConfigReadiness,
  type CredentialStatus,
  type TunnelConfig,
  type GetCredentialStatusResponse,
  type GetConfigResponse,
  type SetCloudflareCredentialsResponse,
  type ClearCloudflareCredentialsResponse,
  type SyncResponse,
} from "@vrooli/proto-types/tunnel-manager/v1/config/config_pb";

import { transport } from "./client";

// configClient is the generated Connect-Web client for ConfigService —
// Cloudflare ingress + mode management. Read surface for Overview/Settings;
// Sync/SwitchMode are operator actions.
export const configClient = createClient(ConfigService, transport);

/** getConfigState returns persisted config plus browser-safe readiness. */
export async function getConfigState(): Promise<GetConfigResponse> {
  return configClient.getConfig({});
}

/** getConfig returns the persisted tunnel configuration. */
export async function getConfig(): Promise<TunnelConfig | undefined> {
  const resp = await getConfigState();
  return resp.config;
}

/** sync reconciles live ingress with the routes manifest (dryRun previews). */
export async function sync(dryRun = false): Promise<SyncResponse> {
  return configClient.sync({ dryRun });
}

export async function getCredentialStatus(): Promise<GetCredentialStatusResponse> {
  return configClient.getCredentialStatus({});
}

export async function setCloudflareCredentials(values: {
  accountId?: string;
  tunnelId?: string;
  apiToken?: string;
}): Promise<SetCloudflareCredentialsResponse> {
  return configClient.setCloudflareCredentials(values);
}

export async function clearCloudflareCredentials(fields = ["all"]): Promise<ClearCloudflareCredentialsResponse> {
  return configClient.clearCloudflareCredentials({ fields });
}

export { Mode };
export type {
  ClearCloudflareCredentialsResponse,
  ConfigReadiness,
  CredentialStatus,
  GetCredentialStatusResponse,
  GetConfigResponse,
  SetCloudflareCredentialsResponse,
  TunnelConfig,
  SyncResponse,
};
