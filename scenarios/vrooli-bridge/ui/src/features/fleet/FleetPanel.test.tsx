import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { Code, ConnectError } from "@connectrpc/connect";

import { renderWithProviders } from "../../test-utils";
import { NodeStatus } from "../../api/nodes";
import { strings } from "../../consts/strings";
import { selectors } from "../../consts/selectors";
import { makeNode } from "./mocks/factories";

const { listNodes, revokeNode } = vi.hoisted(() => ({
  listNodes: vi.fn(),
  revokeNode: vi.fn(),
}));

vi.mock("../../api/nodes", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/nodes")>();
  return {
    ...actual,
    nodesClient: { listNodes, revokeNode },
  };
});

import { FleetPanel } from "./FleetPanel";

describe("FleetPanel", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the loading state before the list resolves", () => {
    listNodes.mockReturnValue(new Promise(() => {}));
    renderWithProviders(<FleetPanel />);
    expect(screen.getByTestId(selectors.fleet.loading)).toBeInTheDocument();
  });

  it("lists nodes with presence and offers revoke on a live node", async () => {
    listNodes.mockResolvedValue({
      nodes: [makeNode({ id: "n1", name: "ubuntu-ci", os: "linux", arch: "amd64" })],
    });
    renderWithProviders(<FleetPanel />);

    await waitFor(() => expect(screen.getByTestId(selectors.fleet.row({ id: "n1" }))).toBeInTheDocument());
    expect(screen.getByText(/ubuntu-ci/)).toBeInTheDocument();
    expect(screen.getByText(/linux\/amd64/)).toBeInTheDocument();
    // presence is conveyed by a labelled dot AND a status text label
    expect(screen.getByLabelText(strings.fleet.onlineLabel)).toBeInTheDocument();
    expect(screen.getByText(strings.fleet.status.online)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.fleet.revoke({ id: "n1" }))).toBeInTheDocument();
  });

  it("shows an offline node with its last-seen and offline label", async () => {
    listNodes.mockResolvedValue({
      nodes: [
        makeNode({
          id: "off1",
          status: NodeStatus.OFFLINE,
          online: false,
          lastSeenAt: timestampFromDate(new Date("2026-01-02T03:04:00Z")),
        }),
      ],
    });
    renderWithProviders(<FleetPanel />);

    await waitFor(() => expect(screen.getByTestId(selectors.fleet.row({ id: "off1" }))).toBeInTheDocument());
    expect(screen.getByLabelText(strings.fleet.offlineLabel)).toBeInTheDocument();
    expect(screen.getByText(strings.fleet.status.offline)).toBeInTheDocument();
  });

  it("hides revoke on an already-revoked node", async () => {
    listNodes.mockResolvedValue({
      nodes: [makeNode({ id: "r1", status: NodeStatus.REVOKED, online: false })],
    });
    renderWithProviders(<FleetPanel />);

    await waitFor(() => expect(screen.getByTestId(selectors.fleet.row({ id: "r1" }))).toBeInTheDocument());
    expect(screen.queryByTestId(selectors.fleet.revoke({ id: "r1" }))).not.toBeInTheDocument();
    expect(screen.getByText(strings.fleet.status.revoked)).toBeInTheDocument();
  });

  it("revokes a node only after confirmation", async () => {
    const user = userEvent.setup();
    listNodes.mockResolvedValue({ nodes: [makeNode({ id: "n1", name: "win-box" })] });
    revokeNode.mockResolvedValue({});
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(true);
    renderWithProviders(<FleetPanel />);

    await waitFor(() => expect(screen.getByTestId(selectors.fleet.row({ id: "n1" }))).toBeInTheDocument());
    await user.click(screen.getByTestId(selectors.fleet.revoke({ id: "n1" })));

    expect(confirmSpy).toHaveBeenCalled();
    await waitFor(() => expect(revokeNode).toHaveBeenCalledWith({ id: "n1" }));
    confirmSpy.mockRestore();
  });

  it("does not revoke when the confirm is dismissed", async () => {
    const user = userEvent.setup();
    listNodes.mockResolvedValue({ nodes: [makeNode({ id: "n1" })] });
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderWithProviders(<FleetPanel />);

    await waitFor(() => expect(screen.getByTestId(selectors.fleet.row({ id: "n1" }))).toBeInTheDocument());
    await user.click(screen.getByTestId(selectors.fleet.revoke({ id: "n1" })));

    expect(revokeNode).not.toHaveBeenCalled();
    confirmSpy.mockRestore();
  });

  it("renders the empty state when the fleet has no nodes", async () => {
    listNodes.mockResolvedValue({ nodes: [] });
    renderWithProviders(<FleetPanel />);
    await waitFor(() => expect(screen.getByTestId(selectors.fleet.empty)).toBeInTheDocument());
  });

  it("surfaces a typed error when the list query fails", async () => {
    listNodes.mockRejectedValue(new ConnectError("denied", Code.Unavailable));
    renderWithProviders(<FleetPanel />);
    await waitFor(() => expect(screen.getByTestId(selectors.fleet.error)).toBeInTheDocument());
    expect(screen.getByText(strings.errors.unavailable)).toBeInTheDocument();
  });
});
