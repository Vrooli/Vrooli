import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { EvalsPanel } from "../features/evals/EvalsPanel";
import { useTranslation } from "../i18n";

/**
 * EvalsPage hosts the search-quality baseline harness: per-provider golden
 * suites, their tagged run history, and run comparison. It sits beside the
 * search page as the observability/experimentation surface.
 */
export function EvalsPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.evals}
      aria-labelledby="evals-heading"
      className="flex flex-col gap-4"
    >
      <div className="flex flex-col gap-1">
        <h2 id="evals-heading" className="text-2xl font-semibold">
          {t(strings.pages.evals.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.pages.evals.description)}</p>
      </div>
      <EvalsPanel />
    </section>
  );
}
