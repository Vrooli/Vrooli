import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { DeploymentAnalysisReport, ScenarioSummary } from "../../types";
import { renderWithProviders } from "../../test-utils";
import { DeploymentDashboard } from "./DeploymentDashboard";

const mocks = vi.hoisted(() => ({ fetchDeploymentReport: vi.fn(), buildScenarioApiUrl: vi.fn((path: string) => `http://example.test${path}`) }));
vi.mock("../../api/client", () => mocks);

const alpha: ScenarioSummary = { name: "alpha", display_name: "Alpha Scenario" };
const beta: ScenarioSummary = { name: "beta", display_name: "Beta Scenario" };
function report(name: string, fitness = 0.8): DeploymentAnalysisReport {
  return { scenario: name, report_version: 1, generated_at: "2026-10-16T12:00:00Z", dependencies: [], aggregates: { desktop: { fitness_score: fitness, dependency_count: 0, blocking_dependencies: [], estimated_requirements: { ram_mb: 512, disk_mb: 1024, cpu_cores: 1 } } } };
}
function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => { resolve = done; });
  return { promise, resolve };
}
function props(scenarios: ScenarioSummary[]) {
  return { scenarios, loading: false, onRefresh: vi.fn(), onScanScenario: vi.fn(async (_name: string, _apply?: boolean) => {}), onSelectScenario: vi.fn() };
}

describe("deployment report lifecycle", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.fetchDeploymentReport.mockImplementation(async (name: string) => report(name));
  });

  it("ignores a superseded report even when the transport resolves after abort", async () => {
    const first = deferred<DeploymentAnalysisReport>();
    const second = deferred<DeploymentAnalysisReport>();
    mocks.fetchDeploymentReport.mockImplementation((name: string) => name === "alpha" ? first.promise : second.promise);
    const callbacks = props([alpha]);
    const view = renderWithProviders(<DeploymentDashboard {...callbacks} />);
    await waitFor(() => expect(mocks.fetchDeploymentReport).toHaveBeenCalledTimes(1));
    const firstSignal = mocks.fetchDeploymentReport.mock.calls[0]![1] as AbortSignal;
    view.rerender(<DeploymentDashboard {...callbacks} scenarios={[beta]} />);
    expect(firstSignal.aborted).toBe(true);
    await act(async () => second.resolve(report("beta")));
    expect(await screen.findByText("Beta Scenario")).toBeInTheDocument();
    await act(async () => first.resolve(report("alpha")));
    expect(screen.queryByText("Alpha Scenario")).not.toBeInTheDocument();
    expect(screen.getByText("Beta Scenario")).toBeInTheDocument();
  });

  it("clears loaded rows, selected details and busy state when the list becomes empty", async () => {
    const callbacks = props([alpha]);
    const view = renderWithProviders(<DeploymentDashboard {...callbacks} />);
    fireEvent.click(await screen.findByText("Alpha Scenario"));
    expect(screen.getByText("Deployment details: Alpha Scenario")).toBeInTheDocument();
    view.rerender(<DeploymentDashboard {...callbacks} scenarios={[]} />);
    expect(screen.queryByText("Alpha Scenario")).not.toBeInTheDocument();
    expect(screen.queryByText("Deployment details: Alpha Scenario")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Refresh" })).toBeEnabled();
  });

  it("clears an old connection error after a replacement report succeeds", async () => {
    mocks.fetchDeploymentReport.mockResolvedValueOnce(null);
    const callbacks = props([alpha]);
    const view = renderWithProviders(<DeploymentDashboard {...callbacks} />);
    expect(await screen.findByText("API unreachable")).toBeInTheDocument();
    expect(screen.getByText("vrooli scenario start scenario-dependency-analyzer")).toBeInTheDocument();
    view.rerender(<DeploymentDashboard {...callbacks} scenarios={[beta]} />);
    expect(await screen.findByText("Beta Scenario")).toBeInTheDocument();
    expect(screen.queryByText("API unreachable")).not.toBeInTheDocument();
  });

  it("loads at most five reports at once and starts the remaining batch only after settlement", async () => {
    const firstBatch = deferred<DeploymentAnalysisReport>();
    mocks.fetchDeploymentReport.mockReturnValue(firstBatch.promise);
    renderWithProviders(<DeploymentDashboard {...props(Array.from({ length: 6 }, (_, i) => ({ name: `s${i}`, display_name: `Scenario ${i}` })))} />);
    await waitFor(() => expect(mocks.fetchDeploymentReport).toHaveBeenCalledTimes(5));
    await act(async () => firstBatch.resolve(report("fixture")));
    await waitFor(() => expect(mocks.fetchDeploymentReport).toHaveBeenCalledTimes(6));
    expect(await screen.findByText("Scenario 5")).toBeInTheDocument();
  });

  it("aborts pending requests on unmount and never begins a later batch", async () => {
    const pending = deferred<DeploymentAnalysisReport>();
    mocks.fetchDeploymentReport.mockReturnValue(pending.promise);
    const view = renderWithProviders(<DeploymentDashboard {...props(Array.from({ length: 6 }, (_, i) => ({ name: `s${i}`, display_name: `Scenario ${i}` })))} />);
    await waitFor(() => expect(mocks.fetchDeploymentReport).toHaveBeenCalledTimes(5));
    const signals = mocks.fetchDeploymentReport.mock.calls.map((call) => call[1] as AbortSignal);
    view.unmount();
    expect(signals.every((signal) => signal.aborted)).toBe(true);
    await act(async () => pending.resolve(report("fixture")));
    expect(mocks.fetchDeploymentReport).toHaveBeenCalledTimes(5);
  });

  it("updates the selected detail from the latest scan instead of a stale status copy", async () => {
    mocks.fetchDeploymentReport.mockResolvedValueOnce(report("alpha", 0.2)).mockResolvedValueOnce(report("alpha", 0.9));
    const callbacks = props([alpha]);
    renderWithProviders(<DeploymentDashboard {...callbacks} />);
    fireEvent.click(await screen.findByText("Alpha Scenario"));
    expect(screen.getByText("20%")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Scan" }));
    await waitFor(() => expect(callbacks.onScanScenario).toHaveBeenCalledWith("alpha", false));
    expect(await screen.findByText("90%")).toBeInTheDocument();
    expect(screen.queryByText("20%")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Scan & Apply" }));
    await waitFor(() => expect(callbacks.onScanScenario).toHaveBeenCalledWith("alpha", true));
    expect(callbacks.onSelectScenario).toHaveBeenCalledWith("alpha");
  });

  it("keeps a failed scan retryable with actionable connection guidance", async () => {
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
    const callbacks = props([alpha]);
    callbacks.onScanScenario.mockRejectedValueOnce(new Error("scan failed"));
    renderWithProviders(<DeploymentDashboard {...callbacks} />);
    await screen.findByText("Alpha Scenario");
    fireEvent.click(screen.getByRole("button", { name: "Scan" }));
    expect(await screen.findByText(/Scan failed\. Check the API connection and retry/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Scan" })).toBeEnabled();
    consoleError.mockRestore();
  });

  it("filters by displayed or internal name, switches tier and selects a result with the keyboard", async () => {
    const callbacks = props([alpha, beta]);
    renderWithProviders(<DeploymentDashboard {...callbacks} />);
    await screen.findByText("Alpha Scenario");
    fireEvent.change(screen.getByRole("textbox", { name: "Search scenarios" }), { target: { value: "  BETA  " } });
    expect(screen.queryByText("Alpha Scenario")).not.toBeInTheDocument();
    const row = screen.getByRole("button", { name: /Beta Scenario/ });
    fireEvent.keyDown(row, { key: "Enter" });
    expect(callbacks.onSelectScenario).toHaveBeenCalledWith("beta");
    fireEvent.click(screen.getByRole("button", { name: "Tier 3 · Mobile" }));
    expect(screen.getByText("Deployment details: Beta Scenario")).toBeInTheDocument();
    expect(within(row).getByText("No tier score")).toBeInTheDocument();
  });
});
