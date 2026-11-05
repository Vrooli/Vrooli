/**
 * Findings test-data factories. Build real proto messages via `create()` so
 * tests exercise the same field shapes the connect client decodes at runtime.
 * Co-located with the feature so deleting the folder takes the doubles along.
 */
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import {
  BusinessContractReportSchema,
  CapabilityRollupSchema,
  ContractFindingSchema,
  ValidateScenarioResponseSchema,
} from "@vrooli/proto-types/business-health/v1/contract/contract_pb";
import type {
  BusinessContractReport,
  CapabilityRollup,
  ContractFinding,
  ValidateScenarioResponse,
} from "@vrooli/proto-types/business-health/v1/contract/contract_pb";
import {
  FixCandidateSchema,
  FixResponseSchema,
} from "@vrooli/proto-types/scenario-validation/v1/validation_pb";
import type {
  FixCandidate,
  FixResponse,
} from "@vrooli/proto-types/scenario-validation/v1/validation_pb";

export const makeContractFinding = (
  overrides: MessageInitShape<typeof ContractFindingSchema> = {},
): ContractFinding =>
  create(ContractFindingSchema, {
    code: "intent.ot_orphan",
    severity: "error",
    title: "Operational target has no linked requirement",
    message: "OT-P0-001 declares no requirement covering it.",
    location: "PRD.md:42",
    remediation: "Add a requirement whose prd_ref points at OT-P0-001.",
    autofixAvailable: false,
    fixClass: "manual",
    ...overrides,
  });

export const makeCapabilityRollup = (
  overrides: MessageInitShape<typeof CapabilityRollupSchema> = {},
): CapabilityRollup =>
  create(CapabilityRollupSchema, {
    capabilityId: "intent_linkage",
    levelId: "L1",
    levelName: "Linked",
    findingCount: 1,
    errorCount: 1,
    warningCount: 0,
    ...overrides,
  });

export const makeBusinessContractReport = (
  overrides: MessageInitShape<typeof BusinessContractReportSchema> = {},
): BusinessContractReport =>
  create(BusinessContractReportSchema, {
    capabilities: [makeCapabilityRollup()],
    matrix: [],
    drift: [],
    findings: [makeContractFinding()],
    ...overrides,
  });

export const makeValidateScenarioResponse = (
  overrides: MessageInitShape<typeof ValidateScenarioResponseSchema> = {},
): ValidateScenarioResponse =>
  create(ValidateScenarioResponseSchema, {
    scenario: "business-health",
    status: "FAILED",
    summary: "1 finding",
    targetPath: "scenarios/business-health",
    degradedReason: "",
    report: makeBusinessContractReport(),
    nextSteps: [],
    ...overrides,
  });

export const makeFixCandidate = (
  overrides: MessageInitShape<typeof FixCandidateSchema> = {},
): FixCandidate =>
  create(FixCandidateSchema, {
    ruleId: "intent.ot_orphan",
    filePath: "scenarios/business-health/requirements/BH-INTENT-001.md",
    description: "Create a requirement linking OT-P0-001.",
    before: "",
    after: "---\nprd_ref: OT-P0-001\n---\n",
    applied: false,
    ...overrides,
  });

export const makeFixResponse = (
  overrides: MessageInitShape<typeof FixResponseSchema> = {},
): FixResponse =>
  create(FixResponseSchema, {
    scenario: "business-health",
    applied: false,
    candidates: [makeFixCandidate()],
    messages: [],
    ...overrides,
  });
