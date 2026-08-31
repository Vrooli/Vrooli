/**
 * @libraryId react-component-library:CollectionPage
 * @displayName CollectionPage
 * @description A collection template with URL-ready filtering, data-source-owned records, and recovery states.
 * @version 1.0.7
 * @tags ["template","page","data-source","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";

/** @vrooliComponentSource react-component-library:CollectionPage */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import type { ReactNode } from "react";
import { useState } from "react";

export type CollectionPageMode = "controlled" | "uncontrolled";
const panel = {
  boxSizing: "border-box" as const,
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, 0.5rem)",
  background: "var(--color-surface, #ffffff)",
  color: "var(--color-foreground, #0f172a)",
  padding: "var(--space-md, 24px)",
  minWidth: 0,
  boxShadow: "var(--elev-raised, 0 1px 2px rgba(9, 18, 22, .06), 0 1px 3px rgba(9, 18, 22, .10))",
};
const muted = { color: "var(--color-muted-foreground, #64748b)" };
const button = {
  minHeight: 44,
  border: 0,
  borderRadius: "var(--radius-control, 0.375rem)",
  background: "var(--color-primary, #2563eb)",
  color: "var(--color-primary-foreground, #ffffff)",
  paddingInline: 16,
  font: "inherit",
  fontWeight: 700,
};
const collectionFilterStyles = `.rcl-collection-filter{box-sizing:border-box;display:flex;flex-wrap:wrap}.rcl-collection-filter input{box-sizing:border-box}.rcl-collection-filter button{box-sizing:border-box}@media (max-width:520px){.rcl-collection-filter{flex-direction:column}.rcl-collection-filter input{flex:0 1 auto !important}.rcl-collection-filter button{width:100%}}`;
type State =
  | "loading"
  | "refreshing"
  | "stale"
  | "empty"
  | "partial-error"
  | "fatal-error"
  | "offline"
  | "ready";
function StateView({ state, children }: { state: State; children: ReactNode }) {
  const strings = useStrings();
  if (state === "loading")
    return (
      <div role="status" style={{ ...panel, textAlign: "center" }}>
        {strings("templates.collection-page.loading", "Loading…")}
      </div>
    );
  if (state === "refreshing")
    return (
      <div role="status" style={panel}>
        <strong>{strings("templates.collection-page.refreshing", "Refreshing")}</strong>
        {children}
      </div>
    );
  if (state === "stale")
    return (
      <div style={{ display: "grid", gap: 12 }}>
        <span style={{ color: "var(--color-primary, #2563eb)" }}>
          {strings("templates.collection-page.showing-stale-data", "Showing stale data")}
        </span>
        {children}
      </div>
    );
  if (state === "empty")
    return (
      <div data-state="empty" style={{ ...panel, textAlign: "center" }}>
        {strings("templates.collection-page.nothing-here", "Nothing here")}
      </div>
    );
  if (state === "partial-error")
    return <div role="status">Some sections need attention{children}</div>;
  if (state === "fatal-error")
    return (
      <div role="alert" style={{ ...panel, borderColor: "var(--color-danger, #dc2626)" }}>
        {strings("templates.collection-page.unable-to-load-this-page", "Unable to load this page")}
      </div>
    );
  if (state === "offline") return <div role="status">Offline{children}</div>;
  return <>{children}</>;
}
export const CollectionPage = withClassName(function CollectionPage({
  state = "ready",
  data,
  mode,
  query,
  defaultQuery = "",
  onQueryChange,
  onFilterSubmit,
}: {
  state?: State;
  data?: { items?: string[]; bulkActions?: ReactNode; inspector?: ReactNode };
  mode?: CollectionPageMode;
  query?: string;
  defaultQuery?: string;
  onQueryChange?: (query: string) => void;
  onFilterSubmit?: (query: string) => void;
}) {
  const strings = useStrings();
  const [uncontrolledQuery, setUncontrolledQuery] = useState(defaultQuery);
  const resolvedMode: CollectionPageMode =
    mode ?? (query === undefined ? "uncontrolled" : "controlled");
  const resolvedQuery = resolvedMode === "controlled" ? (query ?? "") : uncontrolledQuery;
  const updateQuery = (next: string) => {
    if (resolvedMode === "uncontrolled") setUncontrolledQuery(next);
    onQueryChange?.(next);
  };
  return (
    <StateView state={state}>
      <StyleSheet name="collection-page-1-0-4-1" css={collectionFilterStyles} />
      <div style={{ display: "grid", gap: 16, minWidth: 0, width: "100%" }}>
        <header>
          <h1 style={{ margin: 0, fontSize: 24 }}>
            {strings("templates.collection-page.collection", "Collection")}
          </h1>
          <p style={muted}>
            {strings(
              "templates.collection-page.browse-filter-and-act-on-your-resources-p-header",
              "Browse, filter, and act on your resources.",
            )}
          </p>
        </header>
        <form
          className="rcl-collection-filter"
          role="search"
          style={{ ...panel, boxSizing: "border-box", gap: 12, width: "100%" }}
          onSubmit={(event) => {
            event.preventDefault();
            onFilterSubmit?.(resolvedQuery);
          }}
          data-collection-page-mode={resolvedMode}
        >
          <input
            data-testid="templates.collection-page"
            aria-label={strings("templates.collection-page.filter-query", "Filter query")}
            placeholder={strings(
              "templates.collection-page.search-by-name-or-status-value-resolvedquery-onc",
              "Search by name or status",
            )}
            value={resolvedQuery}
            onChange={(event) => updateQuery(event.currentTarget.value)}
            style={{
              minHeight: 44,
              flex: "1 1 220px",
              minWidth: 0,
              border: "1px solid var(--color-border, #cbd5e1)",
              borderRadius: 8,
              paddingInline: 12,
              font: "inherit",
            }}
          />
          <button data-testid="templates.collection-page" type="submit" style={button}>
            {strings("templates.collection-page.apply-filters", "Apply filters")}
          </button>
        </form>
        <ul
          aria-label={strings("templates.collection-page.collection-results", "Collection results")}
          style={{
            ...panel,
            display: "grid",
            gap: 8,
            listStyle: "none",
            margin: 0,
            minWidth: 0,
          }}
        >
          {(data?.items ?? []).map((item, index) => (
            <li
              key={item + String(index)}
              style={{
                background: "var(--color-surface-muted, #f1f5f9)",
                borderRadius: 8,
                padding: 12,
                overflowWrap: "anywhere",
              }}
            >
              {item}
            </li>
          ))}
        </ul>
        <div style={muted}>
          {data?.bulkActions ?? "Bulk actions"} · {data?.inspector ?? "Inspector"}
        </div>
      </div>
    </StateView>
  );
});
