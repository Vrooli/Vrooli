import { lazy } from "react";
import { useQuery } from "@tanstack/react-query";
import { fetchDashboard, type DashboardResponse } from "../lib/api";
import { DashboardLayout } from "./DashboardLayout";
import { MetricList } from "./MetricList";
import { SceneCanvas } from "./SceneCanvas";
import { StaleBadge } from "./StaleBadge";
import type { ThemeKey } from "./ThemeProvider";

const TrivialCube = lazy(() => import("../scenes/trivialCube"));

interface PlaceholderPageProps {
  themeKey: ThemeKey;
  dashboardId: string;
  title: string;
}

/**
 * Shared page used by the five dashboards that ship as themed placeholders.
 * Fetches the dashboard payload, renders the MetricList alongside a trivial
 * rotating-cube scene, and surfaces stale/error states.
 */
export function PlaceholderPage({ themeKey, dashboardId, title }: PlaceholderPageProps) {
  const { data, isLoading, error } = useQuery<DashboardResponse>({
    queryKey: ["dashboard", dashboardId],
    queryFn: () => fetchDashboard(dashboardId),
  });

  const metrics = data?.metrics ?? [];
  const sources = data?.sources ?? {};
  const stalestTs = Object.values(sources)
    .map((meta) => meta.staleness_ts)
    .filter((ts): ts is string => typeof ts === "string")
    .sort()
    .pop();

  return (
    <DashboardLayout
      themeKey={themeKey}
      title={title}
      aside={<MetricList metrics={metrics} />}
    >
      {error ? (
        <div className="cc-error" data-testid="error-banner">
          Unable to load dashboard. Showing last known data when available.
        </div>
      ) : null}
      {stalestTs !== undefined ? (
        <div style={{ marginBottom: "0.75rem" }}>
          <StaleBadge ts={stalestTs} />
        </div>
      ) : null}
      {isLoading ? (
        <div className="cc-loading" data-testid="loading">
          Loading {title}…
        </div>
      ) : (
        <SceneCanvas scene={<TrivialCube />} />
      )}
    </DashboardLayout>
  );
}
