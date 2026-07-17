import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { Code, ConnectError } from "@connectrpc/connect";

import { renderWithProviders } from "../../test-utils";
import { NodeStatus } from "../../api/nodes";
import { OnboardingState } from "../../api/onboard";
import { strings } from "../../consts/strings";
import { selectors } from "../../consts/selectors";
import { makeGetOnboardingResponse, makeNode, makeOnboardingOp, makeStepEvent } from "./mocks/factories";

const { listNodes, revokeNode } = vi.hoisted(() => ({
  listNodes: vi.fn(),
  revokeNode: vi.fn(),
}));
const { listQueue } = vi.hoisted(() => ({ listQueue: vi.fn() }));
const { listOnboardings, getOnboarding, removeFailedOnboarding } = vi.hoisted(() => ({ listOnboardings: vi.fn(), getOnboarding: vi.fn(), removeFailedOnboarding: vi.fn() }));
const { fetchBridgeReadiness, performBridgeFirewallAction } = vi.hoisted(() => ({ fetchBridgeReadiness: vi.fn(), performBridgeFirewallAction: vi.fn() }));

vi.mock("../../api/nodes", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/nodes")>();
  return {
    ...actual,
    nodesClient: { listNodes, revokeNode },
  };
});

vi.mock("../../api/queue", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/queue")>();
  return { ...actual, queueClient: { listQueue } };
});
vi.mock("../../api/onboard", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/onboard")>();
  return { ...actual, onboardClient: { listOnboardings, getOnboarding, removeFailedOnboarding } };
});
vi.mock("../../api/readiness", () => ({ fetchBridgeReadiness, performBridgeFirewallAction }));

import { FleetPanel } from "./FleetPanel";

describe("FleetPanel", () => {
  beforeEach(() => {
    // The queue overlay is best-effort; default it to empty so the panel
    // renders idle job status without a real network attempt.
    listQueue.mockResolvedValue({ nodes: [] });
    listOnboardings.mockResolvedValue({ ops: [] });
    fetchBridgeReadiness.mockResolvedValue({ status: "ready", endpoint: "http://bridge.test:18767", port: 18767, endpoint_source: "configured", reachability_mode: "lan", local_api: true });
  });

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

    const row = await screen.findByTestId(selectors.fleet.row({ id: "n1" }));
    expect(screen.getByText(/ubuntu-ci/)).toBeInTheDocument();
    // OS and arch are now rendered as discrete labelled metadata fields; assert
    // via the row's textContent (data values, not copy).
    expect(row).toHaveTextContent("linux");
    expect(row).toHaveTextContent("amd64");
    // presence is conveyed by a labelled dot AND a status text label (the label
    // also repeats as the "Health" metadata value, so >= 1 occurrence).
    expect(screen.getByLabelText(strings.fleet.onlineLabel)).toBeInTheDocument();
    expect(screen.getAllByText(strings.fleet.status.online).length).toBeGreaterThan(0);
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
    expect(screen.getAllByText(strings.fleet.status.offline).length).toBeGreaterThan(0);
  });

  it("hides revoke on an already-revoked node", async () => {
    listNodes.mockResolvedValue({
      nodes: [makeNode({ id: "r1", status: NodeStatus.REVOKED, online: false })],
    });
    renderWithProviders(<FleetPanel />);

    await waitFor(() => expect(screen.getByTestId(selectors.fleet.row({ id: "r1" }))).toBeInTheDocument());
    expect(screen.queryByTestId(selectors.fleet.revoke({ id: "r1" }))).not.toBeInTheDocument();
    expect(screen.getAllByText(strings.fleet.status.revoked).length).toBeGreaterThan(0);
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

  it("keeps a failed onboarding target visible and sends its non-secret identity to retry", async () => {
    const onRetryOnboarding = vi.fn();
    const user = userEvent.setup();
    listNodes.mockResolvedValue({ nodes: [] });
    listOnboardings.mockResolvedValue({
      ops: [makeOnboardingOp({ id: "op-build-node", nodeName: "build-node", host: "192.0.2.10", state: OnboardingState.FAILED })],
    });
    renderWithProviders(<FleetPanel onRetryOnboarding={onRetryOnboarding} />);

    await screen.findByTestId(selectors.fleet.failedOnboardings);
    await user.click(screen.getByTestId(selectors.fleet.onboardRetry({ id: "op-build-node" })));
    expect(onRetryOnboarding).toHaveBeenCalledWith(expect.objectContaining({ host: "192.0.2.10" }));
  });

  it("retrieves persisted diagnostics for a failed onboarding after reload", async () => {
    const user = userEvent.setup();
    const failed = makeOnboardingOp({ id: "op-history", host: "192.0.2.9", state: OnboardingState.FAILED });
    listNodes.mockResolvedValue({ nodes: [] });
    listOnboardings.mockResolvedValue({ ops: [failed] });
    getOnboarding.mockResolvedValue(
      makeGetOnboardingResponse(
        { ...failed, failureDetail: "make setup: checksum mismatch" },
        [makeStepEvent({ stepId: "setup", detail: "checksum mismatch", status: 4 })],
      ),
    );
    renderWithProviders(<FleetPanel />);

    await user.click(await screen.findByTestId(selectors.fleet.onboardViewLogs({ id: "op-history" })));
    await waitFor(() => expect(getOnboarding).toHaveBeenCalledWith({ id: "op-history" }));
    expect(await screen.findByTestId(selectors.fleet.onboard.failureOutput)).toHaveTextContent("make setup: checksum mismatch");
    expect(screen.getByTestId(selectors.fleet.onboard.failureOutput)).toHaveTextContent("setup");
  });

  it("removes a failed onboarding target from durable history", async () => {
    const user = userEvent.setup();
    listNodes.mockResolvedValue({ nodes: [] });
    listOnboardings.mockResolvedValue({
      ops: [makeOnboardingOp({ id: "op-remove", state: OnboardingState.FAILED })],
    });
    removeFailedOnboarding.mockResolvedValue({});
    renderWithProviders(<FleetPanel />);

    await user.click(await screen.findByTestId(selectors.fleet.onboardRemove({ id: "op-remove" })));
    await waitFor(() => expect(removeFailedOnboarding).toHaveBeenCalledWith({ id: "op-remove" }));
  });

  it("invites adding the first node from the empty state and fires the callback", async () => {
    const onAddNode = vi.fn();
    const user = userEvent.setup();
    listNodes.mockResolvedValue({ nodes: [] });
    renderWithProviders(<FleetPanel onAddNode={onAddNode} />);

    await waitFor(() => expect(screen.getByTestId(selectors.fleet.empty)).toBeInTheDocument());
    // The empty card carries the welcoming heading and a single primary CTA.
    expect(screen.getByText(strings.fleet.emptyHeading)).toBeInTheDocument();
    await user.click(screen.getByTestId(selectors.fleet.onboard.addNode));
    expect(onAddNode).toHaveBeenCalledTimes(1);
  });

  it("shows an Add-a-node header button once the fleet has nodes", async () => {
    const onAddNode = vi.fn();
    const user = userEvent.setup();
    listNodes.mockResolvedValue({ nodes: [makeNode({ id: "n1" })] });
    renderWithProviders(<FleetPanel onAddNode={onAddNode} />);

    await screen.findByTestId(selectors.fleet.row({ id: "n1" }));
    await user.click(screen.getByTestId(selectors.fleet.onboard.addNode));
    expect(onAddNode).toHaveBeenCalledTimes(1);
  });

  it("surfaces a typed error when the list query fails", async () => {
    listNodes.mockRejectedValue(new ConnectError("denied", Code.Unavailable));
    renderWithProviders(<FleetPanel />);
    await waitFor(() => expect(screen.getByTestId(selectors.fleet.error)).toBeInTheDocument());
    expect(screen.getByText(strings.errors.unavailable)).toBeInTheDocument();
  });

  it("confirms a scoped broker admission for the recorded failed candidate", async () => {
    const user = userEvent.setup();
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(true);
    listNodes.mockResolvedValue({ nodes: [] });
    fetchBridgeReadiness.mockResolvedValue({
      status: "candidate_blocked", endpoint: "http://bridge.test:18767", port: 18767, endpoint_source: "configured", reachability_mode: "lan", local_api: true,
      last_candidate: { host: "mini", endpoint: "http://bridge.test:18767", mode: "lan", state: "failed", source_ip: "192.168.1.176" },
      firewall: { available: true, inspectable: true, active: true, rule_found: false, privileged: true, broker_available: true, broker_status: "verified" },
    });
    performBridgeFirewallAction.mockResolvedValue({ status: "changed", changed: true });
    renderWithProviders(<FleetPanel />);
    await user.click(await screen.findByText(strings.fleet.bridgeReadinessAllow));
    expect(confirmSpy).toHaveBeenCalled();
    await waitFor(() => expect(performBridgeFirewallAction).toHaveBeenCalledWith("allow", "192.168.1.176", true));
    confirmSpy.mockRestore();
  });

  it("previews and revokes only the recorded candidate rule", async () => {
    const user = userEvent.setup();
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(true);
    listNodes.mockResolvedValue({ nodes: [] });
    fetchBridgeReadiness.mockResolvedValue({
      status: "candidate_blocked", endpoint: "http://bridge.test:18767", port: 18767, endpoint_source: "configured", reachability_mode: "lan", local_api: true,
      last_candidate: { host: "mini", endpoint: "http://bridge.test:18767", mode: "lan", state: "failed", source_ip: "192.168.1.176" },
      firewall: { available: true, inspectable: true, active: true, rule_found: true, privileged: true, broker_available: true, broker_status: "verified" },
    });
    performBridgeFirewallAction.mockResolvedValue({ status: "changed", changed: true });
    renderWithProviders(<FleetPanel />);
    await user.click(await screen.findByText(strings.fleet.bridgeReadinessPreview));
    await waitFor(() => expect(performBridgeFirewallAction).toHaveBeenCalledWith("preview", "192.168.1.176", false));
    await user.click(screen.getByText(strings.fleet.bridgeReadinessRevoke));
    await waitFor(() => expect(performBridgeFirewallAction).toHaveBeenCalledWith("revoke", "192.168.1.176", true));
    confirmSpy.mockRestore();
  });
});
