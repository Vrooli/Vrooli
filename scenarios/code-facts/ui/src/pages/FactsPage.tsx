import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { FactsWorkbench } from "../features/facts/FactsWorkbench";
import { useTranslation } from "../i18n";

export function FactsPage() {
  const { t } = useTranslation();

  return (
    <section data-testid={selectors.pages.facts} aria-labelledby="facts-heading" className="flex flex-col gap-4">
      <h2 id="facts-heading" className="text-2xl font-semibold">
        {t(strings.pages.facts.title)}
      </h2>
      <p className="text-app-muted-foreground">{t(strings.pages.facts.description)}</p>
      <FactsWorkbench />
    </section>
  );
}
