import { createClient } from "@connectrpc/connect";
import {
  DiscoveryService,
  type DiscoveryResult,
  type DiscoverySource,
  type DraftBrand,
} from "@vrooli/proto-types/brand-manager/v1/discovery/discovery_pb";

import { transport } from "./client";

export const discoveryClient = createClient(DiscoveryService, transport);

/**
 * discoverScenario scans a scenario for branding state and returns the draft
 * brand it would import, the sources it found, an overall confidence, and
 * suggestions for missing data. It is read-only. The mutating ImportBrand RPC
 * (which creates a brand) is a CLI/wizard action, so the UI surfaces only the
 * safe scan.
 */
export async function discoverScenario(input: { scenarioName: string }): Promise<DiscoveryResult> {
  return discoveryClient.discoverScenario({ scenarioName: input.scenarioName });
}

export type { DiscoveryResult, DiscoverySource, DraftBrand };
