import { createClient } from "@connectrpc/connect";
import {
  RoutesService,
  Tier,
  type Route,
  type ListRoutesResponse,
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

export { Tier };
export type { Route, ListRoutesResponse };
