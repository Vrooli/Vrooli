import { fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { assessChangeIntent } from "../../api/changeIntent";
import { renderWithProviders } from "../../test-utils";
import { ChangeIntentPanel } from "./ChangeIntentPanel";

vi.mock("../../api/changeIntent", () => ({ assessChangeIntent: vi.fn() }));

describe("ChangeIntentPanel", () => {
  beforeEach(() => vi.clearAllMocks());

  it("renders exact anchors and says support is not verified implementation", async () => {
    vi.mocked(assessChangeIntent).mockResolvedValue({
      scenario: "business-health", intent: "Change BH-SRCH-001", scope: "one scenario, checked-in source only",
      results: [{ scenario: "business-health", requirement_id: "BH-SRCH-001", title: "Intent search corpus", description: "One record per requirement", declared_status: "planned", relation: "support", relation_basis: "explicit_requirement_id_reference_in_statement", lexical_overlap: 0, requirement_source: { path: "requirements/05-search/module.json", line: 4, sha256: "req-sha", kind: "requirement_declaration", attestation: "SOURCE_ONLY" }, implementation_anchors: [{ path: "api/internal/aisearch/recall_test.go", line: 58, sha256: "src-sha", kind: "test_reference", attestation: "REFERENCE_ONLY" }] }],
      limitations: ["support does not mean implementation is verified"], degradations: [],
    });
    renderWithProviders(<ChangeIntentPanel />);
    fireEvent.change(screen.getByLabelText("Change-intent statement · business-health"), { target: { value: "Change BH-SRCH-001" } });
    fireEvent.click(screen.getByRole("button", { name: "Trace intent" }));
    expect(await screen.findByText(/BH-SRCH-001 · Intent search corpus/)).toBeInTheDocument();
    expect(screen.getByText(/Relation: SUPPORT/)).toBeInTheDocument();
    expect(screen.getByText(/Scenario: business-health/)).toBeInTheDocument();
    expect(screen.getByText(/recall_test.go:58.*test_reference.*REFERENCE_ONLY.*SHA-256 src-sha/)).toBeInTheDocument();
    expect(screen.getByText("Limit: support does not mean implementation is verified")).toBeInTheDocument();
    expect(assessChangeIntent).toHaveBeenCalledWith("Change BH-SRCH-001", 10);
  });

  it("preserves the no-match and bounded-degradation state", async () => {
    vi.mocked(assessChangeIntent).mockResolvedValue({ scenario: "business-health", intent: "new unrelated idea", scope: "bounded", results: [], limitations: [], degradations: ["source_scan_bounded"] });
    renderWithProviders(<ChangeIntentPanel />);
    fireEvent.change(screen.getByLabelText("Change-intent statement · business-health"), { target: { value: "new unrelated idea" } });
    fireEvent.click(screen.getByRole("button", { name: "Trace intent" }));
    expect(await screen.findByText(/No requirement candidate matched/)).toBeInTheDocument();
    expect(screen.getByText("Partial scan: source_scan_bounded")).toBeInTheDocument();
  });

  it("does not promote a lexical candidate above unknown", async () => {
    vi.mocked(assessChangeIntent).mockResolvedValue({
      scenario: "business-health", intent: "intent corpus", scope: "bounded", limitations: [], degradations: [],
      results: [{ scenario: "business-health", requirement_id: "BH-SRCH-001", title: "Intent search corpus", description: "candidate", declared_status: "planned", relation: "unknown", relation_basis: "lexical_candidate_only", lexical_overlap: 2, requirement_source: { path: "requirements/05-search/module.json", line: 4, sha256: "req-sha", kind: "requirement_declaration", attestation: "SOURCE_ONLY" }, implementation_anchors: [] }],
    });
    renderWithProviders(<ChangeIntentPanel />);
    fireEvent.change(screen.getByLabelText("Change-intent statement · business-health"), { target: { value: "intent corpus" } });
    fireEvent.click(screen.getByRole("button", { name: "Trace intent" }));
    expect(await screen.findByText(/Relation: UNKNOWN/)).toBeInTheDocument();
    expect(screen.getByText("No tagged source reference was found in the bounded scan.")).toBeInTheDocument();
  });
});
