// API client for the validation domain. Thin wrapper over the generated
// ScenarioValidationService Connect client; exports plain TS types so callers
// stay decoupled from generated message shapes.
import { createClient } from "@connectrpc/connect";

import {
  type AssessmentFinding as ProtoFinding,
  FixAffordance,
  type MaturityAssessment,
  type PhasePresentation,
} from "@vrooli/proto-types/common/v1/maturity_pb";
import {
  ScenarioValidationService,
  ValidationStatus,
  type ValidateScenarioResponse as ProtoResponse,
} from "@vrooli/proto-types/scenario-validation/v1/validation_pb";

import { transport } from "./client";

export const validationClient = createClient(ScenarioValidationService, transport);

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

export type ValidationPresentationFinding = {
  code: string;
  severity: FindingSeverity;
  count: number;
  remediation: string;
  fixAffordance: string;
};

export type ValidationCapabilityPresentation = {
  id: string;
  label: string;
  currentLevel: string;
  nextLevel: string;
  currentSummary: string;
  nextUnlock: string;
  findings: ValidationPresentationFinding[];
};

export type ValidationPresentation = {
  contractVersion: string;
  currentLevel: string;
  nextLevel: string;
  northStar: string;
  nextAction: string;
  nextActionReason: string;
  focusCapabilityLabel: string;
  atMaximum: boolean;
  documentationTopics: string[];
  capabilities: ValidationCapabilityPresentation[];
};

export type ValidationResult = {
  scenario: string;
  passed: boolean;
  findings: ValidationFinding[];
  summary: ValidationSummary;
  presentation?: ValidationPresentation;
  ranAt: string;
};

export async function validateScenario(scenario: string): Promise<ValidationResult> {
  const resp = await validationClient.validateScenario({ scenario });
  return resultFromProto(resp);
}

function severityFromProto(s: string): FindingSeverity {
  switch (s) {
    case "SEVERITY_ERROR":
      return "error";
    case "SEVERITY_WARNING":
      return "warning";
    case "SEVERITY_INFO":
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
    suggestion: p.remediation,
  };
}

function summaryFromAssessment(p: MaturityAssessment | undefined): ValidationSummary {
  return {
    errors: p?.findingsBySeverity["SEVERITY_ERROR"] ?? 0,
    warnings: p?.findingsBySeverity["SEVERITY_WARNING"] ?? 0,
    infos: p?.findingsBySeverity["SEVERITY_INFO"] ?? 0,
  };
}

function presentationFromProto(p: PhasePresentation | undefined): ValidationPresentation | undefined {
  if (!p || !p.contractVersion) return undefined;
  return {
    contractVersion: p.contractVersion,
    currentLevel: p.currentLevel,
    nextLevel: p.nextLevel,
    northStar: p.northStar,
    nextAction: p.nextAction,
    nextActionReason: p.nextActionReason,
    focusCapabilityLabel: p.focusCapabilityLabel,
    atMaximum: p.atMaximum,
    documentationTopics: [...p.documentationTopics],
    capabilities: p.capabilities.map((capability) => ({
      id: capability.id,
      label: capability.label,
      currentLevel: capability.currentLevel,
      nextLevel: capability.nextLevel,
      currentSummary: capability.currentSummary,
      nextUnlock: capability.nextUnlock,
      findings: capability.findings.map((finding) => ({
        code: finding.code,
        severity: severityFromProto(finding.severity),
        count: finding.count,
        remediation: finding.remediation,
        fixAffordance: FixAffordance[finding.fixAffordance],
      })),
    })),
  };
}

function resultFromProto(p: ProtoResponse): ValidationResult {
  const assessment = p.assessment;
  return {
    scenario: p.scenario,
    passed: p.status === ValidationStatus.PASSED,
    findings: (assessment?.findings ?? []).map(findingFromProto),
    summary: summaryFromAssessment(assessment),
    presentation: presentationFromProto(assessment?.presentation),
    ranAt: new Date().toISOString(),
  };
}
