import { useMemo } from "react";
import { Link, useLocation } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Bell, Lock } from "lucide-react";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import type { Notification } from "@vrooli/proto-types/notification-hub/v1/shared/types_pb";

import { conversationsClient, deliveryClient } from "../api/notifications";
import { ListGroup, ListItem } from "../components/ListRow";
import { Page } from "../components/Page";
import { EmptyNotice, InlineNote, QueryError } from "../components/QueryNotice";
import { SkeletonRows } from "../components/Skeleton";
import { SourceLink } from "../components/SourceLink";
import { Time } from "../components/Time";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { safeContextUrl } from "../features/asks/askView";
import {
  DELIVERY_STATE_EXCEPTIONAL, DELIVERY_STATE_LABEL, DELIVERY_STATE_MEANING, DELIVERY_STATE_TONE, deliveryStateKey,
} from "../features/delivery/deliveryView";
import { UpdatesTabs } from "../features/delivery/UpdatesTabs";
import { preserveRoutedTestMode } from "../features/routedTestMode";
import { useSession } from "../features/session/useSession";

const UPDATE_LIMIT = 20;
const ASK_LOOKUP_LIMIT = 200;
const URGENCY_LABEL: Record<string, string> = {
  low: strings.asks.urgency.low,
  high: strings.asks.urgency.high,
  critical: strings.asks.urgency.critical,
};
const LINK = "inline-flex min-h-11 items-center rounded-control text-sm font-medium text-app-primary hover:underline focus-visible:outline focus-visible:outline-2";

/** Updates is an informational feed; viewing it never changes notification state. */
export function DashboardPage() {
  const { t } = useTranslation();
  const { search } = useLocation();
  const session = useSession();
  const signedIn = session.isSuccess && session.data.signedIn;
  const subject = signedIn ? session.data.subject : "";
  const updates = useQuery({
    queryKey: ["updates"],
    queryFn: () => deliveryClient.getTimeline({ limit: UPDATE_LIMIT }),
    refetchInterval: 60_000,
  });
  const asks = useQuery({
    queryKey: ["updates", "linked-asks", subject],
    queryFn: () => conversationsClient.listAsks({ openOnly: false, limit: ASK_LOOKUP_LIMIT }),
    refetchInterval: 60_000,
    enabled: signedIn,
  });
  const askByNotification = useMemo(() => {
    const byNotification = new Map<string, string>();
    if (!signedIn || asks.isError) return byNotification;
    for (const ask of asks.data?.asks ?? []) {
      if (ask.notificationId && !byNotification.has(ask.notificationId)) {
        byNotification.set(ask.notificationId, ask.id);
      }
    }
    return byNotification;
  }, [asks.data, asks.isError, signedIn]);
  const notifications = updates.data?.notifications ?? [];
  const failedWithoutData = updates.isError && !updates.data;

  return (
    <Page
      testId={selectors.pages.dashboard}
      headingId="updates-heading"
      title={t(strings.pages.dashboard.title)}
      description={t(strings.pages.dashboard.description)}
    >
      <UpdatesTabs current="updates" />
      {updates.isPending && <SkeletonRows rows={4} />}
      {failedWithoutData && <QueryError error={updates.error} title={t(strings.pages.dashboard.loadError)} onRetry={() => void updates.refetch()} retrying={updates.isFetching} />}
      {updates.isError && updates.data && <InlineNote tone="warning">{t(strings.pages.dashboard.stale)}</InlineNote>}
      {asks.isError && !failedWithoutData && <InlineNote>{t(strings.pages.dashboard.askLinksUnavailable)}</InlineNote>}
      {!updates.isPending && !updates.isError && notifications.length === 0 && (
        <EmptyNotice icon={<Bell size={32} />} title={t(strings.pages.dashboard.emptyReturned)} detail={t(strings.pages.dashboard.emptyDetail)} />
      )}
      {notifications.length > 0 && (
        <ListGroup aria-label={t(strings.pages.dashboard.feedHeading)}>
          {notifications.map((item) => (
            <UpdateRow key={item.id} item={item} askId={askByNotification.get(item.id)} search={search} />
          ))}
        </ListGroup>
      )}
      {notifications.length > 0 && (
        <p className="text-xs text-app-muted-foreground">{t(strings.pages.dashboard.loadedScope, { limit: UPDATE_LIMIT })}</p>
      )}
    </Page>
  );
}

function UpdateRow({ item, askId, search }: { item: Notification; askId: string | undefined; search: string }) {
  const { t } = useTranslation();
  const state = deliveryStateKey(item.state);
  const contextUrl = item.contextUrl ? safeContextUrl(item.contextUrl) : null;
  const showUrgency = item.urgency !== "" && item.urgency !== "normal";
  const reachFailed = state === "failed" || state === "unroutable";
  return (
    <ListItem tone={reachFailed ? "attention" : "normal"} className="px-4 py-3">
      <article className="flex flex-col gap-1.5">
        <div className="flex items-start justify-between gap-3">
          <h3 className="break-words text-base font-semibold leading-snug">{item.title || t(strings.pages.dashboard.untitled)}</h3>
          <StatusBadge tone={DELIVERY_STATE_TONE[state]}>{t(DELIVERY_STATE_LABEL[state])}</StatusBadge>
        </div>
        {item.body && <p className="line-clamp-3 whitespace-pre-wrap break-words text-sm">{item.body}</p>}
        <p className="flex flex-wrap items-center gap-x-1.5 gap-y-1 text-sm text-app-muted-foreground">
          <span className="[overflow-wrap:anywhere]">{item.requestedBy || t(strings.pages.history.unknownSource)}</span>
          <span aria-hidden="true">·</span>
          <Time value={item.createdAt} />
          {showUrgency && <StatusBadge tone={item.urgency === "critical" ? "danger" : "warning"}>{URGENCY_LABEL[item.urgency] ? t(URGENCY_LABEL[item.urgency] as typeof strings.asks.urgency.low) : item.urgency}</StatusBadge>}
          {item.sensitivityLabel && (
            <span className="inline-flex items-center gap-1"><Lock size={12} aria-hidden="true" />{item.sensitivityLabel}</span>
          )}
        </p>
        {DELIVERY_STATE_EXCEPTIONAL.has(state)
          ? <p className="text-sm">{t(DELIVERY_STATE_MEANING[state])}</p>
          : <p className="sr-only">{t(DELIVERY_STATE_MEANING[state])}</p>}
        <div className="-mb-1 flex flex-wrap gap-x-4">
          {askId && (
            <Link to={preserveRoutedTestMode(`/asks/${encodeURIComponent(askId)}`, search)} className={LINK}>
              {t(strings.pages.dashboard.openDecision)}
            </Link>
          )}
          <Link to={preserveRoutedTestMode(`/delivery/${encodeURIComponent(item.id)}`, search)} className={LINK}>
            {t(strings.pages.dashboard.deliveryDetails)}
          </Link>
          {contextUrl && <SourceLink href={contextUrl} className={LINK} />}
        </div>
      </article>
    </ListItem>
  );
}
