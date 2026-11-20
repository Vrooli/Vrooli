import { useMemo, useState, type KeyboardEvent } from "react";
import type { TFunction } from "i18next";
import {
  ArrowRight,
  Boxes,
  Command,
  FileText,
  Layers,
  LayoutTemplate,
  Search,
  ShieldCheck,
  Smartphone,
} from "lucide-react";
import { useNavigate } from "react-router-dom";

import { SearchKind, type SearchResult } from "../../api/search";
import { useTranslation } from "../../i18n";
import { useWorkbenchSearch } from "./useWorkbenchSearch";

const KIND_ORDER: SearchKind[] = [
  SearchKind.WORKSPACE,
  SearchKind.PAGE,
  SearchKind.DISPLAY,
  SearchKind.STATE,
  SearchKind.PAGE_TYPE,
  SearchKind.ASSET,
  SearchKind.COMMAND,
  SearchKind.EVIDENCE,
];

const KIND_ICON: Partial<Record<SearchKind, typeof Search>> = {
  [SearchKind.WORKSPACE]: Boxes,
  [SearchKind.PAGE]: FileText,
  [SearchKind.DISPLAY]: Smartphone,
  [SearchKind.STATE]: Layers,
  [SearchKind.PAGE_TYPE]: LayoutTemplate,
  [SearchKind.ASSET]: Boxes,
  [SearchKind.COMMAND]: Command,
  [SearchKind.EVIDENCE]: ShieldCheck,
};

export interface SearchSurfaceProps {
  initialQuery?: string;
  compact?: boolean;
  onNavigate?: () => void;
}

/**
 * The one search surface shared by Start and the command palette. It renders
 * server-ranked groups; it never filters a client-side asset list.
 */
export function SearchSurface({
  initialQuery = "",
  compact = false,
  onNavigate,
}: SearchSurfaceProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [query, setQuery] = useState(initialQuery);
  const [active, setActive] = useState(0);
  const { results, response, loading, error } = useWorkbenchSearch(query, {
    limit: compact ? 12 : 30,
  });

  const groups = useMemo(() => {
    const grouped = new Map<SearchKind, SearchResult[]>();
    for (const kind of KIND_ORDER) {
      const rows = results.filter((result) => result.kind === kind);
      if (rows.length > 0) grouped.set(kind, rows);
    }
    return grouped;
  }, [results]);

  const flat = useMemo(() => [...groups.values()].flat(), [groups]);
  const status = response?.status;
  // An empty field is an invitation, not a failed query. Start and the command
  // palette both put their own content under the field, so the surface stays
  // quiet until the person types.
  const idle = query.trim() === "";

  const open = (result: SearchResult) => {
    onNavigate?.();
    void navigate(result.href || "/");
  };

  const onKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "ArrowDown") {
      event.preventDefault();
      setActive((value) => Math.min(value + 1, Math.max(flat.length - 1, 0)));
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      setActive((value) => Math.max(value - 1, 0));
    } else if (event.key === "Enter") {
      const result = flat[active];
      if (result) {
        event.preventDefault();
        open(result);
      }
    }
  };

  return (
    <div data-testid="search-surface" data-compact={compact ? "true" : undefined}>
      <label className="wb-search">
        <Search aria-hidden size={16} />
        <input
          value={query}
          onChange={(event) => {
            setQuery(event.target.value);
            setActive(0);
          }}
          onKeyDown={onKeyDown}
          placeholder={t("workbench.search.placeholder", {
            defaultValue: "Search pages, workspaces, states, assets, commands…",
          })}
          aria-label={t("workbench.search.label", { defaultValue: "Search the library" })}
          aria-controls="search-surface-results"
          aria-activedescendant={flat[active] ? `search-result-${active}` : undefined}
        />
        {loading ? <span className="wb-tag">{t("workbench.search.loading", { defaultValue: "Searching" })}</span> : null}
        <kbd className="wb-search-kbd" aria-hidden>
          {t("workbench.search.shortcut", { defaultValue: "⌘K" })}
        </kbd>
      </label>

      {status && !status.indexed ? (
        <p className="wb-muted wb-small" role="status">
          {t("workbench.search.indexEmpty", {
            defaultValue: "The index is empty. Run Reindex from Settings or the CLI.",
          })}
        </p>
      ) : null}
      {status?.stale ? (
        <p className="wb-muted wb-small" role="status">
          {t("workbench.search.stale", {
            defaultValue: "Showing the last successful index. Refresh it to include recent changes.",
          })}
        </p>
      ) : null}
      {error ? (
        <p className="wb-muted wb-small" role="alert">
          {t("workbench.search.error", {
            defaultValue: "Search is unavailable right now. Try again when the API is reachable.",
          })}
        </p>
      ) : null}

      <div
        id="search-surface-results"
        className="wb-command-results"
        aria-live="polite"
        hidden={idle}
      >
        {!loading && !error && flat.length === 0 && !idle ? (
          <div className="wb-empty">
            <b className="wb-empty-title">
              {t("workbench.search.noMatch", { defaultValue: "No matches" })}
            </b>
            <p>
              {t("workbench.search.noMatchHint", {
                defaultValue: "Try an outcome such as “landing page” or “onboarding flow”.",
              })}
            </p>
          </div>
        ) : null}
        {!idle && [...groups.entries()].map(([kind, rows]) => {
          const Icon = KIND_ICON[kind] ?? Boxes;
          return (
            <section key={kind} className="wb-command-group" aria-label={kindLabel(kind, t)}>
              <span className="wb-caption">{kindLabel(kind, t)}</span>
              {rows.map((result) => {
                const index = flat.indexOf(result);
                return (
                  <button
                    key={`${result.kind}-${result.id}`}
                    id={`search-result-${index}`}
                    type="button"
                    className="wb-command-row"
                    data-active={index === active ? "true" : undefined}
                    onMouseEnter={() => setActive(index)}
                    onClick={() => open(result)}
                  >
                    <span className="wb-command-icon" aria-hidden>
                      <Icon size={14} />
                    </span>
                    <span style={{ minWidth: 0, flex: 1 }}>
                      <b>{result.title}</b>
                      <small>{result.subtitle || result.snippet}</small>
                    </span>
                    {result.badges.slice(0, 2).map((badge) => (
                      <span key={badge} className="wb-tag">
                        {badge}
                      </span>
                    ))}
                    <ArrowRight aria-hidden size={13} />
                  </button>
                );
              })}
            </section>
          );
        })}
      </div>
    </div>
  );
}

function kindLabel(kind: SearchKind, t: TFunction): string {
  switch (kind) {
    case SearchKind.WORKSPACE:
      return t("workbench.search.kind.workspace", { defaultValue: "Workspaces" });
    case SearchKind.PAGE:
      return t("workbench.search.kind.page", { defaultValue: "Pages" });
    case SearchKind.DISPLAY:
      return t("workbench.search.kind.display", { defaultValue: "Displays" });
    case SearchKind.STATE:
      return t("workbench.search.kind.state", { defaultValue: "States" });
    case SearchKind.PAGE_TYPE:
      return t("workbench.search.kind.pageType", { defaultValue: "Page types" });
    case SearchKind.ASSET:
      return t("workbench.search.kind.asset", { defaultValue: "Assets" });
    case SearchKind.COMMAND:
      return t("workbench.search.kind.command", { defaultValue: "Commands" });
    case SearchKind.EVIDENCE:
      return t("workbench.search.kind.evidence", { defaultValue: "Evidence" });
    default:
      return t("workbench.search.kind.result", { defaultValue: "Results" });
  }
}
