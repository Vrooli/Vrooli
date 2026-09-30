import assert from "node:assert/strict";
import { act, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import type { ComponentProps } from "react";
import { test, vi } from "vitest";
import { QuickRunDialog } from "../../src/components/QuickRunDialog.js";
import { renderWithProviders } from "../../src/test-utils/index.js";
import { ExecutionMode, RunMode, type Run, type RunFormData, type Task, type TaskFormData } from "../../src/types.js";

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

  const renderResult = renderWithProviders(
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
    ...renderResult,
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
    roleRef: "code.default",
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

test("QuickRunDialog blocks protected interactive runs and sends permitted interactive configuration", async () => {
  const user = userEvent.setup();
  const calls = renderQuickRunDialog();
  await user.click(screen.getByRole("button", { name: "Agent" }));
  await user.selectOptions(screen.getByLabelText("Execution Mode *"), String(ExecutionMode.INTERACTIVE));
  assert.ok(screen.getByTestId("interactive-protected-warning"));
  await user.click(screen.getByRole("button", { name: /^Start Run$/i }));
  assert.equal(calls.onCreateTask.mock.calls.length, 0);
  assert.equal(screen.getAllByText(/Interactive execution is not available/).length, 2);
  await user.selectOptions(screen.getByLabelText("Run Mode *"), String(RunMode.IN_PLACE));
  await user.click(screen.getByRole("button", { name: /^Start Run$/i }));
  await waitFor(() => assert.equal(calls.onCreateRun.mock.calls.length, 1));
  assert.equal(calls.onCreateRun.mock.calls[0]?.[0].executionMode, ExecutionMode.INTERACTIVE);
  assert.equal(calls.onCreateRun.mock.calls[0]?.[0].runMode, RunMode.IN_PLACE);
});

test("QuickRunDialog uses a selected profile and shows its operational capabilities on review", async () => {
  const user = userEvent.setup();
  const profile = {
    id: "profile-1", name: "Reliable Investigator", roleRef: "investigator",
    description: "Investigates failure evidence", networkAccess: 2,
    features: { enableBrowser: true }, sandboxConfig: { mode: 2, manualReview: true },
    extraFlags: { codex: { flags: ["--full-auto", "--search"] } },
  } as never;
  const calls = renderQuickRunDialog({ profiles: [profile] });
  await user.click(screen.getByRole("button", { name: "Agent" }));
  await user.click(screen.getByRole("tab", { name: /Use Profile/ }));
  await user.selectOptions(screen.getByLabelText("Agent Profile *"), "profile-1");
  assert.ok(screen.getByText("Profile Details"));
  await user.click(screen.getByRole("button", { name: "Next" }));
  assert.ok(screen.getByText("Using Profile"));
  assert.ok(screen.getByText("Reliable Investigator"));
  assert.ok(screen.getByText("codex: --full-auto"));
  assert.ok(screen.getByText("codex: --search"));
  await user.click(screen.getByRole("button", { name: /^Start Run$/i }));
  await waitFor(() => assert.equal(calls.onCreateRun.mock.calls.length, 1));
  assert.deepEqual(calls.onCreateRun.mock.calls[0]?.[0], { taskId: "task-1", agentProfileId: "profile-1" });
});

test("QuickRunDialog carries custom browser, network, sandbox, and bounded-limit settings into the run", async () => {
  const user = userEvent.setup();
  const calls = renderQuickRunDialog();
  await user.click(screen.getByRole("button", { name: "Agent" }));
  await user.clear(screen.getByLabelText("Max Turns"));
  await user.type(screen.getByLabelText("Max Turns"), "10001");
  await user.tab();
  await user.clear(screen.getByLabelText("Timeout (minutes)"));
  await user.type(screen.getByLabelText("Timeout (minutes)"), "0");
  await user.tab();
  await user.selectOptions(screen.getByLabelText("Run Mode *"), String(RunMode.IN_PLACE));
  await user.click(screen.getByLabelText("Skip Permission Prompts"));
  await user.selectOptions(screen.getByLabelText("Network Access"), "full");
  await user.click(screen.getByLabelText(/Request browser automation/i));
  await user.type(screen.getByLabelText("Reuse Sandbox ID (optional)"), " sandbox-22 ");
  await user.click(screen.getByRole("button", { name: "Review" }));
  await waitFor(() => assert.ok(screen.getByText("Agent Configuration")));
  assert.ok(screen.getByText("Browser"));
  assert.ok(screen.getByText("In-place"));
  await user.click(screen.getByRole("button", { name: /^Start Run$/i }));

  await waitFor(() => assert.equal(calls.onCreateRun.mock.calls.length, 1));
  expect(calls.onCreateRun.mock.calls[0]?.[0]).toEqual(expect.objectContaining({
    taskId: "task-1", maxTurns: 10000, timeoutMinutes: 120,
    runMode: RunMode.IN_PLACE, skipPermissionPrompt: false, networkAccess: "full",
    features: { enableBrowser: true }, existingSandboxId: "sandbox-22",
  }));
});

test("QuickRunDialog supports keyboard submission and preserves a recoverable draft after task creation fails", async () => {
  const user = userEvent.setup();
  const onCreateTask = vi.fn(async () => {
    throw new Error("task store is unavailable");
  });
  const calls = renderQuickRunDialog({ onCreateTask });

  await user.type(screen.getByLabelText("What should the agent do?"), "Keep the investigation evidence intact");
  await user.click(screen.getByRole("button", { name: "Agent" }));
  await user.click(screen.getByRole("button", { name: "Task" }));
  assert.ok(screen.getByText("Keep the investigation evidence intact"));

  await user.keyboard("{Control>}{Enter}{/Control}");
  await waitFor(() => assert.equal(onCreateTask.mock.calls.length, 1));
  assert.equal(calls.onCreateRun.mock.calls.length, 0);
  assert.ok(screen.getByText("task store is unavailable"));
  assert.equal((screen.getByLabelText("What should the agent do?") as HTMLTextAreaElement).value, "Keep the investigation evidence intact");
  assert.equal(calls.onOpenChange.mock.calls.length, 0);
});

test("QuickRunDialog lets an operator close a draft without discarding it", async () => {
  const user = userEvent.setup();
  const calls = renderQuickRunDialog();
  await user.type(screen.getByLabelText("What should the agent do?"), "Resume only after reviewing receipts");
  await user.click(screen.getByRole("button", { name: "Close" }));

  assert.deepEqual(calls.onOpenChange.mock.calls, [[false]]);
  await waitFor(() => assert.match(
    window.localStorage.getItem("quick-run-task") ?? "",
    /Resume only after reviewing receipts/,
  ));
});

test("QuickRunDialog generates a bounded title and does not advance without required step data", async () => {
  const user = userEvent.setup();
  const calls = renderQuickRunDialog({ defaultProjectRoot: undefined });
  const description = "This investigation description is deliberately long enough to require a bounded automatic title for the run";

  await user.type(screen.getByLabelText("What should the agent do?"), description);
  await user.click(screen.getByRole("button", { name: "Next" }));
  expect(screen.getByLabelText("What should the agent do?")).toBeInTheDocument();

  await user.type(screen.getByLabelText("Project Root"), "/workspace/repo");
  await user.click(screen.getByRole("button", { name: "Next" }));
  expect(screen.getByText(/Load a valid role policy catalog/)).toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "Next" }));
  expect(screen.getByText("Agent Configuration")).toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: /^Start Run$/i }));

  await waitFor(() => assert.equal(calls.onCreateTask.mock.calls.length, 1));
  const truncated = description.slice(0, 60);
  assert.equal(calls.onCreateTask.mock.calls[0]?.[0].title, `${truncated.slice(0, truncated.lastIndexOf(" "))}...`);
});

test("QuickRunDialog keeps a profile run on configuration until a profile is selected", async () => {
  const user = userEvent.setup();
  renderQuickRunDialog({ profiles: [] });

  await user.click(screen.getByRole("button", { name: "Agent" }));
  await user.click(screen.getByRole("tab", { name: /Use Profile/ }));
  expect(screen.getByText(/No profiles available/)).toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "Next" }));
  expect(screen.getByText(/No profiles available/)).toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "Back" }));
  expect(screen.getByLabelText("What should the agent do?")).toBeInTheDocument();
});

test("QuickRunDialog fills an asynchronously discovered project root and generates a timestamp title for an empty request", async () => {
  const user = userEvent.setup();
  const calls = renderQuickRunDialog({ defaultProjectRoot: undefined });
  expect(screen.getByLabelText("Project Root")).toHaveValue("");

  act(() => {
    calls.rerender(createElement(QuickRunDialog, {
      open: true,
      onOpenChange: calls.onOpenChange,
      profiles: [],
      defaultProjectRoot: "/workspace/discovered",
      onCreateTask: calls.onCreateTask,
      onCreateRun: calls.onCreateRun,
      onRunCreated: calls.onRunCreated,
    }));
  });
  await waitFor(() => expect(screen.getByLabelText("Project Root")).toHaveValue("/workspace/discovered"));

  await user.click(screen.getByRole("button", { name: /^Start Run$/i }));
  await waitFor(() => assert.equal(calls.onCreateTask.mock.calls.length, 1));
  expect(calls.onCreateTask.mock.calls[0]?.[0]).toMatchObject({
    title: expect.stringMatching(/^Quick run \d{4}-\d{2}-\d{2} \d{2}:\d{2}$/),
    description: undefined,
    projectRoot: "/workspace/discovered",
  });
});

test("QuickRunDialog uploads an attached image before including its server receipt in the created task", async () => {
  const user = userEvent.setup();
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue({
    ok: true,
    json: async () => ({ id: "attachment-42", file_name: "evidence.png", content_type: "image/png", file_size: 5, storage_path: "attachments/evidence.png", url: "https://example.test/evidence.png" }),
  }));
  const calls = renderQuickRunDialog();
  const input = document.querySelector('input[type="file"]') as HTMLInputElement;
  await user.upload(input, new File(["proof"], "evidence.png", { type: "image/png" }));
  await waitFor(() => expect(fetch).toHaveBeenCalledTimes(1));
  await user.click(screen.getByRole("button", { name: /^Start Run$/i }));
  await waitFor(() => assert.equal(calls.onCreateTask.mock.calls.length, 1));
  expect(calls.onCreateTask.mock.calls[0]?.[0].contextAttachments).toEqual([
    { type: "image", attachment_id: "attachment-42", label: "Uploaded image" },
  ]);
});

test("QuickRunDialog supports a read-only scope and keeps minimal profile details intentionally sparse", async () => {
  const user = userEvent.setup();
  const profile = { id: "minimal", name: "Minimal profile", roleRef: "code.default" } as never;
  const calls = renderQuickRunDialog({ profiles: [profile] });

  await user.click(screen.getByRole("button", { name: "Remove ." }));
  await user.click(screen.getByRole("button", { name: "Agent" }));
  await user.click(screen.getByRole("tab", { name: /Use Profile/ }));
  await user.selectOptions(screen.getByLabelText("Agent Profile *"), "minimal");
  expect(screen.getByText("Profile Details")).toBeInTheDocument();
  expect(screen.queryByText(/Sandbox:/)).not.toBeInTheDocument();
  expect(screen.queryByText(/Manual Review/)).not.toBeInTheDocument();
  expect(screen.queryByText(/Net:/)).not.toBeInTheDocument();
  expect(screen.queryByText("Browser")).not.toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "Next" }));
  expect(screen.getByText("Read-only (no write access)")).toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: /^Start Run$/i }));

  await waitFor(() => assert.equal(calls.onCreateTask.mock.calls.length, 1));
  expect(calls.onCreateTask.mock.calls[0]?.[0].scopePath).toBe(".");
});

test("QuickRunDialog treats a selected profile removed by refresh as unavailable instead of rendering stale settings", async () => {
  const user = userEvent.setup();
  const profile = { id: "temporary", name: "Temporary profile", roleRef: "code.default" } as never;
  const calls = renderQuickRunDialog({ profiles: [profile] });
  await user.click(screen.getByRole("button", { name: "Agent" }));
  await user.click(screen.getByRole("tab", { name: /Use Profile/ }));
  await user.selectOptions(screen.getByLabelText("Agent Profile *"), "temporary");
  expect(screen.getByText("Profile Details")).toBeInTheDocument();

  calls.rerender(createElement(QuickRunDialog, {
    open: true,
    onOpenChange: calls.onOpenChange,
    profiles: [],
    defaultProjectRoot: "/workspace/repo",
    onCreateTask: calls.onCreateTask,
    onCreateRun: calls.onCreateRun,
    onRunCreated: calls.onRunCreated,
  }));
  expect(screen.getByText(/No profiles available/)).toBeInTheDocument();
  expect(screen.queryByText("Profile Details")).not.toBeInTheDocument();
});
