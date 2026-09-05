import { afterEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";
import { SettingsPage } from "./SettingsPage";
import { recipientsClient, registerBrowserPushSubscription } from "../api/notifications";

vi.mock("../api/notifications", () => ({
  recipientsClient: {
    listDevices: vi.fn().mockResolvedValue({
      devices: [{ id: "phone", name: "iPhone", channels: ["web_push"] }],
    }),
  },
  registerBrowserPushSubscription: vi.fn(),
}));

const listDevices = vi.mocked(recipientsClient.listDevices);
const registerPush = vi.mocked(registerBrowserPushSubscription);

describe("SettingsPage browser notification setup", () => {
  afterEach(() => {
    vi.unstubAllEnvs();
    vi.restoreAllMocks();
    Object.defineProperty(navigator, "serviceWorker", { configurable: true, value: undefined });
    Object.defineProperty(window, "PushManager", { configurable: true, value: undefined });
  });

  it("explains when the current origin cannot register push", async () => {
    renderWithProviders(<SettingsPage />);

    expect(await screen.findByText(/iPhone/)).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Enable browser notifications" }));

    expect(await screen.findByRole("status")).toHaveTextContent("Push is not configured for this origin yet.");
    expect(registerPush).not.toHaveBeenCalled();
  });

  it("registers a configured browser and refreshes its device projection", async () => {
    vi.stubEnv("VITE_VAPID_PUBLIC_KEY", "AQID");
    Object.defineProperty(navigator, "serviceWorker", { configurable: true, value: {} });
    Object.defineProperty(window, "PushManager", { configurable: true, value: function PushManager() {} });
    registerPush.mockResolvedValue({ endpoint: "https://push.example/subscription" });

    renderWithProviders(<SettingsPage />);
    await userEvent.click(await screen.findByRole("button", { name: "Enable browser notifications" }));

    await waitFor(() => expect(registerPush).toHaveBeenCalledWith(expect.any(Uint8Array)));
    expect(await screen.findByRole("status")).toHaveTextContent("This browser is registered for Web Push.");
    expect(listDevices).toHaveBeenCalledTimes(2);
  });

  it("uses the runtime public key returned by the recipient surface", async () => {
    vi.stubEnv("VITE_VAPID_PUBLIC_KEY", "");
    vi.mocked(recipientsClient.listDevices).mockResolvedValueOnce({
      devices: [],
      vapidPublicKey: "AQID",
    } as never);
    Object.defineProperty(navigator, "serviceWorker", { configurable: true, value: {} });
    Object.defineProperty(window, "PushManager", { configurable: true, value: function PushManager() {} });
    registerPush.mockResolvedValue({ endpoint: "https://push.example/subscription" });

    renderWithProviders(<SettingsPage />);
    await userEvent.click(await screen.findByRole("button", { name: "Enable browser notifications" }));

    await waitFor(() => expect(registerPush).toHaveBeenCalledWith(expect.any(Uint8Array)));
    expect(await screen.findByRole("status")).toHaveTextContent("This browser is registered for Web Push.");
  });

  it("surfaces both Error and non-Error provider failures", async () => {
    vi.stubEnv("VITE_VAPID_PUBLIC_KEY", "AQID");
    Object.defineProperty(navigator, "serviceWorker", { configurable: true, value: {} });
    Object.defineProperty(window, "PushManager", { configurable: true, value: function PushManager() {} });
    registerPush.mockRejectedValueOnce(new Error("permission denied"));

    renderWithProviders(<SettingsPage />);
    await userEvent.click(await screen.findByRole("button", { name: "Enable browser notifications" }));
    expect(await screen.findByRole("status")).toHaveTextContent("permission denied");

    registerPush.mockRejectedValueOnce("provider unavailable");
    await userEvent.click(screen.getByRole("button", { name: "Enable browser notifications" }));
    expect(await screen.findByRole("status")).toHaveTextContent("Push registration failed.");
  });
});
