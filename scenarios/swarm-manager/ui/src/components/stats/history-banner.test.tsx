import { describe, it, expect } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders as renderWithCanonicalProviders } from "../../test-utils/renderWithProviders";
import { HistoryBanner } from "./history-banner";

describe("HistoryBanner", () => {
  it("renders when history is shorter than 30 days", () => {
    renderWithCanonicalProviders(
      <HistoryBanner
        history={{
          earliest_event_at: "2026-04-18T00:00:00Z",
          history_days: 5,
          has_history: true,
          min_sample_meaningful: 5,
        }}
      />,
    );
    expect(screen.getByText(/Event history covers 5 days/)).toBeInTheDocument();
  });

  it("renders nothing when history is ≥ 30 days", () => {
    const { container } = renderWithCanonicalProviders(
      <HistoryBanner
        history={{
          earliest_event_at: "2026-01-01T00:00:00Z",
          history_days: 120,
          has_history: true,
          min_sample_meaningful: 5,
        }}
      />,
    );
    expect(container.firstChild).toBeNull();
  });

  it("renders nothing when has_history is false", () => {
    const { container } = renderWithCanonicalProviders(
      <HistoryBanner
        history={{
          earliest_event_at: "",
          history_days: 0,
          has_history: false,
          min_sample_meaningful: 5,
        }}
      />,
    );
    expect(container.firstChild).toBeNull();
  });
});
