import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import {
  QueryPartEvidenceSchema,
  QueryPartEvidence_Status,
  QueryResponseSchema,
} from "@vrooli/proto-types/search-hub/v1/routing/routing_pb";

import { renderWithProviders } from "../../test-utils";
import * as searchApi from "../../api/search";
import { ScenarioEvidencePanel } from "./ScenarioEvidencePanel";

vi.mock("../../api/search", () => ({ runQuery: vi.fn() }));

describe("ScenarioEvidencePanel", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("joins scoped cached maturity and requirement evidence while keeping execution proof separate", async () => {
    vi.mocked(searchApi.runQuery).mockResolvedValue(create(QueryResponseSchema, {
      groups: [
        { providerId: "scenario-completeness-scoring.requirements", hits: [{ id: "score:search-hub", title: "search-hub cached maturity", snippet: "Cached maturity rung", metadata: { scenario: "search-hub", source: "ScoreService.GetScore", basis: "cached_artifacts", calculated_at: "2026-11-24T10:00:00Z", tree_digest: "abc123", freshness: { currentDigest: "td:new", phases: [{ phase: "unit", verdict: "stale", lastDigest: "td:old" }] } } }] },
        { providerId: "test-genie.requirements", hits: [{ id: "requirements:search-hub", title: "search-hub requirement status", snippet: "Recorded requirement summary", metadata: { scenario: "search-hub", status: "complete", live_status: "unknown", source: { owner: "Test Genie", generatedAt: "2026-11-24T09:00:00Z", path: "requirements/search-hub.json", currentRunVerified: false } } }] },
      ],
    }));

    renderWithProviders(<ScenarioEvidencePanel />);
    fireEvent.change(screen.getByLabelText("Scenario name"), { target: { value: "search-hub" } });
    fireEvent.click(screen.getByRole("button", { name: "Load evidence" }));

    await waitFor(() => expect(searchApi.runQuery).toHaveBeenCalledWith(expect.objectContaining({
      types: ["requirements"],
      scope: "scenario:search-hub",
      all: false,
    })));
    expect(await screen.findByText("2026-11-24T10:00:00Z")).toBeInTheDocument();
    expect(screen.getByText("abc123")).toBeInTheDocument();
    expect(screen.getAllByText("MATCH: search-hub")).toHaveLength(2);
    expect(screen.getByText(/unit: stale \(recorded digest td:old\)/)).toBeInTheDocument();
    expect(screen.getByText("2026-11-24T09:00:00Z")).toBeInTheDocument();
    expect(screen.getByText(/this requirement snapshot does not verify a current run/)).toBeInTheDocument();
  });

  it("keeps missing and degraded owners explicitly unknown", async () => {
    vi.mocked(searchApi.runQuery).mockResolvedValue(create(QueryResponseSchema, {
      groups: [{
        providerId: "test-genie.requirements",
        note: "owner timeout",
        degradations: [{ leg: "semantic", reason: "timeout", detail: "semantic leg timed out" }],
      }],
      degraded: true,
      degradations: [{ component: "provider", providerId: "test-genie.requirements", reason: "timeout", detail: "provider endpoint unavailable" }],
    }));
    renderWithProviders(<ScenarioEvidencePanel />);
    fireEvent.change(screen.getByLabelText("Scenario name"), { target: { value: "search-hub" } });
    fireEvent.click(screen.getByRole("button", { name: "Load evidence" }));
    expect(await screen.findByText(/Provider degraded: owner timeout/)).toBeInTheDocument();
    expect(screen.getByText("semantic: semantic leg timed out")).toBeInTheDocument();
    expect(screen.getByText(/provider: test-genie.requirements: timeout: provider endpoint unavailable/)).toBeInTheDocument();
    expect(screen.getByText(/UNKNOWN: no response from this owner/)).toBeInTheDocument();
  });

  it("refuses to join mismatched or unidentified owner rows and keeps snapshot time separate from freshness", async () => {
    vi.mocked(searchApi.runQuery).mockResolvedValue(create(QueryResponseSchema, {
      groups: [
        { providerId: "scenario-completeness-scoring.requirements", hits: [{ id: "score:other", title: "other cached score", metadata: { scenario: "other-scenario", maturity: "R3", tree_digest: "old" } }] },
        { providerId: "test-genie.requirements", hits: [{ id: "requirements:unknown", title: "unidentified recorded requirements", metadata: { status: "complete", source: { owner: "Test Genie", generatedAt: "2026-11-24T08:00:00Z" } } }] },
      ],
    }));
    renderWithProviders(<ScenarioEvidencePanel />);
    fireEvent.change(screen.getByLabelText("Scenario name"), { target: { value: "search-hub" } });
    fireEvent.click(screen.getByRole("button", { name: "Load evidence" }));
    expect(await screen.findByText(/MISMATCH: other-scenario; requested search-hub cannot be joined/)).toBeInTheDocument();
    expect(screen.getByText(/UNKNOWN; owner result does not carry a comparable scenario identity/)).toBeInTheDocument();
    expect(screen.getByText(/freshness against the current scenario tree is UNKNOWN/)).toBeInTheDocument();
  });

  it("uses one four-source budget across both owner cards", async () => {
    const hits = (providerId: string, prefix: string) => Array.from({ length: 3 }, (_, index) => ({
      providerId,
      id: `${prefix}-${index + 1}`,
      title: `${prefix} result ${index + 1}`,
      path: `scenarios/search-hub/${prefix}-${index + 1}.json`,
      metadata: { scenario: "search-hub" },
    }));
    vi.mocked(searchApi.runQuery).mockResolvedValue(create(QueryResponseSchema, {
      groups: [
        { providerId: "scenario-completeness-scoring.requirements", count: 3, hits: hits("scenario-completeness-scoring.requirements", "maturity") },
        { providerId: "test-genie.requirements", count: 3, hits: hits("test-genie.requirements", "requirements") },
      ],
    }));
    renderWithProviders(<ScenarioEvidencePanel />);
    fireEvent.change(screen.getByLabelText("Scenario name"), { target: { value: "search-hub" } });
    fireEvent.click(screen.getByRole("button", { name: "Load evidence" }));

    expect(await screen.findByText("maturity result 1")).toBeInTheDocument();
    expect(screen.getByText("maturity result 2")).toBeInTheDocument();
    expect(screen.getByText("maturity result 3")).toBeInTheDocument();
    expect(screen.getByText("requirements result 1")).toBeInTheDocument();
    expect(screen.queryByText("requirements result 2")).not.toBeInTheDocument();
    expect(screen.queryByText("requirements result 3")).not.toBeInTheDocument();
    expect(screen.getByText(/2 additional sources omitted/)).toBeInTheDocument();
    expect(searchApi.runQuery).toHaveBeenCalledTimes(1);
  });

  it("deduplicates the same source across owner cards and names the visible owner", async () => {
    vi.mocked(searchApi.runQuery).mockResolvedValue(create(QueryResponseSchema, {
      groups: [
        { providerId: "scenario-completeness-scoring.requirements", hits: [{ id: "maturity-hit", title: "Shared source record", path: "scenarios/search-hub/evidence.json#L1", metadata: { scenario: "search-hub" } }] },
        { providerId: "test-genie.requirements", hits: [{ id: "requirements-hit", title: "Shared source record from Test Genie", path: "scenarios/search-hub/evidence.json#L20", metadata: { scenario: "search-hub" } }] },
      ],
    }));
    renderWithProviders(<ScenarioEvidencePanel />);
    fireEvent.change(screen.getByLabelText("Scenario name"), { target: { value: "search-hub" } });
    fireEvent.click(screen.getByRole("button", { name: "Load evidence" }));

    expect(await screen.findByText("Shared source record")).toBeInTheDocument();
    expect(screen.queryByText("Shared source record from Test Genie")).not.toBeInTheDocument();
    expect(screen.getByText("Duplicate source is shown under Cached maturity.")).toBeInTheDocument();
  });

  it("keeps missing and over-budget request parts visible with their hit disposition", async () => {
    const hits = Array.from({ length: 5 }, (_, index) => ({
      providerId: "scenario-completeness-scoring.requirements",
      id: `score-${index + 1}`,
      title: `Score evidence ${index + 1}`,
      path: `scenarios/search-hub/score-${index + 1}.json`,
    }));
    vi.mocked(searchApi.runQuery).mockResolvedValue(create(QueryResponseSchema, {
      groups: [{ providerId: "scenario-completeness-scoring.requirements", count: 5, hits }],
      queryParts: [
        create(QueryPartEvidenceSchema, {
          partId: "missing",
          label: "Find the current requirement owner",
          status: QueryPartEvidence_Status.NOT_ROUTED,
          reason: "ambiguous_provider_evidence",
        }),
        create(QueryPartEvidenceSchema, {
          partId: "score",
          label: "Read cached maturity basis",
          status: QueryPartEvidence_Status.CANDIDATES_RETURNED,
          searchedProviderIds: ["scenario-completeness-scoring.requirements"],
          evidence: [{ providerId: "scenario-completeness-scoring.requirements", groupIndex: 0, hitIndex: 4, hitId: "score-5" }],
        }),
      ],
    }));
    renderWithProviders(<ScenarioEvidencePanel />);
    fireEvent.change(screen.getByLabelText("Scenario name"), { target: { value: "search-hub" } });
    fireEvent.click(screen.getByRole("button", { name: "Load evidence" }));

    const parts = await screen.findAllByTestId("scenario-evidence-query-part");
    expect(parts[0]).toHaveTextContent(/Find the current requirement owner.*Not routed/);
    expect(parts[0]).toHaveTextContent(/Possible owners could not be distinguished/);
    expect(parts[1]).toHaveTextContent(/Read cached maturity basis.*Candidates returned/);
    expect(screen.getByText(/supporting hit\(s\) omitted by shared budget/)).toBeInTheDocument();
    expect(screen.queryByText("Score evidence 5")).not.toBeInTheDocument();
  });

  it("counts distinct task evidence and unresolved references in owner coverage", async () => {
    const hits = Array.from({ length: 5 }, (_, index) => ({
      providerId: "scenario-completeness-scoring.requirements",
      id: `score-${index + 1}`,
      title: `Score evidence ${index + 1}`,
      path: `scenarios/search-hub/score-${index + 1}.json`,
      metadata: { scenario: "search-hub" },
    }));
    const reference = (hitIndex: number, hitId: string) => ({
      providerId: "scenario-completeness-scoring.requirements", groupIndex: 0, hitIndex, hitId,
    });
    const duplicateShown = reference(0, "score-1");
    const duplicateOmitted = reference(4, "score-5");
    const duplicateUnresolved = reference(5, "stale-hit");
    vi.mocked(searchApi.runQuery).mockResolvedValue(create(QueryResponseSchema, {
      groups: [{ providerId: "scenario-completeness-scoring.requirements", count: 5, hits }],
      queryParts: [create(QueryPartEvidenceSchema, {
        partId: "score", label: "Read cached maturity basis", status: QueryPartEvidence_Status.CANDIDATES_RETURNED,
        evidence: [duplicateShown, duplicateShown, duplicateOmitted, duplicateOmitted, duplicateUnresolved, duplicateUnresolved],
      })],
    }));
    renderWithProviders(<ScenarioEvidencePanel />);
    fireEvent.change(screen.getByLabelText("Scenario name"), { target: { value: "search-hub" } });
    fireEvent.click(screen.getByRole("button", { name: "Load evidence" }));

    const part = await screen.findByTestId("scenario-evidence-query-part");
    expect(part).toHaveTextContent(/1 supporting hit\(s\) shown/);
    expect(part).toHaveTextContent(/1 supporting hit\(s\) omitted by shared budget/);
    expect(part).toHaveTextContent(/1 hit reference\(s\) could not be resolved/);
  });
});
