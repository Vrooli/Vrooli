import { useMemo } from "react";
import { Link, useLocation } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";

import { conversationsClient, deliveryClient } from "../api/notifications";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { HealthCard } from "../features/health/HealthCard";
import { deliveryStateKey, formatDeliveryTime } from "../features/delivery/deliveryView";
import { preserveRoutedTestMode } from "../features/routedTestMode";

const UPDATE_LIMIT = 20;
const ASK_LOOKUP_LIMIT = 50;

/** Updates is an informational feed; viewing it never changes notification state. */
export function DashboardPage() {
  const { t } = useTranslation();
  const location = useLocation();
  const updates = useQuery({
    queryKey: ["updates"],
    queryFn: () => deliveryClient.getTimeline({ limit: UPDATE_LIMIT }),
    refetchInterval: 60_000,
  });
  const asks = useQuery({
    queryKey: ["updates", "linked-asks"],
    queryFn: () => conversationsClient.listAsks({ openOnly: false, limit: ASK_LOOKUP_LIMIT }),
    refetchInterval: 60_000,
  });
  const askByNotification = useMemo(() => {
    const byNotification = new Map<string, string>();
    for (const ask of asks.data?.asks ?? []) {
      if (ask.notificationId && !byNotification.has(ask.notificationId)) {
        byNotification.set(ask.notificationId, ask.id);
      }
    }
    return byNotification;
  }, [asks.data]);
  const notifications = updates.data?.notifications ?? [];

  return (
    <section
      data-testid={selectors.pages.dashboard}
      aria-labelledby="updates-heading"
      className="flex flex-col gap-4"
    >
      <h2 id="updates-heading" className="text-2xl font-semibold">
        {t(strings.pages.dashboard.title)}
      </h2>
      <p className="max-w-prose text-app-muted-foreground">{t(strings.pages.dashboard.description)}</p>
      <section aria-label={t(strings.pages.dashboard.feedHeading)} className="flex flex-col gap-3">
        <p className="text-sm text-app-muted-foreground">{t(strings.pages.dashboard.loadedScope, { limit: UPDATE_LIMIT })}</p>
        {updates.isPending && <p role="status">{t(strings.asks.detail.loading)}</p>}
        {updates.isError && <p role="alert">{t(strings.pages.dashboard.loadError)}</p>}
        {updates.isError && updates.data && <p role="status">{t(strings.pages.dashboard.stale)}</p>}
        {asks.isError && <p role="status">{t(strings.pages.dashboard.askLinksUnavailable)}</p>}
        {!updates.isPending && !updates.isError && notifications.length === 0 && (
          <p className="text-app-muted-foreground">{t(strings.pages.dashboard.emptyReturned)}</p>
        )}
        {notifications.length > 0 && (
          <ul aria-label={t(strings.pages.dashboard.feedHeading)} className="flex flex-col gap-3">
            {notifications.map((item) => {
              const state = deliveryStateKey(item.state);
              const askId = askByNotification.get(item.id);
              const time = formatDeliveryTime(item.createdAt);
              return (
                <li key={item.id} className="rounded-panel border border-app-border p-4">
                  <article className="flex flex-col gap-2">
                    <h3 className="break-words text-base font-medium">
                      {item.title || t(strings.pages.dashboard.untitled)}
                    </h3>
                    {item.body && <p className="whitespace-pre-wrap break-words">{item.body}</p>}
                    <dl className="grid gap-x-4 gap-y-1 text-sm text-app-muted-foreground sm:grid-cols-2">
                      <div>
                        <dt className="inline font-medium">{t(strings.pages.dashboard.source)} </dt>
                        <dd className="inline break-words">{item.requestedBy || t(strings.pages.dashboard.unknownSource)}</dd>
                      </div>
                      <div>
                        <dt className="inline font-medium">{t(strings.pages.dashboard.created)} </dt>
                        <dd className="inline">
                          {time ? <time dateTime={item.createdAt}>{time}</time> : t(strings.pages.dashboard.timeUnavailable)}
                        </dd>
                      </div>
                      {item.urgency && (
                        <div><dt className="inline font-medium">{t(strings.pages.dashboard.urgency)} </dt><dd className="inline">{item.urgency}</dd></div>
                      )}
                      {item.sensitivityLabel && (
                        <div><dt className="inline font-medium">{t(strings.pages.dashboard.sensitivity)} </dt><dd className="inline">{item.sensitivityLabel}</dd></div>
                      )}
                    </dl>
                    <p className="text-sm">
                      <span className="font-medium">{t(strings.pages.delivery.states[state])}</span>
                      <span className="text-app-muted-foreground"> — {t(strings.pages.delivery.meanings[state])}</span>
                      {item.reason && <span className="text-app-muted-foreground"> — {item.reason}</span>}
                    </p>
                    <div className="flex flex-wrap gap-x-4 gap-y-2 text-sm">
                      <Link
                        to={preserveRoutedTestMode(`/delivery/${encodeURIComponent(item.id)}`, location.search)}
                        className="rounded-sm underline focus-visible:outline focus-visible:outline-2"
                      >
                        {t(strings.pages.dashboard.deliveryDetails)}
                      </Link>
                      {askId && (
                        <Link
                          to={preserveRoutedTestMode(`/asks/${encodeURIComponent(askId)}`, location.search)}
                          className="rounded-sm underline focus-visible:outline focus-visible:outline-2"
                        >
                          {t(strings.pages.dashboard.openDecision)}
                        </Link>
                      )}
                    </div>
                  </article>
                </li>
              );
            })}
          </ul>
        )}
      </section>
      <section aria-label={t(strings.health.title)} className="border-t border-app-border pt-4">
        <HealthCard />
      </section>
    </section>
  );
}
