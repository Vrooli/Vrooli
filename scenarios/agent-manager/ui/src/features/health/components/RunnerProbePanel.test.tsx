import { fireEvent, screen } from "@testing-library/react";
import { createElement } from "react";
import { afterEach, expect, test, vi } from "vitest";
import { RunnerProbePanel } from "./RunnerProbePanel.js";
import { renderWithProviders } from "../../../test-utils/index.js";

const api = vi.hoisted(() => ({ health: { data: null as unknown }, probeRunner: vi.fn() }));
vi.mock("../../../hooks/useApi.js", () => ({ useHealth: () => api.health, probeRunner: api.probeRunner }));

const text = (value: string) => ({ kind: { case: "stringValue", value } });
const dependency = (status: string, error?: string) => ({ kind: { case: "objectValue", value: { fields: { status: text(status), ...(error ? { error: text(error) } : {}) } } } });

afterEach(() => { api.health.data = null; api.probeRunner.mockReset(); });

function showDependencies(dependencies: Record<string, unknown>) {
  api.health.data = { dependencies };
  return renderWithProviders(createElement(RunnerProbePanel));
}

test("shows nothing until the API reports sandbox or runner checks", () => {
  const { container } = showDependencies({});
  expect(container.firstChild).toBeNull();
});

test("names each runner by tool, marks availability and shows why one is unavailable", () => {
  showDependencies({ sandbox: dependency("healthy"), runner_claude_code: dependency("unhealthy", "binary missing"), runner_codex: dependency("healthy") });
  expect(screen.getByText("Workspace sandbox")).toBeTruthy();
  expect(screen.getByText("binary missing")).toBeTruthy();
  expect(screen.getAllByText("Available")).toHaveLength(2);
  expect(screen.getAllByText("Unavailable")).toHaveLength(1);
  expect(screen.getByText("Codex")).toBeTruthy();
  expect(screen.queryByTestId("runner-probe-sandbox")?.querySelector("button")).toBeNull();
});

test("a successful probe reports its latency", async () => {
  api.probeRunner.mockResolvedValue({ success: true, latencyMs: 1500n, error: "" });
  showDependencies({ runner_codex: dependency("healthy") });
  fireEvent.click(screen.getByRole("button", { name: /probe/i }));
  expect(await screen.findByText("Probe succeeded")).toBeTruthy();
  expect(screen.getByText(/1\.5 s/)).toBeTruthy();
});

test("a failed probe keeps the runner's diagnostic, and a thrown request is reported too", async () => {
  api.probeRunner.mockResolvedValueOnce({ success: false, latencyMs: 20n, error: "model not found" });
  showDependencies({ runner_codex: dependency("healthy") });
  fireEvent.click(screen.getByRole("button", { name: /probe/i }));
  expect(await screen.findByText("Probe failed")).toBeTruthy();
  expect(screen.getByText("model not found")).toBeTruthy();
  api.probeRunner.mockRejectedValueOnce(new Error("runner endpoint timed out"));
  fireEvent.click(screen.getByRole("button", { name: /probe/i }));
  expect(await screen.findByText("runner endpoint timed out")).toBeTruthy();
});
