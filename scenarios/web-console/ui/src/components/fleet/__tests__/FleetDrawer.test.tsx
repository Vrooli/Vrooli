import { renderWithProviders as render } from "../../../test-utils";
import { fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { setLocale } from "../../../i18n";
import type { JoinRequest, Machine } from "../../../api/machines";

const fleetMock = vi.hoisted(() => ({
  data: { machines: [] as Machine[], joinRequests: [] as JoinRequest[], presets: [], controlPlane: { reachable: false, endpoint: "", consoleUrl: "" } },
  isLoading: false,
  isFetching: false,
  refetch: vi.fn(),
}));

vi.mock("../../../hooks/useFleet", () => ({
  useFleet: () => fleetMock,
  useFleetMutations: () => ({ issueCode: { mutate: vi.fn(), isPending: false }, decide: { mutate: vi.fn(), isPending: false }, setGrant: { mutate: vi.fn(), isPending: false } }),
}));
vi.mock("../../../hooks/useDevices", () => ({
  useDevices: () => ({ data: [], isLoading: false }),
  useDeviceMutations: () => ({ disconnect: { mutate: vi.fn() }, giveControl: { mutate: vi.fn() }, rename: { mutate: vi.fn() } }),
}));

import FleetDrawer from "../FleetDrawer";

describe("FleetDrawer", () => {
  beforeEach(async () => { await setLocale("en"); });

  it("renders device and machine populations on separate labelled rails", () => {
    render(<FleetDrawer open onClose={vi.fn()} />);
    expect(screen.getByTestId("fleet-rail-devices")).toBeInTheDocument();
    expect(screen.getByTestId("fleet-rail-machines")).toBeInTheDocument();
    expect(screen.getByText("Devices")).toBeInTheDocument();
    expect(screen.getByText("Machines")).toBeInTheDocument();
  });

  it("places pending join requests before linked machines in the machines rail", () => {
    fleetMock.data = {
      ...fleetMock.data,
      joinRequests: [{ id: "request-1", name: "Remote", os: "linux", arch: "x64", endpoint: "", confirmationWords: ["river", "cedar"], keyFingerprint: "fp", requestedAgeSeconds: 20 }],
    };
    render(<FleetDrawer open onClose={vi.fn()} />);
    const rail = screen.getByTestId("fleet-rail-machines");
    expect(rail.querySelector("[data-testid='machines-join-request-request-1']")).toBeInTheDocument();
    fleetMock.data.joinRequests = [];
  });

  it("forwards a machine card's start-session action", () => {
    const machine: Machine = {
      target: { id: "machine-2", kind: "bridge-node", label: "Build host", available: true },
      grant: { summary: "Read terminal", effects: ["read"], appCount: 1, coversAllApps: false, scopes: [], preset: "read" },
      heartbeatAgeSeconds: 2,
      manageable: true,
    };
    fleetMock.data = { ...fleetMock.data, machines: [machine] };
    const onStartSession = vi.fn();

    render(<FleetDrawer open onClose={vi.fn()} onStartSession={onStartSession} />);
    fireEvent.click(screen.getByTestId("machines-start-session-machine-2"));

    expect(onStartSession).toHaveBeenCalledWith(machine);
    fleetMock.data = { ...fleetMock.data, machines: [] };
  });
});
