import { renderWithProviders as render } from "../../../test-utils";
import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import MachineCard from "../MachineCard";
import type { Machine } from "../../../api/machines";

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
  };
}

describe("MachineCard", () => {
  it("uses the machine silhouette for a remote machine", () => {
    const { container } = render(<MachineCard machine={machine()} />);
    expect(screen.getByTestId("fleet-card-machine-machine-1")).toBeInTheDocument();
    expect(container.querySelector("[data-testid=machine-silhouette]")).toBeInTheDocument();
    expect(container.querySelector("[data-testid=device-silhouette]")).toBeNull();
  });

  it("uses the local laptop artwork for the local machine", () => {
    const { container } = render(<MachineCard machine={machine({ kind: "local", label: "This computer" })} />);
    expect(container.querySelector("[data-testid=machine-silhouette]")).toBeInTheDocument();
    expect(screen.getByText("machines.thisComputer")).toBeInTheDocument();
  });

  it("renders unreachable remote state and its recovery action", () => {
    render(<MachineCard machine={machine({ available: false, label: "Offline host" })} onManage={() => {}} />);
    expect(screen.getByText("machines.statusNotResponding")).toBeInTheDocument();
    expect(screen.getByText("machines.reconnect")).toBeInTheDocument();
  });
});
