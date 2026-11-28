import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { HealthCard } from "./HealthCard";

type LibraryHealthCardProps = {
  data?: { timestamp: string };
  onRefresh: () => Promise<void>;
  formatTimestamp: (timestamp: string) => string;
  refreshCount: number;
};

vi.mock("@vrooli/react-component-library/HealthCard/0.1.6", () => ({
  HealthCard: ({ data, onRefresh, formatTimestamp, refreshCount }: LibraryHealthCardProps) => (
    <section>
      <span>{data ? formatTimestamp(data.timestamp) : "loading"}</span>
      <span data-testid="refresh-count">{refreshCount}</span>
      <button type="button" onClick={() => void onRefresh()}>
        refresh
      </button>
    </section>
  ),
}));

describe("HealthCard", () => {
  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
  });

  it("formats health timestamps and tracks refreshes", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({
          status: "healthy",
          service: "scenario-to-android",
          timestamp: "2026-09-09T23:00:00Z",
          readiness: true,
        }), { status: 200 }),
      ),
    );

    renderWithProviders(<HealthCard />);

    await screen.findByText(/2026/);
    expect(screen.getByTestId("refresh-count")).toHaveTextContent("0");

    fireEvent.click(screen.getByRole("button", { name: "refresh" }));
    await waitFor(() => expect(screen.getByTestId("refresh-count")).toHaveTextContent("1"));
  });
});
