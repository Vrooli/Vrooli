import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test-utils";
import { en } from "../../i18n/catalog";
import { setLocale } from "../../i18n";
import { selectors } from "../../consts/selectors";
import { hub } from "../settings/mocks/fakeHub";
import { DeviceAppsPanel } from "./DeviceAppsPanel";

vi.mock("../../api/notifications", async () => (await import("../settings/mocks/fakeHub")).notificationsModule());

const IPHONE = "iPhone";

const apps = () => [
  {
    group: "iphone", label: "iPhone", lastUsed: "2026-10-28T10:00:00Z",
    apps: [
      { subscriptionId: "c1", scenario: "", displayName: "Chimesail", healthy: true, status: "Receives every app not listed here", confirmedAt: "" },
      { subscriptionId: "a1", scenario: "web-console", displayName: "Web Console", healthy: true, status: "Shows Web Console notifications here instead of Chimesail", confirmedAt: "2026-10-28T10:00:00Z" },
    ],
  },
  { group: "mac", label: "Mac · Chrome", lastUsed: "2026-10-20T10:00:00Z", apps: [{ subscriptionId: "c2", scenario: "", displayName: "Chimesail", healthy: true, status: "Receives every app not listed here", confirmedAt: "" }] },
];

describe("Apps on your devices", () => {
  beforeEach(async () => {
    await setLocale("en");
    hub.reset({ deviceApps: apps() });
  });

  it("lists each device with the apps on it and offers to hand back only a scenario's own app", async () => {
    renderWithProviders(<DeviceAppsPanel />);
    const root = await screen.findByTestId(selectors.deviceApps.root);
    expect(await within(root).findByText(IPHONE)).toBeInTheDocument();
    expect(root).toHaveTextContent("Mac · Chrome");
    expect(root).toHaveTextContent("Shows Web Console notifications here instead of Chimesail");
    expect(screen.getAllByTestId(selectors.deviceApps.removeButton)).toHaveLength(1);
  });

  it("asks first, then hands the device back to Chimesail and says so", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DeviceAppsPanel />);
    await user.click(await screen.findByTestId(selectors.deviceApps.removeButton));
    const confirm = await screen.findByRole("alertdialog");
    expect(confirm).toHaveTextContent("Show Web Console notifications in Chimesail on iPhone?");
    expect(hub.appDeliveryClient.removeDeviceApp).not.toHaveBeenCalled();
    await user.click(within(confirm).getByRole("button", { name: "Use Chimesail instead" }));
    expect(await screen.findByTestId(selectors.deviceApps.message)).toHaveTextContent("Web Console notifications on iPhone now arrive in Chimesail.");
    expect(hub.appDeliveryClient.removeDeviceApp).toHaveBeenCalledWith({ subscriptionId: "a1" });
    expect(screen.queryByTestId(selectors.deviceApps.removeButton)).not.toBeInTheDocument();
  });

  it("changes nothing when the confirmation is declined", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DeviceAppsPanel />);
    await user.click(await screen.findByTestId(selectors.deviceApps.removeButton));
    await user.click(within(await screen.findByRole("alertdialog")).getByRole("button", { name: "Cancel" }));
    expect(hub.appDeliveryClient.removeDeviceApp).not.toHaveBeenCalled();
  });

  it("says when a switch back failed", async () => {
    const user = userEvent.setup();
    hub.failNext("removeDeviceApp", "reject");
    renderWithProviders(<DeviceAppsPanel />);
    await user.click(await screen.findByTestId(selectors.deviceApps.removeButton));
    await user.click(within(await screen.findByRole("alertdialog")).getByRole("button", { name: "Use Chimesail instead" }));
    expect(await screen.findByTestId(selectors.deviceApps.message)).toHaveTextContent("Web Console could not be switched back.");
  });

  it("says so when no device has notifications turned on", async () => {
    hub.reset({ deviceApps: [] });
    renderWithProviders(<DeviceAppsPanel />);
    expect(await screen.findByText(en.reach.deviceApps.empty)).toBeInTheDocument();
  });
});
