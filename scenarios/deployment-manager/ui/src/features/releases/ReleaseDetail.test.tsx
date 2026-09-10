import { screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { ReleaseDetail } from "./ReleaseDetail";
import * as api from "../../lib/api";

vi.mock("../../lib/api");

describe("ReleaseDetail", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.history.pushState({}, "", "/releases/release-1");
  });

  it("shows one reviewable workspace for identity, execution, and receipts", async () => {
    vi.mocked(api.getRelease).mockResolvedValue({
      id: "release-1", profile_id: "p1", deployment_id: "cloud-deployment-1", git_commit_hash: "commit-1", artifact_digest: "artifact-manifest-1",
      candidate_id: "candidate-1", destination_revision_id: "destination-1", readiness_review_key: "review-1",
      authorization_epoch: 4, release_version: "1.2.0", channel: "stable", status: "published",
      created_at: "2026-09-08T12:00:00Z", updated_at: "2026-09-08T12:01:00Z",
      candidate: { candidate_id: "candidate-1", source_revision: "commit-1", profile_revision: "profile-1", dependency_lock_digest: "lock-1", policy_digest: "policy-1", artifacts: [{ target: { id: "linux-x64", platform: "linux", os: "linux", architecture: "amd64", format: "appimage" }, immutable_ref: "lpbs://linux-x64/1", digest: "sha512:linux", size_bytes: 10, signature_digest: "sha256:sig", signer_ref: "release-authority" }] },
      destination_revision: { destination_revision_id: "destination-1", kind: "lpbs", destination_id: "staging", configuration_digest: "config-1", channel: "stable" },
      review_binding: { review_id: "review-1", candidate_id: "candidate-1", destination_revision_id: "destination-1", targets: ["linux-x64"], channel: "stable", evidence_set_digest: "evidence-1", policy_digest: "policy-1", authorization_epoch: 4 },
      publication_receipts: [{ candidate_id: "candidate-1", destination_revision_id: "destination-1", target_id: "linux-x64", artifact_digest: "sha512:linux", destination_object: "lpbs://staging/stable/linux-x64/1", producer: "lpbs", external_receipt: "upload-1", outcome: "verified", observed_at: "2026-09-08T12:01:00Z" }],
      recovery_receipts: [{ release_id: "release-1", candidate_id: "candidate-1", destination_revision_id: "destination-1", deployment_id: "cloud-deployment-1", action: "halt", outcome: "halted", health: "stopped", external_receipt: "recovery-1", observed_at: "2026-09-08T12:02:00Z", dry_run: false }],
      platforms: [{ release_id: "release-1", platform: "linux-x64", status: "published" }],
    });
    vi.mocked(api.getReleaseDossier).mockResolvedValue({
      schema_version: 1, generated_at: "2026-09-08T12:03:00Z",
      release: {} as api.Release,
      health: { release_id: "release-1", status: "healthy", observed_at: "2026-09-08T12:03:00Z", publication_verified: true, client_updates_healthy: true, alerts: [], known_durations_millis: {}, supported_controls: ["halt"], unsupported_controls: ["cohort_rollout"] },
      missing_proof: [],
    });
    vi.mocked(api.getReleaseOperation).mockResolvedValue({
      operation_id: "operation-1", release_id: "release-1", profile_id: "p1", idempotency_key: "release-1-stable",
      status: "publishing", active_stage: "publish_lpbs", created_at: "2026-09-08T12:00:00Z", updated_at: "2026-09-08T12:01:00Z",
    });

    renderWithProviders(<MemoryRouter initialEntries={["/releases/release-1?operation_id=operation-1"]}><Routes><Route path="/releases/:id" element={<ReleaseDetail />} /></Routes></MemoryRouter>, { withoutRouter: true });
    expect(await screen.findByTestId("release-detail")).toBeInTheDocument();
    expect(screen.getByText("Release identity")).toBeInTheDocument();
    expect(screen.getByText("Candidate evidence")).toBeInTheDocument();
    expect(screen.getByText("Current release standing")).toBeInTheDocument();
    expect(screen.getByText("Health: healthy")).toBeInTheDocument();
    expect(screen.getByText("External effect receipts")).toBeInTheDocument();
    expect(screen.getByText("Release operation")).toBeInTheDocument();
    expect(screen.getByText("publishing")).toBeInTheDocument();
    expect(screen.getByText("publish_lpbs")).toBeInTheDocument();
    expect(screen.getAllByText("cloud-deployment-1").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("lpbs://staging/stable/linux-x64/1")).toBeInTheDocument();
    expect(screen.getByText("halt · owner recovery")).toBeInTheDocument();
    expect(screen.getByText("release-authority")).toBeInTheDocument();
    expect(api.getRelease).toHaveBeenCalledWith("release-1");
    expect(api.getReleaseOperation).toHaveBeenCalledWith("operation-1");
  });
});
