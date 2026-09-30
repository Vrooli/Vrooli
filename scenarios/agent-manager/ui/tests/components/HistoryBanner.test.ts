import { screen } from "@testing-library/react";
import { createElement } from "react";
import { describe, it, expect } from "vitest";
import { HistoryBanner } from "../../src/components/stats/HistoryBanner.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

describe("HistoryBanner", () => {
  it("renders when history is shorter than 30 days", () => {
    renderWithProviders(
      createElement(HistoryBanner, {
        history: {
          earliest_event_at: "2026-04-30T00:00:00Z",
          history_days: 5,
          has_history: true,
          min_sample_meaningful: 5,
        },
      }),
    );
    expect(screen.getByText(/Event history covers 5 days/)).toBeTruthy();
  });

  it("renders nothing when history has reached 30+ days", () => {
    const { container } = renderWithProviders(
      createElement(HistoryBanner, {
        history: {
          earliest_event_at: "2026-01-01T00:00:00Z",
          history_days: 120,
          has_history: true,
          min_sample_meaningful: 5,
        },
      }),
    );
    expect(container.firstChild).toBeNull();
  });

  it("renders nothing when has_history is false", () => {
    const { container } = renderWithProviders(
      createElement(HistoryBanner, {
        history: {
          earliest_event_at: "",
          history_days: 0,
          has_history: false,
          min_sample_meaningful: 5,
        },
      }),
    );
    expect(container.firstChild).toBeNull();
  });
});
