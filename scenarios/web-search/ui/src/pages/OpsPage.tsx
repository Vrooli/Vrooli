import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { OpsPanel } from "../features/ops/OpsPanel";
import { useTranslation } from "../i18n";

/**
 * Operations page. Dependency status (SearXNG reachability, findings store)
 * plus the last live-search health readout.
 */
export function OpsPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.ops}
      aria-labelledby="ops-heading"
      className="flex flex-col gap-6"
    >
      <div className="flex flex-col gap-1">
        <h2 id="ops-heading" className="text-2xl font-semibold">
          {t(strings.pages.ops.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.pages.ops.description)}</p>
      </div>
      <OpsPanel />
    </section>
  );
}
