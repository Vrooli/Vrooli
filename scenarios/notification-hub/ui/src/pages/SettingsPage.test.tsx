import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { en } from "../i18n/catalog";
import { selectors } from "../consts/selectors";
import { setLocale } from "../i18n";
import { expectNoA11yViolations, interp } from "../test-utils";
import { device, hub, quietWindow, type FakeSource } from "../features/settings/mocks/fakeHub";
import { findPageHeading, renderSettings } from "../test-utils/settingsHarness";

vi.mock("../api/notifications", async () => (await import("../features/settings/mocks/fakeHub")).notificationsModule());
vi.mock("../api/preferences", async () => (await import("../features/settings/mocks/fakeHub")).preferencesModule());
vi.mock("../api/serviceWorker", async () => (await import("../features/settings/mocks/fakeHub")).serviceWorkerModule());

const OWNER_EMAIL = "owner@example.test";
const RUNS_FINISHED = "Runs finished";
const RUN_ENDS = "When a run ends";

const row = (key: string) => screen.findByTestId(`settings-row-${key}`);

describe("Settings list", () => {
  beforeEach(async () => {
    await setLocale("en");
    window.localStorage.clear();
    hub.reset({
      devices: [device({ id: "phone", channels: ["web_push"] }), device({ id: "mac", name: "Mac", channels: ["email"] })],
      windows: [1, 2, 3, 4, 5].map((day) => quietWindow(day, { start: "22:00", end: "23:30" })),
      chain: ["web_push", "email"],
    });
  });
  afterEach(() => document.documentElement.removeAttribute("data-theme"));

  it("is one page of rows that say where each setting stands, in plain words", async () => {
    renderSettings("/settings");
    await findPageHeading("Settings");
    await waitFor(() => expect(screen.getByTestId(selectors.settings.row({ key: "reach" }))).toHaveTextContent("2 devices"));
    expect(screen.getByTestId(selectors.settings.row({ key: "quietHours" }))).toHaveTextContent("10:00 PM–11:30 PM, Weekdays");
    expect(screen.getByTestId(selectors.settings.row({ key: "escalation" }))).toHaveTextContent("Push notification, then Email");
    expect(screen.getByTestId(selectors.settings.row({ key: "appearance" }))).toHaveTextContent("System · English");
    expect(screen.getByTestId(selectors.settings.row({ key: "session" }))).toHaveTextContent("Signed in as owner@example.test");
    expect(screen.getByTestId(selectors.settings.row({ key: "sources" }))).toHaveTextContent("None yet");
  });

  it("groups settings by what they are for, with the developer plumbing last and quiet", async () => {
    renderSettings("/settings");
    await findPageHeading("Settings");
    const groups = screen.getAllByRole("heading", { level: 2 }).map((heading) => heading.textContent);
    expect(groups).toEqual(["Reaching you", "This app", "Advanced"]);
    expect(within(screen.getByRole("region", { name: "Advanced" })).getByTestId(selectors.settings.row({ key: "sources" }))).toBeInTheDocument();
  });

  it("opens each setting at its own address", async () => {
    renderSettings("/settings");
    const expected = {
      reach: "/settings/reach",
      quietHours: "/settings/quiet-hours",
      escalation: "/settings/escalation",
      appearance: "/settings/appearance",
      session: "/settings/session",
      sources: "/settings/sources",
    };
    for (const [key, href] of Object.entries(expected)) expect(await row(key)).toHaveAttribute("href", href);
  });

  it("stays usable while signed out: owner settings say so, appearance still works", async () => {
    hub.signOut();
    renderSettings("/settings");
    await findPageHeading("Settings");
    expect(await screen.findByText(en.settings.signedOut.title)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.settings.row({ key: "reach" }))).toHaveTextContent("Sign in to view");
    expect(screen.getByTestId(selectors.settings.row({ key: "quietHours" }))).toHaveTextContent("Sign in to view");
    expect(screen.getByTestId(selectors.settings.row({ key: "session" }))).toHaveTextContent("Not signed in");
    expect(screen.getByTestId(selectors.settings.row({ key: "appearance" }))).toHaveTextContent("System");
    expect(hub.recipientsClient.listDevices).not.toHaveBeenCalled();
    expect(hub.recipientsClient.listQuietWindows).not.toHaveBeenCalled();
  });

  it("says a summary is unavailable instead of leaving it blank when its read fails", async () => {
    hub.failAlways("listQuietWindows");
    renderSettings("/settings");
    await waitFor(() => expect(screen.getByTestId(selectors.settings.row({ key: "quietHours" }))).toHaveTextContent("Couldn’t load"));
    expect(screen.getByTestId(selectors.settings.row({ key: "reach" }))).toHaveTextContent("2 devices");
  });

  it("has no accessibility violations on a phone", async () => {
    const { container } = renderSettings("/settings");
    await waitFor(() => expect(screen.getByTestId(selectors.settings.row({ key: "reach" }))).toHaveTextContent("2 devices"));
    await expectNoA11yViolations(container);
  });
});

describe("Settings on a wide screen", () => {
  beforeEach(async () => {
    await setLocale("en");
    hub.reset({ devices: [device({ id: "phone", channels: ["web_push"] })] });
  });

  it("opens the first setting beside the list instead of an empty page", async () => {
    const { router } = renderSettings("/settings", "desktop");
    await findPageHeading("Devices and channels");
    expect(router.state.location.pathname).toBe("/settings/reach");
    const nav = screen.getByRole("navigation", { name: "Settings" });
    expect(within(nav).getByTestId(selectors.settings.row({ key: "reach" }))).toHaveAttribute("aria-current", "page");
    expect(within(nav).getByTestId(selectors.settings.row({ key: "quietHours" }))).not.toHaveAttribute("aria-current");
  });

  it("keeps each navigation row to one line: a title, with no summary under it", async () => {
    renderSettings("/settings/reach", "desktop");
    await findPageHeading("Devices and channels");
    const row = within(screen.getByRole("navigation", { name: "Settings" })).getByTestId(selectors.settings.row({ key: "quietHours" }));
    expect(row).toHaveTextContent(/^Quiet hours$/);
  });

  it("switches the open setting from the list and keeps one page heading", async () => {
    const user = userEvent.setup();
    renderSettings("/settings/reach", "desktop");
    await findPageHeading("Devices and channels");
    await user.click(within(screen.getByRole("navigation", { name: "Settings" })).getByTestId(selectors.settings.row({ key: "quietHours" })));
    await findPageHeading("Quiet hours");
    expect(screen.getAllByRole("heading", { level: 1 })).toHaveLength(1);
    expect(within(screen.getByRole("navigation", { name: "Settings" })).getByTestId(selectors.settings.row({ key: "quietHours" }))).toHaveAttribute("aria-current", "page");
  });

  it("drops the back link to the list, because the list is already there", async () => {
    renderSettings("/settings/quiet-hours", "desktop");
    await findPageHeading("Quiet hours");
    expect(screen.queryByRole("link", { name: /^Settings$/ })).not.toBeInTheDocument();
  });

  it("gives a phone a back link to the list on the same screen", async () => {
    renderSettings("/settings/quiet-hours", "phone");
    await findPageHeading("Quiet hours");
    expect(screen.getByRole("link", { name: "Settings" })).toHaveAttribute("href", "/settings");
  });
});

describe("Appearance and language", () => {
  beforeEach(async () => {
    await setLocale("en");
    window.localStorage.clear();
    hub.reset();
    hub.signOut();
  });
  afterEach(() => {
    document.documentElement.removeAttribute("data-theme");
    vi.restoreAllMocks();
  });

  it("works without signing in and applies a theme at once", async () => {
    const user = userEvent.setup();
    renderSettings("/settings/appearance");
    await findPageHeading("Appearance and language");
    await user.click(screen.getByRole("radio", { name: "Dark" }));
    expect(document.documentElement).toHaveAttribute("data-theme", "dark");
    expect(screen.getByRole("radio", { name: "Dark" })).toBeChecked();
  });

  it("switches language in place, including to a right-to-left one", async () => {
    const user = userEvent.setup();
    renderSettings("/settings/appearance");
    await findPageHeading("Appearance and language");
    await user.click(screen.getByRole("radio", { name: "العربية" }));
    await findPageHeading("المظهر واللغة");
    expect(document.documentElement).toHaveAttribute("dir", "rtl");
    await user.click(screen.getByRole("radio", { name: "English" }));
    await findPageHeading("Appearance and language");
    expect(document.documentElement).toHaveAttribute("dir", "ltr");
  });

  it("applies a choice and says so when the browser will not remember it", async () => {
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => { throw new Error("denied"); });
    const user = userEvent.setup();
    renderSettings("/settings/appearance");
    await findPageHeading("Appearance and language");
    await user.click(screen.getByRole("radio", { name: "Dark" }));
    expect(document.documentElement).toHaveAttribute("data-theme", "dark");
    expect(await screen.findByText(en.preferenceStorage.title)).toBeInTheDocument();
  });
});

describe("Session", () => {
  beforeEach(async () => {
    await setLocale("en");
    hub.reset();
  });

  it("shows who is signed in, with a way out that asks first", async () => {
    renderSettings("/settings/session");
    await findPageHeading("Session");
    expect(await screen.findByText(interp(en.session.signedInAs, { email: OWNER_EMAIL }))).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Sign out of Hub" })).toBeInTheDocument();
  });

  it("offers the sign-in form in place when signed out", async () => {
    hub.signOut();
    renderSettings("/settings/session");
    await findPageHeading("Session");
    expect(await screen.findByRole("region", { name: "Sign in" })).toBeInTheDocument();
  });
});

describe("Event sources", () => {
  const sources: FakeSource[] = [
    {
      source_scenario: "agent-manager",
      display_name: "Agent Manager",
      icon: "",
      deep_link_base: "",
      declared: true,
      categories: [
        { category: "run-finished", display_name: "Runs finished", description: "When a run ends", declared: true, default_enabled: true, default_min_urgency: "normal" },
        { category: "run-failed", display_name: "Runs failed", description: "", declared: true, default_enabled: false, default_min_urgency: "low" },
      ],
    },
    { source_scenario: "plain-source", display_name: "", icon: "", deep_link_base: "", declared: false, categories: [] },
  ];

  beforeEach(async () => {
    await setLocale("en");
    hub.reset({ sources });
  });

  it("lists each source with its categories, falling back to the scenario name and one switch for a source with none", async () => {
    renderSettings("/settings/sources");
    await findPageHeading("Event sources");
    const agent = await screen.findByTestId(selectors.settings.source({ scenario: "agent-manager" }));
    expect(within(agent).getByText(RUNS_FINISHED)).toBeInTheDocument();
    expect(within(agent).getByText(RUN_ENDS)).toBeInTheDocument();
    const plain = screen.getByTestId(selectors.settings.source({ scenario: "plain-source" }));
    expect(within(plain).getByRole("heading", { name: "plain-source" })).toBeInTheDocument();
    expect(within(plain).getByText(en.settings.sources.everything)).toBeInTheDocument();
  });

  it("saves a switch at once, sending exactly the preference and nothing else", async () => {
    const user = userEvent.setup();
    renderSettings("/settings/sources");
    await user.click(await screen.findByRole("switch", { name: "Allow Runs failed" }));
    await waitFor(() => expect(hub.updateNotificationPreference).toHaveBeenCalledTimes(1));
    expect(hub.updateNotificationPreference).toHaveBeenCalledWith({ source_scenario: "agent-manager", category: "run-failed", enabled: true, min_urgency: "low", channel_override: "" });
    await waitFor(() => expect(screen.getByRole("switch", { name: "Allow Runs failed" })).toBeChecked());
  });

  it("shows the lowest-urgency choice only for what is allowed, and saves a change to it", async () => {
    const user = userEvent.setup();
    renderSettings("/settings/sources");
    await screen.findByTestId(selectors.settings.source({ scenario: "agent-manager" }));
    expect(screen.queryByRole("button", { name: "Lowest urgency for Runs failed" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Lowest urgency for Runs finished" }));
    await user.click(await screen.findByRole("option", { name: "High" }));
    await waitFor(() => expect(hub.state.preferences).toEqual([expect.objectContaining({ category: "run-finished", min_urgency: "high", enabled: true })]));
  });

  it("puts a refused change back and says which one, instead of showing a setting the Hub does not hold", async () => {
    const user = userEvent.setup();
    hub.failNext("updateNotificationPreference", "reject");
    renderSettings("/settings/sources");
    const toggle = await screen.findByRole("switch", { name: "Allow Runs finished" });
    expect(toggle).toBeChecked();
    await user.click(toggle);
    expect(await screen.findByText(interp(en.settings.sources.saveFailed, { name: `Agent Manager · ${RUNS_FINISHED}` }))).toBeInTheDocument();
    await waitFor(() => expect(screen.getByRole("switch", { name: "Allow Runs finished" })).toBeChecked());
  });

  it("says plainly when no app has sent anything yet", async () => {
    hub.reset({ sources: [] });
    renderSettings("/settings/sources");
    expect(await screen.findByText(en.settings.sources.emptyTitle)).toBeInTheDocument();
  });

  it("recovers from a failed read with a retry", async () => {
    const user = userEvent.setup();
    hub.failNext("fetchNotificationCatalog", "reject");
    renderSettings("/settings/sources");
    expect(await screen.findByText(en.settings.sources.loadFailed)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Try again" }));
    expect(await screen.findByTestId(selectors.settings.source({ scenario: "agent-manager" }))).toBeInTheDocument();
  });
});

describe("A session that ends mid-edit", () => {
  const expire = () => act(() => { hub.signOut(); window.dispatchEvent(new Event("notification-hub:session-expired")); });

  beforeEach(async () => {
    await setLocale("en");
    hub.reset({ devices: [device({ id: "phone", channels: ["email"] })], chain: ["web_push", "email"] });
  });

  it("hides owner content, asks to sign in, and brings the same owner's draft back", async () => {
    const user = userEvent.setup();
    renderSettings("/settings/escalation");
    await user.click(await screen.findByRole("button", { name: "Move Email up" }));
    expire();
    expect(await screen.findByText(en.settings.sessionEnded.title)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Move Email up" })).not.toBeInTheDocument();

    await user.type(screen.getByLabelText(en.settings.channel.email), "owner@example.test");
    await user.type(screen.getByLabelText(en.session.password), "secret");
    await user.click(screen.getByRole("button", { name: "Sign in" }));
    await waitFor(() => expect(screen.queryByText(en.settings.sessionEnded.title)).not.toBeInTheDocument());
    const list = screen.getByRole("list", { name: "Escalation order" });
    expect(within(list).getAllByRole("listitem")[0]).toHaveAttribute("data-testid", selectors.escalation.step({ channel: "email" }));
  });

  it("gives a different owner a clean page, not the previous owner's draft", async () => {
    const user = userEvent.setup();
    renderSettings("/settings/escalation");
    await user.click(await screen.findByRole("button", { name: "Move Email up" }));
    expire();
    hub.loginSubject = "someone-else";
    await user.type(await screen.findByLabelText(en.settings.channel.email), "someone-else@example.test");
    await user.type(screen.getByLabelText(en.session.password), "secret");
    await user.click(screen.getByRole("button", { name: "Sign in" }));
    await waitFor(() => expect(screen.queryByText(en.settings.sessionEnded.title)).not.toBeInTheDocument());
    const list = await screen.findByRole("list", { name: "Escalation order" });
    expect(within(list).getAllByRole("listitem")[0]).toHaveAttribute("data-testid", selectors.escalation.step({ channel: "web_push" }));
  });
});
