/**
 * [REQ:BRG-P1-005] Fleet dashboard — node list surfaces OS / arch / version /
 * health with non-color-only status; pairing + revoke flows handle
 * loading / error / empty states; live per-node job status renders from the
 * scheduler overlay.
 *
 * This path is required verbatim by the requirements tracker. It exercises the
 * same FleetPanel + PairNodeForm the dashboard composes, asserting the
 * Phase-5 dashboard contract rather than re-testing the Phase-1 internals
 * already covered by FleetPanel.test.tsx.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Code, ConnectError } from "@connectrpc/connect";

import { renderWithProviders } from "../../test-utils";
import { NodeStatus } from "../../api/nodes";
import { strings } from "../../consts/strings";
import { selectors } from "../../consts/selectors";
import { makeNode, makeNodeQueue } from "./mocks/factories";

const { listNodes, revokeNode } = vi.hoisted(() => ({
  listNodes: vi.fn(),
  revokeNode: vi.fn(),
}));
const { listQueue } = vi.hoisted(() => ({ listQueue: vi.fn() }));
const { issuePairingCode, listPairingRequests, approvePairing } = vi.hoisted(() => ({
  issuePairingCode: vi.fn(),
  listPairingRequests: vi.fn(),
  approvePairing: vi.fn(),
}));

vi.mock("../../api/nodes", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/nodes")>();
  return { ...actual, nodesClient: { listNodes, revokeNode } };
});
vi.mock("../../api/queue", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/queue")>();
  return { ...actual, queueClient: { listQueue } };
});
vi.mock("../../api/pairing", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/pairing")>();
  return { ...actual, pairingClient: { issuePairingCode, listPairingRequests, approvePairing } };
});

import { FleetPanel } from "./FleetPanel";
import { PairNodeForm } from "./PairNodeForm";
import { PendingPairingPanel } from "./PendingPairingPanel";

function renderFleet() {
  listPairingRequests.mockResolvedValue({ requests: [], presets: [] });
  return renderWithProviders(
    <>
      <PairNodeForm />
      <FleetPanel />
    </>,
  );
}

describe("[REQ:BRG-P1-005] Fleet dashboard", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the node list with OS / arch / version / health and non-color-only status", async () => {
    listNodes.mockResolvedValue({
      nodes: [
        makeNode({
          id: "n1",
          name: "ubuntu-ci",
          os: "linux",
          arch: "amd64",
          revision: "deadbeefcafe1234",
          status: NodeStatus.ONLINE,
          online: true,
        }),
      ],
    });
    listQueue.mockResolvedValue({ nodes: [] });
    renderFleet();

    const row = await screen.findByTestId(selectors.fleet.row({ id: "n1" }));
    const scoped = within(row);

    // OS / arch / version / health labels are all present.
    expect(scoped.getByText(strings.fleet.osLabel)).toBeInTheDocument();
    expect(scoped.getByText(strings.fleet.archLabel)).toBeInTheDocument();
    expect(scoped.getByText(strings.fleet.versionLabel)).toBeInTheDocument();
    expect(scoped.getByText(strings.fleet.healthLabel)).toBeInTheDocument();

    // Concrete data values (asserted via textContent, not copy-driven queries).
    expect(row).toHaveTextContent("linux");
    expect(row).toHaveTextContent("amd64");
    expect(row).toHaveTextContent("deadbeefca"); // 10-char short revision
    // Health label appears as text (status), not just a color.
    expect(scoped.getAllByText(strings.fleet.status.online).length).toBeGreaterThan(0);
    // Presence is a labelled image (icon/dot) — non-color-only signal.
    expect(scoped.getByLabelText(strings.fleet.onlineLabel)).toBeInTheDocument();
  });

  it("surfaces live per-node job status from the queue overlay", async () => {
    listNodes.mockResolvedValue({ nodes: [makeNode({ id: "n1" })] });
    listQueue.mockResolvedValue({
      nodes: [makeNodeQueue({ nodeId: "n1", running: 1, queued: 2 })],
    });
    renderFleet();

    const jobs = await screen.findByTestId(selectors.fleet.jobs({ id: "n1" }));
    // Tests run in i18n cimode, so the panel renders the raw key path; a busy
    // node shows the "busy" key (running/queued counts), not the "idle" key.
    await waitFor(() => expect(jobs).toHaveTextContent(strings.fleet.jobsBusy));
    expect(jobs).not.toHaveTextContent(strings.fleet.jobsIdle);
  });

  it("shows the node-list loading state and empty state", async () => {
    // Loading: query never resolves.
    listNodes.mockReturnValue(new Promise(() => {}));
    listQueue.mockResolvedValue({ nodes: [] });
    const { unmount } = renderFleet();
    expect(screen.getByTestId(selectors.fleet.loading)).toBeInTheDocument();
    unmount();
    cleanup();

    // Empty: resolves to no nodes.
    listNodes.mockResolvedValue({ nodes: [] });
    renderFleet();
    await waitFor(() => expect(screen.getByTestId(selectors.fleet.empty)).toBeInTheDocument());
  });

  it("surfaces a typed error when the node list fails to load", async () => {
    listNodes.mockRejectedValue(new ConnectError("denied", Code.Unavailable));
    listQueue.mockResolvedValue({ nodes: [] });
    renderFleet();
    await waitFor(() => expect(screen.getByTestId(selectors.fleet.error)).toBeInTheDocument());
    expect(screen.getByText(strings.errors.unavailable)).toBeInTheDocument();
  });

  it("revokes a node after confirmation", async () => {
    const user = userEvent.setup();
    listNodes.mockResolvedValue({ nodes: [makeNode({ id: "n1", name: "win-box" })] });
    listQueue.mockResolvedValue({ nodes: [] });
    revokeNode.mockResolvedValue({});
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(true);
    renderFleet();

    await screen.findByTestId(selectors.fleet.row({ id: "n1" }));
    await user.click(screen.getByTestId(selectors.fleet.revoke({ id: "n1" })));

    expect(confirmSpy).toHaveBeenCalled();
    await waitFor(() => expect(revokeNode).toHaveBeenCalledWith({ id: "n1" }));
    confirmSpy.mockRestore();
  });

  describe("pairing flow", () => {
    it("requires matching confirmation words before approving and sends the selected preset", async () => {
      const user = userEvent.setup();
      listPairingRequests.mockResolvedValue({
        requests: [{ id: "req-1", name: "scratch-mac", os: "darwin", arch: "arm64", endpoint: "192.168.1.20", confirmationWords: ["amber", "orbit", "cedar"] }],
        presets: [
          { name: "read-only", description: "Read status and telemetry", scopes: ["vrooli:read"], withholds: ["write", "destructive"] },
          { name: "operate", description: "Read and operate", scopes: ["vrooli:read", "vrooli:write"], withholds: ["destructive"] },
        ],
      });
      approvePairing.mockResolvedValue({ status: 2, nodeId: "node-1" });
      renderWithProviders(<PendingPairingPanel />);

      const row = await screen.findByTestId(selectors.fleet.pairingRequests.row({ id: "req-1" }));
      expect(within(row).getByTestId(selectors.fleet.pairingRequests.approve({ id: "req-1" }))).toBeDisabled();
      expect(within(row).getByTestId(selectors.fleet.pairingRequests.words({ id: "req-1" }))).toHaveTextContent("amber orbit cedar");

      await user.click(within(row).getByTestId(selectors.fleet.pairingRequests.wordsMatch({ id: "req-1" })));
      await user.selectOptions(within(row).getByTestId(selectors.fleet.pairingRequests.preset({ id: "req-1" })), "operate");
      await user.click(within(row).getByTestId(selectors.fleet.pairingRequests.approve({ id: "req-1" })));

      await waitFor(() => expect(approvePairing).toHaveBeenCalledWith({
        requestId: "req-1",
        approve: true,
        scopes: ["vrooli:read", "vrooli:write"],
        confirmationWords: ["amber", "orbit", "cedar"],
      }));
    });

    it("starts with no result (empty), then surfaces the minted code on success", async () => {
      const user = userEvent.setup();
      listNodes.mockResolvedValue({ nodes: [] });
      listQueue.mockResolvedValue({ nodes: [] });
      issuePairingCode.mockResolvedValue({
        code: "AB12-CD34",
        controlPlanePublicKey: "base64key==",
        expiresAt: undefined,
      });
      renderFleet();

      // Empty: no pairing result before submitting.
      expect(screen.queryByTestId(selectors.fleet.pairing.result)).not.toBeInTheDocument();

      await user.type(screen.getByTestId(selectors.fleet.pairing.nameInput), "mac-mini");
      await user.click(screen.getByTestId(selectors.fleet.pairing.submit));

      await waitFor(() => expect(issuePairingCode).toHaveBeenCalledWith({ name: "mac-mini" }));
      const result = await screen.findByTestId(selectors.fleet.pairing.result);
      expect(within(result).getByTestId(selectors.fleet.pairing.code)).toHaveTextContent("AB12-CD34");
    });

    it("shows a loading state while the code is being minted", async () => {
      const user = userEvent.setup();
      listNodes.mockResolvedValue({ nodes: [] });
      listQueue.mockResolvedValue({ nodes: [] });
      let resolve!: (v: unknown) => void;
      issuePairingCode.mockReturnValue(new Promise((r) => (resolve = r)));
      renderFleet();

      await user.type(screen.getByTestId(selectors.fleet.pairing.nameInput), "mac-mini");
      await user.click(screen.getByTestId(selectors.fleet.pairing.submit));

      const submit = screen.getByTestId(selectors.fleet.pairing.submit);
      await waitFor(() => expect(submit).toBeDisabled());
      expect(submit).toHaveTextContent(strings.fleet.pairing.submitting);

      resolve({ code: "X", controlPlanePublicKey: "k", expiresAt: undefined });
    });

    it("surfaces an error when minting fails", async () => {
      const user = userEvent.setup();
      listNodes.mockResolvedValue({ nodes: [] });
      listQueue.mockResolvedValue({ nodes: [] });
      issuePairingCode.mockRejectedValue(new ConnectError("nope", Code.PermissionDenied));
      renderFleet();

      await user.type(screen.getByTestId(selectors.fleet.pairing.nameInput), "mac-mini");
      await user.click(screen.getByTestId(selectors.fleet.pairing.submit));

      await waitFor(() =>
        expect(screen.getByTestId(selectors.fleet.pairing.error)).toBeInTheDocument(),
      );
      expect(screen.getByText(strings.errors.permission_denied)).toBeInTheDocument();
    });
  });
});
