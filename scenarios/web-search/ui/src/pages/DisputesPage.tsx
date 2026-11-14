import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { DisputesPanel } from "../features/disputes/DisputesPanel";
import { useTranslation } from "../i18n";

/**
 * Dispute review queue page. Surfaces DISPUTED findings (sources conflict) and
 * lets an operator resolve each one — keep (back to ACTIVE) or supersede.
 */
export function DisputesPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.disputes}
      aria-labelledby="disputes-heading"
      className="flex flex-col gap-6"
    >
      <div className="flex flex-col gap-1">
        <h2 id="disputes-heading" className="text-2xl font-semibold">
          {t(strings.pages.disputes.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.pages.disputes.description)}</p>
      </div>
      <DisputesPanel />
    </section>
  );
}
