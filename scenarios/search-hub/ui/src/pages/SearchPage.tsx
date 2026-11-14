import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SearchPanel } from "../features/search/SearchPanel";
import { useTranslation } from "../i18n";

/**
 * SearchPage is the scenario's headline surface: one federated query box over
 * every registered corpus. It is the index route — the search hub's whole point
 * is "just search", so it is what an operator lands on.
 */
export function SearchPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.search}
      aria-labelledby="search-heading"
      className="flex flex-col gap-4"
    >
      <div className="flex flex-col gap-1">
        <h2 id="search-heading" className="text-2xl font-semibold">
          {t(strings.pages.search.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.pages.search.description)}</p>
      </div>
      <SearchPanel />
    </section>
  );
}
