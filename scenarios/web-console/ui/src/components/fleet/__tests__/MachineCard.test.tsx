import { renderWithProviders as render } from "../../../test-utils";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import MachineCard, { JoinRequestCard } from "../MachineCard";
import type { JoinRequest, Machine } from "../../../api/machines";

function machine(overrides: Partial<Machine["target"]> = {}): Machine {
  return {
    target: {
      id: "machine-1",
      kind: "bridge-node",
      label: "Remote machine",
      available: true,
      ...overrides,
    },
    grant: { summary: "Read terminal", effects: ["read"], appCount: 1, coversAllApps: false, scopes: [], preset: "read" },
    heartbeatAgeSeconds: 4,
    manageable: true,
    drift: [],
  };
}

function joinRequest(): JoinRequest {
  return {
    id: "join-1",
    name: "newhost",
    os: "linux",
    arch: "arm64",
    endpoint: "192.168.1.20:8080",
    confirmationWords: ["amber", "kettle", "north"],
    keyFingerprint: "SHA256:abc",
    requestedAgeSeconds: 30,
  };
}

describe("MachineCard", () => {
  it("uses the machine silhouette for a remote machine", () => {
    const { container } = render(<MachineCard machine={machine()} />);
    expect(screen.getByTestId("fleet-card-machine-machine-1")).toBeInTheDocument();
    expect(container.querySelector("[data-testid=machine-silhouette]")).toBeInTheDocument();
    expect(container.querySelector("[data-testid=device-silhouette]")).toBeNull();
  });

  it("uses the device artwork for the local machine, not a chassis", () => {
    // The local machine is the screen the operator is already looking at, so it
    // is drawn by the same silhouette the devices row uses. A chassis here
    // would claim a headless box that is demonstrably not headless.
    const { container } = render(<MachineCard machine={machine({ kind: "local", label: "This computer" })} />);
    expect(container.querySelector("[data-testid=device-silhouette]")).toBeInTheDocument();
    expect(container.querySelector("[data-testid=machine-silhouette]")).toBeNull();
    expect(screen.getByText("machines.thisComputer")).toBeInTheDocument();
  });

  it.each([
    ["dispatchable", { available: true }, 4],
    ["offline", { available: false }, 90],
    ["unenrolled", { available: false }, 0],
  ])("draws the %s lamp state", (state, overrides, heartbeatAgeSeconds) => {
    const item = { ...machine(overrides), heartbeatAgeSeconds };
    const { container } = render(<MachineCard machine={item} />);
    expect(container.querySelector("[data-testid=machine-silhouette]")).toHaveAttribute("data-state", state);
  });

  it("draws a join request as a machine that has never answered", () => {
    const { container } = render(<JoinRequestCard request={joinRequest()} onReview={() => {}} />);
    expect(container.querySelector("[data-testid=machine-silhouette]")).toHaveAttribute("data-state", "unenrolled");
  });

  it("swaps the primary verb for an unreachable machine rather than offering a dead session", () => {
    render(<MachineCard machine={machine({ available: false, label: "Offline host" })} onOpen={() => {}} onStartSession={() => {}} />);
    expect(screen.getByText("machines.statusNotResponding")).toBeInTheDocument();
    expect(screen.getByTestId("machines-reconnect-machine-1")).toBeInTheDocument();
    expect(screen.queryByTestId("machines-start-session-machine-1")).not.toBeInTheDocument();
  });

  it("counts drift and missing capabilities into one row instead of listing them on the card", () => {
    const onOpen = vi.fn();
    const item = {
      ...machine(),
      drift: [
        { kind: "capability", name: "ai-cli:codex", reason: "required capability is not reported by the node" },
        { kind: "profile", name: "managed-connection", reason: "profile has not been applied" },
      ],
    };
    render(<MachineCard machine={item} onOpen={onOpen} />);

    const issues = screen.getByTestId("machines-issues-machine-1");
    // The count is what the card carries; the list belongs to the tab that owns it.
    expect(issues).toHaveTextContent("machines.needsAttention");
    expect(screen.queryByText("profile has not been applied")).not.toBeInTheDocument();

    fireEvent.click(issues);
    expect(onOpen).toHaveBeenCalledWith(item, "configuration");
  });

  it("says so when a machine has nothing wrong, so every card on the shelf is the same height", () => {
    render(<MachineCard machine={machine()} onOpen={() => {}} />);
    expect(screen.getByText("machines.nothingToFix")).toBeInTheDocument();
    expect(screen.queryByTestId("machines-issues-machine-1")).not.toBeInTheDocument();
  });

  it("starts a session on the selected machine and opens the detail sheet separately", () => {
    const onStartSession = vi.fn();
    const onOpen = vi.fn();
    const item = machine();
    render(<MachineCard machine={item} onStartSession={onStartSession} onOpen={onOpen} />);

    fireEvent.click(screen.getByTestId("machines-start-session-machine-1"));
    expect(onStartSession).toHaveBeenCalledWith(item);

    fireEvent.click(screen.getByTestId("machines-details-machine-1"));
    expect(onOpen).toHaveBeenCalledWith(item);
  });
});
