import assert from "node:assert/strict";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import { test, vi } from "vitest";
import { TaskDetail } from "../../src/components/TaskDetail.js";
import { TaskStatus } from "../../src/types.js";
import { renderWithProviders } from "../../src/test-utils/index.js";
import { makeTask } from "../testutil/tasks.js";

vi.mock("../../src/components/markdown/index.js", () => ({ MarkdownRenderer: ({ content }: { content: string }) => createElement("span", null, content) }));
vi.mock("../../src/components/ContextAttachmentModal.js", () => ({ ContextAttachmentModal: ({ attachment }: { attachment: { label?: string } | null }) => attachment ? createElement("div", null, `modal ${attachment.label}`) : null }));

test("TaskDetail renders evidence and dispatches queued task actions", async () => {
  const user = userEvent.setup();
  const callbacks = { onEdit: vi.fn(), onRun: vi.fn(), onCancel: vi.fn(), onDelete: vi.fn() };
  const task = makeTask({ status: TaskStatus.QUEUED, contextAttachments: [{ type: "note", key: "facts", label: "Facts", content: "Use reports", tags: ["investigation"] }] as any });
  renderWithProviders(createElement(TaskDetail, { task, ...callbacks }));
  assert.ok(screen.getByText("Tighten the dashboard test coverage"));
  assert.ok(screen.getByText("Facts"));
  await user.click(screen.getByRole("button", { name: "Edit" }));
  await user.click(screen.getByRole("button", { name: "Run" }));
  await user.click(screen.getByRole("button", { name: "Cancel" }));
  await user.click(screen.getByText("Facts"));
  assert.deepEqual(callbacks.onEdit.mock.calls, [[task]]);
  assert.deepEqual(callbacks.onRun.mock.calls, [[task]]);
  assert.deepEqual(callbacks.onCancel.mock.calls, [[task.id]]);
  assert.ok(screen.getByText("modal Facts"));
});

test("TaskDetail falls back for missing description and offers deletion for cancelled tasks", async () => {
  const user = userEvent.setup();
  const onDelete = vi.fn();
  const task = makeTask({ status: TaskStatus.CANCELLED, description: "", projectRoot: "", contextAttachments: [] });
  renderWithProviders(createElement(TaskDetail, { task, onEdit: vi.fn(), onRun: vi.fn(), onCancel: vi.fn(), onDelete }));
  assert.ok(screen.getByText("No description provided"));
  await user.click(screen.getByRole("button", { name: "Delete" }));
  assert.deepEqual(onDelete.mock.calls, [[task.id]]);
});
