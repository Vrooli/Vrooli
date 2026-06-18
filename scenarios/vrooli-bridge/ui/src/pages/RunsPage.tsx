import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { RunHistory } from "../features/runs/RunHistory";
import { useTranslation } from "../i18n";

/**
 * Runs page. Hosts the full run-history feed (durable remote execution): an
 * operator drills into any run to watch its live output, track progress + ETA,
 * cancel an in-flight job, and download the artifacts it produced.
 */
export function RunsPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.runs}
      aria-labelledby="runs-page-heading"
      className="flex flex-col gap-4"
    >
      <h2 id="runs-page-heading" className="text-2xl font-semibold">
        {t(strings.pages.runs.title)}
      </h2>
      <p className="text-app-muted-foreground">{t(strings.pages.runs.description)}</p>
      <RunHistory />
    </section>
  );
}
