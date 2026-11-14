import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { NodeStatus, type Node } from "../../api/nodes";
import { strings } from "../../consts/strings";
import { renderWithProviders } from "../../test-utils";

const { mutationState, updateMutate, revokeMutate, removeMutate } = vi.hoisted(() => ({
  mutationState: { updatePending: false, revokePending: false, removePending: false },
  updateMutate: vi.fn(),
  revokeMutate: vi.fn(),
  removeMutate: vi.fn(),
}));

vi.mock("./queries", () => ({
  useUpdateNodeMutation: () => ({ mutate: updateMutate, isPending: mutationState.updatePending }),
  useRevokeNodeMutation: () => ({ mutate: revokeMutate, isPending: mutationState.revokePending }),
  useRemoveNodeMutation: () => ({ mutate: removeMutate, isPending: mutationState.removePending }),
}));

import { NodeManagementPanel } from "./NodeManagementPanel";

const makeNode = (status: NodeStatus): Node => ({
  id: "node-1",
  name: "mini",
  endpoint: "https://mini.local",
  capabilities: ["scenario"],
  scopes: ["scenario test*"],
  revision: "rev-1",
  os: "",
  arch: "",
  status,
} as unknown as Node);

describe("NodeManagementPanel", () => {
  beforeEach(() => {
    vi.spyOn(window, "confirm").mockReturnValue(false);
  });

  afterEach(() => {
    mutationState.updatePending = false;
    mutationState.revokePending = false;
    mutationState.removePending = false;
    cleanup();
    vi.restoreAllMocks();
    vi.clearAllMocks();
  });

  it("edits typed metadata and keeps removal blocked until revocation", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    renderWithProviders(<NodeManagementPanel node={makeNode(NodeStatus.ONLINE)} onClose={onClose} />);

    expect(screen.getByText(`${strings.fleet.unknownValue} · ${strings.fleet.unknownValue}`)).toBeInTheDocument();
    expect(screen.getByText(strings.fleet.management.removeBlocked)).toBeInTheDocument();
    await user.clear(screen.getByTestId("fleet-node-management-name"));
    await user.type(screen.getByTestId("fleet-node-management-name"), "release-mini");
    await user.click(screen.getByRole("button", { name: strings.fleet.management.save }));
    expect(updateMutate).toHaveBeenCalledWith(expect.objectContaining({ name: "release-mini", capabilities: ["scenario"], scopes: ["scenario test*"] }));

    await user.click(screen.getByRole("button", { name: strings.fleet.management.close }));
    expect(onClose).toHaveBeenCalledOnce();
    await user.click(screen.getByRole("button", { name: new RegExp(strings.fleet.management.revoke) }));
    expect(revokeMutate).not.toHaveBeenCalled();
  });

  it("requires confirmation for both revocation and removal of a revoked node", async () => {
    const user = userEvent.setup();
    vi.spyOn(window, "confirm").mockReturnValue(true);
    renderWithProviders(<NodeManagementPanel node={makeNode(NodeStatus.REVOKED)} onClose={vi.fn()} />);

    expect(screen.getByRole("button", { name: strings.fleet.management.removeAction })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: new RegExp(strings.fleet.management.revoke) }));
    await user.click(screen.getByRole("button", { name: strings.fleet.management.removeAction }));

    await waitFor(() => expect(revokeMutate).toHaveBeenCalledWith("node-1", expect.any(Object)));
    expect(removeMutate).toHaveBeenCalledWith("node-1", expect.any(Object));
  });

  it("disables each destructive action while its mutation is pending", () => {
    mutationState.updatePending = true;
    mutationState.revokePending = true;
    mutationState.removePending = true;
    renderWithProviders(<NodeManagementPanel node={makeNode(NodeStatus.REVOKED)} onClose={vi.fn()} />);

    expect(screen.getByRole("button", { name: strings.fleet.management.saving })).toBeDisabled();
    expect(screen.getByRole("button", { name: new RegExp(strings.fleet.management.revoke) })).toBeDisabled();
    expect(screen.getByRole("button", { name: strings.fleet.management.removeAction })).toBeDisabled();
  });
});
