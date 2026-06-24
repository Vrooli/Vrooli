import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { ConvergenceBoard } from "../features/convergence/ConvergenceBoard";
import { useTranslation } from "../i18n";

/** Convergence page: per-template fitness + reference-scenario health. */
export function ConvergencePage() {
  const { t } = useTranslation();
  return (
    <section
      data-testid={selectors.pages.convergence}
      aria-labelledby="convergence-heading"
      className="flex flex-col gap-4"
    >
      <h2 id="convergence-heading" className="text-2xl font-semibold">
        {t(strings.pages.convergence.title)}
      </h2>
      <p className="text-app-muted-foreground">{t(strings.pages.convergence.description)}</p>
      <ConvergenceBoard />
    </section>
  );
}
