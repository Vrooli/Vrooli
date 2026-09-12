import { createClient } from "@connectrpc/connect";
import { fromBinary, toJson } from "@bufbuild/protobuf";
import { StructSchema } from "@bufbuild/protobuf/wkt";
import {
  ScenarioValidationService,
  ValidationStatus,
  type FixResponse,
  type ValidateScenarioResponse,
} from "@vrooli/proto-types/scenario-validation/v1/validation_pb";

import { transport } from "./client";

const validationClient = createClient(ScenarioValidationService, transport);

export interface ValidationNativeDetail {
  scenario?: string;
  target?: {
    scenario?: string;
    root_path?: string;
    resolution?: string;
    api_kind?: string;
    api_dir?: string;
    has_api_dir?: boolean;
    health_probe?: {
      requested?: boolean;
      url?: string;
      status_code?: number;
      content_type?: string;
      elapsed_millis?: number;
      failure_class?: string;
      error?: string;
      schema_valid?: boolean;
      schema_violations?: string[];
      payload?: {
        status?: string;
        service?: string;
        timestamp?: string;
        readiness?: boolean;
        version?: string;
        dependency_count?: number;
      };
    };
    http_semantics?: {
      routes?: unknown[];
      response_patterns?: unknown[];
    };
    runtime_hygiene?: {
      signals?: unknown[];
    };
  };
  summary?: {
    errors?: number;
    warnings?: number;
    infos?: number;
    passed?: boolean;
  };
}

export interface ValidationReport {
  response: ValidateScenarioResponse;
  nativeDetail: ValidationNativeDetail;
}

export async function validateScenario(input: {
  scenario: string;
  path?: string;
  includeExecution?: boolean;
}): Promise<ValidationReport> {
  const response = await validationClient.validateScenario({
    scenario: input.scenario,
    path: input.path ?? "",
    includeExecution: input.includeExecution ?? false,
  });
  return {
    response,
    nativeDetail: decodeNativeDetail(response),
  };
}

export async function previewFix(input: {
  scenario: string;
  path?: string;
  ruleIds?: string[];
}): Promise<FixResponse> {
  return validationClient.previewFix({
    scenario: input.scenario,
    path: input.path ?? "",
    ruleIds: input.ruleIds ?? [],
  });
}

export function statusLabel(status: ValidationStatus): string {
  switch (status) {
    case ValidationStatus.PASSED:
      return "passed";
    case ValidationStatus.FAILED:
      return "failed";
    case ValidationStatus.DEGRADED:
      return "degraded";
    case ValidationStatus.ERROR:
      return "error";
    case ValidationStatus.SKIPPED:
      return "skipped";
    default:
      return "unspecified";
  }
}

function decodeNativeDetail(response: ValidateScenarioResponse): ValidationNativeDetail {
  if (!response.nativeDetail?.value.length) {
    return {};
  }
  try {
    const struct = fromBinary(StructSchema, response.nativeDetail.value);
    return toJson(StructSchema, struct) as ValidationNativeDetail;
  } catch {
    return {};
  }
}

export type { FixResponse, ValidateScenarioResponse };
