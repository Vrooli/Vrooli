import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { SeverityBadge } from "../../components/SeverityBadge";

/**
 * Legend for the package graph. Two states: a plain package and a package
 * participating in an import cycle. The cycle marker carries a label+icon
 * (via SeverityBadge), never color alone.
 */
export function GraphLegend() {
  const { t } = useTranslation();
  return (
    <section
      data-testid={selectors.features.explorer.legend.root}
      aria-label={t(strings.explorer.legend.title)}
      className="flex flex-col gap-2 rounded-panel border border-app-border bg-app-surface p-3 backdrop-blur-sm"
    >
      <h4 className="text-sm font-semibold">{t(strings.explorer.legend.title)}</h4>
      <ul className="flex flex-wrap items-center gap-3 text-xs">
        <li className="flex items-center gap-2">
          <span
            aria-hidden="true"
            className="inline-block h-3 w-6 rounded-control border border-app-border bg-app-surface-muted"
          />
          <span>{t(strings.explorer.legend.package)}</span>
        </li>
        <li
          data-testid={selectors.features.explorer.legend.severity({ level: "high" })}
          className="flex items-center gap-2"
        >
          <span
            aria-hidden="true"
            className="inline-block h-3 w-6 rounded-control border-2 border-app-danger bg-app-surface-muted"
          />
          <SeverityBadge level="high" label={t(strings.explorer.legend.cycle)} />
        </li>
      </ul>
    </section>
  );
}
