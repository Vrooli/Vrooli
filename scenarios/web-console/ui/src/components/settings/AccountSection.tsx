import { useEffect, useMemo, useState } from "react";
import { ArrowUpRight, Check, CircleHelp, CreditCard, KeyRound, RefreshCw, ShieldCheck, UserRound } from "lucide-react";
import { Alert } from "@vrooli/react-component-library/Alert/1";
import { Avatar } from "@vrooli/react-component-library/Avatar/1";
import { BoundedMeter } from "@vrooli/react-component-library/BoundedMeter/1";
import { Button } from "@vrooli/react-component-library/Button/2";
import { ButtonGroup } from "@vrooli/react-component-library/ButtonGroup/1";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1";
import { DescriptionList } from "@vrooli/react-component-library/DescriptionList/1";
import { Input } from "@vrooli/react-component-library/Input/1";
import { InputGroup } from "@vrooli/react-component-library/InputGroup";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { PasswordInput } from "@vrooli/react-component-library/PasswordInput/2";
import { StatusBadge, type StatusTone } from "@vrooli/react-component-library/StatusBadge/1";
import { PendingSyncBadge, PlanBadge } from "@vrooli/react-component-library/MonetizationAccount/2";
import {
  deleteSubscriptionSession,
  getSubscriptionSession,
  getSubscriptionSummary,
  provisionOpenRouterKey,
  provisionSubscriptionSession,
  removeOpenRouterKey,
  testOpenRouterKey,
  type SubscriptionSummary,
} from "../../api/monetization";
import { useAuthStore } from "../../stores/authStore";
import { LANDING_PAGE_URL } from "../../shared/upgradeDestination";

type PlanTier = "free" | "solo" | "pro" | "studio" | "business";
interface CreditSnapshot { balance: number | null; limit: number | null; }

function asNumber(value: unknown): number | null {
  return typeof value === "number" && Number.isFinite(value) ? value : null;
}

function creditSnapshot(value: unknown): CreditSnapshot {
  if (typeof value === "number") return { balance: value, limit: null };
  if (!value || typeof value !== "object") return { balance: null, limit: null };
  const record = value as Record<string, unknown>;
  const find = (keys: string[]) => keys.map((key) => asNumber(record[key])).find((candidate) => candidate !== null) ?? null;
  return { balance: find(["remaining", "balance", "balance_credits", "available", "value"]), limit: find(["limit", "monthly_limit", "included", "monthly_included_credits", "total"]) };
}

function formatNumber(value: number | null): string {
  return value === null ? "Unavailable" : new Intl.NumberFormat(undefined, { notation: "compact", maximumFractionDigits: 1 }).format(value);
}

function statusTone(status: string | undefined): StatusTone {
  if (status === "active" || status === "trialing" || status === "connected") return "success";
  if (status === "past_due" || status === "canceled") return "warning";
  if (status === "inactive" || status === "signed_out") return "neutral";
  return "info";
}

function statusLabel(status: string | undefined, configured: boolean): string {
  if (status === "past_due") return "Payment needs attention";
  if (status === "canceled") return "Canceled";
  if (status === "active") return "Active";
  if (status === "trialing") return "Trial";
  return configured ? "Connected" : "Not connected";
}

function renewalLabel(value: string | undefined): string | null {
  if (!value) return null;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? null : `Access through ${date.toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" })}`;
}

function openAccountPage(path: string) {
  const destination = new URL(path, LANDING_PAGE_URL).toString();
  window.open(destination, "_blank", "noopener,noreferrer");
}

function ActionButton({ children, onClick, variant = "secondary", icon }: { children: React.ReactNode; onClick?: () => void; variant?: "primary" | "secondary" | "ghost"; icon?: React.ReactNode; }) {
  return <Button type="button" size="sm" shape="pill" variant={variant} onClick={onClick} icon={icon}>{children}</Button>;
}

export default function AccountSection() {
  const [status, setStatus] = useState({ configured: false });
  const [summary, setSummary] = useState<SubscriptionSummary>({ configured: false });
  const [token, setToken] = useState("");
  const [key, setKey] = useState("");
  const [message, setMessage] = useState("");
  const [keyCheck, setKeyCheck] = useState("");
  const [busy, setBusy] = useState<string | null>(null);
  const auth = useAuthStore();
  const configured = status.configured;
  const credits = useMemo(() => creditSnapshot(summary.credits), [summary.credits]);
  const plan = (summary.plan_tier as PlanTier | undefined) ?? "free";
  const subscriptionStatus = summary.status ?? (configured ? "connected" : "signed_out");
  const renewal = renewalLabel(summary.not_after);

  const refresh = async () => {
    setBusy("refresh");
    try {
      const [session, account] = await Promise.all([getSubscriptionSession(), getSubscriptionSummary()]);
      setStatus(session); setSummary(account); setMessage("");
    } catch { setMessage("Account status is temporarily unavailable. Your local workspace remains available."); }
    finally { setBusy(null); }
  };

  useEffect(() => {
    void auth.initialize().then(refresh);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [auth.initialize]);

  const signIn = async () => {
    setBusy("sign-in");
    try {
      if (token.trim()) {
        await provisionSubscriptionSession(token.trim()); setToken(""); await refresh(); setMessage("Subscription connected securely on this machine.");
      } else { await auth.signIn(); if (auth.error) setMessage(auth.error); }
    } catch (error) { setMessage(error instanceof Error ? error.message : "Sign-in failed. Please try again."); }
    finally { setBusy(null); }
  };

  const signOut = async () => {
    setBusy("sign-out");
    try { await auth.signOut(); await deleteSubscriptionSession(); setStatus({ configured: false }); setSummary({ configured: false }); setMessage("Signed out. Local terminal use remains available."); }
    catch (error) { setMessage(error instanceof Error ? error.message : "Sign-out failed. Please try again."); }
    finally { setBusy(null); }
  };

  const saveKey = async () => {
    if (!key.trim()) return;
    setBusy("key-save");
    try { await provisionOpenRouterKey(key.trim()); setKey(""); setMessage("OpenRouter key saved securely. The secret is never returned to this browser."); }
    catch (error) { setMessage(error instanceof Error ? error.message : "Could not save the OpenRouter key."); }
    finally { setBusy(null); }
  };

  const removeKey = async () => {
    setBusy("key-remove");
    try { await removeOpenRouterKey(); setMessage("OpenRouter key removed. The next available source will be used."); }
    catch (error) { setMessage(error instanceof Error ? error.message : "Could not remove the OpenRouter key."); }
    finally { setBusy(null); }
  };

  const testKey = async () => {
    setBusy("key-test");
    try { const result = await testOpenRouterKey(); setKeyCheck(`${result.valid ? "Verified" : "Rejected"} · checked ${new Date(result.checked_at).toLocaleString()}`); }
    catch (error) { setKeyCheck(error instanceof Error ? error.message : "Key verification failed"); }
    finally { setBusy(null); }
  };

  return (
    <section data-testid="settings-account" className="mx-auto w-full max-w-3xl space-y-6 pb-8">
      <PageHeader
        level={2}
        eyebrow="Account"
        title="Subscription and access"
        description="Manage the account that powers Vrooli services on this machine. Web Console remains fully useful without an account."
        leading={<div className="grid h-11 w-11 shrink-0 place-items-center rounded-2xl bg-wc-accent/15 text-wc-accent"><UserRound className="h-5 w-5" aria-hidden="true" /></div>}
        actions={<ActionButton onClick={() => void refresh()} icon={<RefreshCw className={`h-4 w-4 ${busy === "refresh" ? "animate-spin" : ""}`} aria-hidden="true" />}>Refresh</ActionButton>}
        testId="settings-account-header"
      />

      <Card data-testid="account-identity-card" className="overflow-hidden"><CardContent className="flex flex-wrap items-center justify-between gap-4 p-5"><div className="flex min-w-0 items-center gap-3"><Avatar name={configured ? "Vrooli account" : "Not signed in"} size="md" presence={configured ? "online" : "offline"} presenceLabel={configured ? "Account connected" : "Not signed in"} /><div className="min-w-0"><p className="truncate font-semibold text-wc-text-primary">{configured ? "Vrooli account connected" : "Not signed in"}</p><p className="mt-1 text-sm text-wc-text-faint">{configured ? "Subscription access is shared with Vrooli apps on this machine." : "Sign in only when you want Vrooli-provided services."}</p></div></div>{configured ? <ActionButton onClick={() => { void signOut(); }} variant="ghost">{busy === "sign-out" ? "Signing out…" : "Sign out"}</ActionButton> : <ActionButton onClick={() => { void signIn(); }} variant="primary">{busy === "sign-in" ? "Connecting…" : "Sign in"}</ActionButton>}</CardContent>{!configured && <div className="border-t border-wc-default/70 bg-wc-surface-input/25 px-5 py-4"><PasswordInput data-testid="account-refresh-token" value={token} onValueChange={setToken} label="Optional refresh token" description="Use this only when an administrator has provided a device-scoped token." revealable={false} autoComplete="off" /></div>}</Card>

      <Card data-testid="account-overview-card" className="overflow-hidden">
        <CardHeader className="border-b border-wc-default/70 bg-gradient-to-br from-wc-surface-input/60 to-transparent p-5">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div>
              <CardTitle as="h3" className="flex items-center gap-2"><CreditCard className="h-4 w-4 text-wc-accent" aria-hidden="true" />Subscription overview</CardTitle>
              <CardDescription className="mt-1">Your Vrooli-provided services and current account status.</CardDescription>
            </div>
            <div className="flex flex-wrap items-center gap-2"><PlanBadge plan={plan} size="sm" /><StatusBadge tone={statusTone(subscriptionStatus)}>{statusLabel(subscriptionStatus, configured)}</StatusBadge></div>
          </div>
        </CardHeader>
        <CardContent className="space-y-5 p-5">
          <DescriptionList entries={[{ term: "Plan", description: plan }, { term: "AI credits", description: formatNumber(credits.balance) }, { term: "Renewal", description: renewal ?? "Managed by your account" }]} />
          {credits.balance !== null && credits.limit !== null && credits.limit > 0 && <BoundedMeter label="AI credits" value={credits.balance} min={0} max={credits.limit} valueText={`${formatNumber(credits.balance)} remaining`} description="Credits are used only when Vrooli provides the inference service. Local Ollama and your own key stay outside this balance." tone={credits.balance / credits.limit < 0.1 ? "warning" : "success"} testId="account-credits-meter" />}
          <ButtonGroup label="Billing actions">
            <ActionButton onClick={() => { openAccountPage("/account"); }} icon={<ArrowUpRight className="h-4 w-4" aria-hidden="true" />}>Manage billing</ActionButton>
            <ActionButton onClick={() => { openAccountPage("/pricing"); }} variant="ghost" icon={<CircleHelp className="h-4 w-4" aria-hidden="true" />}>Compare plans</ActionButton>
          </ButtonGroup>
        </CardContent>
      </Card>

      <Card data-testid="account-free-access-card"><CardHeader className="p-5"><CardTitle as="h3" className="flex items-center gap-2"><ShieldCheck className="h-4 w-4 text-wc-accent" aria-hidden="true" />Always available without an account</CardTitle><CardDescription className="mt-1">An account adds convenience; it never takes away the local-first experience.</CardDescription></CardHeader><CardContent className="grid gap-3 p-5 pt-0 sm:grid-cols-2">{["Full terminal workspace", "Local Ollama generation", "Your own OpenRouter key", "Local voice features", "Your own connected machines"].map((item) => <div key={item} className="flex items-center gap-2 text-sm text-wc-text-secondary"><span className="grid h-5 w-5 shrink-0 place-items-center rounded-full bg-emerald-400/10 text-emerald-300"><Check className="h-3 w-3" aria-hidden="true" /></span>{item}</div>)}</CardContent></Card>

      <Card data-testid="account-key-card">
        <CardHeader className="p-5"><CardTitle as="h3" className="flex items-center gap-2"><KeyRound className="h-4 w-4 text-wc-accent" aria-hidden="true" />Bring your own OpenRouter key</CardTitle><CardDescription className="mt-1">Your key is tried before the Vrooli subscription and is stored by the credential authority—not in browser storage.</CardDescription></CardHeader>
        <CardContent className="space-y-4 p-5 pt-0">
          <div className="space-y-2">
            <label htmlFor="openrouter-key-input" className="block text-sm font-medium text-wc-text-primary">OpenRouter API key</label>
            <InputGroup testId="openrouter-key-group" size="lg">
              <InputGroup.Field>
                <Input id="openrouter-key-input" data-testid="openrouter-key-input" type="password" value={key} onChange={(event) => { setKey(event.target.value); }} aria-describedby="openrouter-key-description" placeholder="sk-or-…" autoComplete="off" className="text-wc-text-primary placeholder:text-wc-text-faint" />
              </InputGroup.Field>
              <InputGroup.Segment side="trailing" emphasis="solid" aria-label="Save OpenRouter key" onClick={() => void saveKey()} disabled={!key.trim() || busy !== null}>{busy === "key-save" ? "Saving…" : "Save"}</InputGroup.Segment>
            </InputGroup>
            <p id="openrouter-key-description" className="text-xs leading-5 text-wc-text-faint">The key is write-only here and will never be displayed again.</p>
          </div>
          <ButtonGroup label="OpenRouter key actions">
            <ActionButton onClick={() => void testKey()}>{busy === "key-test" ? "Testing…" : "Test key"}</ActionButton>
            <ActionButton onClick={() => void removeKey()} variant="ghost">Remove key</ActionButton>
          </ButtonGroup>
          {keyCheck && <Alert tone={keyCheck.startsWith("Verified") ? "success" : "danger"} title="OpenRouter key check" description={keyCheck} />}
          <Alert tone="info" title="How requests are resolved" description="Local Ollama → your OpenRouter key → Vrooli subscription. Only the final source uses subscription credits." />
        </CardContent>
      </Card>

      {(summary.pending_sync ?? 0) > 0 && <Alert tone="warning" title="Usage waiting to sync" description="Your local activity will be reconciled when this machine is online." actions={<PendingSyncBadge pending={summary.pending_sync ?? 0} />} />}
      {message && <Alert tone="info" title="Account update" description={message} />}
      <p data-testid="account-pending-sync" className="sr-only">Pending sync: {summary.pending_sync ?? 0}</p>
    </section>
  );
}
