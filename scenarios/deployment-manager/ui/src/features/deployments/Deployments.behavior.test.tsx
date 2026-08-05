import { fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { Routes, Route } from "react-router-dom";
import { Deployments } from "./Deployments";
import * as api from "../../lib/api";
import { renderWithProviders } from "../../test-utils/renderWithProviders";

vi.mock("../../lib/api");

describe("Deployments behavior", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.listTelemetry).mockResolvedValue([]);
    vi.mocked(api.listProfiles).mockResolvedValue([]);
  });

  it("explains the workflow and opens the guided deployment dialog", async () => {
    renderWithProviders(<Deployments />);
    expect(await screen.findByText("Deployments")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /How this works/i }));
    expect(screen.getByText("How deployments work")).toBeInTheDocument();
    const [startGuided] = screen.getAllByRole("button", { name: /Start guided flow/i });
    if (startGuided) fireEvent.click(startGuided);
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });

  it("renders a completed deployment status and telemetry details", async () => {
    vi.mocked(api.getDeploymentStatus).mockResolvedValue({ id: "d1", status: "completed", profile_id: "p1", started_at: "2026-08-04T10:00:00Z", completed_at: "2026-08-04T10:01:00Z", artifacts: ["bundle.zip"], message: "ready" });
    vi.mocked(api.listTelemetry).mockResolvedValue([{ scenario: "p1", path: "/tmp/p1", total_events: 2, failure_counts: {}, recent_events: [] }]);
    renderWithProviders(
      <Routes><Route path="/deployments/:id" element={<Deployments />} /></Routes>,
      { route: "/deployments/d1" },
    );
    expect(await screen.findByText("completed")).toBeInTheDocument();
    expect(screen.getByText("1 file(s)")).toBeInTheDocument();
    expect(screen.getAllByText("p1").length).toBeGreaterThanOrEqual(1);
  });

  it("uploads telemetry from a deployment detail view", async () => {
    vi.mocked(api.getDeploymentStatus).mockResolvedValue({ id: "d2", status: "failed", profile_id: "p2", started_at: "2026-08-04T10:00:00Z", completed_at: "2026-08-04T10:01:00Z", artifacts: [], message: "needs attention" });
    vi.mocked(api.uploadTelemetry).mockResolvedValue({ path: "/tmp/p2" });
    renderWithProviders(
      <Routes><Route path="/deployments/:id" element={<Deployments />} /></Routes>,
      { route: "/deployments/d2" },
    );
    expect(await screen.findByText("failed")).toBeInTheDocument();
    const file = new File(["{}"], "deployment-telemetry.jsonl", { type: "application/json" });
    fireEvent.change(screen.getByLabelText("Telemetry file"), { target: { files: [file] } });
    fireEvent.click(screen.getByRole("button", { name: "Upload telemetry" }));
    expect(await screen.findByText(/Uploaded\. Telemetry saved/)).toBeInTheDocument();
  });
});
