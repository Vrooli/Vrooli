import { createClient } from "@connectrpc/connect";
import {
  ValidationService,
  DomainStatus,
  Tier,
  Severity,
  type FleetEntry,
  type DomainCoverage,
  type MeasureSummary,
  type ValidateScenarioResponse,
  type ListFleetCoverageResponse,
} from "@vrooli/proto-types/measures-health/v1/validation/validation_pb";

import { transport } from "./client";

/**
 * Connect-Web client for the measures `ValidationService`. The fleet
 * dashboard calls `listFleetCoverage({})` for the cross-scenario rollup and
 * `validateScenario({ scenario })` for the per-scenario drill-down.
 *
 * The enums are the load-bearing contract — the UI colors and orders strictly
 * off `DomainStatus` / `Tier`, never off a free-text string — so the visual
 * semantics match the gating semantics (an UNCOVERED stateful domain is the
 * only thing that fails a scenario, mirroring `passed=false iff any ERROR`).
 */
export const fleetClient = createClient(ValidationService, transport);

export { DomainStatus, Tier, Severity };
export type {
  FleetEntry,
  DomainCoverage,
  MeasureSummary,
  ValidateScenarioResponse,
  ListFleetCoverageResponse,
};
