import { useState, useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { Loader2, BookOpen } from "lucide-react";
import { fetchGlossary } from "../../lib/api";
import { useDebouncedValue } from "../../hooks/useDebouncedValue";
import { SearchInput } from "@vrooli/react-component-library/SearchInput/1";
import { Button } from "@vrooli/react-component-library/Button/2";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { i18n } from "../../i18n";

export function GlossaryPanel() {
  const [searchTerm, setSearchTerm] = useState("");
  const [activeCategory, setActiveCategory] = useState<string | null>(null);
  const searchRef = useRef<HTMLInputElement>(null);
  const { debounced: debouncedSearch, isPending: debouncing } = useDebouncedValue(searchTerm, 300);

  const { data, isLoading } = useQuery({
    queryKey: ["glossary", debouncedSearch],
    queryFn: () => fetchGlossary(debouncedSearch || undefined),
  });

  const entries = data?.entries ?? [];
  const categories = Array.from(new Set(entries.map((entry) => entry.category))).sort();
  const visibleEntries = activeCategory ? entries.filter((entry) => entry.category === activeCategory) : entries;
  return (
    <div data-testid="glossary-panel" className="glossary-surface">
      {/* Header */}
      <header className="surface-heading">
        <div>
          <p className="surface-eyebrow">{i18n.t("onboarding.glossary.eyebrow")}</p>
          <h1>{i18n.t("onboarding.glossary.heading")}</h1>
          <p className="mt-1 text-sm text-muted">
            {i18n.t("onboarding.glossary.intro")}
          </p>
        </div>
      </header>

      <div className="glossary-search">
        <SearchInput ref={searchRef} value={searchTerm} onChange={(event) => { setSearchTerm(event.target.value); }} placeholder={i18n.t("onboarding.glossary.searchPlaceholder")} aria-label={i18n.t("onboarding.glossary.searchLabel")} data-testid="glossary-search" />
        {searchTerm && <Button type="button" variant="ghost" data-testid="glossary-clear-search" aria-label={i18n.t("onboarding.glossary.clearLabel")} onClick={() => { setSearchTerm(""); searchRef.current?.focus(); }}>{i18n.t("onboarding.glossary.clear")}</Button>}
      </div>

      {categories.length > 0 && (
        <div className="glossary-categories" data-testid="glossary-categories">
          <div className="flex flex-wrap gap-2" role="group" aria-label={i18n.t("onboarding.glossary.filterLabel")}>
            <Button type="button" size="sm" variant={activeCategory === null ? "secondary" : "ghost"} aria-pressed={activeCategory === null} onClick={() => setActiveCategory(null)}>{i18n.t("onboarding.glossary.allCategories")}</Button>
            {categories.map((category) => <Button key={category} type="button" size="sm" variant={activeCategory === category ? "secondary" : "ghost"} aria-pressed={activeCategory === category} onClick={() => setActiveCategory(category)}>{category}</Button>)}
          </div>
        </div>
      )}
      <div>
        {debouncing && (
          <span className="mt-1 block text-xs text-muted" data-testid="glossary-debounce-indicator" aria-live="polite">
            {i18n.t("onboarding.glossary.updating")}
          </span>
        )}
      </div>

      {/* Results count */}
      {!isLoading && visibleEntries.length > 0 && (
        <p className="mt-3 text-xs text-muted" aria-live="polite" data-testid="glossary-count">
          {visibleEntries.length} {visibleEntries.length === 1 ? i18n.t("onboarding.glossary.term") : i18n.t("onboarding.glossary.terms")}{searchTerm ? ` ${i18n.t("onboarding.glossary.matching", { query: searchTerm })}` : ""}
        </p>
      )}

      {/* Content */}
      <div className={visibleEntries.length > 0 ? "mt-2" : "mt-6"}>
        {isLoading ? (
          <div data-testid="glossary-loading" className="flex items-center justify-center py-12 text-muted" aria-live="polite">
            <Loader2 className="h-5 w-5 animate-spin" aria-hidden="true" />
            <StatusBadge className="ml-2 text-sm">{i18n.t("onboarding.glossary.loading")}</StatusBadge>
          </div>
        ) : visibleEntries.length === 0 ? (
          <div data-testid="glossary-empty" className="flex flex-col items-center justify-center py-12 text-muted">
            <BookOpen className="h-8 w-8" aria-hidden="true" />
            <p className="mt-3 text-sm font-medium">{i18n.t("onboarding.glossary.emptyHeading")}</p>
            {searchTerm ? (
              <p className="mt-1 text-xs text-muted">
                {i18n.t("onboarding.glossary.tryDifferent")} {" "}
                <Button variant="ghost"
                  type="button"
                  onClick={() => { setSearchTerm(""); searchRef.current?.focus(); }}
                  className="text-primary underline underline-offset-2 hover:text-primary focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-focus"
                >
                  {i18n.t("onboarding.glossary.clearSearch")}
                </Button>
                .
              </p>
            ) : (
              <p className="mt-1 text-xs text-muted">
                {i18n.t("onboarding.glossary.populate")}
              </p>
            )}
          </div>
        ) : (
          <dl data-testid="glossary-list" className="glossary-list">
            {visibleEntries.map((entry) => (
              <div
                key={entry.term}
                data-testid={`glossary-entry-${entry.term}`}
                className="glossary-entry"
              >
                <dt className="flex items-baseline gap-2">
                  <span className="font-medium text-foreground">{entry.term}</span>
                  <span className="rounded bg-surface-subtle px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wider text-muted">
                    {entry.category}
                  </span>
                </dt>
                <dd className="mt-1.5 text-sm leading-relaxed text-muted">
                  {entry.description}
                </dd>
              </div>
            ))}
          </dl>
        )}
      </div>
    </div>
  );
}
