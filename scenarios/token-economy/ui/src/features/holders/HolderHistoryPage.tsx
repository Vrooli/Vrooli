import { ArrowDownCircle, ArrowUpCircle, History } from "lucide-react";
import { EventKind } from "@vrooli/proto-types/token-economy/v1/access/access_pb";

import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { eventDate, useHolderEconomy } from "./useHolderEconomy";

export function HolderHistoryPage() {
  const { t } = useTranslation();
  const economy = useHolderEconomy();
  const events = economy.data?.events ?? [];

  return (
    <section className="grid gap-5" data-testid="page-holder-history" aria-labelledby="holder-history-title">
      <div>
        <h1 id="holder-history-title" className="text-3xl font-semibold">{t(strings.holderView.historyTitle)}</h1>
        <p className="mt-2 text-base text-app-muted-foreground">{t(strings.holderView.historyDescription)}</p>
      </div>
      {economy.isLoading && <p role="status">{t(strings.common.loading)}</p>}
      {economy.error && <p role="alert">{t(strings.holderView.signInRequired)}</p>}
      {events.length === 0 && !economy.isLoading && !economy.error ? (
        <EmptyState title={t(strings.holderView.noHistory)} description={t(strings.holderView.noHistoryDescription)} icon={<History aria-hidden className="h-7 w-7" />} />
      ) : (
        <ol className="grid gap-3" aria-label={t(strings.holderView.historyListLabel)}>
          {events.map((event) => {
            const debit = event.kind === EventKind.DEBIT;
            const Icon = debit ? ArrowDownCircle : ArrowUpCircle;
            return (
              <li key={event.id} className="flex min-w-0 gap-4 rounded-panel border border-app-border bg-app-surface p-4">
                <Icon aria-hidden className="mt-1 h-7 w-7 shrink-0 text-app-primary" />
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-start justify-between gap-2">
                    <p className="text-lg font-semibold">{event.reason || t(strings.holderView.journalEvent)}</p>
                    <p className="font-mono text-xl font-semibold">{debit ? "−" : "+"}{event.amount.toString()}</p>
                  </div>
                  <p className="mt-1 text-sm text-app-muted-foreground">{eventDate(event)} · {event.tokenTypeId}</p>
                </div>
              </li>
            );
          })}
        </ol>
      )}
    </section>
  );
}
