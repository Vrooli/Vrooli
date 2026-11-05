import { fireEvent, render, screen, waitFor } from "@/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { SpawnAgentButton } from "./SpawnAgentButton";

const mocks = vi.hoisted(() => ({
  createTask: vi.fn(),
  refresh: vi.fn(),
  investigation: {
    isAgentAvailable: true,
    isAgentLoading: false,
    isRunning: false,
    activeTaskId: null as string | null,
  },
}));

vi.mock("../../hooks/useInvestigation", () => ({
  usePipelineInvestigation: () => ({
    ...mocks.investigation,
    refresh: mocks.refresh,
  }),
}));
vi.mock("../../lib/api", () => ({ createTask: mocks.createTask }));

function triggerButton() {
  const buttons = screen.getAllByRole("button", { name: "Spawn Agent" });
  const button = buttons[buttons.length - 1];
  if (!button) throw new Error("expected task trigger button");
  fireEvent.click(button);
}

describe("SpawnAgentButton", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.investigation.isAgentAvailable = true;
    mocks.investigation.isAgentLoading = false;
    mocks.investigation.isRunning = false;
    mocks.investigation.activeTaskId = null;
  });

  it("creates an investigate task with the default logs effort and selected context", async () => {
    mocks.createTask.mockResolvedValue({ id: "task-1" });
    const onTaskStarted = vi.fn();
    render(
      <SpawnAgentButton pipelineId="pipe-1" onTaskStarted={onTaskStarted} />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Spawn Agent" }));
    expect(
      screen.getByRole("heading", { name: "Spawn Agent" }),
    ).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Additional notes for agent"), {
      target: { value: " investigate the package failure " },
    });
    triggerButton();

    await waitFor(() => {
      expect(mocks.createTask).toHaveBeenCalledWith(
        "pipe-1",
        expect.objectContaining({
          note: "investigate the package failure",
          focus: { harness: true, subject: true },
          includeContexts: [
            "task-metadata",
            "error-info",
            "pipeline-config",
            "pipeline-results",
            "build-logs",
            "generator-connection",
            "architecture-guide",
          ],
        }),
      );
    });
    expect(onTaskStarted).toHaveBeenCalledWith("task-1");
    expect(mocks.refresh).toHaveBeenCalled();
    expect(
      screen.queryByRole("heading", { name: "Spawn Agent" }),
    ).not.toBeInTheDocument();
  });

  it("switches to fix and sends selected permissions", async () => {
    mocks.createTask.mockResolvedValue({ id: "task-2" });
    render(<SpawnAgentButton pipelineId="pipe-2" />);
    fireEvent.click(screen.getByRole("button", { name: "Spawn Agent" }));
    fireEvent.click(screen.getByRole("button", { name: /Fix.*Apply fixes/ }));
    fireEvent.click(screen.getByRole("button", { name: /Permanent.*Code/ }));
    triggerButton();

    await waitFor(() => {
      expect(mocks.createTask).toHaveBeenCalledWith(
        "pipe-2",
        expect.objectContaining({
          permissions: { immediate: true, permanent: true, prevention: false },
        }),
      );
    });
  });

  it("shows a task creation failure without closing the dialog", async () => {
    mocks.createTask.mockRejectedValue(
      new Error("Agent service rejected the request"),
    );
    render(<SpawnAgentButton pipelineId="pipe-3" />);
    fireEvent.click(screen.getByRole("button", { name: "Spawn Agent" }));
    triggerButton();

    expect(
      await screen.findByText("Agent service rejected the request"),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "Spawn Agent" }),
    ).toBeInTheDocument();
  });

  it("surfaces unavailable agents and opens the already-running task", () => {
    mocks.investigation.isAgentAvailable = false;
    mocks.investigation.isRunning = true;
    mocks.investigation.activeTaskId = "active-task";
    const onTaskStarted = vi.fn();
    render(
      <SpawnAgentButton pipelineId="pipe-4" onTaskStarted={onTaskStarted} />,
    );

    fireEvent.click(screen.getByRole("button", { name: "View Task" }));
    expect(onTaskStarted).toHaveBeenCalledWith("active-task");
    expect(
      screen.queryByRole("heading", { name: "Spawn Agent" }),
    ).not.toBeInTheDocument();
  });
});
