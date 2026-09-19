import { ArrowDownCircle, ArrowUpCircle, CircleCheck, Clock3, Coins, History } from "lucide-react";
import { Link } from "react-router-dom";
import { EventKind, RedemptionState } from "@vrooli/proto-types/token-economy/v1/access/access_pb";

import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { eventDate, useHolderEconomy } from "./useHolderEconomy";

export function HolderHomePage() {
  const { t } = useTranslation();
  const economy = useHolderEconomy();
  const view = economy.data;
  const recentEvents = view?.events.slice(0, 4) ?? [];
  const pending = view?.redemptions.filter((item) => item.state === RedemptionState.PENDING_APPROVAL) ?? [];

  return (
    <section className="grid gap-6" data-testid="page-holder-home" aria-labelledby="holder-home-title">
      <div>
        <p className="text-sm font-semibold text-app-primary">{t(strings.holderView.eyebrow)}</p>
        <h1 id="holder-home-title" className="mt-1 text-3xl font-semibold">
          {view?.holder ? t(strings.holderView.welcome, { name: view.holder.displayName }) : t(strings.holderView.title)}
        </h1>
        <p className="mt-2 text-base text-app-muted-foreground">{t(strings.holderView.balanceExplanation)}</p>
      </div>

      {economy.isLoading && <p role="status">{t(strings.common.loading)}</p>}
      {economy.error && <p role="alert">{t(strings.holderView.signInRequired)}</p>}

      <div className="grid gap-3 sm:grid-cols-2" aria-label={t(strings.holderView.balances)}>
        {(view?.balances ?? []).map((balance) => (
          <article key={balance.tokenTypeId} className="rounded-panel border border-app-border bg-app-surface p-5">
            <Coins aria-hidden className="h-7 w-7 text-app-primary" />
            <p className="mt-3 text-sm text-app-muted-foreground">{balance.tokenTypeId}</p>
            <p className="mt-1 text-4xl font-semibold" data-testid={`holder-balance-${balance.tokenTypeId}`}>
              {balance.amount.toString()}
            </p>
          </article>
        ))}
      </div>
      {!economy.isLoading && !economy.error && (view?.balances.length ?? 0) === 0 && (
        <EmptyState title={t(strings.holderView.noBalance)} description={t(strings.holderView.noBalanceDescription)} icon={<Coins aria-hidden className="h-7 w-7" />} />
      )}

      <section className="rounded-panel border border-app-border bg-app-surface p-5" aria-labelledby="holder-recent-title">
        <div className="flex items-center justify-between gap-3">
          <h2 id="holder-recent-title" className="text-xl font-semibold">{t(strings.holderView.recentHistory)}</h2>
          <Link to="/me/history" className="inline-flex min-h-12 items-center gap-2 rounded-control px-3 font-semibold text-app-primary hover:bg-app-surface-muted">
            <History aria-hidden className="h-5 w-5" />{t(strings.holderView.seeAll)}
          </Link>
        </div>
        <ul className="mt-3 grid gap-3">
          {recentEvents.map((event) => {
            const debit = event.kind === EventKind.DEBIT;
            const Icon = debit ? ArrowDownCircle : ArrowUpCircle;
            return (
              <li key={event.id} className="flex min-w-0 items-center gap-3 rounded-control bg-app-surface-muted p-3">
                <Icon aria-hidden className="h-6 w-6 shrink-0 text-app-primary" />
                <div className="min-w-0 flex-1">
                  <p className="font-semibold">{event.reason || t(strings.holderView.journalEvent)}</p>
                  <p className="text-sm text-app-muted-foreground">{eventDate(event)}</p>
                </div>
                <span className="font-mono text-lg font-semibold">{debit ? "−" : "+"}{event.amount.toString()}</span>
              </li>
            );
          })}
        </ul>
        {recentEvents.length === 0 && <p className="mt-3 text-app-muted-foreground">{t(strings.holderView.noHistory)}</p>}
      </section>

      <section className="rounded-panel border border-app-border bg-app-surface p-5" aria-labelledby="holder-pending-title">
        <h2 id="holder-pending-title" className="text-xl font-semibold">{t(strings.holderView.pendingTitle)}</h2>
        {pending.length === 0 ? (
          <p className="mt-2 flex items-center gap-2 text-app-muted-foreground"><CircleCheck aria-hidden className="h-5 w-5" />{t(strings.holderView.noPending)}</p>
        ) : (
          <ul className="mt-3 grid gap-3">
            {pending.map((redemption) => (
              <li key={redemption.id} className="flex items-center justify-between gap-3 rounded-control bg-app-surface-muted p-3">
                <span>{t(strings.holderView.rewardRequest, { amount: redemption.amount.toString() })}</span>
                <StatusBadge tone="warning"><Clock3 aria-hidden className="mr-1 h-4 w-4" />{t(strings.common.pending)}</StatusBadge>
              </li>
            ))}
          </ul>
        )}
      </section>
    </section>
  );
}
