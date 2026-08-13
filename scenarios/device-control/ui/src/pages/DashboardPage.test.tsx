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
}));

vi.mock("../api/deviceControl", () => api);
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
});
