import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../test-utils";
import { SourceDistributionPanel } from "./SourceDistributionPanel";

const candidate = { distribution_id: "distribution-1", scenario: "demo", source_digest: "sha256:source", closure_digest: "sha256:closure", recipe_digest: "sha256:recipe", policy_digest: "sha256:policy", artifact_id: "artifact-1", artifact_digest: "sha256:artifact", verification_status: "passed", verification_receipt: "receipt-1", deployment_manager_decision: "decision-1", publication_status: "awaiting_human", destination: "operator/repo", destination_revision: "", readback_receipt: "", drift_state: "up_to_date", source_of_truth: "scenario-to-repository", source_timestamp: "2026-01-01T00:00:00Z", freshness: "current", updated_at: "2026-01-01T00:00:00Z", workflow_url: "" };

afterEach(() => vi.unstubAllGlobals());

describe("SourceDistributionPanel", () => {
  it("renders the exact tuple, safe exclusions, handoff, and drift separately", async () => {
    const detailPayload = { available: true, distribution: candidate, contents: [{ path: "api/main.go", source_path: "api/main.go", category: "application", digest: "sha256:file", size_bytes: 12 }], exclusions: [{ path: ".env", category: "private material", safe_reason: "privacy policy" }], unresolved_obligations: [], runtime_requirements: [], handoff: { status: "awaiting_human", artifact_id: "artifact-1", artifact_digest: "sha256:artifact", destination: "operator/repo", human_action: "Publish manually", readback_oracle: "exact read-back", preconditions: [], approval_reference: "decision-1", source_of_truth: "scenario-to-repository", freshness: "current" }, drift: { state: "destination_edits_detected", source_changed: false, destination_changed: true, current_source_digest: "sha256:source", recorded_source_digest: "sha256:source", current_artifact_digest: "sha256:artifact", recorded_artifact_digest: "sha256:artifact", actions: ["review independent edits"], source_of_truth: "scenario-to-repository", freshness: "current" } };
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const isDetail = String(input).includes("/source-distributions/");
      return { ok: true, status: 200, json: async () => isDetail ? detailPayload : { available: true, distributions: [candidate] } } as Response;
    });
    vi.stubGlobal("fetch", fetchMock);
    renderWithProviders(<SourceDistributionPanel onClose={() => undefined} />);
    await waitFor(() => expect(screen.getByText("demo")).toBeInTheDocument());
    fireEvent.click(screen.getByText("demo"));
    await waitFor(() => expect(screen.getByText("sha256:closure")).toBeInTheDocument());
    expect(screen.getByText(/private material/)).toBeInTheDocument();
    expect(screen.getByText(/destination edits detected/)).toBeInTheDocument();
    expect(screen.getByText("Approval is not publication.")).toBeInTheDocument();
  });

  it("explains an unavailable source-ramp without inventing records", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ available: false, unavailable_reason: "owner offline", distributions: [] }), { status: 503, headers: { "content-type": "application/json" } })));
    renderWithProviders(<SourceDistributionPanel onClose={() => undefined} />);
    await waitFor(() => expect(screen.getByText(/Source-ramp unavailable/)).toBeInTheDocument());
  });
});
