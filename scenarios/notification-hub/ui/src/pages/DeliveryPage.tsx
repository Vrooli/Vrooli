import { useLocation } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Radio } from "lucide-react";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import type { GetAnalyticsResponse } from "@vrooli/proto-types/notification-hub/v1/delivery/delivery_pb";

import { deliveryClient } from "../api/notifications";
import { ListGroup, ListItem, ListRow } from "../components/ListRow";
import { Page } from "../components/Page";
import { EmptyNotice, InlineNote, QueryError } from "../components/QueryNotice";
import { SkeletonRows } from "../components/Skeleton";
import { Time } from "../components/Time";
import { isUnauthenticated } from "../lib/apiError";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { formatDate, formatNumber } from "../i18n/format";
import { HubStatus } from "../features/health/HubStatus";
import {
  DELIVERY_STATES, DELIVERY_STATE_EXCEPTIONAL, DELIVERY_STATE_LABEL, DELIVERY_STATE_MEANING, DELIVERY_STATE_TONE, deliveryStateKey,
} from "../features/delivery/deliveryView";
import { UpdatesTabs } from "../features/delivery/UpdatesTabs";
import { preserveRoutedTestMode } from "../features/routedTestMode";

const DELIVERY_LIMIT = 20;

/** Did it reach me: recent notifications with their delivery state, plus how each channel is doing. */
export function DeliveryPage() {
  const { t } = useTranslation();
  const { search } = useLocation();
  const timeline = useQuery({
    queryKey: ["delivery", "overview"],
    queryFn: () => deliveryClient.getTimeline({ limit: DELIVERY_LIMIT }),
    refetchInterval: 60_000,
  });
  const analytics = useQuery({ queryKey: ["delivery", "analytics"], queryFn: () => deliveryClient.getAnalytics({}), retry: false });
  const notifications = timeline.data?.notifications ?? [];

  return (
    <Page
      width="wide"
      testId={selectors.pages.delivery}
      headingId="delivery-heading"
      title={t(strings.pages.delivery.title)}
      description={t(strings.pages.delivery.description)}
    >
      <UpdatesTabs current="delivery" />
      <div className="grid items-start gap-6 lg:grid-cols-[minmax(0,1fr)_20rem]">
        <section aria-labelledby="delivery-recent-heading" className="flex min-w-0 flex-col gap-3">
          <h2 id="delivery-recent-heading" className="text-base font-semibold">{t(strings.pages.delivery.recentTitle)}</h2>
          {timeline.isPending && <SkeletonRows rows={4} />}
          {timeline.isError && !timeline.data && (
            <QueryError error={timeline.error} title={t(strings.pages.delivery.loadError)} onRetry={() => void timeline.refetch()} retrying={timeline.isFetching} />
          )}
          {timeline.isError && timeline.data && <InlineNote tone="warning">{t(strings.pages.delivery.stale)}</InlineNote>}
          {!timeline.isPending && !timeline.isError && notifications.length === 0 && (
            <EmptyNotice icon={<Radio size={32} />} title={t(strings.pages.delivery.emptyReturned)} detail={t(strings.pages.delivery.emptyDetail)} />
          )}
          {notifications.length > 0 && (
            <ListGroup>
              {notifications.map((notification) => {
                const state = deliveryStateKey(notification.state);
                return (
                  <ListItem key={notification.id} tone={state === "failed" || state === "unroutable" ? "attention" : "normal"}>
                    <ListRow
                      to={preserveRoutedTestMode(`/delivery/${encodeURIComponent(notification.id)}`, search)}
                      data-testid={selectors.delivery.recordDetailsLink({ id: notification.id })}
                      trailing={<StatusBadge tone={DELIVERY_STATE_TONE[state]}>{t(DELIVERY_STATE_LABEL[state])}</StatusBadge>}
                    >
                      <p className="line-clamp-2 break-words text-base font-semibold leading-snug">{notification.title || t(strings.pages.dashboard.untitled)}</p>
                      <p className="mt-1 flex flex-wrap items-center gap-x-1.5 text-sm text-app-muted-foreground">
                        <span className="[overflow-wrap:anywhere]">{notification.requestedBy || t(strings.pages.history.unknownSource)}</span>
                        <span aria-hidden="true">·</span>
                        <Time value={notification.createdAt} />
                      </p>
                      {DELIVERY_STATE_EXCEPTIONAL.has(state)
                        ? <p className="mt-1 text-sm">{t(DELIVERY_STATE_MEANING[state])}</p>
                        : <p className="sr-only">{t(DELIVERY_STATE_MEANING[state])}</p>}
                    </ListRow>
                  </ListItem>
                );
              })}
            </ListGroup>
          )}
          {notifications.length > 0 && <p className="text-xs text-app-muted-foreground">{t(strings.pages.delivery.loadedScope, { limit: DELIVERY_LIMIT })}</p>}
        </section>

        <aside className="flex min-w-0 flex-col gap-4">
          <HubStatus />
          {!isUnauthenticated(analytics.error) && <section aria-labelledby="delivery-reliability-heading" className="flex flex-col gap-2 rounded-panel border border-app-border bg-app-surface p-4">
            <h2 id="delivery-reliability-heading" className="text-sm font-semibold">{t(strings.pages.delivery.reliabilityTitle)}</h2>
            {analytics.isPending && <SkeletonRows rows={1} />}
            {analytics.isError && (
              <QueryError
                error={analytics.error}
                title={t(strings.pages.delivery.analyticsError)}
                onRetry={() => void analytics.refetch()}
                retrying={analytics.isFetching}
                retryTestId={selectors.delivery.analyticsRetry}
              />
            )}
            {analytics.data && <ChannelReliability analytics={analytics.data} />}
          </section>}
          <details className="rounded-panel border border-app-border bg-app-surface px-4">
            <summary className="flex min-h-11 cursor-pointer items-center text-sm font-semibold">{t(strings.pages.delivery.statesTitle)}</summary>
            <p className="pb-2 text-sm text-app-muted-foreground">{t(strings.pages.delivery.stateGuideDescription)}</p>
            <dl className="flex flex-col pb-2">
              {DELIVERY_STATES.map((key) => (
                <div key={key} className="border-t border-app-border py-2">
                  <dt className="text-sm font-semibold">{t(DELIVERY_STATE_LABEL[key])}</dt>
                  <dd className="text-sm text-app-muted-foreground">{t(DELIVERY_STATE_MEANING[key])}</dd>
                </div>
              ))}
            </dl>
          </details>
        </aside>
      </div>
    </Page>
  );
}

function ChannelReliability({ analytics }: { analytics: GetAnalyticsResponse }) {
  const { t } = useTranslation();
  const since = analytics.since ? Date.parse(analytics.since) : NaN;
  return (
    <>
      {!Number.isNaN(since) && (
        <p className="text-xs text-app-muted-foreground">{t(strings.pages.delivery.reliabilitySince, { since: formatDate(since, { month: "short", day: "numeric", hour: "numeric", minute: "2-digit" }) })}</p>
      )}
      {analytics.channels.length === 0 && <p className="text-sm text-app-muted-foreground">{t(strings.pages.delivery.noChannels)}</p>}
      <ul className="flex flex-col divide-y divide-app-border">
        {analytics.channels.map((channel) => {
          const failed = Number(channel.failed);
          return (
            <li key={channel.channel} className="flex flex-col gap-0.5 py-2 first:pt-0 last:pb-0">
              <p className="flex items-baseline justify-between gap-2 text-sm font-medium">
                <span className="[overflow-wrap:anywhere]">{channel.channel}</span>
                <span className={failed > 0 ? "text-app-danger" : "text-app-muted-foreground"}>
                  {t(strings.pages.delivery.failureRate, { rate: formatNumber(channel.failureRate, { style: "percent", maximumFractionDigits: 1 }) })}
                </span>
              </p>
              <p className="text-sm text-app-muted-foreground">
                {t(strings.pages.delivery.channelSummary, { delivered: formatNumber(Number(channel.delivered)), failed: formatNumber(failed) })}
                {channel.averageLatencyMs > 0 && ` · ${t(strings.pages.delivery.latency, { ms: formatNumber(Math.round(channel.averageLatencyMs)) })}`}
              </p>
            </li>
          );
        })}
      </ul>
    </>
  );
}
