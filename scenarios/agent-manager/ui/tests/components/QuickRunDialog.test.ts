import assert from "node:assert/strict";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import type { ComponentProps } from "react";
import { test, vi } from "vitest";
import { QuickRunDialog } from "../../src/components/QuickRunDialog.js";
import { renderWithProviders } from "../../src/test-utils/index.js";
import { RunMode, RunnerType, type Run, type RunFormData, type Task, type TaskFormData } from "../../src/types.js";

const persistedEnvelope = <T,>(data: T) =>
  JSON.stringify({
    version: 1,
    data,
    updatedAt: Date.UTC(2026, 4, 1),
  });

function renderQuickRunDialog(overrides: Partial<ComponentProps<typeof QuickRunDialog>> = {}) {
  const onCreateTask = vi.fn(async (task: TaskFormData) => ({
    id: "task-1",
    title: task.title,
    description: task.description,
    scopePath: task.scopePath,
    projectRoot: task.projectRoot,
  } as Task));
  const onCreateRun = vi.fn(async (run: RunFormData) => ({
    id: "run-1",
    taskId: run.taskId,
    status: 1,
  } as Run));
  const onOpenChange = vi.fn();
  const onRunCreated = vi.fn();

  renderWithProviders(
    createElement(QuickRunDialog, {
      open: true,
      onOpenChange,
      profiles: [],
      defaultProjectRoot: "/workspace/repo",
      onCreateTask,
      onCreateRun,
      onRunCreated,
      ...overrides,
    }),
  );

  return {
    onCreateTask,
    onCreateRun,
    onOpenChange,
    onRunCreated,
  };
}

test("QuickRunDialog starts a default custom run and clears persisted draft state after success", async () => {
  const user = userEvent.setup();
  const calls = renderQuickRunDialog();

  await user.clear(screen.getByLabelText("What should the agent do?"));
  await user.type(screen.getByLabelText("What should the agent do?"), "Refactor the stats widgets");
  await user.clear(screen.getByLabelText("Project Root"));
  await user.type(screen.getByLabelText("Project Root"), "/workspace/repo");
  await user.click(screen.getByRole("button", { name: /^Start Run$/i }));

  await waitFor(() => {
    assert.equal(calls.onCreateRun.mock.calls.length, 1);
  });

  assert.deepEqual(calls.onCreateTask.mock.calls[0]?.[0], {
    title: "Refactor the stats widgets",
    description: "Refactor the stats widgets",
    scopePath: ".",
    projectRoot: "/workspace/repo",
  });
  assert.deepEqual(calls.onCreateRun.mock.calls[0]?.[0], {
    taskId: "task-1",
    runnerType: RunnerType.CLAUDE_CODE,
    maxTurns: 500,
    timeoutMinutes: 120,
    runMode: RunMode.SANDBOXED,
    skipPermissionPrompt: true,
    networkAccess: "localhost",
  });
  assert.equal(calls.onOpenChange.mock.calls[0]?.[0], false);
  assert.equal(calls.onRunCreated.mock.calls[0]?.[0].id, "run-1");
  assert.equal(window.localStorage.getItem("quick-run-task"), null);
  assert.equal(window.localStorage.getItem("quick-run-config"), null);
  assert.equal(window.localStorage.getItem("quick-run-scope"), null);
});

test("QuickRunDialog restores persisted task draft fields before submit", async () => {
  const user = userEvent.setup();
  window.localStorage.setItem(
    "quick-run-task",
    persistedEnvelope({
      title: "Persisted title",
      description: "Persisted description",
      projectRoot: "/workspace/persisted",
    }),
  );

  const calls = renderQuickRunDialog();

  assert.equal(screen.getByLabelText("Task Title").getAttribute("value"), "Persisted title");
  assert.equal(screen.getByLabelText("Project Root").getAttribute("value"), "/workspace/persisted");
  assert.equal(screen.getByLabelText("What should the agent do?").textContent, "Persisted description");

  await user.click(screen.getByRole("button", { name: /^Start Run$/i }));

  await waitFor(() => {
    assert.equal(calls.onCreateTask.mock.calls.length, 1);
  });
  assert.deepEqual(calls.onCreateTask.mock.calls[0]?.[0], {
    title: "Persisted title",
    description: "Persisted description",
    scopePath: ".",
    projectRoot: "/workspace/persisted",
  });
});
