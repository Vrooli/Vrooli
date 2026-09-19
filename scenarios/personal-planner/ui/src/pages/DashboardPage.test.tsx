import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import { WorkItemSchema } from "@vrooli/proto-types/personal-planner/v1/work/work_pb";

import { renderWithProviders } from "../test-utils";
import { DashboardPage } from "./DashboardPage";
import { fetchTodayAllocations } from "../api/calendar";
import { createWorkItem, fetchWorkItems } from "../api/work";
import { fetchCurrentFocus, pauseFocus, startFocus } from "../api/focus";

vi.mock("../api/calendar", () => ({ fetchTodayAllocations: vi.fn() }));
vi.mock("../api/work", () => ({ fetchWorkItems: vi.fn(), createWorkItem: vi.fn() }));
vi.mock("../api/focus", () => ({ fetchCurrentFocus: vi.fn(), startFocus: vi.fn(), pauseFocus: vi.fn() }));

afterEach(() => { cleanup(); vi.clearAllMocks(); });

const emptyPlan = { allocations: [], plannedMinutes: 0, availableMinutes: 480, breathingRoomMinutes: 480 };
const noFocus = null;

describe("Observatory Today", () => {
  it("renders the first real work item from the work domain", async () => {
    vi.mocked(fetchWorkItems).mockResolvedValue([create(WorkItemSchema, {
      title: "Draft the Aquila launch story",
      description: "Explain one useful workflow.",
      remainingMinutes: 45,
      sourceLabel: "Cadence",
    })]);
    vi.mocked(fetchTodayAllocations).mockResolvedValue({ ...emptyPlan, allocations: [
      { id: "a-1", title: "Draft block", sourceLabel: "Cadence", startMinutes: 600, durationMinutes: 45 },
      { id: "a-2", title: "Review block", sourceLabel: "Projects", startMinutes: 720, durationMinutes: 30 },
    ] } as never);
    vi.mocked(fetchCurrentFocus).mockResolvedValue(noFocus);

    renderWithProviders(<DashboardPage />);

    expect(await screen.findByRole("heading", { name: "Draft the Aquila launch story" })).toBeInTheDocument();
    expect(screen.getAllByText("Cadence").length).toBeGreaterThan(0);
    expect(screen.queryByText("Capture your next useful action")).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Day" }));
    await userEvent.click(screen.getByRole("button", { name: "Open draft" }));
    expect(screen.getByRole("dialog", { name: "Work item details" })).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Close draft" }));
  });

  it("keeps the focus control honest and stateful", async () => {
    const user = userEvent.setup();
    vi.mocked(fetchWorkItems).mockResolvedValue([]);
    vi.mocked(fetchTodayAllocations).mockResolvedValue(emptyPlan as never);
    vi.mocked(fetchCurrentFocus).mockResolvedValue(noFocus);
    vi.mocked(startFocus).mockResolvedValue({ state: "running", id: "session-1", revision: 0, title: "Spontaneous focus" } as never);
    renderWithProviders(<DashboardPage />);

    const button = await screen.findByRole("button", { name: "Start focus" });
    await user.click(button);
    expect(screen.getByRole("button", { name: "Pause focus" })).toBeInTheDocument();
    expect(screen.getByText("FOCUS IN PROGRESS")).toBeInTheDocument();
  });

  it("makes the empty state actionable with a persisted capture form", async () => {
    const user = userEvent.setup();
    vi.mocked(fetchWorkItems).mockResolvedValue([]);
    vi.mocked(fetchTodayAllocations).mockResolvedValue(emptyPlan as never);
    vi.mocked(fetchCurrentFocus).mockResolvedValue(noFocus);
    vi.mocked(createWorkItem).mockResolvedValue({ id: "work-1", title: "Write release note" } as never);
    renderWithProviders(<DashboardPage />);

    const captureButtons = await screen.findAllByRole("button", { name: "Capture task" });
    await user.click(captureButtons[0]!);
    expect(screen.getByRole("dialog", { name: "Capture task" })).toBeInTheDocument();
    await user.type(screen.getByLabelText("Task title"), "Write release note");
    await user.type(screen.getByLabelText(/Why it matters/), "Explain the useful workflow");
    await user.click(screen.getByRole("button", { name: "Save task" }));
    expect(createWorkItem).toHaveBeenCalledWith(expect.objectContaining({ title: "Write release note", description: "Explain the useful workflow" }));
  });

  it("shows an unavailable accepted schedule honestly", async () => {
    vi.mocked(fetchWorkItems).mockResolvedValue([]);
    vi.mocked(fetchTodayAllocations).mockRejectedValue(new Error("offline"));
    vi.mocked(fetchCurrentFocus).mockResolvedValue(noFocus);
    renderWithProviders(<DashboardPage />);
    expect(await screen.findByText("Plan data unavailable")).toBeInTheDocument();
  });

  it("keeps a failed task capture honest", async () => {
    const user = userEvent.setup();
    vi.mocked(fetchWorkItems).mockResolvedValue([]);
    vi.mocked(fetchTodayAllocations).mockResolvedValue(emptyPlan as never);
    vi.mocked(fetchCurrentFocus).mockResolvedValue(noFocus);
    vi.mocked(createWorkItem).mockRejectedValue(new Error("offline"));
    renderWithProviders(<DashboardPage />);
    await user.click((await screen.findAllByRole("button", { name: "Capture task" }))[0]!);
    await user.type(screen.getByLabelText("Task title"), "Keep the error visible");
    await user.click(screen.getByRole("button", { name: "Save task" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("That task did not save");
  });

  it("makes imported calendar occupancy visible in Today capacity", async () => {
    vi.mocked(fetchWorkItems).mockResolvedValue([]);
    vi.mocked(fetchTodayAllocations).mockResolvedValue({ ...emptyPlan, externalEventCount: 2, externalBusyMinutes: 90 } as never);
    vi.mocked(fetchCurrentFocus).mockResolvedValue(noFocus);
    renderWithProviders(<DashboardPage />);
    expect(await screen.findByText(/2 read-only calendar events occupy 90 minutes/)).toBeInTheDocument();
  });

  it("pauses the durable current session instead of toggling local state", async () => {
    vi.mocked(fetchWorkItems).mockResolvedValue([]);
    vi.mocked(fetchTodayAllocations).mockResolvedValue(emptyPlan as never);
    vi.mocked(fetchCurrentFocus).mockResolvedValue({ id: "session-1", state: "running", revision: 2, title: "Draft" } as never);
    vi.mocked(pauseFocus).mockResolvedValue({ id: "session-1", state: "paused", revision: 3, title: "Draft" } as never);
    renderWithProviders(<DashboardPage />);
    await userEvent.click(await screen.findByRole("button", { name: "Pause focus" }));
    expect(pauseFocus).toHaveBeenCalledWith(expect.objectContaining({ id: "session-1", revision: 2 }));
  });

  it("reports a failed focus transition without assuming it saved", async () => {
    vi.mocked(fetchWorkItems).mockResolvedValue([]);
    vi.mocked(fetchTodayAllocations).mockResolvedValue(emptyPlan as never);
    vi.mocked(fetchCurrentFocus).mockResolvedValue(noFocus);
    vi.mocked(startFocus).mockRejectedValue(new Error("offline"));
    renderWithProviders(<DashboardPage />);
    await userEvent.click(await screen.findByRole("button", { name: "Start focus" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("That focus transition did not save.");
  });
});
