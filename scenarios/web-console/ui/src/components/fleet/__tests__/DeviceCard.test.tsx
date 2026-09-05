import { renderWithProviders as render } from "../../../test-utils";
import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import DeviceCard, { type RosterDevice } from "../DeviceCard";

function device(overrides: Partial<RosterDevice> = {}): RosterDevice {
  return {
    deviceId: "device-1",
    deviceLabel: "Phone",
    deviceClass: "phone",
    connectionCount: 1,
    isSelf: false,
    reconnecting: false,
    sessions: [],
    ...overrides,
  };
}

describe("DeviceCard", () => {
  it.each([
    ["in-control", { sessions: [{ sessionId: "session-1", holdsLease: true }] }],
    ["following", { sessions: [{ sessionId: "session-1", holdsLease: false }] }],
    ["idle", { connectionCount: 1 }],
    ["reconnecting", { reconnecting: true, sessions: [{ sessionId: "session-1", holdsLease: true }] }],
    ["not-connected", { connectionCount: 0 }],
  ])("renders the %s state", (_name, overrides) => {
    render(<DeviceCard device={device(overrides)} />);
    expect(screen.getByTestId("fleet-card-device-device-1")).toBeInTheDocument();
  });

  it("marks the caller's device and keeps the declared artwork population", () => {
    const { container } = render(<DeviceCard device={device({ isSelf: true, deviceClass: "tablet" })} />);
    expect(screen.getByText("fleet.you")).toBeInTheDocument();
    expect(container.querySelector("svg")).toBeInTheDocument();
    expect(container.querySelector("[data-testid=machine-silhouette]")).toBeNull();
  });

  it("lights the screen only while the device owns a lease", () => {
    const { rerender } = render(<DeviceCard device={device({ sessions: [{ sessionId: "session-1", holdsLease: true }] })} />);
    expect(screen.getByTestId("device-silhouette")).toHaveAttribute("data-screen-lit", "true");
    rerender(<DeviceCard device={device({ sessions: [{ sessionId: "session-1", holdsLease: false }] })} />);
    expect(screen.getByTestId("device-silhouette")).toHaveAttribute("data-screen-lit", "false");
  });

  it("offers control transfer for a following device and identifies its session", () => {
    const onGiveControl = vi.fn();
    render(<DeviceCard device={device({ sessions: [{ sessionId: "session-1", holdsLease: false }] })} onGiveControl={onGiveControl} />);
    screen.getByRole("button", { name: "fleet.giveControl" }).click();
    expect(onGiveControl).toHaveBeenCalledWith(expect.objectContaining({ deviceId: "device-1" }));
  });
});
