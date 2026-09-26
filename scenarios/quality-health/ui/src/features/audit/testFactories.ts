import type { MessageInitShape } from "@bufbuild/protobuf";
import {
  AuditQualityResponseSchema,
  ExplainFindingResponseSchema,
  FixConfigResponseSchema,
} from "@vrooli/proto-types/quality-health/v1/audit/audit_pb";
import type {
  AuditQualityResponse,
  ExplainFindingResponse,
  FixConfigResponse,
} from "@vrooli/proto-types/quality-health/v1/audit/audit_pb";

export const makeAuditQualityResponse = (
  overrides: MessageInitShape<typeof AuditQualityResponseSchema> = {},
): AuditQualityResponse => {
  const base: MessageInitShape<typeof AuditQualityResponseSchema> = {
    runId: "run-1",
    status: "failed",
    summary: "Quality Health found strictness drift.",
    scenario: "quality-health",
    targetKind: "scenario",
    targetPath: "/repo/scenarios/quality-health",
    surfaces: [
      {
        id: "ui",
        kind: "ui",
        language: "typescript",
        framework: "react-vite",
        rootPath: "scenarios/quality-health/ui",
        packageManager: "pnpm",
        status: "failed",
        confidence: 1,
      },
      {
        id: "api",
        kind: "api",
        language: "go",
        framework: "connect",
        rootPath: "scenarios/quality-health/api",
        packageManager: "go",
        status: "passed",
        confidence: 1,
      },
    ],
    contracts: [
      {
        contractId: "typescript-react-vite-strict",
        surfaceId: "ui",
        status: "failed",
        ruleIds: ["TS_CONFIG_STRICT"],
      },
    ],
    findings: [
      {
        id: "finding-tsconfig",
        scenario: "quality-health",
        targetKind: "scenario",
        surfaceId: "ui",
        surfaceKind: "ui",
        language: "typescript",
        framework: "react-vite",
        ruleId: "TS_CONFIG_STRICT",
        category: "config",
        severity: "error",
        filePath: "ui/tsconfig.json",
        message: "TypeScript strictness guardrail is missing.",
        evidence: "noUncheckedIndexedAccess is false.",
        expected: "strict true and guardrail comment present",
        observed: "noUncheckedIndexedAccess false",
        whyItMatters: "Agents must fix nullability rather than weakening validation.",
        remediation: "Restore strict TypeScript config and the protective comments.",
        autofixAvailable: true,
        autofixCommand: "quality-health fix-config quality-health --rule TS_CONFIG_STRICT --apply",
      },
      {
        id: "finding-pattern",
        scenario: "quality-health",
        targetKind: "scenario",
        surfaceId: "ui",
        surfaceKind: "ui",
        language: "typescript",
        framework: "react-vite",
        ruleId: "TS_DANGEROUS_PATTERNS",
        category: "source",
        severity: "warning",
        filePath: "ui/src/App.tsx",
        message: "Dangerous TypeScript suppression found.",
        evidence: "as any",
        expected: "specific type guard",
        observed: "broad assertion",
        whyItMatters: "Unsafe casts hide runtime crashes.",
        remediation: "Replace broad casts with a real type guard.",
      },
    ],
    commandResults: [
      {
        name: "ui lint",
        command: "pnpm run lint",
        workingDirectory: "scenarios/quality-health/ui",
        status: "passed",
        exitCode: 0,
        timeoutSeconds: 120,
      },
    ],
    maturity: {
      rung: 2,
      label: "Contracted",
      rationale: "Contracts are registered but one UI guardrail fails.",
    },
    counts: {
      errors: 1,
      warnings: 1,
      infos: 0,
      surfaces: 2,
      contracts: 1,
    },
    nextSteps: ["Run fix-config for TS_CONFIG_STRICT."],
    autofixCandidates: [
      {
        ruleId: "TS_CONFIG_STRICT",
        filePath: "ui/tsconfig.json",
        description: "Restore strict options and guardrail comments.",
        before: "\"noUncheckedIndexedAccess\": false",
        after: "\"noUncheckedIndexedAccess\": true",
      },
    ],
  };
  return { ...base, ...overrides } as unknown as AuditQualityResponse;
};

export const makeExplainFindingResponse = (
  overrides: MessageInitShape<typeof ExplainFindingResponseSchema> = {},
): ExplainFindingResponse => {
  const base: MessageInitShape<typeof ExplainFindingResponseSchema> = {
    whyItMatters: "Strict TypeScript settings catch null and indexing bugs before runtime.",
    remediation: "Keep strict settings and fix unsafe code paths directly.",
    nextSteps: ["Preview the deterministic TS config fix."],
  };
  return { ...base, ...overrides } as unknown as ExplainFindingResponse;
};

export const makeFixConfigResponse = (
  overrides: MessageInitShape<typeof FixConfigResponseSchema> = {},
): FixConfigResponse => {
  const base: MessageInitShape<typeof FixConfigResponseSchema> = {
    scenario: "quality-health",
    applied: false,
    candidates: [
      {
        ruleId: "TS_CONFIG_STRICT",
        filePath: "ui/tsconfig.json",
        description: "Restore strict TypeScript config.",
        before: "\"strict\": false",
        after: "\"strict\": true",
      },
    ],
    messages: ["Preview only."],
  };
  return { ...base, ...overrides } as unknown as FixConfigResponse;
};
