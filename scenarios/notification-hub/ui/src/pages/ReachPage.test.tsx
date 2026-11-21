import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { en } from "../i18n/catalog";
import { selectors } from "../consts/selectors";
import { setLocale } from "../i18n";
import { device, hub } from "../features/settings/mocks/fakeHub";
import { findPageHeading, go, renderSettings } from "../test-utils/settingsHarness";

vi.mock("../api/notifications", async () => (await import("../features/settings/mocks/fakeHub")).notificationsModule());
vi.mock("../api/preferences", async () => (await import("../features/settings/mocks/fakeHub")).preferencesModule());
vi.mock("../api/serviceWorker", async () => (await import("../features/settings/mocks/fakeHub")).serviceWorkerModule());

const PATH = "/settings/reach";

/** What the browser does when asked for notification permission. */
function stubPermissionPrompt(answer: NotificationPermission) {
  const requestPermission = vi.fn(() => {
    hub.state.browser.permission = answer;
    return Promise.resolve(answer);
  });
  vi.stubGlobal("Notification", { get permission() { return hub.state.browser.permission; }, requestPermission });
  return requestPermission;
}

const turnOn = async (user: ReturnType<typeof userEvent.setup>) => {
  await user.click(await screen.findByRole("button", { name: "Turn on notifications" }));
  await user.click(within(await screen.findByRole("alertdialog")).getByRole("button", { name: "Continue" }));
};

describe("Devices and channels", () => {
  beforeEach(async () => {
    await setLocale("en");
    hub.reset({ devices: [device({ id: "phone", name: "iPhone", channels: ["web_push", "email"] }), device({ id: "mac", name: "Mac", channels: [] })] });
  });
  afterEach(() => vi.unstubAllGlobals());

  it("lists devices with the channels each can be reached on and opens each at its own address", async () => {
    renderSettings(PATH);
    await findPageHeading("Devices and channels");
    const phone = await screen.findByTestId(selectors.reach.device({ id: "phone" }));
    expect(phone).toHaveTextContent("iPhone");
    expect(phone).toHaveTextContent("Push notification, Email");
    expect(phone).toHaveAttribute("href", "/settings/reach/devices/phone");
    expect(screen.getByTestId(selectors.reach.device({ id: "mac" }))).toHaveTextContent("No channels reported");
  });

  it("keeps the read-only device list after expiry and clears device-app reads", async () => {
    renderSettings(PATH);
    expect(await screen.findByTestId(selectors.reach.device({ id: "phone" }))).toBeInTheDocument();
    expect(await screen.findByTestId(selectors.deviceApps.root)).toBeInTheDocument();
    const deviceReads = hub.recipientsClient.listDevices.mock.calls.length;
    const appReads = hub.appDeliveryClient.listDeviceApps.mock.calls.length;

    fireEvent(window, new Event("notification-hub:session-expired"));

    expect(await screen.findByText(en.settings.sessionEnded.title)).toBeInTheDocument();
    const visiblePhone = (await screen.findAllByTestId(selectors.reach.device({ id: "phone" }))).find((row) => !row.closest("[hidden]"));
    expect(visiblePhone).toBeInTheDocument();
    await waitFor(() => {
      expect(hub.recipientsClient.listDevices.mock.calls.length).toBeGreaterThanOrEqual(deviceReads);
      expect(hub.appDeliveryClient.listDeviceApps).toHaveBeenCalledTimes(appReads);
    });
  });

  it("marks the device that is this browser", async () => {
    hub.reset({ devices: [device({ id: "here", name: window.location.origin, channels: ["web_push"] })] });
    renderSettings(PATH);
    expect(await screen.findByTestId(selectors.reach.device({ id: "here" }))).toHaveTextContent("This browser");
  });

  it("says plainly that there are no devices, and that answering decisions still works", async () => {
    hub.reset();
    renderSettings(PATH);
    expect(await screen.findByText(en.reach.devices.emptyTitle)).toBeInTheDocument();
    expect(screen.getByText(/You can still answer decisions here/)).toBeInTheDocument();
  });

  it("never mistakes a failed read for an empty list, and recovers with a retry", async () => {
    const user = userEvent.setup();
    hub.failNext("listDevices", "reject");
    renderSettings(PATH);
    expect(await screen.findByText(en.reach.devices.loadFailed)).toBeInTheDocument();
    expect(screen.queryByText(en.reach.devices.emptyTitle)).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Try again" }));
    expect(await screen.findByTestId(selectors.reach.device({ id: "phone" }))).toBeInTheDocument();
  });

  it("keeps showing the last devices, labelled stale, when a refresh fails", async () => {
    const user = userEvent.setup();
    renderSettings(PATH);
    await screen.findByTestId(selectors.reach.device({ id: "phone" }));
    hub.failNext("listDevices", "reject");
    await user.click(screen.getByRole("button", { name: "Refresh devices" }));
    expect(await screen.findByText(/Showing what loaded earlier/)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.reach.device({ id: "phone" }))).toBeInTheDocument();
  });

  it("reads the device list afresh on arrival instead of trusting cached rows", async () => {
    const { router } = renderSettings(PATH);
    await screen.findByTestId(selectors.reach.device({ id: "phone" }));
    hub.state.devices = [device({ id: "tablet", name: "Tablet", channels: ["email"] })];
    await go(router, "/settings");
    await go(router, PATH);
    expect(await screen.findByTestId(selectors.reach.device({ id: "tablet" }))).toBeInTheDocument();
    await waitFor(() => expect(screen.queryByTestId(selectors.reach.device({ id: "phone" }))).not.toBeInTheDocument());
  });

  it("shows a read-only device list and sign-in card when signed out", async () => {
    hub.signOut();
    renderSettings(PATH);
    expect(await screen.findByTestId(selectors.session.form)).toBeInTheDocument();
    const phoneRows = await screen.findAllByTestId(selectors.reach.device({ id: "phone" }));
    expect(phoneRows.some((row) => !row.closest("[hidden]"))).toBe(true);
    expect(screen.queryByRole("button", { name: "Turn on notifications" })).not.toBeInTheDocument();
    expect(screen.queryByTestId(selectors.deviceApps.root)).not.toBeInTheDocument();
    expect(hub.recipientsClient.removeDevice).not.toHaveBeenCalled();
  });
});

describe("This browser", () => {
  beforeEach(async () => {
    await setLocale("en");
    hub.reset();
  });
  afterEach(() => vi.unstubAllGlobals());

  it("says when push is not available here and offers no way to turn it on", async () => {
    hub.state.browser.supported = false;
    hub.state.browser.permission = "unsupported";
    renderSettings(PATH);
    expect(await screen.findByText(en.reach.browser.unsupportedTitle)).toBeInTheDocument();
    expect(screen.getByText(/open Chimesail from the Home Screen/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Turn on notifications" })).not.toBeInTheDocument();
  });

  it("explains blocked notifications without prompting again", async () => {
    const prompt = stubPermissionPrompt("granted");
    hub.state.browser.permission = "denied";
    renderSettings(PATH);
    expect(await screen.findByText(en.reach.browser.deniedTitle)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Turn on notifications" })).not.toBeInTheDocument();
    expect(prompt).not.toHaveBeenCalled();
  });

  it("explains the scope, asks the browser, registers, and confirms from the Hub's own list", async () => {
    const user = userEvent.setup();
    const prompt = stubPermissionPrompt("granted");
    renderSettings(PATH);
    await user.click(await screen.findByRole("button", { name: "Turn on notifications" }));
    const dialog = await screen.findByRole("alertdialog");
    expect(dialog).toHaveTextContent("including sensitive and critical ones");
    expect(prompt).not.toHaveBeenCalled();
    await user.click(within(dialog).getByRole("button", { name: "Continue" }));

    expect(await screen.findByText(en.reach.browser.result.registered)).toBeInTheDocument();
    expect(prompt).toHaveBeenCalledTimes(1);
    expect(hub.registerBrowserPushSubscription).toHaveBeenCalledTimes(1);
    expect(await screen.findByTestId(selectors.reach.browserState)).toHaveTextContent("Notifications are on");
  });

  it("registers nothing when the permission prompt is dismissed", async () => {
    const user = userEvent.setup();
    stubPermissionPrompt("default");
    renderSettings(PATH);
    await turnOn(user);
    expect(await screen.findByText(en.reach.browser.result.dismissed)).toBeInTheDocument();
    expect(hub.registerBrowserPushSubscription).not.toHaveBeenCalled();
  });

  it("registers nothing when the prompt is refused, and stops offering the button", async () => {
    const user = userEvent.setup();
    stubPermissionPrompt("denied");
    renderSettings(PATH);
    await turnOn(user);
    expect(await screen.findByTestId(selectors.reach.enrollResult)).toHaveTextContent("The browser blocked notifications");
    expect(hub.registerBrowserPushSubscription).not.toHaveBeenCalled();
    await waitFor(() => expect(screen.queryByRole("button", { name: "Turn on notifications" })).not.toBeInTheDocument());
  });

  it("does not claim registration when the request was accepted but the Hub's list lacks this browser", async () => {
    const user = userEvent.setup();
    stubPermissionPrompt("granted");
    hub.registerBrowserPushSubscription.mockImplementationOnce(() => Promise.resolve({}));
    renderSettings(PATH);
    await turnOn(user);
    expect(await screen.findByTestId(selectors.reach.enrollResult)).toHaveTextContent("Couldn’t confirm the registration");
    expect(screen.queryByText(en.reach.browser.result.registered)).not.toBeInTheDocument();
  });

  it("says it was not registered when the request failed and the Hub holds nothing", async () => {
    const user = userEvent.setup();
    stubPermissionPrompt("granted");
    hub.failNext("registerBrowserPushSubscription", "reject");
    renderSettings(PATH);
    await turnOn(user);
    expect(await screen.findByTestId(selectors.reach.enrollResult)).toHaveTextContent("This browser wasn’t registered");
  });

  it("counts a registration whose answer was lost when the Hub's list shows this browser", async () => {
    const user = userEvent.setup();
    stubPermissionPrompt("granted");
    hub.failNext("registerBrowserPushSubscription", "lose");
    renderSettings(PATH);
    await turnOn(user);
    expect(await screen.findByText(en.reach.browser.result.registered)).toBeInTheDocument();
  });

  it("flags a registration the browser does not hold a subscription for", async () => {
    const user = userEvent.setup();
    stubPermissionPrompt("granted");
    hub.registerBrowserPushSubscription.mockImplementationOnce(() => {
      hub.state.devices.push(device({ id: "here", name: window.location.origin, channels: ["web_push"] }));
      return Promise.resolve({});
    });
    renderSettings(PATH);
    await turnOn(user);
    expect(await screen.findByTestId(selectors.reach.enrollResult)).toHaveTextContent("reports no active push subscription");
  });

  it("does not register when the Hub has no push key", async () => {
    const user = userEvent.setup();
    stubPermissionPrompt("granted");
    hub.state.vapidPublicKey = "";
    renderSettings(PATH);
    await turnOn(user);
    expect(await screen.findByTestId(selectors.reach.enrollResult)).toHaveTextContent("no push key");
    expect(hub.registerBrowserPushSubscription).not.toHaveBeenCalled();
  });

  it("offers to update a stale worker and reads the technical state only on request", async () => {
    const user = userEvent.setup();
    hub.state.browser.stale = true;
    hub.state.browser.runningVersion = "1";
    hub.state.browser.availableVersion = "2";
    renderSettings(PATH);
    await user.click(await screen.findByRole("button", { name: "Update this browser" }));
    expect(hub.forceServiceWorkerUpdate).toHaveBeenCalledTimes(1);
    expect(await screen.findByText(en.reach.browser.details)).toBeInTheDocument();
  });
});
