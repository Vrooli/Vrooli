import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createMemoryRouter, MemoryRouter, Route, Routes, RouterProvider } from "react-router-dom";
import { QueryClient } from "@tanstack/react-query";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { setLocale } from "../i18n";
import { ReachPage } from "./ReachPage";
import { SettingsPage } from "./SettingsPage";
import {
  identityClient,
  recipientsClient,
  registerBrowserPushSubscription,
} from "../api/notifications";
import { fetchNotificationCatalog } from "../api/preferences";
import { readConnectionStatus, type ConnectionStatus } from "../api/serviceWorker";

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
    listQuietWindows: vi.fn().mockResolvedValue({ windows: [] }),
    setQuietWindow: vi.fn().mockResolvedValue({ id: "quiet-1" }),
    deleteQuietWindow: vi.fn().mockResolvedValue({}),
    getEscalationChain: vi.fn().mockResolvedValue({ steps: [] }),
    setEscalationChain: vi.fn().mockResolvedValue({ steps: [] }),
    listDevices: vi.fn().mockResolvedValue({
      devices: [{ id: "phone", name: "iPhone", machineId: "local", channels: ["web_push"] }],
      vapidPublicKey: "AQID",
    }),
    removeDevice: vi.fn().mockResolvedValue({}),
    upsertChannelAddress: vi.fn().mockResolvedValue({ id: "email-1" }),
    listChannelAddresses: vi.fn().mockResolvedValue({ addresses: [] }),
    createChannelAddress: vi.fn().mockResolvedValue({ id: "email-1" }),
    updateChannelAddress: vi.fn().mockResolvedValue({ id: "email-1" }),
  },
  registerBrowserPushSubscription: vi.fn(),
}));

vi.mock("../api/serviceWorker", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/serviceWorker")>();
  return {
    ...actual,
    readConnectionStatus: vi.fn(),
  };
});

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
const readStatus = vi.mocked(readConnectionStatus);

const connectionStatus = (subscribed: boolean): ConnectionStatus => ({
  supported: true,
  permission: "granted",
  controlled: true,
  runningVersion: "2026-10-14.5",
  availableVersion: "2026-10-14.5",
  stale: false,
  subscribed,
  endpointHost: subscribed ? "web.push.apple.com" : "",
});

const setAuthoritativeRegistrationReadback = (registered: boolean) => {
  listDevices.mockReset()
    .mockResolvedValueOnce({ devices: [{ id: "phone", name: "Work browser", machineId: "machine-1", channels: ["web_push"] }], vapidPublicKey: "AQID" } as never)
    .mockResolvedValueOnce({ devices: registered ? [{ id: "origin-device", name: window.location.origin, machineId: "machine-1", channels: ["web_push"] }] : [], vapidPublicKey: "AQID" } as never);
};

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

describe("ReachPage", () => {
  beforeEach(async () => {
    await setLocale("en");
    vi.mocked(fetchNotificationCatalog).mockResolvedValue(populatedCatalog);
    readStatus.mockResolvedValue(connectionStatus(true));
    listDevices.mockResolvedValue({
      devices: [{ id: "phone", name: "Work browser", machineId: "machine-1", channels: ["web_push"] }],
      vapidPublicKey: "AQID",
    } as never);
    vi.mocked(recipientsClient.listChannelAddresses).mockResolvedValue({ addresses: [] } as never);
    vi.mocked(recipientsClient.createChannelAddress).mockResolvedValue({ id: "email-1" } as never);
    vi.mocked(recipientsClient.updateChannelAddress).mockResolvedValue({ id: "email-1" } as never);
  });

  afterEach(() => {
    vi.unstubAllEnvs();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
    Object.defineProperty(navigator, "serviceWorker", { configurable: true, value: undefined });
    Object.defineProperty(window, "PushManager", { configurable: true, value: undefined });
  });

  it("matches the selected focused empty-device entry and keeps enrollment behind deliberate review", async () => {
    await setLocale("en");
    listDevices.mockResolvedValue({ devices: [], vapidPublicKey: "AQID" } as never);
    readStatus.mockResolvedValue({ ...connectionStatus(false), permission: "default" });
    const requestPermission = vi.fn();
    vi.stubGlobal("Notification", { requestPermission });
    renderWithProviders(<ReachPage />);

    expect(await screen.findByRole("heading", { name: "Set up how you’re reached" })).toHaveTextContent("Set up how you’re reached");
    expect(screen.getByTestId("reach-empty-devices")).toHaveTextContent("No devices are registered. You can still answer requests here.");
    expect(screen.getByText("Choose this browser deliberately")).toBeInTheDocument();
    expect(screen.getByText("We’ll explain the registration’s content scope before asking for browser permission.")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Back to requests" })).toHaveAttribute("href", "/");
    expect(screen.getByRole("link", { name: "Preferences" })).toHaveAttribute("href", "/settings?view=preferences");
    const addThisBrowser = screen.getByRole("button", { name: "Add this browser" });
    expect(addThisBrowser).toHaveAttribute("data-testid", "reach-add-browser");
    expect(addThisBrowser).toHaveClass("min-h-11");
    await userEvent.click(screen.getByRole("button", { name: "Add this browser" }));
    expect(await screen.findByTestId("reach-enroll-preflight")).toBeInTheDocument();
    expect(screen.getByText(/This Hub’s current registration endpoint approves Public, Private, Sensitive and Critical full content/i)).toBeInTheDocument();
    expect(requestPermission).not.toHaveBeenCalled();
    expect(registerPush).not.toHaveBeenCalled();
  });

  it("keeps an initial device read failure distinct from an empty list and recovers by readback", async () => {
    listDevices
      .mockRejectedValueOnce(new Error("device list unavailable"))
      .mockResolvedValueOnce({
        devices: [{ id: "phone", name: "Work browser", machineId: "machine-1", channels: ["web_push"] }],
        vapidPublicKey: "AQID",
      } as never);

    renderWithProviders(<ReachPage />);

    expect(await screen.findByRole("heading", { name: "Devices couldn’t be loaded" })).toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent("This is not an empty list.");
    expect(screen.getByText("No device list was returned. Your saved devices are unchanged.")).toBeInTheDocument();
    expect(screen.queryByTestId("reach-empty-devices")).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Preferences" })).toHaveAttribute("href", "/settings?view=preferences");

    await userEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(await screen.findByRole("heading", { name: "Work browser" })).toBeInTheDocument();
    expect(listDevices).toHaveBeenCalledTimes(2);
    expect(registerPush).not.toHaveBeenCalled();
  });

  it("opens the focused channel read-gap recovery without exposing stale values and restores focus on return", async () => {
    const privateAddress = "https://push.example.test/private-endpoint";
    listDevices.mockReset().mockResolvedValue({
      devices: [{ id: "phone", name: "Work browser", machineId: "machine-1", channels: ["web_push"] }],
      vapidPublicKey: "AQID",
    } as never);
    vi.mocked(recipientsClient.listChannelAddresses).mockReset().mockRejectedValue(new Error(privateAddress));
    const { container } = renderWithProviders(<ReachPage />);

    expect(await screen.findByRole("alert")).toHaveTextContent("Saved destination and approved labels could not be read.");
    const deviceOption = screen.getByRole("button", { name: /Work browser/ });
    await userEvent.click(deviceOption);

    expect(await screen.findByTestId("reach-channel-read-gap")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Saved access isn’t available" })).toBeInTheDocument();
    expect(screen.getByText((_, element) => element?.tagName === "P" && element.textContent === "Work browser · phone")).toBeInTheDocument();
    expect(screen.getByText("Channel")).toBeInTheDocument();
    expect(screen.queryByText("Channels")).not.toBeInTheDocument();
    expect(screen.getByText("Existing values are preserved")).toBeInTheDocument();
    expect(screen.getByText("This device response does not include the saved address or approved content labels.")).toBeInTheDocument();
    expect(screen.getByText("Editing stays unavailable until those values can be read safely. No content labels have been assumed.")).toBeInTheDocument();
    expect(screen.getByText("machine-1")).toBeInTheDocument();
    expect(screen.getByText("Web Push")).toBeInTheDocument();
    expect(screen.queryByText(privateAddress)).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Edit content access" })).toBeDisabled();
    await expectNoA11yViolations(container);

    await userEvent.click(screen.getByRole("button", { name: "Back to device" }));
    expect(await screen.findByTestId("reach-device-phone")).toBeInTheDocument();
    expect(document.activeElement).toBe(screen.getByRole("button", { name: /Work browser/ }));
    expect(registerPush).not.toHaveBeenCalled();
  });

  it("refreshes the owner device list on Reach entry instead of showing cached rows as current", async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } });
    queryClient.setQueryData(["recipient-devices"], {
      devices: [{ id: "stale-device", name: "Previous route device", machineId: "old", channels: ["email"] }],
      vapidPublicKey: "AQID",
    });
    listDevices.mockResolvedValue({ devices: [], vapidPublicKey: "AQID" } as never);

    renderWithProviders(<ReachPage />, { queryClient });

    expect(await screen.findByRole("heading", { name: "Set up how you’re reached" })).toHaveTextContent("Set up how you’re reached");
    expect(screen.queryByText("Previous route device")).not.toBeInTheDocument();
    expect(listDevices).toHaveBeenCalled();
  });

  it("exposes the Settings to Reach journey and has an accessible empty entry", async () => {
    listDevices.mockResolvedValue({ devices: [], vapidPublicKey: "AQID" } as never);
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } });
    queryClient.setQueryData(["recipient-devices"], {
      devices: [{ id: "stale-device", name: "Previous route device", machineId: "old", channels: ["email"] }],
      vapidPublicKey: "AQID",
    });
    const router = createMemoryRouter([
      { path: "/settings", element: <SettingsPage /> },
      { path: "/settings/reach", element: <ReachPage /> },
    ], { initialEntries: ["/settings"] });
    const { container } = renderWithProviders(
      <RouterProvider router={router} />,
      { withoutRouter: true, queryClient },
    );
    const reachLink = screen.getByRole("link", { name: "Manage devices and channels" });
    expect(reachLink).toHaveAttribute("href", "/settings/reach");
    await act(async () => { await router.navigate(reachLink.getAttribute("href")!); });
    await waitFor(() => expect(router.state.location.pathname).toBe("/settings/reach"));
    expect(await screen.findByRole("heading", { name: "Set up how you’re reached" })).toHaveTextContent("Set up how you’re reached");
    expect(screen.queryByText("Previous route device")).not.toBeInTheDocument();
    await expectNoA11yViolations(container);
  });

  it("keeps routed test mode when opening Reach from Settings", async () => {
    renderWithProviders(
      <MemoryRouter initialEntries={["/settings?view=all&vrooli_test_mode=1&e22_fixture=settings-owner&e21_fixture=reach-empty"]}>
        <Routes>
          <Route path="/settings" element={<SettingsPage />} />
        </Routes>
      </MemoryRouter>,
      { withoutRouter: true },
    );
    expect(screen.getByRole("link", { name: "Manage devices and channels" })).toHaveAttribute(
      "href",
      "/settings/reach?vrooli_test_mode=1&e21_fixture=reach-empty&e22_fixture=settings-owner",
    );
  });

  const startEnrollment = async () => {
    await userEvent.click(await screen.findByRole("button", { name: "Add this browser" }));
    expect(await screen.findByTestId("reach-enroll-preflight")).toHaveTextContent("Public, Private, Sensitive and Critical");
    expect(within(screen.getByTestId("reach-enroll-preflight")).queryByRole("checkbox")).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Continue to browser prompt" }));
  };

  it("distinguishes unsupported browser push without blocking other Reach settings", async () => {
    readStatus.mockResolvedValue({ ...connectionStatus(false), supported: false, permission: "unsupported" });
    renderWithProviders(<ReachPage />);
    expect(await screen.findByText(/Browser push is unsupported here/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Add this browser" })).not.toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Registered devices" })).toBeInTheDocument();
  });

  it("reviews the real all-label scope before enrollment and separates permission from registration", async () => {
    setAuthoritativeRegistrationReadback(true);
    registerPush.mockResolvedValue({ endpoint: "https://push.example/subscription" });
    renderWithProviders(<ReachPage />);
    await startEnrollment();
    expect(await screen.findByRole("status")).toHaveTextContent("Hub registration is confirmed in the owner-scoped device list, and this browser reports an active subscription");
    expect(screen.getByRole("status")).not.toHaveTextContent(/delivered|arrived/i);
    expect(registerPush).toHaveBeenCalledWith(expect.any(Uint8Array));
    expect(listDevices).toHaveBeenCalledTimes(2);
  });

  it("does not claim a browser subscription when the post-registration readback is inactive", async () => {
    setAuthoritativeRegistrationReadback(true);
    registerPush.mockResolvedValue({ endpoint: "https://push.example/subscription" });
    readStatus.mockResolvedValue(connectionStatus(false));
    renderWithProviders(<ReachPage />);
    await startEnrollment();
    expect(await screen.findByText(/Hub registration is confirmed in the owner-scoped device list, but browser subscription readback is unavailable or inactive/)).toBeInTheDocument();
    expect(screen.getByTestId("connection-subscription-value")).toHaveTextContent("not subscribed");
  });

  it("keeps registration unknown when a successful request is absent from the owner-scoped readback", async () => {
    setAuthoritativeRegistrationReadback(false);
    registerPush.mockResolvedValue({ endpoint: "https://push.example/subscription" });
    renderWithProviders(<ReachPage />);
    await startEnrollment();
    expect(await screen.findByText(/Hub accepted the registration request, but the latest owner-scoped device list does not confirm/)).toBeInTheDocument();
    expect(screen.queryByText(/Hub registration is confirmed/)).not.toBeInTheDocument();
    expect(registerPush).toHaveBeenCalledTimes(1);
  });

  it("keeps registration unknown when the accepted request's owner-scoped readback fails", async () => {
    listDevices.mockReset()
      .mockResolvedValueOnce({ devices: [{ id: "phone", name: "Work browser", machineId: "machine-1", channels: ["web_push"] }], vapidPublicKey: "AQID" } as never)
      .mockRejectedValueOnce(new Error("read unavailable"))
      .mockRejectedValueOnce(new Error("read unavailable"));
    registerPush.mockResolvedValue({ endpoint: "https://push.example/subscription" });
    renderWithProviders(<ReachPage />);
    await startEnrollment();
    expect(await screen.findByText(/registration request was accepted, but the owner-scoped device readback is unavailable/)).toBeInTheDocument();
    expect(screen.queryByText(/Hub registration is confirmed/)).not.toBeInTheDocument();
    expect(registerPush).toHaveBeenCalledTimes(1);
  });

  it("keeps a newer registration readback when an older browser check settles late", async () => {
    setAuthoritativeRegistrationReadback(true);
    registerPush.mockResolvedValue({ endpoint: "https://push.example/subscription" });
    let finishOlderRead: ((status: ConnectionStatus) => void) | undefined;
    readStatus
      .mockResolvedValueOnce(connectionStatus(false))
      .mockImplementationOnce(() => new Promise((resolve) => { finishOlderRead = resolve; }))
      .mockResolvedValueOnce(connectionStatus(true));
    renderWithProviders(<ReachPage />);
    expect(await screen.findByTestId("connection-subscription-value")).toHaveTextContent("not subscribed");
    await userEvent.click(screen.getByRole("button", { name: "Check again" }));
    await waitFor(() => expect(readStatus).toHaveBeenCalledTimes(2));
    await startEnrollment();
    expect(await screen.findByTestId("connection-subscription-value")).toHaveTextContent("present (web.push.apple.com)");
    await act(async () => { finishOlderRead?.(connectionStatus(false)); });
    expect(screen.getByTestId("connection-subscription-value")).toHaveTextContent("present (web.push.apple.com)");
  });

  it("reconciles a conflicted enrollment by exact origin and does not repeat the request", async () => {
    listDevices
      .mockResolvedValueOnce({ devices: [{ id: "current", name: window.location.origin, machineId: "local", channels: ["web_push"] }], vapidPublicKey: "AQID" } as never)
      .mockResolvedValueOnce({ devices: [{ id: "current", name: window.location.origin, machineId: "local", channels: ["web_push"] }], vapidPublicKey: "AQID" } as never);
    registerPush.mockRejectedValueOnce(new Error("registration conflict"));
    renderWithProviders(<ReachPage />);
    await startEnrollment();
    expect(await screen.findByText(/device list shows a Web Push registration for this Hub origin/)).toBeInTheDocument();
    expect(registerPush).toHaveBeenCalledTimes(1);
  });

  it("keeps a lost registration readback unknown and does not retry automatically", async () => {
    listDevices
      .mockResolvedValueOnce({ devices: [], vapidPublicKey: "AQID" } as never)
      .mockRejectedValueOnce(new Error("read unavailable"));
    registerPush.mockRejectedValueOnce(new Error("response lost"));
    renderWithProviders(<ReachPage />);
    await startEnrollment();
    expect(await screen.findByText(/Registration is not confirmed/)).toBeInTheDocument();
    expect(registerPush).toHaveBeenCalledTimes(1);
  });

  it("shows permission recovery without prompting, then returns to the devices view", async () => {
    readStatus.mockResolvedValue({ ...connectionStatus(false), permission: "denied" });
    renderWithProviders(<ReachPage />);
    expect(await screen.findByTestId("reach-permission-recovery")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Notifications weren’t allowed" })).toBeInTheDocument();
    expect(screen.getByText("Your browser did not grant notification permission.")).toBeInTheDocument();
    const guidance = screen.getByRole("status");
    expect(guidance).toHaveStyle({ borderInlineStartColor: "var(--reach-guidance-rail)" });
    expect(guidance.querySelector("p.mt-2")).toHaveClass("text-sm");
    expect(screen.queryByRole("button", { name: "Add this browser" })).not.toBeInTheDocument();
    expect(registerPush).not.toHaveBeenCalled();
    const backToDevices = screen.getByRole("button", { name: "Back to devices" });
    await userEvent.tab();
    expect(backToDevices).toHaveFocus();
    await userEvent.keyboard("{Enter}");
    expect(await screen.findByText("Registered devices")).toBeInTheDocument();
    expect(screen.getByText(/Browser permission is denied/)).toBeInTheDocument();
    expect(registerPush).not.toHaveBeenCalled();
  });

  it("does not offer another enrollment after the native prompt is denied", async () => {
    const requestPermission = vi.fn().mockResolvedValue("denied");
    vi.stubGlobal("Notification", { requestPermission });
    readStatus.mockResolvedValueOnce({ ...connectionStatus(false), permission: "default" }).mockResolvedValue({ ...connectionStatus(false), permission: "denied" });
    renderWithProviders(<ReachPage />);
    await userEvent.click(await screen.findByRole("button", { name: "Add this browser" }));
    await userEvent.click(screen.getByRole("button", { name: "Continue to browser prompt" }));
    expect(await screen.findByText(/Permission was denied/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Add this browser" })).not.toBeInTheDocument();
    expect(requestPermission).toHaveBeenCalledTimes(1);
    expect(registerPush).not.toHaveBeenCalled();
  });

  it("keeps permission pending separate and never starts Hub registration", async () => {
    const requestPermission = vi.fn().mockResolvedValue("default");
    vi.stubGlobal("Notification", { requestPermission });
    readStatus.mockResolvedValue({ ...connectionStatus(false), permission: "default" });
    renderWithProviders(<ReachPage />);
    await userEvent.click(await screen.findByRole("button", { name: "Add this browser" }));
    await userEvent.click(screen.getByRole("button", { name: "Continue to browser prompt" }));
    expect(await screen.findByRole("status")).toHaveTextContent("Permission is still pending");
    expect(requestPermission).toHaveBeenCalledTimes(1);
    expect(registerPush).not.toHaveBeenCalled();
  });

  it("shows the live browser permission wait and prevents a second prompt request", async () => {
    let resolvePermission: ((permission: NotificationPermission) => void) | undefined;
    const requestPermission = vi.fn(() => new Promise<NotificationPermission>((resolve) => { resolvePermission = resolve; }));
    vi.stubGlobal("Notification", { requestPermission });
    readStatus.mockResolvedValue({ ...connectionStatus(false), permission: "default" });
    renderWithProviders(<ReachPage />);
    await userEvent.click(await screen.findByRole("button", { name: "Add this browser" }));
    await userEvent.click(screen.getByRole("button", { name: "Continue to browser prompt" }));
    expect(await screen.findByText(/Waiting for your browser’s notification permission/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add this browser" })).toBeDisabled();
    expect(requestPermission).toHaveBeenCalledTimes(1);
    expect(registerPush).not.toHaveBeenCalled();
    await act(async () => { resolvePermission?.("default"); });
    expect(await screen.findByText(/Permission is still pending/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add this browser" })).toBeEnabled();
    expect(requestPermission).toHaveBeenCalledTimes(1);
    expect(registerPush).not.toHaveBeenCalled();
  });

  it("does not register when this Hub has no public key", async () => {
    vi.stubEnv("VITE_VAPID_PUBLIC_KEY", "");
    listDevices.mockResolvedValueOnce({ devices: [], vapidPublicKey: "" } as never).mockResolvedValueOnce({ devices: [], vapidPublicKey: "" } as never);
    renderWithProviders(<ReachPage />);
    await startEnrollment();
    expect(await screen.findByText(/This Hub has no public push key configured/)).toBeInTheDocument();
    expect(registerPush).not.toHaveBeenCalled();
  });

  it("keeps the channel read gap and never fabricates saved labels or destination", async () => {
    renderWithProviders(<ReachPage />);
    expect(await screen.findByText(/Content access isn’t available to read here yet/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Edit content access" })).toBeDisabled();
    expect(screen.getByText(/Saved destination and approved labels are unchanged and unknown/)).toBeInTheDocument();
  });

  it("validates a new email, requires disclosure review, and holds pointer-only save", async () => {
    renderWithProviders(<ReachPage />);
    await userEvent.click(await screen.findByRole("button", { name: "Add email channel" }));
    const address = screen.getByLabelText("Email address");
    await userEvent.type(address, "not-an-address");
    expect(address).toHaveAttribute("aria-invalid", "true");
    expect(address).toHaveAccessibleDescription(/destination address/);
    expect(screen.getByRole("alert")).toHaveTextContent("Enter a valid email address");
    expect(screen.getByRole("button", { name: "Review content access" })).toBeDisabled();
    await userEvent.clear(address);
    await userEvent.type(address, "owner@example.test");
    await userEvent.click(screen.getByLabelText("Public"));
    expect(screen.getByText(/Pointer-only access is held/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Review content access" })).toBeDisabled();
    await userEvent.click(screen.getByLabelText("Public"));
    await userEvent.click(screen.getByLabelText("Sensitive"));
    await userEvent.click(screen.getByRole("button", { name: "Review content access" }));
    expect(screen.getByText("Public, Sensitive")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save destination" })).toBeEnabled();
    expect(recipientsClient.upsertChannelAddress).not.toHaveBeenCalled();
    await userEvent.click(screen.getByRole("button", { name: "Keep editing" }));
    await userEvent.click(screen.getByRole("button", { name: "Discard" }));
    const discardReview = screen.getByTestId("reach-channel-discard-review");
    expect(screen.getByRole("region", { name: "Discard Work browser changes?" })).toBeInTheDocument();
    expect(discardReview).toHaveTextContent("Saved full content");
    expect(discardReview).toHaveTextContent("Draft adds Public");
    expect(within(discardReview).getByRole("button", { name: "Discard changes" })).toBeVisible();
    const discardTitle = screen.getByRole("heading", { name: "Discard Work browser changes?" });
    expect(discardTitle).toHaveFocus();
    const keepEditing = within(discardReview).getByRole("button", { name: /^Keep editing$/ });
    await userEvent.click(keepEditing);
    expect(screen.getByLabelText("Email address")).toHaveValue("owner@example.test");
    expect(screen.getByRole("button", { name: "Discard" })).toHaveFocus();
    expect(recipientsClient.createChannelAddress).not.toHaveBeenCalled();
    expect(recipientsClient.updateChannelAddress).not.toHaveBeenCalled();
    await userEvent.click(screen.getByRole("button", { name: "Discard" }));
    await userEvent.click(screen.getByRole("button", { name: "Discard changes" }));
    expect(screen.queryByTestId("reach-channel-editor")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add email channel" })).toHaveFocus();
    expect(recipientsClient.createChannelAddress).not.toHaveBeenCalled();
    expect(recipientsClient.updateChannelAddress).not.toHaveBeenCalled();
  });

  it("shows exact saved and draft channel values before local discard", async () => {
    const saved = { id: "email-owned-1", deviceId: "phone", channel: "email", address: "saved@example.test", approvedLabels: ["public"] };
    listDevices.mockResolvedValue({ devices: [{ id: "phone", name: "Work browser", machineId: "machine-1", channels: ["email"] }], vapidPublicKey: "AQID" } as never);
    vi.mocked(recipientsClient.listChannelAddresses).mockResolvedValue({ addresses: [saved] } as never);
    renderWithProviders(<ReachPage />);
    await screen.findByRole("button", { name: "Edit content access" });
    await waitFor(() => expect(screen.getByRole("button", { name: "Edit content access" })).toBeEnabled());
    await userEvent.click(screen.getByRole("button", { name: "Edit content access" }));
    const emailAddress = await screen.findByLabelText("Email address");
    await userEvent.clear(emailAddress);
    await userEvent.type(emailAddress, "draft@example.test");
    await userEvent.click(screen.getByLabelText("Sensitive"));
    await userEvent.click(screen.getByRole("button", { name: "Cancel" }));

    const review = screen.getByTestId("reach-channel-discard-review");
    expect(review).toHaveTextContent("saved@example.test");
    expect(review).toHaveTextContent("Public");
    expect(review).toHaveTextContent("draft@example.test");
    expect(review).toHaveTextContent("Draft adds Sensitive");
    expect(review).not.toHaveTextContent("Draft removes");
    expect(screen.getByRole("heading", { name: "Discard Work browser changes?" })).toHaveFocus();
    expect(recipientsClient.updateChannelAddress).not.toHaveBeenCalled();

    await userEvent.click(within(review).getByRole("button", { name: "Keep editing" }));
    expect(screen.getByLabelText("Email address")).toHaveValue("draft@example.test");
    expect(screen.getByLabelText("Sensitive")).toBeChecked();
    expect(screen.getByRole("button", { name: "Cancel" })).toHaveFocus();
    await userEvent.click(screen.getByRole("button", { name: "Cancel" }));
    await userEvent.click(screen.getByRole("button", { name: "Discard changes" }));
    expect(screen.queryByTestId("reach-channel-editor")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Edit content access" })).toHaveFocus();
    expect(recipientsClient.updateChannelAddress).not.toHaveBeenCalled();
    expect(recipientsClient.createChannelAddress).not.toHaveBeenCalled();
  });

  it("explains that invalid email drafts leave the saved destination and approvals unchanged", async () => {
    const saved = { id: "email-owned-1", deviceId: "phone", channel: "email", address: "owner@example.test", approvedLabels: ["public", "sensitive"] };
    listDevices.mockResolvedValue({ devices: [{ id: "phone", name: "Work browser", machineId: "machine-1", channels: ["email"] }], vapidPublicKey: "AQID" } as never);
    vi.mocked(recipientsClient.listChannelAddresses).mockResolvedValue({ addresses: [saved] } as never);
    renderWithProviders(<ReachPage />);
    await userEvent.click(await screen.findByRole("button", { name: "Edit email content access" }));
    const address = screen.getByLabelText("Email address");
    expect(screen.getByTestId("reach-email-focused-header")).toHaveTextContent("Check the destination");
    await userEvent.clear(address);
    await userEvent.type(address, "review@");
    expect(screen.getByTestId("reach-invalid-email-preserves-saved")).toHaveTextContent("The saved destination and approvals have not been replaced.");
    expect(screen.getByRole("button", { name: "Review changes" })).toBeDisabled();
    expect(recipientsClient.updateChannelAddress).not.toHaveBeenCalled();
    expect(recipientsClient.createChannelAddress).not.toHaveBeenCalled();
  });

  it("creates an email destination with a stable ID and confirms exact owner-scoped readback", async () => {
    renderWithProviders(<ReachPage />);
    await userEvent.click(await screen.findByRole("button", { name: "Add email channel" }));
    await userEvent.type(screen.getByLabelText("Email address"), "owner@example.test");
    await userEvent.click(screen.getByRole("button", { name: "Review content access" }));
    const id = "11111111-1111-4111-8111-111111111111";
    vi.spyOn(crypto, "randomUUID").mockReturnValue(id);
    vi.mocked(recipientsClient.listChannelAddresses)
      .mockResolvedValueOnce({ addresses: [{ id, deviceId: "phone", channel: "email", address: "owner@example.test", approvedLabels: ["public"] }] } as never);
    await userEvent.click(screen.getByRole("button", { name: "Save destination" }));
    await waitFor(() => expect(recipientsClient.createChannelAddress).toHaveBeenCalledWith({ id, deviceId: "phone", channel: "email", address: "owner@example.test", approvedLabels: ["public"] }));
    expect(await screen.findByText(/confirmed by owner-scoped readback/)).toBeInTheDocument();
    expect(screen.queryByTestId("reach-channel-editor")).not.toBeInTheDocument();
    expect(recipientsClient.updateChannelAddress).not.toHaveBeenCalled();
  });

  it("uses secure random bytes when randomUUID is unavailable", async () => {
    vi.stubGlobal("crypto", { getRandomValues: (bytes: Uint8Array) => { bytes.fill(0x23); return bytes; } });
    renderWithProviders(<ReachPage />);
    await userEvent.click(await screen.findByRole("button", { name: "Add email channel" }));
    await userEvent.type(screen.getByLabelText("Email address"), "owner@example.test");
    await userEvent.click(screen.getByRole("button", { name: "Review content access" }));
    const id = "23232323-2323-4323-a323-232323232323";
    vi.mocked(recipientsClient.listChannelAddresses).mockResolvedValueOnce({ addresses: [{ id, deviceId: "phone", channel: "email", address: "owner@example.test", approvedLabels: ["public"] }] } as never);
    await userEvent.click(screen.getByRole("button", { name: "Save destination" }));
    await waitFor(() => expect(recipientsClient.createChannelAddress).toHaveBeenCalledWith({ id, deviceId: "phone", channel: "email", address: "owner@example.test", approvedLabels: ["public"] }));
    expect(await screen.findByText(/saved and confirmed by owner-scoped readback/)).toBeInTheDocument();
  });

  it("updates only the exact saved channel ID and confirms the approved labels by readback", async () => {
    const saved = { id: "email-owned-1", deviceId: "phone", channel: "email", address: "old@example.test", approvedLabels: ["public", "sensitive"] };
    listDevices.mockResolvedValue({ devices: [{ id: "phone", name: "Work browser", machineId: "machine-1", channels: ["email"] }], vapidPublicKey: "AQID" } as never);
    vi.mocked(recipientsClient.listChannelAddresses).mockResolvedValue({ addresses: [saved] } as never);
    renderWithProviders(<ReachPage />);
    expect(await screen.findByText("old@example.test")).toBeInTheDocument();
    await userEvent.click(await screen.findByRole("button", { name: "Edit content access" }));
    expect(screen.getByLabelText("Email address")).toHaveValue("old@example.test");
    await userEvent.clear(screen.getByLabelText("Email address"));
    await userEvent.type(screen.getByLabelText("Email address"), "new@example.test");
    await userEvent.click(screen.getByLabelText("Sensitive"));
    await userEvent.click(screen.getByLabelText("Critical"));
    await userEvent.click(screen.getByRole("button", { name: "Review changes" }));
    expect(screen.getByText("Currently approved:").parentElement).toHaveTextContent("Public, Sensitive");
    expect(screen.getByText(/Approval change:/)).toHaveTextContent("adds critical; removes sensitive");
    vi.mocked(recipientsClient.listChannelAddresses).mockResolvedValueOnce({ addresses: [{ ...saved, address: "new@example.test", approvedLabels: ["public", "critical"] }] } as never);
    await userEvent.click(screen.getByRole("button", { name: "Save destination" }));
    await waitFor(() => expect(recipientsClient.updateChannelAddress).toHaveBeenCalledWith({ id: "email-owned-1", address: "new@example.test", approvedLabels: ["public", "critical"] }));
    expect(recipientsClient.createChannelAddress).not.toHaveBeenCalled();
    expect(await screen.findByText(/saved and confirmed by owner-scoped readback/)).toBeInTheDocument();
  });

  it("edits Web Push content access by exact ID without displaying or changing its private destination", async () => {
    const saved = { id: "push-owned-1", deviceId: "phone", channel: "web_push", address: "https://push.example.test/private-endpoint", approvedLabels: ["public", "private", "team-internal"] };
    listDevices.mockResolvedValue({ devices: [{ id: "phone", name: "Work browser", machineId: "machine-1", channels: ["web_push"] }], vapidPublicKey: "AQID" } as never);
    vi.mocked(recipientsClient.listChannelAddresses).mockResolvedValue({ addresses: [saved] } as never);
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } });
    queryClient.setQueryData(["recipient-devices"], { devices: [{ id: "phone", name: "Work browser", machineId: "machine-1", channels: ["web_push"] }], vapidPublicKey: "AQID" });
    queryClient.setQueryData(["recipient-channel-addresses"], { addresses: [saved] });
    renderWithProviders(<ReachPage />, { queryClient });

    const devicePanel = await screen.findByTestId("reach-device-phone");
    expect(devicePanel.textContent).toContain("Web Push");
    expect(within(devicePanel).getByTestId("reach-device-readiness-unknown")).toHaveClass("border-s-4", "border-app-warning");
    expect(within(devicePanel).getByText(/Unknown — no exact channel and machine readiness record is available/)).toBeInTheDocument();
    expect(within(devicePanel).getByRole("button", { name: "Edit content access" })).toHaveAttribute("id", "reach-channel-entry-phone-web_push");
    expect(within(devicePanel).getByRole("button", { name: "Edit content access" })).toHaveClass("min-h-11");
    expect(within(devicePanel).getByRole("button", { name: "Add email channel" })).toHaveClass("min-h-11");
    expect(within(devicePanel).getByRole("button", { name: "Remove device" })).toHaveClass("min-h-11");
    await userEvent.click(within(devicePanel).getByRole("button", { name: "Edit content access" }));
    expect(screen.getAllByText(/This browser · unchanged/)).toHaveLength(1);
    expect(screen.queryByTestId("reach-web-push-destination")).not.toBeInTheDocument();
    expect(screen.queryByText(saved.address)).not.toBeInTheDocument();
    expect(screen.getByTestId("reach-unknown-label-preserved")).toHaveTextContent("team-internal");
    expect(screen.getByLabelText("Sensitive")).toHaveClass("h-5", "w-5");
    expect(screen.getByLabelText("Sensitive").parentElement).toHaveClass("min-h-11");
    await userEvent.click(screen.getByLabelText("Sensitive"));
    expect(screen.getByTestId("reach-content-access-draft-diff")).toHaveTextContent("Saved: Public, Private. This draft adds Sensitive.");
    await userEvent.click(screen.getByRole("button", { name: "Review changes" }));

    expect(screen.getByRole("heading", { name: "Allow Sensitive content?" })).toBeInTheDocument();
    expect(screen.getByText("Before:").parentElement).toHaveTextContent("Public, Private");
    expect(screen.getByText("After:").parentElement).toHaveTextContent("Public, Private, Sensitive");
    expect(screen.getByRole("alert")).toHaveTextContent("More content can appear outside the Hub.");
    expect(screen.getByRole("alert")).toHaveTextContent("Sensitive titles and bodies may be displayed by your browser.");
    expect(screen.getByText(/Unrecognized label team-internal stays preserved/)).toBeInTheDocument();
    expect(screen.getByText("Critical stays unchecked.")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Keep editing" }));
    await userEvent.click(screen.getByLabelText("Critical"));
    await userEvent.click(screen.getByRole("button", { name: "Review changes" }));
    expect(screen.queryByText("Critical stays unchecked.")).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Keep editing" }));
    await userEvent.click(screen.getByLabelText("Critical"));
    await userEvent.click(screen.getByRole("button", { name: "Review changes" }));
    expect(screen.getByText("Critical stays unchecked.")).toBeInTheDocument();
    expect(screen.getAllByText(/This browser · unchanged/)).toHaveLength(1);
    expect(screen.queryByText(saved.address)).not.toBeInTheDocument();
    vi.mocked(recipientsClient.listChannelAddresses).mockResolvedValueOnce({ addresses: [{ ...saved, approvedLabels: [...saved.approvedLabels, "sensitive"] }] } as never);
    await userEvent.click(screen.getByRole("button", { name: "Save content access" }));

    await waitFor(() => expect(recipientsClient.updateChannelAddress).toHaveBeenCalledWith({
      id: saved.id,
      address: saved.address,
      approvedLabels: [...saved.approvedLabels, "sensitive"],
    }));
    expect(recipientsClient.createChannelAddress).not.toHaveBeenCalled();
    expect(recipientsClient.removeDevice).not.toHaveBeenCalled();
    expect(registerPush).not.toHaveBeenCalled();
    expect(await screen.findByText(/Content access saved and confirmed by owner-scoped readback/)).toBeInTheDocument();
  });

  it("retains a reviewed email draft and stable create ID when response and readback are uncertain", async () => {
    renderWithProviders(<ReachPage />);
    await userEvent.click(await screen.findByRole("button", { name: "Add email channel" }));
    await userEvent.type(screen.getByLabelText("Email address"), "owner@example.test");
    await userEvent.click(screen.getByRole("button", { name: "Review content access" }));
    const id = "22222222-2222-4222-8222-222222222222";
    vi.spyOn(crypto, "randomUUID").mockReturnValue(id);
    vi.mocked(recipientsClient.createChannelAddress).mockRejectedValue(new Error("response lost"));
    await userEvent.click(screen.getByRole("button", { name: "Save destination" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("save response was uncertain");
    expect(screen.getByTestId("reach-channel-editor")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Keep editing" }));
    await userEvent.click(screen.getByRole("button", { name: "Review content access" }));
    await userEvent.click(screen.getByRole("button", { name: "Save destination" }));
    await waitFor(() => expect(recipientsClient.createChannelAddress).toHaveBeenLastCalledWith({ id, deviceId: "phone", channel: "email", address: "owner@example.test", approvedLabels: ["public"] }));
    expect(recipientsClient.createChannelAddress).toHaveBeenCalledTimes(2);
    expect(screen.getByTestId("reach-channel-editor")).toBeInTheDocument();
    expect(screen.queryByText(/saved and confirmed by owner-scoped readback/)).not.toBeInTheDocument();
  });

  it("reports a natural-key conflict without claiming the other destination was changed", async () => {
    renderWithProviders(<ReachPage />);
    await userEvent.click(await screen.findByRole("button", { name: "Add email channel" }));
    await userEvent.type(screen.getByLabelText("Email address"), "draft@example.test");
    await userEvent.click(screen.getByRole("button", { name: "Review content access" }));
    vi.spyOn(crypto, "randomUUID").mockReturnValue("33333333-3333-4333-8333-333333333333");
    vi.mocked(recipientsClient.createChannelAddress).mockRejectedValue(new Error("already exists"));
    vi.mocked(recipientsClient.listChannelAddresses).mockResolvedValueOnce({ addresses: [{ id: "other-row", deviceId: "phone", channel: "email", address: "existing@example.test", approvedLabels: ["sensitive"] }] } as never);
    await userEvent.click(screen.getByRole("button", { name: "Save destination" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Another email destination is already saved");
    expect(screen.getByTestId("reach-channel-editor")).toBeInTheDocument();
    expect(screen.queryByText(/saved and confirmed by owner-scoped readback/)).not.toBeInTheDocument();
  });

  it("keeps device removal behind review and reports an unconfirmed failed readback", async () => {
    vi.mocked(recipientsClient.removeDevice).mockResolvedValueOnce({} as never);
    listDevices
      .mockResolvedValueOnce({ devices: [{ id: "phone", name: "Work browser", machineId: "machine-1", channels: ["web_push"] }], vapidPublicKey: "AQID" } as never)
      .mockResolvedValueOnce({ devices: [{ id: "phone", name: "Work browser", machineId: "machine-1", channels: ["web_push"] }], vapidPublicKey: "AQID" } as never)
      .mockResolvedValue({ devices: [{ id: "phone", name: "Work browser", machineId: "machine-1", channels: ["web_push"] }], vapidPublicKey: "AQID" } as never);
    renderWithProviders(<ReachPage />);
    await userEvent.click(await screen.findByRole("button", { name: "Remove device" }));
    const review = screen.getByTestId("reach-remove-review");
    expect(review).toHaveTextContent("does not revoke browser permission");
    expect(review).toHaveTextContent("Device ID");
    expect(review).toHaveTextContent("machine-1");
    expect(review).not.toContainElement(screen.queryByTestId("reach-device-phone"));
    await waitFor(() => expect(document.activeElement).toBe(within(review).getByRole("heading", { level: 1 })));
    await userEvent.click(within(review).getByRole("button", { name: "Keep device" }));
    await waitFor(() => expect(document.activeElement).toHaveAttribute("id", "reach-remove-trigger-phone"));
    await userEvent.click(screen.getByRole("button", { name: "Remove device" }));
    const reopenedReview = screen.getByTestId("reach-remove-review");
    await waitFor(() => expect(document.activeElement).toBe(within(reopenedReview).getByRole("heading", { level: 1 })));
    await userEvent.click(within(reopenedReview).getByRole("button", { name: "Remove device" }));
    expect(await screen.findByRole("status")).toHaveTextContent(/removal is not confirmed/i);
    expect(screen.getByTestId("reach-remove-review")).toBeInTheDocument();
    expect(screen.getByTestId("reach-remove-review")).toHaveTextContent("phone");
    expect(recipientsClient.removeDevice).toHaveBeenCalledWith({ id: "phone" });
  });

  it("distinguishes a successful empty list from a failed device read", async () => {
    listDevices.mockResolvedValueOnce({ devices: [], vapidPublicKey: "AQID" } as never);
    renderWithProviders(<ReachPage />);
    expect(await screen.findByTestId("reach-settings")).toBeInTheDocument();
    expect(screen.queryByText("Couldn’t load registered devices. This is not an empty list.")).not.toBeInTheDocument();
  });

  it("shows loading as distinct from the authoritative empty list", async () => {
    let finishRead: ((value: { devices: never[]; vapidPublicKey: string }) => void) | undefined;
    listDevices.mockImplementationOnce(() => new Promise((resolve) => { finishRead = resolve; }) as never);
    renderWithProviders(<ReachPage />);
    expect(screen.getByText("Loading registered devices…")).toBeInTheDocument();
    expect(screen.queryByText("No devices are registered. You can still answer requests here.")).not.toBeInTheDocument();
    await act(async () => { finishRead?.({ devices: [], vapidPublicKey: "AQID" }); });
    expect(await screen.findByTestId("reach-settings")).toBeInTheDocument();
  });

  it("keeps device read failure distinct from an empty list", async () => {
    listDevices.mockRejectedValueOnce(new Error("owner read unavailable"));
    renderWithProviders(<ReachPage />);
    expect(await screen.findByText("Couldn’t load registered devices. This is not an empty list.")).toBeInTheDocument();
    expect(screen.queryByText("No devices are registered. You can still answer requests here.")).not.toBeInTheDocument();
  });

  it("keeps the prior device rows and labels them stale after a refresh failure", async () => {
    renderWithProviders(<ReachPage />);
    expect(await screen.findByTestId("reach-device-phone")).toBeInTheDocument();
    listDevices.mockRejectedValueOnce(new Error("read unavailable"));
    await userEvent.click(screen.getByRole("button", { name: "Refresh registered devices" }));
    expect(await screen.findByText(/Showing the last successful device list. Refresh failed/)).toBeInTheDocument();
    expect(screen.getByTestId("reach-device-phone")).toBeInTheDocument();
  });

  it("reconciles device removal when the owner-scoped list no longer contains its ID", async () => {
    listDevices
      .mockResolvedValueOnce({ devices: [{ id: "phone", name: "Work browser", machineId: "machine-1", channels: ["web_push"] }], vapidPublicKey: "AQID" } as never)
      .mockResolvedValueOnce({ devices: [{ id: "demo-phone", name: "Travel phone", machineId: "demo-mobile", channels: ["web_push"] }], vapidPublicKey: "AQID" } as never)
      .mockResolvedValue({ devices: [{ id: "demo-phone", name: "Travel phone", machineId: "demo-mobile", channels: ["web_push"] }], vapidPublicKey: "AQID" } as never);
    renderWithProviders(<ReachPage />);
    await userEvent.click(await screen.findByRole("button", { name: "Remove device" }));
    await userEvent.click(within(screen.getByTestId("reach-remove-review")).getByRole("button", { name: "Remove device" }));
    const result = await screen.findByTestId("reach-device-removed");
    expect(result).toHaveTextContent("Work browser · phone");
    expect(result).toHaveTextContent("Removal confirmed by the latest list");
    expect(result).toHaveTextContent("Work browser is absent from the authorized device list.");
    expect(result).toHaveTextContent("Browser notification permission is unchanged.");
    expect(result).toHaveTextContent("Travel phone");
    expect(result).toHaveTextContent("demo-phone");
    expect(screen.queryByTestId("reach-device-phone")).not.toBeInTheDocument();
    await userEvent.click(within(result).getByRole("button", { name: "Back to devices" }));
    expect(await screen.findByTestId("reach-device-demo-phone")).toBeInTheDocument();
  });

  it("reconciles a lost removal response only after the exact ID is absent from owner readback", async () => {
    vi.mocked(recipientsClient.removeDevice).mockRejectedValueOnce(new Error("response lost"));
    listDevices
      .mockResolvedValueOnce({ devices: [{ id: "phone", name: "Work browser", machineId: "machine-1", channels: ["web_push"] }], vapidPublicKey: "AQID" } as never)
      .mockResolvedValueOnce({ devices: [], vapidPublicKey: "AQID" } as never)
      .mockResolvedValue({ devices: [], vapidPublicKey: "AQID" } as never);
    renderWithProviders(<ReachPage />);
    await userEvent.click(await screen.findByRole("button", { name: "Remove device" }));
    await userEvent.click(within(screen.getByTestId("reach-remove-review")).getByRole("button", { name: "Remove device" }));
    expect(await screen.findByText(/response was uncertain, but the latest owner-scoped device list no longer contains this device/)).toBeInTheDocument();
    expect(screen.queryByTestId("reach-device-phone")).not.toBeInTheDocument();
    expect(recipientsClient.removeDevice).toHaveBeenCalledTimes(1);
  });

  it("preserves the signed-out enrollment gate", async () => {
    vi.mocked(identityClient.getSession).mockResolvedValueOnce({ signedIn: false } as never);
    renderWithProviders(<ReachPage />);
    expect(await screen.findByRole("button", { name: "Sign in" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Add this browser" })).not.toBeInTheDocument();
  });
});
