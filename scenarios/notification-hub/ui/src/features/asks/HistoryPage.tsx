import { useEffect, useRef, useState } from "react";
import { Link, useLocation, useSearchParams } from "react-router-dom";
import { keepPreviousData, useInfiniteQuery } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";
import { IconButton } from "@vrooli/react-component-library/IconButton/3";
import { CheckCircle2, Inbox, Search, SearchX, X } from "lucide-react";

import { conversationsClient } from "../../api/notifications";
import { FilterChips } from "../../components/FilterChips";
import { ListGroup } from "../../components/ListRow";
import { Page } from "../../components/Page";
import { EmptyNotice, InlineNote, QueryError } from "../../components/QueryNotice";
import { SkeletonRows } from "../../components/Skeleton";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useNow } from "../../hooks/useNow";
import { useNeedsYou } from "./useNeedsYou";
import { useTranslation } from "../../i18n";
import { isUnauthenticated } from "../../lib/apiError";
import { formatAgo } from "../../lib/ago";
import { preserveRoutedTestMode } from "../routedTestMode";
import { filterCount, groupByTier, listViewRequest, readListView, TIER_LABEL, writeListView, type ListView } from "./askListView";
import { parseStateFilter, STATE_FILTERS, STATE_LABEL } from "./askState";
import { DecisionRow } from "./DecisionRow";
import { ListControls } from "./ListControls";
import { NativeConsents } from "./NativeConsents";
import { useNativeConsents } from "./useNativeConsents";

const PAGE_SIZE = 50;
const HEADING_ID = "request-list-heading";
let returnPosition = { from: "", top: 0, askId: "", index: 0 };

export function HistoryPage() {
  return <RequestList />;
}

/**
 * Shared decision browser. The queue ("Needs you") fixes the status to waiting
 * and keeps search behind an icon; History keeps search open and offers every
 * status. Both search the full authorized scope on the server before paging.
 */
export function RequestList({ queue = false }: { queue?: boolean }) {
  const { t } = useTranslation();
  const location = useLocation();
  const [params, setParams] = useSearchParams();
  const state = queue ? "awaiting" : parseStateFilter(params.get("state"));
  const search = params.get("q") ?? "";
  const view = readListView(params, queue);
  const [query, setQuery] = useState(search.trim());
  const [searchOpen, setSearchOpen] = useState(false);
  const searchInput = useRef<HTMLInputElement>(null);
  const returnFocusAfterReload = useRef(false);
  const from = `${queue ? "/asks" : "/history"}${location.search}`;
  const showSearch = !queue || searchOpen || search !== "";
  useEffect(() => {
    if (search.trim() === query) return;
    const timer = setTimeout(() => setQuery(search.trim()), 200);
    return () => clearTimeout(timer);
  }, [search, query]);
  useEffect(() => {
    if (queue && searchOpen) searchInput.current?.focus();
  }, [queue, searchOpen]);
  const searching = query !== search.trim();
  const list = useInfiniteQuery({
    queryKey: ["asks", queue ? "open" : "history", query, state, view],
    initialPageParam: "",
    queryFn: async ({ pageParam, signal }) => conversationsClient.listAsks({
      limit: PAGE_SIZE, cursor: pageParam, query, stateFilter: state,
      ...listViewRequest(view, queue),
    }, { signal }),
    // Changing a filter keeps the last answer's facets on screen, so the open
    // filter sheet does not vanish under the finger that just touched it.
    placeholderData: keepPreviousData,
    getNextPageParam: (page) => page.nextCursor || undefined,
    retry: false,
    staleTime: 30_000,
    refetchInterval: 60_000,
  });
  // While a new search or filter is being read, the previous answer stays under
  // the controls (their facets) but its rows are not shown as the new result.
  const answered = list.isPlaceholderData ? undefined : list.data;
  const facets = list.data?.pages[0]?.facets;
  // A live state change can move a row between pages; show each identity once.
  const rows = [...new Map(answered?.pages.flatMap((page) => page.asks).map((ask) => [ask.id, ask]) ?? []).values()];
  const data = answered;
  const { hasNextPage, isFetching, isError, fetchNextPage } = list;
  // Coming back from a decision: the position this list remembered when the row
  // was opened, or else the row the decision page asked to have restored.
  const restoreAskId = (location.state as { restoreAskId?: string } | null)?.restoreAskId;
  const restoredAskId = useRef("");
  useEffect(() => {
    const remembered = returnPosition.from === from && returnPosition.askId ? returnPosition : null;
    const asked = !remembered && restoreAskId && restoredAskId.current !== restoreAskId ? { from, askId: restoreAskId, index: 0, top: -1 } : null;
    const restoring = remembered ?? asked;
    if (!data || searching || !restoring) return;
    const nodes = [...document.querySelectorAll<HTMLElement>("[data-request-id]")];
    const exact = nodes.find((node) => node.dataset.requestId === restoring.askId);
    // If the cache expired while reading a detail, restore the required pages
    // through their normal cursor before focusing. A resolved/moved row falls
    // back to its nearest surviving neighbor rather than trapping navigation.
    if (!exact && nodes.length <= restoring.index && hasNextPage) {
      if (!isFetching && !isError) void fetchNextPage();
      return;
    }
    const heading = document.getElementById(HEADING_ID);
    heading?.setAttribute("tabindex", "-1");
    const target = exact ?? nodes[Math.min(restoring.index, nodes.length - 1)] ?? heading;
    target?.focus({ preventScroll: true });
    if (restoring.top >= 0) document.querySelector<HTMLElement>("main")?.scrollTo({ top: restoring.top });
    else target?.scrollIntoView({ block: "center" });
    restoredAskId.current = restoreAskId ?? "";
    returnPosition = { ...returnPosition, askId: "" };
  }, [data, hasNextPage, isFetching, isError, fetchNextPage, searching, from, restoreAskId]);
  useEffect(() => {
    if (!returnFocusAfterReload.current || list.isFetching || !list.isSuccess) return;
    returnFocusAfterReload.current = false;
    searchInput.current?.focus({ preventScroll: true });
  }, [list.isFetching, list.isSuccess]);
  const changeSearch = (value: string) => setParams((current) => {
    const next = new URLSearchParams(current);
    if (value) next.set("q", value); else next.delete("q");
    return next;
  }, { replace: true });
  const changeView = (next: ListView) => setParams((current) => writeListView(current, next, queue), { replace: true });
  const clear = () => {
    setParams((current) => {
      const next = writeListView(current, { ...view, urgencies: [], senders: [] }, queue);
      next.delete("q");
      next.delete("state");
      return next;
    }, { replace: true });
    searchInput.current?.focus();
  };
  // The decisions the search and filters leave, across every page: the urgency
  // counts already apply the sender filter and the search, so selected urgencies
  // (or all of them, when none is selected) add up to it.
  const matching = (facets?.urgencies ?? [])
    .filter((facet) => view.urgencies.length === 0 || view.urgencies.some((urgency) => urgency === facet.value))
    .reduce((sum, facet) => sum + facet.count, 0);
  const consents = useNativeConsents(query, queue);
  const consentsFailed = queue && consents.isError && !isUnauthenticated(consents.error);
  const reload = () => {
    returnFocusAfterReload.current = true;
    if (initialError || refreshError) void list.refetch();
    if (consentsFailed) void consents.refetch();
  };
  const hasFilters = Boolean(search || (!queue && state !== "all") || filterCount(view) > 0);
  const initialError = list.isError && !list.data;
  const refreshError = list.isError && Boolean(list.data) && !list.isFetchNextPageError;
  const settled = list.isSuccess && !list.isPlaceholderData && !searching;
  const needsYou = useNeedsYou();
  const totalAwaiting = answered?.pages[0]?.totalAwaiting ?? 0;
  const description = queue && settled
    ? hasFilters
      ? t(strings.asks.list.matching, { count: matching, total: totalAwaiting })
      : rows.length === 0 && needsYou.count === 0
        ? t(strings.asks.list.allClear)
        : t(strings.asks.list.waiting, { count: Math.max(rows.length, needsYou.count) })
    : undefined;
  const pending = list.isPending || list.isPlaceholderData || searching;
  // Ordered by attention the list reads as groups (escalated, then each urgency),
  // which is what makes the order explain itself; one group needs no heading.
  const groups = view.sort === "attention" ? groupByTier(rows) : [{ tier: "normal" as const, asks: rows }];
  const grouped = view.sort === "attention" && groups.length > 1;
  const now = useNow();
  const updatedAgo = list.dataUpdatedAt > 0 ? formatAgo(new Date(list.dataUpdatedAt).toISOString(), now) : null;

  const searchField = showSearch && (
    <div className="request-search-field flex min-h-12 items-center gap-3 rounded-control border border-app-border bg-app-surface px-3">
      <Search size={18} className="shrink-0 text-app-muted-foreground" aria-hidden="true" />
      <input
        ref={searchInput}
        id="request-search"
        type="search"
        maxLength={500}
        aria-label={t(strings.pages.history.searchLabel)}
        value={search}
        onChange={(event) => changeSearch(event.currentTarget.value)}
        placeholder={t(strings.pages.history.searchHint)}
        className="min-h-12 min-w-0 flex-1 bg-transparent text-base text-app-foreground outline-none placeholder:text-app-muted-foreground"
      />
      {hasFilters && (
        <button type="button" onClick={clear} className="inline-flex min-h-11 shrink-0 items-center gap-1 rounded-control px-1 text-sm text-app-muted-foreground hover:text-app-foreground">
          <X size={14} aria-hidden="true" />{t(strings.pages.history.clearFilters)}
        </button>
      )}
    </div>
  );

  return (
    <Page
      testId={queue ? selectors.pages.asks : selectors.pages.history}
      headingId={HEADING_ID}
      title={queue ? t(strings.asks.list.title) : t(strings.pages.history.title)}
      description={description}
      actions={queue && (
        <IconButton
          aria-label={t(searchOpen || search ? strings.asks.list.searchClose : strings.asks.list.searchOpen)}
          aria-pressed={searchOpen || search !== ""}
          surface="ghost"
          shape="rounded"
          size="lg"
          onClick={() => {
            if (searchOpen || search) { setSearchOpen(false); changeSearch(""); } else setSearchOpen(true);
          }}
        >
          <Search size={20} aria-hidden="true" />
        </IconButton>
      )}
    >
      {searchField}
      {!queue && (
        <FilterChips
          testId={selectors.history.filterStrip}
          label={t(strings.pages.history.stateFilters)}
          chips={STATE_FILTERS.map((value) => {
            const next = new URLSearchParams(params);
            if (value === "all") next.delete("state"); else next.set("state", value);
            return {
              id: value,
              label: t(STATE_LABEL[value]),
              current: state === value,
              testId: selectors.history.filterLink({ state: value }),
              to: preserveRoutedTestMode(`/history${next.size ? `?${next}` : ""}`, location.search),
            };
          })}
        />
      )}
      {queue && <NativeConsents list={consents} />}
      <ListControls view={view} facets={facets} matching={matching} onChange={changeView} onClear={clear} />
      {pending && <SkeletonRows />}
      {(initialError || refreshError || consentsFailed) && (
        <QueryError
          error={initialError || refreshError ? list.error : consents.error}
          title={t(initialError || refreshError
            ? consentsFailed ? strings.asks.list.loadErrorBoth : queue ? strings.asks.list.loadError : strings.pages.history.loadError
            : strings.nativeConsent.unavailable)}
          detail={initialError || refreshError ? (refreshError ? t(strings.asks.list.stale) : t(strings.asks.list.notCleared)) : undefined}
          onRetry={reload}
          retrying={list.isFetching || consents.isFetching}
          retryTestId={selectors.asks.loadRetry}
        />
      )}
      {settled && rows.length === 0 && (
        <div data-testid={queue ? selectors.asks.empty : undefined}>
          <EmptyNotice
            icon={hasFilters ? <SearchX size={32} /> : queue ? <CheckCircle2 size={32} /> : <Inbox size={32} />}
            title={t(hasFilters ? strings.pages.history.noMatches : queue ? strings.pages.history.noAwaiting : strings.pages.history.empty)}
            detail={t(hasFilters ? strings.pages.history.trySearch : queue ? strings.pages.history.awaitingEmptyDetail : strings.pages.history.emptyDetail)}
          >
            {queue && !hasFilters && (
              <Link to={preserveRoutedTestMode("/history", location.search)} className="inline-flex min-h-11 items-center text-sm font-medium text-app-primary hover:underline">
                {t(strings.pages.history.seeHistory)}
              </Link>
            )}
            {hasFilters && <Button type="button" variant="outline" onClick={clear}>{t(strings.pages.history.clearFilters)}</Button>}
          </EmptyNotice>
        </div>
      )}
      {rows.length > 0 && (
        <div data-testid={queue ? selectors.asks.list : undefined} aria-busy={searching || list.isFetching} className="flex flex-col gap-4">
          {groups.map(({ tier, asks }) => {
            const list = (
              <ListGroup key={tier}>
                {asks.map((ask) => (
                  <DecisionRow
                    key={ask.id}
                    ask={ask}
                    queue={queue}
                    search={location.search}
                    from={from}
                    onOpen={() => { returnPosition = { from, askId: ask.id, index: rows.indexOf(ask), top: document.querySelector<HTMLElement>("main")?.scrollTop ?? window.scrollY }; }}
                  />
                ))}
              </ListGroup>
            );
            return grouped ? (
              <section key={tier} data-testid={selectors.asks.group} aria-labelledby={`${HEADING_ID}-${tier}`}>
                <h2 id={`${HEADING_ID}-${tier}`} className="mb-2 text-sm font-semibold text-app-muted-foreground">{t(TIER_LABEL[tier])}</h2>
                {list}
              </section>
            ) : list;
          })}
        </div>
      )}
      {list.isFetchNextPageError && <p role="alert" className="text-sm">{t(strings.pages.history.moreError)}</p>}
      {list.hasNextPage && (
        <Button type="button" variant="outline" disabled={list.isFetching || searching} onClick={() => void list.fetchNextPage()}>
          {t(list.isFetchingNextPage ? strings.pages.history.loadingMore : list.isFetchNextPageError ? strings.pages.history.retryMore : strings.pages.history.loadMore)}
        </Button>
      )}
      {list.isFetchingNextPage && <InlineNote>{t(strings.pages.history.loadingMore)}</InlineNote>}
      {settled && rows.length > 0 && (
        <footer className="flex flex-wrap items-center justify-between gap-x-4 gap-y-1 text-xs text-app-muted-foreground">
          <p role="status">{t(list.hasNextPage ? strings.pages.history.loadedCount : strings.pages.history.end, { count: rows.length })}</p>
          {updatedAgo && <p>{t(strings.pages.history.updated, { time: updatedAgo })}</p>}
        </footer>
      )}
      {queue && !pending && !initialError && (
        <Link to={preserveRoutedTestMode(`/history${search ? `?q=${encodeURIComponent(search)}` : ""}`, location.search)} className="inline-flex min-h-11 w-fit items-center text-sm font-medium text-app-primary hover:underline">
          {t(strings.pages.history.searchAllStates)}
        </Link>
      )}
    </Page>
  );
}
