import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import DeploymentsSection from "./DeploymentsSection";
import { confirmStagingRecovery, getStagingRecovery, getStagingState, listStagingOperations, previewStagingRecovery, requestStagingStart, resumeStagingOperation } from "../../api/deployments";

vi.mock("../../api/deployments", () => ({ getStagingState: vi.fn(), listStagingOperations: vi.fn(), requestStagingStart: vi.fn(), resumeStagingOperation: vi.fn(), getStagingRecovery: vi.fn(), previewStagingRecovery: vi.fn(), confirmStagingRecovery: vi.fn() }));

function renderSection() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={client}><DeploymentsSection open /></QueryClientProvider>);
}

describe("DeploymentsSection", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(getStagingState).mockResolvedValue({
      profile: { id: "web-console/staging", scenario: "web-console", environment: "staging", domain: "staging.vrooli.com", provider: "hetzner", offer: "CX23", regions: ["NBG1", "FSN1"], cap_currency: "USD", cap_minor_units: 709, approval: "A2 approved", health_path: "/health" },
      readiness: { state: "blocked", reasons: ["Compute Manager quote and spend admission are not attached", "Compute Manager instance reference is not attached"] },
      deployment_manager_profile_id: "web-console/staging",
    });
    vi.mocked(listStagingOperations).mockResolvedValue([{ operation_id: "release-op-1", release_id: "1", profile_id: "web-console/staging", status: "RELEASE_OPERATION_STATUS_AMBIGUOUS", resume_available: false, resume_reason: "cloud step beyond safe boundary", active_stage: "cloud_handoff", updated_at: "2026-11-24T12:00:00Z", cloud_spend_quote_idempotency_key: "fixture-quote-key", cloud_compute_manager_instance_id: "fixture-cm-instance", tls: { state: "healthy", sampled_at: "2026-11-24T12:00:00Z", reason: "certificate_valid" }, certificate: { state: "valid", not_after: "2030-01-01T00:00:00Z", sampled_at: "2026-11-24T12:00:00Z", renewal: "blocked", renewal_reason: "renewal_state_not_reported_by_tls_snapshot" }, cloud_step: { state: "recovery_decision_required", reason: "fixture-only verification failed", deployment_id: "fixture-deployment", owner_operation_id: "s2c-fixture-operation", request_key: "fixture-request-key", request_digest: "sha256:fixture", verification_receipt_json: "{\"healthy\":false}" } }]);
    vi.mocked(getStagingRecovery).mockResolvedValue({ recovery: { state: "awaiting_preview", operation_id: "release-op-1", deployment_id: "fixture-deployment", current_release_digest: "sha256:current", current_bundle_sha256: "sha256:bundle", last_good_release_digest: "sha256:last-good", review_ref: "review-fixture", data_compatibility: "compatible" } });
    vi.mocked(previewStagingRecovery).mockResolvedValue({ recovery: { state: "preview_ready", operation_id: "release-op-1", deployment_id: "fixture-deployment", current_release_digest: "sha256:current", current_bundle_sha256: "sha256:bundle", last_good_release_digest: "sha256:last-good", last_good_bundle_sha256: "sha256:last-good-bundle", review_ref: "review-fixture", data_compatibility: "compatible", preview_ref: "preview-fixture", route_kind: "bundle_pipeline", impact_summary: "Activates last-good release." } });
    vi.mocked(confirmStagingRecovery).mockResolvedValue({ recovery: { state: "pending", operation_id: "release-op-1", deployment_id: "fixture-deployment", current_release_digest: "sha256:current", current_bundle_sha256: "sha256:bundle", last_good_release_digest: "sha256:last-good", review_ref: "review-fixture", data_compatibility: "compatible", preview_ref: "preview-fixture", route_kind: "bundle_pipeline", recovery_operation_id: "s2c-recovery-1", recovery_status: "running" } });
    vi.mocked(requestStagingStart).mockResolvedValue({ state: "blocked", reasons: ["quote and candidate gates are missing"], deployment_manager_called: false });
    vi.mocked(resumeStagingOperation).mockResolvedValue({ operation: { operation_id: "release-op-1", release_id: "1", profile_id: "web-console/staging", status: "RELEASE_OPERATION_STATUS_RUNNING", resume_available: false, tls: { state: "unavailable", reason: "no persisted verification sample" }, certificate: { state: "unavailable", renewal: "blocked", reason: "no persisted verification sample" } } });
  });

  it("shows the fixed A2 profile and blocks while quote and instance are absent", async () => {
    renderSection();
    expect(await screen.findByText("staging.vrooli.com")).toBeInTheDocument();
    expect(screen.getByTestId("staging-blocked")).toHaveTextContent("quote and spend admission are not attached");
    expect(screen.getByTestId("staging-blocked")).toHaveTextContent("instance reference is not attached");
  });

  it("renders durable DM operation progress and blocked hand-off read-back", async () => {
    renderSection();
    expect(await screen.findByTestId("staging-operation")).toHaveTextContent("cloud_handoff");
    expect(screen.getByTestId("staging-operation")).toHaveTextContent("fixture-quote-key");
    expect(screen.getByTestId("staging-operation")).toHaveTextContent("fixture-cm-instance");
    expect(screen.getByTestId("staging-operation")).toHaveTextContent("fixture-only verification failed");
    expect(screen.getByTestId("staging-operation")).toHaveTextContent("fixture-deployment");
    expect(screen.getByTestId("staging-operation")).toHaveTextContent("fixture-request-key");
    expect(screen.getByTestId("staging-operation")).toHaveTextContent("sha256:fixture");
    expect(screen.getByTestId("staging-tls-status")).toHaveTextContent("TLS health: healthy");
    expect(screen.getByTestId("staging-tls-status")).toHaveTextContent("certificate_valid");
    expect(screen.getByTestId("staging-certificate-status")).toHaveTextContent("Certificate: valid");
    expect(screen.getByTestId("staging-certificate-status").querySelector("time")?.getAttribute("datetime")).toBe("2030-01-01T00:00:00Z");
    expect(screen.getByTestId("staging-certificate-status")).toHaveTextContent("Renewal: blocked");
    expect(screen.getByTestId("staging-certificate-status")).toHaveTextContent("renewal_state_not_reported_by_tls_snapshot");
    expect(screen.getByTestId("staging-recovery-required")).toHaveTextContent("Recovery decision required");
    expect(screen.getByTestId("staging-recovery-preview-action")).toBeInTheDocument();
    expect(screen.getByTestId("staging-verification-receipt")).toHaveTextContent("healthy");
    expect(screen.queryByTestId("staging-resume")).not.toBeInTheDocument();
  });

  it("requires compatibility and a second explicit confirmation after canonical preview", async () => {
    renderSection();
    await screen.findByTestId("staging-recovery-preview-action");
    const previewButton = screen.getByTestId("staging-recovery-preview");
    expect(previewButton).toBeDisabled();
    fireEvent.click(screen.getByRole("checkbox", { name: /data written by the current release/i }));
    fireEvent.click(previewButton);
    expect(await screen.findByTestId("staging-recovery-preview-details")).toHaveTextContent("Activates last-good release.");
    expect(screen.getByTestId("staging-recovery-preview-details")).toHaveTextContent("sha256:last-good-bundle");
    const confirmButton = screen.getByTestId("staging-recovery-confirm");
    expect(confirmButton).toBeDisabled();
    fireEvent.click(screen.getByRole("checkbox", { name: /authorize this rollback/i }));
    fireEvent.click(confirmButton);
    await waitFor(() => expect(confirmStagingRecovery).toHaveBeenCalledWith("release-op-1", "preview-fixture"));
  });

  it("refreshes standing without creating a second operation", async () => {
    renderSection();
    await screen.findByTestId("staging-operation");
    fireEvent.click(screen.getByRole("button", { name: "Refresh" }));
    await waitFor(() => expect(listStagingOperations).toHaveBeenCalledTimes(2));
  });

  it("shows a blocked start decision instead of claiming a release", async () => {
    renderSection();
    fireEvent.click(await screen.findByTestId("staging-start-check"));
    expect(await screen.findByTestId("staging-start-blocked")).toHaveTextContent("quote and candidate gates are missing");
    expect(requestStagingStart).toHaveBeenCalledTimes(1);
  });

  it("does not offer Resume when durable DM read-back is not eligible", async () => {
    renderSection();
    await screen.findByTestId("staging-operation");
    expect(screen.queryByTestId("staging-resume")).not.toBeInTheDocument();
  });

  it("requests Resume only when DM read-back marks the operation eligible", async () => {
    vi.mocked(listStagingOperations).mockResolvedValue([{ operation_id: "release-op-1", release_id: "1", profile_id: "web-console/staging", status: "RELEASE_OPERATION_STATUS_FAILED", resume_available: true, active_stage: "cloud_handoff", tls: { state: "unavailable", reason: "no persisted verification sample" }, certificate: { state: "unavailable", renewal: "blocked", reason: "no persisted verification sample" } }]);
    renderSection();
    fireEvent.click(await screen.findByTestId("staging-resume"));
    await waitFor(() => expect(resumeStagingOperation).toHaveBeenCalledWith("release-op-1"));
  });
});
