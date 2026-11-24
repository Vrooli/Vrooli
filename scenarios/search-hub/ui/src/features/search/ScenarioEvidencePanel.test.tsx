import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { QueryResponseSchema } from "@vrooli/proto-types/search-hub/v1/routing/routing_pb";

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
      groups: [{ providerId: "test-genie.requirements", degraded: true, note: "owner timeout" }],
      degraded: true,
    }));
    renderWithProviders(<ScenarioEvidencePanel />);
    fireEvent.change(screen.getByLabelText("Scenario name"), { target: { value: "search-hub" } });
    fireEvent.click(screen.getByRole("button", { name: "Load evidence" }));
    expect(await screen.findByText(/Provider degraded: owner timeout/)).toBeInTheDocument();
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
});
