import { Link } from "react-router-dom";
import { X } from "lucide-react";

import { Button } from "../../components/ui/button";
import { EmptyState } from "../../components/EmptyState";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";
import { encodeScenarioPath } from "../../hooks/useScenarioPath";
import { type RecentTarget } from "./hooks/useRecentTargets";

export interface RecentTargetsListProps {
  recent: readonly RecentTarget[];
  onRemove: (scenario: string) => void;
}

/**
 * Lists the most-recently-opened scenarios. Each row links into the target
 * workspace and exposes a `Remove` affordance so users can prune the list
 * without having to clear localStorage by hand.
 */
export function RecentTargetsList({ recent, onRemove }: RecentTargetsListProps) {
  const { t } = useTranslation();

  if (recent.length === 0) {
    return (
      <div data-testid={selectors.features.targets.recent.empty}>
        <EmptyState
          title={t(strings.targets.recent.emptyTitle)}
          description={t(strings.targets.recent.emptyDescription)}
        />
      </div>
    );
  }

  return (
    <ul
      data-testid={selectors.features.targets.recent.root}
      className="flex flex-col divide-y divide-app-border rounded-panel border border-app-border bg-app-surface"
    >
      {recent.map((entry) => {
        const openedAt = formatDate(new Date(entry.lastOpenedAt), {
          dateStyle: "medium",
          timeStyle: "short",
        });
        return (
          <li
            key={entry.scenario}
            data-testid={selectors.features.targets.recent.item({ scenario: entry.scenario })}
            className="flex items-center justify-between gap-3 p-4"
          >
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-medium text-app-foreground">{entry.scenario}</p>
              <p className="mt-1 truncate text-xs text-app-muted-foreground">
                {t(strings.targets.recent.openedAt, { at: openedAt })}
              </p>
            </div>
            <div className="flex shrink-0 items-center gap-2">
              <Button
                asChild
                size="sm"
                variant="outline"
                data-testid={selectors.features.targets.recent.openButton({
                  scenario: entry.scenario,
                })}
              >
                <Link to={`/targets/${encodeScenarioPath(entry.scenario)}`}>
                  {t(strings.targets.recent.openButton)}
                </Link>
              </Button>
              <Button
                size="sm"
                variant="outline"
                onClick={() => onRemove(entry.scenario)}
                aria-label={t(strings.targets.recent.removeAriaLabel, {
                  scenario: entry.scenario,
                })}
                data-testid={selectors.features.targets.recent.removeButton({
                  scenario: entry.scenario,
                })}
              >
                <X aria-hidden="true" className="h-4 w-4" />
              </Button>
            </div>
          </li>
        );
      })}
    </ul>
  );
}
