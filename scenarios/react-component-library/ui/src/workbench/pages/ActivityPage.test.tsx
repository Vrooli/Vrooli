import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  listActivity,
  getActivitySummary,
  type ActivityEvent,
  type GetActivitySummaryResponse,
} from "../../api/activity";
import { strings } from "../../consts/strings";
import { renderWithProviders } from "../../test-utils/renderWithProviders";
import { ActivityPage } from "./ActivityPage";

vi.mock("../../api/activity", () => ({
  listActivity: vi.fn(),
  getActivitySummary: vi.fn(),
}));
vi.mock("../data", () => ({
  formatDay: (date: Date) => date.toISOString().slice(0, 10),
  formatNumber: (value: number) => String(value),
  formatTime: (date: Date) =>
    Number.isNaN(date.getTime()) ? "invalid time" : date.toISOString().slice(11, 16),
}));

function event(overrides: Partial<ActivityEvent> = {}): ActivityEvent {
  return {
    $typeName: "vrooli.react_component_library.v1.activity.ActivityEvent",
    id: "event-1",
    actor: "mira",
    action: "workspace.saved",
    subjectKind: "workspace",
    subjectId: "ws-harbor",
    summary: "Saved Harbor",
    at: "2026-11-21T12:30:00Z",
    beforeJson: "",
    afterJson: "",
    ...overrides,
  } as ActivityEvent;
}

function summary(overrides: Partial<GetActivitySummaryResponse> = {}): GetActivitySummaryResponse {
  return {
    $typeName: "vrooli.react_component_library.v1.activity.GetActivitySummaryResponse",
    total: 2,
    byActor: [{ actor: "mira", count: 2 }],
    byAction: [{ action: "workspace.saved", count: 2 }],
    ...overrides,
  } as GetActivitySummaryResponse;
}

const entries = [
  event({ id: "save-1", actor: "mira", subjectId: "ws-harbor", summary: "Saved the landing page" }),
  event({
    id: "delete-1",
    actor: "jon",
    action: "workspace.deleted",
    subjectId: "ws-old",
    summary: "Removed an old draft",
    at: "2026-11-20T09:15:00Z",
    beforeJson: '{"name":"Old draft"}',
  }),
  event({
    id: "unknown-date",
    actor: "ops",
    action: "capture.failed",
    subjectId: "ws-capture",
    summary: "Capture could not be imported",
    at: "invalid-date",
    afterJson: '{"reason":"missing L0"}',
  }),
];

describe("ActivityPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(listActivity).mockResolvedValue(entries);
    vi.mocked(getActivitySummary).mockResolvedValue(summary());
  });

  it("loads the append-only log and summary, groups events, and expands the recorded payload", async () => {
    renderWithProviders(<ActivityPage />);

    expect(await screen.findByText(/Saved the landing page/)).toBeInTheDocument();
    expect(listActivity).toHaveBeenCalledWith({ limit: 200 });
    expect(getActivitySummary).toHaveBeenCalledWith(7);
    expect(document.querySelectorAll(".wb-day")[0]).toHaveTextContent("2026-11-21 · 1");
    expect(document.querySelectorAll(".wb-day")[1]).toHaveTextContent("2026-11-20 · 1");
    expect(document.querySelectorAll(".wb-day")[2]).toHaveTextContent("Unknown day · 1");
    expect(document.querySelector(".wb-event small")).toHaveTextContent(
      "Saved the landing page · mira",
    );
    expect(document.querySelectorAll("[data-rcl-timeline-item]")).toHaveLength(3);
    expect(document.querySelectorAll("details .wb-row-between")).toHaveLength(2);
    expect(document.querySelectorAll(".wb-mono")).toHaveLength(1);

    fireEvent.click(screen.getAllByText(strings.workbench.activity.payload)[0]!);
    expect(document.querySelector(".wb-event pre")).toHaveTextContent('{"name":"Old draft"}');
  });

  it("filters by action and searches actor, subject, action, and summary", async () => {
    renderWithProviders(<ActivityPage />);
    await screen.findByText(/Saved the landing page/);

    fireEvent.change(screen.getByRole("combobox", { name: strings.workbench.activity.filter }), {
      target: { value: "workspace.deleted" },
    });
    expect(screen.getByText(/Removed an old draft/)).toBeInTheDocument();
    expect(screen.queryByText(/Saved the landing page/)).not.toBeInTheDocument();

    fireEvent.change(screen.getByRole("combobox", { name: strings.workbench.activity.filter }), {
      target: { value: "all" },
    });
    fireEvent.change(screen.getByRole("textbox", { name: strings.workbench.activity.search }), {
      target: { value: "WS-CAPTURE" },
    });
    expect(screen.getByText(/Capture could not be imported/)).toBeInTheDocument();
    expect(screen.queryByText(/Saved the landing page/)).not.toBeInTheDocument();
  });

  it("distinguishes a measured empty log from an unavailable activity service", async () => {
    vi.mocked(listActivity).mockResolvedValue([]);
    vi.mocked(getActivitySummary).mockResolvedValue(
      summary({ total: 0, byActor: [], byAction: [] }),
    );
    renderWithProviders(<ActivityPage />);

    expect(await screen.findByText(strings.workbench.activity.emptyTitle)).toBeInTheDocument();
    expect(screen.getByText(strings.workbench.activity.emptyBody)).toBeInTheDocument();
    expect(screen.getByText(strings.workbench.activity.noActors)).toBeInTheDocument();
    expect(document.querySelector("[data-rcl-timeline-state]")).toHaveTextContent("No events yet.");
    expect(screen.queryByText(strings.workbench.activity.emptyUnavailable)).not.toBeInTheDocument();
  });

  it("reports the log unavailable on failure rather than presenting a false empty state", async () => {
    vi.mocked(listActivity).mockRejectedValue(new Error("activity service offline"));
    vi.mocked(getActivitySummary).mockRejectedValue(new Error("summary service offline"));
    renderWithProviders(<ActivityPage embedded />);

    expect(await screen.findByText(strings.workbench.activity.errorTitle)).toBeInTheDocument();
    expect(screen.getByText(strings.workbench.activity.emptyUnavailable)).toBeInTheDocument();
    expect(screen.getByText(strings.workbench.activity.summaryUnavailable)).toBeInTheDocument();
    expect(screen.getByTestId("activity-page")).toHaveClass("is-embedded");
  });

  it("keeps the skeleton visible until the event request completes", async () => {
    let resolveEvents: ((items: ActivityEvent[]) => void) | undefined;
    vi.mocked(listActivity).mockReturnValue(
      new Promise((resolve) => {
        resolveEvents = resolve;
      }),
    );
    renderWithProviders(<ActivityPage embedded />);

    expect(screen.getByTestId("activity-page")).toHaveClass("is-embedded");
    expect(document.querySelector(".wb-skeleton")).toBeInTheDocument();
    await waitFor(() => expect(listActivity).toHaveBeenCalledOnce());
    act(() => resolveEvents?.([]));
    expect(await screen.findByText(strings.workbench.activity.emptyTitle)).toBeInTheDocument();
  });
});
