import { createClient } from "@connectrpc/connect";
import {
  RoutesService,
  RouteSource,
  Tier,
  type Route,
  type ListRoutesResponse,
  type CreateRouteResponse,
  type DeleteRouteResponse,
} from "@vrooli/proto-types/tunnel-manager/v1/routes/routes_pb";

import { transport } from "./client";

// routesClient is the generated Connect-Web client for the RoutesService.
// The exposure manifest (SSOT) read/mutation surface the UI features under
// ui/src/features/ call. Proves the proto contract reaches TypeScript
// end-to-end.
export const routesClient = createClient(RoutesService, transport);

/** listRoutes returns all manifest routes, optionally filtered by tier. */
export async function listRoutes(tier: Tier = Tier.UNSPECIFIED): Promise<Route[]> {
  const resp = await routesClient.listRoutes({ tier });
  return resp.routes;
}

/**
 * createExternalRoute adds a route that points at an arbitrary local service
 * target rather than a known scenario's UI port. External routes carry
 * RouteSource.EXTERNAL and skip the scenario/local-port validation; they
 * require a serviceTarget (e.g. http://127.0.0.1:9000).
 */
export async function createExternalRoute(values: {
  subdomain: string;
  serviceTarget: string;
  domain?: string;
}): Promise<CreateRouteResponse> {
  return routesClient.createRoute({
    subdomain: values.subdomain,
    serviceTarget: values.serviceTarget,
    domain: values.domain ?? "",
    source: RouteSource.EXTERNAL,
  });
}

/** deleteRoute removes a manifest route by its id. */
export async function deleteRoute(id: string): Promise<DeleteRouteResponse> {
  return routesClient.deleteRoute({ id });
}

export { Tier, RouteSource };
export type { Route, ListRoutesResponse, CreateRouteResponse, DeleteRouteResponse };
