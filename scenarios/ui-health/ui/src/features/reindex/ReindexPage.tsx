import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

export function ReindexPage() {
  const { t } = useTranslation();
  return (
    <section
      data-testid={selectors.pages.reindex}
      aria-labelledby="reindex-heading"
      className="flex flex-col gap-4"
    >
      <header className="flex flex-col gap-1">
        <h2 id="reindex-heading" className="text-2xl font-semibold tracking-tight">
          {t(strings.pages.reindex.title)}
        </h2>
        <p className="text-sm text-app-muted-foreground">{t(strings.pages.reindex.description)}</p>
      </header>
      <button
        type="button"
        className="self-start rounded-control bg-app-primary px-3 py-2 text-sm font-medium text-app-primary-foreground hover:opacity-90"
      >
        {t(strings.pages.reindex.trigger)}
      </button>
      <div className="rounded-panel border border-app-border bg-app-surface p-6 text-sm text-app-muted-foreground">
        {t(strings.pages.reindex.empty)}
      </div>
    </section>
  );
}
