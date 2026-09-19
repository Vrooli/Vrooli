import { createClient } from "@connectrpc/connect";
import {
  DependencyService,
  Ecosystem,
  Mode,
  type DependencyRecord,
  type SearchResult,
  type SearchResponse,
  type StatusResponse,
} from "@vrooli/proto-types/security-health/v1/dependencies/dependencies_pb";

import { transport } from "./client";

/**
 * Connect-Web client for the DependencyService. The Dependencies surface
 * issues `search(...)` queries (free text plus the `ecosystem` /
 * `vulnerableOnly` / `nameGlob` structured filters) and reads `status()` for
 * the indexed/vulnerable counts and last-reconcile time. `mode_used` lets the
 * UI render a "(text mode)" hint when the semantic backend is degraded.
 */
export const dependencyClient = createClient(DependencyService, transport);

export { Ecosystem, Mode };
export type { DependencyRecord, SearchResult, SearchResponse, StatusResponse };
