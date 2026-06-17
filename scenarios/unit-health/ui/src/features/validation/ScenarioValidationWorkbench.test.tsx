import { cleanup, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { renderWithProviders } from "../../test-utils";
import { makeValidateScenarioResponse } from "./testFactories";
import { ScenarioValidationWorkbench } from "./ScenarioValidationWorkbench";

const mocks = vi.hoisted(() => ({
  validateScenario: vi.fn(),
}));

vi.mock("../../api/validation", () => ({
  validationClient: mocks,
}));

const runDefault = async () => {
  const user = userEvent.setup();
  mocks.validateScenario.mockResolvedValue(makeValidateScenarioResponse());
  renderWithProviders(<ScenarioValidationWorkbench />);
  await user.click(screen.getByTestId(selectors.validationWorkbench.runButton));
  await screen.findByTestId(selectors.validationWorkbench.status);
  return user;
};

describe("ScenarioValidationWorkbench", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("shows the idle prompt before a validation has run", () => {
    renderWithProviders(<ScenarioValidationWorkbench />);
    expect(screen.getByTestId(selectors.validationWorkbench.idle)).toHaveTextContent(
      strings.validation.idle,
    );
  });

  it("runs the default scenario and renders the maturity verdict, counts, and findings", async () => {
    await runDefault();

    expect(screen.getByTestId(selectors.validationWorkbench.status)).toHaveTextContent("failed");
    expect(screen.getByTestId(selectors.validationWorkbench.maturity)).toHaveTextContent("R2");
    expect(
      screen.getByTestId(selectors.validationWorkbench.findingRow({ id: "finding-coverage" })),
    ).toHaveTextContent("UNIT_COVERAGE_BELOW_THRESHOLD");
    expect(
      screen.getByTestId(selectors.validationWorkbench.findingRow({ id: "finding-flake" })),
    ).toBeInTheDocument();

    expect(mocks.validateScenario).toHaveBeenCalledWith({
      scenario: "unit-health",
      includeExecution: true,
      useCache: true,
    });
  });

  it("renders the local-maturity summary with next-level and exit criteria", async () => {
    await runDefault();

    const summary = screen.getByTestId(selectors.validationWorkbench.maturitySummary);
    expect(summary).toHaveTextContent("R3 Reliable");
    expect(within(summary).getByTestId(selectors.validationWorkbench.nextLevel)).toHaveTextContent(
      "R3 Reliable",
    );
    // Blocking finding codes are listed.
    expect(summary).toHaveTextContent("UNIT_FLAKY_TEST");
    // Exit criteria for the next level are surfaced.
    expect(summary).toHaveTextContent("Zero flakes over N runs");
  });

  it("renders the test plan with a noncanonical-framework flag and timeouts", async () => {
    await runDefault();

    const ui = screen.getByTestId(selectors.validationWorkbench.workspaceRow({ id: "ui" }));
    expect(ui).toHaveTextContent("pnpm test");
    // cimode echoes the key path for the interpolated timeout string.
    expect(ui).toHaveTextContent(strings.validation.timeoutSeconds);

    const api = screen.getByTestId(selectors.validationWorkbench.workspaceRow({ id: "api" }));
    // gotest != go-test canonical => noncanonical flag (key path) shown.
    expect(api).toHaveTextContent(strings.validation.noncanonicalFramework);
    expect(api).toHaveTextContent("gotest");
    expect(api).toHaveTextContent("go test ./...");
  });

  it("renders execution results and toggles captured output", async () => {
    const user = await runDefault();

    const uiRow = screen.getByTestId(selectors.validationWorkbench.commandRow({ name: "ui test" }));
    expect(uiRow).toHaveTextContent("failed");
    expect(uiRow).toHaveTextContent("test_failure");
    // durationMs is interpolated; cimode echoes the key path.
    expect(uiRow).toHaveTextContent(strings.validation.durationMs);

    // Output hidden until toggled.
    expect(
      screen.queryByTestId(selectors.validationWorkbench.commandOutput({ name: "ui test" })),
    ).not.toBeInTheDocument();

    await user.click(
      screen.getByTestId(selectors.validationWorkbench.commandOutputToggle({ name: "ui test" })),
    );
    const output = screen.getByTestId(
      selectors.validationWorkbench.commandOutput({ name: "ui test" }),
    );
    expect(output).toHaveTextContent("AssertionError");
    expect(output).toHaveTextContent("1 failed | 12 passed");
    expect(output).toHaveTextContent("assertion failure");

    // Toggle again collapses.
    await user.click(
      screen.getByTestId(selectors.validationWorkbench.commandOutputToggle({ name: "ui test" })),
    );
    expect(
      screen.queryByTestId(selectors.validationWorkbench.commandOutput({ name: "ui test" })),
    ).not.toBeInTheDocument();

    // The passing api command has no output, so no toggle button.
    expect(
      screen.queryByTestId(selectors.validationWorkbench.commandOutputToggle({ name: "api test" })),
    ).not.toBeInTheDocument();
  });

  it("renders the coverage dashboard with per-surface roll-up and per-file rows", async () => {
    await runDefault();

    const surface = screen.getByTestId(
      selectors.validationWorkbench.coverageSurface({ id: "ui" }),
    );
    // Roll-up text is interpolated; cimode echoes the key path.
    expect(surface).toHaveTextContent(strings.validation.coverageSurfaceRollup);

    const row = screen.getByTestId(
      selectors.validationWorkbench.coverageRow({ id: "ui-coverage" }),
    );
    expect(row).toHaveTextContent("180 / 240");
    expect(row).toHaveTextContent("75%");
    expect(row).toHaveTextContent("80%");
    expect(row).toHaveTextContent("below");
  });

  it("renders findings grouped by category with remediation context", async () => {
    await runDefault();

    expect(
      screen.getByTestId(selectors.validationWorkbench.findingCategory({ category: "coverage" })),
    ).toBeInTheDocument();
    const flake = screen.getByTestId(
      selectors.validationWorkbench.findingCategory({ category: "flake" }),
    );
    expect(flake).toHaveTextContent("Stabilize the async assertion");
    expect(flake).toHaveTextContent("deterministic test outcome");
  });

  it("renders diagnostics", async () => {
    await runDefault();
    const diagnostics = screen.getByTestId(selectors.validationWorkbench.diagnostics);
    expect(diagnostics).toHaveTextContent("HealthCard test failed only on retry.");
  });

  it("renders the global-impact grouping, recommended skills, and next steps", async () => {
    await runDefault();

    const impact = screen.getByTestId(
      selectors.validationWorkbench.impactRow({ key: "regression_risk" }),
    );
    expect(impact).toHaveTextContent("regression_risk");
    expect(impact).toHaveTextContent("2");

    const skills = screen.getByTestId(selectors.validationWorkbench.recommendedSkills);
    expect(skills).toHaveTextContent("raise-coverage");

    const nextSteps = screen.getByTestId(selectors.validationWorkbench.nextSteps);
    expect(nextSteps).toHaveTextContent("Raise UI coverage above 80%.");
  });

  it("submits a typed scenario name", async () => {
    const user = userEvent.setup();
    mocks.validateScenario.mockResolvedValue(makeValidateScenarioResponse());

    renderWithProviders(<ScenarioValidationWorkbench />);

    const input = screen.getByTestId(selectors.validationWorkbench.scenarioInput);
    await user.clear(input);
    await user.type(input, "web-console");
    await user.click(screen.getByTestId(selectors.validationWorkbench.runButton));

    await waitFor(() => {
      expect(mocks.validateScenario).toHaveBeenLastCalledWith({
        scenario: "web-console",
        includeExecution: true,
        useCache: true,
      });
    });
  });

  it("ignores a blank scenario submission", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ScenarioValidationWorkbench />);
    const input = screen.getByTestId(selectors.validationWorkbench.scenarioInput);
    await user.clear(input);
    await user.click(screen.getByTestId(selectors.validationWorkbench.runButton));
    expect(mocks.validateScenario).not.toHaveBeenCalled();
  });

  it("surfaces the degraded reason and all empty states with no data", async () => {
    const user = userEvent.setup();
    mocks.validateScenario.mockResolvedValue(
      makeValidateScenarioResponse({
        status: "degraded",
        degradedReason: "Code Facts discovery returned no surfaces.",
        summary: "",
        surfaces: [],
        workspaces: [],
        plan: undefined,
        commandResults: [],
        coverage: [],
        findings: [],
        diagnostics: [],
        maturity: undefined,
        assessment: undefined,
        nextSteps: [],
        counts: { errors: 0, warnings: 0, infos: 0, surfaces: 0, workspaces: 0, coverageTargets: 0 },
      }),
    );

    renderWithProviders(<ScenarioValidationWorkbench />);
    await user.click(screen.getByTestId(selectors.validationWorkbench.runButton));

    expect(await screen.findByTestId(selectors.validationWorkbench.degraded)).toHaveTextContent(
      "Code Facts discovery returned no surfaces.",
    );
    // Maturity metric falls back to Unknown when absent.
    expect(screen.getByTestId(selectors.validationWorkbench.maturity)).toHaveTextContent(
      strings.validation.unknown,
    );
    // No local assessment => maturity summary panel is not rendered.
    expect(
      screen.queryByTestId(selectors.validationWorkbench.maturitySummary),
    ).not.toBeInTheDocument();
    expect(screen.getByTestId(selectors.validationWorkbench.testPlanEmpty)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.validationWorkbench.executionEmpty)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.validationWorkbench.coverageEmpty)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.validationWorkbench.empty)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.validationWorkbench.diagnosticsEmpty)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.validationWorkbench.globalImpactEmpty)).toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.validationWorkbench.recommendedSkillsEmpty),
    ).toBeInTheDocument();
  });

  it("handles a passed run with empty status, no blockers, and no next level", async () => {
    const user = userEvent.setup();
    mocks.validateScenario.mockResolvedValue(
      makeValidateScenarioResponse({
        status: "",
        assessment: {
          scenario: "unit-health",
          provider: "unit-health",
          phase: "validate",
          version: "1.0.0",
          local: {
            currentLevel: "R4 Hardened",
            nextLevel: "",
            levels: [],
            blockingFindingCodes: [],
          },
          findings: [],
          findingsByGlobalImpact: {},
          findingsBySeverity: {},
          recommendedSkillIds: [],
        },
        coverage: [
          {
            id: "empty-cov",
            language: "go",
            surfaceId: "api",
            filePath: "",
            coveredLines: 0n,
            totalLines: 0n,
            coveragePercent: 0,
            threshold: 0,
            status: "ok",
          },
        ],
      }),
    );

    renderWithProviders(<ScenarioValidationWorkbench />);
    await user.click(screen.getByTestId(selectors.validationWorkbench.runButton));

    // Empty status falls back to Unknown.
    expect(await screen.findByTestId(selectors.validationWorkbench.status)).toHaveTextContent(
      strings.validation.unknown,
    );
    const summary = screen.getByTestId(selectors.validationWorkbench.maturitySummary);
    expect(summary).toHaveTextContent(strings.validation.noNextLevel);
    expect(summary).toHaveTextContent(strings.validation.noBlockers);
    // Zero-line coverage roll-up renders without crashing (0/0 division guard).
    expect(
      screen.getByTestId(selectors.validationWorkbench.coverageRow({ id: "empty-cov" })),
    ).toHaveTextContent("0 / 0");
  });

  it("renders the loading state while the mutation is pending", async () => {
    const user = userEvent.setup();
    let resolve: ((value: unknown) => void) | undefined;
    mocks.validateScenario.mockReturnValue(
      new Promise((res) => {
        resolve = res;
      }),
    );

    renderWithProviders(<ScenarioValidationWorkbench />);
    await user.click(screen.getByTestId(selectors.validationWorkbench.runButton));

    expect(await screen.findByTestId(selectors.validationWorkbench.loading)).toBeInTheDocument();
    resolve?.(makeValidateScenarioResponse());
    await screen.findByTestId(selectors.validationWorkbench.status);
  });

  it("renders the error state when validation fails", async () => {
    const user = userEvent.setup();
    mocks.validateScenario.mockRejectedValue(new Error("boom"));

    renderWithProviders(<ScenarioValidationWorkbench />);
    await user.click(screen.getByTestId(selectors.validationWorkbench.runButton));

    expect(await screen.findByTestId(selectors.validationWorkbench.error)).toBeInTheDocument();
  });
});
