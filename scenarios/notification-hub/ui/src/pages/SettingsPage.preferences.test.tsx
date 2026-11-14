import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient } from "@tanstack/react-query";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import { renderWithProviders } from "../test-utils";
import { SettingsPage } from "./SettingsPage";
import { AppShell } from "../layout/AppShell";
import { fetchEventIntegrationConfig, updateEventIntegrationConfig } from "../api/integrations";
import { fetchNotificationCatalog, updateNotificationPreference, type NotificationPreference } from "../api/preferences";
import { identityClient, recipientsClient } from "../api/notifications";
import { detectInitialLocale, getCurrentLocale, setLocale } from "../i18n";
import { readLocalPreference } from "../theme/ThemeProvider";
import { selectors } from "../consts/selectors";

/** The SettingsList row holding the category label, enable switch, and urgency select. */
function preferenceRowFor(label: RegExp): HTMLElement {
  const category = screen.getByText(label);
  const row = category.closest("[data-rcl-settings-row]");
  if (!(row instanceof HTMLElement)) throw new Error(`No settings row found for ${label}`);
  return row;
}

// Keep the session signed in and the device/event surfaces quiet so the
// "Notification preferences" section is the only thing under test.
vi.mock("../api/notifications", () => ({
  identityClient: {
    getSession: vi.fn(() =>
      Promise.resolve({ signedIn: true, subject: "owner-1", email: "owner@example.test" }),
    ),
    login: vi.fn(),
    logout: vi.fn(),
  },
  recipientsClient: {
    listDevices: vi.fn().mockResolvedValue({ devices: [] }),
    listQuietWindows: vi.fn().mockResolvedValue({ windows: [] }),
    setQuietWindow: vi.fn().mockResolvedValue({ id: "quiet-1" }),
    deleteQuietWindow: vi.fn().mockResolvedValue({}),
    getEscalationChain: vi.fn().mockResolvedValue({ steps: [] }),
    setEscalationChain: vi.fn().mockResolvedValue({ steps: [] }),
  },
  registerBrowserPushSubscription: vi.fn(),
}));

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
    fetchNotificationCatalog: vi.fn(),
    updateNotificationPreference: vi.fn(),
  };
});

const fetchCatalog = vi.mocked(fetchNotificationCatalog);
const updatePref = vi.mocked(updateNotificationPreference);
const fetchIntegration = vi.mocked(fetchEventIntegrationConfig);
const updateIntegration = vi.mocked(updateEventIntegrationConfig);

const catalogWithOneSource = {
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

describe("SettingsPage notification preferences", () => {
  beforeEach(async () => {
    window.localStorage.clear();
    readLocalPreference("theme", "vrooli.theme");
    readLocalPreference("locale", "vrooli.locale");
    await setLocale("en");
    vi.mocked(identityClient.getSession).mockResolvedValue({
      signedIn: true,
      subject: "owner-1",
      email: "owner@example.test",
    } as never);
    vi.mocked(recipientsClient.listDevices).mockResolvedValue({ devices: [] } as never);
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.clearAllMocks();
  });

  it("focuses the escalation editor and returns to Settings without remounting its state", async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    queryClient.setQueryData(["owner-session"], { signedIn: true, subject: "owner-1", email: "owner@example.test" });
    await setLocale("en");
    fetchCatalog.mockResolvedValue({ sources: [], preferences: [] } as never);
    vi.mocked(recipientsClient.getEscalationChain).mockResolvedValue({ steps: [{ ordinal: 0, channel: "email" }] } as never);
    vi.mocked(recipientsClient.listDevices).mockResolvedValue({ devices: [{ id: "owner-phone", channels: ["email"] }] } as never);
    renderWithProviders(<SettingsPage />, { queryClient, initialEntries: ["/settings?view=all"] });

    const preferencesHeading = document.getElementById("settings-heading");
    expect(preferencesHeading?.tagName).toBe("H2");
    expect(preferencesHeading).toBeVisible();
    expect(screen.queryByRole("list", { name: "Escalation order" })).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Edit order" }));
    expect(await screen.findByRole("list", { name: "Escalation order" })).toBeInTheDocument();
    expect(screen.getByText("Notification preferences")).not.toBeVisible();
    expect(within(screen.getByRole("list", { name: "Escalation order" })).getByText(/Email/)).toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Back to Settings" }));
    expect(preferencesHeading).toBeVisible();
    expect(screen.getByText("Notification preferences")).toBeInTheDocument();
    expect(screen.queryByRole("list", { name: "Escalation order" })).not.toBeInTheDocument();
  });

  it("composes the selected weekday into the shell header and stages the draft when returning to Settings", async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    queryClient.setQueryData(["owner-session"], { signedIn: true, subject: "owner-1", email: "owner@example.test" });
    fetchCatalog.mockResolvedValue({ sources: [], preferences: [] } as never);
    vi.mocked(recipientsClient.listQuietWindows).mockResolvedValue({ windows: [
      { id: "quiet-demo-1", weekday: 1, start: "13:00", end: "14:00", timezone: "Etc/UTC", criticalOverride: false },
    ] } as never);
    const user = userEvent.setup();
    renderWithProviders(
      <MemoryRouter initialEntries={["/settings"]}>
        <Routes>
          <Route path="/settings" element={<AppShell />}>
            <Route index element={<SettingsPage />} />
          </Route>
        </Routes>
      </MemoryRouter>,
      { queryClient, withoutRouter: true },
    );

    const shellMain = document.getElementById(`${selectors.layout.shell}-main`)!;
    shellMain.scrollTop = 456;
    await user.click(await screen.findByTestId("quiet-hours-edit-day-1"));
    await waitFor(() => expect(shellMain.scrollTop).toBe(0));
    expect(shellMain).toHaveClass("quiet-editor-main");
    expect(document.querySelector("[data-rcl-app-shell]")).toHaveClass("quiet-editor-shell");
    expect(document.querySelector("[data-rcl-app-shell]")).toHaveAttribute("data-main-mode", "fill");
    expect(screen.getByTestId("quiet-hours-root").parentElement).toHaveClass("flex", "min-h-0", "flex-1", "flex-col");
    const header = screen.getByTestId(`${selectors.layout.shell}-header`);
    expect(within(header).getByRole("button", { name: "Week" })).toBeInTheDocument();
    expect(within(header).getByText("Monday")).toBeInTheDocument();
    expect(within(header).getByTestId(`${selectors.layout.shell}-menu`)).toBeInTheDocument();
    expect(within(header).queryByText("Chimesail")).not.toBeInTheDocument();
    const duplicatePreferences = screen.queryByText("Notification preferences");
    if (duplicatePreferences) expect(duplicatePreferences).not.toBeVisible();

    const start = screen.getByTestId("quiet-hours-start-1");
    await user.clear(start);
    await user.type(start, "13:30");
    await user.click(within(header).getByRole("button", { name: "Week" }));

    expect(screen.getByTestId("quiet-hours-summary-1")).toHaveTextContent("13:30–14:00 · Etc/UTC");
    expect(screen.queryByText("Notification preferences")).not.toBeVisible();
    const weeklyHeader = screen.getByTestId(`${selectors.layout.shell}-header`);
    await user.click(within(weeklyHeader).getByRole("button", { name: "Settings" }));
    expect(screen.getByText("Notification preferences")).toBeVisible();
    expect(document.querySelector("[data-rcl-app-shell]")).toHaveAttribute("data-main-mode", "scroll");
    await waitFor(() => expect(screen.getByTestId(selectors.quietHours.openWeek)).toHaveFocus());
    expect(recipientsClient.setQuietWindow).not.toHaveBeenCalled();

    document.documentElement.dir = "rtl";
    await user.click(screen.getByTestId("quiet-hours-edit-day-1"));
    const rtlHeader = screen.getByTestId(`${selectors.layout.shell}-header`);
    expect(within(rtlHeader).getByRole("button", { name: "Week" }).querySelector("[aria-hidden=true]")).toHaveTextContent("→");
    await user.click(within(rtlHeader).getByRole("button", { name: "Week" }));
    document.documentElement.dir = "ltr";
  });

  it("falls back safely when the initial locale storage read is denied", () => {
    vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
      throw new DOMException("blocked", "SecurityError");
    });

    expect(detectInitialLocale()).toBe("en");
    expect(readLocalPreference("locale", "vrooli.locale")).toBeNull();
  });

  it("applies theme and every supported locale in the current page when browser writes fail", async () => {
    await setLocale("en");
    const deniedWrite = vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new DOMException("blocked", "SecurityError");
    });
    fetchCatalog.mockResolvedValue({ sources: [], preferences: [] });

    renderWithProviders(<SettingsPage />);
    fireEvent.click(screen.getByRole("radio", { name: "Dark" }));
    expect(document.documentElement.getAttribute("data-theme")).toBe("dark");
    const warning = (await screen.findByText("Preference applied here")).closest("[role=status]")!;
    expect(warning).toHaveTextContent("Preference applied here");
    expect(warning).toHaveTextContent("This browser couldn’t save the choice.");
    expect(warning).toHaveTextContent("No account preference was changed.");
    const themeSelect = screen.getByRole("radio", { name: "Dark" });
    expect(themeSelect.compareDocumentPosition(warning) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();

    fireEvent.click(screen.getByRole("radio", { name: "日本語" }));
    await waitFor(() => expect(document.documentElement.lang).toBe("ja"));
    expect(document.documentElement.dir).toBe("ltr");
    expect(getCurrentLocale()).toBe("ja");
    expect(screen.getByText("このページに設定を適用しました")).toBeInTheDocument();
    expect(deniedWrite).toHaveBeenCalledWith("vrooli.locale", "ja");

    fireEvent.click(screen.getByRole("radio", { name: "العربية" }));
    await waitFor(() => expect(document.documentElement.lang).toBe("ar"));
    expect(document.documentElement.dir).toBe("rtl");
    expect(getCurrentLocale()).toBe("ar");
    expect(deniedWrite).toHaveBeenCalledWith("vrooli.locale", "ar");
    expect(screen.getByText("طُبّق التفضيل هنا").closest("[role=status]")).toHaveTextContent("تعذّر على هذا المتصفح حفظ اختيارك");
  });

  it("persists a locale after storage recovers and clears the shared warning", async () => {
    await setLocale("en");
    const nativeSetItem = Storage.prototype.setItem;
    let denyWrites = true;
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(function (this: Storage, key: string, value: string) {
      if (denyWrites) throw new DOMException("blocked", "SecurityError");
      return nativeSetItem.call(this, key, value);
    });
    fetchCatalog.mockResolvedValue({ sources: [], preferences: [] });

    renderWithProviders(<SettingsPage />);
    fireEvent.click(screen.getByRole("radio", { name: "日本語" }));
    await waitFor(() => expect(document.documentElement.lang).toBe("ja"));
    expect(screen.getByText("このページに設定を適用しました")).toBeInTheDocument();

    denyWrites = false;
    fireEvent.click(screen.getByRole("radio", { name: "العربية" }));
    await waitFor(() => expect(document.documentElement.lang).toBe("ar"));
    expect(window.localStorage.getItem("vrooli.locale")).toBe("ar");
    expect(screen.queryByText("طُبّق التفضيل هنا")).not.toBeInTheDocument();
  });

  it("lists each catalog source behind a branded header and its categories with urgency controls", async () => {
    fetchCatalog.mockResolvedValue(catalogWithOneSource);

    renderWithProviders(<SettingsPage />);

    expect(
      await screen.findByRole("heading", { name: "Git Control Tower" }),
    ).toBeInTheDocument();
    expect(screen.getByText(/Review requests/)).toBeInTheDocument();
    // The source is grouped under a testid'd card with a branded (lettered) avatar.
    const card = screen.getByTestId("notification-source-git-control-tower");
    expect(within(card).getByRole("img", { name: "Git Control Tower" })).toBeInTheDocument();
    // One category row exposes a min-urgency select.
    const select = within(card).getByRole("combobox");
    expect(select).toBeInTheDocument();
    // The urgency ladder is rendered as options.
    expect(screen.getAllByRole("option", { name: "Critical" }).length).toBeGreaterThanOrEqual(1);
  });

  it("shows recipient preferences and the Reach entry while signed out but disables writes", async () => {
    vi.mocked(identityClient.getSession).mockResolvedValue({ signedIn: false } as never);
    vi.mocked(recipientsClient.listDevices).mockResolvedValue({
      devices: [{ id: "device-1", name: "Phone", machineId: "local", channels: ["web_push"] }],
      vapidPublicKey: "public-key",
    } as never);
    fetchCatalog.mockResolvedValue(catalogWithOneSource);

    renderWithProviders(<SettingsPage />);

    expect(await screen.findByRole("heading", { name: "Sign in to manage settings" })).toBeInTheDocument();
    await userEvent.click(screen.getByRole("link", { name: /Settings/ }));

    expect(await screen.findByRole("heading", { name: "Git Control Tower" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Manage devices and channels" })).toHaveAttribute("href", "/settings/reach");
    const appearanceHeading = screen.getByRole("heading", { name: "Appearance" });
    const languageHeading = screen.getByRole("heading", { name: "Language" });
    const reachLink = screen.getByRole("link", { name: "Manage devices and channels" });
    expect(appearanceHeading.compareDocumentPosition(reachLink) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(languageHeading.compareDocumentPosition(reachLink) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    const row = preferenceRowFor(/Review requests/);
    expect(within(row).getByRole("switch")).toBeDisabled();
    expect(within(row).getByRole("combobox")).toBeDisabled();
    expect(updatePref).not.toHaveBeenCalled();
    expect(await screen.findByText(/Sign in to change notification preferences/)).toBeInTheDocument();
  });

  it("separates owner access from browser-only preferences and keeps the broader read-only settings reachable", async () => {
    vi.mocked(identityClient.getSession).mockResolvedValue({ signedIn: false } as never);
    fetchCatalog.mockResolvedValue(catalogWithOneSource);

    renderWithProviders(<SettingsPage />);

    expect(await screen.findByText("Devices, quiet hours and escalation need a verified owner session.")).toBeInTheDocument();
    expect(fetchCatalog).not.toHaveBeenCalled();
    expect(screen.queryByTestId(selectors.session.form)).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole("link", { name: /Appearance & language/ }));
    expect(screen.getByRole("heading", { name: "Appearance & language" })).toBeInTheDocument();
    expect(screen.getByText("This browser only")).toBeInTheDocument();
    expect(screen.getByRole("combobox", { name: "Theme" })).toBeInTheDocument();
    expect(screen.getByRole("combobox", { name: "Language" })).toBeInTheDocument();
    expect(screen.getByText("Display language")).toBeInTheDocument();
    expect(screen.getAllByRole("link", { name: /Settings/ }).map((link) => link.getAttribute("href"))).toEqual(expect.arrayContaining(["/settings", "/settings?view=all"]));
    expect(screen.queryByTestId(selectors.session.form)).not.toBeInTheDocument();
    expect(fetchCatalog).not.toHaveBeenCalled();
    await userEvent.click(screen.getByRole("button", { name: "Sign in" }));
    expect(await screen.findByTestId(selectors.session.form)).toBeInTheDocument();
    expect(identityClient.login).not.toHaveBeenCalled();
    await userEvent.click(screen.getByRole("button", { name: /Back to owner session/ }));
    await userEvent.click(screen.getAllByRole("link", { name: /Settings/ }).find((link) => link.getAttribute("href") === "/settings?view=all")!);
    expect(await screen.findByRole("heading", { name: "Git Control Tower" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Manage devices and channels" })).toHaveAttribute("href", "/settings/reach");
    expect(within(preferenceRowFor(/Review requests/)).getByRole("switch")).toBeDisabled();
    expect(updatePref).not.toHaveBeenCalled();
  });

  it("applies appearance and RTL language from browser-only native selectors", async () => {
    vi.mocked(identityClient.getSession).mockResolvedValue({ signedIn: false } as never);
    fetchCatalog.mockResolvedValue({ sources: [], preferences: [] } as never);
    renderWithProviders(<SettingsPage />);

    expect(await screen.findByText("Devices, quiet hours and escalation need a verified owner session.")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("link", { name: /Appearance & language/ }));
    await userEvent.selectOptions(screen.getByRole("combobox", { name: "Theme" }), "dark");
    await waitFor(() => expect(document.documentElement.getAttribute("data-theme")).toBe("dark"));
    await userEvent.selectOptions(screen.getByRole("combobox", { name: "Language" }), "ar");
    await waitFor(() => expect(document.documentElement.lang).toBe("ar"));

    expect(document.documentElement.dir).toBe("rtl");
    expect(screen.getByText("المظهر")).toBeInTheDocument();
    expect(window.localStorage.getItem("vrooli.theme")).toBe("dark");
    expect(window.localStorage.getItem("vrooli.locale")).toBe("ar");
    expect(fetchCatalog).not.toHaveBeenCalled();
    expect(updatePref).not.toHaveBeenCalled();
  });

  it("keeps native appearance selection in memory and explains denied local storage", async () => {
    vi.mocked(identityClient.getSession).mockResolvedValue({ signedIn: false } as never);
    fetchCatalog.mockResolvedValue({ sources: [], preferences: [] } as never);
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new DOMException("blocked", "SecurityError");
    });
    renderWithProviders(<SettingsPage />);

    expect(await screen.findByText("Devices, quiet hours and escalation need a verified owner session.")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("link", { name: /Appearance & language/ }));
    await userEvent.selectOptions(screen.getByRole("combobox", { name: "Theme" }), "dark");

    expect(document.documentElement.getAttribute("data-theme")).toBe("dark");
    const warning = screen.getByText("Preference applied here").closest("[role=status]")!;
    expect(warning).toHaveTextContent("Preference applied here");
    expect(warning).toHaveTextContent("This browser couldn’t save the choice.");
    expect(screen.getByText(/No account preference was changed/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Continue" })).toHaveAttribute("href", "/settings?view=all");
    expect(fetchCatalog).not.toHaveBeenCalled();
    expect(updatePref).not.toHaveBeenCalled();
  });

  it("keeps an active owner in the browser-only preferences route with cookie session controls", async () => {
    vi.mocked(identityClient.getSession).mockResolvedValue({
      signedIn: true,
      subject: "owner-1",
      email: "owner@example.test",
    } as never);

    renderWithProviders(<SettingsPage />, { initialEntries: ["/settings?view=preferences"] });

    expect(await screen.findByRole("heading", { name: "Preferences and session" })).toBeInTheDocument();
    expect(screen.getByText("Appearance and language changes apply only in this browser.")).toBeInTheDocument();
    expect(screen.getByRole("combobox", { name: "Theme" })).toBeInTheDocument();
    expect(screen.getByText("Hub session active")).toBeInTheDocument();
    expect(screen.getByText(/owner@example\.test/)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.session.signOut)).toBeInTheDocument();
    expect(fetchCatalog).not.toHaveBeenCalled();
    expect(fetchIntegration).not.toHaveBeenCalled();
  });

  it("hides protected settings on session expiry and offers explicit same-owner recovery", async () => {
    fetchCatalog.mockResolvedValue({ sources: [], preferences: [] } as never);
    const view = renderWithProviders(<SettingsPage />, { initialEntries: ["/settings?view=all"] });

    expect(await screen.findByTestId(selectors.session.signOut)).toBeInTheDocument();
    fetchCatalog.mockClear();
    window.dispatchEvent(new Event("notification-hub:session-expired"));
    view.queryClient.setQueryData(["owner-session"], { signedIn: false, subject: "", email: "" });

    expect(await screen.findByRole("heading", { name: "Refresh your session" })).toBeInTheDocument();
    expect(screen.getByText(/Sign in with the same owner account/)).toBeInTheDocument();
    expect(screen.getByText(/Nothing is submitted automatically/)).toBeInTheDocument();
    expect(screen.getByText(/A different owner cannot reopen/)).toBeInTheDocument();
    expect(screen.getByText("Preferences and session")).not.toBeVisible();
    expect(fetchCatalog).not.toHaveBeenCalled();
    expect(screen.getByRole("button", { name: "Sign in" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Preferences" })).toHaveAttribute("href", "/settings?view=preferences");

    await userEvent.click(screen.getByRole("button", { name: "Sign in" }));
    const forms = await screen.findAllByTestId(selectors.session.form);
    expect(forms.some((form) => form.closest("[hidden]") === null)).toBe(true);
  });

  it("keeps a protected settings draft through expiry for its owner and isolates it from another owner", async () => {
    const user = userEvent.setup();
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    queryClient.setQueryData(["owner-session"], { signedIn: true, subject: "owner-1", email: "owner@example.test" });
    vi.mocked(recipientsClient.listQuietWindows).mockResolvedValue({ windows: [] } as never);
    fetchCatalog.mockResolvedValue({ sources: [], preferences: [] } as never);
    renderWithProviders(<SettingsPage />, { queryClient, initialEntries: ["/settings?view=all"] });

    await user.click(await screen.findByTestId("quiet-hours-edit-day-0"));
    await user.click(screen.getByTestId("quiet-hours-enabled-0"));
    await user.clear(screen.getByTestId("quiet-hours-start-0"));
    await user.type(screen.getByTestId("quiet-hours-start-0"), "09:00");
    await user.clear(screen.getByTestId("quiet-hours-end-0"));
    await user.type(screen.getByTestId("quiet-hours-end-0"), "17:00");
    expect(recipientsClient.setQuietWindow).not.toHaveBeenCalled();
    expect(recipientsClient.deleteQuietWindow).not.toHaveBeenCalled();
    expect(recipientsClient.setEscalationChain).not.toHaveBeenCalled();

    window.dispatchEvent(new Event("notification-hub:session-expired"));
    queryClient.setQueryData(["owner-session"], { signedIn: false, subject: "", email: "" });
    expect(await screen.findByText("Your settings draft stays in this tab")).toBeInTheDocument();
    const hiddenDraft = document.querySelector('[data-testid="quiet-hours-start-0"]') as HTMLInputElement;
    expect(hiddenDraft).toHaveValue("09:00");
    expect(hiddenDraft.closest("[hidden]")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Sign in" }));
    queryClient.setQueryData(["owner-session"], { signedIn: true, subject: "owner-1", email: "owner@example.test" });
    await waitFor(() => expect(screen.getByTestId("quiet-hours-start-0")).toBeVisible());
    expect(screen.getByTestId("quiet-hours-start-0")).toHaveValue("09:00");
    expect(recipientsClient.setQuietWindow).not.toHaveBeenCalled();
    expect(recipientsClient.deleteQuietWindow).not.toHaveBeenCalled();
    expect(recipientsClient.setEscalationChain).not.toHaveBeenCalled();

    window.dispatchEvent(new Event("notification-hub:session-expired"));
    queryClient.setQueryData(["owner-session"], { signedIn: false, subject: "", email: "" });
    await screen.findByText("Your settings draft stays in this tab");
    await user.click(screen.getByRole("button", { name: "Sign in" }));
    queryClient.setQueryData(["owner-session"], { signedIn: true, subject: "owner-2", email: "other@example.test" });

    await user.click(await screen.findByTestId("quiet-hours-edit-day-0"));
    await user.click(screen.getByTestId("quiet-hours-enabled-0"));
    expect(screen.getByTestId("quiet-hours-start-0")).toHaveValue("22:00");
    expect(recipientsClient.setQuietWindow).not.toHaveBeenCalled();
    expect(recipientsClient.deleteQuietWindow).not.toHaveBeenCalled();
    expect(recipientsClient.setEscalationChain).not.toHaveBeenCalled();
  });

  it("lets the owner leave the expiry screen for browser-only preferences", async () => {
    const view = renderWithProviders(<SettingsPage />, { initialEntries: ["/settings?view=preferences"] });
    expect(await screen.findByRole("heading", { name: "Preferences and session" })).toBeInTheDocument();
    window.dispatchEvent(new Event("notification-hub:session-expired"));
    view.queryClient.setQueryData(["owner-session"], { signedIn: false, subject: "", email: "" });

    expect(await screen.findByRole("heading", { name: "Refresh your session" })).toBeInTheDocument();
    await userEvent.click(screen.getByRole("link", { name: "Preferences" }));

    expect(await screen.findByRole("heading", { name: "Appearance & language" })).toBeInTheDocument();
    expect(screen.getByRole("combobox", { name: "Theme" })).toBeInTheDocument();
  });

  it("keeps browser-only selectors visible while session identity is being checked", async () => {
    let resolveSession!: (value: unknown) => void;
    vi.mocked(identityClient.getSession).mockReturnValue(new Promise((resolve) => { resolveSession = resolve; }) as never);

    renderWithProviders(<SettingsPage />, { initialEntries: ["/settings?view=preferences"] });

    expect(screen.getByRole("combobox", { name: "Theme" })).toBeInTheDocument();
    expect(screen.queryByText("No Hub session is active")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Sign in" })).not.toBeInTheDocument();
    expect(fetchCatalog).not.toHaveBeenCalled();

    resolveSession({ signedIn: false });
    expect(await screen.findByText("No Hub session is active")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Sign in" })).toBeInTheDocument();
  });

  it("keeps integration settings locked without reading or changing the unsafe configuration route", async () => {
    fetchCatalog.mockResolvedValue({ sources: [], preferences: [] });

    renderWithProviders(<SettingsPage />);

    expect(await screen.findByRole("status", { name: "Event integration: Editing is locked" })).toHaveTextContent("Editing is locked");
    expect(screen.getByText(/No integration configuration was loaded or changed here/)).toBeInTheDocument();
    expect(screen.queryByLabelText("Events API base")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Save event integration" })).not.toBeInTheDocument();
    expect(fetchIntegration).not.toHaveBeenCalled();
    expect(updateIntegration).not.toHaveBeenCalled();
  });

  it("PUTs a preference when the enable switch is toggled", async () => {
    fetchCatalog.mockResolvedValue(catalogWithOneSource);
    updatePref.mockResolvedValue({} as never);

    renderWithProviders(<SettingsPage />);
    await screen.findByText(/Review requests/);

    const categoryRow = preferenceRowFor(/Review requests/);
    expect(categoryRow).toHaveAttribute("data-rcl-settings-row");
    expect(screen.getByTestId("notification-source-git-control-tower")).toHaveAttribute("data-rcl-settings-list");
    const toggle = within(categoryRow).getByRole("switch", { name: "Enable Review requests" });
    await userEvent.click(toggle);

    await waitFor(() => expect(updatePref).toHaveBeenCalledTimes(1));
    const [sent] = updatePref.mock.calls[0] as [NotificationPreference];
    expect(sent.source_scenario).toBe("git-control-tower");
    expect(sent.category).toBe("review");
    // Default was enabled:true, so toggling drives it false.
    expect(sent.enabled).toBe(false);
  });

  it("localizes the shared switch, minimum urgency label, and urgency options", async () => {
    fetchCatalog.mockResolvedValue(catalogWithOneSource);
    await setLocale("ja");

    const view = renderWithProviders(<SettingsPage />);
    expect(await screen.findByRole("switch", { name: "Review requestsを有効にする" })).toBeInTheDocument();
    expect(screen.getByRole("combobox", { name: "Review requests · 最低緊急度" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "重大" })).toBeInTheDocument();

    await setLocale("ar");
    expect(await screen.findByRole("switch", { name: "تفعيل Review requests" })).toBeInTheDocument();
    expect(screen.getByRole("combobox", { name: "Review requests · الحد الأدنى للاستعجال" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "حرجة" })).toBeInTheDocument();
    view.unmount();
  });

  it("PUTs a preference when the min-urgency select is changed", async () => {
    fetchCatalog.mockResolvedValue(catalogWithOneSource);
    updatePref.mockResolvedValue({} as never);

    renderWithProviders(<SettingsPage />);
    await screen.findByText(/Review requests/);

    const categoryRow = preferenceRowFor(/Review requests/);
    const select = within(categoryRow).getByRole("combobox");
    await userEvent.selectOptions(select, "critical");

    await waitFor(() => expect(updatePref).toHaveBeenCalledTimes(1));
    const [urgencySent] = updatePref.mock.calls[0] as [NotificationPreference];
    expect(urgencySent.min_urgency).toBe("critical");
    expect(urgencySent.category).toBe("review");
  });

  it("falls back to the scenario id when a source/category omits a display name and honours a stored preference", async () => {
    fetchCatalog.mockResolvedValue({
      sources: [
        {
          source_scenario: "bare-scenario",
          display_name: "",
          icon: "",
          deep_link_base: "",
          declared: false,
          categories: [
            {
              category: "alerts",
              display_name: "",
              description: "",
              declared: false,
              default_enabled: false,
              default_min_urgency: "",
            },
          ],
        },
      ],
      // A stored preference for the category exercises effectivePref's stored
      // branch; its empty min_urgency exercises the PreferenceRow "|| low" fallback.
      preferences: [
        {
          source_scenario: "bare-scenario",
          category: "alerts",
          enabled: true,
          min_urgency: "",
          channel_override: "",
        },
      ],
    });
    updatePref.mockResolvedValue({} as never);

    renderWithProviders(<SettingsPage />);

    // Heading and the category row both fall back to the raw id.
    expect(await screen.findByRole("heading", { name: "bare-scenario" })).toBeInTheDocument();
    const categoryRow = preferenceRowFor(/^alerts$/);
    // The stored preference's empty min_urgency falls back to "low" in the select.
    expect(within(categoryRow).getByRole("combobox")).toHaveValue("low");
    expect(within(categoryRow).getByRole("switch")).toBeChecked();
  });

  it("renders a single 'All notifications' row for a source with no categories, wired to category ''", async () => {
    fetchCatalog.mockResolvedValue({
      sources: [
        {
          source_scenario: "coarse-scenario",
          display_name: "Coarse Scenario",
          icon: "",
          deep_link_base: "",
          declared: true,
          categories: [],
        },
      ],
      preferences: [],
    });
    updatePref.mockResolvedValue({} as never);

    renderWithProviders(<SettingsPage />);

    expect(await screen.findByRole("heading", { name: "Coarse Scenario" })).toBeInTheDocument();
    // Exactly one coarse row stands in for the whole source.
    const sourceRow = preferenceRowFor(/All notifications/);
    await userEvent.click(within(sourceRow).getByRole("switch"));
    await waitFor(() => expect(updatePref).toHaveBeenCalledTimes(1));
    const [coarseSent] = updatePref.mock.calls[0] as [NotificationPreference];
    expect(coarseSent.source_scenario).toBe("coarse-scenario");
    expect(coarseSent.category).toBe("");
  });

  it("shows the preference-update failure message when a PUT rejects", async () => {
    fetchCatalog.mockResolvedValue(catalogWithOneSource);
    updatePref.mockRejectedValue(new Error("put failed"));

    renderWithProviders(<SettingsPage />);
    await screen.findByText(/Review requests/);

    const categoryRow = preferenceRowFor(/Review requests/);
    await userEvent.click(within(categoryRow).getByRole("switch"));

    expect(
      await screen.findByText(/Notification preference update failed\./),
    ).toBeInTheDocument();
  });

  it("shows a shaped skeleton while the catalog is loading", async () => {
    // A pending promise keeps the query in its loading state.
    fetchCatalog.mockReturnValue(new Promise(() => {}) as never);

    renderWithProviders(<SettingsPage />);

    expect(await screen.findByTestId("notifications-loading")).toBeInTheDocument();
    // The skeleton is shaped, not a bare "Loading…" label.
    expect(screen.queryByText(/No scenarios/)).not.toBeInTheDocument();
  });

  it("renders a composed empty-state with a refresh button that refetches", async () => {
    fetchCatalog.mockResolvedValue({ sources: [], preferences: [] });

    renderWithProviders(<SettingsPage />);

    const empty = await screen.findByTestId("notifications-empty");
    expect(empty).toHaveTextContent("No notifications to manage yet");
    // Plain language, no "scenarios/categories" jargon.
    expect(within(empty).queryByText(/scenario/i)).not.toBeInTheDocument();

    await userEvent.click(await screen.findByTestId("notifications-refresh"));
    await waitFor(() => expect(fetchCatalog).toHaveBeenCalledTimes(2));
  });

  it("renders a composed error-state with a retry button that refetches", async () => {
    fetchCatalog.mockRejectedValue(new Error("boom"));

    renderWithProviders(<SettingsPage />);

    const error = await screen.findByTestId("notifications-error");
    expect(within(error).getByText(/Can't reach notifications right now/)).toBeInTheDocument();

    fetchCatalog.mockResolvedValue({ sources: [], preferences: [] });
    await userEvent.click(screen.getByTestId("notifications-retry"));
    // Retry refetches and resolves into the empty state.
    expect(await screen.findByTestId("notifications-empty")).toBeInTheDocument();
  });
});
