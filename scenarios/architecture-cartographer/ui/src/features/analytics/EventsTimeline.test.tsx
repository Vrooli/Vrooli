import { cleanup, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { EventKind } from "@vrooli/proto-types/architecture-cartographer/v1/analytics/analytics_pb";

vi.mock("./controllers/useAnalyticsController", () => ({
  useEvents: vi.fn(),
}));

import { selectors } from "../../consts/selectors";
import { renderWithProviders } from "../../test-utils";
import { useEvents } from "./controllers/useAnalyticsController";
import { EventsTimeline } from "./EventsTimeline";

afterEach(() => {
  cleanup();
  vi.mocked(useEvents).mockReset();
});

function mockEvents(state: Partial<ReturnType<typeof useEvents>>) {
  vi.mocked(useEvents).mockReturnValue({
    isPending: false,
    isError: false,
    data: { events: [] },
    error: null,
    refetch: vi.fn(),
    ...state,
  } as unknown as ReturnType<typeof useEvents>);
}

describe("EventsTimeline", () => {
  it("renders loading, error, and empty states", () => {
    mockEvents({ isPending: true });
    const { rerender } = renderWithProviders(<EventsTimeline scenario="demo" />);
    expect(screen.getByTestId(selectors.features.analytics.events.loading)).toBeInTheDocument();

    mockEvents({ isError: true, error: new Error("events unavailable") });
    rerender(<EventsTimeline scenario="demo" />);
    expect(screen.getByTestId(selectors.features.analytics.events.error)).toHaveTextContent(
      "events unavailable",
    );

    mockEvents({ data: { events: [] } as never });
    rerender(<EventsTimeline scenario="demo" />);
    expect(screen.getByTestId(selectors.features.analytics.events.empty)).toBeInTheDocument();
  });

  it("renders every event kind label branch in the table", () => {
    mockEvents({
      data: {
        events: [
          EventKind.CONFLICT_DETECTED,
          EventKind.CONFLICT_ASSIGNED,
          EventKind.CONFLICT_RESOLVED,
          EventKind.CONFLICT_REOPENED,
          EventKind.CONFLICT_FORCE_RESOLVED,
          EventKind.VERDICT_PRODUCED,
          EventKind.PLACEMENT_AUTO,
          EventKind.PLACEMENT_SUGGEST,
          EventKind.OVERRIDE_RECORDED,
          EventKind.APPLY_PLANNED,
          EventKind.APPLY_RAN,
          EventKind.APPLY_BUILD_GREEN,
          EventKind.APPLY_BUILD_RED,
          EventKind.APPLY_REVERTED,
          EventKind.UNSPECIFIED,
        ].map((kind, index) => ({
          id: `event-${index}`,
          kind,
          domain: index % 2 === 0 ? "graph" : "",
          conflictId: index % 3 === 0 ? "afid:abc" : "",
          actor: index % 4 === 0 ? "agent" : "",
        })),
      } as never,
    });

    renderWithProviders(<EventsTimeline scenario="demo" />);

    expect(screen.getByTestId(selectors.features.analytics.events.root)).toBeInTheDocument();
    expect(screen.getAllByRole("row")).toHaveLength(16);
    expect(screen.getByTestId(selectors.shared.dataTable.row({ id: "event-0" }))).toHaveTextContent("graph");
    expect(screen.getByTestId(selectors.shared.dataTable.row({ id: "event-1" }))).toHaveTextContent("—");
  });
});
