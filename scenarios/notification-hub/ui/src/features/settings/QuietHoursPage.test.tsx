import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { en } from "../../i18n/catalog";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { hub, quietWindow } from "./mocks/fakeHub";
import { findPageHeading, renderSettings } from "../../test-utils/settingsHarness";

vi.mock("../../api/notifications", async () => (await import("./mocks/fakeHub")).notificationsModule());
vi.mock("../../api/preferences", async () => (await import("./mocks/fakeHub")).preferencesModule());
vi.mock("../../api/serviceWorker", async () => (await import("./mocks/fakeHub")).serviceWorkerModule());

const PATH = "/settings/quiet-hours";

/** Weekdays Monday to Friday, all with the same hours. */
const weekdays = (over: Parameters<typeof quietWindow>[1] = {}) => [1, 2, 3, 4, 5].map((day) => quietWindow(day, over));

async function openEditor(user: ReturnType<typeof userEvent.setup>, name: RegExp) {
  await user.click(await screen.findByRole("button", { name }));
  return screen.findByRole("dialog");
}

const setTime = (dialog: HTMLElement, label: RegExp, value: string) => {
  const input = within(dialog).getByLabelText(label);
  // A time input takes a value, not keystrokes, in jsdom.
  return userEvent.setup().clear(input).then(() => userEvent.setup().type(input, value));
};

describe("Quiet hours", () => {
  beforeEach(async () => {
    await setLocale("en");
    hub.reset();
  });

  it("groups days with the same hours into one schedule a person can read", async () => {
    hub.reset({ windows: [...weekdays(), quietWindow(6, { start: "09:00", end: "11:00" })] });
    renderSettings(PATH);
    await findPageHeading("Quiet hours");
    expect(await screen.findByRole("button", { name: /Weekdays/ })).toHaveTextContent("12:00 PM–1:00 PM");
    expect(screen.getByRole("button", { name: /Sat/ })).toHaveTextContent("9:00 AM–11:00 AM");
  });

  it("explains an empty state in a sentence and offers the one next step", async () => {
    renderSettings(PATH);
    expect(await screen.findByText(en.settings.quietHours.emptyTitle)).toBeInTheDocument();
    expect(screen.getByText(en.settings.quietHours.emptyDetail)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add quiet hours" })).toBeInTheDocument();
  });

  it("adds a schedule across several days and trusts only the read-back", async () => {
    const user = userEvent.setup();
    renderSettings(PATH);
    const dialog = await openEditor(user, /Add quiet hours/);
    // A new schedule starts on the weekdays, so only the hours need entering.
    await setTime(dialog, /From/, "22:00");
    await setTime(dialog, /Until/, "23:30");
    await user.click(within(dialog).getByRole("button", { name: "Save" }));

    await waitFor(() => expect(hub.state.windows.map((window) => window.weekday).sort()).toEqual([1, 2, 3, 4, 5]));
    expect(hub.state.windows.every((window) => window.start === "22:00" && window.end === "23:30")).toBe(true);
    expect(await screen.findByText(en.settings.quietHours.saved)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Weekdays/ })).toHaveTextContent("10:00 PM–11:30 PM");
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("edits one schedule without touching another", async () => {
    const user = userEvent.setup();
    hub.reset({ windows: [...weekdays(), quietWindow(6, { start: "09:00", end: "11:00" })] });
    renderSettings(PATH);
    const dialog = await openEditor(user, /Weekdays/);
    await setTime(dialog, /Until/, "14:00");
    await user.click(within(dialog).getByRole("button", { name: "Save" }));
    await screen.findByText(en.settings.quietHours.saved);
    expect(hub.state.windows.filter((window) => window.weekday <= 5).every((window) => window.end === "14:00")).toBe(true);
    expect(hub.state.windows.find((window) => window.weekday === 6)).toMatchObject({ start: "09:00", end: "11:00" });
  });

  it("moves a day out of a schedule by deleting only that day", async () => {
    const user = userEvent.setup();
    hub.reset({ windows: weekdays() });
    renderSettings(PATH);
    const dialog = await openEditor(user, /Weekdays/);
    await user.click(within(dialog).getByRole("button", { name: "Friday" }));
    await user.click(within(dialog).getByRole("button", { name: "Save" }));
    await screen.findByText(en.settings.quietHours.saved);
    expect(hub.state.windows.map((window) => window.weekday).sort()).toEqual([1, 2, 3, 4]);
    expect(hub.recipientsClient.setQuietWindow).not.toHaveBeenCalled();
    expect(hub.recipientsClient.deleteQuietWindow).toHaveBeenCalledTimes(1);
  });

  it("asks before removing a whole schedule, then removes every day of it", async () => {
    const user = userEvent.setup();
    hub.reset({ windows: [...weekdays(), quietWindow(6, { start: "09:00", end: "11:00" })] });
    renderSettings(PATH);
    const dialog = await openEditor(user, /Weekdays/);
    await user.click(within(dialog).getByRole("button", { name: "Remove" }));
    const confirm = await screen.findByRole("alertdialog");
    expect(hub.recipientsClient.deleteQuietWindow).not.toHaveBeenCalled();
    await user.click(within(confirm).getByRole("button", { name: "Remove quiet hours" }));
    expect(await screen.findByText(en.settings.quietHours.removed)).toBeInTheDocument();
    expect(hub.state.windows.map((window) => window.weekday)).toEqual([6]);
  });

  it("keeps a day that belongs to another schedule out of reach", async () => {
    const user = userEvent.setup();
    hub.reset({ windows: [...weekdays(), quietWindow(6, { start: "09:00", end: "11:00" })] });
    renderSettings(PATH);
    const dialog = await openEditor(user, /Weekdays/);
    expect(within(dialog).getByRole("button", { name: "Saturday" })).toBeDisabled();
    expect(within(dialog).getByRole("button", { name: "Monday" })).toBeEnabled();
  });

  it("saves overnight hours and explains when they end", async () => {
    const user = userEvent.setup();
    renderSettings(PATH);
    const dialog = await openEditor(user, /Add quiet hours/);
    await setTime(dialog, /From/, "22:00");
    await setTime(dialog, /Until/, "07:00");
    expect(within(dialog).queryByRole("alert")).not.toBeInTheDocument();
    expect(await within(dialog).findByText(/start on each day you choose and end the next morning, at 7:00 AM/)).toBeInTheDocument();
    await user.click(within(dialog).getByRole("button", { name: "Save" }));
    expect(await screen.findByText(en.settings.quietHours.saved)).toBeInTheDocument();
    expect(hub.state.windows.map((window) => [window.weekday, window.start, window.end])).toEqual([1, 2, 3, 4, 5].map((day) => [day, "22:00", "07:00"]));
    expect(await screen.findByRole("button", { name: /Weekdays/ })).toHaveTextContent("10:00 PM–7:00 AM");
  });

  it("will not save equal start and end times", async () => {
    const user = userEvent.setup();
    renderSettings(PATH);
    const dialog = await openEditor(user, /Add quiet hours/);
    await setTime(dialog, /Until/, "12:00");
    expect(await within(dialog).findByRole("alert")).toHaveTextContent("Start and end must be different times.");
    expect(within(dialog).getByRole("button", { name: "Save" })).toBeDisabled();
  });

  it("names the days that did not save, keeps the draft, and a retry redoes only those", async () => {
    const user = userEvent.setup();
    renderSettings(PATH);
    const dialog = await openEditor(user, /Add quiet hours/);
    await setTime(dialog, /Until/, "14:00");
    // Monday and Tuesday land; Wednesday's write fails and the run stops there.
    hub.failNext("setQuietWindow", "reject", 1, 2);
    await user.click(within(dialog).getByRole("button", { name: "Save" }));

    const problem = await within(dialog).findByTestId(selectors.quietHours.problem);
    expect(problem).toHaveTextContent("Not everything saved");
    expect(problem).toHaveTextContent("Wednesday, Thursday, and Friday");
    expect(problem).toHaveTextContent("Saved: Monday and Tuesday.");
    expect(hub.state.windows.map((window) => window.weekday).sort()).toEqual([1, 2]);
    expect(screen.queryByText(en.settings.quietHours.saved)).not.toBeInTheDocument();

    hub.recipientsClient.setQuietWindow.mockClear();
    await user.click(within(dialog).getByRole("button", { name: "Save" }));
    await screen.findByText(en.settings.quietHours.saved);
    expect(hub.state.windows.map((window) => window.weekday).sort()).toEqual([1, 2, 3, 4, 5]);
    // Days the Hub already held with these hours are not written again.
    expect(hub.recipientsClient.setQuietWindow.mock.calls.map(([request]) => (request as { weekday: number }).weekday)).toEqual([3, 4, 5]);
  });

  it("counts a save whose answer was lost but whose read-back confirms it", async () => {
    const user = userEvent.setup();
    renderSettings(PATH);
    const dialog = await openEditor(user, /Add quiet hours/);
    await user.click(within(dialog).getByRole("button", { name: "Saturday" }));
    await user.click(within(dialog).getByRole("button", { name: "Monday" }));
    await user.click(within(dialog).getByRole("button", { name: "Tuesday" }));
    await user.click(within(dialog).getByRole("button", { name: "Wednesday" }));
    await user.click(within(dialog).getByRole("button", { name: "Thursday" }));
    await user.click(within(dialog).getByRole("button", { name: "Friday" }));
    await setTime(dialog, /Until/, "14:00");
    hub.failNext("setQuietWindow", "lose");
    await user.click(within(dialog).getByRole("button", { name: "Save" }));
    expect(await screen.findByText(en.settings.quietHours.saved)).toBeInTheDocument();
    expect(hub.state.windows.map((window) => window.weekday)).toEqual([6]);
  });

  it("does not claim success when the read-back fails", async () => {
    const user = userEvent.setup();
    renderSettings(PATH);
    const dialog = await openEditor(user, /Add quiet hours/);
    await setTime(dialog, /Until/, "14:00");
    hub.failNext("listQuietWindows", "reject");
    await user.click(within(dialog).getByRole("button", { name: "Save" }));
    expect(await within(dialog).findByText(en.settings.quietHours.unconfirmedTitle)).toBeInTheDocument();
    expect(screen.queryByText(en.settings.quietHours.saved)).not.toBeInTheDocument();
  });

  it("asks before throwing away an edit", async () => {
    const user = userEvent.setup();
    hub.reset({ windows: weekdays() });
    renderSettings(PATH);
    const dialog = await openEditor(user, /Weekdays/);
    await user.click(within(dialog).getByRole("button", { name: "Cancel" }));
    expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();

    const again = await openEditor(user, /Weekdays/);
    await user.click(within(again).getByRole("button", { name: "Friday" }));
    await user.click(within(again).getByRole("button", { name: "Cancel" }));
    const confirm = await screen.findByRole("alertdialog");
    await user.click(within(confirm).getByRole("button", { name: "Keep editing" }));
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    await user.click(within(screen.getByRole("dialog")).getByRole("button", { name: "Cancel" }));
    await user.click(within(await screen.findByRole("alertdialog")).getByRole("button", { name: "Discard" }));
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
    expect(hub.state.windows).toHaveLength(5);
  });

  it("recovers from a failed read with a retry that works", async () => {
    const user = userEvent.setup();
    hub.reset({ windows: weekdays() });
    hub.failNext("listQuietWindows", "reject");
    renderSettings(PATH);
    expect(await screen.findByText(en.settings.quietHours.loadFailed)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Try again" }));
    expect(await screen.findByRole("button", { name: /Weekdays/ })).toBeInTheDocument();
  });

  it("shows the sign-in card in place of the schedules when signed out, and reads nothing", async () => {
    hub.signOut();
    renderSettings(PATH);
    expect(await screen.findByRole("region", { name: "Sign in" })).toBeInTheDocument();
    expect(hub.recipientsClient.listQuietWindows).not.toHaveBeenCalled();
  });
});
