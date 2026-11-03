import { useEffect, useState } from "react";
import { getBusinessAccountUsage, getBusinessAccounts, type BusinessAccount, type BusinessAccountUsageSummary } from "../../api/monetization";

/** The accounts a metered Vrooli-credits launch may bill, and the chosen one's remaining credits. */
export function useBillingAccount(enabled: boolean, accountId: string, setAccountId: (id: string) => void) {
  const [accounts, setAccounts] = useState<BusinessAccount[]>([]);
  const [accountsLoading, setAccountsLoading] = useState(false);
  const [accountsFailed, setAccountsFailed] = useState(false);
  const [maxCredits, setMaxCredits] = useState(5000);
  const [usage, setUsage] = useState<BusinessAccountUsageSummary | null>(null);
  const [usageLoading, setUsageLoading] = useState(false);
  const [usageFailed, setUsageFailed] = useState(false);

  useEffect(() => {
    setAccounts([]);
    setAccountsFailed(false);
    setAccountId("");
    if (!enabled) { setAccountsLoading(false); return; }
    let cancelled = false;
    setAccountsLoading(true);
    void getBusinessAccounts().then((response) => {
      if (cancelled) return;
      // The server encodes an empty list as null.
      const list = (response.accounts as BusinessAccount[] | null) ?? [];
      setAccounts(list);
      setAccountId(list[0]?.id ?? "");
    }).catch(() => {
      if (!cancelled) setAccountsFailed(true);
    }).finally(() => { if (!cancelled) setAccountsLoading(false); });
    return () => { cancelled = true; };
  }, [enabled]);

  useEffect(() => {
    setUsage(null);
    setUsageFailed(false);
    if (!enabled || !accountId) { setUsageLoading(false); return; }
    let cancelled = false;
    setUsageLoading(true);
    void getBusinessAccountUsage(accountId).then((summary) => {
      if (cancelled) return;
      if (summary.business_account_id === accountId) setUsage(summary);
      else setUsageFailed(true);
    }).catch(() => {
      if (!cancelled) setUsageFailed(true);
    }).finally(() => { if (!cancelled) setUsageLoading(false); });
    return () => { cancelled = true; };
  }, [accountId, enabled]);

  const usageCurrent = usage?.business_account_id === accountId ? usage : null;
  return {
    accounts, accountsLoading, accountsFailed, accountId, setAccountId, maxCredits, setMaxCredits, usage: usageCurrent, usageLoading, usageFailed,
    /** A paid launch cannot start until the account and its credits are known. */
    blocked: enabled && (accountsLoading || accountsFailed || !accountId || usageLoading || usageFailed || !usageCurrent
      || typeof usageCurrent.remaining.ai_credits !== "number" || usageCurrent.remaining.ai_credits === 0),
  };
}

export type BillingAccountController = ReturnType<typeof useBillingAccount>;
