import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CalendarMonth } from "./CalendarMonth";
import { renderWithProviders } from "../test-utils";

const api = vi.hoisted(() => ({
  fetchCalendarEvents: vi.fn(),
  fetchCalendarEvent: vi.fn(),
  createCalendarEvent: vi.fn(),
  updateCalendarEvent: vi.fn(),
  lookupCalendarEventByIdempotencyKey: vi.fn(),
}));
vi.mock("../api/calendar", () => api);

function renderMonth() {
  return renderWithProviders(<CalendarMonth />);
}

afterEach(() => { vi.clearAllMocks(); vi.unstubAllGlobals(); });

describe("CalendarMonth", () => {
  it("shows loading and error states while keeping navigation available", async () => {
    api.fetchCalendarEvents.mockRejectedValue(new Error("offline"));
    renderMonth();
    await screen.findByRole("alert");
    expect(screen.getByText(/Events for this month are unavailable/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Previous month" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Today" })).toBeEnabled();
  });

  it("keeps an empty month grid visible and navigates with keyboard controls", async () => {
    api.fetchCalendarEvents.mockResolvedValue([]);
    const user = userEvent.setup();
    renderMonth();
    await screen.findByLabelText(/dates$/);
    const grid = screen.getByLabelText(/dates$/);
    expect(within(grid).getAllByRole("region").length).toBeGreaterThanOrEqual(28);
    expect(screen.getByText("No events this month. The date grid stays open so you can add one.")).toBeInTheDocument();

    screen.getByRole("button", { name: "Next month" }).focus();
    await user.keyboard("{Enter}");
    await waitFor(() => expect(api.fetchCalendarEvents).toHaveBeenCalledTimes(2));
    expect(api.fetchCalendarEvents.mock.calls[0]).not.toEqual(api.fetchCalendarEvents.mock.calls[1]);
    expect(screen.getByLabelText(/dates$/)).toBeInTheDocument();
  });

  it("supports canceling a populated all-day draft after editing every event detail", async () => {
    api.fetchCalendarEvents.mockResolvedValue([]);
    const user = userEvent.setup();
    renderMonth();
    await user.click(screen.getByRole("button", { name: "Previous month" }));
    await user.click(screen.getByRole("button", { name: "Today" }));
    fireEvent.click((await screen.findAllByRole("button", { name: /Add event on/ }))[0]!);
    await user.type(screen.getByRole("textbox", { name: "Event title" }), "Draft details");
    await user.type(screen.getByRole("textbox", { name: "Context subject" }), "family awareness");
    fireEvent.change(screen.getAllByLabelText("Event availability")[1]!, { target: { value: "free" } });
    await user.clear(screen.getByLabelText("Timezone"));
    await user.type(screen.getByLabelText("Timezone"), "America/New_York");
    fireEvent.change(screen.getByLabelText("Event start date"), { target: { value: "2026-10-04" } });
    fireEvent.change(screen.getByLabelText("Exclusive event end date"), { target: { value: "2026-10-07" } });
    await user.type(screen.getByLabelText("Event notes"), "Private synthetic context");
    await user.click(screen.getByTestId("calendar-event-all-day"));
    expect(screen.getByLabelText("Event start instant")).toBeInTheDocument();
    expect(screen.queryByLabelText("Event start date")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("calendar-event-all-day"));
    expect(screen.getByLabelText("Event start date")).toHaveValue("");
    await user.click(screen.getByRole("button", { name: "Close" }));
    expect(screen.queryByTestId("calendar-event-editor")).not.toBeInTheDocument();
  });

  it("creates a date-only event with an exclusive end and reports its durable identity", async () => {
    const event = { id: "synthetic-event-1", title: "Synthetic trip", subject: "owner", notes: "", availability: "busy", timezone: "America/New_York", all_day: true, start_date: "2026-10-04", end_date_exclusive: "2026-10-07", start_at: "", end_at: "", revision: 1 };
    api.fetchCalendarEvents.mockResolvedValue([]);
    api.createCalendarEvent.mockResolvedValue(event);
    const user = userEvent.setup();
    renderMonth();
    const add = await screen.findByRole("button", { name: /Add event on .*October 4/ });
    fireEvent.click(add);
    await user.type(screen.getByRole("textbox", { name: "Event title" }), "Synthetic trip");
    await user.type(screen.getByRole("textbox", { name: "Context subject" }), "owner");
    fireEvent.change(screen.getByLabelText("Event start date"), { target: { value: "2026-10-04" } });
    fireEvent.change(screen.getByLabelText("Exclusive event end date"), { target: { value: "2026-10-07" } });
    await user.click(screen.getByRole("button", { name: "Create event" }));
    await screen.findByText(/Saved Synthetic trip as event synthetic-event-1, revision 1/);
    expect(api.createCalendarEvent).toHaveBeenCalledWith(expect.objectContaining({ start_date: "2026-10-04", end_date_exclusive: "2026-10-07", all_day: true }), expect.any(String));
  });

  it("creates with a collision-resistant fallback key when randomUUID is unavailable", async () => {
    vi.stubGlobal("crypto", {});
    api.fetchCalendarEvents.mockResolvedValue([]);
    api.createCalendarEvent.mockResolvedValue({ id: "fallback-key-event", title: "Fallback", subject: "owner", notes: "", availability: "busy", timezone: "America/New_York", all_day: true, start_date: "2026-10-04", end_date_exclusive: "2026-10-05", start_at: "", end_at: "", revision: 1 });
    const user = userEvent.setup();
    renderMonth();
    await user.click((await screen.findAllByRole("button", { name: /Add event on/ }))[0]!);
    await user.type(screen.getByRole("textbox", { name: "Event title" }), "Fallback");
    await user.click(screen.getByRole("button", { name: "Create event" }));
    await screen.findByText(/Saved Fallback as event fallback-key-event/);
    expect(api.createCalendarEvent.mock.calls[0]?.[1]).toMatch(/^calendar-\d+-[0-9a-f]+$/);
  });

  it("renders a New York DST-fold timed event on its civil date and edits by revision", async () => {
    const event = { id: "timed-fold-1", title: "DST fold", subject: "owner", notes: "Keep both offsets", availability: "free", timezone: "America/New_York", all_day: false, start_date: "", end_date_exclusive: "", start_at: "2026-11-01T01:30:00-04:00", end_at: "2026-11-01T01:30:00-05:00", revision: 1 };
    api.fetchCalendarEvents.mockResolvedValue([event]);
    api.fetchCalendarEvent.mockResolvedValue(event);
    api.updateCalendarEvent.mockResolvedValue({ ...event, title: "DST fold revised", revision: 2 });
    const user = userEvent.setup();
    renderMonth();
    await user.click(screen.getByRole("button", { name: "Next month" }));
    const rendered = await screen.findByRole("button", { name: /Edit DST fold, event timed-fold-1/ });
    expect(rendered).toHaveAttribute("data-testid", "calendar-timed-event");
    expect(rendered).toHaveAttribute("data-event-title", "DST fold");
    await user.click(rendered);
    expect(await screen.findByLabelText("Event start instant")).toHaveValue("2026-11-01T01:30:00-04:00");
    expect(screen.getByLabelText("Event end instant")).toHaveValue("2026-11-01T01:30:00-05:00");
    await user.clear(screen.getByRole("textbox", { name: "Event title" }));
    await user.type(screen.getByRole("textbox", { name: "Event title" }), "DST fold revised");
    await user.click(screen.getByRole("button", { name: "Save event" }));
    await screen.findByText("Updated DST fold revised to revision 2.");
    expect(api.updateCalendarEvent).toHaveBeenCalledWith("timed-fold-1", expect.objectContaining({ all_day: false, start_at: event.start_at, end_at: event.end_at }), 1);
  });

  it("keeps an edit draft visible when the expected revision conflicts", async () => {
    const event = { id: "event-conflict", title: "Original", subject: "owner", notes: "", availability: "busy", timezone: "America/New_York", all_day: true, start_date: "2026-10-04", end_date_exclusive: "2026-10-05", start_at: "", end_at: "", revision: 3 };
    api.fetchCalendarEvents.mockResolvedValue([event]);
    api.fetchCalendarEvent.mockResolvedValue(event);
    api.updateCalendarEvent.mockRejectedValue(new Error("revision conflict"));
    const user = userEvent.setup();
    renderMonth();
    await user.click(await screen.findByRole("button", { name: /Edit Original, event event-conflict/ }));
    const title = await screen.findByRole("textbox", { name: "Event title" });
    await user.clear(title);
    await user.type(title, "My draft");
    await user.click(screen.getByRole("button", { name: "Save event" }));
    expect(await screen.findByRole("status")).toHaveTextContent(/changed elsewhere/);
    expect(title).toHaveValue("My draft");
  });

  it("checks an uncertain create by idempotency key before allowing retry", async () => {
    api.fetchCalendarEvents.mockResolvedValue([]);
    api.createCalendarEvent.mockRejectedValue(new Error("network lost"));
    api.lookupCalendarEventByIdempotencyKey.mockResolvedValue(null);
    const user = userEvent.setup();
    renderMonth();
    fireEvent.click((await screen.findAllByRole("button", { name: /Add event on/ }))[0]!);
    await user.type(screen.getByRole("textbox", { name: "Event title" }), "Synthetic retry");
    await user.click(screen.getByRole("button", { name: "Create event" }));
    await screen.findByRole("button", { name: "Check save status" });
    const firstKey = api.createCalendarEvent.mock.calls[0]![1];
    await user.click(screen.getByRole("button", { name: "Check save status" }));
    await screen.findByText(/No event exists for this key yet/);
    await user.click(screen.getByRole("button", { name: "Retry same event" }));
    await waitFor(() => expect(api.createCalendarEvent).toHaveBeenCalledTimes(2));
    expect(api.createCalendarEvent.mock.calls[1]![1]).toBe(firstKey);
    expect(api.lookupCalendarEventByIdempotencyKey).toHaveBeenCalledWith(firstKey);
    expect(screen.getByRole("textbox", { name: "Event title" })).toBeDisabled();
  });

  it("recovers a create that succeeded before its response was lost", async () => {
    const saved = { id: "saved-after-timeout", title: "Recovered", subject: "family awareness", notes: "", availability: "free", timezone: "America/New_York", all_day: true, start_date: "2026-10-04", end_date_exclusive: "2026-10-05", start_at: "", end_at: "", revision: 1 };
    api.fetchCalendarEvents.mockResolvedValue([]);
    api.createCalendarEvent.mockRejectedValue(new Error("response lost"));
    api.lookupCalendarEventByIdempotencyKey.mockResolvedValue(saved);
    const user = userEvent.setup();
    renderMonth();
    await user.click((await screen.findAllByRole("button", { name: /Add event on/ }))[0]!);
    await user.type(screen.getByRole("textbox", { name: "Event title" }), "Recovered");
    await user.click(screen.getByRole("button", { name: "Create event" }));
    await user.click(await screen.findByRole("button", { name: "Check save status" }));
    await screen.findByText(/The earlier save succeeded: Recovered, event saved-after-timeout, revision 1/);
    expect(api.createCalendarEvent).toHaveBeenCalledTimes(1);
  });

  it("keeps an uncertain draft unchanged when status lookup is unavailable", async () => {
    api.fetchCalendarEvents.mockResolvedValue([]);
    api.createCalendarEvent.mockRejectedValue(new Error("response lost"));
    api.lookupCalendarEventByIdempotencyKey.mockRejectedValue(new Error("offline"));
    const user = userEvent.setup();
    renderMonth();
    await user.click((await screen.findAllByRole("button", { name: /Add event on/ }))[0]!);
    const title = screen.getByRole("textbox", { name: "Event title" });
    await user.type(title, "Keep me");
    await user.click(screen.getByRole("button", { name: "Create event" }));
    await user.click(await screen.findByRole("button", { name: "Check save status" }));
    expect(await screen.findByText(/Event status is still unavailable/)).toBeInTheDocument();
    expect(title).toHaveValue("Keep me");
    expect(title).toBeDisabled();
  });
});
