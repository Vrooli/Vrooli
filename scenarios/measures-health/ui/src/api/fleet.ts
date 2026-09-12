import { createClient } from "@connectrpc/connect";
import { fromBinary } from "@bufbuild/protobuf";
import {
  ValidationService,
  DomainStatus,
  Tier,
  Severity,
  type FleetEntry,
  type DomainCoverage,
  type MeasureSummary,
  type ScenarioCoverageReport,
  type ListFleetCoverageResponse,
  ScenarioCoverageReportSchema,
} from "@vrooli/proto-types/measures-health/v1/validation/validation_pb";
import { ScenarioValidationService } from "@vrooli/proto-types/scenario-validation/v1/validation_pb";

import { transport } from "./client";

/**
 * Connect-Web clients for measures validation. The fleet dashboard calls the
 * native `ValidationService.ListFleetCoverage` for the cross-scenario rollup
 * and shared `ScenarioValidationService.ValidateScenario` for the per-scenario
 * drill-down, unpacking measures-health's native ScenarioCoverageReport from
 * native_detail.
 *
 * The enums are the load-bearing contract — the UI colors and orders strictly
 * off `DomainStatus` / `Tier`, never off a free-text string — so the visual
 * semantics match the gating semantics (an UNCOVERED stateful domain is the
 * only thing that fails a scenario, mirroring `passed=false iff any ERROR`).
 */
const nativeFleetClient = createClient(ValidationService, transport);
const sharedValidationClient = createClient(ScenarioValidationService, transport);

export const fleetClient = {
  listFleetCoverage: nativeFleetClient.listFleetCoverage.bind(nativeFleetClient),
  async validateScenario(input: { scenario: string; probe?: boolean }): Promise<ScenarioCoverageReport> {
    const response = await sharedValidationClient.validateScenario({
      scenario: input.scenario,
      includeExecution: input.probe ?? false,
    });
    if (!response.nativeDetail?.value) {
      throw new Error("measures-health validation response did not include native coverage detail");
    }
    return fromBinary(ScenarioCoverageReportSchema, response.nativeDetail.value);
  },
};

export { DomainStatus, Tier, Severity };
export type {
  FleetEntry,
  DomainCoverage,
  MeasureSummary,
  ScenarioCoverageReport,
  ListFleetCoverageResponse,
};
