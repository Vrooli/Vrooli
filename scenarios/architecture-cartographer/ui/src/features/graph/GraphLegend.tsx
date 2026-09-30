import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { SeverityBadge } from "../../components/SeverityBadge";
import type { SeverityLevel } from "../../components/SeverityBadge";

const LEVELS: readonly SeverityLevel[] = ["info", "low", "medium", "high", "critical"];

const SEVERITY_LABEL_KEY = {
  info: strings.shared.severity.info,
  low: strings.shared.severity.low,
  medium: strings.shared.severity.medium,
  high: strings.shared.severity.high,
  critical: strings.shared.severity.critical,
} as const satisfies Record<SeverityLevel, string>;

export function GraphLegend() {
  const { t } = useTranslation();
  return (
    <section
      data-testid={selectors.features.graph.legend.root}
      aria-label={t(strings.features.graph.legend.title)}
      className="flex flex-col gap-2 rounded-panel border border-app-border bg-app-surface p-3 backdrop-blur-sm"
    >
      <h4 className="text-sm font-semibold">{t(strings.features.graph.legend.title)}</h4>
      <ul className="flex flex-wrap items-center gap-3 text-xs">
        <li
          data-testid={selectors.features.graph.legend.noConflict}
          className="flex items-center gap-2"
        >
          <span
            aria-hidden="true"
            className="inline-block h-3 w-6 rounded-control border border-app-border bg-app-surface-muted"
          />
          <span>{t(strings.features.graph.legend.noConflict)}</span>
        </li>
        {LEVELS.map((level) => {
          const label = t(SEVERITY_LABEL_KEY[level]);
          return (
            <li
              key={level}
              data-testid={selectors.features.graph.legend.severity({ level })}
              className="flex items-center gap-2"
            >
              <SeverityBadge level={level} label={label} />
              <span className="text-app-muted-foreground">
                {t(strings.features.graph.legend.conflictPrefix, { level: label })}
              </span>
            </li>
          );
        })}
      </ul>
    </section>
  );
}
