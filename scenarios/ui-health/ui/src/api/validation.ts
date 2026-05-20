// API client for the validation domain. Thin wrapper over the generated
// ValidationService Connect client; exports plain TS types so callers stay
// decoupled from generated message shapes.
import { createClient } from "@connectrpc/connect";

import {
  ValidationService,
  Severity,
  type Finding as ProtoFinding,
  type Summary as ProtoSummary,
  type ValidateScenarioResponse as ProtoResponse,
} from "@vrooli/proto-types/ui-health/v1/validation/validation_pb";

import { transport } from "./client";

export const validationClient = createClient(ValidationService, transport);

export type FindingSeverity = "error" | "warning" | "info" | "unspecified";

export type ValidationFinding = {
  severity: FindingSeverity;
  code: string;
  location: string;
  message: string;
  suggestion: string;
};

export type ValidationSummary = {
  errors: number;
  warnings: number;
  infos: number;
};

export type ValidationResult = {
  scenario: string;
  passed: boolean;
  findings: ValidationFinding[];
  summary: ValidationSummary;
  ranAt: string;
};

export async function validateScenario(scenario: string): Promise<ValidationResult> {
  const resp = await validationClient.validateScenario({ scenario });
  return resultFromProto(resp);
}

function severityFromProto(s: Severity): FindingSeverity {
  switch (s) {
    case Severity.ERROR:
      return "error";
    case Severity.WARNING:
      return "warning";
    case Severity.INFO:
      return "info";
    default:
      return "unspecified";
  }
}

function findingFromProto(p: ProtoFinding): ValidationFinding {
  return {
    severity: severityFromProto(p.severity),
    code: p.code,
    location: p.location,
    message: p.message,
    suggestion: p.suggestion,
  };
}

function summaryFromProto(p: ProtoSummary | undefined): ValidationSummary {
  return { errors: p?.errors ?? 0, warnings: p?.warnings ?? 0, infos: p?.infos ?? 0 };
}

function resultFromProto(p: ProtoResponse): ValidationResult {
  return {
    scenario: p.scenario,
    passed: p.passed,
    findings: p.findings.map(findingFromProto),
    summary: summaryFromProto(p.summary),
    ranAt: new Date().toISOString(),
  };
}
