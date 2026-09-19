/**
 * Matrix test-data factories. Build real proto messages via `create()` so
 * tests exercise the same field shapes the connect client decodes at runtime.
 * Co-located with the feature so deleting the folder takes the doubles along.
 */
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import {
  GetMatrixResponseSchema,
  MatrixRowSchema,
  RegistrySummarySchema,
  EvidenceCellSchema,
  ManualAttestationSchema,
  LogManualValidationResponseSchema,
} from "@vrooli/proto-types/business-health/v1/contract/contract_pb";
import type {
  GetMatrixResponse,
  MatrixRow,
  RegistrySummary,
  EvidenceCell,
  ManualAttestation,
  LogManualValidationResponse,
} from "@vrooli/proto-types/business-health/v1/contract/contract_pb";

export const makeManualAttestation = (
  overrides: MessageInitShape<typeof ManualAttestationSchema> = {},
): ManualAttestation =>
  create(ManualAttestationSchema, {
    requirementId: "BH-UX-001",
    attestedBy: "agent:test",
    expired: false,
    notes: "verified by hand",
    ...overrides,
  });

export const makeEvidenceCell = (
  overrides: MessageInitShape<typeof EvidenceCellSchema> = {},
): EvidenceCell =>
  create(EvidenceCellSchema, {
    liveStatus: "passed",
    stale: false,
    ...overrides,
  });

export const makeRegistrySummary = (
  overrides: MessageInitShape<typeof RegistrySummarySchema> = {},
): RegistrySummary =>
  create(RegistrySummarySchema, {
    moduleCount: 7,
    requirementCount: 24,
    operationalTargetCount: 12,
    statusCounts: { planned: 10, complete: 14 },
    starterTemplate: false,
    ...overrides,
  });

export const makeMatrixRow = (
  overrides: MessageInitShape<typeof MatrixRowSchema> = {},
): MatrixRow =>
  create(MatrixRowSchema, {
    otId: "OT-P0-001",
    otTitle: "Validate the business contract",
    otChecked: true,
    otPriority: "P0",
    requirementId: "BH-UX-001",
    requirementTitle: "Traceability matrix view",
    requirementStatus: "complete",
    criticality: "P1",
    validations: [
      { type: "test", phase: "unit", status: "implemented", ref: "ui/x.test.tsx", refExists: true },
    ],
    evidence: makeEvidenceCell(),
    unproven: false,
    unprovenReason: "",
    ...overrides,
  });

export const makeGetMatrixResponse = (
  overrides: MessageInitShape<typeof GetMatrixResponseSchema> = {},
): GetMatrixResponse =>
  create(GetMatrixResponseSchema, {
    scenario: "business-health",
    matrix: [makeMatrixRow()],
    registry: makeRegistrySummary(),
    degradedReason: "",
    ...overrides,
  });

export const makeLogManualValidationResponse = (
  overrides: MessageInitShape<typeof LogManualValidationResponseSchema> = {},
): LogManualValidationResponse =>
  create(LogManualValidationResponseSchema, {
    attestation: makeManualAttestation(),
    ledgerPath: "scenarios/business-health/.vrooli/manual-validations.json",
    ...overrides,
  });
