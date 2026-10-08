import { afterEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";
import { strings } from "../consts/strings";
import { SettingsPage } from "./SettingsPage";
import {
  identityClient,
  recipientsClient,
  registerBrowserPushSubscription,
} from "../api/notifications";

vi.mock("../api/notifications", () => ({
  identityClient: {
    getSession: vi.fn(() =>
      Promise.resolve({
        signedIn: true,
        subject: "owner-1",
        email: "owner@example.test",
      }),
    ),
    login: vi.fn(),
    logout: vi.fn(),
  },
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
    Object.defineProperty(navigator, "serviceWorker", {
      configurable: true,
      value: undefined,
    });
    Object.defineProperty(window, "PushManager", {
      configurable: true,
      value: undefined,
    });
  });

  it("explains when the current origin cannot register push", async () => {
    Object.defineProperty(navigator, "serviceWorker", { configurable: true, value: {} });
    Object.defineProperty(window, "PushManager", { configurable: true, value: function PushManager() {} });
    renderWithProviders(<SettingsPage />);

    expect(await screen.findByText(/iPhone/)).toBeInTheDocument();
    await userEvent.click(
      screen.getByRole("button", { name: "Enable browser notifications" }),
    );

    expect(await screen.findByRole("status")).toHaveTextContent(
      "This hub has no push key configured.",
    );
    expect(registerPush).not.toHaveBeenCalled();
  });

  it("tells an iPhone Safari tab to install the app before enabling push", async () => {
    vi.stubEnv("VITE_VAPID_PUBLIC_KEY", "AQID");
    Reflect.deleteProperty(window, "PushManager");
    renderWithProviders(<SettingsPage />);

    await userEvent.click(
      await screen.findByRole("button", { name: "Enable browser notifications" }),
    );

    expect(await screen.findByRole("status")).toHaveTextContent(
      "Add to Home Screen",
    );
    expect(registerPush).not.toHaveBeenCalled();
  });

  it("registers a configured browser and refreshes its device projection", async () => {
    vi.stubEnv("VITE_VAPID_PUBLIC_KEY", "AQID");
    Object.defineProperty(navigator, "serviceWorker", {
      configurable: true,
      value: {},
    });
    Object.defineProperty(window, "PushManager", {
      configurable: true,
      value: function PushManager() {},
    });
    registerPush.mockResolvedValue({
      endpoint: "https://push.example/subscription",
    });

    renderWithProviders(<SettingsPage />);
    await userEvent.click(
      await screen.findByRole("button", {
        name: "Enable browser notifications",
      }),
    );

    await waitFor(() =>
      expect(registerPush).toHaveBeenCalledWith(expect.any(Uint8Array)),
    );
    expect(await screen.findByRole("status")).toHaveTextContent(
      "This browser is registered for Web Push.",
    );
    // Initial load, the tap-time key read (the cached list carried no key), and the post-registration refresh.
    expect(listDevices).toHaveBeenCalledTimes(3);
  });

  it("uses the runtime public key returned by the recipient surface", async () => {
    vi.stubEnv("VITE_VAPID_PUBLIC_KEY", "");
    vi.mocked(recipientsClient.listDevices).mockResolvedValueOnce({
      devices: [],
      vapidPublicKey: "AQID",
    } as never);
    Object.defineProperty(navigator, "serviceWorker", {
      configurable: true,
      value: {},
    });
    Object.defineProperty(window, "PushManager", {
      configurable: true,
      value: function PushManager() {},
    });
    registerPush.mockResolvedValue({
      endpoint: "https://push.example/subscription",
    });

    renderWithProviders(<SettingsPage />);
    await userEvent.click(
      await screen.findByRole("button", {
        name: "Enable browser notifications",
      }),
    );

    await waitFor(() =>
      expect(registerPush).toHaveBeenCalledWith(expect.any(Uint8Array)),
    );
    expect(await screen.findByRole("status")).toHaveTextContent(
      "This browser is registered for Web Push.",
    );
  });

  it("surfaces both Error and non-Error provider failures", async () => {
    vi.stubEnv("VITE_VAPID_PUBLIC_KEY", "AQID");
    Object.defineProperty(navigator, "serviceWorker", {
      configurable: true,
      value: {},
    });
    Object.defineProperty(window, "PushManager", {
      configurable: true,
      value: function PushManager() {},
    });
    registerPush.mockRejectedValueOnce(new Error("permission denied"));

    renderWithProviders(<SettingsPage />);
    await userEvent.click(
      await screen.findByRole("button", {
        name: "Enable browser notifications",
      }),
    );
    expect(await screen.findByRole("status")).toHaveTextContent(
      "permission denied",
    );

    registerPush.mockRejectedValueOnce("provider unavailable");
    await userEvent.click(
      screen.getByRole("button", { name: "Enable browser notifications" }),
    );
    expect(await screen.findByRole("status")).toHaveTextContent(
      "Push registration failed.",
    );
  });

  it("asks a signed-out browser to sign in before it can enable notifications", async () => {
    vi.mocked(identityClient.getSession).mockResolvedValueOnce({
      signedIn: false,
    } as never);

    renderWithProviders(<SettingsPage />);

    expect(
      await screen.findByRole("button", { name: strings.session.submit }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Enable browser notifications" }),
    ).not.toBeInTheDocument();
  });
});
