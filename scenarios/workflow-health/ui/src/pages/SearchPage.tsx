import { SlidersHorizontal } from "lucide-react";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { searchResults, uiText } from "./workflowData";

export function SearchPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.search}
      aria-labelledby="search-heading"
      className="flex flex-col gap-5"
    >
      <div>
        <h2 id="search-heading" className="text-3xl font-semibold">
          {t(strings.pages.search.title)}
        </h2>
        <p className="mt-2 max-w-3xl text-sm text-app-muted-foreground">
          {t(strings.pages.search.description)}
        </p>
      </div>

      <section className="rounded-panel border border-app-border bg-app-surface p-4">
        <div className="grid gap-3 lg:grid-cols-[1fr_220px_auto]">
          <label className="flex flex-col gap-1 text-sm font-medium text-app-muted-foreground">
            {uiText.search.query}
            <input
              data-testid={selectors.workflow.searchInput}
              className="rounded-md border border-app-border bg-app-background px-3 py-2 text-app-foreground"
              defaultValue="validate workflow assets"
            />
          </label>
          <label className="flex flex-col gap-1 text-sm font-medium text-app-muted-foreground">
            {uiText.search.type}
            <select
              data-testid={selectors.workflow.searchTypeFilter}
              className="rounded-md border border-app-border bg-app-background px-3 py-2 text-app-foreground"
              defaultValue="all"
            >
              <option value="all">{uiText.search.allLeaves}</option>
              <option value="workflow.flow">{uiText.search.flows}</option>
              <option value="workflow.test">{uiText.search.tests}</option>
              <option value="workflow.fragment">{uiText.search.fragments}</option>
            </select>
          </label>
          <button
            type="button"
            className="inline-flex items-center justify-center gap-2 rounded-md bg-app-primary px-4 py-2 text-sm font-semibold text-app-primary-foreground lg:self-end"
          >
            <SlidersHorizontal aria-hidden="true" className="h-4 w-4" />
            {uiText.search.rank}
          </button>
        </div>
      </section>

      <div className="grid gap-3">
        {searchResults.map((result) => (
          <article
            key={`${result.type}:${result.name}`}
            className="rounded-panel border border-app-border bg-app-surface p-4"
          >
            <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
              <div>
                <p className="text-xs font-semibold uppercase tracking-normal text-app-muted-foreground">
                  {result.type}
                </p>
                <h3 className="mt-1 text-lg font-semibold">{result.name}</h3>
                <p className="mt-2 text-sm text-app-muted-foreground">{result.intent}</p>
              </div>
              <div className="flex flex-wrap gap-2 text-xs font-medium">
                <span className="rounded-full bg-sky-100 px-2.5 py-1 text-sky-800">
                  {result.safety}
                </span>
                <span className="rounded-full bg-app-muted px-2.5 py-1 text-app-muted-foreground">
                  {result.rank}
                </span>
              </div>
            </div>
          </article>
        ))}
      </div>
    </section>
  );
}
