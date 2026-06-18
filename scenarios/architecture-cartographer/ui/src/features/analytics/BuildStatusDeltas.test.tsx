import { cleanup, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { EventKind } from "@vrooli/proto-types/architecture-cartographer/v1/analytics/analytics_pb";

vi.mock("./controllers/useAnalyticsController", () => ({
  useEvents: vi.fn(),
}));

import { selectors } from "../../consts/selectors";
import { renderWithProviders } from "../../test-utils";
import { BuildStatusDeltas } from "./BuildStatusDeltas";
import { useEvents } from "./controllers/useAnalyticsController";

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

describe("BuildStatusDeltas", () => {
  it("renders nothing while analytics events are unavailable", () => {
    mockEvents({ isPending: true });
    const { container, rerender } = renderWithProviders(<BuildStatusDeltas scenario="demo" />);
    expect(container).toBeEmptyDOMElement();

    mockEvents({ isError: true });
    rerender(<BuildStatusDeltas scenario="demo" />);
    expect(container).toBeEmptyDOMElement();
  });

  it("renders empty state when no build events are present", () => {
    mockEvents({
      data: {
        events: [{ id: "event-1", kind: EventKind.APPLY_RAN }],
      } as never,
    });

    renderWithProviders(<BuildStatusDeltas scenario="demo" />);

    expect(screen.getByTestId(selectors.features.analytics.buildDeltas.empty)).toBeInTheDocument();
  });

  it("renders green, red, and reverted build deltas", () => {
    mockEvents({
      data: {
        events: [
          { id: "ignored", kind: EventKind.APPLY_RAN },
          { id: "green-event", runId: "run-green", kind: EventKind.APPLY_BUILD_GREEN },
          { id: "red-event", runId: "run-red", kind: EventKind.APPLY_BUILD_RED },
          { id: "reverted-event", runId: "", kind: EventKind.APPLY_REVERTED },
        ],
      } as never,
    });

    renderWithProviders(<BuildStatusDeltas scenario="demo" />);

    expect(screen.getByTestId(selectors.features.analytics.buildDeltas.root)).toBeInTheDocument();
    expect(screen.getByText("run-green")).toBeInTheDocument();
    expect(screen.getByText("run-red")).toBeInTheDocument();
    expect(screen.getByText("reverted-event")).toBeInTheDocument();
    expect(screen.getByText("green")).toBeInTheDocument();
    expect(screen.getByText("red")).toBeInTheDocument();
    expect(screen.getByText("reverted")).toBeInTheDocument();
  });
});
