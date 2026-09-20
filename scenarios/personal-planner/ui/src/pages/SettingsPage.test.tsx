import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";
import { setLocale } from "../i18n";
import { fetchPlanningProfile, replaceAvailability, updatePlanningProfile } from "../api/workspace";
import { SettingsPage } from "./SettingsPage";


vi.mock("../api/workspace", () => ({
  fetchPlanningProfile: vi.fn().mockResolvedValue({ id: "default", timezone: "UTC", weekStart: "monday", dailyCapacityMinutes: 480, reserveMinutes: 60, focusSessionMinutes: 45, revision: 1 }),
  updatePlanningProfile: vi.fn().mockImplementation(async (profile, input) => ({ ...profile, ...input, revision: profile.revision + 1 })),
  fetchAvailability: vi.fn().mockResolvedValue({ windows: [], exceptions: [], revision: 1n }),
  replaceAvailability: vi.fn().mockImplementation(async (input) => ({ windows: input.windows, exceptions: input.exceptions, revision: input.expectedRevision + 1n })),
}));

vi.mock("../api/integrations", () => ({
  fetchConnections: vi.fn().mockResolvedValue([]),
  createFixtureConnection: vi.fn().mockResolvedValue({ id: "fixture-1", provider: "fixture", displayName: "Observatory sample calendar", sourceKind: "fixture", status: "connected", healthMessage: "Synthetic read-only adapter", readOnly: true, calendarCount: 0, importedEventCount: 0, busyMinutes: 0, revision: 1n, lastSyncAt: undefined }),
  syncConnection: vi.fn().mockResolvedValue({ id: "fixture-1", provider: "fixture", displayName: "Observatory sample calendar", sourceKind: "fixture", status: "synced", healthMessage: "Synthetic read-only adapter", readOnly: true, calendarCount: 1, importedEventCount: 3, busyMinutes: 120, revision: 2n, lastSyncAt: undefined }),
  disconnectConnection: vi.fn().mockResolvedValue({ id: "fixture-1", provider: "fixture", displayName: "Observatory sample calendar", sourceKind: "fixture", status: "disconnected", healthMessage: "Synthetic read-only adapter", readOnly: true, calendarCount: 0, importedEventCount: 0, busyMinutes: 0, revision: 2n, lastSyncAt: undefined }),
}));

afterEach(() => {
  cleanup();
  window.localStorage.clear();
});

describe("SettingsPage", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  it("states the included core and deferred commercial hypothesis honestly", () => {
    renderWithProviders(<SettingsPage />);

    expect(screen.getByText("Shape the Observatory around your day—appearance, capacity, availability, and connections.")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "A calm core, kept honest" })).toBeInTheDocument();
    expect(screen.getByText(/Nothing essential is hidden behind an upgrade/)).toBeInTheDocument();
    expect(screen.getByText(/No billing is active/)).toBeInTheDocument();
  });

  it("keeps dense preference and availability controls on the shared field contract", async () => {
    renderWithProviders(<SettingsPage />);

    expect(await screen.findByLabelText("Timezone")).toBeInTheDocument();
    expect(screen.getAllByTestId("forms.form-field").length).toBeGreaterThan(14);
    expect(screen.getByRole("radio", { name: "Day" })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: "Night" })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: "Auto" })).toBeInTheDocument();
    expect(screen.getByLabelText("1 start")).toHaveAttribute("data-rcl-form-field-control");
    expect(screen.getByLabelText("1 end")).toHaveAttribute("data-rcl-form-field-control");
  });

  it("loads and saves the explicit planning profile", async () => {
    const user = userEvent.setup();
    renderWithProviders(<SettingsPage />);

    const timezone = await screen.findByLabelText("Timezone");
    await waitFor(() => expect(timezone).toHaveValue("UTC"));
    await user.click(screen.getByLabelText("Art-free mode"));
    await user.clear(timezone);
    await user.type(timezone, "America/New_York");
    await user.click(screen.getByRole("button", { name: "Week starts" }));
    await user.click(await screen.findByRole("option", { name: "Sunday" }));
    await user.clear(screen.getByLabelText("Daily capacity (minutes)"));
    await user.type(screen.getByLabelText("Daily capacity (minutes)"), "420");
    await user.clear(screen.getByLabelText("Protected reserve (minutes)"));
    await user.type(screen.getByLabelText("Protected reserve (minutes)"), "60");
    await user.clear(screen.getByLabelText("Preferred focus session (minutes)"));
    await user.type(screen.getByLabelText("Preferred focus session (minutes)"), "50");
    await user.click(screen.getByRole("button", { name: "Save planning profile" }));

    await waitFor(() => expect(updatePlanningProfile).toHaveBeenCalledWith(expect.objectContaining({ timezone: "UTC", revision: 1 }), expect.objectContaining({ timezone: "America/New_York", weekStart: "sunday", dailyCapacityMinutes: 420, reserveMinutes: 60, focusSessionMinutes: 50 })));
    expect(screen.getByRole("status", { name: "" })).toHaveTextContent("Planning profile saved.");
    expect(fetchPlanningProfile).toHaveBeenCalled();

    const mondayStart = await screen.findByLabelText("1 start");
    await user.clear(mondayStart);
    await user.type(mondayStart, "09:00");
    await user.click(screen.getByRole("button", { name: "Save availability" }));
    await waitFor(() => expect(replaceAvailability).toHaveBeenCalledWith(expect.objectContaining({ expectedRevision: 1n, windows: expect.arrayContaining([expect.objectContaining({ weekday: 1 })]) })));
  });

  it("keeps provider setup honest while exposing the synthetic adapter boundary", async () => {
    const user = userEvent.setup();
    renderWithProviders(<SettingsPage />);

    expect(await screen.findByText("No calendar is connected. Manual planning stays complete.")).toBeInTheDocument();
    expect(screen.getByText(/Live provider OAuth and credential storage are not configured/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Add sample calendar" }));
    await waitFor(() => expect(screen.getByText("Sample calendar added.")).toBeInTheDocument());
  });
});
