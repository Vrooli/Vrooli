import { describe, expect, it } from "vitest";
import { renderWithProviders, screen } from "../test-utils/renderWithProviders";
import { makeReading } from "../test-utils/readings";
import { PostureReadout } from "./PostureReadout";

const postureReading = (trust: "CACHED" | "UNAVAILABLE") => makeReading({
  id: "landing_page_default_alive", label: "Default alive", kind: "posture", coverage: "NOW", trust,
  value: { revenueMinor: 10_000, burnMinor: 5_000, cashMinor: 20_000, gap: "15 days", source: "ledger" } as unknown as number,
  observedAt: new Date(Date.now() - 3 * 60_000).toISOString(), trustReason: "producer observation is stale", sample: null,
});

describe("PostureReadout trust", () => {
  it("formats ledger amounts in the source book currency", () => {
    const reading = makeReading({
      id: "offer_posture", label: "Default-alive posture", kind: "posture", coverage: "NOW", trust: "VALID",
      value: { revenueMinor: 10_000, burnMinor: 5_000, cashMinor: 20_000, currency: "EUR" } as unknown as number,
      observedAt: new Date().toISOString(), sample: null,
    });
    renderWithProviders(<PostureReadout reading={reading} />);
    expect(screen.getByText("€100")).toBeInTheDocument();
  });

  it("labels cached amounts, reason, and age while keeping their dimmed treatment", () => {
    const { container } = renderWithProviders(<PostureReadout reading={postureReading("CACHED")} />);
    expect(container.querySelector("[data-reading]" )).toHaveAttribute("data-ink", "dimmed");
    expect(container.querySelector(".cc-posture-magnitudes")).not.toBeNull();
    expect(screen.getByText("CACHED", { selector: "strong" })).toBeInTheDocument();
    expect(screen.getByText(/last good 3m ago · producer observation is stale/)).toBeInTheDocument();
  });

  it("does not present expired amounts as current and labels their last-good age", () => {
    const { container } = renderWithProviders(<PostureReadout reading={postureReading("UNAVAILABLE")} />);
    expect(container.querySelector("[data-reading]" )).toHaveAttribute("data-ink", "unavailable");
    expect(screen.getAllByText("—")).toHaveLength(3);
    expect(screen.getByText("UNAVAILABLE", { selector: "strong" })).toBeInTheDocument();
    const trustLine = screen.getByRole("group", { name: /Default alive: UNAVAILABLE.*producer observation is stale · last good 3m ago/ });
    expect(trustLine.querySelector("[data-qualifier]")).toHaveTextContent("source not answering · last good 3m ago");
    expect(trustLine.querySelector("[data-qualifier]")).toHaveAttribute("title", expect.stringContaining("producer observation is stale"));
  });
});
