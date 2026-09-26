/**
 * ProviderLifecycleService Connect-RPC wrapper.
 *
 * Thin adapter over the generated Connect client; the UI never
 * hand-types proto shapes. Mutating helpers accept an optional
 * `dryRun` flag and forward it as the canonical X-Dry-Run header.
 */
import { createClient } from "@connectrpc/connect";
import {
  ProviderLifecycleService,
  type GetProviderLogsRequest,
  type ListLocalProvidersResponse,
  type LogLine,
  type PullModelResponse,
  type RestartProviderResponse,
  type StartProviderResponse,
  type StopProviderResponse,
} from "@vrooli/proto-types/audio-tools/v1/provider_lifecycle/provider_lifecycle_pb";

import { transport } from "./client";

const client = createClient(ProviderLifecycleService, transport);

const DRY_RUN_HEADER = "X-Dry-Run";

function dryRunHeaders(dryRun?: boolean): HeadersInit | undefined {
  if (!dryRun) return undefined;
  return { [DRY_RUN_HEADER]: "true" };
}

export async function listLocalProviders(): Promise<ListLocalProvidersResponse> {
  return client.listLocalProviders({});
}

export async function startProvider(
  providerId: string,
  dryRun?: boolean,
): Promise<StartProviderResponse> {
  return client.startProvider({ providerId }, { headers: dryRunHeaders(dryRun) });
}

export async function stopProvider(
  providerId: string,
  dryRun?: boolean,
): Promise<StopProviderResponse> {
  return client.stopProvider({ providerId }, { headers: dryRunHeaders(dryRun) });
}

export async function restartProvider(
  providerId: string,
  dryRun?: boolean,
): Promise<RestartProviderResponse> {
  return client.restartProvider({ providerId }, { headers: dryRunHeaders(dryRun) });
}

export async function pullModel(
  modelName: string,
  dryRun?: boolean,
): Promise<PullModelResponse> {
  return client.pullModel(
    { providerId: "ollama", modelName },
    { headers: dryRunHeaders(dryRun) },
  );
}

export function streamProviderLogs(
  req: Partial<GetProviderLogsRequest> & { providerId: string },
  signal: AbortSignal,
): AsyncIterable<LogLine> {
  return client.getProviderLogs(
    {
      providerId: req.providerId,
      follow: req.follow ?? false,
      tailLines: req.tailLines ?? 0,
    },
    { signal },
  );
}

export type {
  ListLocalProvidersResponse,
  LogLine,
  PullModelResponse,
  RestartProviderResponse,
  StartProviderResponse,
  StopProviderResponse,
};
