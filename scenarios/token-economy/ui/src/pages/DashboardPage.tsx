import { useQuery } from "@tanstack/react-query";
import { BookOpenCheck, Coins, Gift, Users } from "lucide-react";

import { minterClient } from "../api/tokenEconomy";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { EmptyState } from "../components/ui/empty-state";
import { HealthCard } from "../components/HealthCard";
import { Metric } from "../components/Metric";
import { StatusBadge } from "../components/ui/status-badge";
import { useTranslation } from "../i18n";

/**
 * Minter dashboard. Every number is derived from the same generated authority
 * client used by its owning feature page.
 */
export function DashboardPage() {
  const { t } = useTranslation();
  const summary = useQuery({
    queryKey: ["token-economy", "minter", "dashboard"],
    queryFn: async () => {
      const [tokens, holders, catalog, approvals] = await Promise.all([
        minterClient.listTokenTypes({ includeRetired: false }),
        minterClient.listHolders({}),
        minterClient.listCatalogEntries({ includeRetired: false }),
        minterClient.listPendingRedemptions({}),
      ]);
      return {
        tokens: tokens.tokenTypes.length,
        holders: holders.holders.length,
        catalog: catalog.entries.length,
        approvals: approvals.redemptions.length,
      };
    },
  });

  return (
    <section
      data-testid={selectors.pages.dashboard}
      aria-labelledby="dashboard-heading"
      className="flex flex-col gap-4"
    >
      <h2 id="dashboard-heading" className="text-2xl font-semibold">
        {t(strings.pages.dashboard.title)}
      </h2>
      <p className="text-app-muted-foreground">{t(strings.pages.dashboard.description)}</p>
      <div className="flex items-center gap-2" role="status">
        <StatusBadge tone={summary.isError ? "danger" : summary.isPending ? "warning" : "success"}>
          {summary.isError ? t(strings.common.requestError) : summary.isPending ? t(strings.common.loading) : t(strings.dashboard.current)}
        </StatusBadge>
      </div>
      {summary.isError ? <EmptyState title={t(strings.common.requestError)} icon={<BookOpenCheck aria-hidden className="h-6 w-6" />} /> : null}
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Metric label={t(strings.dashboard.tokenTypes)} value={summary.data?.tokens} icon={<Coins aria-hidden className="h-5 w-5" />} />
        <Metric label={t(strings.dashboard.holders)} value={summary.data?.holders} icon={<Users aria-hidden className="h-5 w-5" />} />
        <Metric label={t(strings.dashboard.rewards)} value={summary.data?.catalog} icon={<Gift aria-hidden className="h-5 w-5" />} />
        <Metric label={t(strings.dashboard.pendingApprovals)} value={summary.data?.approvals} icon={<BookOpenCheck aria-hidden className="h-5 w-5" />} />
      </div>
      <HealthCard />
    </section>
  );
}
