import { createClient } from "@connectrpc/connect";
import {
  FleetService,
  type ScanFleetResponse,
  type FleetScenarioEntry,
  type RuleConformance,
  type ProfileDistribution,
  type FleetScanError,
} from "@vrooli/proto-types/structure-health/v1/fleet/fleet_pb";

import { transport } from "./client";

/**
 * Connect-Web client for the structure-health fleet dashboard. `ScanFleet`
 * statically grades every discovered scenario (or the requested subset) and
 * returns one structure rollup per scenario plus fleet-wide profile
 * distribution and per-rule conformance.
 *
 * The dashboard reads strictly off the typed proto contract — verdicts mirror
 * the gating semantics (`passed=false` iff a scenario has any error-severity
 * structure finding) so the visual semantics match the gating semantics.
 */
const client = createClient(FleetService, transport);

export const fleetClient = {
  scanFleet: (input: { scenarios?: string[] } = {}): Promise<ScanFleetResponse> =>
    client.scanFleet({ scenarios: input.scenarios ?? [] }),
};

export type {
  ScanFleetResponse,
  FleetScenarioEntry,
  RuleConformance,
  ProfileDistribution,
  FleetScanError,
};
