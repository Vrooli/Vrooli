import { describe, expect, it, vi, beforeEach } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { ScenarioProvider } from "../perf/ScenarioContext";
import { TraceAnalyzer } from "./TraceAnalyzer";

const scanFleet = vi.fn();
const analyzeTrace = vi.fn();

vi.mock("../../api/perf", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/perf")>();
  return {
    ...actual,
    perfClient: {
      scanFleet: (...a: unknown[]) => scanFleet(...a),
      analyzeTrace: (...a: unknown[]) => analyzeTrace(...a),
    },
  };
});

const renderTrace = (entries: string[] = ["/trace"]) =>
  renderWithProviders(
    <ScenarioProvider>
      <TraceAnalyzer />
    </ScenarioProvider>,
    { routerEntries: entries },
  );

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

describe("TraceAnalyzer (cimode — copy-independent)", () => {
  it("shows the empty state before any trace is analyzed", () => {
    renderTrace();
    expect(screen.getByTestId(selectors.trace.empty)).toBeInTheDocument();
    // Analyze is disabled with no artifact entered.
    expect(screen.getByTestId(selectors.trace.analyzeButton)).toBeDisabled();
  });

  it("analyzes a trace and renders vitals + the component table + findings", async () => {
    analyzeTrace.mockResolvedValue({
      scenario: "performance-health",
      lcpMs: 900n,
      fcpMs: 400n,
      longTaskMs: 120n,
      components: [
        { component: "List", commitCount: 12, avgMs: 4.5, maxMs: 18.2, definition: "ui/List.tsx:10" },
      ],
      findings: [
        {
          code: "PERF_SLOW_COMPONENT",
          severity: "warning",
          component: "List",
          message: "List commits too often",
          definition: "ui/List.tsx:10",
          evidence: "12 commits",
        },
      ],
    });

    renderTrace();
    fireEvent.change(screen.getByTestId(selectors.trace.artifactInput), {
      target: { value: "/tmp/performance.json" },
    });
    fireEvent.click(screen.getByTestId(selectors.trace.analyzeButton));

    await waitFor(() =>
      expect(screen.getByTestId(selectors.trace.components)).toBeInTheDocument(),
    );
    expect(screen.getByTestId(selectors.trace.vitals)).toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.trace.componentRow({ component: "List" })),
    ).toBeInTheDocument();
    expect(screen.getByTestId(selectors.trace.findings)).toHaveTextContent("List");
  });

  it("renders the empty component/findings sub-states for a Tier-0 trace", async () => {
    analyzeTrace.mockResolvedValue({
      scenario: "performance-health",
      lcpMs: 900n,
      fcpMs: 400n,
      longTaskMs: 0n,
      components: [],
      findings: [],
    });
    renderTrace();
    fireEvent.change(screen.getByTestId(selectors.trace.artifactInput), {
      target: { value: "/tmp/t.json" },
    });
    fireEvent.click(screen.getByTestId(selectors.trace.analyzeButton));
    await waitFor(() =>
      expect(screen.getByTestId(selectors.trace.findingsEmpty)).toBeInTheDocument(),
    );
  });

  it("shows an actionable error state when analysis fails", async () => {
    analyzeTrace.mockRejectedValue(new Error("analyze boom"));
    renderTrace();
    fireEvent.change(screen.getByTestId(selectors.trace.artifactInput), {
      target: { value: "/tmp/t.json" },
    });
    fireEvent.click(screen.getByTestId(selectors.trace.analyzeButton));
    await waitFor(() =>
      expect(screen.getByTestId(selectors.trace.error)).toBeInTheDocument(),
    );
    expect(screen.getByTestId(selectors.state.errorRetry)).toBeInTheDocument();
  });

  it("adopts the artifact from the ?artifact= deep link", async () => {
    renderTrace(["/trace?scenario=performance-health&artifact=/deep/link.json"]);
    await waitFor(() =>
      expect(screen.getByTestId(selectors.trace.artifactInput)).toHaveValue("/deep/link.json"),
    );
  });
});
