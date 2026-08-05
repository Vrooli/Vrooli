import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { EvidenceReview } from "./EvidenceReview";
import * as api from "../../lib/api";
import { renderWithProviders } from "../../test-utils/renderWithProviders";

vi.mock("../../lib/api");

const profile: api.DeploymentProfile = {
  id: "profile-1",
  name: "Production",
  scenario: "picker-wheel",
  tiers: [2],
  version: 4,
};

describe("EvidenceReview", () => {
  beforeEach(() => vi.clearAllMocks());

  it("groups a desktop journey, shows its gate reason, and links producer evidence", async () => {
    vi.mocked(api.listProfiles).mockResolvedValue([profile]);
    vi.mocked(api.getEvidenceReview).mockResolvedValue({
      profile_id: profile.id,
      git_commit_hash: "abc123",
      ready: false,
      reason: "linux desktop evidence is degraded",
      verdicts: [{
        run_id: "run-1",
        disposition: "DISPOSITION_FAILED",
        target: { ramp: "scenario-to-desktop", platform: "linux", os: "linux", device_kind: "HOST" },
        refs: [{ producer: "scenario-to-desktop", artifact_id: "journey-1", kind: "journey", checksum: "sha256:x", size_bytes: 12 }],
        detail: JSON.stringify({
          recording_url: "https://producer.example/runs/run-1.mp4",
          journey: {
            degraded_reason: "xdotool_unavailable",
            steps: [{ name: "maximize", action: "window_maximize", disposition: "failed", degraded_reason: "xdotool_unavailable" }],
          },
        }),
      }],
    });

    renderWithProviders(<EvidenceReview />);
    await screen.findByRole("option", { name: profile.name });
    fireEvent.change(screen.getByRole("combobox"), { target: { value: profile.id } });
    fireEvent.change(screen.getByLabelText("Exact commit"), { target: { value: "abc123" } });
    fireEvent.click(screen.getByRole("button", { name: "Review" }));

    expect(await screen.findByText("scenario-to-desktop/linux")).toBeInTheDocument();
    expect(screen.getByText("Gate: blocked")).toBeInTheDocument();
    expect(screen.getAllByText("xdotool_unavailable")).toHaveLength(2);
    expect(screen.getByText("maximize")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Open producer recording/i })).toHaveAttribute("href", "https://producer.example/runs/run-1.mp4");
    await waitFor(() => expect(api.getEvidenceReview).toHaveBeenCalledWith(profile.id, "abc123"));
  });

  it("renders a safe error when the review service is unavailable", async () => {
    vi.mocked(api.listProfiles).mockResolvedValue([profile]);
    vi.mocked(api.getEvidenceReview).mockRejectedValue(new Error("review unavailable"));
    renderWithProviders(<EvidenceReview />);
    await screen.findByRole("option", { name: profile.name });
    fireEvent.change(screen.getByRole("combobox"), { target: { value: profile.id } });
    fireEvent.change(screen.getByLabelText("Exact commit"), { target: { value: "abc123" } });
    fireEvent.click(screen.getByRole("button", { name: "Review" }));
    expect(await screen.findByText("review unavailable")).toBeInTheDocument();
  });

  it("renders passed, pending, unknown, malformed, and unsafe producer details safely", async () => {
    vi.mocked(api.listProfiles).mockResolvedValue([profile]);
    vi.mocked(api.getEvidenceReview).mockResolvedValue({
      profile_id: profile.id,
      git_commit_hash: "abc123",
      ready: true,
      reason: "",
      verdicts: [
        { run_id: "pass", disposition: "DISPOSITION_PASSED", refs: [], detail: JSON.stringify({ recording_url: "javascript:alert(1)" }) },
        { run_id: "pending", disposition: "DISPOSITION_PENDING", target: { ramp: "desktop", platform: "windows", os: "windows", device_kind: "HOST" }, refs: [], detail: JSON.stringify({ journey: { steps: [{ error: "broken" }] } }) },
        { run_id: "unknown", disposition: "DISPOSITION_OTHER", target: { ramp: "", platform: "", os: "", device_kind: "" }, refs: [], detail: "not json" },
      ],
    });
    renderWithProviders(<EvidenceReview />);
    await screen.findByRole("option", { name: profile.name });
    fireEvent.change(screen.getByRole("combobox"), { target: { value: profile.id } });
    fireEvent.change(screen.getByLabelText("Exact commit"), { target: { value: "abc123" } });
    fireEvent.click(screen.getByRole("button", { name: "Review" }));
    expect(await screen.findByText("Gate: ready")).toBeInTheDocument();
    expect(screen.getByText("desktop/windows")).toBeInTheDocument();
    expect(screen.getAllByText("pending").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("unknown/unknown")).toBeInTheDocument();
    expect(screen.getByText("broken")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /Open producer recording/i })).not.toBeInTheDocument();
  });
});
