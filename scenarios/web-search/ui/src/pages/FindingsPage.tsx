import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { FindingsPanel } from "../features/findings/FindingsPanel";
import { useTranslation } from "../i18n";

/**
 * Findings management page. Lists the citation-backed knowledge store filtered
 * by status, with add / edit / supersede / flag / prune actions.
 */
export function FindingsPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.findings}
      aria-labelledby="findings-heading"
      className="flex flex-col gap-6"
    >
      <div className="flex flex-col gap-1">
        <h2 id="findings-heading" className="text-2xl font-semibold">
          {t(strings.pages.findings.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.pages.findings.description)}</p>
      </div>
      <FindingsPanel />
    </section>
  );
}
