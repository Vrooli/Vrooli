import { useEffect, useMemo, useState, type KeyboardEvent } from "react";
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
import { SearchInput } from "@vrooli/react-component-library/SearchInput/1.1.1";
import { SearchResults } from "@vrooli/react-component-library/SearchResults/1.0.10";

import { SearchKind, SemanticLegState, type SearchResult } from "../../api/search";
import { useTranslation } from "../../i18n";
import { Tabs } from "../primitives";
import { useWorkbenchSearch } from "./useWorkbenchSearch";
import "./search-surface.css";

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
const EMPTY_KINDS: SearchKind[] = [];
const START_KIND_FILTERS = [
  { id: "all", kinds: EMPTY_KINDS, labelKey: "workbench.search.filter.all", fallback: "All" },
  { id: "workspaces", kinds: [SearchKind.WORKSPACE], labelKey: "workbench.search.filter.workspaces", fallback: "Workspaces" },
  { id: "pages", kinds: [SearchKind.PAGE, SearchKind.PAGE_TYPE], labelKey: "workbench.search.filter.pages", fallback: "Pages" },
  { id: "assets", kinds: [SearchKind.ASSET], labelKey: "workbench.search.filter.assets", fallback: "Assets" },
  { id: "commands", kinds: [SearchKind.COMMAND], labelKey: "workbench.search.filter.commands", fallback: "Commands" },
] as const;

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
  initialKinds?: SearchKind[];
  compact?: boolean;
  onNavigate?: () => void;
  onClearKinds?: () => void;
}

/**
 * The one search surface shared by Start and the command palette. It renders
 * server-ranked groups; it never filters a client-side asset list.
 */
export function SearchSurface({
  initialQuery = "",
  initialKinds = EMPTY_KINDS,
  compact = false,
  onNavigate,
  onClearKinds,
}: SearchSurfaceProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [query, setQuery] = useState(initialQuery);
  const [kinds, setKinds] = useState<SearchKind[]>(initialKinds);
  const [ask, setAsk] = useState(false);
  const [submittedAskQuery, setSubmittedAskQuery] = useState("");
  const [active, setActive] = useState(0);
  const requestQuery = ask ? submittedAskQuery : query;
  const askEnabled = !ask || (submittedAskQuery.trim() !== "" && submittedAskQuery === query);
  // An empty Start surface is an invitation to browse, not a search request.
  // Avoid an unnecessary request (including the semantic readiness check).
  const idle = query.trim() === "" && kinds.length === 0;
  useEffect(() => {
    setKinds(initialKinds);
    setActive(0);
  }, [initialKinds]);
  const { results, response, loading, error } = useWorkbenchSearch(requestQuery, {
    kinds,
    limit: compact ? 12 : 30,
    ask,
    enabled: askEnabled && !idle,
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
  const activeFilter = START_KIND_FILTERS.find(
    (filter) => filter.kinds.length === kinds.length && filter.kinds.every((kind, index) => kinds[index] === kind),
  );
  // An empty field is an invitation, not a failed query. Start and the command
  // palette both put their own content under the field, so the surface stays
  // quiet until the person types.
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
      if (ask) {
        event.preventDefault();
        setSubmittedAskQuery(query);
        return;
      }
      const result = flat[active];
      if (result) {
        event.preventDefault();
        open(result);
      }
    }
  };

  return (
    <div data-testid="search-surface" data-compact={compact ? "true" : undefined}>
      <div className="wb-search">
        <Search aria-hidden size={16} />
        <SearchInput
          value={query}
          onChange={(event) => {
            setQuery(event.target.value);
            setActive(0);
          }}
          onKeyDown={onKeyDown}
          placeholder={
            ask
              ? t("workbench.search.askPlaceholder", {
                  defaultValue: "Ask about your indexed workbench…",
                })
              : t("workbench.search.placeholder", {
                  defaultValue: "Search pages, workspaces, states, assets, commands…",
                })
          }
          aria-label={
            ask
              ? t("workbench.search.askLabel", { defaultValue: "Ask about the workbench" })
              : t("workbench.search.label", { defaultValue: "Search the library" })
          }
          aria-controls="search-surface-results"
          aria-activedescendant={flat[active] ? `search-result-${active}` : undefined}
        />
        {loading ? (
          <span className="wb-tag">
            {ask
              ? t("workbench.search.answering", { defaultValue: "Answering" })
              : t("workbench.search.loading", { defaultValue: "Searching" })}
          </span>
        ) : null}
        <kbd className="wb-search-kbd" aria-hidden>
          {t("workbench.search.shortcut", { defaultValue: "⌘K" })}
        </kbd>
      </div>

      <div
        className="wb-search-modes"
        role="group"
        aria-label={t("workbench.search.mode", { defaultValue: "Search mode" })}
      >
        <button
          type="button"
          className="wb-btn sm quiet"
          aria-pressed={!ask}
          data-selected={!ask ? "true" : undefined}
          onClick={() => setAsk(false)}
        >
          {t("workbench.search.searchMode", { defaultValue: "Search" })}
        </button>
        <button
          type="button"
          className="wb-btn sm quiet"
          aria-pressed={ask}
          data-selected={ask ? "true" : undefined}
          onClick={() => {
            setAsk(true);
            setSubmittedAskQuery("");
          }}
        >
          {t("workbench.search.askMode", { defaultValue: "Ask" })}
        </button>
        {ask ? (
          <button
            type="button"
            className="wb-btn sm secondary"
            disabled={!query.trim() || loading}
            onClick={() => setSubmittedAskQuery(query)}
          >
            {t("workbench.search.getAnswer", { defaultValue: "Get answer" })}
          </button>
        ) : null}
        {kinds.length > 0 ? (
          <button
            type="button"
            className="wb-btn sm quiet"
            data-testid="search-clear-kind-filter"
            onClick={() => {
              setKinds([]);
              onClearKinds?.();
            }}
          >
            {t("workbench.search.clearKindFilter", { defaultValue: "All results" })}
          </button>
        ) : null}
      </div>

      {compact ? (
        <div className="wb-search-kind-filters">
          <Tabs
            label={t("workbench.search.kindFilters", { defaultValue: "Search result types" })}
            value={activeFilter?.id ?? "all"}
            onChange={(id) => {
              const next = START_KIND_FILTERS.find((filter) => filter.id === id);
              if (!next) return;
              setKinds([...next.kinds]);
              onClearKinds?.();
            }}
            items={START_KIND_FILTERS.map((filter) => ({
              id: filter.id,
              label: t(filter.labelKey, { defaultValue: filter.fallback }),
            }))}
          />
        </div>
      ) : null}

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
            defaultValue:
              "Showing the last successful index. Refresh it to include recent changes.",
          })}
        </p>
      ) : null}
      {status?.semanticLeg === SemanticLegState.DEGRADED ? (
        <p className="wb-muted wb-small" role="status">
          {t("workbench.search.semanticDegraded", {
            defaultValue: "Semantic search is unavailable. Showing keyword matches.",
          })}
        </p>
      ) : null}
      {error ? (
        <p className="wb-muted wb-small" role="alert">
          {ask
            ? t("workbench.search.askError", {
                defaultValue:
                  "An answer is unavailable. Try again, or switch to Search for indexed matches.",
              })
            : t("workbench.search.error", {
                defaultValue:
                  "Search is unavailable right now. Try again when the API is reachable.",
              })}
        </p>
      ) : null}

      {ask &&
      submittedAskQuery === query &&
      !loading &&
      !error &&
      response?.query === query &&
      response.answer ? (
        <section
          className="wb-search-answer"
          aria-label={t("workbench.search.answerTitle", {
            defaultValue: "Answer from indexed results",
          })}
        >
          <b>
            {t("workbench.search.answerTitle", { defaultValue: "Answer from indexed results" })}
          </b>
          <p>{response.answer.text}</p>
          {response.answer.citations.length > 0 ? (
            <div
              className="wb-search-sources"
              aria-label={t("workbench.search.sources", { defaultValue: "Sources" })}
            >
              {response.answer.citations.map((citation, index) => {
                const result = results.find((item) => item.id === citation.resultId);
                if (!result) return null;
                return (
                  <button
                    key={citation.resultId}
                    type="button"
                    className="wb-btn sm secondary"
                    onClick={() => open(result)}
                  >
                    {t("workbench.search.source", {
                      number: index + 1,
                      title: result.title,
                      defaultValue: "Source {{number}}: {{title}}",
                    })}
                  </button>
                );
              })}
            </div>
          ) : null}
        </section>
      ) : null}

      <div
        id="search-surface-results"
        data-testid="search-surface-results"
        className="wb-command-results"
        aria-live="polite"
        hidden={idle}
      >
        {!ask && !loading && !error && flat.length === 0 && !idle ? (
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
        {/* Keep result IDs and order from the server-derived `flat` list; an empty local query
            makes SearchResults a renderer here rather than a second filtering stage. */}
        {!idle && !ask && flat.length > 0 ? (
          <SearchResults
            query=""
            items={flat.map((result) => result.id)}
            renderItem={(id, index) => {
              const result = flat[index];
              if (!result || result.id !== id) return null;
              const Icon = KIND_ICON[result.kind] ?? Boxes;
              const startsGroup = index === 0 || flat[index - 1]?.kind !== result.kind;
              return (
                <div
                  className={
                    startsGroup && index > 0
                      ? "wb-command-group wb-command-group-start"
                      : "wb-command-group"
                  }
                >
                  {startsGroup ? (
                    <span className="wb-caption">{kindLabel(result.kind, t)}</span>
                  ) : null}
                  <button
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
                </div>
              );
            }}
          />
        ) : null}
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
