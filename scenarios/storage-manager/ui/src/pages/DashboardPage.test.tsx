import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";

import { DashboardPage } from "./DashboardPage";
import { selectors } from "../consts/selectors";
import { renderWithProviders } from "../test-utils";

const snapshot = {
  snapshot_id: "snap-1",
  observed_at: "2026-08-02T16:00:00Z",
  root: "/fixture",
  measured_bytes: 4096,
  attributed_bytes: 3072,
  unattributed_bytes: 1024,
  closed: false,
  accounting_identity: true,
  confidence: "degraded",
  entries: [{ owner: "scenario-a", kind: "scenario", name: "data", path: "/fixture/data", bytes: 3072, declared: true }],
  findings: [{ code: "unattributed_storage", severity: "warning", path: "/fixture", message: "1024 bytes have no declaration" }],
};

const responseFor = (url: string) => {
  if (url.endsWith("/health")) return { status: "ok", service: "storage-manager", timestamp: "2026-08-02T16:00:00Z" };
  if (url.includes("/storage/inventory")) return { repo_root: "/repo", owners: [{ kind: "scenario", id: "scenario-a", manifest_path: "/repo/scenarios/scenario-a/.vrooli/service.json", storage_entries: [{ name: "data", kind: "directory" }] }], findings: [] };
  if (url.includes("/census/history")) return [snapshot];
  if (url.includes("/adoption")) return { total_owners: 1, findings: 0, by_kind: { scenario: { total: 1, with_storage: 1, with_budget: 0 } }, suggestions: [] };
  if (url.includes("/infra-health/storage")) return { owner_count: 1, owners_with_declared_ceiling: 0, declared_ceiling_coverage: 0, snapshot_count: 1, confidence: "degraded", latest_snapshot: snapshot };
  if (url.includes("/retention/owners")) return { owners: [], findings: [] };
  if (url.includes("/placement/audit")) return [];
  if (url.includes("/placement")) return { platform: "linux", owners: [], lever_warnings: [] };
  throw new Error(`Unhandled fixture URL: ${url}`);
};

describe("DashboardPage", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL) => new Response(JSON.stringify(responseFor(String(input))), { status: 200, headers: { "Content-Type": "application/json" } })));
  });

  afterEach(() => vi.unstubAllGlobals());

  it("renders API-backed accounting and owner data", async () => {
    renderWithProviders(<DashboardPage />);

    await waitFor(() => expect(screen.getByTestId(selectors.console.summary)).toBeInTheDocument());
    expect(screen.getAllByText("3.0 KB")).toHaveLength(2);
    expect(screen.getByText("scenario-a")).toBeInTheDocument();
    expect(screen.getByText("Accounting is open.")).toBeInTheDocument();
    expect(screen.getByTestId(selectors.console.confidence)).toHaveTextContent("degraded confidence");
  });

  it("keeps an empty ledger explicit instead of inventing totals", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockImplementation(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.endsWith("/health")) return new Response(JSON.stringify({ status: "ok", service: "storage-manager", timestamp: "2026-08-02T16:00:00Z" }));
      if (url.includes("/storage/inventory")) return new Response(JSON.stringify({ repo_root: "/repo", owners: [], findings: [] }));
      if (url.includes("/census/history")) return new Response("[]");
      if (url.includes("/adoption")) return new Response(JSON.stringify({ total_owners: 0, findings: 0, by_kind: {}, suggestions: [] }));
      if (url.includes("/infra-health/storage")) return new Response(JSON.stringify({ owner_count: 0, owners_with_declared_ceiling: 0, declared_ceiling_coverage: 0, snapshot_count: 0, confidence: "unknown" }));
      if (url.includes("/placement")) return new Response(JSON.stringify({ platform: "linux", owners: [] }));
      if (url.includes("/retention/owners")) return new Response(JSON.stringify({ owners: [], findings: [] }));
      if (url.includes("/placement/audit")) return new Response("[]");
      throw new Error(`Unhandled fixture URL: ${url}`);
    });

    renderWithProviders(<DashboardPage />);
    await waitFor(() => expect(screen.getByTestId(selectors.console.empty)).toBeInTheDocument());
    expect(screen.getByText("No census snapshot yet")).toBeInTheDocument();
  });

  it("renders measured suggestions, placement warnings, and audit history", async () => {
    const measured = { ...snapshot, closed: true, confidence: "high", unattributed_bytes: 0, findings: [], entries: [{ owner: "tool-a", kind: "tool", name: "cache", path: "/fixture/cache", bytes: 3072, declared: true }] };
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.endsWith("/health")) return new Response(JSON.stringify({ status: "ok", service: "storage-manager", timestamp: "2026-08-02T16:00:00Z" }));
      if (url.includes("/storage/inventory")) return new Response(JSON.stringify({ repo_root: "/repo", owners: [{ kind: "tool", id: "tool-a", manifest_path: "/repo/internal/tools/tool-a/tool.json", storage_entries: [] }], findings: [{ code: "missing_budget", severity: "warning", owner_id: "tool-a", message: "budget needs review" }] }));
      if (url.includes("/census/history")) return new Response(JSON.stringify([measured]));
      if (url.includes("/adoption")) return new Response(JSON.stringify({ total_owners: 1, findings: 0, by_kind: { tool: { total: 1, with_storage: 0, with_budget: 0 } }, suggestions: [{ kind: "tool", owner: "tool-a", manifest_path: "/repo/internal/tools/tool-a/tool.json", priority: "review", reason: "owner has no storage.entries declaration", observed_bytes: 8192, measurement_complete: true }] }));
      if (url.includes("/infra-health/storage")) return new Response(JSON.stringify({ owner_count: 1, owners_with_declared_ceiling: 0, declared_ceiling_coverage: 0, snapshot_count: 1, confidence: "high", latest_snapshot: measured }));
      if (url.includes("/retention/owners")) return new Response(JSON.stringify({ owners: [{ kind: "tool", id: "tool-a", manifest_path: "/repo/internal/tools/tool-a/tool.json", budgets: [{ name: "cache", target_kind: "file", max_bytes: 1024 }] }], findings: [] }));
      if (url.includes("/placement/audit")) return new Response(JSON.stringify([{ id: "audit-1", plan_id: "plan-1", entry: "cache", source: "/a", destination: "/b", status: "completed", source_preserved: false }]));
      if (url.includes("/placement")) return new Response(JSON.stringify({ platform: "linux", owners: [{ kind: "tool", owner: "tool-a", entry: "cache", rung: "owned", error: "unsupported on this platform", applicable: false }], lever_warnings: [{ key: "cache", message: "manual approval" }], lever_error: "lever registry failed" }));
      throw new Error(`Unhandled fixture URL: ${url}`);
    });

    renderWithProviders(<DashboardPage />);
    await waitFor(() => expect(screen.getByTestId(selectors.console.summary)).toBeInTheDocument());
    expect(screen.getByTestId(selectors.console.confidence)).toHaveTextContent("high confidence");
    expect(screen.getAllByText("tool-a")).toHaveLength(2);
    expect(screen.queryByText("No migrations recorded")).not.toBeInTheDocument();
    expect(screen.getByText("lever registry failed")).toBeInTheDocument();
    expect(screen.getByText("Review")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("tab", { name: "Coverage" }));
    expect(screen.getByRole("tab", { name: "Coverage" })).toHaveAttribute("aria-selected", "true");
  });

  it("labels partial API failure instead of hiding it", async () => {
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes("/infra-health/storage")) return new Response("unavailable", { status: 503 });
      return new Response(JSON.stringify(responseFor(url)), { status: 200, headers: { "Content-Type": "application/json" } });
    });

    renderWithProviders(<DashboardPage />);
    await waitFor(() => expect(screen.getByTestId(selectors.console.error)).toBeInTheDocument());
    expect(screen.getByText("Some operational surfaces are unavailable. Values below are labeled by source and may be incomplete.")).toBeInTheDocument();
  });
});
