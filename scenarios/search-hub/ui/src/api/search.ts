/**
 * UI ↔ search-hub API boundary for the federated search surface.
 *
 * Three Connect services back the search page:
 *   - RoutingService.Query  — fan out a query and return ranked/grouped hits.
 *   - RoutingService.Status — per-provider reachability + classifier/reranker
 *     availability (provider freshness on results).
 *   - RegistryService.ListProviders — the ACTIVE leaves, used to build the
 *     type facets (we route on what is actually registered, never a hardcoded
 *     list).
 *
 * Each call is exported as a thin async function (not just the raw client) so
 * tests mock this module the same way they mock ./api/health.
 */
import { createClient } from "@connectrpc/connect";
import { RoutingService } from "@vrooli/proto-types/search-hub/v1/routing/routing_pb";
import type {
  QueryResponse,
  StatusResponse,
} from "@vrooli/proto-types/search-hub/v1/routing/routing_pb";
import { RegistryService } from "@vrooli/proto-types/search-hub/v1/registry/registry_pb";
import {
  ProviderState,
  type ProviderDescriptor,
} from "@vrooli/proto-types/search-hub/v1/registry/registry_pb";

import { transport } from "./client";

const routingClient = createClient(RoutingService, transport);
const registryClient = createClient(RegistryService, transport);

/** Parameters captured from the search form and sent to RoutingService.Query. */
export interface SearchInput {
  /** Natural-language query text. */
  query: string;
  /** Explicit leaf types to route to; empty + all=false ⇒ the classifier routes. */
  types: string[];
  /** Fan out to every active provider (the "expand search" affordance). */
  all: boolean;
  /** Per-provider result cap. */
  limit?: number;
}

/**
 * Run a federated query. Always requests the routing explanation so the UI can
 * show why these providers were chosen.
 */
export async function runQuery(input: SearchInput): Promise<QueryResponse> {
  return routingClient.query({
    query: input.query,
    types: input.types,
    all: input.all,
    limit: input.limit ?? 0,
    explain: true,
  });
}

/** Fetch federation health: per-provider reachability + model availability. */
export async function fetchFederationStatus(): Promise<StatusResponse> {
  return routingClient.status({});
}

/**
 * List the ACTIVE registered providers (the federation's live leaves) so the UI
 * can build type facets from the registry rather than a hardcoded list.
 */
export async function listActiveProviders(): Promise<ProviderDescriptor[]> {
  const res = await registryClient.listProviders({ state: ProviderState.ACTIVE });
  return res.providers;
}

export type { QueryResponse, StatusResponse, ProviderDescriptor };
