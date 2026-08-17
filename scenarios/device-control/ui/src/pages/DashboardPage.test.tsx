import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";

const api = vi.hoisted(() => ({
  acquireSession: vi.fn(),
  connectDevice: vi.fn(),
  killSession: vi.fn(),
  listDevices: vi.fn(),
  listSessions: vi.fn(),
  listStrategies: vi.fn(),
  listAuthProfiles: vi.fn(),
  getAuthProfile: vi.fn(),
}));

vi.mock("../api/deviceControl", () => api);
vi.mock("../api/authentication", () => ({
  listAuthProfiles: api.listAuthProfiles,
  getAuthProfile: api.getAuthProfile,
}));
vi.mock("../features/health/HealthCard", () => ({
  HealthCard: () => <div data-testid="health-card" />,
}));

import { DashboardPage } from "./DashboardPage";

const device = {
  id: "android-phone-1",
  name: "Galaxy A03s",
  kind: "physical",
  serial: "R9TT608Q6MH",
  model: "SM_A037U",
  os_version: "Android 13",
  transport: "usb",
  strategy_id: "android-adb",
  status: "available",
  health: "ready",
  health_reason: "ADB transport is ready",
  capabilities: [{ name: "screenshot", status: "available", next_action: "Ready" }],
};

beforeEach(() => {
  vi.clearAllMocks();
  api.listDevices.mockResolvedValue({ devices: [device] });
  api.listStrategies.mockResolvedValue({ strategies: [] });
  api.listSessions.mockResolvedValue({ sessions: [] });
  api.killSession.mockResolvedValue({});
  api.listAuthProfiles.mockResolvedValue({ profiles: [] });
});

describe("DashboardPage", () => {
  it("renders every onboarding rung returned by a live re-probe", async () => {
    api.connectDevice.mockResolvedValue({
      kind: "android",
      first_next_action: "Enable USB debugging and authorize this computer.",
      rungs: [
        { id: "data-cable", status: "available", next_action: "Ready" },
        { id: "file-transfer", status: "unavailable", next_action: "Choose File Transfer on the phone." },
        { id: "rsa-authorized", status: "available", next_action: "Ready" },
      ],
    });

    const user = userEvent.setup();
    renderWithProviders(<DashboardPage />);
    await screen.findByText(device.id);
    const fleet = screen.getByTestId(selectors.pages.dashboardDevices);
    await user.click(within(fleet).getByRole("button", { name: strings.pages.dashboard.reprobe }));

    const report = await screen.findByTestId(selectors.pages.onboardingReport);
    expect(report).toHaveTextContent("data-cable");
    expect(report).toHaveTextContent("available");
    expect(report).toHaveTextContent("file-transfer");
    expect(report).toHaveTextContent("Choose File Transfer on the phone.");
    expect(report).toHaveTextContent("rsa-authorized");
    expect(api.connectDevice).toHaveBeenCalledWith("android");
  });

  it("offers onboarding from the zero-device state", async () => {
    api.listDevices.mockResolvedValue({ devices: [] });
    api.connectDevice.mockResolvedValue({
      kind: "android",
      first_next_action: "Attach and authorize an Android device.",
      rungs: [{ id: "usb-bus", status: "unavailable", next_action: "Attach the device over USB." }],
    });

    const user = userEvent.setup();
    renderWithProviders(<DashboardPage />);

    await user.click(await screen.findByTestId(selectors.pages.dashboardEmptyReprobe));
    expect(api.connectDevice).toHaveBeenCalledWith("android");
    expect(await screen.findByTestId(selectors.pages.onboardingReport)).toHaveTextContent("usb-bus");
  });

  it("kills a live lease from the dashboard controls", async () => {
    const session = { id: "lease-1", device_id: device.id, actor: "operator", state: "held", expires_at: "later", created_at: "now" };
    api.listSessions.mockResolvedValueOnce({ sessions: [session] }).mockResolvedValue({ sessions: [] });
    const user = userEvent.setup();
    renderWithProviders(<DashboardPage />);

    await user.click(await screen.findByRole("button", { name: strings.pages.dashboard.killSession }));
    await waitFor(() => expect(api.killSession).toHaveBeenCalledWith(session.id));
  });

  it("shows the lease control for an Android physical device", async () => {
    api.acquireSession.mockResolvedValue({ session: { id: "lease-2", device_id: device.id, actor: "browser-operator", state: "held", expires_at: "later", created_at: "now" } });
    const user = userEvent.setup();
    renderWithProviders(<DashboardPage />);

    await user.click(await screen.findByTestId(selectors.pages.dashboardAcquire));
    expect(api.acquireSession).toHaveBeenCalledWith(device.id, "browser-operator");
  });

  it("keeps inventory visible while an optional auth probe is unavailable", async () => {
    api.listAuthProfiles.mockRejectedValue(new Error("auth provider unavailable"));
    renderWithProviders(<DashboardPage />);
    expect(await screen.findByText(device.id)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.pages.dashboardDevices)).toBeInTheDocument();
  });

  it("renders provider-backed authentication metadata alongside inventory", async () => {
    api.listAuthProfiles.mockResolvedValue({
      profiles: [{
        id: "profile-1",
        device_id: device.id,
        method: "pin",
        credential_identity: "device-control/profile-1",
        credential_field: "unlock",
        verification: "fresh_lock_state_unlocked",
        policy: { max_attempts: 1, attempt_limit: 15, settle: 1 },
        status: "active",
        created_at: "now",
        updated_at: "now",
      }],
    });
    api.getAuthProfile.mockResolvedValue({ provider: { provider: "test", provider_state: "available", configured: true } });
    renderWithProviders(<DashboardPage />);
    expect(await screen.findByText(/profile-1/)).toBeInTheDocument();
    expect(api.getAuthProfile).toHaveBeenCalledWith("profile-1");
  });

  it("renders each transport profile for a merged device identity", async () => {
    api.listDevices.mockResolvedValue({
      devices: [{
        ...device,
        transports: [
          { strategy_id: "android-adb", name: "usb", endpoint: "R9TT608Q6MH", health: "ready", capabilities: { input: { name: "input", status: "available" } } },
          { strategy_id: "android-tv-remote", name: "mdns", endpoint: "living-room", health: "unavailable", health_reason: "pairing required", capabilities: { media: { name: "media", status: "available" } } },
        ],
      }],
    });
    renderWithProviders(<DashboardPage />);

    const transports = await screen.findByTestId(`device-transports-${device.id}`);
    expect(transports).toHaveTextContent("android-adb · usb");
    expect(transports).toHaveTextContent("android-tv-remote · mdns");
    expect(transports).toHaveTextContent("pairing required");
    expect(transports).toHaveTextContent("media: available");
  });

  it("renders unavailable-device fallbacks and non-promotable strategy state", async () => {
    api.listDevices.mockResolvedValue({
      devices: [{
        ...device,
        id: "offline-emulator",
        name: "Offline emulator",
        kind: "emulator",
        status: "offline",
        model: "",
        serial: "",
        os_version: "",
        transport: "",
        health: "",
        health_reason: "",
        capabilities: "not-an-array",
      }],
    });
    api.listStrategies.mockResolvedValue({ strategies: [{ id: "unknown", description: "", status: "unknown", tiers: [], capabilities: {}, promotable: false }] });
    renderWithProviders(<DashboardPage />);
    expect(await screen.findByText(/offline-emulator/)).toBeInTheDocument();
    expect(screen.getByText(/Offline emulator/)).toBeInTheDocument();
    const fleet = screen.getByTestId(selectors.pages.dashboardDevices);
    expect(fleet).toHaveTextContent("pages.dashboard.osUnavailable");
    expect(fleet).toHaveTextContent("pages.dashboard.localTransport");
    expect(screen.getByText(/pages\.dashboard\.unknown/)).toBeInTheDocument();
    expect(screen.getByText(/^pages\.dashboard\.no$/)).toBeInTheDocument();
  });

  it("surfaces probe and lease failures as actionable dashboard errors", async () => {
    api.connectDevice.mockRejectedValue(new Error("probe failed"));
    api.acquireSession.mockRejectedValue(new Error("lease failed"));
    const user = userEvent.setup();
    renderWithProviders(<DashboardPage />);
    const fleet = await screen.findByTestId(selectors.pages.dashboardDevices);
    await user.click(within(fleet).getByTestId(selectors.pages.dashboardReprobe));
    expect(await screen.findByRole("alert")).toHaveTextContent("probe failed");
    await user.click(within(fleet).getByTestId(selectors.pages.dashboardAcquire));
    expect(await screen.findByRole("alert")).toHaveTextContent("lease failed");
  });
});
