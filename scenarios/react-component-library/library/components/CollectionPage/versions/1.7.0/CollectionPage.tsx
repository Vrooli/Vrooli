/**
 * @libraryId react-component-library:CollectionPage
 * @displayName CollectionPage
 * @description A route-level surface for browsing a resource collection: title and actions, search and filters, a table or card presentation, selection with bulk actions, and pagination — owning every generic state so an adopting scenario supplies only its records and its domain actions.
 * @version 1.7.0
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";

/** @vrooliComponentSource react-component-library:CollectionPage */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import type { ReactNode } from "react";
import { useEffect, useLayoutEffect, useRef, useState } from "react";

export interface CollectionPageRegions {
  header?: ReactNode;
  filters?: ReactNode;
  collection?: ReactNode;
  bulkActions?: ReactNode;
  inspector?: ReactNode;
  /** Persistent actions belonging to the selected detail, such as a reply composer. */
  inspectorActions?: ReactNode;
  inspectorNotice?: ReactNode;
  overlays?: ReactNode;
}

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
const collectionWorkspaceStyles = `
[data-rcl-collection-back] { display: none; }
@media (max-width: 55.999rem) {
  [data-rcl-collection-page][data-mobile-pane="inspector"] [data-rcl-collection-filters],
  [data-rcl-collection-page][data-mobile-pane="inspector"] [data-rcl-collection-primary],
  [data-rcl-collection-page][data-mobile-pane="collection"] [data-rcl-collection-inspector] { display: none; }
  [data-rcl-collection-page][data-mobile-pane="inspector"] [data-rcl-collection-back] { display: block; justify-self: start; }
}
[data-rcl-collection-workspace] { display: grid; gap: var(--space-md, 24px); min-inline-size: 0; align-items: start; }
[data-rcl-collection-primary], [data-rcl-collection-inspector] { display: grid; gap: var(--space-sm, 16px); min-inline-size: 0; }
[data-rcl-collection-inspector-actions] { min-inline-size: 0; padding-block-start: var(--space-xs); border-block-start: var(--border-hairline) solid var(--color-border); }
@media (min-width: 56rem) {
  [data-rcl-collection-workspace][data-has-inspector="true"] { grid-template-columns: minmax(16rem, 0.8fr) minmax(0, 1.6fr); }
}
`;
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
  regions,
  data,
  mode,
  query,
  defaultQuery = "",
  onQueryChange,
  onFilterSubmit,
  mobilePane,
  onMobilePaneChange,
  backLabel,
  detailLabel,
  detailTitle,
  filterPlacement = "page",
  gutter = "none",
}: {
  gutter?: "none" | "page";
  state?: State;
  regions?: CollectionPageRegions;
  data?: { items?: string[]; bulkActions?: ReactNode; inspector?: ReactNode };
  mode?: CollectionPageMode;
  query?: string;
  defaultQuery?: string;
  onQueryChange?: (query: string) => void;
  onFilterSubmit?: (query: string) => void;
  /** Controlled mobile navigation; omit to retain the stacked layout. */
  mobilePane?: "collection" | "inspector";
  onMobilePaneChange?: (pane: "collection" | "inspector") => void;
  backLabel?: string;
  detailLabel?: string;
  /** Visible selected-item context above the inspector on every viewport. */
  detailTitle?: string;
  /** Keep search beside its results in a split workspace. */
  filterPlacement?: "page" | "collection";
}) {
  const strings = useStrings();
  const primaryRef = useRef<HTMLDivElement>(null);
  const inspectorRef = useRef<HTMLDivElement>(null);
  const returnFocusRef = useRef<HTMLElement | null>(null);
  const previousPane = useRef<string>();
  const [narrow, setNarrow] = useState(
    () => typeof window !== "undefined" && window.matchMedia("(max-width: 55.999rem)").matches,
  );
  const hasInspector = (regions?.inspector ?? data?.inspector) != null;
  const resolvedPane = hasInspector ? mobilePane : mobilePane ? "collection" : undefined;
  useEffect(() => {
    const media = window.matchMedia("(max-width: 55.999rem)");
    const update = () => setNarrow(media.matches);
    update();
    media.addEventListener("change", update);
    return () => media.removeEventListener("change", update);
  }, []);
  useLayoutEffect(() => {
    const before = previousPane.current;
    previousPane.current = narrow ? resolvedPane : undefined;
    if (!narrow || !resolvedPane || before === resolvedPane) return;
    if (resolvedPane === "inspector") {
      const active = document.activeElement;
      if (active instanceof HTMLElement && primaryRef.current?.contains(active))
        returnFocusRef.current = active;
      inspectorRef.current?.focus();
    } else if (before === "inspector") {
      const target = returnFocusRef.current;
      if (target?.isConnected && target.getClientRects().length) target.focus();
      else primaryRef.current?.focus();
    }
  }, [narrow, resolvedPane]);
  const [uncontrolledQuery, setUncontrolledQuery] = useState(defaultQuery);
  const resolvedMode: CollectionPageMode =
    mode ?? (query === undefined ? "uncontrolled" : "controlled");
  const resolvedQuery = resolvedMode === "controlled" ? (query ?? "") : uncontrolledQuery;
  const updateQuery = (next: string) => {
    if (resolvedMode === "uncontrolled") setUncontrolledQuery(next);
    onQueryChange?.(next);
  };
  const filterContent = (
    <div data-rcl-collection-filters>
      {regions?.filters ?? (
        <form
          className="rcl-collection-filter"
          role="search"
          style={{
            ...panel,
            boxSizing: "border-box",
            gap: 12,
            width: "100%",
          }}
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
      )}
    </div>
  );
  return (
    <StateView state={state}>
      <StyleSheet
        name="collection-page-1-2"
        css={collectionFilterStyles + collectionWorkspaceStyles}
      />
      <div
        data-rcl-collection-page
        data-mobile-pane={resolvedPane}
        style={{
          display: "grid",
          gap: "var(--space-md)",
          minWidth: 0,
          width: "100%",
          boxSizing: "border-box",
          padding: gutter === "page" ? "clamp(var(--space-sm), 2vw, var(--space-xl))" : undefined,
        }}
      >
        {regions?.header ?? (
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
        )}
        {filterPlacement === "page" && filterContent}
        <div
          data-rcl-collection-workspace
          data-has-inspector={(regions?.inspector ?? data?.inspector) != null || undefined}
        >
          <div
            data-rcl-collection-primary
            ref={primaryRef}
            tabIndex={-1}
            onFocusCapture={(event) => {
              if (resolvedPane === "collection")
                returnFocusRef.current = event.target as HTMLElement;
            }}
          >
            {filterPlacement === "collection" && filterContent}
            {regions?.collection ?? (
              <ul
                aria-label={strings(
                  "templates.collection-page.collection-results",
                  "Collection results",
                )}
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
            )}
            {regions?.bulkActions ?? data?.bulkActions}
          </div>
          {(regions?.inspector ?? data?.inspector) != null && (
            <div
              data-rcl-collection-inspector
              ref={inspectorRef}
              tabIndex={-1}
              role="region"
              aria-label={
                detailLabel ??
                detailTitle ??
                strings("templates.collection-page.detail", "Selected item details")
              }
            >
              {onMobilePaneChange && (
                <button
                  type="button"
                  data-rcl-collection-back
                  style={button}
                  onClick={() => onMobilePaneChange("collection")}
                >
                  {backLabel ?? strings("templates.collection-page.back", "Back to collection")}
                </button>
              )}
              {detailTitle && (
                <h2
                  data-rcl-collection-detail-title
                  style={{ margin: 0, fontSize: 20, overflowWrap: "anywhere" }}
                >
                  {detailTitle}
                </h2>
              )}
              {regions?.inspector ?? data?.inspector}
              {regions?.inspectorNotice != null && (
                <div data-rcl-collection-inspector-notice>{regions.inspectorNotice}</div>
              )}
              {regions?.inspectorActions != null && (
                <div data-rcl-collection-inspector-actions>{regions.inspectorActions}</div>
              )}
            </div>
          )}
        </div>
      </div>
      {regions?.overlays}
    </StateView>
  );
});
