import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { en } from "../../i18n/catalog";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { device, hub } from "../settings/mocks/fakeHub";
import { findPageHeading, go, renderSettings } from "../../test-utils/settingsHarness";

vi.mock("../../api/notifications", async () => (await import("../settings/mocks/fakeHub")).notificationsModule());
vi.mock("../../api/preferences", async () => (await import("../settings/mocks/fakeHub")).preferencesModule());
vi.mock("../../api/serviceWorker", async () => (await import("../settings/mocks/fakeHub")).serviceWorkerModule());

const NEW_EMAIL = "/settings/reach/devices/mac/channels/email";
const EDIT_EMAIL = "/settings/reach/devices/phone/channels/email";
const EDIT_PUSH = "/settings/reach/devices/phone/channels/web_push";

const save = (user: ReturnType<typeof userEvent.setup>) => user.click(screen.getByRole("button", { name: "Save" }));
const typeAddress = async (user: ReturnType<typeof userEvent.setup>, value: string) => {
  const field = await screen.findByLabelText(/Email address/);
  await user.clear(field);
  await user.type(field, value);
};

describe("Editing a channel", () => {
  beforeEach(async () => {
    await setLocale("en");
    hub.reset({
      devices: [
        device({ id: "phone", name: "iPhone", channels: ["web_push", "email"] }),
        device({ id: "mac", name: "Mac", channels: ["macos_notification"] }),
      ],
      addresses: [
        { id: "a-push", deviceId: "phone", channel: "web_push", address: "https://push.example/secret-endpoint", approvedLabels: ["public", "private"] },
        { id: "a-mail", deviceId: "phone", channel: "email", address: "me@example.com", approvedLabels: ["public"] },
      ],
    });
  });

  describe("adding an email", () => {
    it("holds Save until the address is valid and explains an invalid one next to the field", async () => {
      const user = userEvent.setup();
      renderSettings(NEW_EMAIL);
      await findPageHeading("Email");
      await typeAddress(user, "not-an-address");
      expect(screen.getByText(en.reach.channel.invalidEmail)).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
      await typeAddress(user, "me@example.com");
      expect(screen.getByRole("button", { name: "Save" })).toBeEnabled();
    });

    it("confirms what will be allowed in full before creating, then creates with an ID the Hub can recognise", async () => {
      const user = userEvent.setup();
      renderSettings(NEW_EMAIL);
      await typeAddress(user, "new@example.com");
      await user.click(screen.getByLabelText(/^Private/));
      await save(user);
      const confirm = await screen.findByRole("alertdialog");
      expect(confirm).toHaveTextContent("new@example.com will be able to receive Public and Private content in full.");
      expect(hub.recipientsClient.createChannelAddress).not.toHaveBeenCalled();
      await user.click(within(confirm).getByRole("button", { name: "Save" }));

      await findPageHeading("Mac");
      expect(await screen.findByTestId(selectors.reach.channelSaved)).toHaveTextContent("Email settings saved.");
      expect(hub.state.addresses.find((row) => row.deviceId === "mac")).toMatchObject({ channel: "email", address: "new@example.com", approvedLabels: ["public", "private"] });
      expect(hub.recipientsClient.createChannelAddress.mock.calls[0]?.[0]).toMatchObject({ id: expect.stringMatching(/^[0-9a-f-]{36}$/) as string, deviceId: "mac" });
    });

    it("keeps the draft after a failed save and retries with the same ID, so nothing is created twice", async () => {
      const user = userEvent.setup();
      renderSettings(NEW_EMAIL);
      await typeAddress(user, "new@example.com");
      hub.failNext("createChannelAddress", "reject");
      await save(user);
      await user.click(within(await screen.findByRole("alertdialog")).getByRole("button", { name: "Save" }));
      expect(await screen.findByTestId(selectors.reach.channelProblem)).toHaveTextContent("Couldn’t confirm the save");
      expect(screen.getByLabelText(/Email address/)).toHaveValue("new@example.com");

      await save(user);
      await user.click(within(await screen.findByRole("alertdialog")).getByRole("button", { name: "Save" }));
      await findPageHeading("Mac");
      const ids = hub.recipientsClient.createChannelAddress.mock.calls.map(([request]) => (request as { id: string }).id);
      expect(ids).toHaveLength(2);
      expect(ids[0]).toBe(ids[1]);
      expect(hub.state.addresses.filter((row) => row.deviceId === "mac")).toHaveLength(1);
    });

    it("treats a create whose answer was lost as saved when a fresh read confirms it", async () => {
      const user = userEvent.setup();
      renderSettings(NEW_EMAIL);
      await typeAddress(user, "new@example.com");
      hub.failNext("createChannelAddress", "lose");
      await save(user);
      await user.click(within(await screen.findByRole("alertdialog")).getByRole("button", { name: "Save" }));
      expect(await screen.findByTestId(selectors.reach.channelSaved)).toHaveTextContent("The response was unclear, but a fresh read confirms them.");
    });

    it("reports a second address for the same device as a conflict without claiming it changed the other", async () => {
      const user = userEvent.setup();
      renderSettings(NEW_EMAIL);
      await typeAddress(user, "new@example.com");
      // Another tab added an address for this device while this form was open.
      hub.state.addresses.push({ id: "other", deviceId: "mac", channel: "email", address: "existing@example.com", approvedLabels: ["public"] });
      await save(user);
      await user.click(within(await screen.findByRole("alertdialog")).getByRole("button", { name: "Save" }));
      expect(await screen.findByTestId(selectors.reach.channelProblem)).toHaveTextContent("Another address is already saved");
      expect(hub.state.addresses.find((row) => row.id === "other")?.address).toBe("existing@example.com");
    });

    it("does not make a person confirm leaving a form they never touched", async () => {
      const user = userEvent.setup();
      renderSettings(NEW_EMAIL);
      await screen.findByLabelText(/Email address/);
      await user.click(screen.getByRole("button", { name: "Cancel" }));
      await findPageHeading("Mac");
      expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument();
    });
  });

  describe("changing what an email may receive", () => {
    it("shows the saved address and approvals, and holds Save until something changes", async () => {
      renderSettings(EDIT_EMAIL);
      expect(await screen.findByLabelText(/Email address/)).toHaveValue("me@example.com");
      expect(screen.getByLabelText(/^Public/)).toBeChecked();
      expect(screen.getByLabelText(/^Private/)).not.toBeChecked();
      expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    });

    it("warns that sensitive details will leave the Hub and confirms before widening", async () => {
      const user = userEvent.setup();
      renderSettings(EDIT_EMAIL);
      await user.click(await screen.findByLabelText(/^Sensitive/));
      expect(screen.getByText(/Sensitive and critical details will leave Chimesail/)).toBeInTheDocument();
      await save(user);
      const confirm = await screen.findByRole("alertdialog");
      expect(confirm).toHaveTextContent("me@example.com will be able to receive Public and Sensitive content in full.");
      await user.click(within(confirm).getByRole("button", { name: "Save" }));
      await findPageHeading("iPhone");
      expect(hub.recipientsClient.updateChannelAddress).toHaveBeenCalledWith({ id: "a-mail", address: "me@example.com", approvedLabels: ["public", "sensitive"] });
    });

    it("saves a narrowing straight away, with no confirmation", async () => {
      const user = userEvent.setup();
      hub.state.addresses[1] = { id: "a-mail", deviceId: "phone", channel: "email", address: "me@example.com", approvedLabels: ["public", "private"] };
      renderSettings(EDIT_EMAIL);
      await user.click(await screen.findByLabelText(/^Private/));
      await save(user);
      await findPageHeading("iPhone");
      expect(hub.state.addresses.find((row) => row.id === "a-mail")?.approvedLabels).toEqual(["public"]);
    });

    it("keeps an approval it does not recognise exactly as it was", async () => {
      const user = userEvent.setup();
      hub.state.addresses[1] = { id: "a-mail", deviceId: "phone", channel: "email", address: "me@example.com", approvedLabels: ["public", "classified"] };
      renderSettings(EDIT_EMAIL);
      expect(await screen.findByTestId(selectors.reach.unknownLabelPreserved)).toHaveTextContent("classified");
      await user.click(await screen.findByLabelText(/^Private/));
      await save(user);
      await user.click(within(await screen.findByRole("alertdialog")).getByRole("button", { name: "Save" }));
      await findPageHeading("iPhone");
      expect(hub.state.addresses.find((row) => row.id === "a-mail")?.approvedLabels).toEqual(["public", "classified", "private"]);
    });

    it("will not save with nothing approved, and says why", async () => {
      const user = userEvent.setup();
      renderSettings(EDIT_EMAIL);
      await user.click(await screen.findByLabelText(/^Public/));
      expect(screen.getByText(/Choose at least one/)).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    });

    it("asks before an edit is lost to navigation", async () => {
      const user = userEvent.setup();
      const { router } = renderSettings(EDIT_EMAIL);
      await user.click(await screen.findByLabelText(/^Private/));
      await go(router, "/settings");
      const confirm = await screen.findByRole("alertdialog");
      expect(router.state.location.pathname).toBe(EDIT_EMAIL);
      await user.click(within(confirm).getByRole("button", { name: "Discard" }));
      await waitFor(() => expect(router.state.location.pathname).toBe("/settings"));
    });
  });

  describe("browser push content access", () => {
    it("edits only what may be shown, never the private destination", async () => {
      const user = userEvent.setup();
      renderSettings(EDIT_PUSH);
      await findPageHeading("Push notification");
      expect(screen.queryByLabelText(/Email address/)).not.toBeInTheDocument();
      await user.click(await screen.findByLabelText(/^Sensitive/));
      expect(screen.getByText(/may appear in notifications on this browser/)).toBeInTheDocument();
      await save(user);
      const confirm = await screen.findByRole("alertdialog");
      expect(confirm).toHaveTextContent("Notifications on this browser will be able to show Public, Private, and Sensitive content in full.");
      await user.click(within(confirm).getByRole("button", { name: "Save" }));
      await findPageHeading("iPhone");
      expect(hub.recipientsClient.updateChannelAddress).toHaveBeenCalledWith({ id: "a-push", address: "https://push.example/secret-endpoint", approvedLabels: ["public", "private", "sensitive"] });
    });
  });

  describe("channels with nothing to edit", () => {
    it("says so rather than showing an empty form, and offers the way back", async () => {
      renderSettings("/settings/reach/devices/mac/channels/macos_notification");
      expect(await screen.findByText(en.reach.channel.notEditableTitle)).toBeInTheDocument();
      expect(screen.getAllByRole("link", { name: "Back to device" }).length).toBeGreaterThan(0);
    });

    it("will not create browser push from here, because that is done by turning notifications on", async () => {
      renderSettings("/settings/reach/devices/mac/channels/web_push");
      expect(await screen.findByText(en.reach.channel.notEditableTitle)).toBeInTheDocument();
    });

    it("will not edit an email the device lists but whose saved settings are missing", async () => {
      hub.state.addresses = hub.state.addresses.filter((row) => row.channel !== "email");
      renderSettings(EDIT_EMAIL);
      expect(await screen.findByText(en.reach.channel.notEditableTitle)).toBeInTheDocument();
    });
  });

  it("shows the sign-in card when signed out", async () => {
    hub.signOut();
    renderSettings(EDIT_EMAIL);
    expect(await screen.findByRole("region", { name: "Sign in" })).toBeInTheDocument();
  });
});
