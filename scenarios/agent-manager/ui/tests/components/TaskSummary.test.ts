import assert from "node:assert/strict";
import { fireEvent, screen } from "@testing-library/react";
import { createElement } from "react";
import { test } from "vitest";
import { TaskSummary } from "../../src/components/RunDetailParts.js";
import { renderWithProviders } from "../../src/test-utils/index.js";
import { TaskStatus } from "../../src/types.js";

test("TaskSummary renders task evidence, scope, rich attachments, and attachment inspection", () => {
  renderWithProviders(createElement(TaskSummary, { task: {
    id: "task-1", title: "Investigate CLI failures", status: TaskStatus.NEEDS_REVIEW,
    description: "Review the **run report** before retrying.", scopePath: "api:cli", projectRoot: "/repo",
    contextAttachments: [
      { type: "file", key: "trace", label: "Trace file", path: "logs/run.json", tags: ["evidence", "failure"] },
      { type: "link", label: "Run dashboard", url: "https://example.test/run" },
      { type: "note", content: "Check timeout receipts" },
    ],
  } as never }));
  assert.ok(screen.getByText("Investigate CLI failures"));
  assert.ok(screen.getByText("needs review"));
  assert.ok(screen.getByText("api:cli")); assert.ok(screen.getByText("/repo"));
  assert.ok(screen.getByText("evidence")); assert.ok(screen.getByText("failure"));
  fireEvent.click(screen.getByText("Trace file"));
  assert.ok(screen.getByRole("dialog"));
  assert.equal(screen.getAllByText("logs/run.json").length, 2);
  fireEvent.click(screen.getAllByRole("button", { name: "Close" }).at(-1)!);
  assert.equal(screen.queryByRole("dialog"), null);
});

test("TaskSummary makes absent task description explicit", () => {
  renderWithProviders(createElement(TaskSummary, { task: { id: "task-2", title: "No details", status: TaskStatus.QUEUED, scopePath: "." } as never }));
  assert.ok(screen.getByText("No description provided"));
});
