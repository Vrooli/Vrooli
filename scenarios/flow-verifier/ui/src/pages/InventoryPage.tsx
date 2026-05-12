/**
 * InventoryPage — flat global flow inventory across every scenario.
 *
 * Primary navigation surface for flows is `/scenarios`; this page is
 * the secondary "all flows everywhere" view. State is URL-first so
 * deep links work, with a debounced echo to user settings handled
 * elsewhere (filters value alone is local).
 *
 * The page composes two queries: `/api/v1/scenarios` for the filter
 * options (so the dropdown is populated and we can resolve a
 * scenario's filesystem path before calling verify), and
 * `/api/v1/flows[?scenario=<id>]` for the rows themselves.
 */
import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useSearchParams } from "react-router-dom";

import {
  fetchFlows,
  fetchRuns,
  verifyFlow,
  type FlowSummary,
  type RunRow,
} from "../api/inventory";
import { fetchScenarios } from "../api/scenarios";
import { errorMessage } from "../lib/errorMessage";
import { useTranslation } from "../i18n";
import {
  InventoryFilters,
  type InventoryFilterState,
  type LanguageKey,
  type SortDir,
  type SortKey,
  type StatusKey,
} from "../features/inventory/InventoryFilters";
import { InventoryTable } from "../features/inventory/InventoryTable";

const FLOWS_KEY = (scenarioId: string) => ["flows", scenarioId || "all"] as const;
const RUNS_KEY = ["runs", "all"] as const;
const SCENARIOS_KEY = ["scenarios"] as const;

function defaultState(): InventoryFilterState {
  return {
    scenarioId: "",
    search: "",
    language: "all",
    status: [],
    sort: { key: "flowId", dir: "asc" },
  };
}

function readUrl(params: URLSearchParams): InventoryFilterState {
  const d = defaultState();
  return {
    scenarioId: params.get("scenario") ?? d.scenarioId,
    search: params.get("q") ?? d.search,
    language: (params.get("lang") as LanguageKey | null) ?? d.language,
    status: (params.get("status") ?? "")
      .split(",")
      .map((s) => s.trim())
      .filter(Boolean) as StatusKey[],
    sort: {
      key: (params.get("sort") as SortKey | null) ?? d.sort.key,
      dir: (params.get("dir") as SortDir | null) ?? d.sort.dir,
    },
  };
}

function writeUrl(s: InventoryFilterState): URLSearchParams {
  const out = new URLSearchParams();
  if (s.scenarioId) out.set("scenario", s.scenarioId);
  if (s.search) out.set("q", s.search);
  if (s.language !== "all") out.set("lang", s.language);
  if (s.status.length > 0) out.set("status", s.status.join(","));
  if (s.sort.key !== "flowId") out.set("sort", s.sort.key);
  if (s.sort.dir !== "asc") out.set("dir", s.sort.dir);
  return out;
}

export function InventoryPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();

  const [state, setState] = useState<InventoryFilterState>(() => readUrl(searchParams));

  useEffect(() => {
    setSearchParams(writeUrl(state), { replace: true });
  }, [state, setSearchParams]);

  const scenariosQuery = useQuery({
    queryKey: SCENARIOS_KEY,
    queryFn: fetchScenarios,
  });
  const flowsQuery = useQuery({
    queryKey: FLOWS_KEY(state.scenarioId),
    queryFn: () => fetchFlows({ scenarioId: state.scenarioId || undefined }),
  });
  const runsQuery = useQuery({
    queryKey: RUNS_KEY,
    queryFn: () => fetchRuns({ limit: 200 }),
  });

  // Lookup table from scenarioId → filesystem path so verify mutations
  // can pass the path through to the verifications endpoint (which
  // takes a root, not a scenario id).
  const pathByScenario = useMemo(() => {
    const m = new Map<string, string>();
    for (const s of scenariosQuery.data?.scenarios ?? []) m.set(s.id, s.path);
    return m;
  }, [scenariosQuery.data]);

  const latestByFlow = useMemo(() => {
    const m = new Map<string, RunRow>();
    for (const r of runsQuery.data ?? []) {
      const existing = m.get(r.flowId);
      if (!existing || r.finishedAt > existing.finishedAt) m.set(r.flowId, r);
    }
    return m;
  }, [runsQuery.data]);

  const filtered = useMemo(() => {
    const q = state.search.trim().toLowerCase();
    let rows = (flowsQuery.data ?? []).filter((f) => {
      if (q && !f.flowId.toLowerCase().includes(q)) return false;
      if (state.language !== "all" && f.language !== state.language) return false;
      if (state.status.length > 0) {
        const last = latestByFlow.get(f.flowId);
        const key: StatusKey = (last?.status as StatusKey | undefined) ?? "none";
        if (!state.status.includes(key)) return false;
      }
      return true;
    });
    rows = sortRows(rows, latestByFlow, state.sort.key, state.sort.dir);
    return rows;
  }, [flowsQuery.data, latestByFlow, state]);

  const verifyAll = useMutation({
    mutationFn: async (flows: FlowSummary[]) => {
      for (const flow of flows) {
        const root = pathByScenario.get(flow.scenarioId ?? "") ?? "";
        if (!root) continue;
        await verifyFlow(root, flow.flowId);
        await queryClient.invalidateQueries({ queryKey: RUNS_KEY });
      }
    },
  });
  const verifyOne = useMutation({
    mutationFn: async (flow: FlowSummary) => {
      const root = pathByScenario.get(flow.scenarioId ?? "") ?? "";
      if (!root) throw new Error("scenario path unresolved for " + flow.flowId);
      await verifyFlow(root, flow.flowId);
      await queryClient.invalidateQueries({ queryKey: RUNS_KEY });
    },
  });

  const anyPending = verifyAll.isPending || verifyOne.isPending;

  return (
    <div data-testid="inventory-page" className="flex flex-col gap-4">
      <header>
        <h1 className="text-2xl font-semibold text-app-foreground">
          {t("inventory.heading", { defaultValue: "All flows" })}
        </h1>
        <p className="mt-1 text-sm text-app-muted-foreground">
          {t("inventory.subtitle", {
            defaultValue: "Search, filter, and verify every discovered flow across every scenario.",
          })}
        </p>
      </header>

      <InventoryFilters
        value={state}
        scenarios={scenariosQuery.data?.scenarios ?? []}
        onChange={setState}
        onReload={() => {
          void flowsQuery.refetch();
          void runsQuery.refetch();
          void scenariosQuery.refetch();
        }}
        onVerifyAll={() => verifyAll.mutate(filtered)}
        verifyingAll={verifyAll.isPending}
        flowsCount={filtered.length}
      />

      {flowsQuery.isLoading && (
        <p data-testid="inventory-loading" className="text-sm text-app-muted-foreground">
          {t("inventory.loading", { defaultValue: "Discovering flows…" })}
        </p>
      )}
      {flowsQuery.error && (
        <p data-testid="inventory-error" className="text-sm text-app-danger">
          {errorMessage(flowsQuery.error, t)}
        </p>
      )}
      {!flowsQuery.isLoading && filtered.length === 0 && !flowsQuery.error && (
        <p data-testid="inventory-empty" className="text-sm text-app-muted-foreground">
          {(flowsQuery.data ?? []).length === 0
            ? t("inventory.emptyAll", {
                defaultValue:
                  "No flows discovered yet. Generate or migrate a scenario to use schema-v6 flow.json files.",
              })
            : t("inventory.empty", { defaultValue: "No flows match these filters." })}
        </p>
      )}
      {filtered.length > 0 && (
        <InventoryTable
          flows={filtered}
          latestByFlow={latestByFlow}
          onVerifyOne={(flow) => verifyOne.mutate(flow)}
          verifyingFlowId={verifyOne.isPending ? verifyOne.variables.flowId : undefined}
          anyPending={anyPending}
        />
      )}

      {verifyAll.error && (
        <p data-testid="inventory-verify-error" className="text-sm text-app-danger">
          {errorMessage(verifyAll.error, t)}
        </p>
      )}
      {verifyOne.error && (
        <p data-testid="inventory-verify-one-error" className="text-sm text-app-danger">
          {errorMessage(verifyOne.error, t)}
        </p>
      )}
    </div>
  );
}

function sortRows(
  rows: FlowSummary[],
  latest: Map<string, RunRow>,
  key: SortKey,
  dir: SortDir,
): FlowSummary[] {
  const mul = dir === "asc" ? 1 : -1;
  return [...rows].sort((a, b) => {
    const va = sortValue(a, latest, key);
    const vb = sortValue(b, latest, key);
    if (va < vb) return -1 * mul;
    if (va > vb) return 1 * mul;
    return 0;
  });
}

function sortValue(
  f: FlowSummary,
  latest: Map<string, RunRow>,
  key: SortKey,
): string {
  if (key === "flowId") return f.flowId;
  if (key === "language") return f.language;
  const last = latest.get(f.flowId);
  if (key === "status") return last?.status ?? "";
  return last?.finishedAt ?? "";
}
