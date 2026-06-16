import { cleanup, screen, waitFor } from "@testing-library/react";
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
    const user = userEvent.setup();
    mocks.validateScenario.mockResolvedValue(makeValidateScenarioResponse());

    renderWithProviders(<ScenarioValidationWorkbench />);
    await user.click(screen.getByTestId(selectors.validationWorkbench.runButton));

    expect(await screen.findByTestId(selectors.validationWorkbench.status)).toHaveTextContent("failed");
    expect(screen.getByTestId(selectors.validationWorkbench.maturity)).toHaveTextContent("R2");
    expect(screen.getByTestId(selectors.validationWorkbench.findingRow({ id: "finding-coverage" }))).toHaveTextContent(
      "UNIT_COVERAGE_BELOW_THRESHOLD",
    );
    expect(screen.getByTestId(selectors.validationWorkbench.findingRow({ id: "finding-flake" }))).toBeInTheDocument();

    expect(mocks.validateScenario).toHaveBeenCalledWith({
      scenario: "unit-health",
      includeExecution: true,
      useCache: true,
    });
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

  it("surfaces the degraded reason and empty findings state", async () => {
    const user = userEvent.setup();
    mocks.validateScenario.mockResolvedValue(
      makeValidateScenarioResponse({
        status: "degraded",
        degradedReason: "Code Facts discovery returned no surfaces.",
        findings: [],
        counts: { errors: 0, warnings: 0, infos: 0, surfaces: 0, workspaces: 0, coverageTargets: 0 },
      }),
    );

    renderWithProviders(<ScenarioValidationWorkbench />);
    await user.click(screen.getByTestId(selectors.validationWorkbench.runButton));

    expect(await screen.findByTestId(selectors.validationWorkbench.degraded)).toHaveTextContent(
      "Code Facts discovery returned no surfaces.",
    );
    expect(screen.getByTestId(selectors.validationWorkbench.empty)).toBeInTheDocument();
  });

  it("renders the error state when validation fails", async () => {
    const user = userEvent.setup();
    mocks.validateScenario.mockRejectedValue(new Error("boom"));

    renderWithProviders(<ScenarioValidationWorkbench />);
    await user.click(screen.getByTestId(selectors.validationWorkbench.runButton));

    expect(await screen.findByTestId(selectors.validationWorkbench.error)).toBeInTheDocument();
  });
});
