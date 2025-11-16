import { buildApiUrl } from "@vrooli/api-base";
import { useCallback, useEffect, useMemo, useState } from "react";
import { getApiBaseUrl } from "../lib/utils";
import type { ScenarioDetailResponse, ScenarioSummary } from "../types";

interface ScanOptions {
  apply?: boolean;
  applyResources?: boolean;
  applyScenarios?: boolean;
}

export function useScenarioCatalog() {
  const apiBase = useMemo(() => getApiBaseUrl(), []);
  const [summaries, setSummaries] = useState<ScenarioSummary[]>([]);
  const [loadingSummaries, setLoadingSummaries] = useState(false);
  const [selectedScenario, setSelectedScenario] = useState<string | null>(null);
  const [detail, setDetail] = useState<ScenarioDetailResponse | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [scanLoading, setScanLoading] = useState(false);
  const [optimizeLoading, setOptimizeLoading] = useState(false);

  const fetchSummaries = useCallback(async () => {
    try {
      setLoadingSummaries(true);
      const response = await fetch(buildApiUrl("/scenarios", { baseUrl: apiBase }));
      if (!response.ok) throw new Error("Failed to load scenarios");
      const payload = (await response.json()) as ScenarioSummary[];
      setSummaries(payload);
    } catch (error) {
      console.error("Failed to fetch scenarios", error);
    } finally {
      setLoadingSummaries(false);
    }
  }, [apiBase]);

  const fetchDetail = useCallback(
    async (name: string) => {
      try {
        setDetailLoading(true);
        const response = await fetch(buildApiUrl(`/scenarios/${name}`, { baseUrl: apiBase }));
        if (!response.ok) throw new Error("Failed to load scenario detail");
        const payload = (await response.json()) as ScenarioDetailResponse;
        setDetail(payload);
      } catch (error) {
        console.error("Failed to fetch scenario detail", error);
      } finally {
        setDetailLoading(false);
      }
    },
    [apiBase]
  );

  const selectScenario = useCallback(
    (name: string) => {
      setSelectedScenario(name);
      setDetail(null);
      void fetchDetail(name);
    },
    [fetchDetail]
  );

  const scanScenario = useCallback(
    async (name: string, options?: ScanOptions) => {
      try {
        setScanLoading(true);
        const response = await fetch(buildApiUrl(`/scenarios/${name}/scan`, { baseUrl: apiBase }), {
          method: "POST",
          headers: {
            "Content-Type": "application/json"
          },
          body: JSON.stringify({
            apply: options?.apply ?? false,
            apply_resources: options?.applyResources ?? false,
            apply_scenarios: options?.applyScenarios ?? false
          })
        });
        if (!response.ok) throw new Error("Scan failed");
        await response.json();
        await fetchDetail(name);
        await fetchSummaries();
      } catch (error) {
        console.error("Failed to scan scenario", error);
      } finally {
        setScanLoading(false);
      }
    },
    [apiBase, fetchDetail, fetchSummaries]
  );

  const optimizeScenario = useCallback(
    async (name: string, options?: { apply?: boolean }) => {
      try {
        setOptimizeLoading(true);
        const response = await fetch(buildApiUrl("/optimize", { baseUrl: apiBase }), {
          method: "POST",
          headers: {
            "Content-Type": "application/json"
          },
          body: JSON.stringify({
            scenario: name,
            type: "all",
            apply: options?.apply ?? false
          })
        });
        if (!response.ok) {
          throw new Error("Optimization failed");
        }
        await response.json();
        await fetchDetail(name);
        await fetchSummaries();
      } catch (error) {
        console.error("Failed to optimize scenario", error);
      } finally {
        setOptimizeLoading(false);
      }
    },
    [apiBase, fetchDetail, fetchSummaries]
  );

  useEffect(() => {
    fetchSummaries().catch((error) => console.error(error));
  }, [fetchSummaries]);

  return {
    summaries,
    loadingSummaries,
    selectedScenario,
    detail,
    detailLoading,
    scanLoading,
     optimizeLoading,
    selectScenario,
    refreshSummaries: fetchSummaries,
    scanScenario,
    optimizeScenario
  };
}
