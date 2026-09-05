import "@testing-library/jest-dom";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { vi } from "vitest";
import { renderWithProviders } from "./test-utils/renderWithProviders";

const state = vi.hoisted(() => ({ enabled: true, available: true, running: false, modal: null as string | null, task: { id: "task-1" } }));
const createTask = vi.hoisted(() => vi.fn());

vi.mock("./lib/api", () => ({ createTask }));
vi.mock("./hooks/useInvestigation", () => ({
  useDeploymentInvestigation: () => ({ isAgentEnabled: state.enabled, isAgentAvailable: state.available, isAgentLoading: false, isRunning: state.running, activeInvestigationId: "active-1", refresh: vi.fn() }),
}));
vi.mock("./hooks/useDeploymentUrl", async () => {
  const React = await vi.importActual<typeof import("react")>("react");
  return { useDeploymentUrl: () => {
    const [, refresh] = React.useState(0);
    const update = (fn: () => void) => { fn(); refresh((n) => n + 1); };
    return {
      state: { modal: state.modal, modalParams: {}, deploymentId: "dep-1", tab: "overview", subtab: "processes" },
      openModal: (modal: string) => update(() => { state.modal = modal; }),
      closeModal: () => update(() => { state.modal = null; }),
    };
  } };
});

import { SpawnAgentButton } from "./components/wizard/SpawnAgentButton";

describe("spawn-agent workflow", () => {
  beforeEach(() => {
    state.enabled = true; state.available = true; state.running = false; state.modal = null;
    createTask.mockReset(); createTask.mockResolvedValue({ task: state.task });
  });

  it("handles disabled, unavailable, investigate, advanced contexts, and successful creation", async () => {
    state.enabled = false;
    const { rerender } = renderWithProviders(<SpawnAgentButton deploymentId="dep-1" />);
    expect(screen.getByTitle(/not enabled/)).toBeDisabled();
    state.enabled = true; state.available = false;
    rerender(<SpawnAgentButton deploymentId="dep-1" />);
    fireEvent.click(screen.getByRole("button", { name: /Spawn Agent/ }));
    expect(screen.getByText("Agent Manager Unavailable")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Advanced Options/ }));
    fireEvent.click(screen.getByLabelText(/Deployment Manifest/));
    fireEvent.change(screen.getByPlaceholderText(/This started/), { target: { value: "check logs" } });
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    state.available = true;
    rerender(<SpawnAgentButton deploymentId="dep-1" />);
    fireEvent.click(screen.getByRole("button", { name: "Spawn Agent" }));
    fireEvent.click(screen.getByText("Trace"));
    const spawnButtons = screen.getAllByRole("button", { name: "Spawn Agent" });
    const spawnButton = spawnButtons[spawnButtons.length - 1];
    if (!spawnButton) throw new Error("expected spawn action button");
    fireEvent.click(spawnButton);
    await waitFor(() => expect(createTask).toHaveBeenCalled());
  });

  it("validates fix permissions and handles API failures and running tasks", async () => {
    const onStarted = vi.fn();
    const { rerender } = renderWithProviders(<SpawnAgentButton deploymentId="dep-1" onTaskStarted={onStarted} />);
    fireEvent.click(screen.getByRole("button", { name: /Spawn Agent/ }));
    fireEvent.click(screen.getByText("Fix"));
    fireEvent.click(screen.getByText("Immediate"));
    expect(screen.getByText("Select at least one permission")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Immediate"));
    fireEvent.click(screen.getByText("Permanent"));
    fireEvent.click(screen.getByText("Prevention"));
    fireEvent.click(screen.getByText("Harness"));
    fireEvent.click(screen.getByText("Subject"));
    expect(screen.getByText("Select at least one focus area")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Harness"));
    createTask.mockRejectedValueOnce(new Error("creation failed"));
    const spawnButtons = screen.getAllByRole("button", { name: "Spawn Agent" });
    const spawnButton = spawnButtons[spawnButtons.length - 1];
    if (!spawnButton) throw new Error("expected spawn action button");
    fireEvent.click(spawnButton);
    expect(await screen.findByText("creation failed")).toBeInTheDocument();
    state.running = true;
    rerender(<SpawnAgentButton deploymentId="dep-1" onTaskStarted={onStarted} />);
    fireEvent.click(screen.getByRole("button", { name: "View Task" }));
    expect(onStarted).toHaveBeenCalledWith("active-1");
  });
});
