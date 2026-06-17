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
      {
        id: "api",
        language: "go",
        rootPath: "scenarios/unit-health/api",
        framework: "gotest",
        canonicalFramework: "go-test",
        testCommand: "go test ./...",
        coverageCommand: "go test -cover ./...",
        packageManager: "go",
        status: "passed",
        degradedReason: "",
      },
    ],
    plan: {
      commands: [
        {
          workspaceId: "ui",
          name: "ui test",
          command: "pnpm test",
          workingDirectory: "scenarios/unit-health/ui",
          timeoutSeconds: 120,
        },
        {
          workspaceId: "api",
          name: "api test",
          command: "go test ./...",
          workingDirectory: "scenarios/unit-health/api",
          timeoutSeconds: 300,
        },
      ],
      notes: "Two workspaces discovered.",
    },
    commandResults: [
      {
        name: "ui test",
        command: "pnpm test",
        workingDirectory: "scenarios/unit-health/ui",
        status: "failed",
        exitCode: 1,
        stdoutExcerpt: "1 failed | 12 passed",
        stderrExcerpt: "AssertionError: expected true to be false",
        timeoutSeconds: 120,
        failureReason: "assertion failure",
        failureClass: "test_failure",
        durationMs: 4200n,
      },
      {
        name: "api test",
        command: "go test ./...",
        workingDirectory: "scenarios/unit-health/api",
        status: "passed",
        exitCode: 0,
        stdoutExcerpt: "",
        stderrExcerpt: "",
        timeoutSeconds: 300,
        failureReason: "",
        failureClass: "",
        durationMs: 1800n,
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
        filePath: "ui/src/features/validation/ScenarioValidationWorkbench.test.tsx",
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
    artifacts: [
      { label: "Validation run", kind: "run", reference: "run-1" },
      { label: "Target", kind: "target", reference: "/repo/scenarios/unit-health" },
      { label: "ui test", kind: "command", reference: "scenarios/unit-health/ui" },
      { label: "Coverage (ui)", kind: "coverage", reference: "scenarios/unit-health/ui" },
    ],
    assessment: {
      scenario: "unit-health",
      provider: "unit-health",
      phase: "validate",
      version: "1.0.0",
      local: {
        currentLevel: "R2 Covered",
        nextLevel: "R3 Reliable",
        levels: [
          {
            id: "R2",
            name: "Covered",
            description: "Test commands run and coverage is measured.",
            entryCriteria: ["Surfaces discovered", "Commands run"],
            exitCriteria: ["Coverage at or above threshold", "No flaky tests"],
          },
          {
            id: "R3",
            name: "Reliable",
            description: "Tests are deterministic and coverage holds.",
            entryCriteria: ["Coverage at or above threshold"],
            exitCriteria: ["Zero flakes over N runs", "Mutation score above target"],
          },
        ],
        blockingFindingCodes: ["UNIT_COVERAGE_BELOW_THRESHOLD", "UNIT_FLAKY_TEST"],
      },
      findings: [],
      findingsByGlobalImpact: {
        regression_risk: 2,
        maintainability: 1,
      },
      findingsBySeverity: {
        error: 1,
        warning: 1,
      },
      recommendedSkillIds: ["raise-coverage", "stabilize-flaky-tests"],
    },
  };
  return { ...base, ...overrides } as unknown as ValidateScenarioResponse;
};
