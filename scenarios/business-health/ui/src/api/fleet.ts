import { createClient, type Client } from "@connectrpc/connect";
import { FleetService } from "@vrooli/proto-types/business-health/v1/fleet/fleet_pb";

import { transport } from "./client";

/**
 * Connect-RPC client for the business-health FleetService (worst-first fleet
 * scan). See `api/contract.ts` for the client-usage convention.
 */
export const fleetClient: Client<typeof FleetService> = createClient(
  FleetService,
  transport,
);
