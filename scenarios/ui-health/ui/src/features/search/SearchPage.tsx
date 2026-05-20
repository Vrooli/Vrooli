import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

export function SearchPage() {
  const { t } = useTranslation();
  return (
    <section
      data-testid={selectors.pages.search}
      aria-labelledby="search-heading"
      className="flex flex-col gap-4"
    >
      <header className="flex flex-col gap-1">
        <h2 id="search-heading" className="text-2xl font-semibold tracking-tight">
          {t(strings.pages.search.title)}
        </h2>
        <p className="text-sm text-app-muted-foreground">{t(strings.pages.search.description)}</p>
      </header>
      <input
        type="search"
        placeholder={t(strings.pages.search.placeholder)}
        aria-label={t(strings.pages.search.title)}
        className="w-full rounded-control border border-app-border bg-app-surface px-3 py-2 text-sm text-app-foreground placeholder:text-app-muted-foreground focus:outline-none focus:ring-2 focus:ring-app-focus"
      />
      <div className="rounded-panel border border-app-border bg-app-surface p-6 text-sm text-app-muted-foreground">
        {t(strings.pages.search.empty)}
      </div>
    </section>
  );
}
