import assert from "node:assert/strict";
import { fireEvent, screen } from "@testing-library/react";
import { createElement } from "react";
import { afterEach, test, vi } from "vitest";
import { FindingsPage } from "./FindingsPage.js";
import { renderWithProviders } from "../test-utils/index.js";
import { getErrorNotice } from "../test-utils/feedback.js";

const { useRecurringFindings } = vi.hoisted(() => ({ useRecurringFindings: vi.fn() }));

vi.mock("../hooks/useApi.js", () => ({ useRecurringFindings }));

afterEach(() => vi.resetAllMocks());

test("FindingsPage renders empty, loading, and failed finding states", () => {
  const refetch = vi.fn();
  useRecurringFindings.mockReturnValue({ data: [], loading: false, error: "", refetch });
  const { rerender } = renderWithProviders(createElement(FindingsPage));
  assert.ok(screen.getByText("No findings yet"));
  fireEvent.click(screen.getByRole("button", { name: "Refresh findings" }));
  assert.equal(refetch.mock.calls.length, 1);

  useRecurringFindings.mockReturnValue({ data: null, loading: true, error: null, refetch });
  rerender(createElement(FindingsPage));
  assert.equal(screen.queryByText("No findings yet"), null, "loading is not an empty result");
  assert.ok(screen.getByText(/Loading findings/));

  useRecurringFindings.mockReturnValue({ data: null, loading: false, error: "Request failed: 503", refetch });
  rerender(createElement(FindingsPage));
  const notice = getErrorNotice("findings", "Request failed: 503");
  assert.match(notice.textContent ?? "", /Agent Manager is unreachable/);
  assert.equal(screen.queryByText("No findings yet"), null, "an unreachable API is not an empty result");
});

test("FindingsPage renders durable finding evidence and a run drill-down", () => {
  useRecurringFindings.mockReturnValue({
    loading: false, error: "", refetch: vi.fn(),
    data: [{ id: "finding-1", runId: "run-1", recommendation: "Use the report first", category: "Efficiency/Friction", severity: "Major", occurrences: 3, decision: "accepted", evidence: "Two repeated bash calls", targetPath: "cli/cmd_run_inspection.go" }],
  });
  renderWithProviders(createElement(FindingsPage));
  assert.ok(screen.getByText("Use the report first"));
  assert.match(screen.getByText(/Efficiency\/Friction/).textContent ?? "", /occurrences: 3/);
  assert.equal(screen.getByText("Open source run").getAttribute("href"), "/runs/run-1");
  assert.ok(screen.getByText("Two repeated bash calls"));
});
