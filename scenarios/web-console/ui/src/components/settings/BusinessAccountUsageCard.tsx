import { useEffect, useMemo, useState } from "react";
import { Building2, Loader2 } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1";
import { getBusinessAccountUsage, getBusinessAccounts, type BusinessAccount, type BusinessAccountUsageSummary } from "../../api/monetization";

function credits(value: number | undefined): string {
  if (value === undefined) return "—";
  if (value < 0) return "Unlimited";
  return new Intl.NumberFormat().format(value);
}

export default function BusinessAccountUsageCard({ enabled }: { enabled: boolean }) {
  const [accounts, setAccounts] = useState<BusinessAccount[]>([]);
  const [selectedId, setSelectedId] = useState("");
  const [summary, setSummary] = useState<BusinessAccountUsageSummary | null>(null);
  const [loadingAccounts, setLoadingAccounts] = useState(false);
  const [loadingSummary, setLoadingSummary] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!enabled) {
      setAccounts([]);
      setSelectedId("");
      setSummary(null);
      setError(null);
      return;
    }
    let active = true;
    setLoadingAccounts(true);
    setError(null);
    void getBusinessAccounts().then((response) => {
      if (!active) return;
      const nextAccounts = response.accounts ?? [];
      setAccounts(nextAccounts);
      setSelectedId((current) => nextAccounts.some((account) => account.id === current) ? current : nextAccounts[0]?.id ?? "");
    }).catch((cause: unknown) => {
      if (active) setError(cause instanceof Error ? cause.message : "Business accounts are unavailable.");
    }).finally(() => { if (active) setLoadingAccounts(false); });
    return () => { active = false; };
  }, [enabled]);

  useEffect(() => {
    if (!enabled || !selectedId) {
      setSummary(null);
      return;
    }
    let active = true;
    setLoadingSummary(true);
    setError(null);
    setSummary(null);
    void getBusinessAccountUsage(selectedId).then((next) => {
      if (active && next.business_account_id === selectedId) setSummary(next);
    }).catch((cause: unknown) => {
      if (active) {
        setSummary(null);
        setError(cause instanceof Error ? cause.message : "Account usage is unavailable.");
      }
    }).finally(() => { if (active) setLoadingSummary(false); });
    return () => { active = false; };
  }, [enabled, selectedId]);

  const selectedAccount = useMemo(() => accounts.find((account) => account.id === selectedId), [accounts, selectedId]);
  const limit = summary?.limits.ai_credits;
  const used = summary?.usage.ai_credits;
  const reserved = summary?.reserved.ai_credits;
  const remaining = summary?.remaining.ai_credits;

  return (
    <Card data-testid="business-account-usage-card" className="overflow-hidden">
      <CardHeader className="border-b border-wc-default/70 p-5">
        <CardTitle as="h3" className="flex items-center gap-2"><Building2 className="h-4 w-4 text-wc-accent" aria-hidden="true" />Business account usage</CardTitle>
        <CardDescription className="mt-1">View the selected account’s metered AI balance and pending reservations.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4 p-5">
        {enabled ? (
          <>
            {loadingAccounts ? <p role="status" className="flex items-center gap-2 text-sm text-wc-text-muted"><Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />Loading accounts…</p> : accounts.length > 0 ? (
              <label className="flex min-h-11 flex-col gap-1 text-xs font-medium text-wc-text-secondary">
                Billing account
                <select data-testid="business-account-select" aria-label="Billing account" value={selectedId} onChange={(event) => { setSelectedId(event.target.value); }} className="min-h-11 rounded-lg border border-wc-default bg-wc-surface-input px-3 text-sm text-wc-text-primary focus-visible:outline focus-visible:outline-2 focus-visible:outline-wc-accent">
                  {accounts.map((account) => <option key={account.id} value={account.id}>{account.display_name}</option>)}
                </select>
              </label>
            ) : <p className="text-sm text-wc-text-muted">No business accounts are available.</p>}
            {error && <p role="alert" className="text-sm text-rose-200">{error}</p>}
            {loadingSummary && <p role="status" className="text-sm text-wc-text-muted">Loading account usage…</p>}
            {summary && summary.business_account_id === selectedId && selectedAccount && (
              <dl data-testid="business-account-usage-summary" className="grid grid-cols-2 gap-3 rounded-xl border border-wc-default bg-wc-surface-input/30 p-4 text-sm">
                <div><dt className="text-xs text-wc-text-faint">Selected account</dt><dd className="mt-1 font-medium text-wc-text-primary">{selectedAccount.display_name}</dd></div>
                <div><dt className="text-xs text-wc-text-faint">Plan</dt><dd className="mt-1 font-medium text-wc-text-primary">{summary.tier}</dd></div>
                <div><dt className="text-xs text-wc-text-faint">AI credits used</dt><dd className="mt-1 font-medium text-wc-text-primary">{credits(used)}</dd></div>
                <div><dt className="text-xs text-wc-text-faint">Reserved</dt><dd className="mt-1 font-medium text-wc-text-primary">{credits(reserved)}</dd></div>
                <div className="col-span-2"><dt className="text-xs text-wc-text-faint">Remaining this period</dt><dd className="mt-1 font-semibold text-wc-text-primary">{credits(remaining)} <span className="font-normal text-wc-text-faint">of {credits(limit)} · {summary.billing_period}</span></dd></div>
              </dl>
            )}
          </>
        ) : <p className="text-sm text-wc-text-muted">Connect a subscription account to view business account usage.</p>}
      </CardContent>
    </Card>
  );
}
