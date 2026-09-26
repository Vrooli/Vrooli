import { useMutation, useQueryClient } from "@tanstack/react-query";
import { CircleCheck, Gift, ShieldAlert } from "lucide-react";
import { useState } from "react";

import { holderClient, nextIdempotencyKey } from "../../api/tokenEconomy";
import { Button } from "@vrooli/react-component-library/Button/2";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { eligibleGrantId, holderEconomyKey, useHolderCatalog, useHolderEconomy } from "./useHolderEconomy";

export function HolderRewardsPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const economy = useHolderEconomy();
  const catalog = useHolderCatalog();
  const [success, setSuccess] = useState("");
  const redeem = useMutation({
    mutationFn: ({ catalogEntryId, grantId }: { catalogEntryId: string; grantId: string }) =>
      holderClient.requestRedemption({
        redemption: { catalogEntryId, grantId },
        idempotencyKey: nextIdempotencyKey("redeem"),
      }),
    onSuccess: async (response) => {
      setSuccess(response.redemption?.id ?? "recorded");
      await queryClient.invalidateQueries({ queryKey: holderEconomyKey });
    },
  });
  const failureReason = redeem.error instanceof Error ? redeem.error.message : "";

  return (
    <section className="grid gap-5" data-testid="page-holder-rewards" aria-labelledby="holder-rewards-title">
      <div>
        <h1 id="holder-rewards-title" className="text-3xl font-semibold">{t(strings.holderView.rewardsTitle)}</h1>
        <p className="mt-2 text-base text-app-muted-foreground">{t(strings.holderView.rewardsDescription)}</p>
      </div>
      {(catalog.isLoading || economy.isLoading) && <p role="status">{t(strings.common.loading)}</p>}
      {(catalog.error || economy.error) && <p role="alert">{t(strings.holderView.signInRequired)}</p>}
      {success && (
        <p role="status" className="flex items-center gap-2 rounded-panel border border-app-success/30 bg-app-success/10 p-4 font-semibold text-app-success">
          <CircleCheck aria-hidden className="h-6 w-6" />{t(strings.holderView.redemptionRecorded)}
        </p>
      )}
      {failureReason && (
        <p role="alert" className="flex items-start gap-2 rounded-panel border border-app-danger/30 bg-app-danger/10 p-4 text-app-danger">
          <ShieldAlert aria-hidden className="mt-0.5 h-6 w-6 shrink-0" />
          <span>{t(strings.holderView.redemptionRefused, { reason: failureReason })}</span>
        </p>
      )}
      {(catalog.data?.entries.length ?? 0) === 0 && !catalog.isLoading && !catalog.error ? (
        <EmptyState title={t(strings.holderView.noRewards)} description={t(strings.holderView.noRewardsDescription)} icon={<Gift aria-hidden className="h-7 w-7" />} />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2">
          {(catalog.data?.entries ?? []).map((entry) => {
            const grantId = eligibleGrantId(entry, economy.data?.events ?? []);
            return (
              <article key={entry.id} className="flex flex-col rounded-panel border border-app-border bg-app-surface p-5">
                <Gift aria-hidden className="h-8 w-8 text-app-primary" />
                <h2 className="mt-3 text-xl font-semibold">{entry.title}</h2>
                <p className="mt-2 flex-1 text-app-muted-foreground">{entry.description}</p>
                <p className="mt-4 text-lg font-semibold">{t(strings.holderView.cost, { amount: entry.costAmount.toString() })}</p>
                {!grantId && <p className="mt-2 text-sm text-app-muted-foreground">{t(strings.holderView.noEligibleGrant)}</p>}
                <Button
                  className="mt-4 min-h-14 w-full text-base"
                  data-testid={`holder-redeem-${entry.id}`}
                  disabled={!grantId || redeem.isPending}
                  onClick={() => redeem.mutate({ catalogEntryId: entry.id, grantId })}
                >
                  <Gift aria-hidden className="h-5 w-5" />{t(strings.holderView.redeem)}
                </Button>
              </article>
            );
          })}
        </div>
      )}
    </section>
  );
}
