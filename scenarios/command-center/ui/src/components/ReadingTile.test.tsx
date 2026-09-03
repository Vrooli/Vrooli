import { describe, expect, it } from "vitest";
import type { Reading } from "../lib/api";
import { renderWithProviders, screen } from "../test-utils/renderWithProviders";
import { authoredSample, makeReading } from "../test-utils/readings";
import { ReadingTile } from "./ReadingTile";

const reading = (overrides: Partial<Reading>): Reading => makeReading({
  id: "credit_balances", label: "Credits outstanding", format: "compact", source: { team: "monetization", binding: "scenario:landing-page-business-suite" },
  coverage: "IN-REACH", trust: "UNAVAILABLE", ttlSeconds: 300, owner: "monetization", whatIsNeeded: "an aggregation endpoint", firstObservedMissing: "2026-09-01", gapOpenDays: 0,
  sample: authoredSample(128400, [92000, 128400]), ...overrides,
});

describe("ReadingTile", () => {
  it("draws an in-reach reading hollow with a dashed authored series and what it needs", () => {
    const { container } = renderWithProviders(<ul><ReadingTile reading={reading({})} /></ul>);
    const tile = container.querySelector("[data-reading]");
    expect(tile).toHaveAttribute("data-ink", "hollow");
    expect(tile).toHaveAttribute("data-provenance", "sample");
    expect(container.querySelector("[data-rcl-sample-series]")).toHaveAttribute("data-illustrative", "true");
    expect(screen.getByText("illustrative · needs an aggregation endpoint")).toBeInTheDocument();
  });
  it("draws a cached reading dimmed with its age and no sparkline", () => {
    const observedAt = new Date(Date.now() - 4 * 60_000).toISOString();
    const { container } = renderWithProviders(<ul><ReadingTile reading={reading({ coverage: "NOW", trust: "CACHED", value: 128400, observedAt, sample: null, trustReason: "connection refused" })} /></ul>);
    expect(container.querySelector("[data-reading]")).toHaveAttribute("data-ink", "dimmed");
    expect(container.querySelector("[data-rcl-sample-series]")).toBeNull();
    expect(container.querySelector("[data-rcl-freshness='cached']")).not.toBeNull();
    expect(screen.getByText("last good 4m ago · connection refused")).toBeInTheDocument();
  });
  it("frames a silent sensor instead of showing a zero", () => {
    const { container } = renderWithProviders(<ul><ReadingTile reading={reading({ coverage: "NOW", trust: "UNAVAILABLE", sample: null, trustReason: "deadline exceeded" })} /></ul>);
    expect(container.querySelector("[data-reading]")).toHaveAttribute("data-ink", "unavailable");
    expect(screen.getByText(/not answering · deadline exceeded/)).toBeInTheDocument();
  });
  it("shows a safe origin label without exposing a URL", () => {
    const { container } = renderWithProviders(<ul><ReadingTile reading={reading({ origin: "production", origin_env: "production", origin_display: "Production instance" })} /></ul>);
    expect(container.querySelector("[data-reading]")).toHaveAttribute("data-origin-env", "production");
    expect(screen.getByText("Production instance")).toBeInTheDocument();
    expect(container.textContent).not.toMatch(/https?:\/\/|127\.0\.0\.1|:\d{2,5}/);
  });
});
