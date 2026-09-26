import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { DependenciesCard } from "../features/dependencies/DependenciesCard";
import { useTranslation } from "../i18n";

/**
 * Dependencies page: the fleet SBOM search with ecosystem + vulnerable-only
 * filters over the continuously-reconciled corpus.
 */
export function DependenciesPage() {
  const { t } = useTranslation();

  return (
    <section data-testid={selectors.pages.dependencies} className="flex flex-col gap-4">
      <div>
        <h2 className="text-2xl font-semibold">
          {t(strings.pages.dependencies.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.pages.dependencies.description)}</p>
      </div>
      <DependenciesCard />
    </section>
  );
}
