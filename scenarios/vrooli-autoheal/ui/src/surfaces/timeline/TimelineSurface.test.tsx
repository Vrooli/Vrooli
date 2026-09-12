import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { TimelineSurface } from "./TimelineSurface";
import * as api from "../../lib/api";
import { createSystemEvent, createSystemEventsResponse, renderWithProviders } from "../../test-utils";

vi.mock("../../lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../lib/api")>();
  return {
    ...actual,
    fetchSystemEvents: vi.fn(),
    refreshSystemEvents: vi.fn(),
  };
});

describe("TimelineSurface", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.fetchSystemEvents).mockResolvedValue(
      createSystemEventsResponse({
        events: [
          createSystemEvent({
            id: 1,
            category: "kernel",
            title: "Kernel installed",
            summary: "linux-image-6.17.0-23-generic installed",
          }),
          createSystemEvent({
            id: 2,
            fingerprint: "event-2",
            category: "crash",
            severity: "critical",
            source: "journalctl",
            title: "Hardware/reset signal",
            summary: "uncorrected error caused a data fabric sync flood event",
          }),
        ],
        correlations: [{
          title: "Kernel change before crash",
          summary: "A kernel event occurred before the first crash/reset event in this window.",
          rationale: "Temporal proximity only.",
          eventIds: [1, 2],
          eventSources: ["dpkg-log", "journalctl"],
          timeDelta: "16h",
          confidence: "temporal",
        }],
      })
    );
    vi.mocked(api.refreshSystemEvents).mockResolvedValue({ ingested: 1, deduped: 0, sources: [], durationMs: 10 });
  });

  it("renders system events and correlation hints", async () => {
    renderWithProviders(<TimelineSurface />);

    await waitFor(() => {
      expect(screen.getByText("Kernel installed")).toBeInTheDocument();
      expect(screen.getByText("Hardware/reset signal")).toBeInTheDocument();
      expect(screen.getByText("Kernel change before crash")).toBeInTheDocument();
    });
  });

  it("passes category filter to the API", async () => {
    renderWithProviders(<TimelineSurface />);

    await waitFor(() => expect(screen.getByLabelText("Filter category")).toBeInTheDocument());
    fireEvent.change(screen.getByLabelText("Filter category"), { target: { value: "driver" } });

    await waitFor(() => {
      expect(api.fetchSystemEvents).toHaveBeenLastCalledWith(expect.objectContaining({ category: "driver" }));
    });
  });

  it("refreshes system events on demand", async () => {
    renderWithProviders(<TimelineSurface />);

    await waitFor(() => expect(screen.getByText("Kernel installed")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: /refresh/i }));

    await waitFor(() => {
      expect(api.refreshSystemEvents).toHaveBeenCalled();
    });
  });
});

