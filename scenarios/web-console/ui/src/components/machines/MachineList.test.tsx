import { fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders as render } from "../../test-utils";
import { setLocale } from "../../i18n";
import type { Fleet, JoinRequest, Machine } from "../../api/machines";
import MachineList from "./MachineList";

function machine(overrides: Partial<Machine> & { target?: Partial<Machine["target"]> } = {}): Machine {
  return {
    target: {
      id: overrides.target?.id ?? "bridge-node:alpha",
      kind: overrides.target?.kind ?? "bridge-node",
      label: overrides.target?.label ?? "alpha",
      available: overrides.target?.available ?? true,
      os: overrides.target?.os ?? "linux",
      arch: overrides.target?.arch ?? "amd64",
      readiness: overrides.target?.readiness ?? [],
      recovery_action: overrides.target?.recovery_action,
    },
    grant: overrides.grant ?? { summary: "Read terminal", effects: ["read"], appCount: 1, coversAllApps: false, scopes: [], preset: "read" },
    heartbeatAgeSeconds: overrides.heartbeatAgeSeconds ?? 4,
    manageable: overrides.manageable ?? true,
    drift: overrides.drift ?? [],
  } as Machine;
}

function joinRequest(): JoinRequest {
  return { id: "req-1", name: "laptop", os: "macos", arch: "arm64", endpoint: "10.0.0.2", confirmationWords: ["alpha", "beta"], keyFingerprint: "fp", requestedAgeSeconds: 12 };
}

function fleet(machines: Machine[], joinRequests: JoinRequest[] = []): Fleet {
  return {
    status: "ready",
    machines,
    joinRequests,
    presets: [],
    message: "",
    recoveryAction: "",
    controlPlane: { reachable: true, endpoint: "", detail: "", consoleUrl: "" },
  };
}

describe("MachineList", () => {
  beforeEach(async () => { await setLocale("en"); });

  it("renders a reachable manageable machine with a manage control", () => {
    const onManage = vi.fn();
    render(
      <MachineList
        fleet={fleet([machine()])}
        loading={false}
        refreshing={false}
        onRefresh={vi.fn()}
        onAdd={vi.fn()}
        onManage={onManage}
        onReview={vi.fn()}
      />,
    );
    expect(screen.getByTestId("machines-row-bridge-node-alpha")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("machines-manage-bridge-node-alpha"));
    expect(onManage).toHaveBeenCalledTimes(1);
  });

  it("renders the local, never-responded, and not-responding status pills", () => {
    render(
      <MachineList
        fleet={fleet([
          machine({ target: { id: "local", kind: "local", label: "this", available: false }, heartbeatAgeSeconds: 0, manageable: false }),
          machine({ target: { id: "bridge-node:new", label: "new", available: false }, heartbeatAgeSeconds: 0 }),
          machine({ target: { id: "bridge-node:lost", label: "lost", available: false, recovery_action: "Restart the agent" }, heartbeatAgeSeconds: 90 }),
        ])}
        loading={false}
        refreshing={false}
        onRefresh={vi.fn()}
        onAdd={vi.fn()}
        onManage={vi.fn()}
        onReview={vi.fn()}
      />,
    );
    // The lost machine surfaces its recovery action inline.
    expect(screen.getByText("Restart the agent")).toBeInTheDocument();
    // The local machine is not manageable, so it renders no manage button.
    expect(screen.queryByTestId("machines-manage-local")).not.toBeInTheDocument();
  });

  it("renders a join-request banner and forwards review", () => {
    const onReview = vi.fn();
    render(
      <MachineList
        fleet={fleet([], [joinRequest()])}
        loading={false}
        refreshing={false}
        onRefresh={vi.fn()}
        onAdd={vi.fn()}
        onManage={vi.fn()}
        onReview={onReview}
      />,
    );
    fireEvent.click(screen.getByTestId("machines-review-req-1"));
    expect(onReview).toHaveBeenCalledWith(expect.objectContaining({ id: "req-1" }));
  });

  it("forwards refresh and add, and reflects the refreshing state as disabled", () => {
    const onRefresh = vi.fn();
    const onAdd = vi.fn();
    render(
      <MachineList
        fleet={fleet([])}
        loading={false}
        refreshing
        onRefresh={onRefresh}
        onAdd={onAdd}
        onManage={vi.fn()}
        onReview={vi.fn()}
      />,
    );
    const refresh = screen.getByTestId("machines-refresh");
    expect(refresh).toBeDisabled();
    fireEvent.click(screen.getByTestId("machines-add"));
    expect(onAdd).toHaveBeenCalledTimes(1);
  });

  it("shows the empty shelf when no remote machines are linked", () => {
    render(
      <MachineList
        fleet={fleet([])}
        loading={false}
        refreshing={false}
        onRefresh={vi.fn()}
        onAdd={vi.fn()}
        onManage={vi.fn()}
        onReview={vi.fn()}
      />,
    );
    expect(screen.getByTestId("machines-empty")).toBeInTheDocument();
  });

  it("shows the loading placeholder while the first fleet load is in flight", () => {
    render(
      <MachineList
        fleet={undefined}
        loading
        refreshing={false}
        onRefresh={vi.fn()}
        onAdd={vi.fn()}
        onManage={vi.fn()}
        onReview={vi.fn()}
      />,
    );
    expect(screen.getByTestId("machines-loading")).toBeInTheDocument();
  });
});
