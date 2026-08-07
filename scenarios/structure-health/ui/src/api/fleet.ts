import { createClient } from "@connectrpc/connect";
import {
  FleetService,
  type ScanFleetResponse,
  type FleetScenarioEntry,
  type RuleConformance,
  type ProfileDistribution,
  type FleetScanError,
  FleetTargetSchema,
} from "@vrooli/proto-types/structure-health/v1/fleet/fleet_pb";
import { create } from "@bufbuild/protobuf";

import { transport } from "./client";

/**
 * Connect-Web client for the structure-health fleet dashboard. `ScanFleet`
 * statically grades every discovered target (or the requested subset) and
 * returns one structure rollup per target plus fleet-wide profile
 * distribution and per-rule conformance.
 *
 * The dashboard reads strictly off the typed proto contract — verdicts mirror
 * the gating semantics (`passed=false` iff a scenario has any error-severity
 * structure finding) so the visual semantics match the gating semantics.
 */
const client = createClient(FleetService, transport);

export const fleetClient = {
  scanFleet: (input: {
    scenarios?: string[];
    targets?: { kind: string; id: string; path?: string }[];
  } = {}): Promise<ScanFleetResponse> =>
    client.scanFleet({
      scenarios: input.scenarios ?? [],
      targets: (input.targets ?? []).map((target) =>
        create(FleetTargetSchema, {
          kind: target.kind,
          id: target.id,
          path: target.path ?? "",
        }),
      ),
    }),
};

export type {
  ScanFleetResponse,
  FleetScenarioEntry,
  RuleConformance,
  ProfileDistribution,
  FleetScanError,
};
