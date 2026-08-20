import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, within } from "@testing-library/react";
import { QueryClient } from "@tanstack/react-query";

import { renderWithProviders } from "../test-utils";

const mocks = vi.hoisted(() => ({
  fetchCells: vi.fn(),
  fetchCondition: vi.fn(),
  fetchCoverage: vi.fn(),
  fetchFocus: vi.fn(),
  fetchTrust: vi.fn(),
}));

vi.mock("../api/reliability", () => mocks);

import { ConditionPage } from "./condition/ConditionPage";
import { CoveragePage } from "./coverage/CoveragePage";
import { FocusPage } from "./focus/FocusPage";

function renderPage(ui: Parameters<typeof renderWithProviders>[0]) {
  return renderWithProviders(ui, {
    queryClient: new QueryClient({ defaultOptions: { queries: { retry: false } } }),
  });
}

describe("reliability detail pages", () => {
  beforeEach(() => vi.clearAllMocks());
  afterEach(() => cleanup());

  it("names loading state on each detail page", () => {
    const pending = () => new Promise<never>(() => undefined);
    mocks.fetchCondition.mockImplementation(pending);
    mocks.fetchTrust.mockImplementation(pending);
    mocks.fetchCoverage.mockImplementation(pending);
    mocks.fetchCells.mockImplementation(pending);
    mocks.fetchFocus.mockImplementation(pending);

    const { unmount } = renderPage(<ConditionPage />);
    expect(screen.getByText("Reading trusted condition…")).toBeInTheDocument();
    unmount();
    renderPage(<CoveragePage />);
    expect(screen.getByText("Reading coverage spaces…")).toBeInTheDocument();
    cleanup();
    renderPage(<FocusPage />);
    expect(screen.getByText("Ranking next findings…")).toBeInTheDocument();
  });

  it("renders condition readings, trust, history, and unavailable sources", async () => {
    mocks.fetchCondition.mockResolvedValue({
      readings: [{ id: "r-1", cellRef: "api/latency", value: 42, unit: "ms", source: "autoheal", trustVerdict: 1, bandVerdict: 1 }],
      sources: [{ source: "autoheal", available: false, reason: "provider paused" }],
    });
    mocks.fetchTrust.mockResolvedValue({ trust: { distribution: [{ verdict: 1, count: 1 }], checkedDenominator: 1, total: 2 } });

    renderPage(<ConditionPage />);

    expect(await screen.findByText("api/latency")).toBeInTheDocument();
    expect(screen.getByText(/provider paused/)).toBeInTheDocument();
    expect(screen.getByText("Select a cell to inspect history")).toBeInTheDocument();
  });

  it("reports a condition source error without inferring health", async () => {
    mocks.fetchCondition.mockRejectedValue(new Error("condition down"));
    mocks.fetchTrust.mockRejectedValue(new Error("trust down"));
    renderPage(<ConditionPage />);
    expect(await screen.findByText("Condition unavailable")).toBeInTheDocument();
    expect(screen.getByText(/No healthy reading is inferred/)).toBeInTheDocument();
  });

  it("renders coverage ratios, dated open loops, and integrity findings", async () => {
    mocks.fetchCoverage.mockResolvedValue({
      projections: [{ projection: 1, available: true, ratio: { value: 0.5 }, confidence: { level: 1, rationale: "owner-authored" }, nowCount: 1, inReachCount: 1, missingCount: 1 }],
      integrityFindings: [{ code: "MISSING_TARGET", location: "api/latency", message: "target is absent" }],
    });
    mocks.fetchCells.mockResolvedValue({ cells: [{ projection: 1, id: "api/latency", status: 3, question: "Can latency be read?", gapOpenedOn: "2026-08-01", gapOpenDays: 19 }] });

    renderPage(<CoveragePage />);

    expect(await screen.findByText("supervision")).toBeInTheDocument();
    expect(screen.getByText("Open loop (1)")).toBeInTheDocument();
    expect(screen.getByText("Can latency be read?")).toBeInTheDocument();
    expect(screen.getByText("MISSING_TARGET")).toBeInTheDocument();
  });

  it("reports a coverage source error", async () => {
    mocks.fetchCoverage.mockRejectedValue(new Error("coverage down"));
    mocks.fetchCells.mockRejectedValue(new Error("cells down"));
    renderPage(<CoveragePage />);
    expect(await screen.findByText("Coverage unavailable")).toBeInTheDocument();
    expect(screen.getByText(/fabricating a ratio/)).toBeInTheDocument();
  });

  it("renders ranked focus findings and source health", async () => {
    mocks.fetchFocus.mockResolvedValue({
      allSourcesUnavailable: false,
      noFindings: false,
      sources: [{ id: "condition", label: "Condition", available: true, findingCount: 1 }],
      findings: [{ id: "f-1", title: "Read latency", source: "condition", message: "Verify the sensor", rationale: { rank: 1, cascadeStage: "condition", explanation: "first failing stage" } }],
    });
    renderPage(<FocusPage />);
    expect(await screen.findByText("Read latency")).toBeInTheDocument();
    expect(screen.getByText(/Condition/)).toBeInTheDocument();
    expect(screen.getByText(/first failing stage/)).toBeInTheDocument();
  });

  it("reports a focus source error", async () => {
    mocks.fetchFocus.mockRejectedValue(new Error("focus down"));
    renderPage(<FocusPage />);
    expect(await screen.findByText("Focus unavailable")).toBeInTheDocument();
    expect(screen.getByText(/could not read its finding sources/)).toBeInTheDocument();
  });

  it("names empty and unavailable reliability data explicitly", async () => {
    mocks.fetchCondition.mockResolvedValue({ readings: [], sources: [] });
    mocks.fetchTrust.mockResolvedValue({ trust: undefined });
    renderPage(<ConditionPage />);
    expect(await screen.findByText("Nothing to report")).toBeInTheDocument();
    expect(screen.getByText("No source report")).toBeInTheDocument();

    cleanup();
    mocks.fetchCoverage.mockResolvedValue({
      projections: [{ projection: 2, available: false, ratio: undefined, confidence: { level: 2, rationale: "not reachable" }, nowCount: 0, inReachCount: 0, missingCount: 0, unavailableReason: "source unavailable" }],
      integrityFindings: [],
    });
    mocks.fetchCells.mockResolvedValue({ cells: [] });
    renderPage(<CoveragePage />);
    expect(await screen.findByText("No dated coverage gaps.")).toBeInTheDocument();
    expect(screen.getByText("No setpoint-integrity findings were returned.")).toBeInTheDocument();

    cleanup();
    mocks.fetchFocus.mockResolvedValue({
      allSourcesUnavailable: true,
      noFindings: false,
      sources: [{ id: "condition", label: "Condition", available: false, findingCount: 0, reason: "offline" }],
      findings: [],
    });
    renderPage(<FocusPage />);
    expect(await screen.findByText("Nothing could be read")).toBeInTheDocument();
    expect(within(screen.getByTestId("focus-sources")).getByText(/offline/)).toBeInTheDocument();

    cleanup();
    mocks.fetchFocus.mockResolvedValue({ allSourcesUnavailable: false, noFindings: true, sources: [], findings: [] });
    renderPage(<FocusPage />);
    expect(await screen.findByText("Nothing to report")).toBeInTheDocument();
  });
});
