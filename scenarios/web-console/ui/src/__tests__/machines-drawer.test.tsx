import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent, waitFor, within } from "@testing-library/react";
import { setLocale } from "../i18n";

// The machines surface is the flow that has to work before any fleet feature
// matters: a person adds a second machine, decides what it may do, and then
// uses it — without ever opening the control plane's own interface.

vi.mock("../api/machines", async () => {
  const actual = await vi.importActual<typeof import("../api/machines")>("../api/machines");
  return {
    ...actual,
    listFleet: vi.fn(),
    issueJoinCode: vi.fn(),
    decideJoinRequest: vi.fn(),
    setMachineGrant: vi.fn(),
    forgetMachine: vi.fn(),
  };
});

import {
  listFleet as _listFleet,
  issueJoinCode as _issueJoinCode,
  decideJoinRequest as _decideJoinRequest,
  setMachineGrant as _setMachineGrant,
  type Fleet,
  type JoinRequest,
  type Machine,
} from "../api/machines";
import FleetDrawer from "../components/fleet/FleetDrawer";

const MachinesDrawer = FleetDrawer;

const listFleet = vi.mocked(_listFleet);
const issueJoinCode = vi.mocked(_issueJoinCode);
const decideJoinRequest = vi.mocked(_decideJoinRequest);
const setMachineGrant = vi.mocked(_setMachineGrant);

function machine(overrides: Partial<Machine> & { id: string; label: string }): Machine {
  const { id, label, ...rest } = overrides;
  return {
    target: {
      id,
      kind: id === "local" ? "local" : "bridge-node",
      label,
      os: "darwin",
      arch: "amd64",
      available: true,
      state: "dispatchable",
      ...(rest.target ?? {}),
    },
    grant: {
      summary: "Read only; changes are not permitted",
      effects: ["read"],
      appCount: 0,
      coversAllApps: true,
      scopes: ["*:read"],
      preset: "",
      ...(rest.grant ?? {}),
    },
    heartbeatAgeSeconds: rest.heartbeatAgeSeconds ?? 8,
    manageable: rest.manageable ?? id !== "local",
    drift: rest.drift ?? [],
  };
}

const pendingRequest: JoinRequest = {
  id: "req-1",
  name: "Studio Mac",
  os: "darwin",
  arch: "arm64",
  endpoint: "192.168.1.44",
  confirmationWords: ["amber", "dolphin", "quartz"],
  keyFingerprint: "ed25519:9f3c…a71d",
  requestedAgeSeconds: 12,
};

function fleet(overrides: Partial<Fleet> = {}): Fleet {
  return {
    status: "ready",
    machines: [
      machine({ id: "local", label: "This machine", target: { id: "local", kind: "local", label: "This machine", os: "linux", arch: "x86_64", available: true } }),
      machine({ id: "bridge-node:node-a", label: "minimouse" }),
    ],
    joinRequests: [],
    presets: [
      { name: "read-only", title: "Read only", description: "See its metrics, logs and status. Change nothing.", scopes: ["system-monitor:read", "web-console:read"], withholds: ["write", "destructive"], summary: "Read only; changes are not permitted", effects: ["read"], appCount: 2 },
      { name: "operate", title: "Operate", description: "Also start, stop and restart apps on it.", scopes: ["system-monitor:read", "system-monitor:write"], withholds: ["destructive"], summary: "Read and operate; destructive actions withheld", effects: ["read", "write"], appCount: 1 },
    ],
    message: "",
    recoveryAction: "",
    controlPlane: {
      reachable: true,
      // The API base the server dials, and the UI the browser opens, are
      // different ports on purpose. The fixture keeps them apart so a test can
      // tell which one a surface used.
      endpoint: "http://localhost:18767",
      detail: "",
      consoleUrl: "http://localhost:22054",
    },
    ...overrides,
  };
}

// This file renders real English rather than i18next's `cimode` pseudo-locale,
// because what it asserts is that specific *facts* reach the screen — the age a
// machine last answered, the key-derived words, the endpoint machines register
// against. Under cimode an interpolated value never renders at all, so those
// claims would be unfalsifiable. Structural assertions still use test ids.
describe("machines surface", () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    listFleet.mockResolvedValue(fleet());
    await setLocale("en");
  });

  it("renders nothing while closed, and does not read the fleet", () => {
    const { container } = render(<MachinesDrawer open={false} onClose={vi.fn()} />);
    expect(container.innerHTML).toBe("");
    expect(listFleet).not.toHaveBeenCalled();
  });

  it("lists every linked machine with permission as a sentence", async () => {
    render(<MachinesDrawer open onClose={vi.fn()} />);
    const row = await screen.findByTestId("machines-row-bridge-node-node-a");
    expect(within(row).getByText("minimouse")).toBeTruthy();
    // The card gives permission one line, so the posture and its breadth are
    // one sentence rather than two stacked rows. A wildcard reaches apps that
    // do not exist yet, so the breadth is never a count.
    expect(within(row).getByText(/Read only; changes are not permitted · every app/)).toBeTruthy();
  });

  it("states how long ago a machine answered rather than implying it", async () => {
    listFleet.mockResolvedValue(
      fleet({
        machines: [
          machine({ id: "local", label: "This machine", target: { id: "local", kind: "local", label: "This machine", available: true } }),
          machine({
            id: "bridge-node:stale",
            label: "swarminator",
            target: { id: "bridge-node:stale", kind: "bridge-node", label: "swarminator", os: "linux", arch: "arm64", available: false, state: "offline", recovery_action: "Reconnect the Bridge agent on this node, then refresh the catalog" },
            heartbeatAgeSeconds: 7 * 24 * 3600,
          }),
        ],
      }),
    );
    render(<MachinesDrawer open onClose={vi.fn()} />);
    const row = await screen.findByTestId("machines-row-bridge-node-stale");
    expect(within(row).getByText(/7 days ago/)).toBeTruthy();
    expect(within(row).getByText("Not responding")).toBeTruthy();
    // Every unreachable card offers something the operator can do here: the
    // primary verb changes rather than offering a session that cannot open.
    expect(within(row).getByTestId("machines-reconnect-bridge-node-stale")).toBeTruthy();
    expect(within(row).queryByTestId("machines-start-session-bridge-node-stale")).toBeNull();
    // The sentence explaining the recovery is variable length, so it lives in
    // the detail sheet rather than making cards on one shelf different heights.
    expect(within(row).queryByText(/Reconnect the Bridge agent/)).toBeNull();
    fireEvent.click(within(row).getByTestId("machines-details-bridge-node-stale"));
    expect(await screen.findByTestId("machine-detail-recovery-bridge-node-stale")).toHaveTextContent(
      "Reconnect the Bridge agent",
    );
  });

  it("explains an unenrolled installation instead of showing an empty list", async () => {
    listFleet.mockResolvedValue(
      fleet({
        status: "unenrolled",
        machines: [machine({ id: "local", label: "This machine", target: { id: "local", kind: "local", label: "This machine", available: true } })],
        message: "Bridge credentials not configured",
        recoveryAction: "Enroll this machine with Bridge, then refresh the catalog",
      }),
    );
    render(<MachinesDrawer open onClose={vi.fn()} />);
    const empty = await screen.findByTestId("machines-empty");
    expect(within(empty).getByText("This computer is not part of a fleet")).toBeTruthy();
    expect(within(empty).getByText(/Enroll this machine with Bridge/)).toBeTruthy();
  });

  it("surfaces a machine asking to join without the operator looking for it", async () => {
    listFleet.mockResolvedValue(fleet({ joinRequests: [pendingRequest] }));
    render(<MachinesDrawer open onClose={vi.fn()} />);
    const banner = await screen.findByTestId("machines-join-request-req-1");
    expect(within(banner).getByText(/Studio Mac/)).toBeTruthy();
  });

  it("shows the key-derived words and fingerprint before it offers to link", async () => {
    listFleet.mockResolvedValue(fleet({ joinRequests: [pendingRequest] }));
    render(<MachinesDrawer open onClose={vi.fn()} />);
    fireEvent.click(await screen.findByTestId("machines-review-req-1"));

    expect(screen.getByTestId("machines-confirmation-words").textContent).toContain("amber");
    expect(screen.getByTestId("machines-key-fingerprint").textContent).toContain("ed25519");
    expect(screen.getByTestId("machines-words-match")).toBeTruthy();
  });

  it("refuses to continue when the request carries no derived words", async () => {
    listFleet.mockResolvedValue(fleet({ joinRequests: [{ ...pendingRequest, confirmationWords: [] }] }));
    render(<MachinesDrawer open onClose={vi.fn()} />);
    fireEvent.click(await screen.findByTestId("machines-review-req-1"));

    expect(screen.getByTestId("machines-no-words")).toBeTruthy();
    expect(screen.getByTestId("machines-words-match")).toHaveProperty("disabled", true);
  });

  it("denies a request without asking for a permission choice", async () => {
    listFleet.mockResolvedValue(fleet({ joinRequests: [pendingRequest] }));
    decideJoinRequest.mockResolvedValue("This machine was not linked.");
    render(<MachinesDrawer open onClose={vi.fn()} />);
    fireEvent.click(await screen.findByTestId("machines-review-req-1"));
    fireEvent.click(screen.getByTestId("machines-deny"));

    await waitFor(() => {
      expect(decideJoinRequest).toHaveBeenCalledWith(
        expect.objectContaining({ requestId: "req-1", approve: false }),
      );
    });
  });

  it("defaults to the narrowest preset and shows what it withholds", async () => {
    listFleet.mockResolvedValue(fleet({ joinRequests: [pendingRequest] }));
    render(<MachinesDrawer open onClose={vi.fn()} />);
    fireEvent.click(await screen.findByTestId("machines-review-req-1"));
    fireEvent.click(screen.getByTestId("machines-words-match"));

    // The card surface is decoration; the radio underneath is the selection.
    expect(screen.getByTestId("machines-preset-read-only")).toBeChecked();
    expect(screen.getByTestId("machines-withheld-write")).toBeTruthy();
    expect(screen.getByTestId("machines-withheld-destructive")).toBeTruthy();
  });

  it("sends the confirmed words and the chosen preset when linking", async () => {
    listFleet.mockResolvedValue(fleet({ joinRequests: [pendingRequest] }));
    decideJoinRequest.mockResolvedValue("This machine is linked.");
    render(<MachinesDrawer open onClose={vi.fn()} />);
    fireEvent.click(await screen.findByTestId("machines-review-req-1"));
    fireEvent.click(screen.getByTestId("machines-words-match"));
    fireEvent.click(screen.getByTestId("machines-preset-operate"));
    fireEvent.click(screen.getByTestId("machines-grant-confirm"));

    await waitFor(() => {
      expect(decideJoinRequest).toHaveBeenCalledWith({
        requestId: "req-1",
        approve: true,
        confirmationWords: ["amber", "dolphin", "quartz"],
        preset: "operate",
      });
    });
  });

  it("keeps the expanded scope list available for audit without leading with it", async () => {
    listFleet.mockResolvedValue(fleet({ joinRequests: [pendingRequest] }));
    render(<MachinesDrawer open onClose={vi.fn()} />);
    fireEvent.click(await screen.findByTestId("machines-review-req-1"));
    fireEvent.click(screen.getByTestId("machines-words-match"));

    expect(screen.queryByTestId("machines-scope-audit")).toBeNull();
    fireEvent.click(screen.getByTestId("machines-scope-audit-toggle"));
    expect(within(screen.getByTestId("machines-scope-audit")).getByText("system-monitor:read")).toBeTruthy();
  });

  it("changes an existing machine's grant through the same preset vocabulary", async () => {
    setMachineGrant.mockResolvedValue(null);
    render(<MachinesDrawer open onClose={vi.fn()} />);
    // Permission is a tab of the machine, not a peer button on its card.
    fireEvent.click(await screen.findByTestId("machines-details-bridge-node-node-a"));
    fireEvent.click(await screen.findByTestId("machine-detail-tab-permissions"));
    fireEvent.click(screen.getByTestId("machines-preset-operate"));
    fireEvent.click(screen.getByTestId("machines-grant-confirm"));

    await waitFor(() => {
      expect(setMachineGrant).toHaveBeenCalledWith("bridge-node:node-a", "operate");
    });
  });

  it("issues a join code on request and never before", async () => {
    issueJoinCode.mockResolvedValue({ code: "7K4M92QXAB3D", expiresInSeconds: 581 });
    render(<MachinesDrawer open onClose={vi.fn()} />);
    fireEvent.click(await screen.findByTestId("machines-add"));
    expect(issueJoinCode).not.toHaveBeenCalled();

    fireEvent.click(screen.getByTestId("machines-issue-code"));
    await waitFor(() => {
      expect(screen.getByTestId("machines-join-code").textContent).toContain("7K4M");
    });
    // Two things this must not be: a race against a live one-second countdown,
    // and a match on translated copy — a sibling suite can leave the locale
    // switched, and this file's own beforeEach is not enough to stop that.
    // The state is what matters: a code was issued, and it has not expired.
    const expiry = screen.getByTestId("machines-code-expiry");
    expect(expiry).toHaveAttribute("data-expired", "false");
    expect(expiry.textContent).toMatch(/\d+:[0-5]\d/);
  });

  it("states where linked machines are registered", async () => {
    render(<MachinesDrawer open onClose={vi.fn()} />);
    expect(await screen.findByTestId("machines-footer")).toBeTruthy();
    expect(screen.getByTestId("machines-control-plane").textContent).toContain("18767");
  });

  it("hands off to the control plane through a link a browser can open", async () => {
    render(<MachinesDrawer open onClose={vi.fn()} />);
    fireEvent.click(await screen.findByTestId("machines-add"));
    fireEvent.click(screen.getByTestId("machines-door-server"));

    const link = screen.getByTestId("machines-open-bridge");
    // The API base is what this server dials; it answers a browser with 404 and
    // names loopback on whichever machine the server runs on. The link has to
    // carry the interface address, resolved against the caller's own origin.
    expect(link.getAttribute("href")).toBe("http://localhost:22054");
    expect(link.getAttribute("href")).not.toContain("18767");
  });

  it("offers no handoff at all when the control plane interface cannot be located", async () => {
    listFleet.mockResolvedValue(
      fleet({ controlPlane: { reachable: true, endpoint: "http://localhost:18767", detail: "", consoleUrl: "" } }),
    );
    render(<MachinesDrawer open onClose={vi.fn()} />);
    fireEvent.click(await screen.findByTestId("machines-add"));
    fireEvent.click(screen.getByTestId("machines-door-server"));

    // A link that cannot work is worse than an absent one: it spends the
    // person's trust and returns an error page.
    expect(screen.queryByTestId("machines-open-bridge")).toBeNull();
    expect(screen.getByTestId("machines-server-command")).toBeTruthy();
  });

  it("reports a refused action instead of failing silently", async () => {
    listFleet.mockResolvedValue(fleet({ joinRequests: [pendingRequest] }));
    decideJoinRequest.mockRejectedValue(new Error("the confirmation words do not match"));
    render(<MachinesDrawer open onClose={vi.fn()} />);
    fireEvent.click(await screen.findByTestId("machines-review-req-1"));
    fireEvent.click(screen.getByTestId("machines-words-match"));
    fireEvent.click(screen.getByTestId("machines-grant-confirm"));

    const failure = await screen.findByTestId("machines-failure");
    expect(failure.textContent).toContain("do not match");
  });
});
