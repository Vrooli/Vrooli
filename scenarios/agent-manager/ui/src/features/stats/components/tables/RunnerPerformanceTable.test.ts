import assert from "node:assert/strict";
import { getErrorNotice } from "../../../../test-utils/feedback";
import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import { test, vi } from "vitest";
import type { useQuery } from "@tanstack/react-query";
import { RunnerPerformanceTable } from "./RunnerPerformanceTable.js";
import type { RunnerBreakdownResponse } from "../../api/types.js";
import { useRunnerPerformance } from "../../hooks/useRunnerPerformance.js";
import { renderWithProviders } from "../../../../test-utils/index.js";
import { makeRunnerBreakdownResponse } from "../../../../test-utils/fixtures/stats.js";

vi.mock("../../hooks/useRunnerPerformance.js", () => ({
  useRunnerPerformance: vi.fn(),
}));

type QueryResult = ReturnType<typeof useQuery<RunnerBreakdownResponse, Error>>;

function queryResult(overrides: Partial<QueryResult>): QueryResult {
  return {
    data: undefined,
    isLoading: false,
    error: null,
    ...overrides,
  } as QueryResult;
}

function bodyRunnerNames(): string[] {
  return screen
    .getAllByRole("row")
    .slice(1)
    .map((row) => within(row).getAllByRole("cell")[0]?.textContent ?? "");
}

test("RunnerPerformanceTable renders formatted runner metrics sorted by run count", () => {
  vi.mocked(useRunnerPerformance).mockReturnValue(queryResult({
    data: makeRunnerBreakdownResponse(),
  }));

  renderWithProviders(createElement(RunnerPerformanceTable));

  assert.ok(screen.getByText("Runner Performance"));
  assert.deepEqual(bodyRunnerNames(), ["codex", "claude-code", "opencode"]);
  assert.ok(screen.getByText("88.9%"));
  assert.ok(screen.getByText("100.0%"));
  assert.ok(screen.getByText("50.0%"));
  assert.ok(screen.getByText("$8.25"));
  assert.ok(screen.getByText("1.5m"));
});

test("RunnerPerformanceTable toggles sort order from visible headers", async () => {
  const user = userEvent.setup();
  vi.mocked(useRunnerPerformance).mockReturnValue(queryResult({
    data: makeRunnerBreakdownResponse(),
  }));

  renderWithProviders(createElement(RunnerPerformanceTable));

  await user.click(screen.getByRole("button", { name: /runner/i }));
  assert.deepEqual(bodyRunnerNames(), ["opencode", "codex", "claude-code"]);

  await user.click(screen.getByRole("button", { name: /runner/i }));
  assert.deepEqual(bodyRunnerNames(), ["claude-code", "codex", "opencode"]);
});

test("RunnerPerformanceTable renders empty, loading, and error states", () => {
  vi.mocked(useRunnerPerformance).mockReturnValue(queryResult({
    data: makeRunnerBreakdownResponse({ runners: [] }),
  }));

  const { unmount } = renderWithProviders(createElement(RunnerPerformanceTable));
  assert.ok(screen.getByText("No runner data available"));

  unmount();
  vi.mocked(useRunnerPerformance).mockReturnValue(queryResult({
    isLoading: true,
  }));

  const loading = renderWithProviders(createElement(RunnerPerformanceTable));
  assert.equal(document.querySelectorAll(".animate-pulse").length, 1);

  loading.unmount();
  vi.mocked(useRunnerPerformance).mockReturnValue(queryResult({
    error: new Error("runner stats unavailable"),
  }));

  renderWithProviders(createElement(RunnerPerformanceTable));
  assert.ok(getErrorNotice("Runner performance", "runner stats unavailable"));
});

test("RunnerPerformanceTable headers sit over their own columns and announce the sort", async () => {
  const user = userEvent.setup();
  vi.mocked(useRunnerPerformance).mockReturnValue(queryResult({ data: makeRunnerBreakdownResponse() }));
  renderWithProviders(createElement(RunnerPerformanceTable));
  const headers = screen.getAllByRole("columnheader");
  assert.deepEqual(headers.map((header) => header.textContent?.trim()), ["Runner", "Runs", "Success", "Cost", "Tokens", "Avg Time"]);
  const [codexRow] = screen.getAllByRole("row").slice(1);
  const cells = within(codexRow as HTMLElement).getAllByRole("cell");
  assert.equal(cells[3]?.textContent, "$8.25", "the Cost header is over the cost cell");
  assert.equal(headers[1]?.getAttribute("aria-sort"), "descending");
  await user.click(within(headers[0] as HTMLElement).getByRole("button", { name: "Runner" }));
  assert.equal(headers[0]?.getAttribute("aria-sort"), "descending");
  assert.equal(headers[1]?.getAttribute("aria-sort"), "none");
});
