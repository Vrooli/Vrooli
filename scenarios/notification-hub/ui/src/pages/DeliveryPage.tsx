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
  const timeline = useQuery({
    queryKey: ["delivery", "overview"],
    queryFn: () => deliveryClient.getTimeline({ limit: DELIVERY_LIMIT }),
    refetchInterval: 60_000,
  });
  const notifications = timeline.data?.notifications ?? [];

  return (
    <section data-testid={selectors.pages.delivery} aria-labelledby="delivery-states-heading" className="mx-auto flex w-full max-w-4xl flex-col gap-4">
      <section aria-labelledby="delivery-states-heading" className="flex flex-col gap-2">
        <p className="text-sm text-app-muted-foreground">{t(strings.pages.delivery.stateGuideEyebrow)}</p>
        <h2 id="delivery-states-heading" className="text-3xl font-bold">{t(strings.pages.delivery.stateGuideTitle)}</h2>
        <p className="max-w-prose">{t(strings.pages.delivery.stateGuideDescription)}</p>
        <dl className="flex flex-col">
          {DELIVERY_STATES.map((key) => (
            <div key={key} className="border-b border-app-border py-2">
              <dt className="font-semibold">{t(strings.pages.delivery.states[key])}</dt>
              <dd className="mt-1 text-app-muted-foreground">{t(strings.pages.delivery.meanings[key])}</dd>
            </div>
          ))}
        </dl>
      </section>
      <p className="text-sm text-app-muted-foreground">{t(strings.pages.delivery.loadedScope, { limit: DELIVERY_LIMIT })}</p>
      {timeline.isPending && <p role="status">{t(strings.asks.detail.loading)}</p>}
      {timeline.isError && <p role="alert">{t(strings.pages.delivery.loadError)}</p>}
      {timeline.isError && timeline.data && <p role="status">{t(strings.pages.delivery.stale)}</p>}
      {!timeline.isPending && !timeline.isError && notifications.length === 0 && (
        <p className="text-app-muted-foreground">{t(strings.pages.delivery.emptyReturned)}</p>
      )}
      {notifications.length > 0 && (
        <ul aria-label={t(strings.pages.delivery.recordsHeading)} className="flex flex-col gap-3">
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
                  {notification.reason && <p className="break-words text-sm text-app-muted-foreground">{notification.reason}</p>}
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
  );
}
