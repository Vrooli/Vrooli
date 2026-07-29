import assert from "node:assert/strict";
import { fireEvent, screen } from "@testing-library/react";
import { createElement } from "react";
import { test, vi } from "vitest";
import { DashboardPage } from "../../src/pages/DashboardPage.js";
import { renderWithProviders } from "../../src/test-utils/index.js";
import { RunStatus } from "../../src/types.js";
import { makeRun } from "../testutil/runs.js";

test("DashboardPage presents run counts, review navigation, task names, and refresh", async () => {
  const refresh = vi.fn();
  const navigate = vi.fn();
  const getTask = vi.fn(async () => ({ id: "task-1", title: "Review task" }));
  renderWithProviders(createElement(DashboardPage, {
    health: null,
    statusCounts: { running: 1, needsReview: 1, total: 2 },
    runs: [makeRun({ id: "review", taskId: "task-1", status: RunStatus.NEEDS_REVIEW, changedFiles: 2 }), makeRun({ id: "running", status: RunStatus.RUNNING })],
    onRefresh: refresh, onNavigateToRun: navigate, onGetTask: getTask,
  }));
  assert.ok(screen.getByText("Awaiting Review (1)"));
  const taskLabels = await screen.findAllByText("Review task");
  assert.equal(taskLabels.length, 2);
  fireEvent.click(taskLabels[0]?.closest('[role="button"]') as Element);
  assert.deepEqual(navigate.mock.calls[0], ["review", "diff"]);
  fireEvent.click(screen.getByTitle("Refresh"));
  assert.equal(refresh.mock.calls.length, 1);
});
