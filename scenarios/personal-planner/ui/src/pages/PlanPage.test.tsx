import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { createAllocation, createRoutine, fetchAllocations, fetchRoutineOccurrences, fetchRoutines, fetchTodayAllocations, rescheduleRoutineOccurrence, skipRoutineOccurrence } from "../api/calendar";
import { fetchWorkItems } from "../api/work";
import { renderWithProviders } from "../test-utils";
import { PlanPage } from "./PlanPage";

vi.mock("../api/calendar", () => ({ fetchTodayAllocations: vi.fn(), fetchAllocations: vi.fn(), createAllocation: vi.fn(), fetchRoutines: vi.fn(), createRoutine: vi.fn(), fetchRoutineOccurrences: vi.fn(), rescheduleRoutineOccurrence: vi.fn(), skipRoutineOccurrence: vi.fn() }));
vi.mock("../api/work", () => ({ fetchWorkItems: vi.fn() }));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

beforeEach(() => {
  vi.mocked(fetchAllocations).mockResolvedValue({ allocations: [] } as never);
  vi.mocked(fetchRoutines).mockResolvedValue([]);
  vi.mocked(fetchRoutineOccurrences).mockResolvedValue([]);
});

describe("PlanPage", () => {
  it("shows capacity and a positioned accepted allocation", async () => {
    vi.mocked(fetchTodayAllocations).mockResolvedValue({
      plannedMinutes: 45,
      availableMinutes: 315,
      breathingRoomMinutes: 45,
      allocations: [{ id: "allocation-1", workItemId: "work-1", title: "Draft the launch story", startMinutes: 600, durationMinutes: 45, sourceLabel: "Cadence", state: "accepted" }],
    } as never);
    vi.mocked(fetchWorkItems).mockResolvedValue([]);

    renderWithProviders(<PlanPage />);

    expect(await screen.findByRole("heading", { name: "Draft the launch story" })).toBeInTheDocument();
    expect(screen.getAllByText("45 min")).toHaveLength(2);
    expect(screen.getByText("10:00 · 45 min")).toBeInTheDocument();
    expect(screen.getByText("Accepted placement is durable schedule state. Completing focus does not silently mark an allocation or work item complete.")).toBeInTheDocument();
  });

  it("keeps an empty plan explicit", async () => {
    vi.mocked(fetchTodayAllocations).mockResolvedValue({ plannedMinutes: 0, availableMinutes: 360, breathingRoomMinutes: 60, allocations: [] } as never);
    vi.mocked(fetchWorkItems).mockResolvedValue([]);
    renderWithProviders(<PlanPage />);
    expect(await screen.findByText("No accepted allocations yet. Place the next work item when you are ready.")).toBeInTheDocument();
  });

  it("announces an unavailable plan", async () => {
    vi.mocked(fetchTodayAllocations).mockRejectedValue(new Error("offline"));
    renderWithProviders(<PlanPage />);
    expect(await screen.findByRole("alert")).toHaveTextContent("Today’s plan is unavailable right now.");
  });

  it("places the next work item into the accepted schedule", async () => {
    vi.mocked(fetchTodayAllocations).mockResolvedValue({ plannedMinutes: 0, availableMinutes: 480, breathingRoomMinutes: 480, allocations: [] } as never);
    vi.mocked(fetchWorkItems).mockResolvedValue([{ id: "work-1", title: "Draft", remainingMinutes: 45, sourceLabel: "Cadence" }] as never);
    vi.mocked(createAllocation).mockResolvedValue({ id: "allocation-1" } as never);
    renderWithProviders(<PlanPage />);
    const placementButton = await screen.findByRole("button", { name: "Accept placement" });
    await waitFor(() => expect(placementButton).toBeEnabled());
    await userEvent.clear(screen.getByLabelText("Start"));
    await userEvent.type(screen.getByLabelText("Start"), "13:30");
    await userEvent.clear(screen.getByLabelText("Minutes"));
    await userEvent.type(screen.getByLabelText("Minutes"), "60");
    await userEvent.click(placementButton);
    expect(createAllocation).toHaveBeenCalledWith(expect.objectContaining({ workItemId: "work-1", startMinutes: 810, durationMinutes: 60 }));
    expect(await screen.findByRole("status")).toHaveTextContent("Placed on today’s accepted schedule.");
  });

  it("keeps rejected placement explicit", async () => {
    vi.mocked(fetchTodayAllocations).mockResolvedValue({ plannedMinutes: 0, availableMinutes: 480, breathingRoomMinutes: 480, allocations: [] } as never);
    vi.mocked(fetchWorkItems).mockResolvedValue([{ id: "work-1", title: "Draft", remainingMinutes: 45 }] as never);
    vi.mocked(createAllocation).mockRejectedValue(new Error("overlap"));
    renderWithProviders(<PlanPage />);
    await userEvent.click(await screen.findByRole("button", { name: "Accept placement" }));
    expect(await screen.findByRole("status")).toHaveTextContent("That placement could not be accepted.");
  });

  it("projects the shared allocation set as a week", async () => {
    vi.mocked(fetchTodayAllocations).mockResolvedValue({ plannedMinutes: 0, availableMinutes: 480, breathingRoomMinutes: 480, allocations: [] } as never);
    vi.mocked(fetchWorkItems).mockResolvedValue([]);
    vi.mocked(fetchAllocations).mockResolvedValue({ allocations: [{ id: "allocation-2", localDate: "2026-09-22", title: "Review the brief", startMinutes: 660, durationMinutes: 30, sourceLabel: "Daily" }] } as never);
    renderWithProviders(<PlanPage />);
    await userEvent.click(await screen.findByRole("tab", { name: "Week" }));
    expect(await screen.findByRole("heading", { name: "Tue, Sep 22" })).toBeInTheDocument();
    expect(screen.getByText("Review the brief")).toBeInTheDocument();
    expect(screen.getByText("11:00 · 30 min")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("tab", { name: "Agenda" }));
    expect(await screen.findByRole("heading", { name: "Upcoming accepted work" })).toBeInTheDocument();
    await userEvent.click(screen.getByRole("tab", { name: "Day" }));
    expect(screen.getByText("PLANNED")).toBeInTheDocument();
  });

  it("does not turn a range outage into an empty state", async () => {
    vi.mocked(fetchTodayAllocations).mockResolvedValue({ plannedMinutes: 0, availableMinutes: 480, breathingRoomMinutes: 480, allocations: [] } as never);
    vi.mocked(fetchWorkItems).mockResolvedValue([]);
    vi.mocked(fetchAllocations).mockRejectedValue(new Error("offline"));
    renderWithProviders(<PlanPage />);
    await userEvent.click(await screen.findByRole("tab", { name: "Agenda" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("The accepted range is unavailable right now.");
  });

  it("creates a local-time routine and shows its generated rhythm", async () => {
    vi.mocked(fetchTodayAllocations).mockResolvedValue({ plannedMinutes: 0, availableMinutes: 480, breathingRoomMinutes: 480, allocations: [] } as never);
    vi.mocked(fetchWorkItems).mockResolvedValue([]);
    vi.mocked(fetchRoutines).mockResolvedValue([]);
    vi.mocked(fetchRoutineOccurrences).mockResolvedValue([]);
    vi.mocked(createRoutine).mockResolvedValue({ id: "routine-1", title: "Morning run", kind: "fixed", durationMinutes: 30 } as never);
    renderWithProviders(<PlanPage />);
    await userEvent.type(await screen.findByLabelText("Routine title"), "Morning run");
    await userEvent.click(screen.getByRole("button", { name: "Add routine" }));
    await waitFor(() => expect(createRoutine).toHaveBeenCalledWith(expect.objectContaining({ title: "Morning run", kind: "fixed", weekdays: [1] })));
    expect(await screen.findByRole("status")).toHaveTextContent("Routine saved");
  });

  it("skips one generated occurrence without changing the series", async () => {
    vi.mocked(fetchTodayAllocations).mockResolvedValue({ plannedMinutes: 0, availableMinutes: 480, breathingRoomMinutes: 480, allocations: [] } as never);
    vi.mocked(fetchWorkItems).mockResolvedValue([]);
    vi.mocked(fetchRoutines).mockResolvedValue([{ id: "routine-1", title: "Morning run", kind: "fixed", revision: 3n, durationMinutes: 30 }] as never);
    vi.mocked(fetchRoutineOccurrences).mockResolvedValue([{ routineId: "routine-1", localDate: "2026-09-21", title: "Morning run", durationMinutes: 30 }] as never);
    vi.mocked(skipRoutineOccurrence).mockResolvedValue();
    renderWithProviders(<PlanPage />);
    await userEvent.click(await screen.findByRole("button", { name: "Skip once" }));
    expect(skipRoutineOccurrence).toHaveBeenCalledWith({ routineId: "routine-1", localDate: "2026-09-21", expectedRevision: 3n });
    expect(await screen.findByRole("status")).toHaveTextContent("Occurrence skipped");
  });

  it("moves one generated occurrence without changing the series", async () => {
    vi.mocked(fetchTodayAllocations).mockResolvedValue({ plannedMinutes: 0, availableMinutes: 480, breathingRoomMinutes: 480, allocations: [] } as never);
    vi.mocked(fetchWorkItems).mockResolvedValue([]);
    vi.mocked(fetchRoutines).mockResolvedValue([{ id: "routine-1", title: "Morning run", kind: "fixed", revision: 3n, durationMinutes: 30 } ] as never);
    vi.mocked(fetchRoutineOccurrences).mockResolvedValue([{ routineId: "routine-1", localDate: "2026-09-21", title: "Morning run", startMinute: 540, durationMinutes: 30 } ] as never);
    vi.mocked(rescheduleRoutineOccurrence).mockResolvedValue();
    renderWithProviders(<PlanPage />);
    await userEvent.click(await screen.findByRole("button", { name: "Move +30m" }));
    expect(rescheduleRoutineOccurrence).toHaveBeenCalledWith({ routineId: "routine-1", localDate: "2026-09-21", startMinute: 570, expectedRevision: 3n });
    expect(await screen.findByRole("status")).toHaveTextContent("moved 30 minutes");
  });
});
