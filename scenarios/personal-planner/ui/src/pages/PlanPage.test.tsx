import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { applyAllocationProposal, applyScheduleProposal, createRoutine, fetchAllocations, fetchRoutineOccurrences, fetchRoutines, fetchTodayAllocations, previewAllocation, previewSchedule, rescheduleRoutineOccurrence, skipRoutineOccurrence } from "../api/calendar";
import { fetchWorkItems, updateWorkEstimate } from "../api/work";
import { createCommitment, fetchCommitments } from "../api/commitments";
import { fetchForecast, fetchForecastHistory } from "../api/forecasts";
import { renderWithProviders } from "../test-utils";
import { PlanPage } from "./PlanPage";

vi.mock("../api/calendar", () => ({ fetchTodayAllocations: vi.fn(), fetchAllocations: vi.fn(), applyAllocationProposal: vi.fn(), applyScheduleProposal: vi.fn(), previewAllocation: vi.fn(), previewSchedule: vi.fn(), fetchRoutines: vi.fn(), createRoutine: vi.fn(), fetchRoutineOccurrences: vi.fn(), rescheduleRoutineOccurrence: vi.fn(), skipRoutineOccurrence: vi.fn() }));
vi.mock("../api/work", () => ({ fetchWorkItems: vi.fn(), updateWorkEstimate: vi.fn() }));
vi.mock("../api/commitments", () => ({ createCommitment: vi.fn(), fetchCommitments: vi.fn(), updateCommitmentState: vi.fn() }));
vi.mock("../api/forecasts", () => ({ fetchForecast: vi.fn(), fetchForecastHistory: vi.fn() }));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  vi.unstubAllGlobals();
});

beforeEach(() => {
  vi.mocked(fetchAllocations).mockResolvedValue({ allocations: [] } as never);
  vi.mocked(fetchRoutines).mockResolvedValue([]);
  vi.mocked(fetchRoutineOccurrences).mockResolvedValue([]);
  vi.mocked(fetchCommitments).mockResolvedValue([]);
  vi.mocked(fetchForecast).mockResolvedValue({ centralFinish: "2026-10-02", cautiousFinish: "2026-10-03", resultState: "feasible_in_scenario", riskState: "on_track", explanation: "Known work fits.", horizonStart: "2026-10-01", horizonEnd: "2026-10-28", knownWorkMinutes: 60n, reserveMinutes: 1680n, freshness: "current", commitmentOutlooks: [{ id: "commitment-1", result: "Send the brief", promisedBoundary: "2026-10-02", forecastFinish: "2026-10-02", riskState: "on_track", explanation: "Both labeled scenarios finish by the promised boundary of 2026-10-02." }] } as never);
  vi.mocked(fetchForecastHistory).mockResolvedValue([]);
});

describe("PlanPage", () => {
  it("keeps the mobile plan spine to Day and Week until More is opened", async () => {
    const user = userEvent.setup();
    vi.stubGlobal("matchMedia", () => ({ matches: true, addEventListener: vi.fn(), removeEventListener: vi.fn() }));
    vi.mocked(fetchTodayAllocations).mockResolvedValue({ plannedMinutes: 0, availableMinutes: 480, breathingRoomMinutes: 60, allocations: [] } as never);
    vi.mocked(fetchWorkItems).mockResolvedValue([]);
    renderWithProviders(<PlanPage />);
    expect(await screen.findByRole("tab", { name: "Day" })).toBeInTheDocument();
    expect(screen.queryByRole("tab", { name: "Agenda" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("tab", { name: "More" }));
    expect(screen.getByRole("dialog", { name: "More planning views" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Agenda" }));
    expect(screen.queryByRole("dialog", { name: "More planning views" })).not.toBeInTheDocument();
  });

  it("uses the mobile capacity ring instead of the desktop stat strip", async () => {
    vi.stubGlobal("matchMedia", () => ({ matches: true, addEventListener: vi.fn(), removeEventListener: vi.fn() }));
    vi.mocked(fetchTodayAllocations).mockResolvedValue({ plannedMinutes: 90, availableMinutes: 270, breathingRoomMinutes: 60, allocations: [] } as never);
    vi.mocked(fetchWorkItems).mockResolvedValue([]);

    renderWithProviders(<PlanPage />);

    expect(await screen.findByLabelText("90 minutes planned of 360 available minutes")).toBeInTheDocument();
    expect(screen.getByText("25%")).toBeInTheDocument();
    expect(screen.queryByText("PLANNED")).not.toBeInTheDocument();
  });

  it("opens mobile placement as a focused sheet from the thumb-zone action", async () => {
    const user = userEvent.setup();
    vi.stubGlobal("matchMedia", () => ({ matches: true, addEventListener: vi.fn(), removeEventListener: vi.fn() }));
    vi.mocked(fetchTodayAllocations).mockResolvedValue({ plannedMinutes: 0, availableMinutes: 480, breathingRoomMinutes: 60, allocations: [] } as never);
    vi.mocked(fetchWorkItems).mockResolvedValue([{ id: "work-1", title: "Draft", remainingMinutes: 45 }] as never);

    renderWithProviders(<PlanPage />);

    expect(await screen.findByRole("button", { name: "More planning views" })).toBeInTheDocument();
    await user.click(await screen.findByRole("button", { name: "+ Place work" }));
    const dialog = screen.getByRole("dialog", { name: "Plot work on today’s chart" });
    expect(dialog).toBeInTheDocument();
    expect(within(dialog).getByTestId("forms.select")).toHaveTextContent("Draft");
  });

  it("keeps estimate calibration behind the mobile More sheet", async () => {
    const user = userEvent.setup();
    vi.stubGlobal("matchMedia", () => ({ matches: true, addEventListener: vi.fn(), removeEventListener: vi.fn() }));
    vi.mocked(fetchTodayAllocations).mockResolvedValue({ plannedMinutes: 0, availableMinutes: 480, breathingRoomMinutes: 60, allocations: [] } as never);
    vi.mocked(fetchWorkItems).mockResolvedValue([{ id: "work-1", title: "Draft", remainingMinutes: 45 }, { id: "work-2", title: "Review", remainingMinutes: 60 }] as never);
    vi.mocked(updateWorkEstimate).mockResolvedValue({} as never);

    renderWithProviders(<PlanPage />);

    expect(await screen.findByRole("button", { name: "More planning views" })).toBeInTheDocument();
    expect(screen.queryByText("Keep the estimate honest")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "More planning views" }));
    await user.click(screen.getByRole("button", { name: "Update estimate" }));
    const dialog = screen.getByRole("dialog", { name: "Update the estimate" });
    expect(within(dialog).getByText("Keep the estimate honest")).toBeInTheDocument();
    const workSelect = within(dialog).getAllByRole("combobox", { hidden: true }).find((element) => element.getAttribute("aria-label") === "Estimate work item");
    if (!workSelect) throw new Error("Estimate work item select was not rendered");
    await user.selectOptions(workSelect, "work-2");
    const remainingMinutes = within(dialog).getByLabelText("Remaining minutes");
    expect(remainingMinutes).toHaveValue(60);
    await user.clear(remainingMinutes);
    await user.type(remainingMinutes, "30");
    await user.click(within(dialog).getByRole("button", { name: "Save estimate" }));
    await waitFor(() => expect(updateWorkEstimate).toHaveBeenCalledWith(expect.objectContaining({ id: "work-2", remainingMinutes: 30 })));
    await waitFor(() => expect(screen.queryByRole("dialog", { name: "Update the estimate" })).not.toBeInTheDocument());
    await user.click(screen.getByRole("button", { name: "More planning views" }));
    await user.click(screen.getByRole("button", { name: "Update estimate" }));
    const dismiss = screen.queryByTestId("overlays.responsive-dialog.grabber") ?? screen.queryByTestId("overlays.responsive-dialog.close");
    if (!dismiss) throw new Error("Responsive dialog dismiss affordance was not rendered");
    await user.click(dismiss);
    expect(screen.queryByRole("dialog", { name: "Update the estimate" })).not.toBeInTheDocument();
  });

  it("keeps placement visible in the desktop command chart", async () => {
    vi.stubGlobal("matchMedia", () => ({ matches: false, addEventListener: vi.fn(), removeEventListener: vi.fn() }));
    vi.mocked(fetchTodayAllocations).mockResolvedValue({ plannedMinutes: 0, availableMinutes: 480, breathingRoomMinutes: 60, allocations: [] } as never);
    vi.mocked(fetchWorkItems).mockResolvedValue([{ id: "work-1", title: "Draft", remainingMinutes: 45 }] as never);

    renderWithProviders(<PlanPage />);

    expect(await screen.findByRole("button", { name: "+ Place work" })).toBeInTheDocument();
  });

  it("shows capacity and a positioned accepted allocation", async () => {
    vi.mocked(fetchTodayAllocations).mockResolvedValue({
      plannedMinutes: 45,
      availableMinutes: 315,
      breathingRoomMinutes: 45,
      allocations: [{ id: "allocation-1", workItemId: "work-1", title: "Draft the launch story", startMinutes: 600, durationMinutes: 45, sourceLabel: "Cadence", state: "accepted" }],
    } as never);
    vi.mocked(fetchWorkItems).mockResolvedValue([]);

    renderWithProviders(<PlanPage />);

    const block = await screen.findByRole("group", { name: "Draft the launch story, 10:00, 45 minutes" });
    expect(block.querySelector("h2")).toHaveTextContent("45m");
    expect(screen.getAllByText("45 min")).toHaveLength(2);
    expect(screen.getByText("10:00 · 45 min")).toBeInTheDocument();
    expect(screen.getByText("Accepted placement is durable schedule state. Completing focus does not silently mark an allocation or work item complete.")).toBeInTheDocument();
  });

  it("preserves short allocation geometry and opens accepted details", async () => {
    vi.mocked(fetchTodayAllocations).mockResolvedValue({
      plannedMinutes: 15,
      availableMinutes: 345,
      breathingRoomMinutes: 45,
      allocations: [{ id: "allocation-short", workItemId: "work-1", title: "Quick check", startMinutes: 600, durationMinutes: 15, sourceLabel: "Cadence", state: "accepted" }],
    } as never);
    vi.mocked(fetchWorkItems).mockResolvedValue([]);

    renderWithProviders(<PlanPage />);

    const block = await screen.findByRole("group", { name: "Quick check, 10:00, 15 minutes" });
    expect(block).toHaveStyle({ width: "2.5%" });
    expect(block.querySelector("h2")).toHaveTextContent("15m");
    await userEvent.click(block);
    expect(screen.getByRole("dialog", { name: "Accepted allocation details" })).toHaveTextContent("10:00 · 15 min");
  });

  it("projects truthful capacity as its own planning view", async () => {
    vi.mocked(fetchTodayAllocations).mockResolvedValue({
      plannedMinutes: 90,
      availableMinutes: 270,
      breathingRoomMinutes: 60,
      externalBusyMinutes: 45,
      externalEventCount: 2,
      externalFreshness: "fresh",
      allocations: [{ id: "allocation-1", workItemId: "work-1", title: "Draft the launch story", startMinutes: 600, durationMinutes: 90, sourceLabel: "Cadence", state: "accepted" }],
    } as never);
    vi.mocked(fetchWorkItems).mockResolvedValue([]);

    renderWithProviders(<PlanPage />);

    await userEvent.click(await screen.findByRole("tab", { name: "Capacity" }));
    expect(await screen.findByRole("heading", { name: "Make room before you make promises." })).toBeInTheDocument();
    expect(screen.getByText("90 min accepted")).toBeInTheDocument();
    expect(screen.getByText("45 min")).toBeInTheDocument();
    expect(screen.getByText("2 read-only calendar holds.")).toBeInTheDocument();
    expect(screen.getByText("Provider freshness: fresh.")).toBeInTheDocument();
    expect(screen.getByText("10:00 · Draft the launch story")).toBeInTheDocument();
    expect(screen.queryByLabelText("Today’s accepted schedule")).not.toBeInTheDocument();
  });

  it("shows deterministic central and cautious outlooks without calling them probabilities", async () => {
    vi.mocked(fetchTodayAllocations).mockResolvedValue({ plannedMinutes: 0, availableMinutes: 360, breathingRoomMinutes: 60, allocations: [] } as never);
    vi.mocked(fetchWorkItems).mockResolvedValue([]);
    renderWithProviders(<PlanPage />);
    await userEvent.click(await screen.findByRole("tab", { name: "Outlook" }));
    expect(await screen.findByRole("heading", { name: "See what the shared resource can carry." })).toBeInTheDocument();
    expect(screen.getByText("2026-10-02")).toBeInTheDocument();
    expect(screen.getByText("Known work fits.")).toBeInTheDocument();
    expect(screen.getByText(/not probabilities/)).toBeInTheDocument();
    expect(screen.getByText("Promise and outlook, side by side")).toBeInTheDocument();
    expect(screen.getByText("Send the brief")).toBeInTheDocument();
  });

  it("keeps an empty plan explicit", async () => {
    vi.mocked(fetchTodayAllocations).mockResolvedValue({ plannedMinutes: 0, availableMinutes: 360, breathingRoomMinutes: 60, allocations: [] } as never);
    vi.mocked(fetchWorkItems).mockResolvedValue([]);
    renderWithProviders(<PlanPage />);
    expect(await screen.findByText("No accepted allocations yet. Place the next work item when you are ready.")).toBeInTheDocument();
  });

  it("records an explicit commitment and keeps its lifecycle separate from the schedule", async () => {
    vi.mocked(fetchTodayAllocations).mockResolvedValue({ plannedMinutes: 0, availableMinutes: 360, breathingRoomMinutes: 60, allocations: [] } as never);
    vi.mocked(fetchWorkItems).mockResolvedValue([]);
    vi.mocked(createCommitment).mockResolvedValue({ id: "commitment-1", result: "Send the brief", promisedBoundary: "2026-10-01", state: "proposed", risk: "unknown", acknowledgmentStatus: "unknown", revision: 1n } as never);
    renderWithProviders(<PlanPage />);
    await userEvent.click(await screen.findByRole("tab", { name: "Commitments" }));
    expect(await screen.findByRole("heading", { name: "Keep the promise visible." })).toBeInTheDocument();
    await userEvent.type(screen.getByLabelText("Promised result"), "Send the brief");
    await userEvent.type(screen.getByLabelText("Promise boundary"), "2026-10-01");
    await userEvent.type(screen.getByLabelText("What counts as done?"), "A reviewed brief is sent");
    await userEvent.type(screen.getByLabelText("Out of scope"), "Follow-up campaign");
    await userEvent.click(screen.getByRole("button", { name: "Record commitment" }));
    await waitFor(() => expect(createCommitment).toHaveBeenCalledWith(expect.objectContaining({ result: "Send the brief", promisedBoundary: "2026-10-01", definitionOfDone: "A reviewed brief is sent", scopeExclusions: "Follow-up campaign", state: "proposed" })));
    expect(await screen.findByRole("status")).toHaveTextContent("Commitment recorded");
  });

  it("announces an unavailable plan", async () => {
    vi.mocked(fetchTodayAllocations).mockRejectedValue(new Error("offline"));
    renderWithProviders(<PlanPage />);
    expect(await screen.findByRole("alert")).toHaveTextContent("Today’s plan is unavailable right now.");
  });

  it("previews and then places the next work item into the accepted schedule", async () => {
    vi.mocked(fetchTodayAllocations).mockResolvedValue({ plannedMinutes: 0, availableMinutes: 480, breathingRoomMinutes: 480, allocations: [] } as never);
    vi.mocked(fetchWorkItems).mockResolvedValue([{ id: "work-1", title: "Draft", remainingMinutes: 45, sourceLabel: "Cadence" }] as never);
    vi.mocked(previewAllocation).mockResolvedValue({ id: "proposal-1", baseRevision: 1n, state: "feasible", startMinutes: 810, durationMinutes: 60, reason: "Requested time is available." } as never);
    vi.mocked(applyAllocationProposal).mockResolvedValue({ id: "allocation-1" } as never);
    renderWithProviders(<PlanPage />);
    const placementButton = await screen.findByRole("button", { name: "Preview placement" });
    await waitFor(() => expect(placementButton).toBeEnabled());
    await userEvent.clear(screen.getByLabelText("Start"));
    await userEvent.type(screen.getByLabelText("Start"), "13:30");
    await userEvent.clear(screen.getByLabelText("Minutes"));
    await userEvent.type(screen.getByLabelText("Minutes"), "60");
    await userEvent.click(placementButton);
    expect(previewAllocation).toHaveBeenCalledWith(expect.objectContaining({ workItemId: "work-1", startMinutes: 810, durationMinutes: 60 }));
    expect(screen.getByText("Requested 13:30")).toBeInTheDocument();
    expect(screen.getByText("Proposed 13:30")).toBeInTheDocument();
    await userEvent.click(await screen.findByRole("button", { name: "Accept 13:30" }));
    expect(applyAllocationProposal).toHaveBeenCalledWith({ proposalId: "proposal-1", expectedRevision: 1n, idempotencyKey: "proposal-1:apply" });
    expect(await screen.findByRole("status")).toHaveTextContent("Placed on today’s accepted schedule");
  });

  it("records a changed remaining estimate and its reason", async () => {
    vi.mocked(fetchTodayAllocations).mockResolvedValue({ plannedMinutes: 0, availableMinutes: 480, breathingRoomMinutes: 480, allocations: [] } as never);
    vi.mocked(fetchWorkItems).mockResolvedValue([{ id: "work-1", title: "Draft", remainingMinutes: 45 }] as never);
    vi.mocked(updateWorkEstimate).mockResolvedValue(undefined);
    renderWithProviders(<PlanPage />);
    await userEvent.clear(await screen.findByLabelText("Remaining minutes"));
    await userEvent.type(screen.getByLabelText("Remaining minutes"), "75");
    const reasonSelect = screen.getAllByLabelText("Estimate change reason").find((element) => element.tagName === "SELECT");
    if (!reasonSelect) throw new Error("estimate reason select not found");
    await userEvent.selectOptions(reasonSelect, "scope_changed");
    await userEvent.click(screen.getByRole("button", { name: "Save estimate" }));
    await waitFor(() => expect(updateWorkEstimate).toHaveBeenCalledWith({ id: "work-1", remainingMinutes: 75, reason: "scope_changed" }));
    expect(await screen.findByRole("status")).toHaveTextContent("Estimate updated and remembered");
  });

  it("suggests a morning window for deep work and lets the planner accept it", async () => {
    vi.mocked(fetchTodayAllocations).mockResolvedValue({ plannedMinutes: 0, availableMinutes: 480, breathingRoomMinutes: 480, allocations: [] } as never);
    vi.mocked(fetchWorkItems).mockResolvedValue([{ id: "work-1", title: "Draft the launch story", remainingMinutes: 45 }] as never);
    renderWithProviders(<PlanPage />);
    await userEvent.clear(await screen.findByLabelText("Start"));
    await userEvent.type(screen.getByLabelText("Start"), "13:00");
    expect(await screen.findByLabelText("Morning energy suggestion")).toHaveTextContent("This looks like deep work.");
    await userEvent.click(screen.getByRole("button", { name: "Use 09:00" }));
    expect(screen.getByLabelText("Start")).toHaveValue("09:00");
  });

  it("keeps rejected placement explicit", async () => {
    vi.mocked(fetchTodayAllocations).mockResolvedValue({ plannedMinutes: 0, availableMinutes: 480, breathingRoomMinutes: 480, allocations: [] } as never);
    vi.mocked(fetchWorkItems).mockResolvedValue([{ id: "work-1", title: "Draft", remainingMinutes: 45 }] as never);
    vi.mocked(previewAllocation).mockRejectedValue(new Error("overlap"));
    renderWithProviders(<PlanPage />);
    await userEvent.click(await screen.findByRole("button", { name: "Preview placement" }));
    expect(await screen.findByRole("status")).toHaveTextContent("could not be previewed");
  });

  it("does not hide a stale proposal during acceptance", async () => {
    vi.mocked(fetchTodayAllocations).mockResolvedValue({ plannedMinutes: 0, availableMinutes: 480, breathingRoomMinutes: 480, allocations: [] } as never);
    vi.mocked(fetchWorkItems).mockResolvedValue([{ id: "work-1", title: "Draft", remainingMinutes: 45 }] as never);
    vi.mocked(previewAllocation).mockResolvedValue({ id: "proposal-1", baseRevision: 4n, state: "feasible", startMinutes: 540, durationMinutes: 45, reason: "Requested time is available." } as never);
    vi.mocked(applyAllocationProposal).mockRejectedValue(new Error("stale"));
    renderWithProviders(<PlanPage />);
    await userEvent.click(await screen.findByRole("button", { name: "Preview placement" }));
    await userEvent.click(await screen.findByRole("button", { name: "Accept 09:00" }));
    expect(await screen.findByRole("status")).toHaveTextContent("stale or could not be accepted");
    expect(screen.queryByRole("button", { name: "Accept 09:00" })).not.toBeInTheDocument();
  });

  it("reviews and applies a bounded multi-item backlog proposal", async () => {
    vi.mocked(fetchTodayAllocations).mockResolvedValue({ plannedMinutes: 0, availableMinutes: 480, breathingRoomMinutes: 480, allocations: [] } as never);
    vi.mocked(fetchWorkItems).mockResolvedValue([{ id: "work-1", title: "Draft", remainingMinutes: 45 }, { id: "work-2", title: "Review", remainingMinutes: 30 }] as never);
    vi.mocked(previewSchedule).mockResolvedValue({ id: "schedule-1", state: "feasible", baseRevision: 2n, reason: "Placed all 2 selected work items.", placements: [{ workItemId: "work-1", title: "Draft", state: "feasible", startMinutes: 540, durationMinutes: 45, reason: "First feasible" }, { workItemId: "work-2", title: "Review", state: "feasible", startMinutes: 585, durationMinutes: 30, reason: "First feasible" }] } as never);
    vi.mocked(applyScheduleProposal).mockResolvedValue([{ id: "a-1" }, { id: "a-2" }] as never);
    renderWithProviders(<PlanPage />);
    await userEvent.click(await screen.findByRole("button", { name: "Preview next 3" }));
    expect(previewSchedule).toHaveBeenCalledWith(expect.objectContaining({ workItemIds: ["work-1", "work-2"] }));
    expect(await screen.findByText("Next sessions in request order")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Accept feasible sessions" }));
    expect(applyScheduleProposal).toHaveBeenCalledWith({ proposalId: "schedule-1", expectedRevision: 2n, idempotencyKey: "schedule-1:apply" });
    expect(await screen.findByRole("status")).toHaveTextContent("Accepted 2 sessions");
  });

  it("keeps a failed backlog preview explicit", async () => {
    vi.mocked(fetchTodayAllocations).mockResolvedValue({ plannedMinutes: 0, availableMinutes: 480, breathingRoomMinutes: 480, allocations: [] } as never);
    vi.mocked(fetchWorkItems).mockResolvedValue([{ id: "work-1", title: "Draft", remainingMinutes: 45 }, { id: "work-2", title: "Review", remainingMinutes: 30 }] as never);
    vi.mocked(previewSchedule).mockRejectedValue(new Error("offline"));
    renderWithProviders(<PlanPage />);
    await userEvent.click(await screen.findByRole("button", { name: "Preview next 3" }));
    expect(await screen.findByRole("status")).toHaveTextContent("could not be previewed");
  });

  it("keeps a failed backlog apply explicit", async () => {
    vi.mocked(fetchTodayAllocations).mockResolvedValue({ plannedMinutes: 0, availableMinutes: 480, breathingRoomMinutes: 480, allocations: [] } as never);
    vi.mocked(fetchWorkItems).mockResolvedValue([{ id: "work-1", title: "Draft", remainingMinutes: 45 }, { id: "work-2", title: "Review", remainingMinutes: 30 }] as never);
    vi.mocked(previewSchedule).mockResolvedValue({ id: "schedule-1", state: "feasible", baseRevision: 2n, reason: "Placed all 2 selected work items.", placements: [{ workItemId: "work-1", title: "Draft", state: "feasible", startMinutes: 540, durationMinutes: 45, reason: "First feasible" }] } as never);
    vi.mocked(applyScheduleProposal).mockRejectedValue(new Error("stale"));
    renderWithProviders(<PlanPage />);
    await userEvent.click(await screen.findByRole("button", { name: "Preview next 3" }));
    await userEvent.click(await screen.findByRole("button", { name: "Accept feasible sessions" }));
    expect(await screen.findByRole("status")).toHaveTextContent("stale or could not be accepted");
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
    await userEvent.click(screen.getByRole("tab", { name: "Month" }));
    expect(await screen.findByLabelText("Accepted month")).toBeInTheDocument();
    expect(screen.getByText("Review the brief")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("tab", { name: "Timeline" }));
    expect(await screen.findByLabelText("Accepted timeline")).toBeInTheDocument();
    expect(screen.getByText("Review the brief")).toBeInTheDocument();
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
    expect(await screen.findByLabelText("Routine start")).toBeInTheDocument();
    expect(screen.getByLabelText("Routine minutes")).toBeInTheDocument();
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
