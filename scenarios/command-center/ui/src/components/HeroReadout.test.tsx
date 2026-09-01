import { describe, expect, it } from "vitest";
import type { Reading } from "../lib/api";
import { renderWithProviders, screen } from "../test-utils/renderWithProviders";
import { authoredSample, makeReading } from "../test-utils/readings";
import { HeroReadout } from "./HeroReadout";

const reading = (overrides: Partial<Reading>): Reading => makeReading({
  id: "revenue_mrr", label: "Monthly recurring revenue", unit: "usd", format: "currency.compact", source: { team: "monetization", binding: "scenario:landing-page-business-suite" },
  coverage: "MISSING", trust: "UNAVAILABLE", ttlSeconds: 300, owner: "monetization", whatIsNeeded: "a revenue surface", firstObservedMissing: "2026-09-01", gapOpenDays: 3,
  sample: authoredSample(12400, [8100, 12400]), ...overrides,
});

describe("HeroReadout", () => {
  it("renders an absent metric as a dotted figure with its owner and days open", () => {
    renderWithProviders(<HeroReadout reading={reading({})} />);
    expect(screen.getByLabelText("$12.4k").closest("[data-figure]")).toHaveAttribute("data-ink", "dotted");
    expect(screen.getByText(/no substrate · monetization · open 3 days/)).toBeInTheDocument();
  });
  it("renders a measured metric solid with its source and freshness", () => {
    renderWithProviders(<HeroReadout reading={reading({ coverage: "NOW", trust: "VALID", value: 58, observedAt: new Date().toISOString(), sample: null, format: "integer", unit: "count" })} />);
    expect(screen.getByLabelText("58").closest("[data-figure]")).toHaveAttribute("data-ink", "solid");
    expect(screen.getByTestId("freshness-hairline")).toBeInTheDocument();
  });
  it("says so when a room has nothing measured and illustrations are hidden", () => {
    renderWithProviders(<HeroReadout reading={null} emptyReason="Illustrative figures are hidden." />);
    expect(screen.getByText("No measured reading")).toBeInTheDocument();
    expect(screen.getByText("Illustrative figures are hidden.")).toBeInTheDocument();
  });
});
