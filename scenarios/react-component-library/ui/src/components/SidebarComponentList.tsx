/**
 * SidebarComponentList — compact registered-component list shown inside
 * the sidebar.
 *
 * Pulls the cached `components` query via React Query (filled by the
 * Components page); renders rows as links to /components/:id so users
 * can jump straight into the editor without leaving keyboard range of
 * the sidebar.
 */
import { useQuery } from "@tanstack/react-query";
import { NavLink, useParams } from "react-router-dom";

import { componentsClient } from "../api/components";
import { useTranslation } from "../i18n";

interface Props {
  onNavigate?: () => void;
}

export function SidebarComponentList({ onNavigate }: Props) {
  const { t } = useTranslation();
  const { id: active } = useParams();
  const { data, isLoading, error } = useQuery({
    queryKey: ["components", "list", "sidebar"],
    queryFn: () => componentsClient.listComponents({ limit: 100 }),
    staleTime: 30_000,
  });

  const items = data?.components ?? [];

  return (
    <div data-testid="sidebar-component-list">
      <h2 className="px-3 pb-1 text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">
        {t("sidebar.components", { defaultValue: "Components" })}
      </h2>
      {isLoading && (
        <p
          data-testid="sidebar-component-list-loading"
          className="px-3 py-1 text-xs text-app-muted-foreground"
        >
          {t("sidebar.loading", { defaultValue: "Loading…" })}
        </p>
      )}
      {error && (
        <p
          data-testid="sidebar-component-list-error"
          className="px-3 py-1 text-xs text-app-danger"
        >
          {t("sidebar.error", { defaultValue: "Failed to load components" })}
        </p>
      )}
      {!isLoading && !error && items.length === 0 && (
        <p
          data-testid="sidebar-component-list-empty"
          className="px-3 py-1 text-xs text-app-muted-foreground"
        >
          {t("sidebar.empty", { defaultValue: "No components indexed yet" })}
        </p>
      )}
      <ul className="flex flex-col">
        {items.map((c) => (
          <li key={c.id}>
            <NavLink
              to={`/components/${encodeURIComponent(c.id)}`}
              onClick={onNavigate}
              data-testid={`sidebar-component-${c.id}`}
              data-active={active === c.id ? "true" : undefined}
              className={({ isActive }) =>
                [
                  "block truncate rounded-control px-3 py-1.5 text-xs",
                  isActive
                    ? "bg-app-surface-muted font-medium text-app-foreground"
                    : "text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground",
                ].join(" ")
              }
              title={c.libraryId || c.displayName || c.id}
            >
              {c.displayName || c.libraryId || c.id}
            </NavLink>
          </li>
        ))}
      </ul>
    </div>
  );
}
