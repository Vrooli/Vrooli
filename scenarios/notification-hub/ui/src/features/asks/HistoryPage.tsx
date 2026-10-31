import { useEffect, useRef, useState } from "react";
import { Link, useLocation, useSearchParams } from "react-router-dom";
import { useInfiniteQuery } from "@tanstack/react-query";
import { Check, ChevronRight, Clock3, Search, X } from "lucide-react";

import { conversationsClient } from "../../api/notifications";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { preserveRoutedTestMode } from "../routedTestMode";
import { optionLabel } from "./askView";

const PAGE_SIZE = 50;
const states = ["all", "awaiting", "answered", "defaulted", "expired", "withdrawn"] as const;
type StateFilter = typeof states[number];
let returnPosition = { from: "", top: 0, askId: "", index: 0 };

export function HistoryPage() {
  return <RequestList />;
}

/** Shared recipient-scoped browser: the queue fixes the state to Awaiting answer. */
export function RequestList({ queue = false }: { queue?: boolean }) {
  const { t, i18n } = useTranslation();
  const location = useLocation();
  const [params, setParams] = useSearchParams();
  const rawState = params.get("state") ?? "all";
  const state: StateFilter = queue ? "awaiting" : states.includes(rawState as StateFilter) ? rawState as StateFilter : "all";
  const search = params.get("q") ?? "";
  const [query, setQuery] = useState(search.trim());
  const searchInput = useRef<HTMLInputElement>(null);
  const heading = useRef<HTMLHeadingElement>(null);
  const returnFocusAfterReload = useRef(false);
  const from = `${queue ? "/asks" : "/history"}${location.search}`;
  useEffect(() => {
    const timer = setTimeout(() => setQuery(search.trim()), 200);
    return () => clearTimeout(timer);
  }, [search]);
  const searching = query !== search.trim();
  const list = useInfiniteQuery({
    queryKey: ["asks", queue ? "open" : "history", query, state],
    initialPageParam: "",
    queryFn: async ({ pageParam, signal }) => conversationsClient.listAsks({
      limit: PAGE_SIZE, cursor: pageParam, query, stateFilter: state,
    }, { signal }),
    getNextPageParam: (page) => page.nextCursor || undefined,
    retry: false,
    staleTime: 30_000,
    refetchInterval: 60_000,
  });
  // A live state change can move a row between pages; show each identity once.
  const rows = [...new Map(list.data?.pages.flatMap((page) => page.asks).map((ask) => [ask.id, ask]) ?? []).values()];
  const { data, hasNextPage, isFetching, isError, fetchNextPage } = list;
  useEffect(() => {
    if (!data || searching || returnPosition.from !== from || !returnPosition.askId) return;
    const nodes = [...document.querySelectorAll<HTMLElement>("[data-request-id]")];
    const exact = nodes.find((node) => node.dataset.requestId === returnPosition.askId);
    // If the cache expired while reading a detail, restore the required pages
    // through their normal cursor before focusing. A resolved/moved row falls
    // back to its nearest surviving neighbor rather than trapping navigation.
    if (!exact && nodes.length <= returnPosition.index && hasNextPage) {
      if (!isFetching && !isError) void fetchNextPage();
      return;
    }
    const target = exact ?? nodes[Math.min(returnPosition.index, nodes.length - 1)] ?? heading.current;
    target?.focus({ preventScroll: true });
    document.querySelector<HTMLElement>("main")?.scrollTo({ top: returnPosition.top });
    returnPosition = { ...returnPosition, askId: "" };
  }, [data, hasNextPage, isFetching, isError, fetchNextPage, searching, from]);
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
  const clear = () => {
    setParams((current) => { const next = new URLSearchParams(current); next.delete("q"); next.delete("state"); return next; }, { replace: true });
    searchInput.current?.focus();
  };
  const reload = () => { returnFocusAfterReload.current = true; void list.refetch(); };
  const title = queue ? t(strings.asks.list.title) : t(strings.pages.history.title);
  const formatTime = (value: string | undefined) => {
    const date = value ? new Date(value) : null;
    return date && !Number.isNaN(date.valueOf())
      ? new Intl.DateTimeFormat(i18n.language, { month: "short", day: "numeric", year: "numeric", hour: "numeric", minute: "2-digit", timeZoneName: "short" }).format(date)
      : t(strings.pages.history.timeUnavailable);
  };
  const hasFilters = Boolean(search || (!queue && state !== "all"));
  const initialError = list.isError && !list.data;
  const refreshError = list.isError && Boolean(list.data) && !list.isFetchNextPageError;

  return (
    <section data-testid={queue ? selectors.pages.asks : selectors.pages.history} aria-labelledby="request-list-heading" className="mx-auto flex w-full max-w-3xl flex-col gap-5">
      <header className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
        <h2 ref={heading} id="request-list-heading" tabIndex={-1} className="text-2xl font-semibold tracking-tight">{title}</h2>
        {list.dataUpdatedAt > 0 && <p className="text-xs text-app-muted-foreground">{t("pages.history.updated", { time: new Intl.DateTimeFormat(i18n.language, { hour: "numeric", minute: "2-digit" }).format(list.dataUpdatedAt) })}</p>}
      </header>
      <div className="flex flex-col gap-3">
        <div className="flex items-center justify-between gap-3">
          <label htmlFor="request-search" className="text-sm font-medium">{t("pages.history.searchLabel")}</label>
          {hasFilters && <button type="button" onClick={clear} className="inline-flex min-h-11 items-center gap-1 text-sm text-app-muted-foreground hover:text-app-foreground"><X size={14} aria-hidden="true" />{t("pages.history.clearFilters")}</button>}
        </div>
        <div className="request-search-field flex min-h-12 items-center gap-3 rounded-control border border-app-border bg-app-surface px-3">
          <Search size={18} className="shrink-0 text-app-muted-foreground" aria-hidden="true" />
          <input ref={searchInput} id="request-search" type="search" maxLength={500} value={search} onChange={(event) => changeSearch(event.currentTarget.value)} placeholder={t("pages.history.searchHint")} className="min-h-12 min-w-0 flex-1 bg-transparent text-base text-app-foreground outline-none placeholder:text-app-muted-foreground" />
        </div>
        {!queue && <nav aria-label={t("pages.history.stateFilters")} className="history-state-filters flex flex-wrap gap-1 rounded-control border border-app-border bg-app-surface-muted p-1">
          {states.map((value) => {
            const next = new URLSearchParams(params);
            if (value === "all") next.delete("state"); else next.set("state", value);
            return <Link key={value} to={`/history${next.size ? `?${next}` : ""}`} data-testid={selectors.history.filterLink({ state: value })} aria-current={state === value ? "page" : undefined} className={`flex min-h-11 min-w-11 items-center justify-center rounded-control px-3 text-sm font-medium focus-visible:outline focus-visible:outline-2 ${state === value ? "bg-app-primary text-app-primary-foreground shadow-sm" : "text-app-muted-foreground hover:text-app-foreground"}`}>{t(`pages.history.${value}`)}</Link>;
          })}
        </nav>}
        {queue && <Link to={preserveRoutedTestMode(`/history${search ? `?q=${encodeURIComponent(search)}` : ""}`, location.search)} className="self-start text-sm font-medium underline underline-offset-4">{t("pages.history.searchAllStates")}</Link>}
      </div>
      {(list.isPending || searching) && <p role="status" className="text-sm text-app-muted-foreground">{t("pages.history.searching")}</p>}
      {(initialError || refreshError) && <div className="rounded-control border border-app-border bg-app-surface p-4">
        <p role="alert">{queue ? t(strings.asks.list.loadError) : t(strings.pages.history.loadError)}</p>
        {refreshError && <p role="status" className="mt-1 text-sm text-app-muted-foreground">{t(strings.asks.list.stale)}</p>}
        {initialError && <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.asks.list.notCleared)}</p>}
        <div className="mt-2 flex flex-wrap items-center gap-4">
          <button type="button" onClick={reload} className="min-h-11 font-medium underline underline-offset-4">{t(strings.asks.list.retry)}</button>
          <Link to={preserveRoutedTestMode("/settings", location.search)} className="inline-flex min-h-11 items-center text-sm underline">{t(strings.asks.list.openSettings)}</Link>
        </div>
      </div>}
      {list.isSuccess && !searching && rows.length === 0 && <div data-testid={queue ? selectors.asks.empty : undefined} className="rounded-control border border-app-border bg-app-surface px-5 py-8">
        <Clock3 size={24} className="mb-3 text-app-muted-foreground" aria-hidden="true" />
        <h3 className="text-lg font-medium">{t(hasFilters ? "pages.history.noMatches" : queue ? "pages.history.noAwaiting" : "pages.history.empty")}</h3>
        <p className="mt-2 text-sm text-app-muted-foreground">{t(hasFilters ? "pages.history.trySearch" : queue ? "pages.history.awaitingEmptyDetail" : "pages.history.emptyDetail")}</p>
      </div>}
      {rows.length > 0 && <ul data-testid={queue ? selectors.asks.list : undefined} aria-busy={searching || list.isFetching} className="divide-y divide-app-border border-y border-app-border">
        {rows.map((ask, index) => {
          const awaiting = ask.state === "pending" || ask.state === "escalated";
          const stateKey: StateFilter | "unknownState" = awaiting ? "awaiting" : states.includes(ask.state as StateFilter) ? ask.state as StateFilter : "unknownState";
          const outcomeAt = ask.answeredAt || ask.resolvedAt || ask.defaultAppliedAt || ask.createdAt;
          const summary = ask.title && ask.title !== "Decision required" ? ask.title : ask.question;
          const answerSummary = ask.state === "answered" ? ask.answerLabel || (ask.answer ? optionLabel(ask, ask.answer) : "") : "";
          return <li key={ask.id} data-testid={queue ? selectors.asks.item : undefined}>
            <Link data-request-id={ask.id} to={preserveRoutedTestMode(`/asks/${encodeURIComponent(ask.id)}`, location.search)} state={{ from }} onClick={() => { returnPosition = { from, askId: ask.id, index, top: document.querySelector<HTMLElement>("main")?.scrollTop ?? window.scrollY }; }} className="group relative flex min-h-11 items-start gap-3 rounded-sm py-3 focus-visible:outline focus-visible:outline-2">
              <div className="min-w-0 flex-1">
                <p className="line-clamp-2 break-words text-base font-medium leading-snug">{summary}</p>
                {summary !== ask.question && <span className="sr-only">{ask.question}</span>}
                <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-app-muted-foreground">
                  <span className="inline-flex max-w-full flex-wrap items-center gap-1 rounded bg-app-surface-muted px-1.5 py-1 font-medium text-app-foreground">
                    {ask.state === "answered" ? <Check size={12} aria-hidden="true" /> : awaiting ? <Clock3 size={12} aria-hidden="true" /> : null}{t(`pages.history.${stateKey}`)}
                    {ask.state === "escalated" && <span> · {t("pages.history.escalated")}</span>}
                    {answerSummary && <span className="min-w-0 max-w-full [overflow-wrap:anywhere]"> · {t("pages.history.answerSummary", { answer: answerSummary })}</span>}
                  </span>
                  <span className="break-all">{ask.source || t(strings.pages.history.unknownSource)}</span>
                  <time dateTime={outcomeAt || undefined}>{formatTime(outcomeAt)}</time>
                </div>
                {ask.state === "defaulted" && ask.reversible && ask.defaultAnswer && <p className="mt-2 text-sm text-app-muted-foreground">{t(strings.pages.history.lateAnswerAvailable)}</p>}
                {ask.state === "expired" && <span className="sr-only">{t(strings.pages.history.expiredNoReopen)}</span>}
              </div>
              <ChevronRight size={18} className="mt-1 shrink-0 text-app-muted-foreground rtl:rotate-180" aria-hidden="true" />
            </Link>
          </li>;
        })}
      </ul>}
      {list.isFetchNextPageError && <p role="alert" className="text-sm">{t("pages.history.moreError")}</p>}
      {list.hasNextPage && <button type="button" disabled={list.isFetching || searching} onClick={() => void list.fetchNextPage()} className="min-h-11 rounded-control border border-app-border bg-app-surface px-4 py-3 text-sm font-medium disabled:opacity-60">{t(list.isFetchingNextPage ? "pages.history.loadingMore" : list.isFetchNextPageError ? "pages.history.retryMore" : "pages.history.loadMore")}</button>}
      {list.isFetchingNextPage && <p role="status" className="text-sm text-app-muted-foreground">{t("pages.history.loadingMore")}</p>}
      {list.isSuccess && rows.length > 0 && !searching && <p role="status" className="text-center text-xs text-app-muted-foreground">{t(list.hasNextPage ? "pages.history.loadedCount" : "pages.history.end", { count: rows.length })}</p>}
    </section>
  );
}
