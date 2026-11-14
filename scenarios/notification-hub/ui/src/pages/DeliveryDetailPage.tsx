import { useLocation, useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";

import { getDeliveryAttempts, notificationsClient } from "../api/notifications";
import { BackLink } from "../components/BackLink";
import { Facts } from "../components/Facts";
import { Page } from "../components/Page";
import { InlineNote, QueryError } from "../components/QueryNotice";
import { SkeletonRows } from "../components/Skeleton";
import { SourceLink } from "../components/SourceLink";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { DELIVERY_STATE_LABEL, DELIVERY_STATE_MEANING, DELIVERY_STATE_TONE, deliveryStateKey, formatDeliveryTime } from "../features/delivery/deliveryView";
import { externalHost, safeContextUrl } from "../features/asks/askView";
import { preserveRoutedTestMode } from "../features/routedTestMode";

const URGENCY_LABEL: Record<string, string> = {
  low: strings.asks.urgency.low,
  normal: strings.asks.urgency.normal,
  high: strings.asks.urgency.high,
  critical: strings.asks.urgency.critical,
};

/** The story of one notification: where it stands, the receipts adapters returned, and each delivery attempt. */
export function DeliveryDetailPage() {
  const { t } = useTranslation();
  const location = useLocation();
  const { notificationId = "" } = useParams();
  const record = useQuery({
    queryKey: ["delivery", "record", notificationId],
    queryFn: () => notificationsClient.get({ id: notificationId }),
    enabled: notificationId.length > 0,
  });
  const notification = record.data?.notification;
  const receipts = record.data?.receipts ?? [];
  const attempts = useQuery({ queryKey: ["delivery", "attempts", notificationId], queryFn: () => getDeliveryAttempts(notificationId), enabled: Boolean(notification) });
  // The Hub names a native permission request itself; the page never infers one from the link's shape.
  const reviewHref = record.data?.nativeConsentReviewUrl ? safeContextUrl(record.data.nativeConsentReviewUrl) : null;
  const reviewHost = reviewHref ? externalHost(reviewHref) : null;
  const contextUrl = notification?.contextUrl ? safeContextUrl(notification.contextUrl) : null;
  // The review button already is the producer's link for a permission request.
  const contextHref = contextUrl && contextUrl !== reviewHref ? contextUrl : null;
  const state = notification ? deliveryStateKey(notification.state) : "unknown";
  const unknown = Boolean(notification) && state === "unknown";
  const noReceipt = Boolean(notification) && !unknown && receipts.length === 0;
  const unavailable = t(strings.pages.delivery.timeUnavailable);
  const notProvided = t(strings.pages.delivery.unknownValue);
  const checked = record.dataUpdatedAt ? formatDeliveryTime(new Date(record.dataUpdatedAt).toISOString()) : "";

  return (
    <Page
      testId={selectors.pages.deliveryDetail}
      headingId="delivery-detail-heading"
      leading={<BackLink to={preserveRoutedTestMode("/delivery", location.search)}>{t(strings.pages.delivery.back)}</BackLink>}
      title={notification ? notification.title || t(strings.pages.dashboard.untitled) : t(strings.pages.delivery.detailTitle)}
      description={notification && <span className="font-mono text-xs [overflow-wrap:anywhere]">{notification.id}</span>}
    >
      {record.isPending && <SkeletonRows rows={2} />}
      {record.isError && <QueryError error={record.error} title={t(strings.pages.delivery.detailLoadError)} onRetry={() => void record.refetch()} retrying={record.isFetching} />}
      {!record.isPending && !record.isError && !notification && <p role="alert">{t(strings.pages.delivery.notFound)}</p>}

      {notification && (
        <>
          <section className="flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-4">
            {reviewHref && (
              // The decision this record exists for, and the one time-sensitive tap: it leads, above every fact.
              <div className="flex flex-col gap-1">
                <a
                  href={reviewHref}
                  rel="noreferrer"
                  aria-describedby="review-in-agent-manager-note"
                  className="inline-flex min-h-12 items-center justify-center rounded-pill bg-app-primary px-5 py-2 text-base font-medium text-app-primary-foreground focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-app-focus"
                >
                  {t(strings.nativeConsent.open)}
                </a>
                <p id="review-in-agent-manager-note" className="text-sm text-app-muted-foreground">
                  {t(strings.nativeConsent.authorityShort)}
                  {reviewHost && <> <bdi dir="ltr">{reviewHost}</bdi></>}
                </p>
              </div>
            )}
            <div className="flex flex-wrap items-center gap-2">
              <StatusBadge tone={DELIVERY_STATE_TONE[state]}>{t(DELIVERY_STATE_LABEL[state])}</StatusBadge>
              <p className="text-sm text-app-muted-foreground">{t(DELIVERY_STATE_MEANING[state])}</p>
            </div>
            {unknown && (
              <aside aria-labelledby="unknown-status-heading" className="border-s-4 border-app-warning ps-3">
                <h2 id="unknown-status-heading" className="font-semibold">{t(strings.pages.delivery.unknownStatusHeading)}</h2>
                <p className="mt-1 text-sm">{t(strings.pages.delivery.unknownStatusExplanation)}</p>
                <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.pages.delivery.reportIssue)}</p>
              </aside>
            )}
            {noReceipt && (
              <aside aria-labelledby="reach-unknown-heading" className="border-s-4 border-app-warning ps-3">
                <h2 id="reach-unknown-heading" className="font-semibold">{t(strings.pages.delivery.reachUnknownTitle)}</h2>
                <p className="mt-1 text-sm">{t(strings.pages.delivery.noReceiptSummary)}</p>
                <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.pages.delivery.noReceipt)}</p>
              </aside>
            )}
            <Facts
              items={[
                { label: t(strings.pages.delivery.source), value: notification.requestedBy || t(strings.pages.history.unknownSource) },
                { label: t(strings.pages.delivery.created), value: formatDeliveryTime(notification.createdAt) || unavailable },
                { label: t(strings.pages.delivery.updated), value: formatDeliveryTime(notification.updatedAt) || unavailable },
                ...(unknown ? [{ label: t(strings.pages.delivery.lastChecked), value: checked || unavailable }] : []),
                ...(notification.urgency ? [{ label: t(strings.pages.delivery.urgency), value: URGENCY_LABEL[notification.urgency] ? t(URGENCY_LABEL[notification.urgency] as typeof strings.asks.urgency.low) : notification.urgency }] : []),
                ...(notification.sensitivityLabel ? [{ label: t(strings.pages.delivery.sensitivity), value: notification.sensitivityLabel }] : []),
              ]}
            />
            {contextHref && (
              // A push opens this record, never the producer's link: this is the second, deliberate tap.
              <SourceLink href={contextHref} className="inline-flex min-h-11 items-center self-start rounded-control text-sm font-medium text-app-primary hover:underline focus-visible:outline focus-visible:outline-2" />
            )}
            {unknown && (
              <Button type="button" variant="primary" disabled={record.isFetching} onClick={() => void record.refetch()}>
                {t(strings.pages.delivery.refreshStatus)}
              </Button>
            )}
          </section>

          {receipts.length > 0 && (
            <section aria-labelledby="receipts-heading" className="flex flex-col gap-2">
              <h2 id="receipts-heading" className="text-base font-semibold">{t(strings.pages.delivery.receiptsTitle)}</h2>
              <p className="text-sm text-app-muted-foreground">{t(strings.pages.delivery.receiptsExplanation)}</p>
              <ul className="divide-y divide-app-border overflow-hidden rounded-panel border border-app-border bg-app-surface">
                {receipts.map((receipt) => (
                  <li key={receipt.id} className="px-4 py-3">
                    <Facts items={[
                      { label: t(strings.pages.delivery.channel), value: receipt.channel || notProvided },
                      { label: t(strings.pages.delivery.adapterAcceptedAt), value: formatDeliveryTime(receipt.deliveredAt) || unavailable },
                    ]} />
                  </li>
                ))}
              </ul>
            </section>
          )}

          {(attempts.isPending || attempts.isError || attempts.data.attempts.length > 0) && (
            <section aria-labelledby="attempts-heading" className="flex flex-col gap-2">
              <h2 id="attempts-heading" className="text-base font-semibold">{t(strings.pages.delivery.attemptsTitle)}</h2>
              {attempts.isPending && <SkeletonRows rows={1} />}
              {attempts.isError && (
                <QueryError error={attempts.error} title={t(strings.pages.delivery.attemptsError)} onRetry={() => void attempts.refetch()} retrying={attempts.isFetching} />
              )}
              {(attempts.data?.attempts.length ?? 0) > 0 && (
                <ol className="divide-y divide-app-border overflow-hidden rounded-panel border border-app-border bg-app-surface">
                  {attempts.data?.attempts.map((item, index) => (
                    <li key={`${item.created_at}-${item.attempt}-${index}`} className="px-4 py-3">
                      <Facts items={[
                        { label: t(strings.pages.delivery.attemptNumber), value: item.attempt },
                        { label: t(strings.pages.delivery.attemptOutcome), value: item.outcome },
                        { label: t(strings.pages.delivery.channel), value: item.channel || notProvided },
                        { label: t(strings.pages.delivery.created), value: formatDeliveryTime(item.created_at) || unavailable },
                        ...(item.next_attempt_at ? [{ label: t(strings.pages.delivery.nextAttempt), value: formatDeliveryTime(item.next_attempt_at) || unavailable }] : []),
                      ]} />
                    </li>
                  ))}
                </ol>
              )}
              {attempts.data?.truncated && <InlineNote>{t(strings.pages.delivery.attemptsTruncated)}</InlineNote>}
              <InlineNote>{t(strings.pages.delivery.currentStateOnly)}</InlineNote>
            </section>
          )}
        </>
      )}
    </Page>
  );
}
