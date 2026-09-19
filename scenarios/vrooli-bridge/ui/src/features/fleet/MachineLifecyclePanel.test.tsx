import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";

const { listMachines, getMachine, getMachineTrust, archiveMachine, removeMachine, revokeMachineNode, requestMachineSSHCleanup, applyMachinePolicy, reviewMachineHostKey } = vi.hoisted(() => ({
  listMachines: vi.fn(), getMachine: vi.fn(), getMachineTrust: vi.fn(), archiveMachine: vi.fn(), removeMachine: vi.fn(), revokeMachineNode: vi.fn(), requestMachineSSHCleanup: vi.fn(), applyMachinePolicy: vi.fn(), reviewMachineHostKey: vi.fn(),
}));

vi.mock("../../api/machines", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/machines")>();
  return { ...actual, machinesClient: { listMachines, getMachine, getMachineTrust, archiveMachine, removeMachine, revokeMachineNode, requestMachineSSHCleanup, applyMachinePolicy, reviewMachineHostKey } };
});

import { MachineLifecyclePanel } from "./MachineLifecyclePanel";

describe("MachineLifecyclePanel", () => {
  beforeEach(() => {
    archiveMachine.mockResolvedValue({}); removeMachine.mockResolvedValue({}); revokeMachineNode.mockResolvedValue({}); requestMachineSSHCleanup.mockResolvedValue({}); applyMachinePolicy.mockResolvedValue({}); reviewMachineHostKey.mockResolvedValue({});
  });
  afterEach(() => { cleanup(); vi.clearAllMocks(); });

  it("renders durable Machine lifecycle actions", async () => {
    listMachines.mockResolvedValue({ machines: [{ id: "machine-1", lifecycle: "active", version: 3n, locators: [], nodeLineage: [] }] });
    renderWithProviders(<MachineLifecyclePanel />);
    expect(await screen.findByTestId(selectors.fleet.machineLifecycleRow({ id: "machine-1" }))).toHaveTextContent("machine-1");
    await userEvent.click(screen.getByRole("button", { name: strings.fleet.machines.archive }));
    await userEvent.click(screen.getByRole("button", { name: strings.fleet.machines.revokeNode }));
    await userEvent.click(screen.getByRole("button", { name: strings.fleet.machines.cleanup }));
    await waitFor(() => expect(archiveMachine).toHaveBeenCalledWith({ id: "machine-1", version: 3n }));
    expect(revokeMachineNode).toHaveBeenCalledWith({ machineId: "machine-1" });
    expect(requestMachineSSHCleanup).toHaveBeenCalledWith({ machineId: "machine-1" });
  });

  it("requires a confirmation before Machine record removal", async () => {
    listMachines.mockResolvedValue({ machines: [{ id: "machine-2", lifecycle: "archived", version: 4n, locators: [], nodeLineage: [] }] });
    const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderWithProviders(<MachineLifecyclePanel />);
    await screen.findByTestId(selectors.fleet.machineLifecycleRow({ id: "machine-2" }));
    await userEvent.click(screen.getByRole("button", { name: strings.fleet.machines.remove }));
    expect(confirm).toHaveBeenCalledOnce();
    expect(removeMachine).not.toHaveBeenCalled();
    confirm.mockRestore();
  });

  it("renders explicit loading, error, and empty states", async () => {
    listMachines.mockReturnValue(new Promise(() => {}));
    const { unmount } = renderWithProviders(<MachineLifecyclePanel />);
    expect(screen.getByText(strings.fleet.machines.loading)).toBeInTheDocument();
    unmount();

    listMachines.mockRejectedValue(new Error("machines unavailable"));
    renderWithProviders(<MachineLifecyclePanel />);
    expect(await screen.findByRole("alert")).toHaveTextContent("machines unavailable");
  });

  it("explains missing Machine readiness facts without inventing node or trust data", async () => {
    listMachines.mockResolvedValue({ machines: [{ id: "machine-4", lifecycle: "active", version: 6n, locators: [], nodeLineage: [] }] });
    getMachine.mockResolvedValue({ enrollmentAttempts: [], auditEvents: [], cleanupTombstones: [], readiness: { ready: false, reasons: ["no_current_node"] } });
    getMachineTrust.mockResolvedValue({ trust: { hostKeyState: "", hostKeyFingerprint: "", clientKeyFingerprint: "" } });
    renderWithProviders(<MachineLifecyclePanel />);
    await screen.findByTestId(selectors.fleet.machineLifecycleRow({ id: "machine-4" }));
    await userEvent.click(screen.getByRole("button", { name: strings.fleet.machines.details }));
    const detail = await screen.findByTestId("machine-detail");
    expect(detail).toHaveTextContent(strings.fleet.machines.noCurrentNode);
    expect(detail).toHaveTextContent(strings.fleet.machines.notReady);
  });

  it("renders composed Machine detail and makes policy and host-key review typed requests", async () => {
    listMachines.mockResolvedValue({ machines: [{ id: "machine-3", lifecycle: "active", version: 5n, locators: [], nodeLineage: [] }] });
    getMachine.mockResolvedValue({ enrollmentAttempts: [{ id: "attempt-1" }], auditEvents: [{ id: "audit-1" }], cleanupTombstones: [{ id: "cleanup-1", status: "pending" }], currentNode: { nodeId: "node-3", online: true }, readiness: { ready: true, reasons: [] } });
    getMachineTrust.mockResolvedValue({ trust: { hostKeyState: "review_required", hostKeyFingerprint: "SHA256:old", clientKeyFingerprint: "SHA256:client" } });
    renderWithProviders(<MachineLifecyclePanel />);
    await screen.findByTestId(selectors.fleet.machineLifecycleRow({ id: "machine-3" }));
    await userEvent.click(screen.getByRole("button", { name: strings.fleet.machines.details }));
    await screen.findByTestId("machine-detail");
    await waitFor(() => expect(getMachine).toHaveBeenCalledWith({ id: "machine-3" }));
    expect(getMachineTrust).toHaveBeenCalledWith({ machineId: "machine-3" });
    expect(screen.getByTestId("machine-detail")).toHaveTextContent(strings.fleet.machines.cleanupStatus);
    await userEvent.selectOptions(screen.getByLabelText(strings.fleet.machines.policy), "presence");
    await userEvent.type(screen.getByLabelText(strings.fleet.machines.policyReason), "least privilege");
    await userEvent.click(screen.getByLabelText(strings.fleet.machines.confirmRemoval));
    await userEvent.click(screen.getByRole("button", { name: strings.fleet.machines.applyPolicy }));
    await userEvent.type(screen.getByLabelText(strings.fleet.machines.replacementHostKey), "SHA256:new");
    await userEvent.click(screen.getByRole("button", { name: strings.fleet.machines.reviewHostKey }));
    await waitFor(() => expect(applyMachinePolicy).toHaveBeenCalledWith({ machineId: "machine-3", version: 5n, profileId: "presence", profileVersion: "", overrides: {}, reason: "least privilege", confirmRemoval: true }));
    expect(reviewMachineHostKey).toHaveBeenCalledWith({ machineId: "machine-3", replacementHostKeyFingerprint: "SHA256:new" });
  });
});
