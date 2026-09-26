import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { PlansList } from "../features/plans/PlansList";
import { useTranslation } from "../i18n";

/** Plans board page — list + create-from-template. */
export function PlansPage() {
  const { t } = useTranslation();
  return (
    <section
      data-testid={selectors.pages.plans}
      aria-labelledby="plans-heading"
      className="flex flex-col gap-4"
    >
      <header className="flex flex-col gap-1">
        <h2 id="plans-heading" className="text-2xl font-semibold">
          {t(strings.pages.plans.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.pages.plans.description)}</p>
      </header>
      <PlansList />
    </section>
  );
}
