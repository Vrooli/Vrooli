import { NavLink } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { cn } from "../lib/utils";
import { buildWorkspaceSubPath, WORKSPACE_SUBNAV } from "./workspaceSubNav";

/**
 * Per-target workspace sub-nav. Renders a horizontal tab strip on desktop and
 * a horizontally-scrollable strip on mobile. Items that have not yet shipped
 * (Graph/Domains/Apply/Analytics in Phase 4) render as disabled chips with
 * no router target — they flip to `available: true` in their phase.
 */
export interface WorkspaceSubNavProps {
  scenario: string;
}

export function WorkspaceSubNav({ scenario }: WorkspaceSubNavProps) {
  const { t } = useTranslation();
  return (
    <nav
      data-testid={selectors.layout.subnav}
      aria-label={t(strings.layout.subnav.label)}
      className="-mx-1 flex gap-1 overflow-x-auto border-b border-app-border pb-1"
    >
      {WORKSPACE_SUBNAV.map((item) =>
        item.available ? (
          <NavLink
            key={item.key}
            to={buildWorkspaceSubPath(scenario, item)}
            data-testid={selectors.layout.workspaceSubnavLink({ key: item.key })}
            className={({ isActive }) =>
              cn(
                "shrink-0 rounded-control px-3 py-1.5 text-sm font-medium transition-colors",
                isActive
                  ? "bg-app-primary text-app-primary-foreground"
                  : "text-app-foreground hover:bg-app-surface-muted",
              )
            }
          >
            {t(item.labelKey)}
          </NavLink>
        ) : (
          <span
            key={item.key}
            data-testid={selectors.layout.workspaceSubnavLink({ key: item.key })}
            aria-disabled="true"
            className="shrink-0 rounded-control px-3 py-1.5 text-sm font-medium text-app-muted-foreground opacity-60"
          >
            {t(item.labelKey)}
          </span>
        ),
      )}
    </nav>
  );
}
