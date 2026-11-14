import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { en } from "../../i18n/catalog";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { interp } from "../../test-utils";
import { device, hub } from "./mocks/fakeHub";
import { findPageHeading, go, renderSettings } from "../../test-utils/settingsHarness";

vi.mock("../../api/notifications", async () => (await import("./mocks/fakeHub")).notificationsModule());
vi.mock("../../api/preferences", async () => (await import("./mocks/fakeHub")).preferencesModule());
vi.mock("../../api/serviceWorker", async () => (await import("./mocks/fakeHub")).serviceWorkerModule());

const PATH = "/settings/escalation";

const steps = () => within(screen.getByRole("list", { name: "Escalation order" })).getAllByRole("listitem").map((item) => item.getAttribute("data-testid"));

describe("Escalation order", () => {
  beforeEach(async () => {
    await setLocale("en");
    hub.reset({
      devices: [device({ id: "phone", channels: ["web_push", "email"] }), device({ id: "mac", channels: ["macos_notification"] })],
      chain: ["web_push", "email"],
    });
  });

  it("lists the channels in the order the Hub holds, in the operator's words", async () => {
    renderSettings(PATH);
    await findPageHeading("Escalation order");
    await screen.findByRole("list", { name: "Escalation order" });
    expect(steps()).toEqual([selectors.escalation.step({ channel: "web_push" }), selectors.escalation.step({ channel: "email" })]);
    expect(screen.getByText(en.settings.channel.webPush)).toBeInTheDocument();
    expect(screen.getByText(/Each step waits 5 minutes/)).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.escalation.save)).not.toBeInTheDocument();
  });

  it("reorders with touch-sized controls and saves the whole new order once", async () => {
    const user = userEvent.setup();
    renderSettings(PATH);
    await user.click(await screen.findByRole("button", { name: "Move Email up" }));
    expect(steps()).toEqual([selectors.escalation.step({ channel: "email" }), selectors.escalation.step({ channel: "web_push" })]);
    expect(screen.getByRole("button", { name: "Move Email up" })).toBeDisabled();

    await user.click(screen.getByRole("button", { name: "Save order" }));
    expect(await screen.findByText(en.settings.escalation.saved)).toBeInTheDocument();
    expect(hub.recipientsClient.setEscalationChain).toHaveBeenCalledTimes(1);
    expect(hub.recipientsClient.setEscalationChain).toHaveBeenCalledWith({ channels: ["email", "web_push"] });
    expect(screen.queryByRole("button", { name: "Save order" })).not.toBeInTheDocument();
  });

  it("offers only channels that are not already in the order", async () => {
    const user = userEvent.setup();
    renderSettings(PATH);
    await user.click(await screen.findByRole("button", { name: "Add a channel" }));
    const sheet = await screen.findByRole("dialog");
    expect(within(sheet).getAllByRole("button").map((button) => button.getAttribute("data-testid"))).toContain("escalation-choice-macos_notification");
    expect(within(sheet).queryByTestId(selectors.escalation.choice({ channel: "email" }))).not.toBeInTheDocument();
    await user.click(within(sheet).getByTestId(selectors.escalation.choice({ channel: "macos_notification" })));
    expect(steps()).toEqual([selectors.escalation.step({ channel: "web_push" }), selectors.escalation.step({ channel: "email" }), selectors.escalation.step({ channel: "macos_notification" })]);
  });

  it("says when there is nothing left to add and points to devices", async () => {
    const user = userEvent.setup();
    hub.reset({ devices: [device({ id: "phone", channels: ["email"] })], chain: ["email"] });
    renderSettings(PATH);
    await user.click(await screen.findByRole("button", { name: "Add a channel" }));
    const sheet = await screen.findByRole("dialog");
    expect(within(sheet).getByText(/already in the order/)).toBeInTheDocument();
    expect(within(sheet).getByRole("link", { name: "Open devices and channels" })).toHaveAttribute("href", "/settings/reach");
  });

  it("keeps a saved channel no device offers, flags it, and never drops it silently", async () => {
    hub.reset({ devices: [device({ id: "phone", channels: ["email"] })], chain: ["signal", "email"] });
    renderSettings(PATH);
    const row = await screen.findByTestId(selectors.escalation.step({ channel: "signal" }));
    expect(within(row).getByText(interp(en.settings.channel.other, { channel: "signal" }))).toBeInTheDocument();
    expect(within(row).getByText(en.settings.escalation.notOffered)).toBeInTheDocument();
    expect(steps()).toEqual([selectors.escalation.step({ channel: "signal" }), selectors.escalation.step({ channel: "email" })]);
  });

  it("discards a draft in one tap and returns to what the Hub holds", async () => {
    const user = userEvent.setup();
    renderSettings(PATH);
    await user.click(await screen.findByRole("button", { name: "Remove Email" }));
    expect(steps()).toEqual([selectors.escalation.step({ channel: "web_push" })]);
    await user.click(screen.getByRole("button", { name: "Discard changes" }));
    expect(steps()).toEqual([selectors.escalation.step({ channel: "web_push" }), selectors.escalation.step({ channel: "email" })]);
    expect(hub.recipientsClient.setEscalationChain).not.toHaveBeenCalled();
  });

  it("confirms before saving an empty order, because missed critical decisions would then just expire", async () => {
    const user = userEvent.setup();
    renderSettings(PATH);
    await user.click(await screen.findByRole("button", { name: "Remove Push notification" }));
    await user.click(screen.getByRole("button", { name: "Remove Email" }));
    expect(screen.getByText(en.settings.escalation.emptyTitle)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Save order" }));
    const confirm = await screen.findByRole("alertdialog");
    expect(hub.recipientsClient.setEscalationChain).not.toHaveBeenCalled();
    await user.click(within(confirm).getByRole("button", { name: "Save empty order" }));
    expect(await screen.findByText(en.settings.escalation.saved)).toBeInTheDocument();
    expect(hub.state.chain).toEqual([]);
  });

  it("keeps the draft and says what the Hub still holds when the order did not take", async () => {
    const user = userEvent.setup();
    renderSettings(PATH);
    await user.click(await screen.findByRole("button", { name: "Move Email up" }));
    hub.failNext("setEscalationChain", "reject");
    await user.click(screen.getByRole("button", { name: "Save order" }));
    const problem = await screen.findByTestId(selectors.escalation.problem);
    expect(problem).toHaveTextContent("The new order didn’t save");
    expect(problem).toHaveTextContent("Push notification, then Email");
    expect(steps()).toEqual([selectors.escalation.step({ channel: "email" }), selectors.escalation.step({ channel: "web_push" })]);
    await user.click(screen.getByRole("button", { name: "Save order" }));
    await screen.findByText(en.settings.escalation.saved);
    expect(hub.state.chain).toEqual(["email", "web_push"]);
  });

  it("counts a save whose answer was lost when the read-back shows the new order", async () => {
    const user = userEvent.setup();
    renderSettings(PATH);
    await user.click(await screen.findByRole("button", { name: "Move Email up" }));
    hub.failNext("setEscalationChain", "lose");
    await user.click(screen.getByRole("button", { name: "Save order" }));
    expect(await screen.findByText(en.settings.escalation.saved)).toBeInTheDocument();
  });

  it("does not claim success when the order cannot be read back", async () => {
    const user = userEvent.setup();
    renderSettings(PATH);
    await user.click(await screen.findByRole("button", { name: "Move Email up" }));
    hub.failNext("getEscalationChain", "reject");
    await user.click(screen.getByRole("button", { name: "Save order" }));
    expect(await screen.findByText(en.settings.escalation.unconfirmedTitle)).toBeInTheDocument();
    expect(screen.queryByText(en.settings.escalation.saved)).not.toBeInTheDocument();
  });

  it("asks before an unsaved order is lost to navigation", async () => {
    const user = userEvent.setup();
    const { router } = renderSettings(PATH, "phone");
    await user.click(await screen.findByRole("button", { name: "Move Email up" }));
    await go(router, "/settings");
    const confirm = await screen.findByRole("alertdialog");
    expect(router.state.location.pathname).toBe(PATH);
    await user.click(within(confirm).getByRole("button", { name: "Keep editing" }));
    expect(steps()[0]).toBe(selectors.escalation.step({ channel: "email" }));
    await go(router, "/settings");
    await user.click(within(await screen.findByRole("alertdialog")).getByRole("button", { name: "Discard" }));
    await waitFor(() => expect(router.state.location.pathname).toBe("/settings"));
  });

  it("shows the sign-in card in place of the order when signed out", async () => {
    hub.signOut();
    renderSettings(PATH);
    expect(await screen.findByRole("region", { name: "Sign in" })).toBeInTheDocument();
    expect(hub.recipientsClient.getEscalationChain).not.toHaveBeenCalled();
  });
});
