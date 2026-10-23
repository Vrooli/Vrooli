import { useEffect, useMemo, useRef } from "react";
import { Link, useLocation, useSearchParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";

import { conversationsClient } from "../../api/notifications";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { optionLabel } from "./askView";
import { preserveRoutedTestMode } from "../routedTestMode";

const ASK_LIMIT = 200;
let returnPosition = { top: 0, askId: "", index: 0 };

/** Bounded recipient queue; the URL owns the filter so Back restores it. */
export function AsksPage() {
  return <AsksQueue />;
}

function AsksQueue() {
  const { t } = useTranslation();
  const location = useLocation();
  const [searchParams, setSearchParams] = useSearchParams();
  const filter = searchParams.get("q") ?? "";
  const heading = useRef<HTMLHeadingElement>(null);
  const asks = useQuery({
    queryKey: ["asks", "open"],
    queryFn: async () =>
      (await conversationsClient.listAsks({ openOnly: true, limit: ASK_LIMIT })).asks,
    refetchInterval: 60_000,
  });
  const visible = useMemo(() => {
    const query = filter.trim().toLocaleLowerCase();
    if (!query) return asks.data ?? [];
    return (asks.data ?? []).filter((ask) =>
      `${ask.question} ${ask.recommended ? optionLabel(ask, ask.recommended) : ""}`
        .toLocaleLowerCase().includes(query),
    );
  }, [asks.data, filter]);

  useEffect(() => {
    const restoreAskId = location.state?.restoreAskId || returnPosition.askId;
    if (restoreAskId) {
      const node = Array.from(document.querySelectorAll<HTMLElement>("[data-ask-id]"))
        .find((candidate) => candidate.dataset.askId === String(restoreAskId));
      const rows = Array.from(document.querySelectorAll<HTMLElement>("[data-ask-id]"));
      const fallback = rows[Math.min(returnPosition.index, Math.max(0, rows.length - 1))];
      (node ?? fallback ?? heading.current)?.focus();
      document.querySelector<HTMLElement>("main")?.scrollTo({ top: returnPosition.top });
      returnPosition = { ...returnPosition, askId: "", index: 0 };
    }
  }, [asks.data, location.state]);

  const saveReturn = (askId: string, index: number) => {
    const main = document.querySelector<HTMLElement>("main");
    returnPosition = { top: main?.scrollTop ?? window.scrollY, askId, index };
  };

  const loadedEmpty = !asks.isPending && !asks.isFetching && !asks.isError && visible.length === 0 && !filter;
  const failedWithoutData = asks.isError && !asks.data;

  return (
    <section data-testid={selectors.pages.asks} aria-labelledby="asks-heading" className="mx-auto flex w-full max-w-3xl flex-col gap-4">
      {!loadedEmpty && !failedWithoutData && <h2 ref={heading} id="asks-heading" tabIndex={-1} className="text-2xl font-semibold">{t(strings.asks.list.title)}</h2>}
      {!loadedEmpty && !failedWithoutData && <div className="flex flex-wrap items-center gap-3">
        <label className="flex min-w-48 flex-1 flex-col gap-1 text-sm">
          {t(strings.asks.list.filterAll)}
          <input
            type="search"
            value={filter}
            onChange={(event) => setSearchParams((current) => {
              const next = new URLSearchParams(current);
              if (event.target.value) next.set("q", event.target.value);
              else next.delete("q");
              return next;
            })}
            className="min-h-11 rounded-control border border-app-border bg-app-surface px-3"
            aria-label={t(strings.asks.list.filterAll)}
          />
        </label>
        <Link to={preserveRoutedTestMode("/history", location.search)} className="underline">{t(strings.asks.list.historyLink)}</Link>
      </div>}
      {asks.isPending && <p role="status" className="text-app-muted-foreground">{t(strings.asks.detail.loading)}</p>}
      {asks.isFetching && asks.data && visible.length === 0 && !filter && <p role="status" className="text-app-muted-foreground">{t(strings.asks.list.checking)}</p>}
      {failedWithoutData && (
        <div role="alert" className="-mx-4 -mt-4 flex flex-col bg-app-surface-muted">
          <div className="flex flex-col gap-2 px-4 py-4 sm:px-8">
            <h2 ref={heading} id="asks-heading" tabIndex={-1} className="text-2xl font-bold sm:text-4xl">{t(strings.asks.list.loadError)}</h2>
            <p className="text-base">{t(strings.asks.list.connectionError)}</p>
            <div className="mt-1 border-l-4 border-amber-600 pl-4">
              <p className="text-base font-semibold">{t(strings.asks.list.notCleared)}</p>
              <p className="text-base">{t(strings.asks.list.retryRead)}</p>
            </div>
          </div>
          <div className="flex flex-row gap-3 border-t border-app-border px-4 py-3 sm:px-8">
            <button type="button" onClick={() => void asks.refetch()} className="min-h-11 flex-1 rounded-control bg-blue-700 px-4 font-semibold text-white hover:bg-blue-800 focus-visible:outline focus-visible:outline-2">{t(strings.asks.list.retry)}</button>
            <Link to={preserveRoutedTestMode("/settings", location.search)} className="flex min-h-11 flex-1 items-center justify-center rounded-control border border-app-border px-4 text-center underline">{t(strings.asks.list.openSettings)}</Link>
          </div>
        </div>
      )}
      {asks.isError && asks.data && <>
        <p role="alert">{t(strings.asks.list.loadError)} {t(strings.asks.list.connectionError)}</p>
        <p role="status">{t(strings.asks.list.stale)}</p>
        <div className="flex flex-wrap gap-3">
          <button type="button" onClick={() => void asks.refetch()} className="min-h-11 rounded-control bg-blue-700 px-4 font-semibold text-white">{t(strings.asks.list.retry)}</button>
          <Link to={preserveRoutedTestMode("/settings", location.search)} className="flex min-h-11 items-center underline">{t(strings.asks.list.openSettings)}</Link>
        </div>
      </>}
      {loadedEmpty && (
        <div data-testid={selectors.asks.empty} className="-mx-4 -mt-4 flex flex-col gap-4 bg-app-surface-muted px-4 py-4 sm:px-8">
          <div className="flex flex-col gap-3">
            <p className="text-app-muted-foreground">{t(strings.asks.list.needsYou)}</p>
            <h2 ref={heading} id="asks-heading" tabIndex={-1} className="text-2xl font-bold sm:text-4xl">{t(strings.asks.list.emptyTitle)}</h2>
            <p className="text-sm">{t(strings.asks.list.emptyChecked)}</p>
          </div>
          <dl className="flex flex-col gap-3 text-sm text-app-muted-foreground">
            <div className="flex justify-between gap-4"><dt>{t(strings.asks.list.checked)}</dt><dd>{t(strings.asks.list.checkedNow)}</dd></div>
            <div className="flex justify-between gap-4"><dt>{t(strings.asks.list.scope)}</dt><dd>{t(strings.asks.list.latestCap, { count: ASK_LIMIT })}</dd></div>
          </dl>
          <p className="text-sm text-app-muted-foreground">{t(strings.asks.list.olderMayExist)}</p>
        </div>
      )}
      {!asks.isPending && !asks.isError && visible.length === 0 && filter && (
        <p data-testid={selectors.asks.empty} className="text-app-muted-foreground">
          {t(strings.asks.list.emptyFiltered)}
        </p>
      )}
      {visible.length > 0 && (
        <ul data-testid={selectors.asks.list} className="flex flex-col gap-2">
          {visible.map((ask, index) => (
            <li key={ask.id} data-testid={selectors.asks.item}>
              <Link
                to={preserveRoutedTestMode(`/asks/${encodeURIComponent(ask.id)}`, location.search)}
                state={{ from: `${location.pathname}${location.search}`, restoreAskId: ask.id }}
                data-ask-id={ask.id}
                onClick={() => saveReturn(ask.id, index)}
                className="block rounded-panel border border-app-border p-4 hover:bg-app-surface-muted focus-visible:outline focus-visible:outline-2"
              >
                <span className="block font-medium">{ask.question}</span>
                {ask.recommended && <span className="block text-sm text-app-muted-foreground">{t(strings.asks.detail.recommended)}: {optionLabel(ask, ask.recommended)}</span>}
              </Link>
            </li>
          ))}
        </ul>
      )}
      {!asks.isError && asks.data?.length === ASK_LIMIT && <p className="text-sm text-app-muted-foreground">{t(strings.asks.list.latestLoaded, { count: ASK_LIMIT })}</p>}
      {loadedEmpty && <div className="-mx-4 border-t border-app-border px-4 pt-4"><Link to={preserveRoutedTestMode("/history", location.search)} className="flex min-h-11 items-center justify-center rounded-control border border-app-border text-lg underline">{t(strings.asks.list.historyLink)}</Link></div>}
    </section>
  );
}
