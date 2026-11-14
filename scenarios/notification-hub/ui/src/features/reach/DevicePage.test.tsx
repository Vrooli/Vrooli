import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { en } from "../../i18n/catalog";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { device, hub } from "../settings/mocks/fakeHub";
import { findPageHeading, renderSettings } from "../../test-utils/settingsHarness";

vi.mock("../../api/notifications", async () => (await import("../settings/mocks/fakeHub")).notificationsModule());
vi.mock("../../api/preferences", async () => (await import("../settings/mocks/fakeHub")).preferencesModule());
vi.mock("../../api/serviceWorker", async () => (await import("../settings/mocks/fakeHub")).serviceWorkerModule());

const PATH = "/settings/reach/devices/phone";
const MACHINE = "machine-7";

describe("A device", () => {
  beforeEach(async () => {
    await setLocale("en");
    hub.reset({
      devices: [
        device({ id: "phone", name: "iPhone", machineId: MACHINE, channels: ["web_push", "email"] }),
        device({ id: "mac", name: "Mac", channels: ["macos_notification"] }),
      ],
      addresses: [
        { id: "a-push", deviceId: "phone", channel: "web_push", address: "https://push.example/secret-endpoint", approvedLabels: ["public", "private"] },
        { id: "a-mail", deviceId: "phone", channel: "email", address: "me@example.com", approvedLabels: ["public"] },
      ],
    });
  });

  it("is titled by name, with the machine under it, and a link back to the devices", async () => {
    renderSettings(PATH);
    await findPageHeading("iPhone");
    expect(screen.getByText(MACHINE)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Devices and channels" })).toHaveAttribute("href", "/settings/reach");
  });

  it("shows where each channel goes and what it may carry, never the private push endpoint", async () => {
    renderSettings(PATH);
    const push = await screen.findByTestId(selectors.reach.channel({ channel: "web_push" }));
    expect(push).toHaveTextContent("This browser");
    expect(push).toHaveTextContent("Allowed in full: Public and Private");
    expect(push).not.toHaveTextContent("secret-endpoint");
    const mail = screen.getByTestId(selectors.reach.channel({ channel: "email" }));
    expect(mail).toHaveTextContent("me@example.com");
    expect(mail).toHaveTextContent("Allowed in full: Public");
  });

  it("opens each editable channel at its own address", async () => {
    renderSettings(PATH);
    expect(await screen.findByTestId(selectors.reach.channelEdit({ channel: "email" }))).toHaveAttribute("href", "/settings/reach/devices/phone/channels/email");
    expect(screen.getByTestId(selectors.reach.channelEdit({ channel: "web_push" }))).toHaveAttribute("href", "/settings/reach/devices/phone/channels/web_push");
    expect(screen.queryByTestId(selectors.reach.addEmail)).not.toBeInTheDocument();
  });

  it("offers to add email only to a device that has none, and lists channels that cannot be edited without a dead link", async () => {
    renderSettings("/settings/reach/devices/mac");
    expect(await screen.findByTestId(selectors.reach.addEmail)).toHaveAttribute("href", "/settings/reach/devices/mac/channels/email");
    const channel = screen.getByTestId(selectors.reach.channel({ channel: "macos_notification" }));
    expect(channel).toHaveTextContent("Mac notification");
    expect(within(channel).queryByRole("link")).not.toBeInTheDocument();
  });

  it("says saved settings could not be read instead of inventing values, and hides edit links meanwhile", async () => {
    hub.failAlways("listChannelAddresses");
    renderSettings(PATH);
    expect(await screen.findByText(en.reach.device.addressesFailed)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.reach.channel({ channel: "email" }))).toHaveTextContent("Saved settings couldn’t be read.");
    expect(screen.queryByTestId(selectors.reach.channelEdit({ channel: "email" }))).not.toBeInTheDocument();
  });

  it("asks before removing, then confirms from the device list and says so on the list", async () => {
    const user = userEvent.setup();
    renderSettings(PATH);
    await user.click(await screen.findByRole("button", { name: "Remove device" }));
    const confirm = await screen.findByRole("alertdialog");
    expect(confirm).toHaveTextContent("Remove iPhone?");
    expect(hub.recipientsClient.removeDevice).not.toHaveBeenCalled();
    await user.click(within(confirm).getByRole("button", { name: "Remove device" }));

    await findPageHeading("Devices and channels");
    expect(await screen.findByTestId(selectors.reach.deviceRemoved)).toHaveTextContent("iPhone was removed.");
    expect(hub.state.devices.map((row) => row.id)).toEqual(["mac"]);
    expect(screen.queryByTestId(selectors.reach.device({ id: "phone" }))).not.toBeInTheDocument();
  });

  it("leaves the device in place and says the removal was not confirmed when it did not happen", async () => {
    const user = userEvent.setup();
    hub.failNext("removeDevice", "reject");
    renderSettings(PATH);
    await user.click(await screen.findByRole("button", { name: "Remove device" }));
    await user.click(within(await screen.findByRole("alertdialog")).getByRole("button", { name: "Remove device" }));
    expect(await screen.findByText(en.reach.device.removeFailed)).toBeInTheDocument();
    expect(screen.getByRole("heading", { level: 1, name: "iPhone" })).toBeInTheDocument();
    expect(hub.state.devices).toHaveLength(2);
  });

  it("accepts a removal whose answer was lost when the list no longer holds the device", async () => {
    const user = userEvent.setup();
    hub.failNext("removeDevice", "lose");
    renderSettings(PATH);
    await user.click(await screen.findByRole("button", { name: "Remove device" }));
    await user.click(within(await screen.findByRole("alertdialog")).getByRole("button", { name: "Remove device" }));
    expect(await screen.findByTestId(selectors.reach.deviceRemoved)).toHaveTextContent("The response was unclear, but the device list no longer shows it");
  });

  it("says a device that is gone is gone, with the way back", async () => {
    renderSettings("/settings/reach/devices/missing");
    expect(await screen.findByText(en.reach.device.goneTitle)).toBeInTheDocument();
    await waitFor(() => expect(screen.getAllByRole("link", { name: "Devices and channels" }).length).toBeGreaterThan(0));
  });

  it("shows the sign-in card when signed out", async () => {
    hub.signOut();
    renderSettings(PATH);
    expect(await screen.findByRole("region", { name: "Sign in" })).toBeInTheDocument();
  });
});
