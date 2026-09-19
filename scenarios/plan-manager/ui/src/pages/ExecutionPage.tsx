import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { ExecutionRunner } from "../features/execution/ExecutionRunner";
import { useTranslation } from "../i18n";

/** Execution runner page. */
export function ExecutionPage() {
  const { t } = useTranslation();
  return (
    <section
      data-testid={selectors.pages.execution}
      aria-labelledby="execution-heading"
      className="flex flex-col gap-4"
    >
      <header className="flex flex-col gap-1">
        <h2 id="execution-heading" className="text-2xl font-semibold">
          {t(strings.pages.execution.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.pages.execution.description)}</p>
      </header>
      <ExecutionRunner />
    </section>
  );
}
