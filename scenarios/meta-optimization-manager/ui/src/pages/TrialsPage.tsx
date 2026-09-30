import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { TrialsBoard } from "../features/trials/TrialsBoard";
import { useTranslation } from "../i18n";

/** Trials page: the empirical local-model gate — coverage, trend, recent runs. */
export function TrialsPage() {
  const { t } = useTranslation();
  return (
    <section
      data-testid={selectors.pages.trials}
      aria-labelledby="trials-heading"
      className="flex flex-col gap-4"
    >
      <h2 id="trials-heading" className="text-2xl font-semibold">
        {t(strings.pages.trials.title)}
      </h2>
      <p className="text-app-muted-foreground">{t(strings.pages.trials.description)}</p>
      <TrialsBoard />
    </section>
  );
}
