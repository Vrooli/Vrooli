import { Link, useLocation, useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";

import { notificationsClient } from "../api/notifications";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { deliveryStateKey, formatDeliveryTime } from "../features/delivery/deliveryView";
import { preserveRoutedTestMode } from "../features/routedTestMode";

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
  const state = notification ? deliveryStateKey(notification.state) : "unknown";
  const created = formatDeliveryTime(notification?.createdAt ?? "");
  const updated = formatDeliveryTime(notification?.updatedAt ?? "");
  const receiptEvidence = (
    <section aria-labelledby="receipts-heading" className="flex flex-col gap-2">
      <h3 id="receipts-heading" className="text-lg font-semibold">{t(strings.pages.delivery.receiptsTitle)}</h3>
      <p className="max-w-prose text-sm text-app-muted-foreground">{t(strings.pages.delivery.receiptsExplanation)}</p>
      {receipts.length === 0 ? (
        <p className="text-sm text-app-muted-foreground">{t(strings.pages.delivery.noReceipt)}</p>
      ) : (
        <ul className="flex flex-col gap-2">
          {receipts.map((receipt) => {
            const acceptedAt = formatDeliveryTime(receipt.deliveredAt);
            return (
              <li key={receipt.id} className="rounded-panel border border-app-border p-4">
                <dl className="grid gap-x-4 gap-y-1 text-sm sm:grid-cols-3">
                  <div><dt className="font-medium">{t(strings.pages.delivery.channel)}</dt><dd className="text-app-muted-foreground">{receipt.channel || t(strings.pages.delivery.unknownValue)}</dd></div>
                  <div><dt className="font-medium">{t(strings.pages.delivery.machine)}</dt><dd className="break-words text-app-muted-foreground">{receipt.machineId || t(strings.pages.delivery.unknownValue)}</dd></div>
                  <div><dt className="font-medium">{t(strings.pages.delivery.adapterAcceptedAt)}</dt><dd className="text-app-muted-foreground">{acceptedAt || t(strings.pages.delivery.timeUnavailable)}</dd></div>
                </dl>
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );

  return (
    <section data-testid={selectors.pages.deliveryDetail} aria-labelledby="delivery-detail-heading" className="mx-auto flex w-full max-w-4xl flex-col gap-4">
      <Link to={preserveRoutedTestMode("/delivery", location.search)} className="w-fit rounded-sm text-sm underline focus-visible:outline focus-visible:outline-2">
        {t(strings.pages.delivery.back)}
      </Link>
      {notification && state === "unknown" ? (
        <h2 id="delivery-detail-heading" className="text-2xl font-bold">{t(strings.pages.delivery.unknownStatusTitle)}</h2>
      ) : notification && receipts.length === 0 ? (
        <h2 id="delivery-detail-heading" className="text-3xl font-bold">{t(strings.pages.delivery.noReceiptTitle)}</h2>
      ) : (
        <h2 id="delivery-detail-heading" className="text-2xl font-semibold">{t(strings.pages.delivery.detailTitle)}</h2>
      )}
      {record.isPending && <p role="status">{t(strings.asks.detail.loading)}</p>}
      {record.isError && <p role="alert">{t(strings.pages.delivery.detailLoadError)}</p>}
      {!record.isPending && !record.isError && !notification && <p role="alert">{t(strings.pages.delivery.notFound)}</p>}
      {notification && state === "unknown" && (
        <>
          <section className="flex flex-col gap-4">
            <aside className="border-s-4 border-app-warning ps-4 py-2" aria-labelledby="unknown-status-heading">
              <h3 id="unknown-status-heading" className="text-lg font-semibold">{t(strings.pages.delivery.unknownStatusHeading)}</h3>
              <p className="mt-1">{t(strings.pages.delivery.unknownStatusExplanation)}</p>
            </aside>
            <dl className="grid gap-x-4 gap-y-3 text-sm sm:grid-cols-2">
              <div><dt className="font-medium">{t(strings.pages.delivery.notificationId)}</dt><dd className="break-all font-mono text-app-muted-foreground">{notification.id}</dd></div>
              <div><dt className="font-medium">{t(strings.pages.delivery.lastChecked)}</dt><dd className="text-app-muted-foreground">{t(strings.pages.delivery.timeUnavailable)}</dd></div>
              <div><dt className="font-medium">{t(strings.pages.delivery.transportState)}</dt><dd className="text-app-muted-foreground">{t(strings.pages.delivery.states.unknown)}</dd></div>
            </dl>
            <p className="text-sm text-app-muted-foreground">{t(strings.pages.delivery.reportIssue)}</p>
          </section>
          <div className="flex flex-row gap-3 border-t border-app-border pt-4">
            <button
              type="button"
              onClick={() => void record.refetch()}
              disabled={record.isFetching}
              className="flex min-h-12 flex-1 items-center justify-center rounded-control bg-app-primary px-3 py-3 text-center font-semibold text-app-primary-foreground focus-visible:outline focus-visible:outline-2 disabled:opacity-60"
            >
              {t(strings.pages.delivery.refreshStatus)}
            </button>
            <Link to={preserveRoutedTestMode("/delivery", location.search)} className="flex min-h-12 flex-1 items-center justify-center rounded-control border border-app-border px-3 py-3 text-center focus-visible:outline focus-visible:outline-2">
              {t(strings.pages.delivery.back)}
            </Link>
          </div>
          {receipts.length > 0 && receiptEvidence}
        </>
      )}
      {notification && state !== "unknown" && receipts.length === 0 && (
        <>
          <section className="flex flex-col gap-4">
            <p className="max-w-prose text-lg">{t(strings.pages.delivery.noReceiptSummary)}</p>
            <dl className="grid gap-x-4 gap-y-3 text-sm sm:grid-cols-2">
              <div><dt className="font-medium">{t(strings.pages.delivery.notificationId)}</dt><dd className="break-all font-mono text-app-muted-foreground">{notification.id}</dd></div>
              <div>
                <dt className="font-medium">{t(strings.pages.delivery.transportState)}</dt>
                <dd className="text-app-muted-foreground">{t(strings.pages.delivery.states[state])} — {t(strings.pages.delivery.meanings[state])}</dd>
              </div>
              <div className="sm:col-span-2"><dt className="font-medium">{t(strings.pages.delivery.requestState)}</dt><dd className="text-app-muted-foreground">{t(strings.pages.delivery.requestStateSeparate)}</dd></div>
            </dl>
            <aside className="border-s-4 border-app-warning ps-4 py-2" aria-labelledby="reach-unknown-heading">
              <h3 id="reach-unknown-heading" className="text-lg font-semibold">{t(strings.pages.delivery.reachUnknownTitle)}</h3>
              <p className="mt-1">{t(strings.pages.delivery.noReceipt)}</p>
              <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.pages.delivery.noReceiptCaveat)}</p>
            </aside>
          </section>
          <Link to={preserveRoutedTestMode("/delivery", location.search)} className="w-full rounded-sm border border-app-border px-4 py-3 text-center focus-visible:outline focus-visible:outline-2">
            {t(strings.pages.delivery.back)}
          </Link>
        </>
      )}
      {notification && state !== "unknown" && receipts.length > 0 && (
        <>
          <article className="flex flex-col gap-3 rounded-panel border border-app-border p-4">
            <h3 className="break-words text-lg font-medium">{notification.title || t(strings.pages.dashboard.untitled)}</h3>
            <p className="font-medium">{t(strings.pages.delivery.states[state])}</p>
            <p className="text-sm text-app-muted-foreground">{t(strings.pages.delivery.meanings[state])}</p>
            {notification.reason && (
              <div>
                <h4 className="text-sm font-medium">{t(strings.pages.delivery.reason)}</h4>
                <p className="break-words text-sm">{notification.reason}</p>
              </div>
            )}
            <dl className="grid gap-x-4 gap-y-2 text-sm sm:grid-cols-2">
              <div><dt className="font-medium">{t(strings.pages.delivery.notificationId)}</dt><dd className="break-all font-mono text-app-muted-foreground">{notification.id}</dd></div>
              <div><dt className="font-medium">{t(strings.pages.delivery.source)}</dt><dd className="break-words text-app-muted-foreground">{notification.requestedBy || t(strings.pages.dashboard.unknownSource)}</dd></div>
              <div><dt className="font-medium">{t(strings.pages.delivery.created)}</dt><dd className="text-app-muted-foreground">{created || t(strings.pages.delivery.timeUnavailable)}</dd></div>
              <div><dt className="font-medium">{t(strings.pages.delivery.updated)}</dt><dd className="text-app-muted-foreground">{updated || t(strings.pages.delivery.timeUnavailable)}</dd></div>
              {notification.urgency && <div><dt className="font-medium">{t(strings.pages.dashboard.urgency)}</dt><dd className="text-app-muted-foreground">{notification.urgency}</dd></div>}
              {notification.sensitivityLabel && <div><dt className="font-medium">{t(strings.pages.dashboard.sensitivity)}</dt><dd className="text-app-muted-foreground">{notification.sensitivityLabel}</dd></div>}
            </dl>
          </article>
          {receiptEvidence}
          <section aria-labelledby="attempts-heading" className="flex flex-col gap-2 border-t border-app-border pt-4">
            <h3 id="attempts-heading" className="text-lg font-semibold">{t(strings.pages.delivery.attemptsTitle)}</h3>
            <p className="max-w-prose text-sm text-app-muted-foreground">{t(strings.pages.delivery.attemptsUnavailable)}</p>
            <p className="max-w-prose text-sm text-app-muted-foreground">{t(strings.pages.delivery.currentStateOnly)}</p>
          </section>
        </>
      )}
    </section>
  );
}
