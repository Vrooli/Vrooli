import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
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
import { fetchNotificationCatalog } from "../api/preferences";

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

// Keep the event-integration and notification-preference surfaces quiet so the
// browser-push setup flow is the only thing driving a role="status" node.
vi.mock("../api/integrations", () => ({
  fetchEventIntegrationConfig: vi.fn().mockResolvedValue({
    events_api_base: "",
    webhook_url: "",
    pattern: "incident.*",
    sensitivity_by_severity: {},
  }),
  updateEventIntegrationConfig: vi.fn(),
}));

vi.mock("../api/preferences", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/preferences")>();
  return {
    ...actual,
    // A populated catalog settles into the grouped list (no role="status"),
    // unlike the loading/empty/error states which intentionally announce.
    fetchNotificationCatalog: vi.fn().mockResolvedValue({
      sources: [
        {
          source_scenario: "git-control-tower",
          display_name: "Git Control Tower",
          icon: "",
          deep_link_base: "",
          declared: true,
          categories: [
            {
              category: "review",
              display_name: "Review requests",
              description: "A PR needs your review.",
              declared: true,
              default_enabled: true,
              default_min_urgency: "normal",
            },
          ],
        },
      ],
      preferences: [],
    }),
    updateNotificationPreference: vi.fn(),
  };
});

const listDevices = vi.mocked(recipientsClient.listDevices);
const registerPush = vi.mocked(registerBrowserPushSubscription);

const populatedCatalog = {
  sources: [
    {
      source_scenario: "git-control-tower",
      display_name: "Git Control Tower",
      icon: "",
      deep_link_base: "",
      declared: true,
      categories: [
        {
          category: "review",
          display_name: "Review requests",
          description: "A PR needs your review.",
          declared: true,
          default_enabled: true,
          default_min_urgency: "normal",
        },
      ],
    },
  ],
  preferences: [],
};

describe("SettingsPage browser notification setup", () => {
  beforeEach(() => {
    // afterEach's restoreAllMocks wipes the factory's resolved value, so the
    // catalog must be re-settled each test to keep the preferences section in
    // its populated state (which emits no role="status") rather than erroring.
    vi.mocked(fetchNotificationCatalog).mockResolvedValue(populatedCatalog);
  });

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
