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

  it("renders unreachable remote state and its recovery action", () => {
    render(<MachineCard machine={machine({ available: false, label: "Offline host" })} onManage={() => {}} />);
    expect(screen.getByText("machines.statusNotResponding")).toBeInTheDocument();
    expect(screen.getByText("machines.reconnect")).toBeInTheDocument();
  });

  it("renders typed configuration drift from the machine projection", () => {
    render(<MachineCard machine={{ ...machine(), drift: [{ kind: "capability", name: "ai-cli:codex", reason: "required capability is not reported by the node" }] }} />);
    expect(screen.getByTestId("machines-drift-machine-1")).toHaveTextContent("Configuration drift");
    expect(screen.getByText("ai-cli:codex: required capability is not reported by the node")).toBeInTheDocument();
  });

  it("starts a session on the selected machine while keeping management available", () => {
    const onStartSession = vi.fn();
    const item = machine();
    render(<MachineCard machine={item} onStartSession={onStartSession} onManage={() => {}} />);

    fireEvent.click(screen.getByTestId("machines-start-session-machine-1"));

    expect(onStartSession).toHaveBeenCalledWith(item);
    expect(screen.getByTestId("machines-manage-machine-1")).toBeInTheDocument();
  });
});
