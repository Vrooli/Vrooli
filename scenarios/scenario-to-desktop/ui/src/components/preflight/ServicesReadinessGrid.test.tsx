import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ServicesReadinessGrid } from "./ServicesReadinessGrid";
import { renderWithProviders } from "../../test-utils/renderWithProviders";

describe("ServicesReadinessGrid", () => {
  it("shows ready service links, timestamps, and configured health endpoint", () => {
    renderWithProviders(<ServicesReadinessGrid readinessDetails={[["api", { ready: true, started_at: "2026-07-27T12:00:00Z", ready_at: "2026-07-27T12:00:05Z", updated_at: "2026-07-27T12:00:10Z", exit_code: 0 }]]} ports={{ api: { api: 19925 } }} bundleManifest={{ services: [{ id: "api", health: { type: "http", path: "/health", port_name: "api" } }] }} snapshotTs={Date.parse("2026-07-27T12:01:00Z")} tick={Date.parse("2026-07-27T12:01:00Z")} />);
    expect(screen.getByText("Ready")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open" })).toHaveAttribute("href", "http://localhost:19925");
    expect(screen.getByRole("link", { name: "Health" })).toHaveAttribute("href", "http://localhost:19925/health");
    expect(screen.getByText("api:19925")).toBeInTheDocument();
    expect(screen.getByText("Exit code 0")).toBeInTheDocument();
    expect(screen.getByText(/Ready for/)).toBeInTheDocument();
  });

  it("distinguishes skipped and pending dependencies from a configured health probe", () => {
    renderWithProviders(<ServicesReadinessGrid readinessDetails={[["cache", { ready: false, skipped: true, message: "not needed" }], ["worker", { ready: false, message: "pending start", updated_at: "2026-07-27T12:00:00Z" }]]} snapshotTs={Date.parse("2026-07-27T12:00:20Z")} tick={Date.parse("2026-07-27T12:00:20Z")} />);
    expect(screen.getByText("Skipped")).toBeInTheDocument();
    expect(screen.getByText("Waiting")).toBeInTheDocument();
    expect(screen.getByText("Not launched yet; waiting on dependencies or secrets.")).toBeInTheDocument();
    expect(screen.getAllByText("Health check not configured for http proxy inspection.")).toHaveLength(2);
  });
});
