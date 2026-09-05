import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { TriageBoard } from "../features/triage/TriageBoard";
import { useTranslation } from "../i18n";

/** Triage board page. */
export function TriagePage() {
  const { t } = useTranslation();
  return (
    <section
      data-testid={selectors.pages.triage}
      aria-labelledby="triage-heading"
      className="flex flex-col gap-4"
    >
      <header className="flex flex-col gap-1">
        <h2 id="triage-heading" className="text-2xl font-semibold">
          {t(strings.pages.triage.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.pages.triage.description)}</p>
      </header>
      <TriageBoard />
    </section>
  );
}
