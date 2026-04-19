import { lazy } from "react";
import { useQuery } from "@tanstack/react-query";
import { DashboardLayout } from "../components/DashboardLayout";
import { MetricList } from "../components/MetricList";
import { SceneCanvas } from "../components/SceneCanvas";
import { StaleBadge } from "../components/StaleBadge";
import { fetchDashboard, type DashboardResponse } from "../lib/api";

const MissionControlScene = lazy(() => import("../scenes/missionControl"));

export default function MissionControl() {
  const { data, isLoading, error } = useQuery<DashboardResponse>({
    queryKey: ["dashboard", "mission-control"],
    queryFn: () => fetchDashboard("mission-control"),
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
      themeKey="ground-control"
      title="Mission Control"
      aside={<MetricList metrics={metrics} />}
    >
      {error ? (
        <div className="cc-error" data-testid="error-banner">
          Unable to load Mission Control dashboard. Showing last known data when available.
        </div>
      ) : null}
      {stalestTs !== undefined ? (
        <div style={{ marginBottom: "0.75rem" }}>
          <StaleBadge ts={stalestTs} />
        </div>
      ) : null}
      {isLoading ? (
        <div className="cc-loading" data-testid="loading">
          Loading Mission Control…
        </div>
      ) : (
        <SceneCanvas scene={<MissionControlScene />} />
      )}
    </DashboardLayout>
  );
}
