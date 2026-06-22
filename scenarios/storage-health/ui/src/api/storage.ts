import { createClient } from "@connectrpc/connect";

import {
  FleetService,
  type ScanFleetResponse,
  type FleetScenarioEntry,
  type EngineCount,
  type StageCount,
  type FleetScanError,
} from "@vrooli/proto-types/storage-health/v1/fleet/fleet_pb";
import {
  AdvisorService,
  type AnalyzeMigrationsResponse,
  type MigrationHygiene,
  type AdviseEnginesResponse,
  type EngineCandidate,
  type AdvisorScanError,
} from "@vrooli/proto-types/storage-health/v1/advisor/advisor_pb";
import {
  ScenarioValidationService,
  ValidationStatus,
  type ValidateScenarioResponse,
  type FixResponse,
  type FixCandidate,
} from "@vrooli/proto-types/scenario-validation/v1/validation_pb";

import { transport } from "./client";

/**
 * Typed Connect-Web clients for every storage-health product domain the UI
 * speaks to. Each method is a thin wrapper that forwards the request fields the
 * UI actually sets (and the defaults the UI fills in), so screens read strictly
 * off the generated proto contracts — no hand-rolled fetch, no untyped JSON.
 *
 *   - FleetService            — cross-scenario storage inventory + scorecard
 *   - AdvisorService          — engine fitness + migration hygiene
 *   - ScenarioValidationService — per-scenario findings + autofix flow (shared
 *                                 scenario-validation contract)
 */
const fleet = createClient(FleetService, transport);
const advisor = createClient(AdvisorService, transport);
const validation = createClient(ScenarioValidationService, transport);

export const storageClient = {
  // Fleet ---------------------------------------------------------------
  scanFleet: (input: { scenarios?: string[] } = {}): Promise<ScanFleetResponse> =>
    fleet.scanFleet({ scenarios: input.scenarios ?? [] }),
  getInventory: (): Promise<ScanFleetResponse> => fleet.getInventory({}),

  // Advisor -------------------------------------------------------------
  adviseEngines: (input: { scenarios?: string[] } = {}): Promise<AdviseEnginesResponse> =>
    advisor.adviseEngines({ scenarios: input.scenarios ?? [] }),
  analyzeMigrations: (input: { scenarios?: string[] } = {}): Promise<AnalyzeMigrationsResponse> =>
    advisor.analyzeMigrations({ scenarios: input.scenarios ?? [] }),

  // Validation ----------------------------------------------------------
  validateScenario: (input: { scenario: string }): Promise<ValidateScenarioResponse> =>
    validation.validateScenario({ scenario: input.scenario }),
  previewFix: (input: { scenario: string; ruleIds?: string[] }): Promise<FixResponse> =>
    validation.previewFix({ scenario: input.scenario, ruleIds: input.ruleIds ?? [] }),
  applyFix: (input: { scenario: string; ruleIds?: string[] }): Promise<FixResponse> =>
    validation.applyFix({ scenario: input.scenario, ruleIds: input.ruleIds ?? [] }),
};

export { ValidationStatus };
export type {
  ScanFleetResponse,
  FleetScenarioEntry,
  EngineCount,
  StageCount,
  FleetScanError,
  AnalyzeMigrationsResponse,
  MigrationHygiene,
  AdviseEnginesResponse,
  EngineCandidate,
  AdvisorScanError,
  ValidateScenarioResponse,
  FixResponse,
  FixCandidate,
};
