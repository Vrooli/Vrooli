/**
 * SidebarFlowList — compact discovered-flow list shown inside the sidebar.
 *
 * Pulls the cached `flows` query via React Query (filled by the inventory
 * page); if empty, falls back to an inline fetch. Renders as compact
 * link rows so users can jump straight to a flow without leaving keyboard
 * range of the sidebar.
 */
import { useQuery } from "@tanstack/react-query";
import { NavLink, useParams } from "react-router-dom";

import { fetchFlows } from "../api/inventory";
import { useTranslation } from "../i18n";

interface Props {
  onNavigate?: () => void;
}

export function SidebarFlowList({ onNavigate }: Props) {
  const { t } = useTranslation();
  const { flowId: active } = useParams();
  const { data, isLoading, error } = useQuery({
    queryKey: ["flows", "all"],
    queryFn: () => fetchFlows(),
    staleTime: 30_000,
  });

  return (
    <div data-testid="sidebar-flow-list">
      <h2 className="px-3 pb-1 text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">
        {t("sidebar.flows", { defaultValue: "Discovered flows" })}
      </h2>
      {isLoading && (
        <p data-testid="sidebar-flow-list-loading" className="px-3 py-1 text-xs text-app-muted-foreground">
          {t("sidebar.loading", { defaultValue: "Loading…" })}
        </p>
      )}
      {error && (
        <p data-testid="sidebar-flow-list-error" className="px-3 py-1 text-xs text-app-danger">
          {t("sidebar.error", { defaultValue: "Failed to load flows" })}
        </p>
      )}
      {!isLoading && !error && (data?.length ?? 0) === 0 && (
        <p data-testid="sidebar-flow-list-empty" className="px-3 py-1 text-xs text-app-muted-foreground">
          {t("sidebar.empty", { defaultValue: "No flows discovered" })}
        </p>
      )}
      <ul className="flex flex-col">
        {(data ?? []).map((f) => (
          <li key={f.flowId}>
            <NavLink
              to={
                f.scenarioId
                  ? `/flows/${encodeURIComponent(f.flowId)}?scenario=${encodeURIComponent(f.scenarioId)}`
                  : `/flows/${encodeURIComponent(f.flowId)}`
              }
              onClick={onNavigate}
              data-testid={`sidebar-flow-${f.flowId}`}
              data-active={active === f.flowId ? "true" : undefined}
              className={({ isActive }) =>
                [
                  "block truncate rounded-control px-3 py-1.5 text-xs",
                  isActive
                    ? "bg-app-surface-muted font-medium text-app-foreground"
                    : "text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground",
                ].join(" ")
              }
              title={f.flowId}
            >
              {f.flowId}
            </NavLink>
          </li>
        ))}
      </ul>
    </div>
  );
}
