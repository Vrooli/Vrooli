import { createElement } from "react";
import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { TasksPage } from "../../src/pages/TasksPage";
import { renderWithProviders } from "../../src/test-utils";
import { makeTask } from "../testutil/tasks";

describe("TasksPage", () => {
  it("selects a task from the taskId query parameter", () => {
    renderWithProviders(
      createElement(TasksPage, {
        tasks: [
          makeTask({ id: "task-1", title: "First task" }),
          makeTask({ id: "task-2", title: "Deep-linked task" }),
        ],
        profiles: [],
        loading: false,
        error: null,
        onCreateTask: vi.fn(),
        onUpdateTask: vi.fn(),
        onCancelTask: vi.fn(),
        onDeleteTask: vi.fn(),
        onCreateRun: vi.fn(),
        onCreateProfile: vi.fn(),
        onRefresh: vi.fn(),
      }),
      { initialEntries: ["/tasks?taskId=task-2"] },
    );

    expect(screen.getByRole("heading", { name: "Deep-linked task" })).toBeInTheDocument();
  });
});
