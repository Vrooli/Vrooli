/**
 * Discovery domain — UI ↔ API boundary over DiscoveryService. Discovery scans
 * the local environment read-only and returns onboarding *suggestions*: targets
 * worth protecting (well-known ~/.vrooli runtime state) and destinations worth
 * backing up to (mounted volumes / plugged-in drives).
 *
 * Suggestions are derived server-side and never stored — "accepting" one is NOT
 * a discovery call: the UI calls the existing `registerTarget` /
 * `createDestination` with the suggestion's values, reusing their validation
 * (separate-root rule, encryption-on). Only dismissals persist, via
 * `dismissSuggestion`.
 */
import { createClient, type Client } from "@connectrpc/connect";
import { DiscoveryService } from "@vrooli/proto-types/data-backup-manager/v1/discovery/discovery_pb";
import { DriveClass } from "@vrooli/proto-types/data-backup-manager/v1/discovery/discovery_pb";
import type {
  TargetSuggestion,
  DestinationSuggestion,
} from "@vrooli/proto-types/data-backup-manager/v1/discovery/discovery_pb";

import { transport } from "./client";

export const discoveryClient: Client<typeof DiscoveryService> = createClient(
  DiscoveryService,
  transport,
);

export async function listTargetSuggestions(): Promise<TargetSuggestion[]> {
  const res = await discoveryClient.listTargetSuggestions({});
  return res.suggestions;
}

export async function listDestinationSuggestions(): Promise<DestinationSuggestion[]> {
  const res = await discoveryClient.listDestinationSuggestions({});
  return res.suggestions;
}

/** Hides a suggestion permanently by its stable id; returns whether it stuck. */
export async function dismissSuggestion(id: string): Promise<boolean> {
  const res = await discoveryClient.dismissSuggestion({ id });
  return res.dismissed;
}

export { DriveClass };
export type { TargetSuggestion, DestinationSuggestion };
