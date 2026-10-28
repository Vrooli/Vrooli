import { useEffect, useRef, useState } from "react";
import { Link, useLocation } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";

import { deliveryClient } from "../api/notifications";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { deliveryStateKey, formatDeliveryTime } from "../features/delivery/deliveryView";
import { preserveRoutedTestMode } from "../features/routedTestMode";

const DELIVERY_LIMIT = 20;
const DELIVERY_STATES = ["pending", "held", "routed", "delivered", "failed", "unroutable", "suppressed"] as const;

export function DeliveryPage() {
  const { t } = useTranslation();
  const location = useLocation();
  const [showRecords, setShowRecords] = useState(false);
  const recordsRef = useRef<HTMLElement>(null);
  const timeline = useQuery({
    queryKey: ["delivery", "overview"],
    queryFn: () => deliveryClient.getTimeline({ limit: DELIVERY_LIMIT }),
    refetchInterval: 60_000,
  });
  const analytics = useQuery({ queryKey: ["delivery", "analytics"], queryFn: () => deliveryClient.getAnalytics({}), retry: false });
  const notifications = timeline.data?.notifications ?? [];
  const recordsVisible = !analytics.isError || showRecords;

  useEffect(() => {
    if (!analytics.isError || !showRecords) return;
    const records = recordsRef.current;
    if (!records) return;
    records.scrollIntoView?.({ block: "start" });
    records.focus({ preventScroll: true });
  }, [analytics.isError, showRecords]);

  return (
    <section data-testid={selectors.pages.delivery} aria-labelledby={analytics.isError ? "analytics-heading" : "delivery-states-heading"} className="mx-auto flex w-full max-w-4xl flex-col gap-4">
      {analytics.isError && (
        <section role="alert" aria-labelledby="analytics-heading" className="flex flex-col gap-4 border-b border-app-border pb-4">
          <p className="text-sm text-app-muted-foreground">{t(strings.pages.delivery.analyticsEyebrow)}</p>
          <h2 id="analytics-heading" className="text-[1.6875rem] font-bold leading-tight">{t(strings.pages.delivery.analyticsErrorTitle)}</h2>
          <p className="max-w-prose text-[15px]">{t(strings.pages.delivery.analyticsError)}</p>
          <aside className="border-s-4 border-app-warning ps-4 py-2">
            <h3 className="text-lg font-semibold">{t(strings.pages.delivery.analyticsRecordsTitle)}</h3>
            <p className="mt-1">{t(timeline.isError && !timeline.data ? strings.pages.delivery.analyticsRecordsUnavailable : strings.pages.delivery.analyticsRecordsAvailable)}</p>
          </aside>
          <div className="mt-8 flex flex-row gap-3 border-t border-app-border pt-4">
            <button type="button" data-testid={selectors.delivery.analyticsRetry} onClick={() => void analytics.refetch()} disabled={analytics.isFetching} className="flex min-h-12 flex-1 items-center justify-center rounded-control bg-app-primary px-2 py-3 text-center text-sm font-semibold whitespace-nowrap text-app-primary-foreground focus-visible:outline focus-visible:outline-2 disabled:opacity-60">{t(strings.pages.delivery.retryAnalytics)}</button>
            <button
              type="button"
              aria-controls="delivery-record-list"
              data-testid={selectors.delivery.analyticsRecordsAction}
              onClick={() => setShowRecords(true)}
              className="flex min-h-12 flex-1 items-center justify-center rounded-control border border-app-border px-2 py-3 text-center text-sm whitespace-nowrap focus-visible:outline focus-visible:outline-2"
            >
              {t(strings.pages.delivery.viewRecords)}
            </button>
          </div>
        </section>
      )}
      {!analytics.isError && <section aria-labelledby="delivery-states-heading" className="flex flex-col gap-2 text-[15px]">
        <p className="text-sm text-app-muted-foreground">{t(strings.pages.delivery.stateGuideEyebrow)}</p>
        <h2 id="delivery-states-heading" className="text-[1.6875rem] font-bold leading-tight">{t(strings.pages.delivery.stateGuideTitle)}</h2>
        <p className="max-w-prose">{t(strings.pages.delivery.stateGuideDescription)}</p>
        <dl className="flex flex-col">
          {DELIVERY_STATES.map((key) => (
            <div key={key} className="border-b border-app-border py-[18px] leading-5">
              <dt className="font-semibold">{t(strings.pages.delivery.states[key])}</dt>
              <dd className="mt-1 text-app-muted-foreground">{t(strings.pages.delivery.meanings[key])}</dd>
            </div>
          ))}
        </dl>
      </section>}
      {recordsVisible && (
        <section
          ref={recordsRef}
          id="delivery-record-list"
          tabIndex={-1}
          aria-describedby="delivery-records-summary"
          aria-label={t(strings.pages.delivery.recordsHeading)}
          className="mt-16 flex flex-col gap-3 focus-visible:outline focus-visible:outline-2"
        >
          <p id="delivery-records-summary" className="text-sm text-app-muted-foreground">{t(strings.pages.delivery.loadedScope, { limit: DELIVERY_LIMIT })}</p>
          {timeline.isPending && <p role="status">{t(strings.asks.detail.loading)}</p>}
          {timeline.isError && (
            <div role="alert" className="flex flex-col items-start gap-2">
              <p>{t(strings.pages.delivery.loadError)}</p>
              <button type="button" onClick={() => void timeline.refetch()} disabled={timeline.isFetching} className="w-fit rounded-control border border-app-border px-3 py-2 disabled:opacity-60">
                {t(strings.errorBoundary.retry)}
              </button>
            </div>
          )}
          {timeline.isError && timeline.data && <p role="status">{t(strings.pages.delivery.stale)}</p>}
          {!timeline.isPending && !timeline.isError && notifications.length === 0 && (
            <p className="text-app-muted-foreground">{t(strings.pages.delivery.emptyReturned)}</p>
          )}
          {notifications.length > 0 && (
            <ul className="flex flex-col gap-3">
          {notifications.map((notification) => {
            const state = deliveryStateKey(notification.state);
            const time = formatDeliveryTime(notification.createdAt);
            return (
              <li key={notification.id} className="rounded-panel border border-app-border p-4">
                <article className="flex flex-col gap-2">
                  <h3 className="break-words font-medium">{notification.title || t(strings.pages.dashboard.untitled)}</h3>
                  <p className="text-sm">
                    <span className="font-medium">{t(strings.pages.delivery.states[state])}</span>
                    <span className="text-app-muted-foreground"> — {t(strings.pages.delivery.meanings[state])}</span>
                  </p>
                  <p className="text-sm text-app-muted-foreground">
                    {time ? <time dateTime={notification.createdAt}>{time}</time> : t(strings.pages.delivery.timeUnavailable)}
                  </p>
                  <Link
                    to={preserveRoutedTestMode(`/delivery/${encodeURIComponent(notification.id)}`, location.search)}
                    data-testid={selectors.delivery.recordDetailsLink({ id: notification.id })}
                    className="w-fit rounded-sm text-sm underline focus-visible:outline focus-visible:outline-2"
                  >
                    {t(strings.pages.delivery.openDetails)}
                  </Link>
                </article>
              </li>
            );
          })}
            </ul>
          )}
        </section>
      )}
    </section>
  );
}
