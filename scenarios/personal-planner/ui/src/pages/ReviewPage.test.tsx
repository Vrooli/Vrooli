import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { fetchDailyReview, fetchReflection, fetchWeeklyReview, saveReflection } from "../api/review";
import { carryForwardAllocation, fetchAllocations, fetchTodayAllocations } from "../api/calendar";
import { renderWithProviders } from "../test-utils";
import { ReviewPage } from "./ReviewPage";

vi.mock("../api/review", () => ({ fetchDailyReview: vi.fn(), fetchReflection: vi.fn(), fetchWeeklyReview: vi.fn(), saveReflection: vi.fn() }));
vi.mock("../api/calendar", () => ({ carryForwardAllocation: vi.fn(), fetchAllocations: vi.fn(), fetchTodayAllocations: vi.fn() }));
afterEach(() => { cleanup(); vi.clearAllMocks(); });
beforeEach(() => {
  vi.mocked(fetchAllocations).mockResolvedValue({ allocations: [] } as never);
  vi.mocked(fetchTodayAllocations).mockResolvedValue({ availableMinutes: 360n, breathingRoomMinutes: 360n, allocations: [] } as never);
  vi.mocked(fetchReflection).mockResolvedValue({ localDate: "2026-09-19", text: "", updatedAt: "" } as never);
});

describe("ReviewPage", () => {
  it("shows measured activity and unknown coverage separately", async () => {
    vi.mocked(fetchDailyReview).mockResolvedValue({ localDate: "2026-09-19", plannedMinutes: 0n, recordedActiveMinutes: 25n, focusSessionCount: 1n, activeGoalCount: 2n, unrecordedMinutes: 0n, coverageNote: "Recorded focus is measured." } as never);
    renderWithProviders(<ReviewPage />);
    expect(await screen.findByText("25 min")).toBeInTheDocument();
    expect(screen.getByText("Recorded focus is measured.")).toBeInTheDocument();
    expect(screen.getByText(/Unrecorded: unknown/)).toBeInTheDocument();
    expect(fetchDailyReview).toHaveBeenCalledWith(expect.stringMatching(/^\d{4}-\d{2}-\d{2}$/));
  });

  it("lets the user review an adjacent local day", async () => {
    vi.mocked(fetchDailyReview).mockResolvedValue({ localDate: "2026-09-19", plannedMinutes: 0n, recordedActiveMinutes: 0n, focusSessionCount: 0n, activeGoalCount: 0n, unrecordedMinutes: 0n, coverageNote: "No recorded activity." } as never);
    renderWithProviders(<ReviewPage />);
    await userEvent.click(await screen.findByRole("button", { name: "Previous day" }));
    expect(fetchDailyReview).toHaveBeenCalledTimes(2);
    expect(vi.mocked(fetchDailyReview).mock.calls[1]?.[0]).not.toBe(vi.mocked(fetchDailyReview).mock.calls[0]?.[0]);
  });

  it("does not hide a review outage", async () => {
    vi.mocked(fetchDailyReview).mockRejectedValue(new Error("offline"));
    renderWithProviders(<ReviewPage />);
    expect(await screen.findByRole("alert")).toHaveTextContent("Review data is unavailable");
  });

  it("shows a weekly planned-versus-recorded table", async () => {
    vi.mocked(fetchDailyReview).mockResolvedValue({ localDate: "2026-09-19", plannedMinutes: 0n, recordedActiveMinutes: 0n, focusSessionCount: 0n, activeGoalCount: 0n, unrecordedMinutes: 0n, coverageNote: "Daily." } as never);
    vi.mocked(fetchWeeklyReview).mockResolvedValue({ weekStartLocalDate: "2026-09-14", days: [{ localDate: "2026-09-15", plannedMinutes: 120n, recordedActiveMinutes: 90n, focusSessionCount: 2n, activeGoalCount: 1n, unrecordedMinutes: 0n, coverageNote: "Weekly." }], plannedMinutes: 120n, recordedActiveMinutes: 90n, focusSessionCount: 2n, activeGoalCount: 1n, coverageNote: "Weekly coverage remains explicit." } as never);
    renderWithProviders(<ReviewPage />);
    await userEvent.click(screen.getByRole("tab", { name: "Week" }));
    expect(await screen.findByRole("table")).toBeInTheDocument();
    expect(screen.getByText("2026-09-15")).toBeInTheDocument();
    expect(screen.getByText("Weekly coverage remains explicit.")).toBeInTheDocument();
    expect(fetchWeeklyReview).toHaveBeenCalledWith(expect.stringMatching(/^\d{4}-\d{2}-\d{2}$/));
  });

  it("offers selective carry-forward with a target-day capacity preview", async () => {
    vi.mocked(fetchDailyReview).mockResolvedValue({ localDate: "2026-09-19", plannedMinutes: 45n, recordedActiveMinutes: 0n, focusSessionCount: 0n, activeGoalCount: 0n, unrecordedMinutes: 0n, coverageNote: "Coverage." } as never);
    vi.mocked(fetchAllocations).mockResolvedValue({ allocations: [{ id: "allocation-1", title: "Draft", startMinutes: 600, durationMinutes: 45 } as never] } as never);
    vi.mocked(carryForwardAllocation).mockResolvedValue({ id: "allocation-2", carriedFromId: "allocation-1" } as never);
    renderWithProviders(<ReviewPage />);
    expect(await screen.findByText("Target capacity preview: 360 min available · 360 min breathing room before this move.")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Carry this" }));
    expect(carryForwardAllocation).toHaveBeenCalledWith(expect.objectContaining({ allocationId: "allocation-1", targetLocalDate: expect.stringMatching(/^\d{4}-\d{2}-\d{2}$/), startMinutes: 600 }));
    expect(await screen.findByText(/Carried forward once/)).toBeInTheDocument();
  });

  it("saves an optional daily reflection", async () => {
    vi.mocked(fetchDailyReview).mockResolvedValue({ localDate: "2026-09-19", plannedMinutes: 0n, recordedActiveMinutes: 0n, focusSessionCount: 0n, activeGoalCount: 0n, unrecordedMinutes: 0n, coverageNote: "Coverage." } as never);
    vi.mocked(saveReflection).mockResolvedValue({ localDate: "2026-09-19", text: "Protect the first hour.", updatedAt: "" } as never);
    renderWithProviders(<ReviewPage />);
    const input = await screen.findByRole("textbox", { name: "Daily reflection" });
    await userEvent.type(input, "Protect the first hour.");
    await userEvent.click(screen.getByRole("button", { name: "Save reflection" }));
    expect(saveReflection).toHaveBeenCalledWith(expect.objectContaining({ text: "Protect the first hour." }));
    expect(await screen.findByText("Reflection saved.")).toBeInTheDocument();
  });
});
