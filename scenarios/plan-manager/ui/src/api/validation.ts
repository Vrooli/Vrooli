import { createClient } from "@connectrpc/connect";
import {
  ValidationService,
  type DeriveBaselineScopeResponse,
} from "@vrooli/proto-types/plan-manager/v1/validation/validation_pb";
import {
  type Reference,
  type ValidationResult,
  type StalenessTier,
} from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";

import { transport } from "./client";

/**
 * Connect-Web client for the ValidationService — plan health. The operator
 * console (Phase 7) surfaces reference resolution, staleness tiers, derived
 * baseline scopes, and DoD verdicts. Each helper returns the proto-typed shape.
 */
export const validationClient = createClient(ValidationService, transport);

export interface StalenessReport {
  overall: StalenessTier;
  references: Reference[];
  degraded: boolean;
}

export async function resolveReferences(
  planId: string,
  phaseId = "",
): Promise<{ references: Reference[]; degraded: boolean }> {
  const resp = await validationClient.resolveReferences({ planId, phaseId });
  return { references: resp.references, degraded: resp.degraded };
}

export async function computeStaleness(planId: string, phaseId = ""): Promise<StalenessReport> {
  const resp = await validationClient.computeStaleness({ planId, phaseId });
  return { overall: resp.overall, references: resp.references, degraded: resp.degraded };
}

export async function deriveBaselineScope(
  planId: string,
  phaseId = "",
): Promise<DeriveBaselineScopeResponse> {
  return validationClient.deriveBaselineScope({ planId, phaseId });
}

export async function runValidation(
  planId: string,
  phaseId = "",
): Promise<ValidationResult | undefined> {
  const resp = await validationClient.runValidation({ planId, phaseId });
  return resp.result;
}

export async function verifyDefinitionOfDone(
  planId: string,
): Promise<{ result: ValidationResult | undefined; dodMet: boolean }> {
  const resp = await validationClient.verifyDefinitionOfDone({ planId });
  return { result: resp.result, dodMet: resp.dodMet };
}
