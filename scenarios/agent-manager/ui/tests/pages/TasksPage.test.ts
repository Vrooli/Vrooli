import { createElement } from "react";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { TasksPage } from "../../src/pages/TasksPage";
import { renderWithProviders } from "../../src/test-utils";
import { RunMode, TaskStatus } from "../../src/types";
import { makeTask } from "../testutil/tasks";

vi.mock("../../src/components/TaskDetail", () => ({
  TaskDetail: ({ task, onRun, onEdit, onCancel, onDelete }: any) => createElement("section", {},
    createElement("h2", {}, task.title),
    createElement("button", { onClick: () => onRun(task) }, "Start selected task"),
    createElement("button", { onClick: () => onEdit(task) }, "Edit selected task"),
    createElement("button", { onClick: () => onCancel(task.id) }, "Cancel selected task"),
    createElement("button", { onClick: () => onDelete(task.id) }, "Delete selected task"),
  ),
}));

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

  it("creates a task from the operator form with its declared scope and project root", async () => {
    const createTask = vi.fn().mockResolvedValue(makeTask({ id: "created" }));
    renderWithProviders(createElement(TasksPage, {
      tasks: [], profiles: [], loading: false, error: null,
      onCreateTask: createTask, onUpdateTask: vi.fn(), onCancelTask: vi.fn(), onDeleteTask: vi.fn(),
      onCreateRun: vi.fn(), onCreateProfile: vi.fn(), onRefresh: vi.fn(),
    }));

    fireEvent.click(screen.getByRole("button", { name: /create task/i }));
    fireEvent.change(screen.getByLabelText("Title *"), { target: { value: "Investigate tool failures" } });
    fireEvent.change(screen.getByLabelText("Description"), { target: { value: "Capture useful evidence" } });
    fireEvent.change(screen.getByLabelText("Scope Path *"), { target: { value: "scenarios/agent-manager" } });
    fireEvent.change(screen.getByLabelText("Project Root"), { target: { value: "/workspace/vrooli" } });
    fireEvent.click(screen.getAllByRole("button", { name: "Create Task" })[1]!);

    await waitFor(() => expect(createTask).toHaveBeenCalledWith(expect.objectContaining({
      title: "Investigate tool failures", description: "Capture useful evidence",
      scopePath: "scenarios/agent-manager", projectRoot: "/workspace/vrooli",
    })));
    expect(screen.queryByRole("heading", { name: "Create New Task" })).not.toBeInTheDocument();
  });

  it("starts a selected task with the chosen profile and an existing sandbox", async () => {
    const createRun = vi.fn().mockResolvedValue({ id: "run-created" });
    renderWithProviders(createElement(TasksPage, {
      tasks: [makeTask({ id: "task-1", title: "Run this task" })],
      profiles: [{ id: "profile-1", name: "Reliable investigator", roleRef: "investigator" } as never], loading: false, error: null,
      onCreateTask: vi.fn(), onUpdateTask: vi.fn(), onCancelTask: vi.fn(), onDeleteTask: vi.fn(),
      onCreateRun: createRun, onCreateProfile: vi.fn(), onRefresh: vi.fn(),
    }), { initialEntries: ["/tasks?taskId=task-1"] });

    fireEvent.click(screen.getByRole("button", { name: "Start selected task" }));
    fireEvent.change(screen.getByLabelText("Agent Profile *"), { target: { value: "profile-1" } });
    fireEvent.change(screen.getByLabelText("Reuse Sandbox ID (optional)"), { target: { value: " sandbox-7 " } });
    fireEvent.click(screen.getByRole("button", { name: "Start Run" }));

    await waitFor(() => expect(createRun).toHaveBeenCalledWith(expect.objectContaining({
      taskId: "task-1", agentProfileId: "profile-1", existingSandboxId: "sandbox-7",
    })));
  });

  it("starts a custom run with the operator-selected limits, effort, and permission policy", async () => {
    const createRun = vi.fn().mockResolvedValue({ id: "custom-run" });
    const user = userEvent.setup();
    renderWithProviders(createElement(TasksPage, {
      tasks: [makeTask({ id: "task-1", title: "Run custom task" })],
      profiles: [], loading: false, error: null,
      onCreateTask: vi.fn(), onUpdateTask: vi.fn(), onCancelTask: vi.fn(), onDeleteTask: vi.fn(),
      onCreateRun: createRun, onCreateProfile: vi.fn(), onRefresh: vi.fn(),
    }), { initialEntries: ["/tasks?taskId=task-1"] });

    fireEvent.click(screen.getByRole("button", { name: "Start selected task" }));
    await user.click(screen.getByRole("tab", { name: /quick run/i }));
    await waitFor(() => expect(screen.getByLabelText("Max Turns")).toBeInTheDocument());
    fireEvent.change(screen.getByLabelText("Max Turns"), { target: { value: "27" } });
    fireEvent.change(screen.getByLabelText("Timeout (minutes)"), { target: { value: "45" } });
    fireEvent.change(screen.getByLabelText("Reasoning Effort"), { target: { value: "high" } });
    fireEvent.change(screen.getByLabelText("Run Mode *"), { target: { value: String(RunMode.IN_PLACE) } });
    fireEvent.click(screen.getByLabelText("Skip Permission Prompts"));
    fireEvent.change(screen.getByLabelText("Reuse Sandbox ID (optional)"), { target: { value: " sandbox-custom " } });
    fireEvent.click(screen.getByRole("button", { name: "Start Run" }));

    await waitFor(() => expect(createRun).toHaveBeenCalledWith(expect.objectContaining({
      taskId: "task-1", roleRef: "code.default", maxTurns: 27, timeoutMinutes: 45,
      effort: "high", runMode: RunMode.IN_PLACE, skipPermissionPrompt: false,
      existingSandboxId: "sandbox-custom",
    })));
  });

  it("creates and selects a profile from the run dialog with its explicit execution settings", async () => {
    const createProfile = vi.fn().mockResolvedValue({ id: "new-profile", name: "Focused", roleRef: "code.default" });
    renderWithProviders(createElement(TasksPage, {
      tasks: [makeTask({ id: "task-1", title: "Profile task" })],
      profiles: [], loading: false, error: null,
      onCreateTask: vi.fn(), onUpdateTask: vi.fn(), onCancelTask: vi.fn(), onDeleteTask: vi.fn(),
      onCreateRun: vi.fn(), onCreateProfile: createProfile, onRefresh: vi.fn(),
    }), { initialEntries: ["/tasks?taskId=task-1"] });

    fireEvent.click(screen.getByRole("button", { name: "Start selected task" }));
    fireEvent.click(screen.getByRole("button", { name: "Create Profile" }));
    fireEvent.change(screen.getByLabelText("Name *"), { target: { value: "Focused" } });
    fireEvent.change(screen.getByLabelText("Max Turns"), { target: { value: "55" } });
    fireEvent.change(screen.getByLabelText("Timeout (minutes)"), { target: { value: "70" } });
    fireEvent.change(screen.getByLabelText("Reasoning Effort"), { target: { value: "xhigh" } });
    fireEvent.click(screen.getByRole("button", { name: "Create & Select" }));

    await waitFor(() => expect(createProfile).toHaveBeenCalledWith(expect.objectContaining({
      name: "Focused", roleRef: "code.default", maxTurns: 55, timeoutMinutes: 70, effort: "xhigh",
    })));
    await waitFor(() => expect(screen.queryByRole("heading", { name: "Create New Profile" })).not.toBeInTheDocument());
  });

  it("sends the optional profile key, description, sandbox, and network settings", async () => {
    const createProfile = vi.fn().mockResolvedValue({ id: "configured-profile", name: "Configured", roleRef: "code.default" });
    renderWithProviders(createElement(TasksPage, {
      tasks: [makeTask({ id: "task-1", title: "Profile settings task" })],
      profiles: [], loading: false, error: null,
      onCreateTask: vi.fn(), onUpdateTask: vi.fn(), onCancelTask: vi.fn(), onDeleteTask: vi.fn(),
      onCreateRun: vi.fn(), onCreateProfile: createProfile, onRefresh: vi.fn(),
    }), { initialEntries: ["/tasks?taskId=task-1"] });

    fireEvent.click(screen.getByRole("button", { name: "Start selected task" }));
    fireEvent.click(screen.getByRole("button", { name: "Create Profile" }));
    fireEvent.change(screen.getByLabelText("Name *"), { target: { value: "Configured" } });
    fireEvent.change(screen.getByLabelText("Profile Key"), { target: { value: "configured-investigator" } });
    fireEvent.change(screen.getByLabelText("Description"), { target: { value: "Keeps investigations reproducible" } });
    fireEvent.change(screen.getByLabelText("Sandbox Mode"), { target: { value: "tracking" } });
    fireEvent.change(screen.getByLabelText("Network Access"), { target: { value: "none" } });
    fireEvent.click(screen.getByRole("button", { name: "Create & Select" }));

    await waitFor(() => expect(createProfile).toHaveBeenCalledWith(expect.objectContaining({
      profileKey: "configured-investigator", description: "Keeps investigations reproducible",
      sandboxMode: "tracking", networkAccess: "none",
    })));
  });

  it("edits and confirms cancellation and deletion for the selected task", async () => {
    const task = makeTask({ id: "task-1", title: "Original task", description: "old", scopePath: "src" });
    const update = vi.fn().mockResolvedValue(task); const cancel = vi.fn().mockResolvedValue(undefined); const remove = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal("confirm", vi.fn(() => true));
    renderWithProviders(createElement(TasksPage, {
      tasks: [task], profiles: [], loading: false, error: null,
      onCreateTask: vi.fn(), onUpdateTask: update, onCancelTask: cancel, onDeleteTask: remove,
      onCreateRun: vi.fn(), onCreateProfile: vi.fn(), onRefresh: vi.fn(),
    }), { initialEntries: ["/tasks?taskId=task-1"] });
    fireEvent.click(screen.getByRole("button", { name: "Edit selected task" }));
    fireEvent.change(screen.getByLabelText("Title *"), { target: { value: "Edited task" } });
    fireEvent.click(screen.getByRole("button", { name: "Save Changes" }));
    await waitFor(() => expect(update).toHaveBeenCalledWith("task-1", expect.objectContaining({ title: "Edited task" })));
    fireEvent.click(screen.getByRole("button", { name: "Cancel selected task" }));
    fireEvent.click(screen.getByRole("button", { name: "Delete selected task" }));
    await waitFor(() => expect(cancel).toHaveBeenCalledWith("task-1"));
    expect(remove).toHaveBeenCalledWith("task-1");
  });

  it("does not cancel or delete when the confirmation prompt is declined", () => {
    const task = makeTask({ id: "task-1", title: "Protected task" });
    const cancel = vi.fn();
    const remove = vi.fn();
    vi.stubGlobal("confirm", vi.fn(() => false));
    renderWithProviders(createElement(TasksPage, {
      tasks: [task], profiles: [], loading: false, error: null,
      onCreateTask: vi.fn(), onUpdateTask: vi.fn(), onCancelTask: cancel, onDeleteTask: remove,
      onCreateRun: vi.fn(), onCreateProfile: vi.fn(), onRefresh: vi.fn(),
    }), { initialEntries: ["/tasks?taskId=task-1"] });

    fireEvent.click(screen.getByRole("button", { name: "Cancel selected task" }));
    fireEvent.click(screen.getByRole("button", { name: "Delete selected task" }));
    expect(cancel).not.toHaveBeenCalled();
    expect(remove).not.toHaveBeenCalled();
  });

  it("filters task rows by search and status, and exposes the no-match guidance", async () => {
    renderWithProviders(createElement(TasksPage, {
      tasks: [
        makeTask({ id: "queued", title: "Queue investigation", status: TaskStatus.QUEUED }),
        makeTask({ id: "failed", title: "Repair report", description: "Broken aggregation", status: TaskStatus.FAILED }),
      ], profiles: [], loading: false, error: null,
      onCreateTask: vi.fn(), onUpdateTask: vi.fn(), onCancelTask: vi.fn(), onDeleteTask: vi.fn(), onCreateRun: vi.fn(), onCreateProfile: vi.fn(), onRefresh: vi.fn(),
    }));
    fireEvent.change(screen.getByPlaceholderText("Search tasks..."), { target: { value: "aggregation" } });
    expect(screen.getAllByText("Repair report").length).toBeGreaterThan(0);
    expect(screen.queryByText("Queue investigation")).not.toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText("Search tasks..."), { target: { value: "" } });
    fireEvent.click(screen.getByRole("button", { name: "Filter and sort options" }));
    fireEvent.click(screen.getByRole("button", { name: "Filter by status" }));
    fireEvent.click(screen.getByRole("button", { name: "Failed" }));
    expect(screen.getAllByText("Repair report").length).toBeGreaterThan(0);
    expect(screen.queryByText("Queue investigation")).not.toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText("Search tasks..."), { target: { value: "absent" } });
    expect(screen.getByText("No Matching Tasks")).toBeInTheDocument();
    expect(screen.getByText("Try adjusting your filters")).toBeInTheDocument();
  });

  it("keeps operator forms open after API failures and shows profile creation feedback", async () => {
    const createTask = vi.fn(async () => { throw new Error("task service unavailable"); });
    const createProfile = vi.fn(async () => { throw new Error("profile key already exists"); });
    renderWithProviders(createElement(TasksPage, {
      tasks: [makeTask({ id: "task-1", title: "Failure handling" })], profiles: [], loading: false, error: null,
      onCreateTask: createTask, onUpdateTask: vi.fn(), onCancelTask: vi.fn(), onDeleteTask: vi.fn(), onCreateRun: vi.fn(), onCreateProfile: createProfile, onRefresh: vi.fn(),
    }), { initialEntries: ["/tasks?taskId=task-1"] });
    fireEvent.click(screen.getByRole("button", { name: "New" }));
    fireEvent.change(screen.getByLabelText("Title *"), { target: { value: "Cannot create" } });
    fireEvent.click(screen.getByRole("button", { name: "Create Task" }));
    await waitFor(() => expect(createTask).toHaveBeenCalled());
    expect(screen.getByRole("heading", { name: "Create New Task" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    fireEvent.click(screen.getByRole("button", { name: "Start selected task" }));
    fireEvent.click(screen.getByRole("button", { name: "Create Profile" }));
    fireEvent.change(screen.getByLabelText("Name *"), { target: { value: "Duplicate" } });
    fireEvent.click(screen.getByRole("button", { name: "Create & Select" }));
    await waitFor(() => expect(screen.getByText("profile key already exists")).toBeInTheDocument());
    expect(screen.getByRole("heading", { name: "Create New Profile" })).toBeInTheDocument();
  });

  it("renders each task status label and calls refresh from the list toolbar", () => {
    const refresh = vi.fn();
    const tasks = [
      TaskStatus.QUEUED, TaskStatus.RUNNING, TaskStatus.NEEDS_REVIEW, TaskStatus.APPROVED,
      TaskStatus.REJECTED, TaskStatus.FAILED, TaskStatus.CANCELLED,
    ].map((status) => makeTask({ id: `task-${status}`, title: `Status ${status}`, status }));
    renderWithProviders(createElement(TasksPage, {
      tasks, profiles: [], loading: false, error: "Task API unavailable",
      onCreateTask: vi.fn(), onUpdateTask: vi.fn(), onCancelTask: vi.fn(), onDeleteTask: vi.fn(), onCreateRun: vi.fn(), onCreateProfile: vi.fn(), onRefresh: refresh,
    }));
    for (const label of ["queued", "running", "needs review", "approved", "rejected", "failed", "cancelled"]) {
      expect(screen.getByText(label)).toBeInTheDocument();
    }
    expect(screen.getByText("Task API unavailable")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "New" }).previousElementSibling as HTMLButtonElement);
    expect(refresh).toHaveBeenCalledTimes(1);
  });

  it("resets invalid custom run limits to their safe defaults and keeps the dialog open when starting fails", async () => {
    const createRun = vi.fn(async () => { throw new Error("runner unavailable"); });
    const user = userEvent.setup();
    renderWithProviders(createElement(TasksPage, {
      tasks: [makeTask({ id: "task-1", title: "Fallback limits" })], profiles: [], loading: false, error: null,
      onCreateTask: vi.fn(), onUpdateTask: vi.fn(), onCancelTask: vi.fn(), onDeleteTask: vi.fn(),
      onCreateRun: createRun, onCreateProfile: vi.fn(), onRefresh: vi.fn(),
    }), { initialEntries: ["/tasks?taskId=task-1"] });

    await user.click(screen.getByRole("button", { name: "Start selected task" }));
    await user.click(screen.getByRole("tab", { name: /quick run/i }));
    await user.clear(screen.getByLabelText("Max Turns"));
    await user.clear(screen.getByLabelText("Timeout (minutes)"));
    await user.click(screen.getByRole("button", { name: "Start Run" }));

    await waitFor(() => expect(createRun).toHaveBeenCalledWith(expect.objectContaining({ maxTurns: 100, timeoutMinutes: 30 })));
    expect(screen.getByRole("heading", { name: "Start Run" })).toBeInTheDocument();
  });

  it("sorts task rows by title and lets operators reset an active filter", async () => {
    const user = userEvent.setup();
    renderWithProviders(createElement(TasksPage, {
      tasks: [
        makeTask({ id: "z", title: "Zulu task", status: TaskStatus.FAILED }),
        makeTask({ id: "a", title: "Alpha task", status: TaskStatus.QUEUED }),
      ], profiles: [], loading: false, error: null,
      onCreateTask: vi.fn(), onUpdateTask: vi.fn(), onCancelTask: vi.fn(), onDeleteTask: vi.fn(), onCreateRun: vi.fn(), onCreateProfile: vi.fn(), onRefresh: vi.fn(),
    }));
    await user.click(screen.getByRole("button", { name: "Filter and sort options" }));
    await user.click(screen.getByRole("button", { name: "Sort" }));
    await user.click(screen.getByRole("button", { name: "Title A-Z" }));
    const titles = screen.getAllByText(/^(Alpha|Zulu) task$/).map((node) => node.textContent);
    expect(titles.indexOf("Alpha task")).toBeLessThan(titles.indexOf("Zulu task"));

    await user.click(screen.getByRole("button", { name: "Filter by status" }));
    await user.click(screen.getByRole("button", { name: "Failed" }));
    await user.click(screen.getByRole("button", { name: "Reset filters" }));
    expect(screen.getAllByText("Alpha task").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Zulu task").length).toBeGreaterThan(0);
  });

  it("preserves safe optional defaults when editing a task without optional metadata", () => {
    const task = makeTask({
      id: "minimal-task",
      title: "Minimal task",
      description: undefined,
      projectRoot: undefined,
      contextAttachments: undefined,
    });
    renderWithProviders(createElement(TasksPage, {
      tasks: [task], profiles: [], loading: false, error: null,
      onCreateTask: vi.fn(), onUpdateTask: vi.fn(), onCancelTask: vi.fn(), onDeleteTask: vi.fn(),
      onCreateRun: vi.fn(), onCreateProfile: vi.fn(), onRefresh: vi.fn(),
    }), { initialEntries: ["/tasks?taskId=minimal-task"] });

    fireEvent.click(screen.getByRole("button", { name: "Edit selected task" }));
    expect(screen.getByLabelText("Description")).toHaveValue("");
    expect(screen.getByLabelText("Project Root")).toHaveValue("");
  });

  it("does not start a profile-mode run until the operator selects a profile", async () => {
    const createRun = vi.fn();
    renderWithProviders(createElement(TasksPage, {
      tasks: [makeTask({ id: "task-1", title: "Profile required" })],
      profiles: [{ id: "profile-1", name: "Investigator", roleRef: "code.default" } as never], loading: false, error: null,
      onCreateTask: vi.fn(), onUpdateTask: vi.fn(), onCancelTask: vi.fn(), onDeleteTask: vi.fn(),
      onCreateRun: createRun, onCreateProfile: vi.fn(), onRefresh: vi.fn(),
    }), { initialEntries: ["/tasks?taskId=task-1"] });

    fireEvent.click(screen.getByRole("button", { name: "Start selected task" }));
    const start = screen.getByRole("button", { name: "Start Run" });
    expect(start).toBeDisabled();
    fireEvent.click(start);
    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(createRun).not.toHaveBeenCalled();
  });

  it("opens task creation from the empty-state guidance and resets a cancelled draft", async () => {
    const user = userEvent.setup();
    renderWithProviders(createElement(TasksPage, {
      tasks: [], profiles: [], loading: false, error: null,
      onCreateTask: vi.fn(), onUpdateTask: vi.fn(), onCancelTask: vi.fn(), onDeleteTask: vi.fn(),
      onCreateRun: vi.fn(), onCreateProfile: vi.fn(), onRefresh: vi.fn(),
    }));

    expect(screen.getByText("No Tasks")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Create Task" }));
    await user.type(screen.getByLabelText("Title *"), "Discard this draft");
    await user.click(screen.getByRole("button", { name: "Cancel" }));
    await user.click(screen.getByRole("button", { name: "Create Task" }));
    expect(screen.getByLabelText("Title *")).toHaveValue("");
    expect(screen.getByLabelText("Scope Path *")).toHaveValue(".");
  });

  it("retains edit and selected-task state when mutation services reject the request", async () => {
    const task = makeTask({ id: "task-1", title: "Mutable task", description: "before" });
    const update = vi.fn(async () => { throw new Error("update unavailable"); });
    const cancel = vi.fn(async () => { throw new Error("cancel unavailable"); });
    const remove = vi.fn(async () => { throw new Error("delete unavailable"); });
    vi.stubGlobal("confirm", vi.fn(() => true));
    renderWithProviders(createElement(TasksPage, {
      tasks: [task], profiles: [], loading: false, error: null,
      onCreateTask: vi.fn(), onUpdateTask: update, onCancelTask: cancel, onDeleteTask: remove,
      onCreateRun: vi.fn(), onCreateProfile: vi.fn(), onRefresh: vi.fn(),
    }), { initialEntries: ["/tasks?taskId=task-1"] });

    fireEvent.click(screen.getByRole("button", { name: "Edit selected task" }));
    fireEvent.change(screen.getByLabelText("Title *"), { target: { value: "Still editable" } });
    fireEvent.click(screen.getByRole("button", { name: "Save Changes" }));
    await waitFor(() => expect(update).toHaveBeenCalled());
    expect(screen.getByRole("heading", { name: "Edit Task" })).toBeInTheDocument();
    expect(screen.getByLabelText("Title *")).toHaveValue("Still editable");

    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    fireEvent.click(screen.getByRole("button", { name: "Cancel selected task" }));
    fireEvent.click(screen.getByRole("button", { name: "Delete selected task" }));
    await waitFor(() => expect(cancel).toHaveBeenCalledWith("task-1"));
    await waitFor(() => expect(remove).toHaveBeenCalledWith("task-1"));
    expect(screen.getByRole("heading", { name: "Mutable task" })).toBeInTheDocument();
  });

  it("uses safe numeric defaults when a new profile form is cleared before submission", async () => {
    const createProfile = vi.fn().mockResolvedValue({ id: "safe", name: "Safe", roleRef: "code.default" });
    const user = userEvent.setup();
    renderWithProviders(createElement(TasksPage, {
      tasks: [makeTask({ id: "task-1", title: "Profile defaults" })], profiles: [], loading: false, error: null,
      onCreateTask: vi.fn(), onUpdateTask: vi.fn(), onCancelTask: vi.fn(), onDeleteTask: vi.fn(),
      onCreateRun: vi.fn(), onCreateProfile: createProfile, onRefresh: vi.fn(),
    }), { initialEntries: ["/tasks?taskId=task-1"] });
    await user.click(screen.getByRole("button", { name: "Start selected task" }));
    await user.click(screen.getByRole("button", { name: "Create Profile" }));
    await user.type(screen.getByLabelText("Name *"), "Safe");
    await user.clear(screen.getByLabelText("Max Turns"));
    await user.clear(screen.getByLabelText("Timeout (minutes)"));
    await user.click(screen.getByRole("button", { name: "Create & Select" }));

    await waitFor(() => expect(createProfile).toHaveBeenCalledWith(expect.objectContaining({ maxTurns: 100, timeoutMinutes: 30 })));
  });
});
