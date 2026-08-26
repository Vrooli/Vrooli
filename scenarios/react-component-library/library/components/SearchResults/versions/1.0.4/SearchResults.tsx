/**
 * @libraryId react-component-library:SearchResults
 * @displayName SearchResults
 * @description A results surface that distinguishes no matches from an empty collection and preserves query context.
 * @version 1.0.4
 * @tags ["data-display","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource data-display.search-results */
import { translate } from "../../../../hooks/useLocale/versions/1.0.1/useLocale";
import { withClassName } from "../../../../foundations/ClassMerge/versions/1.0.1/ClassMerge";

import type { ReactNode } from "react";

export type SearchResultsState = "default" | "loading" | "empty" | "error" | "offline";

export interface SearchResultsProps {
  query?: string;
  items?: string[];
  state?: SearchResultsState;
  errorMessage?: string;
  onRetry?: () => void;
  emptyMessage?: string;
  noMatchMessage?: string;
  renderItem?: (item: string, index: number) => ReactNode;
}

const styles = `
[data-rcl-search-results] { display: grid; gap: var(--space-sm); min-inline-size: 0; }
[data-rcl-search-results-header] { display: flex; align-items: baseline; justify-content: space-between; gap: var(--space-sm); flex-wrap: wrap; }
[data-rcl-search-results-title] { margin: 0; color: var(--color-foreground); font: var(--text-heading); }
[data-rcl-search-results-count] { color: var(--color-muted-foreground); font: var(--text-caption); }
[data-rcl-search-results-list] { display: grid; gap: var(--space-xs); list-style: none; margin: 0; padding: 0; }
[data-rcl-search-results-item] { min-inline-size: 0; padding: var(--space-md); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: linear-gradient(145deg, color-mix(in srgb, var(--color-surface-raised) 96%, var(--color-primary)), var(--color-surface)); color: var(--color-foreground); box-shadow: var(--elev-raised); overflow-wrap: anywhere; transition: transform 160ms ease, border-color 160ms ease, box-shadow 160ms ease; }
[data-rcl-search-results-item]:hover { transform: translateY(-1px); border-color: color-mix(in srgb, var(--color-primary) 40%, var(--color-border)); box-shadow: var(--elev-overlay); }
[data-rcl-search-results-state] { display: grid; gap: var(--space-xs); padding: var(--space-lg); border: var(--border-hairline) dashed var(--color-border); border-radius: var(--radius-panel); color: var(--color-muted-foreground); font: var(--text-body); }
[data-rcl-search-results-state] strong { color: var(--color-foreground); font: var(--text-label); }
[data-rcl-search-results-state] button { justify-self: start; min-block-size: var(--tap-target-min); border: var(--border-hairline) solid var(--color-primary); border-radius: var(--radius-control); background: var(--color-primary); color: var(--color-primary-foreground); padding-inline: var(--space-sm); font: var(--text-label); cursor: pointer; }
[data-rcl-search-results-state] button:focus-visible { outline: 3px solid color-mix(in srgb, var(--color-focus) 38%, transparent); outline-offset: 2px; }
@media (prefers-reduced-motion: reduce) { [data-rcl-search-results] *, [data-rcl-search-results] *::before, [data-rcl-search-results] *::after { animation-duration: .01ms; transition-duration: .01ms; } }
@media (forced-colors: active) { [data-rcl-search-results-item], [data-rcl-search-results-state] { border-color: CanvasText; background: Canvas; color: CanvasText; box-shadow: none; } [data-rcl-search-results-state] button { border-color: Highlight; background: Highlight; color: HighlightText; } }
`;

export const SearchResults = withClassName(function SearchResults({
  query = "",
  items = [],
  state = "default",
  errorMessage = "The results could not be loaded.",
  onRetry,
  emptyMessage = "Nothing has been added yet.",
  noMatchMessage = "No results match this search.",
  renderItem,
}: SearchResultsProps) {
  const normalizedQuery = query.trim().toLowerCase();
  const results = items.filter((item) => item.toLowerCase().includes(normalizedQuery));
  const isNoMatch = items.length > 0 && results.length === 0;

  return (
    <section
      data-rcl-search-results
      aria-label={translate("data-display.search-results.aria-label.1", "Search results")}
    >
      <style data-rcl-search-results-styles dangerouslySetInnerHTML={{ __html: styles }} />
      <header data-rcl-search-results-header>
        <h2 data-rcl-search-results-title>
          {translate("data-display.search-results.text.1", "Results")}
        </h2>
        {state === "default" ? (
          <span data-rcl-search-results-count aria-live="polite">
            {results.length} result{results.length === 1 ? "" : "s"}
          </span>
        ) : null}
      </header>
      {state === "loading" ? (
        <div data-rcl-search-results-state role="status" aria-live="polite">
          <strong>{translate("data-display.search-results.text.2", "Finding matches")}</strong>
          <span>
            {translate(
              "data-display.search-results.text.3",
              "Keeping your search context while the collection responds…",
            )}
          </span>
        </div>
      ) : state === "error" ? (
        <div data-rcl-search-results-state role="alert">
          <strong>{translate("data-display.search-results.text.4", "Results unavailable")}</strong>
          <span>{errorMessage}</span>
          {onRetry ? (
            <button data-testid="data-display.search-results" type="button" onClick={onRetry}>
              {translate("data-display.search-results.text.5", "Try again")}
            </button>
          ) : null}
        </div>
      ) : state === "offline" ? (
        <div data-rcl-search-results-state role="status" aria-live="polite">
          <strong>
            {translate("data-display.search-results.text.6", "Showing saved results")}
          </strong>
          <span>
            {translate(
              "data-display.search-results.text.7",
              "New matches will appear when the connection returns.",
            )}
          </span>
        </div>
      ) : results.length === 0 ? (
        <div data-rcl-search-results-state>
          <strong>{isNoMatch ? "No matches" : "No results yet"}</strong>
          <span>{isNoMatch ? noMatchMessage : emptyMessage}</span>
        </div>
      ) : (
        <ul data-rcl-search-results-list>
          {results.map((item, index) => (
            <li key={`${item}-${index}`} data-rcl-search-results-item>
              {renderItem ? renderItem(item, index) : item}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
});
