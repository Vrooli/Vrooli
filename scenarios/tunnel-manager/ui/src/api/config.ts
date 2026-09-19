import { createClient } from "@connectrpc/connect";
import {
  ConfigService,
  Mode,
  OwnershipState,
  IngressSource,
  type ConfigReadiness,
  type CredentialStatus,
  type TunnelConfig,
  type GetCredentialStatusResponse,
  type GetConfigResponse,
  type SetCloudflareCredentialsResponse,
  type ClearCloudflareCredentialsResponse,
  type SyncResponse,
  type GetDriftResponse,
  type IngressEntry,
  type DriftCounts,
  type AdoptIngressResponse,
  type IgnoreIngressResponse,
  type PruneIngressResponse,
  type AccessStatus,
  type AccessHostState,
  type SetPublicExposureResponse,
  type GetAccessStatusResponse,
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

/**
 * sync reconciles live ingress with the routes manifest. Additive by default;
 * `prune` also removes orphaned (ledger-managed-but-gone) hostnames. `dryRun`
 * previews without writing.
 */
export async function sync(options: { dryRun?: boolean; prune?: boolean } = {}): Promise<SyncResponse> {
  return configClient.sync({ dryRun: options.dryRun ?? false, prune: options.prune ?? false });
}

/** getDrift returns the classified ingress entries plus per-state counts. */
export async function getDrift(): Promise<GetDriftResponse> {
  return configClient.getDrift({});
}

/**
 * adoptIngress records an unmanaged hostname into the ledger. Pass `scenario`
 * to adopt as a scenario route, or `target` to adopt as an external route;
 * with neither the service resolves provenance automatically.
 */
export async function adoptIngress(values: {
  hostname: string;
  scenario?: string;
  target?: string;
}): Promise<AdoptIngressResponse> {
  return configClient.adoptIngress(values);
}

/** ignoreIngress acknowledges an external hostname so drift stops flagging it. */
export async function ignoreIngress(values: { hostname: string; note?: string }): Promise<IgnoreIngressResponse> {
  return configClient.ignoreIngress(values);
}

/** pruneIngress removes a single hostname from live ingress and/or the ledger. */
export async function pruneIngress(hostname: string): Promise<PruneIngressResponse> {
  return configClient.pruneIngress({ hostname });
}

export async function getCredentialStatus(): Promise<GetCredentialStatusResponse> {
  return configClient.getCredentialStatus({});
}

/**
 * setPublicExposure flips the global /public Access-bypass switch. When off
 * (the default) no per-host bypass apps are reconciled regardless of per-route
 * overrides set to `inherit`. Returns the persisted config after the flip.
 */
export async function setPublicExposure(enabled: boolean): Promise<SetPublicExposureResponse> {
  return configClient.setPublicExposure({ enabled });
}

/**
 * getAccessStatus returns the /public Access-bypass read model: the global
 * switch, whether the Cloudflare Access client is configured, the per-host
 * effective decisions, and the dry-run plan (to-create / to-remove). This is a
 * pure read — it never mutates Cloudflare.
 */
export async function getAccessStatus(): Promise<GetAccessStatusResponse> {
  return configClient.getAccessStatus({});
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

export { Mode, OwnershipState, IngressSource };
export type {
  ClearCloudflareCredentialsResponse,
  ConfigReadiness,
  CredentialStatus,
  GetCredentialStatusResponse,
  GetConfigResponse,
  SetCloudflareCredentialsResponse,
  TunnelConfig,
  SyncResponse,
  GetDriftResponse,
  IngressEntry,
  DriftCounts,
  AdoptIngressResponse,
  IgnoreIngressResponse,
  PruneIngressResponse,
  AccessStatus,
  AccessHostState,
  SetPublicExposureResponse,
  GetAccessStatusResponse,
};
