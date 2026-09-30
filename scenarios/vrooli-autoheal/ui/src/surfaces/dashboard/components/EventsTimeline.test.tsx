import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { EventsTimeline } from "./EventsTimeline";
import * as api from "../../../lib/api";
import { createTimelineEvent, createTimelineResponse, renderWithProviders } from "../../../test-utils";

vi.mock("../../../shared/contexts/CheckMetadataContext", async () => {
  const { useMockCheckMetadata } = await import("../../../test-utils/mocks/checkMetadataContext");
  return { useCheckMetadata: useMockCheckMetadata };
});

vi.mock("../../../lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../../lib/api")>();
  return { ...actual, fetchTimeline: vi.fn() };
});

describe("EventsTimeline", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.fetchTimeline).mockResolvedValue(createTimelineResponse({
      events: [
        createTimelineEvent({ checkId: "healthy", status: "ok", message: "All good" }),
        createTimelineEvent({ checkId: "broken", status: "critical", message: "Service down" }),
      ],
    }));
  });

  it("filters events and displays issue counts", async () => {
    renderWithProviders(<EventsTimeline />);
    expect(await screen.findByText("All good")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "All" }));
    expect(screen.getByRole("button", { name: /Issues \(1\)/i })).toBeInTheDocument();
    expect(screen.queryByText("All good")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Issues/i }));
    expect(screen.getByText("All good")).toBeInTheDocument();
  });

  it("shows more events and handles failures", async () => {
    const events = Array.from({ length: 21 }, (_, index) => createTimelineEvent({
      checkId: `check-${index}`,
      message: `Event ${index}`,
    }));
    vi.mocked(api.fetchTimeline).mockResolvedValueOnce(createTimelineResponse({ events }));
    renderWithProviders(<EventsTimeline />);
    expect(await screen.findByRole("button", { name: /show more/i })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /show more/i }));
    expect(screen.getByText("Event 20")).toBeInTheDocument();

    vi.mocked(api.fetchTimeline).mockRejectedValue(new Error("timeline unavailable"));
    renderWithProviders(<EventsTimeline />);
    await waitFor(() => expect(screen.getAllByText(/retry/i).length).toBeGreaterThan(0), { timeout: 5000 });
    for (const retry of screen.getAllByText(/retry/i)) fireEvent.click(retry);
  });
});
