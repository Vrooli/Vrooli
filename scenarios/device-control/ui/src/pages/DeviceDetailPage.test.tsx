import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { Route, Routes } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { renderWithProviders } from "../test-utils";

const api = vi.hoisted(() => ({
  actuateDevice: vi.fn(),
  acquireSession: vi.fn(),
  describeDevice: vi.fn(),
  listSessions: vi.fn(),
  readDeviceState: vi.fn(),
  releaseSession: vi.fn(),
  watchDeviceEvents: vi.fn(),
}));

vi.mock("../api/deviceControl", () => api);

import { DeviceDetailPage } from "./DeviceDetailPage";

function renderPage(deviceId = "fixture-device") {
  return renderWithProviders(<Routes><Route path="/devices/:deviceId" element={<DeviceDetailPage />} /></Routes>, { routerEntries: [`/devices/${deviceId}`] });
}

beforeEach(() => {
  vi.clearAllMocks();
  api.acquireSession.mockResolvedValue({ session: { id: "lease-1", lease_token: "token" } });
  api.releaseSession.mockResolvedValue({});
  api.listSessions.mockResolvedValue({ sessions: [] });
  api.watchDeviceEvents.mockReturnValue(() => undefined);
});

describe("DeviceDetailPage capability composition", () => {
  it("renders a two-transport television remote and media bar without live view", async () => {
    api.describeDevice.mockResolvedValue({ device: { id: "tv", name: "Living Room", kind: "physical", strategy_id: "android-tv-remote", status: "available", identity_reason: "address-only-correlation-refused", claims: [{ kind: "bluetooth-mac", value: "AA:BB:CC:DD:EE:FF", strategy_id: "android-tv-remote", evidence: "observed" }], capabilities: [{ name: "input", status: "available" }, { name: "media", status: "available" }], transports: [{ strategy_id: "android-tv-remote", name: "mdns", health: "available", capabilities: { input: { name: "input", status: "available" } } }, { strategy_id: "google-cast", name: "cast", health: "available", capabilities: { media: { name: "media", status: "available" } } }] } });
    api.readDeviceState.mockResolvedValue({ properties: { application: { value: "YouTube", status: "available" }, player_state: { value: "PLAYING", status: "available" } } });
    renderPage("tv");
    expect(await screen.findByTestId(selectors.pages.deviceRemotePanel)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.pages.deviceMediaPanel)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.pages.deviceNowPlaying)).toHaveTextContent("YouTube");
    expect(screen.getByTestId(selectors.pages.deviceIdentityClaims)).toHaveTextContent("bluetooth-mac=AA:BB:CC:DD:EE:FF");
    expect(screen.getByTestId(selectors.pages.deviceIdentityReason)).toHaveTextContent("address-only-correlation-refused");
    expect(screen.queryByTestId(selectors.pages.deviceLiveView)).not.toBeInTheDocument();
  });

  it("renders a phone live view and no directional remote", async () => {
    api.describeDevice.mockResolvedValue({ device: { id: "phone", name: "Phone", kind: "physical", strategy_id: "android-adb", status: "available", capabilities: [{ name: "input", status: "available" }, { name: "screenshot", status: "available" }] } });
    api.readDeviceState.mockResolvedValue({ screen_state: "on" });
    renderPage("phone");
    expect(await screen.findByTestId(selectors.pages.deviceLiveView)).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.pages.deviceRemotePanel)).not.toBeInTheDocument();
  });

  it("generates property controls from descriptors", async () => {
    api.describeDevice.mockResolvedValue({ device: { id: "plug", name: "Plug", kind: "physical", strategy_id: "generic", status: "available", capabilities: [{ name: "property", status: "available" }], properties: [{ name: "enabled", value_type: "boolean", writable: true }, { name: "mode", value_type: "string", writable: true, enumeration: ["eco", "boost"] }, { name: "threshold", value_type: "number", writable: true, minimum: 0, maximum: 100 }] } });
    api.readDeviceState.mockResolvedValue({ properties: { enabled: { value: true, status: "available" }, mode: { value: "eco", status: "available" }, threshold: { value: 50, status: "available" } } });
    renderPage("plug");
    const panel = await screen.findByTestId(selectors.pages.devicePropertyPanel);
    expect(panel.querySelector('input[type="checkbox"]')).toBeInTheDocument();
    expect(panel.querySelector("select")).toBeInTheDocument();
    expect(panel.querySelector('input[type="range"]')).toBeInTheDocument();
  });

  it("drives remote, media, sensor, log, and every property control", async () => {
    api.describeDevice.mockResolvedValue({ device: { id: "fixture-device", name: "Receiver", kind: "physical", strategy_id: "fixture", status: "available", capabilities: [{ name: "input", status: "available" }, { name: "media", status: "available" }, { name: "property", status: "available" }, { name: "sensor", status: "available" }, { name: "device-logs", status: "available" }], properties: [{ name: "enabled", value_type: "boolean", writable: true }, { name: "mode", value_type: "string", writable: true, enumeration: ["eco", "boost"] }, { name: "threshold", value_type: "number", writable: true, minimum: 0, maximum: 100 }, { name: "label", value_type: "string", writable: true }] } });
    api.readDeviceState.mockResolvedValue({ properties: { enabled: { value: false, status: "available" }, mode: { value: "eco", status: "available" }, threshold: { value: 50, status: "available" }, label: { value: "Receiver", status: "available" }, temperature: { value: 22, status: "available" } }, unavailable: { screenshot: "not exposed" } });
    renderPage();
    expect(await screen.findByTestId(selectors.pages.deviceSensorPanel)).toHaveTextContent("temperature: 22");
    expect(screen.getByTestId(selectors.pages.deviceLogPanel)).toHaveTextContent("screenshot");
    const remotePanel = screen.getByTestId(selectors.pages.deviceRemotePanel);
    fireEvent.click(remotePanel.querySelector("button") as HTMLButtonElement);
    const remoteText = remotePanel.querySelector("input") as HTMLInputElement;
    fireEvent.change(remoteText, { target: { value: "hello" } });
    fireEvent.keyDown(remoteText, { key: "Enter" });
    const mediaPanel = screen.getByTestId(selectors.pages.deviceMediaPanel);
    fireEvent.click(mediaPanel.querySelector("button") as HTMLButtonElement);
    const propertyPanel = screen.getByTestId(selectors.pages.devicePropertyPanel);
    fireEvent.change(propertyPanel.querySelector('input[aria-label="enabled"]') as HTMLInputElement, { target: { checked: true } });
    fireEvent.change(propertyPanel.querySelector('select[aria-label="mode"]') as HTMLSelectElement, { target: { value: "boost" } });
    fireEvent.change(propertyPanel.querySelector('input[aria-label="threshold"]') as HTMLInputElement, { target: { value: "75" } });
    const label = propertyPanel.querySelector('input[aria-label="label"]') as HTMLInputElement;
    fireEvent.change(label, { target: { value: "New receiver" } });
    fireEvent.blur(label);
    await waitFor(() => expect(api.actuateDevice).toHaveBeenCalled());
    expect(api.acquireSession).toHaveBeenCalled();
    expect(api.releaseSession).toHaveBeenCalled();
  });

  it("renders loading, request failure, unreachable, and stale probe states", async () => {
    renderPage("fixture-device?fixture=loading");
    expect((await screen.findAllByTestId(selectors.pages.deviceDetail)).at(-1)).toHaveTextContent("Loading device");

    api.describeDevice.mockRejectedValueOnce(new Error("backend unavailable"));
    renderPage("fixture-device?fixture=request-error");
    expect((await screen.findAllByTestId(selectors.pages.deviceDetail)).at(-1)).toHaveTextContent("Device request failed");

    api.describeDevice.mockReset();
    api.describeDevice.mockRejectedValueOnce("non-error failure");
    renderPage("fixture-device");
    expect((await screen.findAllByTestId(selectors.pages.deviceDetail)).at(-1)).toHaveTextContent("Device request failed");

    api.describeDevice.mockReset();
    api.describeDevice.mockResolvedValueOnce({ device: { id: "fixture-device", name: "Offline", kind: "physical", strategy_id: "fixture", status: "available", capabilities: [] } });
    api.readDeviceState.mockResolvedValueOnce({});
    api.listSessions.mockRejectedValueOnce(new Error("session lookup unavailable"));
    renderPage("fixture-device?fixture=unreachable");
    expect(await screen.findByTestId(selectors.pages.deviceUnreachableReason)).toHaveTextContent("Host node is offline");
    expect(screen.queryByTestId(selectors.pages.deviceProbeNow)).not.toBeInTheDocument();

    api.describeDevice.mockResolvedValueOnce({ device: { id: "fixture-device", name: "Receiver", kind: "physical", strategy_id: "fixture", status: "available", capabilities: [] } });
    api.readDeviceState.mockResolvedValueOnce({});
    renderPage("fixture-device?fixture=stale");
    expect((await screen.findAllByTestId(selectors.pages.deviceDetail)).at(-1)).toHaveTextContent("Showing the last known snapshot while a probe is in flight.");
  });

  it("shows fallback transport metadata, capability reasons, and an active lease", async () => {
    api.describeDevice.mockResolvedValue({ device: { id: "fixture-device", serial: "SERIAL-1", kind: "physical", status: "available", health: "available", strategy_id: "fixture", transport: "mdns", capabilities: [{ name: "input", status: "unsupported", reason: "not exposed" }, { name: "media", status: "unavailable", prerequisite: "pairing required" }, { name: "sensor", status: "failed", next_action: "retry probe" }], transports: [{ strategy_id: "fixture", name: "bare", health: "available" }] } });
    api.readDeviceState.mockResolvedValue({});
    api.listSessions.mockResolvedValue({ sessions: [{ id: "lease-1", device_id: "fixture-device", actor: "operator", state: "held", expires_at: "later" }] });
    renderPage();
    const detail = await screen.findByTestId(selectors.pages.deviceDetail);
    expect(detail).toHaveTextContent("SERIAL-1");
    expect(detail).toHaveTextContent("not exposed");
    expect(detail).toHaveTextContent("pairing required");
    expect(detail).toHaveTextContent("retry probe");
    expect(screen.getByTestId(selectors.pages.deviceSessionHistory)).toHaveTextContent("Active lease held by operator");
    expect(screen.getByTestId(selectors.pages.deviceTransport)).toHaveTextContent("fixture · bare");
  });

  it("uses the generic reason when an unreachable device has no cause", async () => {
    api.describeDevice.mockResolvedValue({ device: { id: "fixture-device", name: "Offline", kind: "physical", status: "unreachable", health: "unreachable", strategy_id: "fixture", capabilities: [] } });
    api.readDeviceState.mockResolvedValue({});
    renderPage();
    expect(await screen.findByTestId(selectors.pages.deviceUnreachableReason)).toHaveTextContent("The device is unreachable.");
  });
});
