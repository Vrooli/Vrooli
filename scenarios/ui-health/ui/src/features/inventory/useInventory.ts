import { useCallback, useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { useQuery, type UseQueryResult } from "@tanstack/react-query";

import {
  countSurfacesByKind,
  filterSurfaces,
  scanScenario,
  SURFACE_KIND_FILTERS,
  type InventoryScan,
  type SurfaceKindFilter,
  type SurfaceRecord,
} from "../../api/inventory";

const SCENARIO_PARAM = "scenario";
const KIND_PARAM = "kind";

function isSurfaceKindFilter(value: string | null): value is SurfaceKindFilter {
  return value !== null && (SURFACE_KIND_FILTERS as readonly string[]).includes(value);
}

export function inventoryQueryKey(scenario: string): readonly unknown[] {
  return ["inventory", "scan", scenario] as const;
}

export type UseInventoryReturn = {
  scenario: string;
  setScenario: (next: string) => void;
  /** The scenario string actually scanned (matches query data). */
  activeScenario: string;
  submit: (next: string) => void;
  kind: SurfaceKindFilter;
  setKind: (next: SurfaceKindFilter) => void;
  filteredSurfaces: SurfaceRecord[];
  countByKind: Record<SurfaceKindFilter, number>;
  query_: UseQueryResult<InventoryScan>;
};

export function useInventory(): UseInventoryReturn {
  const [params, setParams] = useSearchParams();
  const initialScenario = params.get(SCENARIO_PARAM) ?? "";
  const initialKindRaw = params.get(KIND_PARAM);
  const initialKind: SurfaceKindFilter = isSurfaceKindFilter(initialKindRaw)
    ? initialKindRaw
    : "all";

  const [scenario, setScenarioState] = useState<string>(initialScenario);
  const [activeScenario, setActiveScenario] = useState<string>(initialScenario);
  const [kind, setKindState] = useState<SurfaceKindFilter>(initialKind);

  useEffect(() => {
    const next = new URLSearchParams(params);
    if (activeScenario.length > 0) next.set(SCENARIO_PARAM, activeScenario);
    else next.delete(SCENARIO_PARAM);
    if (kind !== "all") next.set(KIND_PARAM, kind);
    else next.delete(KIND_PARAM);
    if (next.toString() !== params.toString()) {
      setParams(next, { replace: true });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- setParams stable, params via .toString()
  }, [activeScenario, kind]);

  const query_ = useQuery({
    queryKey: inventoryQueryKey(activeScenario),
    queryFn: () => scanScenario(activeScenario),
    enabled: activeScenario.length > 0,
    staleTime: 30_000,
    refetchOnWindowFocus: false,
  });

  const surfaces = useMemo(() => query_.data?.surfaces ?? [], [query_.data]);
  const filteredSurfaces = useMemo(
    () => filterSurfaces(surfaces, kind),
    [surfaces, kind],
  );
  const countByKind = useMemo(() => countSurfacesByKind(surfaces), [surfaces]);

  const submit = useCallback((next: string) => {
    setActiveScenario(next);
  }, []);

  const setScenario = useCallback((next: string) => setScenarioState(next), []);
  const setKind = useCallback((next: SurfaceKindFilter) => setKindState(next), []);

  return {
    scenario,
    setScenario,
    activeScenario,
    submit,
    kind,
    setKind,
    filteredSurfaces,
    countByKind,
    query_,
  };
}
