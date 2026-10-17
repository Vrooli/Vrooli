/**
 * ScenariosPage — searchable, filterable, sortable scenario inventory.
 *
 * Mirrors the shape of /flows: URL-first filter state, a horizontal
 * filter strip, a table body with checkboxes for the bulk action
 * surface, and a single empty/error/loading dispatch. The bulk
 * surface drives Generate / Clear at scenario scope through the
 * artifacts API.
 *
 * The diagnostic empty state surfaces the resolved Vrooli root and
 * count of directories scanned, so a misconfigured deploy is
 * self-debugging instead of producing a silent zero-state.
 */
import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { fetchScenarios, type ScenarioSummary } from "../api/scenarios";
import {
  clearScenarioArtifacts,
  generateScenarioArtifacts,
} from "../api/artifacts";
import { errorMessage } from "../lib/errorMessage";
import { useTranslation } from "../i18n";
import {
  ScenarioFilters,
  type ScenarioFilterState,
  type ScenarioErrorsKey,
  type ScenarioFlowsKey,
  type ScenarioSortDir,
  type ScenarioSortKey,
} from "../features/scenarios/ScenarioFilters";
import { ScenarioTable } from "../features/scenarios/ScenarioTable";
import { useListState } from "../features/listing/useListState";

const SCENARIOS_KEY = ["scenarios"] as const;
const RUNS_KEY = ["runs", "all"];

function defaultState(): ScenarioFilterState {
  return {
    search: "",
    flows: "any",
    errors: "any",
    sort: { key: "name", dir: "asc" },
  };
}

function readUrl(params: URLSearchParams): ScenarioFilterState {
  const d = defaultState();
  return {
    search: params.get("q") ?? d.search,
    flows: (params.get("flows") as ScenarioFlowsKey | null) ?? d.flows,
    errors: (params.get("errors") as ScenarioErrorsKey | null) ?? d.errors,
    sort: {
      key: (params.get("sort") as ScenarioSortKey | null) ?? d.sort.key,
      dir: (params.get("dir") as ScenarioSortDir | null) ?? d.sort.dir,
    },
  };
}

function writeUrl(s: ScenarioFilterState): URLSearchParams {
  const out = new URLSearchParams();
  if (s.search) out.set("q", s.search);
  if (s.flows !== "any") out.set("flows", s.flows);
  if (s.errors !== "any") out.set("errors", s.errors);
  if (s.sort.key !== "name") out.set("sort", s.sort.key);
  if (s.sort.dir !== "asc") out.set("dir", s.sort.dir);
  return out;
}

export function ScenariosPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { state, setState } = useListState<ScenarioFilterState>({
    fromUrl: readUrl,
    toUrl: writeUrl,
  });
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());

  const query = useQuery({
    queryKey: SCENARIOS_KEY,
    queryFn: fetchScenarios,
  });

  const filtered = useMemo(() => {
    const rows = (query.data?.scenarios ?? []).filter((s) => filterScenario(s, state));
    return sortScenarios(rows, state.sort.key, state.sort.dir);
  }, [query.data, state]);

  const targetIds = useMemo(() => {
    if (selectedIds.size > 0) {
      return filtered.filter((s) => selectedIds.has(s.id)).map((s) => s.id);
    }
    return filtered.map((s) => s.id);
  }, [filtered, selectedIds]);

  const generateAll = useMutation({
    mutationFn: async () => {
      for (const id of targetIds) await generateScenarioArtifacts(id);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: RUNS_KEY });
    },
  });
  const clearAll = useMutation({
    mutationFn: async () => {
      for (const id of targetIds) await clearScenarioArtifacts(id);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: RUNS_KEY });
    },
  });

  const toggleOne = (id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };
  const toggleAll = (selectAll: boolean) => {
    setSelectedIds(selectAll ? new Set(filtered.map((s) => s.id)) : new Set());
  };

  const root = query.data?.vrooliRoot ?? "";

  return (
    <div data-testid="scenarios-page" className="flex flex-col gap-4">
      <header>
        <h1 className="text-2xl font-semibold text-app-foreground">
          {t("scenarios.heading", { defaultValue: "Scenarios" })}
        </h1>
        <p className="mt-1 text-sm text-app-muted-foreground">
          {t("scenarios.subtitle", {
            defaultValue: "Every scenario discovered under the Vrooli root. Filter, sort, and bulk-generate artifacts here.",
          })}
        </p>
      </header>

      <ScenarioFilters
        value={state}
        onChange={setState}
        onReload={() => void query.refetch()}
        onGenerateAll={() => generateAll.mutate()}
        onClearAll={() => clearAll.mutate()}
        generatingAll={generateAll.isPending}
        clearingAll={clearAll.isPending}
        scenariosCount={filtered.length}
        selectedCount={selectedIds.size}
      />

      {query.isLoading && (
        <p data-testid="scenarios-loading" className="text-sm text-app-muted-foreground">
          {t("scenarios.loading", { defaultValue: "Discovering scenarios…" })}
        </p>
      )}

      {query.error && (
        <p data-testid="scenarios-error" className="text-sm text-app-danger">
          {errorMessage(query.error, t)}
        </p>
      )}

      {!query.isLoading && !query.error && query.data && (
        <ScenariosBody
          all={query.data.scenarios}
          filtered={filtered}
          root={root}
          selectedIds={selectedIds}
          onToggleOne={toggleOne}
          onToggleAll={toggleAll}
        />
      )}

      {generateAll.error && (
        <p data-testid="scenarios-generate-error" className="text-sm text-app-danger">
          {errorMessage(generateAll.error, t)}
        </p>
      )}
      {clearAll.error && (
        <p data-testid="scenarios-clear-error" className="text-sm text-app-danger">
          {errorMessage(clearAll.error, t)}
        </p>
      )}
    </div>
  );
}

function ScenariosBody({
  all,
  filtered,
  root,
  selectedIds,
  onToggleOne,
  onToggleAll,
}: {
  all: ScenarioSummary[];
  filtered: ScenarioSummary[];
  root: string;
  selectedIds: Set<string>;
  onToggleOne: (id: string) => void;
  onToggleAll: (selectAll: boolean) => void;
}) {
  const { t } = useTranslation();
  if (all.length === 0) {
    return (
      <div
        data-testid="scenarios-empty"
        className="rounded-panel border border-app-border bg-app-surface p-6 text-sm"
      >
        <h2 className="text-base font-semibold text-app-foreground">
          {t("scenarios.empty.title", { defaultValue: "No scenarios found" })}
        </h2>
        <p className="mt-2 text-app-muted-foreground">
          {t("scenarios.empty.body", {
            defaultValue:
              "Looked under {{root}}/scenarios. Generate a scenario from a template, then it will appear here.",
            root,
          })}
        </p>
      </div>
    );
  }
  if (filtered.length === 0) {
    return (
      <p data-testid="scenarios-no-match" className="text-sm text-app-muted-foreground">
        {t("scenarios.noMatch", { defaultValue: "No scenarios match these filters." })}
      </p>
    );
  }
  return (
    <ScenarioTable
      scenarios={filtered}
      selectedIds={selectedIds}
      onToggleOne={onToggleOne}
      onToggleAll={onToggleAll}
    />
  );
}

function filterScenario(s: ScenarioSummary, state: ScenarioFilterState): boolean {
  const q = state.search.trim().toLowerCase();
  if (q) {
    const hay = [s.id, s.displayName, s.description ?? ""].join(" ").toLowerCase();
    if (!hay.includes(q)) return false;
  }
  if (state.flows === "has" && s.flowCount === 0) return false;
  if (state.flows === "empty" && s.flowCount > 0) return false;
  const hasError = Boolean(s.discoveryError);
  if (state.errors === "with" && !hasError) return false;
  if (state.errors === "clean" && hasError) return false;
  return true;
}

function sortScenarios(
  rows: ScenarioSummary[],
  key: ScenarioSortKey,
  dir: ScenarioSortDir,
): ScenarioSummary[] {
  const mul = dir === "asc" ? 1 : -1;
  return [...rows].sort((a, b) => {
    if (key === "flowCount") {
      const diff = a.flowCount - b.flowCount;
      if (diff !== 0) return diff * mul;
      return a.displayName.localeCompare(b.displayName) * mul;
    }
    const va = key === "name" ? a.displayName : a.id;
    const vb = key === "name" ? b.displayName : b.id;
    return va.localeCompare(vb) * mul;
  });
}
