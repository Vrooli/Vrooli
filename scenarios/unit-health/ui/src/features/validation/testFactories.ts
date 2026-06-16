import type { MessageInitShape } from "@bufbuild/protobuf";
import { ValidateScenarioResponseSchema } from "@vrooli/proto-types/unit-health/v1/validation/validation_pb";
import type { ValidateScenarioResponse } from "@vrooli/proto-types/unit-health/v1/validation/validation_pb";

/**
 * Build a stable `ValidateScenarioResponse` for workbench tests. Defaults
 * model the common "found two findings" path so most tests call
 * `makeValidateScenarioResponse()` with no args; pass overrides to exercise
 * empty / degraded / passing variants. The `as unknown as` cast mirrors the
 * sibling quality-health factory: `create()` would also work, but the spread
 * keeps the literal nested shapes readable and the workbench never relies on
 * proto reflection state for the values it renders.
 */
export const makeValidateScenarioResponse = (
  overrides: MessageInitShape<typeof ValidateScenarioResponseSchema> = {},
): ValidateScenarioResponse => {
  const base: MessageInitShape<typeof ValidateScenarioResponseSchema> = {
    runId: "run-1",
    status: "failed",
    summary: "Unit Health found coverage and flake regressions.",
    scenario: "unit-health",
    targetKind: "scenario",
    targetPath: "/repo/scenarios/unit-health",
    degradedReason: "",
    surfaces: [
      {
        id: "ui",
        kind: "ui",
        language: "typescript",
        framework: "react-vite",
        rootPath: "scenarios/unit-health/ui",
        packageManager: "pnpm",
        status: "failed",
        confidence: 1,
      },
      {
        id: "api",
        kind: "api",
        language: "go",
        framework: "connect",
        rootPath: "scenarios/unit-health/api",
        packageManager: "go",
        status: "passed",
        confidence: 1,
      },
    ],
    workspaces: [
      {
        id: "ui",
        language: "typescript",
        rootPath: "scenarios/unit-health/ui",
        framework: "vitest",
        canonicalFramework: "vitest",
        testCommand: "pnpm test",
        coverageCommand: "pnpm test:coverage",
        packageManager: "pnpm",
        status: "failed",
        degradedReason: "",
      },
    ],
    commandResults: [
      {
        name: "ui test",
        command: "pnpm test",
        workingDirectory: "scenarios/unit-health/ui",
        status: "failed",
        exitCode: 1,
        timeoutSeconds: 120,
      },
    ],
    coverage: [
      {
        id: "ui-coverage",
        language: "typescript",
        surfaceId: "ui",
        filePath: "scenarios/unit-health/ui/src",
        coveredLines: 180n,
        totalLines: 240n,
        coveragePercent: 75,
        threshold: 80,
        status: "below",
      },
    ],
    findings: [
      {
        id: "finding-coverage",
        scenario: "unit-health",
        surfaceId: "ui",
        workspaceId: "ui",
        language: "typescript",
        framework: "vitest",
        code: "UNIT_COVERAGE_BELOW_THRESHOLD",
        category: "coverage",
        severity: "error",
        filePath: "ui/src/features/validation/ScenarioValidationWorkbench.tsx",
        symbol: "ScenarioValidationWorkbench",
        message: "Coverage is below the configured threshold.",
        evidence: "75% covered, threshold is 80%.",
        expected: "coverage >= 80%",
        observed: "coverage = 75%",
        whyItMatters: "Untested branches hide regressions that pass review.",
        remediation: "Add tests for the uncovered branches before promoting.",
        sourceCommand: "pnpm test:coverage",
        createdAt: "2026-01-01T00:00:00.000Z",
      },
      {
        id: "finding-flake",
        scenario: "unit-health",
        surfaceId: "ui",
        workspaceId: "ui",
        language: "typescript",
        framework: "vitest",
        code: "UNIT_FLAKY_TEST",
        category: "flake",
        severity: "warning",
        filePath: "ui/src/features/health/HealthCard.test.tsx",
        symbol: "",
        message: "A test failed only on retry.",
        evidence: "1 of 3 runs failed without a code change.",
        expected: "deterministic test outcome",
        observed: "non-deterministic failure",
        whyItMatters: "Flaky tests erode trust in the suite and mask real failures.",
        remediation: "Stabilize the async assertion or pin the clock.",
        sourceCommand: "pnpm test",
        createdAt: "2026-01-01T00:00:00.000Z",
      },
    ],
    diagnostics: [
      {
        kind: "flake",
        workspaceId: "ui",
        message: "HealthCard test failed only on retry.",
        evidence: "1 of 3 runs failed.",
        severity: "warning",
      },
    ],
    maturity: {
      rung: 2,
      label: "Covered",
      rationale: "Commands run but coverage sits below the threshold.",
    },
    counts: {
      errors: 1,
      warnings: 1,
      infos: 0,
      surfaces: 2,
      workspaces: 1,
      coverageTargets: 1,
    },
    nextSteps: ["Raise UI coverage above 80%.", "Stabilize the flaky HealthCard test."],
  };
  return { ...base, ...overrides } as unknown as ValidateScenarioResponse;
};
