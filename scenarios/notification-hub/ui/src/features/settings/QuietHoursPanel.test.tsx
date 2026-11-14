import { afterEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test-utils";
import { recipientsClient } from "../../api/notifications";
import { QuietHoursPanel } from "./QuietHoursPanel";

vi.mock("../../api/notifications", () => ({ recipientsClient: {
  listQuietWindows: vi.fn().mockResolvedValue({ windows: [] }),
  setQuietWindow: vi.fn().mockResolvedValue({ id: "saved" }),
  deleteQuietWindow: vi.fn().mockResolvedValue({}),
} }));

describe("QuietHoursPanel", () => {
  afterEach(() => window.history.replaceState({}, "", "/"));
  it("presents the week Monday first while keeping Sunday on weekday id zero", async () => {
    renderWithProviders(<QuietHoursPanel />);
    await screen.findByTestId("quiet-hours-edit-day-1");
    expect(screen.getByTestId("quiet-hours-edit-day-1")).toHaveClass("min-h-14", "sm:min-h-16");
    const weekdayIds = Array.from(screen.getByRole("list", { name: "Weekly windows" }).querySelectorAll("li[data-testid^='quiet-hours-day-']"), (row) => row.getAttribute("data-testid"));
    expect(weekdayIds).toEqual(["quiet-hours-day-1", "quiet-hours-day-2", "quiet-hours-day-3", "quiet-hours-day-4", "quiet-hours-day-5", "quiet-hours-day-6", "quiet-hours-day-0"]);
  });
  it("offers the weekly view as a focused destination with a Settings return and stable disabled review action", async () => {
    const user = userEvent.setup();
    const onFocusedChange = vi.fn();
    const onEditorHeaderChange = vi.fn();
    const view = renderWithProviders(<QuietHoursPanel onFocusedChange={onFocusedChange} onEditorHeaderChange={onEditorHeaderChange} backToSettingsLabel="Settings" />);
    await screen.findByTestId("quiet-hours-edit-day-1");
    await user.click(screen.getByRole("button", { name: "Edit Weekly windows" }));
    await waitFor(() => expect(onFocusedChange).toHaveBeenCalledWith(true));
    view.rerender(<QuietHoursPanel focused onFocusedChange={onFocusedChange} onEditorHeaderChange={onEditorHeaderChange} backToSettingsLabel="Settings" />);
    const weeklyHeader = onEditorHeaderChange.mock.calls.map(([header]) => header).reverse().find((header) => header?.title === "Weekly windows");
    expect(weeklyHeader).toMatchObject({ title: "Weekly windows", backLabel: "Settings" });
    expect(screen.getByTestId("quiet-hours-review-button")).toBeDisabled();
    expect(screen.queryByText("Set one same-day window per weekday. Each day saves separately; check the latest read after a save.")).not.toBeInTheDocument();
    weeklyHeader.onBack();
    expect(onFocusedChange).toHaveBeenLastCalledWith(false);
  });
  it("uses the selected guidance rail color when weekly windows use mixed time zones", async () => {
    vi.mocked(recipientsClient.listQuietWindows).mockResolvedValueOnce({ windows: [
      { id: "monday", weekday: 1, start: "13:00", end: "14:00", timezone: "Europe/London", criticalOverride: false },
      { id: "tuesday", weekday: 2, start: "13:00", end: "14:00", timezone: "Etc/UTC", criticalOverride: false },
    ] } as never);
    renderWithProviders(<QuietHoursPanel focused onEditorHeaderChange={vi.fn()} />);
    await screen.findByTestId("quiet-hours-edit-day-1");
    await screen.findByRole("heading", { name: "Different time zones" });
    expect(screen.getByText("Each window uses its own saved zone.").closest("aside")).toHaveStyle({ borderInlineStartColor: "var(--reach-guidance-rail)" });
  });
  const openDay = async (user: ReturnType<typeof userEvent.setup>, day: number) => {
    if (screen.queryByTestId("quiet-hours-check-saved")) await user.click(screen.getAllByRole("button", { name: /Return to week|Keep draft/ }).at(-1)!);
    await user.click(screen.getByTestId(`quiet-hours-edit-day-${day}`));
    return screen.getByTestId(`quiet-hours-editor-${day}`);
  };
  const openReview = async (user: ReturnType<typeof userEvent.setup>) => {
    await user.click(screen.getByRole("button", { name: "Review changed days" }));
    return screen.getByTestId("quiet-hours-review");
  };
  const enableDay = async (user: ReturnType<typeof userEvent.setup>, group: HTMLElement) => {
    const checkbox = within(group).getByRole("checkbox", { name: "Set a quiet window" });
    if (!(checkbox as HTMLInputElement).checked) await user.click(checkbox);
  };

  it("blocks equal and overnight times without a write, and reports overnight as held", async () => {
    const user = userEvent.setup();
    renderWithProviders(<QuietHoursPanel />);
    await screen.findByTestId("quiet-hours-edit-day-1");
    const group = await openDay(user, 1);
    await enableDay(user, group);
    const start = within(group).getByTestId("quiet-hours-start-1");
    const end = within(group).getByTestId("quiet-hours-end-1");
    await user.clear(start); await user.type(start, "22:00");
    await user.clear(end); await user.type(end, "22:00");
    expect(screen.getByRole("heading", { name: "Choose different times" })).toBeInTheDocument();
    expect(screen.getByTestId("quiet-hours-day-meta-1")).toHaveTextContent("Monday · unsaved draft");
    const equalTimeAlert = within(group).getByRole("alert");
    expect(equalTimeAlert).toHaveTextContent("Start and end are the same");
    expect(equalTimeAlert).toHaveTextContent("Equal times do not mean an all-day quiet window");
    const startInput = within(group).getByTestId("quiet-hours-start-1");
    expect(startInput).toHaveAttribute("type", "text");
    expect(within(group).getByTestId("quiet-hours-end-1")).toHaveAttribute("type", "text");
    expect(startInput).toHaveAttribute("aria-invalid", "true");
    expect(within(group).getByTestId("quiet-hours-end-1")).toHaveAttribute("aria-invalid", "true");
    expect(group).toHaveTextContent("Applies to Monday only.");
    expect(within(group).queryByTestId("quiet-hours-done-stages-1")).not.toBeInTheDocument();
    expect(equalTimeAlert.compareDocumentPosition(startInput) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(within(group).getByText("Choose different start and end times.")).toBeInTheDocument();
    expect(within(group).getByRole("button", { name: "Done" })).toBeDisabled();
    expect(within(group).getByRole("button", { name: "Keep editing" })).toBeEnabled();
    const overnight = group;
    const overnightEnd = within(overnight).getByTestId("quiet-hours-end-1");
    await user.clear(within(overnight).getByTestId("quiet-hours-start-1")); await user.type(within(overnight).getByTestId("quiet-hours-start-1"), "22:00");
    await user.clear(overnightEnd); await user.type(overnightEnd, "08:00");
    expect(screen.getByRole("heading", { name: "Overnight window isn’t ready" })).toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent("An updated scheduling contract is needed");
    expect(screen.getByRole("alert")).toHaveTextContent("choose a same-day window");
    expect(within(overnight).getByTestId("quiet-hours-timezone-1").compareDocumentPosition(screen.getByRole("alert")) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(within(overnight).getByLabelText("Monday starts")).toHaveValue("22:00");
    expect(within(overnight).getByLabelText("Tuesday ends")).toHaveValue("08:00");
    expect(within(overnight).getByRole("button", { name: "Save overnight window" })).toBeDisabled();
    await user.clear(overnightEnd); await user.type(overnightEnd, "23:00");
    const zone = within(overnight).getByLabelText("Time zone"); await user.clear(zone); await user.type(zone, "Not/A_Real_Zone");
    expect(screen.getByRole("heading", { name: "Check Monday’s time zone" })).toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent("The time zone isn’t recognized");
    expect(screen.getByRole("alert")).toHaveTextContent("IANA time-zone name");
    expect(within(overnight).getByText("Enter a recognized IANA time zone.")).toBeInTheDocument();
    expect(within(overnight).getByTestId("quiet-hours-timezone-1")).toHaveAttribute("aria-invalid", "true");
    expect(screen.getByRole("alert").compareDocumentPosition(zone) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(within(overnight).getByRole("button", { name: "Done" })).toBeDisabled();
    await user.clear(zone); await user.type(zone, "UTC");
    await user.clear(within(overnight).getByTestId("quiet-hours-start-1")); await user.type(within(overnight).getByTestId("quiet-hours-start-1"), "25:30");
    expect(screen.getByRole("heading", { name: "Check Monday’s time" })).toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent("The start time isn’t valid");
    expect(within(overnight).getByText("Enter a time from 00:00 to 23:59.")).toBeInTheDocument();
    expect(within(overnight).getByTestId("quiet-hours-start-1")).toHaveAttribute("aria-invalid", "true");
    expect(overnight).toHaveTextContent("Applies to Monday only.");
    expect(screen.getByRole("alert").compareDocumentPosition(within(overnight).getByTestId("quiet-hours-start-1")) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(within(overnight).getByRole("button", { name: "Done" })).toBeDisabled();
    await user.clear(within(overnight).getByTestId("quiet-hours-start-1")); await user.type(within(overnight).getByTestId("quiet-hours-start-1"), "22:00");
    expect(within(overnight).getByRole("button", { name: "Done" })).toBeEnabled();
    expect(vi.mocked(recipientsClient.setQuietWindow)).not.toHaveBeenCalled();
  });

  it("keeps opaque storage IDs out of the saved-window label", async () => {
    const user = userEvent.setup();
    vi.mocked(recipientsClient.listQuietWindows).mockReset().mockResolvedValue({ windows: [
      { id: "1bd5cb6e-6640-46a4-85d8-d9b700a0ea25", weekday: 1, start: "13:00", end: "14:00", timezone: "Etc/UTC", criticalOverride: false },
    ] } as never);
    renderWithProviders(<QuietHoursPanel />);
    await openDay(user, 1);
    expect(screen.getByTestId("quiet-hours-day-meta-1")).toHaveTextContent("Monday · saved window");
    expect(screen.queryByText(/1bd5cb6e/)).not.toBeInTheDocument();
  });

  it("re-reads after a partial multi-day save and preserves the server-confirmed day", async () => {
    const user = userEvent.setup();
    const onEditorHeaderChange = vi.fn();
    const list = vi.mocked(recipientsClient.listQuietWindows);
    const mondaySaved = { id: "mon", weekday: 1, start: "21:00", end: "23:00", timezone: "UTC", criticalOverride: false };
    list.mockReset().mockResolvedValueOnce({ windows: [] } as never)
      .mockResolvedValueOnce({ windows: [mondaySaved] } as never)
      .mockResolvedValueOnce({ windows: [mondaySaved] } as never);
    vi.mocked(recipientsClient.setQuietWindow).mockReset().mockResolvedValueOnce({ id: "mon" } as never).mockRejectedValueOnce(new Error("write uncertain"));
    renderWithProviders(<QuietHoursPanel onEditorHeaderChange={onEditorHeaderChange} />);
    for (const day of [1, 2]) {
      const group = await openDay(user, day);
      await enableDay(user, group);
      const start = within(group).getByTestId(`quiet-hours-start-${day}`); await user.clear(start); await user.type(start, "21:00");
      const end = within(group).getByTestId(`quiet-hours-end-${day}`); await user.clear(end); await user.type(end, "23:00");
      const zone = within(group).getByLabelText("Time zone"); await user.clear(zone); await user.type(zone, "UTC");
      await user.click(within(group).getByRole("button", { name: "Done" }));
    }
    await openReview(user);
    await user.click(screen.getByRole("button", { name: "Save quiet hours" }));
    expect(await screen.findByRole("status")).toHaveTextContent("Read the saved windows before sending another change.");
    expect(screen.getByRole("heading", { name: "Monday–Tuesday" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Some quiet hours need checking", level: 2 })).toBeInTheDocument();
    const resultHeader = onEditorHeaderChange.mock.calls.map(([header]) => header).reverse().find((header) => header?.title === "Quiet hours");
    expect(resultHeader).toMatchObject({ title: "Quiet hours", backLabel: "Week" });
    expect(screen.getByRole("button", { name: "Keep draft" })).toBeVisible();
    expect(resultHeader?.onBack).toBeTypeOf("function");
    await waitFor(() => expect(list).toHaveBeenCalledTimes(3));
    const monday = await openDay(user, 1);
    expect(within(monday).getByTestId("quiet-hours-start-1")).toHaveValue("21:00");
    await user.click(within(monday).getByRole("button", { name: "Done" }));
    const tuesday = await openDay(user, 2);
    expect(within(tuesday).getByTestId("quiet-hours-start-2")).toHaveValue("21:00");
  });

  it("retains grouped multi-day status when a follow-up saved-window read is still unknown", async () => {
    const user = userEvent.setup();
    const list = vi.mocked(recipientsClient.listQuietWindows);
    const before = [1, 2, 3, 4, 5].map((weekday) => ({ id: "day-" + weekday, weekday, start: "12:00", end: "13:00", timezone: "Etc/UTC", criticalOverride: false }));
    const afterMonday = before.map((row) => row.weekday === 1 ? { ...row, start: "13:00", end: "14:00" } : row);
    const afterTuesday = afterMonday.map((row) => row.weekday === 2 ? { ...row, start: "13:00", end: "14:00" } : row);
    list.mockReset()
      .mockResolvedValueOnce({ windows: before } as never)
      .mockResolvedValueOnce({ windows: afterMonday } as never)
      .mockResolvedValueOnce({ windows: afterTuesday } as never)
      .mockRejectedValueOnce(new Error("Wednesday confirming read unavailable"))
      .mockRejectedValueOnce(new Error("follow-up read unavailable"));
    vi.mocked(recipientsClient.setQuietWindow).mockReset().mockResolvedValue({ id: "saved" } as never);

    renderWithProviders(<QuietHoursPanel />);
    await screen.findByTestId("quiet-hours-edit-day-1");
    for (const day of [1, 2, 3, 4, 5]) {
      const editor = within(await openDay(user, day));
      const start = editor.getByLabelText("Start");
      const end = editor.getByLabelText("End");
      await user.clear(start); await user.type(start, "13:00");
      await user.clear(end); await user.type(end, "14:00");
      await user.click(editor.getByRole("button", { name: "Done" }));
    }
    await openReview(user);
    await user.click(screen.getByRole("button", { name: "Save quiet hours" }));
    expect(await screen.findByRole("heading", { name: "Some quiet hours need checking", level: 2 })).toBeInTheDocument();
    expect(screen.queryByTestId("quiet-hours-outcome-groups")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Check saved windows" })).toBeVisible();

    await user.click(screen.getByRole("button", { name: "Check saved windows" }));
    expect(await screen.findByRole("heading", { name: "Wednesday is still unknown", level: 2 })).toBeInTheDocument();
    expect(screen.getByText("Quiet hours · saved-window check")).toBeInTheDocument();
    const groups = screen.getByTestId("quiet-hours-outcome-groups");
    expect(groups).toHaveTextContent("Confirmed");
    expect(groups).toHaveTextContent("Monday, Tuesday");
    expect(groups).toHaveTextContent("Still uncertain");
    expect(groups).toHaveTextContent("Wednesday");
    expect(groups).toHaveTextContent("Not attempted");
    expect(groups).toHaveTextContent("Thursday, Friday");
    const advice = screen.getByText("Keep the draft while you check again. Don’t replace the entire week to resolve one unknown result.");
    expect(advice).toBeVisible();
    expect(advice.closest("details")).toBeNull();
    expect(screen.getByRole("button", { name: "Keep draft" })).toBeVisible();
    expect(recipientsClient.setQuietWindow).toHaveBeenCalledTimes(3);

    await user.click(screen.getByText("Review day-by-day values"));
    expect(screen.getAllByText("Latest saved value: 12:00–13:00 UTC")).toHaveLength(2);
    expect(list).toHaveBeenCalledTimes(5);
  });

  it("reviews a timezone change explicitly and stages it without saving or converting local clock hours", async () => {
    const user = userEvent.setup();
    const onEditorHeaderChange = vi.fn();
    vi.mocked(recipientsClient.listQuietWindows).mockReset().mockResolvedValue({ windows: [
      { id: "e22-quiet-monday", weekday: 1, start: "13:00", end: "14:00", timezone: "Etc/UTC", criticalOverride: false },
    ] } as never);
    vi.mocked(recipientsClient.setQuietWindow).mockReset().mockResolvedValue({ id: "saved" } as never);
    renderWithProviders(<QuietHoursPanel onEditorHeaderChange={onEditorHeaderChange} />);
    await screen.findByTestId("quiet-hours-edit-day-1");
    const monday = within(await openDay(user, 1));
    expect(screen.getByTestId("quiet-hours-day-meta-1")).toHaveTextContent("Monday · quiet-demo-1");
    await user.clear(monday.getByLabelText("Time zone"));
    await user.type(monday.getByLabelText("Time zone"), "Europe/London");
    const preview = monday.getByTestId("quiet-hours-timezone-preview-1");
    expect(preview).toHaveTextContent("Local clock hours stay the same; the absolute time can shift.");
    expect(preview).toHaveTextContent("Before");
    expect(preview).toHaveTextContent("13:00–14:00 · UTC");
    expect(preview).toHaveTextContent("Draft");
    expect(preview).toHaveTextContent("13:00–14:00 · Europe/London");
    expect(monday.getByTestId("quiet-hours-start-1")).toHaveValue("13:00");
    expect(monday.getByTestId("quiet-hours-end-1")).toHaveValue("14:00");
    await user.click(monday.getByRole("button", { name: "Review time-zone change" }));
    const review = screen.getByTestId("quiet-hours-timezone-review");
    expect(review).toHaveTextContent("Monday’s time zone");
    expect(review).toHaveTextContent("New time zone");
    expect(review).toHaveTextContent("Keep local clock hours");
    expect(review).toHaveTextContent("13:00–14:00 stays 13:00–14:00 in the new zone.");
    expect(review).toHaveTextContent("Before");
    expect(review).toHaveTextContent("13:00–14:00 · Etc/UTC");
    expect(review).toHaveTextContent("Draft");
    expect(review).toHaveTextContent("13:00–14:00 · Europe/London");
    expect(review).toHaveTextContent("Only Monday changes");
    expect(review).toHaveTextContent("Other weekdays keep their saved zones.");
    expect(within(review).getByRole("radio", { name: "Keep local clock hours" })).toBeChecked();
    await waitFor(() => expect(onEditorHeaderChange).toHaveBeenLastCalledWith(expect.objectContaining({ title: "Monday", backLabel: "Monday" })));
    const timezone = within(review).getByTestId("quiet-hours-timezone-review-input-1");
    await user.clear(timezone);
    expect(within(review).getByRole("button", { name: "Stage time-zone change" })).toBeDisabled();
    await user.type(timezone, "Europe/London");
    expect(within(review).getByRole("button", { name: "Stage time-zone change" })).toBeEnabled();
    expect(recipientsClient.setQuietWindow).not.toHaveBeenCalled();
    await user.click(within(review).getByRole("button", { name: "Keep editing" }));
    expect(screen.getByTestId("quiet-hours-timezone-1")).toHaveValue("Europe/London");
    await user.click(screen.getByRole("button", { name: "Review time-zone change" }));
    await user.click(screen.getByRole("button", { name: "Stage time-zone change" }));
    expect(screen.getByTestId("quiet-hours-edit-day-1")).toBeInTheDocument();
    expect(screen.getByTestId("quiet-hours-summary-1")).toHaveTextContent("13:00–14:00 · Europe/London");
    expect(recipientsClient.setQuietWindow).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Review changed days" }));
    expect(screen.getByRole("heading", { name: "Review quiet-hour changes" })).toBeInTheDocument();
    expect(recipientsClient.setQuietWindow).not.toHaveBeenCalled();
  });

  it("uses the selected fixture label without exposing the server row ID", async () => {
    window.history.pushState({}, "", "/settings?vrooli_test_mode=1&e22_fixture=settings-owner");
    vi.mocked(recipientsClient.listQuietWindows).mockReset().mockResolvedValue({ windows: [
      { id: "4eeb9787-1d09-4c16-a0e3-df70fdcb9a81", weekday: 1, start: "13:00", end: "14:00", timezone: "Etc/UTC", criticalOverride: false },
    ] } as never);
    renderWithProviders(<QuietHoursPanel />);
    await screen.findByTestId("quiet-hours-edit-day-1");
    const user = userEvent.setup();
    await openDay(user, 1);
    expect(screen.getByTestId("quiet-hours-day-meta-1")).toHaveTextContent("Monday · quiet-demo-1");
    expect(screen.getByTestId("quiet-hours-day-meta-1")).not.toHaveTextContent("4eeb9787");
  });

  it("constrains the selected editor body and separates its scrollable fields from shell-safe actions", async () => {
    const user = userEvent.setup();
    vi.mocked(recipientsClient.listQuietWindows).mockReset().mockResolvedValue({ windows: [
      { id: "quiet-demo-1", weekday: 1, start: "13:00", end: "14:00", timezone: "Etc/UTC", criticalOverride: false },
    ] } as never);
    renderWithProviders(<QuietHoursPanel onEditorHeaderChange={vi.fn()} />);
    await screen.findByTestId("quiet-hours-edit-day-1");
    const editor = await openDay(user, 1);
    expect(editor.firstElementChild).toHaveClass("max-w-[42.375rem]", "min-h-0", "flex-1", "flex-col", "gap-1.5", "sm:gap-5", "overflow-y-auto", "lg:ms-[6.1875rem]");
    expect(within(editor).getByLabelText("Start")).toHaveClass("min-h-[44px]", "bg-[var(--color-field)]");
    expect(within(editor).getByRole("button", { name: "Remove window" })).toBeVisible();
    const actions = within(editor).getByRole("button", { name: "Done" }).parentElement;
    expect(actions).toHaveClass("mt-auto", "shrink-0");
    expect(actions).not.toHaveClass("max-w-2xl", "lg:ms-[6.1875rem]");
    expect(actions).not.toHaveClass("sticky");
  });

  it("keeps independent time zones and makes removal an explicit reviewed save", async () => {
    const user = userEvent.setup();
    vi.mocked(recipientsClient.listQuietWindows).mockReset().mockResolvedValueOnce({ windows: [
      { id: "mon", weekday: 1, start: "21:00", end: "23:00", timezone: "America/New_York", criticalOverride: false },
      { id: "tue", weekday: 2, start: "22:00", end: "23:30", timezone: "Asia/Tokyo", criticalOverride: true },
    ] } as never).mockResolvedValueOnce({ windows: [{ id: "tue", weekday: 2, start: "22:00", end: "23:30", timezone: "Asia/Tokyo", criticalOverride: true }] } as never);
    renderWithProviders(<QuietHoursPanel />);
    const mondayGroup = await openDay(user, 1);
    const monday = within(mondayGroup);
    expect(monday.getByLabelText("Time zone")).toHaveValue("America/New_York");
    await user.click(monday.getByRole("button", { name: "Remove window" }));
    await user.click(monday.getByRole("button", { name: "Done" }));
    const review = await openReview(user);
    expect(screen.getByRole("heading", { name: "Remove Monday’s window?" })).toBeInTheDocument();
    expect(review).toHaveTextContent("Saved window");
    expect(review).toHaveTextContent("21:00–23:00");
    expect(review).toHaveTextContent("Time zone");
    expect(review).toHaveTextContent("Critical urgency");
    expect(review).toHaveTextContent("Stays quiet");
    const reviewScreen = screen.getByTestId("quiet-hours-root");
    expect(reviewScreen.textContent!.indexOf("Saved window")).toBeLessThan(reviewScreen.textContent!.indexOf("Monday will no longer have this quiet window"));
    expect(screen.getByText("Removal is staged first").closest("aside")).toHaveStyle({ borderInlineStartColor: "var(--reach-guidance-rail)" });
    expect(screen.getByRole("button", { name: "Stage removal" })).toHaveAttribute("data-testid", "quiet-hours-stage-removal");
    await user.click(screen.getByTestId("quiet-hours-stage-removal"));
    expect(recipientsClient.deleteQuietWindow).not.toHaveBeenCalled();
    expect(screen.getByTestId("quiet-hours-summary-1")).toHaveTextContent("No quiet window");
    const finalReview = await openReview(user);
    expect(screen.getByRole("heading", { name: "Review quiet-hour changes" })).toBeInTheDocument();
    expect(finalReview).toHaveTextContent("Saved window");
    expect(finalReview).toHaveTextContent("21:00–23:00");
    expect(finalReview).toHaveTextContent("Time zone");
    expect(finalReview).toHaveTextContent("Critical urgency");
    expect(finalReview).toHaveTextContent("Stays quiet");
    await user.click(screen.getByRole("button", { name: "Remove Monday’s window" }));
    expect(await screen.findByTestId("quiet-hours-message")).toHaveTextContent("Every changed day is confirmed");
    expect(recipientsClient.deleteQuietWindow).toHaveBeenCalledWith({ id: "mon" });
  });

  it("keeps the saved quiet window when staged removal is declined", async () => {
    const user = userEvent.setup();
    vi.mocked(recipientsClient.deleteQuietWindow).mockReset();
    vi.mocked(recipientsClient.setQuietWindow).mockReset();
    vi.mocked(recipientsClient.listQuietWindows).mockReset().mockResolvedValue({ windows: [
      { id: "mon", weekday: 1, start: "21:00", end: "23:00", timezone: "America/New_York", criticalOverride: false },
    ] } as never);
    renderWithProviders(<QuietHoursPanel />);
    const mondayGroup = await openDay(user, 1);
    await user.click(within(mondayGroup).getByRole("button", { name: "Remove window" }));
    await user.click(within(mondayGroup).getByRole("button", { name: "Done" }));
    await openReview(user);
    await user.click(screen.getByRole("button", { name: "Keep editing" }));
    expect(screen.getByTestId("quiet-hours-summary-1")).toHaveTextContent("21:00–23:00 · America/New_York");
    expect(screen.queryByTestId("quiet-hours-review-button")).not.toBeInTheDocument();
    expect(recipientsClient.deleteQuietWindow).not.toHaveBeenCalled();
    expect(recipientsClient.setQuietWindow).not.toHaveBeenCalled();
  });

  it("reviews and saves five weekday changes independently", async () => {
    const user = userEvent.setup();
    const baseline = Array.from({ length: 5 }, (_, index) => ({ id: `day-${index + 1}`, weekday: index + 1, start: "12:00", end: "13:00", timezone: "UTC", criticalOverride: false }));
    const list = vi.mocked(recipientsClient.listQuietWindows);
    list.mockReset().mockResolvedValueOnce({ windows: baseline } as never);
    for (let confirmed = 1; confirmed <= 5; confirmed++) {
      list.mockResolvedValueOnce({ windows: baseline.map((row) => row.weekday <= confirmed ? { ...row, start: "13:00", end: "14:00" } : row) } as never);
    }
    vi.mocked(recipientsClient.setQuietWindow).mockReset().mockResolvedValue({ id: "saved" } as never);
    renderWithProviders(<QuietHoursPanel />);
    for (const day of [1, 2, 3, 4, 5]) {
      const groupElement = await openDay(user, day);
      const group = within(groupElement);
      const start = group.getByTestId(`quiet-hours-start-${day}`); await user.clear(start); await user.type(start, "13:00");
      const end = group.getByTestId(`quiet-hours-end-${day}`); await user.clear(end); await user.type(end, "14:00");
      const zone = group.getByLabelText("Time zone"); await user.clear(zone); await user.type(zone, "UTC");
      await user.click(group.getByRole("button", { name: "Done" }));
    }
    const review = await openReview(user);
    expect(review.querySelectorAll("li")).toHaveLength(5);
    expect(screen.getByRole("heading", { name: "Replace five saved windows?", level: 2 })).toBeInTheDocument();
    expect(review.parentElement).toHaveTextContent("Each changed day will use 13:00–14:00 in UTC.");
    expect(review.parentElement).toHaveTextContent("Critical urgency will also stay quiet.");
    expect(review.parentElement).toHaveTextContent("The override is off for all five days.");
    await user.click(screen.getByRole("button", { name: "Save quiet hours" }));
    expect(await screen.findByRole("status")).toHaveTextContent("Every changed day is confirmed");
    expect(recipientsClient.setQuietWindow).toHaveBeenCalledTimes(5);
  });

  it("stops after the first uncertain day and distinguishes unattempted days", async () => {
    const user = userEvent.setup();
    const list = vi.mocked(recipientsClient.listQuietWindows);
    list.mockReset().mockResolvedValueOnce({ windows: [] } as never).mockResolvedValueOnce({ windows: [
      { id: "day-1", weekday: 1, start: "22:00", end: "23:00", timezone: "UTC", criticalOverride: false },
    ] } as never).mockResolvedValueOnce({ windows: [
      { id: "day-1", weekday: 1, start: "22:00", end: "23:00", timezone: "UTC", criticalOverride: false },
      { id: "day-2", weekday: 2, start: "22:00", end: "23:00", timezone: "UTC", criticalOverride: false },
    ] } as never).mockRejectedValueOnce(new Error("confirming read unavailable"));
    const setWindow = vi.mocked(recipientsClient.setQuietWindow);
    setWindow.mockReset().mockResolvedValueOnce({ id: "day-1" } as never).mockResolvedValueOnce({ id: "day-2" } as never).mockRejectedValueOnce(new Error("write response lost"));
    renderWithProviders(<QuietHoursPanel />);
    for (const weekday of [1, 2, 3, 4, 5]) {
      const editor = await openDay(user, weekday);
      await enableDay(user, editor);
      await user.clear(within(editor).getByTestId(`quiet-hours-start-${weekday}`)); await user.type(within(editor).getByTestId(`quiet-hours-start-${weekday}`), "22:00");
      await user.clear(within(editor).getByTestId(`quiet-hours-end-${weekday}`)); await user.type(within(editor).getByTestId(`quiet-hours-end-${weekday}`), "23:00");
      await user.clear(within(editor).getByLabelText("Time zone")); await user.type(within(editor).getByLabelText("Time zone"), "UTC");
      await user.click(within(editor).getByRole("button", { name: "Done" }));
    }
    await openReview(user);
    await user.click(screen.getByRole("button", { name: "Save quiet hours" }));
    const result = await screen.findByRole("status");
    expect(result).toHaveStyle({ borderInlineStartColor: "var(--reach-guidance-rail)" });
    expect(screen.getByRole("heading", { name: "Some quiet hours need checking", level: 2 })).toBeInTheDocument();
    expect(result).toHaveTextContent("Wednesday’s write may have completed");
    expect(result).toHaveTextContent("Later days were not sent or retried");
    expect(result).toHaveTextContent("The week is only partly confirmed");
    expect(screen.getByTestId("quiet-hours-partial-retention")).toHaveTextContent("Saturday and Sunday are unchanged. Your remaining draft is retained.");
    expect(screen.getAllByText("Wednesday")[0]!.closest("li")).toHaveTextContent("Uncertain");
    const compactOutcomeRow = screen.getAllByTestId("quiet-hours-outcome-rows")[0]?.querySelector("li");
    expect(compactOutcomeRow?.className).toContain("py-3");
    expect(compactOutcomeRow?.className).toContain("sm:py-5");
    await user.click(screen.getByText("Review day-by-day values"));
    expect(screen.getByText("Latest saved value: Unknown")).toBeInTheDocument();
    expect(setWindow).toHaveBeenCalledTimes(3);
    expect(setWindow).toHaveBeenCalledTimes(3);
  });

  it("keeps the draft when a save and its confirming read both fail", async () => {
    const user = userEvent.setup();
    vi.mocked(recipientsClient.listQuietWindows).mockReset().mockResolvedValueOnce({ windows: [] } as never).mockRejectedValueOnce(new Error("read unavailable"));
    vi.mocked(recipientsClient.setQuietWindow).mockRejectedValueOnce(new Error("response lost"));
    renderWithProviders(<QuietHoursPanel />);
    const mondayElement = await openDay(user, 1);
    const monday = within(mondayElement);
    await enableDay(user, mondayElement);
    await user.clear(monday.getByTestId("quiet-hours-start-1")); await user.type(monday.getByTestId("quiet-hours-start-1"), "21:00");
    await user.clear(monday.getByTestId("quiet-hours-end-1")); await user.type(monday.getByTestId("quiet-hours-end-1"), "23:00");
    await user.clear(monday.getByLabelText("Time zone")); await user.type(monday.getByLabelText("Time zone"), "UTC");
    await user.click(monday.getByRole("button", { name: "Done" }));
    await openReview(user);
    await user.click(screen.getByRole("button", { name: "Save quiet hours" }));
    expect(await screen.findByRole("status")).toHaveTextContent("The latest read could not be confirmed");
    expect(screen.queryByText("Quiet hours could not be read.")).not.toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Quiet hours save result is unknown", level: 2 })).toBeInTheDocument();
    const mondayAfterUnknown = within(await openDay(user, 1));
    expect(mondayAfterUnknown.getByTestId("quiet-hours-start-1")).toHaveValue("21:00");
  });

  it("matches selected Japanese quiet-editor labels and explains the critical-label boundary", async () => {
    const previous = document.documentElement.lang;
    document.documentElement.lang = "ja";
    window.history.replaceState({}, "", "/settings?e22_fixture=settings-owner");
    try {
      vi.mocked(recipientsClient.listQuietWindows).mockResolvedValue({ windows: [
        { id: "quiet-demo-1", weekday: 1, start: "13:00", end: "14:00", timezone: "Etc/UTC", criticalOverride: false },
      ] } as never);
      renderWithProviders(<QuietHoursPanel />);
      const user = userEvent.setup();
      const monday = within(await openDay(user, 1));
      expect(screen.getByTestId("quiet-hours-day-meta-1")).toHaveTextContent("月曜日 · quiet-demo-1");
      expect(monday.getByLabelText("通知を停止し始める時刻")).toHaveValue("13:00");
      expect(monday.getByLabelText("通知の停止を終了する時刻")).toHaveValue("14:00");
      expect(monday.getByLabelText("この曜日に適用するタイムゾーン")).toHaveValue("Etc/UTC");
      expect(monday.getByRole("checkbox", { name: "緊急度が最も高い通知をこの時間帯も許可する" })).not.toBeChecked();
      expect(monday.getByText("緊急度の例外は、通知内容の「Critical」ラベルへのアクセス許可とは別の設定です。")).toBeInTheDocument();
      expect(monday.getByTestId("quiet-hours-done-stages-1")).toHaveTextContent("「完了」で月曜日の変更を下書きに反映します。");
    } finally { document.documentElement.lang = previous; }
  });

  it("renders selected Japanese weekday and input labels", async () => {
    const previous = document.documentElement.lang;
    document.documentElement.lang = "ja";
    try {
      vi.mocked(recipientsClient.listQuietWindows).mockResolvedValue({ windows: [] } as never);
      renderWithProviders(<QuietHoursPanel />);
      await screen.findByTestId("quiet-hours-edit-day-1");
      await userEvent.setup().click(screen.getByTestId("quiet-hours-edit-day-1"));
      const monday = within(screen.getByTestId("quiet-hours-editor-1"));
      expect(monday.getByRole("heading", { name: "月曜日の通知を停止する時間帯" })).toBeInTheDocument();
      await userEvent.setup().click(monday.getByRole("checkbox", { name: "通知を控える時間を設定" }));
      expect(monday.getByTestId("quiet-hours-start-1")).toBeInTheDocument();
      expect(monday.getByLabelText("この曜日に適用するタイムゾーン")).toBeInTheDocument();
    } finally { document.documentElement.lang = previous; }
  });
});

