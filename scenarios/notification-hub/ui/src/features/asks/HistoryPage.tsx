import { useMemo } from "react";
import { Link, useLocation, useNavigate, useSearchParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";

import { conversationsClient } from "../../api/notifications";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { formatDeliveryTime } from "../delivery/deliveryView";
import { preserveRoutedTestMode } from "../routedTestMode";
import { optionLabel } from "./askView";

const HISTORY_LIMIT = 200;
const closedStates = new Set(["answered", "defaulted", "expired"]);
type HistoryFilter = "all" | "answered" | "defaulted" | "expired";

const matchesText = (ask: { question?: string; source?: string; state: string; answer?: string; answerLabel?: string; defaultAnswer?: string }, query: string) => {
  if (!query) return true;
  return [ask.question, ask.source, ask.state, ask.answer, ask.answerLabel, ask.defaultAnswer]
    .some((value) => value?.toLocaleLowerCase().includes(query));
};

/** Reads one bounded recipient snapshot. Filtering stays within that returned set. */
export function HistoryPage() {
  const { t } = useTranslation();
  const location = useLocation();
  const navigate = useNavigate();
  const [params, setParams] = useSearchParams();
  const rawFilter = params.get("state") ?? "all";
  const filter: HistoryFilter = ["answered", "defaulted", "expired"].includes(rawFilter)
    ? rawFilter as HistoryFilter
    : "all";
  const searchText = params.get("q") ?? "";
  const normalizedQuery = searchText.trim().toLocaleLowerCase();
  const history = useQuery({
    queryKey: ["asks", "history"],
    queryFn: async () => (await conversationsClient.listAsks({ openOnly: false, limit: HISTORY_LIMIT })).asks,
    refetchInterval: 60_000,
  });
  const rows = useMemo(() => (history.data ?? []).filter((ask) =>
    closedStates.has(ask.state) && (filter === "all" || ask.state === filter) && matchesText(ask, normalizedQuery),
  ), [filter, history.data, normalizedQuery]);
  const filters: HistoryFilter[] = ["all", "answered", "defaulted", "expired"];
  const checkedAt = history.dataUpdatedAt
    ? new Intl.DateTimeFormat("en-GB", { timeZone: "UTC", hour: "2-digit", minute: "2-digit", hour12: false }).format(history.dataUpdatedAt)
    : null;
  const hasFilters = Boolean(searchText || filter !== "all");
  const updateSearch = (value: string) => {
    const next = new URLSearchParams(params);
    if (value) next.set("q", value);
    else next.delete("q");
    setParams(next, { replace: true });
  };
  const clearFilters = () => navigate(preserveRoutedTestMode("/history", location.search), { replace: true });

  return (
    <>
    <section data-testid={selectors.pages.history} aria-labelledby="history-heading" className="mx-auto flex w-full max-w-3xl flex-col gap-4">
      <p className="text-sm text-app-muted-foreground">
        {t(strings.pages.history.title)}{checkedAt ? ` · checked ${checkedAt} UTC` : " · checking"}
      </p>
      <h2 id="history-heading" className="text-2xl font-bold tracking-tight">{normalizedQuery ? "No matching loaded requests" : t(strings.pages.history.title)}</h2>
      <div className="flex flex-col gap-2">
        <label htmlFor="history-search" className="font-semibold">
          {t("pages.history.filterLoaded", { defaultValue: "Filter loaded requests" })}
        </label>
        <input
          id="history-search"
          type="search"
          value={searchText}
          onChange={(event) => updateSearch(event.currentTarget.value)}
          className="min-h-12 w-full rounded-control border border-app-border bg-app-surface px-3 py-3 text-base text-app-foreground"
        />
      </div>
      <nav
        aria-label={t(strings.pages.history.title)}
        data-search-active={normalizedQuery ? "true" : "false"}
        className="history-state-filters flex flex-wrap gap-2"
      >
        {filters.map((value) => (
          <Link
            key={value}
            to={preserveRoutedTestMode(value === "all" ? "/history" : `/history?state=${value}`, location.search)}
            data-testid={selectors.history.filterLink({ state: value })}
            aria-current={filter === value ? "page" : undefined}
            className={`min-h-11 rounded-control border px-3 py-2 text-sm ${filter === value ? "border-app-accent bg-app-surface-muted" : "border-app-border"}`}
          >
            {t(strings.pages.history[value])}
          </Link>
        ))}
      </nav>
      {history.isPending && <p role="status">{t(strings.asks.detail.loading)}</p>}
      {history.isError && <p role="alert">{t(strings.pages.history.loadError)}</p>}
      {history.isError && history.data && <p role="status">{t(strings.asks.list.stale)}</p>}
      {!history.isPending && !history.isError && rows.length === 0 && (
        <>
          <p className="text-lg">
            {normalizedQuery
              ? t("pages.history.searchEmpty", { defaultValue: "Nothing in the latest {{count}} loaded records matches “{{query}}”.", count: HISTORY_LIMIT, query: searchText.trim() })
              : history.data.some((ask) => closedStates.has(ask.state))
                ? t(strings.pages.history.filteredEmpty)
                : t(strings.pages.history.empty)}
          </p>
          {normalizedQuery && <p className="text-sm text-app-muted-foreground">
            {t("pages.history.searchLimit", { defaultValue: "Older matching records may exist. This is not a complete account search." })}
          </p>}
        </>
      )}
      {rows.length > 0 && (
        <ul className="flex flex-col gap-2">
          {rows.map((ask) => {
            const outcomeAt = ask.state === "answered"
              ? ask.answeredAt || ask.resolvedAt
              : ask.state === "defaulted"
                ? ask.defaultAppliedAt || ask.resolvedAt
                : ask.resolvedAt;
            const time = formatDeliveryTime(outcomeAt);
            return (
              <li key={ask.id} className="rounded-panel border border-app-border p-4">
                <article className="flex flex-col gap-2">
                  <Link to={preserveRoutedTestMode(`/asks/${encodeURIComponent(ask.id)}`, location.search)} className="block rounded-sm focus-visible:outline focus-visible:outline-2">
                    <span className="block break-words font-medium">{ask.question}</span>
                  </Link>
                  <p className="text-sm">
                    <span className="font-medium">{t(strings.pages.history[ask.state as Exclude<HistoryFilter, "all">])}</span>
                    {ask.state === "answered" && ` — ${t(strings.asks.detail.answered, { option: ask.answerLabel || ask.answer })}`}
                    {ask.state === "defaulted" && ask.defaultAnswer && ` — ${optionLabel(ask, ask.defaultAnswer)}`}
                  </p>
                  {ask.state === "defaulted" && ask.reversible && ask.defaultAnswer && (
                    <p className="text-sm text-app-muted-foreground">{t(strings.pages.history.lateAnswerAvailable)}</p>
                  )}
                  {ask.state === "expired" && (
                    <p className="text-sm text-app-muted-foreground">{t(strings.pages.history.expiredNoReopen)}</p>
                  )}
                  <dl className="grid gap-x-4 gap-y-1 text-sm text-app-muted-foreground sm:grid-cols-2">
                    <div>
                      <dt className="inline font-medium">{t(strings.pages.history.source)} </dt>
                      <dd className="inline break-words">{ask.source || t(strings.pages.history.unknownSource)}</dd>
                    </div>
                    <div>
                      <dt className="inline font-medium">{t(strings.pages.history.outcomeTime)} </dt>
                      <dd className="inline">{time ? <time dateTime={outcomeAt}>{time}</time> : t(strings.pages.history.timeUnavailable)}</dd>
                    </div>
                  </dl>
                </article>
              </li>
            );
          })}
        </ul>
      )}
      {!normalizedQuery && <p className="text-sm text-app-muted-foreground">
        {t("pages.history.loadedScope", { defaultValue: "History shows up to {{count}} returned decisions. Filters apply only to these loaded results, and older decisions may be omitted.", count: HISTORY_LIMIT })}
      </p>}
    </section>
    {hasFilters && (
      <footer className="history-clear-footer mx-[-1rem] mt-4 border-y border-app-border bg-app-surface px-4 py-2">
        <button type="button" onClick={clearFilters} className="min-h-11 w-full rounded-control border border-app-border bg-app-surface px-3 py-3 text-base font-medium hover:bg-app-surface-muted focus-visible:outline focus-visible:outline-2">
          {t("pages.history.clearFilters", { defaultValue: "Clear filters" })}
        </button>
      </footer>
    )}
    </>
  );
}
