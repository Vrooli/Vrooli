import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { matchInvariantCandidateReceipt, searchInvariantCandidates } from "../../api/invariantCandidates";
import { renderWithProviders } from "../../test-utils";
import { InvariantCandidatePanel } from "./InvariantCandidatePanel";

vi.mock("../../api/invariantCandidates", () => ({ searchInvariantCandidates: vi.fn(), matchInvariantCandidateReceipt: vi.fn() }));

describe("InvariantCandidatePanel", () => {
  beforeEach(() => vi.clearAllMocks());

  it("checks a caller-selected receipt against exact candidate identity without claiming enforcement", async () => {
    vi.mocked(searchInvariantCandidates).mockResolvedValue({
      $typeName: "vrooli.code_facts.v1.facts.SearchInvariantCandidatesResponse",
      query: "cache bound", scenario: "search-hub", scope: "api/cache",
      results: [{
        $typeName: "vrooli.code_facts.v1.facts.InvariantCandidate",
        id: "code-facts:invariant-candidate:search-hub:runtime_guard:api/cache/store.go",
        title: "Cache capacity guard", text: "Candidate guard", scenario: "search-hub", path: "api/cache/store.go", line: 42,
        sha256: "a".repeat(64), evidenceKind: "runtime_guard", score: 0.9, attestation: "CANDIDATE_ONLY",
        enforcement: { $typeName: "vrooli.code_facts.v1.facts.InvariantEnforcementLayers", type: "UNKNOWN", database: "UNKNOWN", test: "UNKNOWN", runtime: "UNKNOWN" },
      }], degradations: [],
    });
    vi.mocked(matchInvariantCandidateReceipt).mockResolvedValue({
      status: "EXACT_MATCH", receiptId: "receipt-1", candidate: {
        candidateId: "code-facts:invariant-candidate:search-hub:runtime_guard:api/cache/store.go", scenario: "search-hub", relativePath: "api/cache/store.go", sourceLine: 42, sourceFileSha256: "a".repeat(64),
      }, reason: "exact identity match",
    });

    renderWithProviders(<InvariantCandidatePanel />);
    fireEvent.change(screen.getByLabelText("Scenario slug"), { target: { value: "search-hub" } });
    fireEvent.change(screen.getByLabelText("Relative source path"), { target: { value: "api/cache" } });
    fireEvent.change(screen.getByLabelText("Candidate question"), { target: { value: "cache bound" } });
    fireEvent.click(screen.getByRole("button", { name: "Search candidates" }));
    expect(await screen.findByText("Cache capacity guard")).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText(/Test Genie receipt ID/), { target: { value: "receipt-1" } });
    fireEvent.click(screen.getByRole("button", { name: "Check exact receipt" }));
    expect(await screen.findByText(/EXACT_MATCH: this candidate's source identity is bound to a successful receipt/)).toBeInTheDocument();
    expect(screen.getByText(/does not prove invariant enforcement or a requirement\/test-tag match/)).toBeInTheDocument();
    await waitFor(() => expect(matchInvariantCandidateReceipt).toHaveBeenCalledWith("receipt-1", {
      candidateId: "code-facts:invariant-candidate:search-hub:runtime_guard:api/cache/store.go", scenario: "search-hub", relativePath: "api/cache/store.go", sourceLine: 42, sourceFileSha256: "a".repeat(64),
    }));
  });

  it("shows bounded candidate provenance and keeps every enforcement layer unknown", async () => {
    vi.mocked(searchInvariantCandidates).mockResolvedValue({
      $typeName: "vrooli.code_facts.v1.facts.SearchInvariantCandidatesResponse",
      query: "cache bound",
      scenario: "search-hub",
      scope: "api/cache",
      results: [{
        $typeName: "vrooli.code_facts.v1.facts.InvariantCandidate",
        id: "code-facts:invariant-candidate:search-hub:type_shape:api/cache/store.go",
        title: "Cache capacity pattern",
        text: "Bounded cache declaration candidate",
        scenario: "search-hub",
        path: "api/cache/store.go",
        line: 42,
        sha256: "file-sha-256",
        evidenceKind: "type_shape",
        score: 0.8,
        attestation: "CANDIDATE_ONLY",
        enforcement: { $typeName: "vrooli.code_facts.v1.facts.InvariantEnforcementLayers", type: "UNKNOWN", database: "UNKNOWN", test: "UNKNOWN", runtime: "UNKNOWN" },
      }],
      degradations: [],
    });

    renderWithProviders(<InvariantCandidatePanel />);
    fireEvent.change(screen.getByLabelText("Scenario slug"), { target: { value: "search-hub" } });
    fireEvent.change(screen.getByLabelText("Relative source path"), { target: { value: "api/cache" } });
    fireEvent.change(screen.getByLabelText("Candidate question"), { target: { value: "cache bound" } });
    fireEvent.click(screen.getByRole("button", { name: "Search candidates" }));

    expect(await screen.findByText("Cache capacity pattern")).toBeInTheDocument();
    expect(screen.getByText("Candidate mechanism: type_shape · CANDIDATE_ONLY")).toBeInTheDocument();
    expect(screen.getByText("Candidate identity: record code-facts:invariant-candidate:search-hub:type_shape:api/cache/store.go · source observation search-hub/api/cache/store.go:42 · file SHA-256 file-sha-256")).toBeInTheDocument();
    expect(screen.getByText(/Test Genie can check one receipt you provide against the candidate's exact source identity/)).toBeInTheDocument();
    expect(screen.getAllByText("UNKNOWN")).toHaveLength(4);
    expect(screen.getByText(/Candidate location is not proof that an invariant is actively enforced or verified/)).toBeInTheDocument();
    expect(screen.getByText(/test-shaped source does not prove a passing verification/)).toBeInTheDocument();
    await waitFor(() => expect(searchInvariantCandidates).toHaveBeenCalledWith("cache bound", "search-hub", "api/cache", 10));
  });

  it.each([
    ["no_candidate_match", "No invariant pattern matched."],
    ["source_scan_bounded", "Source scan hit the configured file limit."],
  ])("preserves the empty/degraded state for %s", async (code, message) => {
    vi.mocked(searchInvariantCandidates).mockResolvedValue({
      $typeName: "vrooli.code_facts.v1.facts.SearchInvariantCandidatesResponse",
      query: "missing rule",
      scenario: "search-hub",
      scope: "api",
      results: [],
      degradations: [{ $typeName: "vrooli.code_facts.v1.facts.InvariantCandidateDegradation", code, message }],
    });

    renderWithProviders(<InvariantCandidatePanel />);
    fireEvent.click(screen.getByRole("button", { name: "Search candidates" }));

    expect(await screen.findByText(/No candidate pattern matched in this bounded source scan/)).toBeInTheDocument();
    expect(screen.getByText(`Partial source evidence · ${code}: ${message}`)).toBeInTheDocument();
    expect(screen.getByText(/This does not establish that an invariant is absent/)).toBeInTheDocument();
    expect(screen.getByText(/Test Genie can check one receipt you provide against the candidate's exact source identity/)).toBeInTheDocument();
  });
});
