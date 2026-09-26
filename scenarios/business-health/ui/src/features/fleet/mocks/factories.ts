/**
 * Fleet test-data factories. Build real proto messages via `create()` so tests
 * exercise the same field shapes the connect client decodes at runtime.
 * Co-located with the feature so deleting the folder takes the doubles along.
 */
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import {
  ScanFleetResponseSchema,
  FleetScenarioEntrySchema,
  FleetScanErrorSchema,
} from "@vrooli/proto-types/business-health/v1/fleet/fleet_pb";
import type {
  ScanFleetResponse,
  FleetScenarioEntry,
  FleetScanError,
} from "@vrooli/proto-types/business-health/v1/fleet/fleet_pb";

export const makeFleetEntry = (
  overrides: MessageInitShape<typeof FleetScenarioEntrySchema> = {},
): FleetScenarioEntry =>
  create(FleetScenarioEntrySchema, {
    scenario: "business-health",
    passed: true,
    errorCount: 0,
    warningCount: 1,
    totalFindings: 1,
    autofixableCount: 0,
    starterRegistry: false,
    templateVersion: "2026.06.01",
    templateLaggard: false,
    orphanedTargets: 0,
    unprovenClaims: 0,
    debtScore: 5,
    degradedReason: "",
    ...overrides,
  });

export const makeFleetScanError = (
  overrides: MessageInitShape<typeof FleetScanErrorSchema> = {},
): FleetScanError =>
  create(FleetScanErrorSchema, {
    scenario: "broken-scenario",
    reason: "intent.json failed to parse",
    ...overrides,
  });

export const makeScanFleetResponse = (
  overrides: MessageInitShape<typeof ScanFleetResponseSchema> = {},
): ScanFleetResponse =>
  create(ScanFleetResponseSchema, {
    entries: [makeFleetEntry()],
    asOf: timestampFromDate(new Date("2026-07-02T12:00:00Z")),
    scenarioCount: 1,
    passingCount: 1,
    starterRegistryCount: 0,
    templateLaggardCount: 0,
    errors: [],
    ...overrides,
  });
