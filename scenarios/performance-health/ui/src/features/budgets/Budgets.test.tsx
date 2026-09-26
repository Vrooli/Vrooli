import { describe, expect, it, vi, beforeEach } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { ScenarioProvider } from "../perf/ScenarioContext";
import { BudgetsPanel } from "./BudgetsPanel";

const scanFleet = vi.fn();
const getBudget = vi.fn();
const setBudget = vi.fn();
const checkBudget = vi.fn();

vi.mock("../../api/perf", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/perf")>();
  return {
    ...actual,
    perfClient: {
      scanFleet: (...a: unknown[]) => scanFleet(...a),
      getBudget: (...a: unknown[]) => getBudget(...a),
      setBudget: (...a: unknown[]) => setBudget(...a),
      checkBudget: (...a: unknown[]) => checkBudget(...a),
    },
  };
});

const renderBudgets = () =>
  renderWithProviders(
    <ScenarioProvider>
      <BudgetsPanel />
    </ScenarioProvider>,
  );

const declaredBudget = {
  budget: {
    scenario: "performance-health",
    goBuildMaxMs: 90_000n,
    uiBuildMaxMs: 180_000n,
    bundleMaxBytes: 0n,
    lcpMaxMs: 2500n,
    startupMaxMs: 0n,
    ratchet: true,
  },
  declared: true,
};

beforeEach(() => {
  vi.clearAllMocks();
  scanFleet.mockResolvedValue({
    entries: [{ scenario: "performance-health", tier: "1" }],
    tierDistribution: [],
    errors: [],
    scenarioCount: 1,
    noBudgetCount: 0,
    regressedCount: 0,
  });
});

describe("BudgetsPanel (cimode — copy-independent)", () => {
  it("loads the declared budget into the form", async () => {
    getBudget.mockResolvedValue(declaredBudget);
    renderBudgets();
    await waitFor(() => expect(screen.getByTestId(selectors.budgets.form)).toBeInTheDocument());
    expect(screen.getByTestId(selectors.budgets.field({ field: "goBuildMaxMs" }))).toHaveValue(90000);
    // A declared budget hides the "not declared" banner.
    expect(screen.queryByTestId(selectors.budgets.notDeclared)).not.toBeInTheDocument();
  });

  it("shows the not-declared banner when no budget exists", async () => {
    getBudget.mockResolvedValue({ budget: undefined, declared: false });
    renderBudgets();
    await waitFor(() =>
      expect(screen.getByTestId(selectors.budgets.notDeclared)).toBeInTheDocument(),
    );
  });

  it("edits a field and saves the budget", async () => {
    getBudget.mockResolvedValue(declaredBudget);
    setBudget.mockResolvedValue({});
    renderBudgets();
    await waitFor(() => expect(screen.getByTestId(selectors.budgets.form)).toBeInTheDocument());
    fireEvent.change(screen.getByTestId(selectors.budgets.field({ field: "lcpMaxMs" })), {
      target: { value: "3000" },
    });
    fireEvent.click(screen.getByTestId(selectors.budgets.ratchet));
    fireEvent.click(screen.getByTestId(selectors.budgets.saveButton));
    await waitFor(() => expect(screen.getByTestId(selectors.budgets.saved)).toBeInTheDocument());
    expect(setBudget).toHaveBeenCalledTimes(1);
  });

  it("runs a budget check and renders a failing verdict with violations", async () => {
    getBudget.mockResolvedValue(declaredBudget);
    checkBudget.mockResolvedValue({
      passed: false,
      violations: [{ axis: "lcp", measured: 4000n, budget: 2500n }],
    });
    renderBudgets();
    await waitFor(() => expect(screen.getByTestId(selectors.budgets.form)).toBeInTheDocument());
    fireEvent.click(screen.getByTestId(selectors.budgets.checkButton));
    await waitFor(() =>
      expect(screen.getByTestId(selectors.budgets.checkResult)).toBeInTheDocument(),
    );
    expect(screen.getByTestId(selectors.budgets.checkVerdict)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.budgets.checkResult)).toHaveTextContent("lcp");
  });

  it("renders a passing verdict with no violation table", async () => {
    getBudget.mockResolvedValue(declaredBudget);
    checkBudget.mockResolvedValue({ passed: true, violations: [] });
    renderBudgets();
    await waitFor(() => expect(screen.getByTestId(selectors.budgets.form)).toBeInTheDocument());
    fireEvent.click(screen.getByTestId(selectors.budgets.checkButton));
    await waitFor(() =>
      expect(screen.getByTestId(selectors.budgets.checkVerdict)).toBeInTheDocument(),
    );
  });

  it("shows an actionable error state when the budget fails to load", async () => {
    getBudget.mockRejectedValue(new Error("budget boom"));
    renderBudgets();
    await waitFor(() =>
      expect(screen.getByTestId(selectors.budgets.error)).toBeInTheDocument(),
    );
    expect(screen.getByTestId(selectors.state.errorRetry)).toBeInTheDocument();
  });
});
