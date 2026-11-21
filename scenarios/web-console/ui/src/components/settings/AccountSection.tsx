import { useEffect, useMemo, useState } from "react";
import { Check, KeyRound, ShieldCheck } from "lucide-react";
import { Alert } from "@vrooli/react-component-library/Alert/1";
import { BoundedMeter } from "@vrooli/react-component-library/BoundedMeter/1";
import { ButtonGroup } from "@vrooli/react-component-library/ButtonGroup/1";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1";
import { Input } from "@vrooli/react-component-library/Input/1";
import { InputGroup } from "@vrooli/react-component-library/InputGroup";
import { PasswordInput } from "@vrooli/react-component-library/PasswordInput/2";
import { MonetizationAccountPanel, type PlanTier } from "@vrooli/react-component-library/MonetizationAccount/2";
import BusinessAccountUsageCard from "./BusinessAccountUsageCard";
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

function statusLabel(status: string | undefined, configured: boolean): string {
  if (status === "past_due") return "Payment needs attention";
  if (status === "canceled") return "Canceled";
  if (status === "active") return "Active";
  if (status === "trialing") return "Trial";
  if (status === "inactive") return "Inactive";
  return configured ? "Connected" : "Not connected";
}

function renewalLabel(value: string | undefined): string | undefined {
  if (!value) return undefined;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? undefined : `Access through ${date.toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" })}`;
}

function openAccountPage(path: string) {
  const destination = new URL(path, LANDING_PAGE_URL).toString();
  window.open(destination, "_blank", "noopener,noreferrer");
}

export default function AccountSection() {
  const [status, setStatus] = useState({ configured: false });
  const [summary, setSummary] = useState<SubscriptionSummary>({ configured: false });
  const [token, setToken] = useState("");
  const [key, setKey] = useState("");
  const [message, setMessage] = useState("");
  const [accountError, setAccountError] = useState("");
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
      setStatus(session); setSummary(account); setMessage(""); setAccountError("");
    } catch {
      setAccountError("Account status is temporarily unavailable. Your local workspace remains available.");
    } finally { setBusy(null); }
  };

  useEffect(() => {
    void auth.initialize().then(refresh);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [auth.initialize]);

  const signIn = async () => {
    setBusy("sign-in"); setAccountError("");
    try {
      if (token.trim()) {
        await provisionSubscriptionSession(token.trim()); setToken(""); await refresh(); setMessage("Subscription connected securely on this machine.");
      } else { await auth.signIn(); if (auth.error) setAccountError(auth.error); }
    } catch (error) { setAccountError(error instanceof Error ? error.message : "Sign-in failed. Please try again."); }
    finally { setBusy(null); }
  };

  const signOut = async () => {
    setBusy("sign-out"); setAccountError("");
    try {
      await auth.signOut(); await deleteSubscriptionSession(); setStatus({ configured: false }); setSummary({ configured: false }); setMessage("Signed out. Local terminal use remains available.");
    } catch (error) { setAccountError(error instanceof Error ? error.message : "Sign-out failed. Please try again."); }
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
    <MonetizationAccountPanel
      configured={configured}
      plan={plan}
      status={subscriptionStatus}
      statusLabel={statusLabel(subscriptionStatus, configured)}
      creditsLabel={formatNumber(credits.balance)}
      renewalLabel={renewal}
      busyAction={busy}
      error={accountError}
      signInDetails={
        <PasswordInput data-testid="account-refresh-token" value={token} onValueChange={setToken} label="Optional refresh token" description="Use this only when an administrator has provided a device-scoped token." revealable={false} autoComplete="off" />
      }
      onSignIn={() => void signIn()}
      onSignOut={() => void signOut()}
      onRefresh={() => void refresh()}
      onManageAccount={() => openAccountPage("/account")}
      onComparePlans={() => openAccountPage("/pricing")}
    >
      {credits.balance !== null && credits.limit !== null && credits.limit > 0 && <BoundedMeter label="AI credits" value={credits.balance} min={0} max={credits.limit} valueText={`${formatNumber(credits.balance)} remaining`} description="Credits are used only when Vrooli provides the inference service. Local Ollama and your own key stay outside this balance." tone={credits.balance / credits.limit < 0.1 ? "warning" : "success"} testId="account-credits-meter" />}
      <BusinessAccountUsageCard enabled={configured} />

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
            <button type="button" onClick={() => void testKey()} disabled={busy !== null}>{busy === "key-test" ? "Testing…" : "Test key"}</button>
            <button type="button" onClick={() => void removeKey()} disabled={busy !== null}>Remove key</button>
          </ButtonGroup>
          {keyCheck && <Alert tone={keyCheck.startsWith("Verified") ? "success" : "danger"} title="OpenRouter key check" description={keyCheck} />}
          <Alert tone="info" title="How requests are resolved" description="Local Ollama → your OpenRouter key → Vrooli subscription. Only the final source uses subscription credits." />
        </CardContent>
      </Card>

      {message && <div role="status"><Alert tone="info" title="Account update" description={message} /></div>}
    </MonetizationAccountPanel>
  );
}
