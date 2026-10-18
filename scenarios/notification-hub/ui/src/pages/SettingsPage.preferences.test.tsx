import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";
import { SettingsPage } from "./SettingsPage";
import { fetchNotificationCatalog, updateNotificationPreference, type NotificationPreference } from "../api/preferences";
import { identityClient, recipientsClient } from "../api/notifications";

/** The PreferenceRow container holding the label, enable checkbox, and urgency select. */
function preferenceRowFor(label: RegExp): HTMLElement {
  // span (label) -> div.min-w-0.flex-col -> div (the row)
  const span = screen.getByText(label);
  return span.parentElement!.parentElement as HTMLElement;
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
  beforeEach(() => {
    vi.mocked(identityClient.getSession).mockResolvedValue({
      signedIn: true,
      subject: "owner-1",
      email: "owner@example.test",
    } as never);
    vi.mocked(recipientsClient.listDevices).mockResolvedValue({ devices: [] } as never);
  });

  afterEach(() => {
    vi.clearAllMocks();
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
    expect(screen.getAllByRole("option", { name: "critical" }).length).toBeGreaterThanOrEqual(1);
  });

  it("shows recipient preferences and device metadata while signed out but disables writes", async () => {
    vi.mocked(identityClient.getSession).mockResolvedValue({ signedIn: false } as never);
    vi.mocked(recipientsClient.listDevices).mockResolvedValue({
      devices: [{ id: "device-1", name: "Phone", machineId: "local", channels: ["web_push"] }],
      vapidPublicKey: "public-key",
    } as never);
    fetchCatalog.mockResolvedValue(catalogWithOneSource);

    renderWithProviders(<SettingsPage />);

    expect(await screen.findByRole("heading", { name: "Git Control Tower" })).toBeInTheDocument();
    expect(screen.getAllByText("Phone").length).toBeGreaterThan(0);
    const row = preferenceRowFor(/Review requests/);
    expect(within(row).getByRole("checkbox")).toBeDisabled();
    expect(within(row).getByRole("combobox")).toBeDisabled();
    expect(updatePref).not.toHaveBeenCalled();
    expect(await screen.findByText(/Sign in to change notification preferences/)).toBeInTheDocument();
  });

  it("PUTs a preference when the enable checkbox is toggled", async () => {
    fetchCatalog.mockResolvedValue(catalogWithOneSource);
    updatePref.mockResolvedValue({} as never);

    renderWithProviders(<SettingsPage />);
    await screen.findByText(/Review requests/);

    const categoryRow = preferenceRowFor(/Review requests/);
    const checkbox = within(categoryRow).getByRole("checkbox");
    await userEvent.click(checkbox);

    await waitFor(() => expect(updatePref).toHaveBeenCalledTimes(1));
    const [sent] = updatePref.mock.calls[0] as [NotificationPreference];
    expect(sent.source_scenario).toBe("git-control-tower");
    expect(sent.category).toBe("review");
    // Default was enabled:true, so toggling drives it false.
    expect(sent.enabled).toBe(false);
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
    expect(within(categoryRow).getByRole("checkbox")).toBeChecked();
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
    await userEvent.click(within(sourceRow).getByRole("checkbox"));
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
    await userEvent.click(within(categoryRow).getByRole("checkbox"));

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
    expect(within(empty).getByText(/No notifications to manage yet/)).toBeInTheDocument();
    // Plain language, no "scenarios/categories" jargon.
    expect(within(empty).queryByText(/scenario/i)).not.toBeInTheDocument();

    await userEvent.click(screen.getByTestId("notifications-refresh"));
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
