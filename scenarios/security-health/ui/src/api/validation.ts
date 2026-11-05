import { createClient } from "@connectrpc/connect";
import {
  ScenarioValidationService,
  ValidationStatus,
  type ValidateScenarioResponse,
} from "@vrooli/proto-types/scenario-validation/v1/validation_pb";

import { transport } from "./client";

export const Severity = {
  UNSPECIFIED: "SEVERITY_UNSPECIFIED",
  ERROR: "SEVERITY_ERROR",
  WARNING: "SEVERITY_WARNING",
  INFO: "SEVERITY_INFO",
} as const;

export type Severity = (typeof Severity)[keyof typeof Severity];

export interface Finding {
  ruleId: string;
  severity: Severity;
  title: string;
  description: string;
  remediation: string;
  filePath: string;
  scanner: string;
}

export interface Summary {
  errors: number;
  warnings: number;
  infos: number;
}

/**
 * Connect-Web client for the shared ScenarioValidationService. Posture
 * surfaces render security findings from `assessment.findings`; severity is
 * the load-bearing contract: ERROR gates, WARNING and INFO are advisory.
 */
export const validationClient = createClient(ScenarioValidationService, transport);

export { ValidationStatus };
export type { ValidateScenarioResponse };

export function findingsFromResponse(resp: ValidateScenarioResponse | undefined): Finding[] {
  return (resp?.assessment?.findings ?? []).map((finding) => {
    const code = finding.code;
    return {
      ruleId: code,
      severity: normalizeSeverity(finding.severity),
      title: finding.title || code,
      description: finding.message,
      remediation: finding.remediation,
      filePath: finding.location,
      scanner: scannerFromCode(code),
    };
  });
}

export function summaryFromResponse(resp: ValidateScenarioResponse | undefined): Summary {
  const bySeverity = resp?.assessment?.findingsBySeverity ?? {};
  return {
    errors: severityCount(bySeverity, "ERROR"),
    warnings: severityCount(bySeverity, "WARNING"),
    infos: severityCount(bySeverity, "INFO"),
  };
}

export function passedFromResponse(resp: ValidateScenarioResponse | undefined): boolean {
  return resp?.status === ValidationStatus.PASSED || resp?.status === ValidationStatus.DEGRADED;
}

function normalizeSeverity(raw: string): Severity {
  switch (raw) {
    case Severity.ERROR:
    case "FINDING_SEVERITY_ERROR":
      return Severity.ERROR;
    case Severity.WARNING:
    case "FINDING_SEVERITY_WARNING":
      return Severity.WARNING;
    case Severity.INFO:
    case "FINDING_SEVERITY_INFO":
      return Severity.INFO;
    default:
      return Severity.UNSPECIFIED;
  }
}

function scannerFromCode(code: string): string {
  const trimmed = code.trim();
  if (!trimmed.includes(".")) {
    return "security-health";
  }
  return trimmed.split(".", 1)[0] || "security-health";
}

function severityCount(counts: Record<string, number>, severity: "ERROR" | "WARNING" | "INFO"): number {
  return (counts[`SEVERITY_${severity}`] ?? 0) + (counts[`FINDING_SEVERITY_${severity}`] ?? 0);
}
