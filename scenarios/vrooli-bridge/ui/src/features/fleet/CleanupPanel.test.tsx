import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { CleanupStatus } from "@vrooli/proto-types/vrooli-bridge/v1/cleanup/cleanup_pb";
import { renderWithProviders } from "../../test-utils";

const { mutationState, nodesQuery, machinesQuery, cleanupQuery, startMutate, confirmMutate, sealPassphrase } = vi.hoisted(() => ({
  mutationState: { startError: null as Error | null, confirmError: null as Error | null },
  nodesQuery: vi.fn(),
  machinesQuery: vi.fn(),
  cleanupQuery: vi.fn(),
  startMutate: vi.fn(),
  confirmMutate: vi.fn(),
  sealPassphrase: vi.fn(),
}));

vi.mock("./queries", () => ({
  useNodesQuery: nodesQuery,
  useMachinesQuery: machinesQuery,
  useCleanupQuery: cleanupQuery,
  useStartCleanupMutation: () => ({ mutate: startMutate, isPending: false, error: mutationState.startError }),
  useConfirmCleanupMutation: () => ({ mutate: confirmMutate, isPending: false, error: mutationState.confirmError }),
}));

vi.mock("../../api/cleanup", () => ({ sealCleanupPassphrase: sealPassphrase }));

import { CleanupPanel } from "./CleanupPanel";

const node = { id: "node-1", name: "mini", nodeLineage: [], locators: [] };
const machine = {
  id: "machine-1",
  nodeLineage: [{ nodeId: "node-1", current: true }],
  locators: [{ kind: "hostname", value: "mini.local" }],
};

function operation(overrides: Record<string, unknown> = {}) {
  return {
    id: "cleanup-1",
    machineId: "machine-1",
    nodeId: "node-1",
    target: "mini.local",
    scope: "all",
    operatorId: "operator-1",
    planHash: "hash-1",
    status: CleanupStatus.PLANNED,
    planJson: new TextEncoder().encode(JSON.stringify({
      remove: [{ path: "/Users/example/.vrooli/bin/vrooli" }],
      keep: [],
      cannot_attribute: [{ path: "/Users/example/.ssh" }],
    })),
    receiptJson: new Uint8Array(),
    sealingPublicKey: new Uint8Array(32),
    ...overrides,
  };
}

describe("CleanupPanel", () => {
  beforeEach(() => {
    nodesQuery.mockReturnValue({ data: [node], error: null });
    machinesQuery.mockReturnValue({ data: [machine], error: null });
    cleanupQuery.mockImplementation((id: string | null) => ({ data: id ? { operation: operation() } : undefined, error: null }));
    startMutate.mockImplementation((_input: unknown, options?: { onSuccess?: (response: unknown) => void }) => {
      options?.onSuccess?.({ operation: { id: "cleanup-1" } });
    });
    sealPassphrase.mockResolvedValue(new Uint8Array([1, 2, 3]));
  });

  afterEach(() => {
    cleanup();
    mutationState.startError = null;
    mutationState.confirmError = null;
    vi.clearAllMocks();
  });

  it("renders a target and starts a read-only cleanup preview", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CleanupPanel />);

    await user.click(screen.getByRole("button", { name: "Preview cleanup" }));

    expect(startMutate).toHaveBeenCalledWith(
      { machineId: "machine-1", nodeId: "node-1", target: "mini.local", scope: "all" },
      expect.any(Object),
    );
    expect(await screen.findByText(/Operation cleanup-1/)).toBeInTheDocument();
    expect(screen.getByText(/Users\/example\/\.vrooli\/bin\/vrooli/)).toBeInTheDocument();
    expect(screen.getByText(/\.ssh/)).toBeInTheDocument();
  });

  it("seals the passphrase only after reviewing a planned frozen plan", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CleanupPanel />);
    await user.click(screen.getByRole("button", { name: "Preview cleanup" }));

    const passphrase = await screen.findByLabelText("Break-glass passphrase");
    await user.type(passphrase, "operator-secret");
    await user.click(screen.getByRole("button", { name: "Confirm and apply frozen plan" }));

    await waitFor(() => expect(sealPassphrase).toHaveBeenCalledWith(
      expect.any(Uint8Array),
      "operator-secret",
      ["vrooli-cleanup-context-v1", "machine-1", "node-1", "mini.local", "all", "hash-1", "cleanup-1", "operator-1"],
    ));
    expect(confirmMutate).toHaveBeenCalledWith({
      id: "cleanup-1",
      target: "mini.local",
      planHash: "hash-1",
      sealedPassphrase: new Uint8Array([1, 2, 3]),
      capability: new Uint8Array(),
      operatorId: "operator-1",
    });
    expect(screen.queryByDisplayValue("operator-secret")).not.toBeInTheDocument();
  });

  it("clears the plaintext and reports a sealing failure", async () => {
    const user = userEvent.setup();
    sealPassphrase.mockRejectedValue(new Error("node sealing key rejected"));
    renderWithProviders(<CleanupPanel />);
    await user.click(screen.getByRole("button", { name: "Preview cleanup" }));
    await user.type(await screen.findByLabelText("Break-glass passphrase"), "operator-secret");
    await user.click(screen.getByRole("button", { name: "Confirm and apply frozen plan" }));

    expect(await screen.findByText("node sealing key rejected")).toBeInTheDocument();
    expect(screen.queryByDisplayValue("operator-secret")).not.toBeInTheDocument();
    expect(confirmMutate).not.toHaveBeenCalled();
  });

  it("shows malformed plans and completed receipts without exposing a passphrase", async () => {
    const user = userEvent.setup();
    cleanupQuery.mockImplementation((id: string | null) => ({
      data: id ? { operation: operation({
        status: CleanupStatus.COMPLETED,
        planJson: new TextEncoder().encode("not-json"),
        receiptJson: new TextEncoder().encode('{"removed_count":1}'),
      }) } : undefined,
      error: null,
    }));
    renderWithProviders(<CleanupPanel />);
    await user.click(screen.getByRole("button", { name: "Preview cleanup" }));

    expect(await screen.findByText("not-json")).toBeInTheDocument();
    expect(screen.getByText(/removed_count/)).toBeInTheDocument();
    expect(screen.queryByLabelText("Break-glass passphrase")).not.toBeInTheDocument();
  });

  it("does not offer preview for a node without a current machine lineage", () => {
    nodesQuery.mockReturnValue({ data: [{ ...node, id: "orphan" }], error: null });
    machinesQuery.mockReturnValue({ data: [{ ...machine, nodeLineage: [{ nodeId: "other", current: false }] }], error: null });
    renderWithProviders(<CleanupPanel />);

    expect(screen.getByRole("button", { name: "Preview cleanup" })).toBeDisabled();
    expect(startMutate).not.toHaveBeenCalled();
  });

  it("surfaces typed start and confirm errors", async () => {
    const user = userEvent.setup();
    mutationState.startError = new Error("preview unavailable");
    mutationState.confirmError = new Error("confirmation rejected");
    renderWithProviders(<CleanupPanel />);

    await user.click(screen.getByRole("button", { name: "Preview cleanup" }));
    expect(await screen.findByText("preview unavailable")).toBeInTheDocument();
    expect(screen.getByText("confirmation rejected")).toBeInTheDocument();
  });
});
