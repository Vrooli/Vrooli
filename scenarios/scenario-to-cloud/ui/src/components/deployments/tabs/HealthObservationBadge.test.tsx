import "@testing-library/jest-dom";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import type { HealthObservation } from "../../../lib/api";
import { HealthObservationBadge } from "./HealthObservationBadge";

const base: HealthObservation = {
  deployment_id: "3c1c9a1e-5f3a-4a4d-9b1f-0d9d9f5b2f11",
  target_id: "host:203.0.113.10",
  observed_release_digest: "sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
  observed_configuration_digest: "sha256:cfg",
  observed_at: "2026-09-09T12:00:00Z",
  status: "HEALTH_STATUS_HEALTHY",
  checks: [],
  freshness: "FRESHNESS_CURRENT",
  producer_ref: "scenario-to-cloud:health:v1",
  partial: false,
  missing_dependencies: [],
  next_actions: [],
};

afterEach(cleanup);

describe("HealthObservationBadge", () => {
  it("renders status, freshness, observed_at and release together", () => {
    render(<HealthObservationBadge observation={base} />);
    const badge = screen.getByTestId("health-observation-badge");
    expect(badge).toHaveAttribute("data-status", "healthy");
    expect(badge).toHaveAttribute("data-freshness", "current");
    expect(badge).toHaveTextContent("Healthy");
    expect(badge).toHaveTextContent("current");
    expect(badge).toHaveTextContent("observed 2026-09-09T12:00:00Z");
    expect(badge).toHaveTextContent("release 9f86d081884c");
  });

  it("shows current and unhealthy without contradiction (P16-A06)", () => {
    render(<HealthObservationBadge observation={{ ...base, status: "HEALTH_STATUS_UNHEALTHY" }} />);
    const badge = screen.getByTestId("health-observation-badge");
    expect(badge).toHaveAttribute("data-status", "unhealthy");
    expect(badge).toHaveAttribute("data-freshness", "current");
    expect(badge).toHaveTextContent("Unhealthy");
    expect(badge).toHaveTextContent("current");
  });

  it("keeps stale healthy evidence visibly stale", () => {
    render(<HealthObservationBadge observation={{ ...base, freshness: "FRESHNESS_STALE" }} />);
    const badge = screen.getByTestId("health-observation-badge");
    expect(badge).toHaveAttribute("data-status", "healthy");
    expect(badge).toHaveAttribute("data-freshness", "stale");
    expect(badge).toHaveTextContent("stale");
  });

  it("never renders unknown, unspecified or foreign status values as healthy", () => {
    for (const status of ["HEALTH_STATUS_UNKNOWN", "HEALTH_STATUS_UNSPECIFIED", "HEALTH_STATUS_SUPER_HEALTHY", undefined]) {
      cleanup();
      render(<HealthObservationBadge observation={{ ...base, status: status as HealthObservation["status"], freshness: "FRESHNESS_UNKNOWN" }} />);
      const badge = screen.getByTestId("health-observation-badge");
      expect(badge).toHaveAttribute("data-status", "unknown");
      expect(badge).toHaveAttribute("data-freshness", "unknown");
      expect(badge).toHaveTextContent("Health unknown");
    }
  });

  it("surfaces partial observations and their missing dependencies", () => {
    render(<HealthObservationBadge observation={{ ...base, partial: true, missing_dependencies: ["edge_dns", "edge_tls"] }} />);
    expect(screen.getByTestId("health-observation-badge")).toHaveTextContent("partial: missing edge_dns, edge_tls");
  });

  it("renders nothing without an observation", () => {
    const { container } = render(<HealthObservationBadge observation={null} />);
    expect(container).toBeEmptyDOMElement();
  });
});
