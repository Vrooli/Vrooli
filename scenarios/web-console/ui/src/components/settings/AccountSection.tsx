import { useEffect, useState } from "react";
import { deleteSubscriptionSession, getSubscriptionSession, getSubscriptionSummary, provisionOpenRouterKey, provisionSubscriptionSession, removeOpenRouterKey, testOpenRouterKey, type SubscriptionSummary } from "../../api/monetization";
import { PendingSyncBadge, PlanBadge } from "@vrooli/react-component-library/MonetizationAccount/2";
import { useAuthStore } from "../../stores/authStore";

export default function AccountSection() {
  const [status, setStatus] = useState({ configured: false });
  const [summary, setSummary] = useState<SubscriptionSummary>({ configured: false });
  const [token, setToken] = useState("");
  const [key, setKey] = useState("");
  const [message, setMessage] = useState("");
  const [keyCheck, setKeyCheck] = useState<string>("");
  const auth = useAuthStore();
  const refresh = () => Promise.all([getSubscriptionSession(), getSubscriptionSummary()]).then(([session, account]) => { setStatus(session); setSummary(account); }).catch(() => setMessage("Account status unavailable"));
  useEffect(() => { void auth.initialize().then(refresh); }, [auth.initialize]);
  const signIn = async () => {
    if (token.trim()) { await provisionSubscriptionSession(token.trim()); setToken(""); await refresh(); setMessage("Subscription session connected. The refresh credential stays in the credential authority."); return; }
    await auth.signIn();
    if (auth.error) setMessage(auth.error);
  };
  const signOut = async () => { await auth.signOut(); await deleteSubscriptionSession(); setStatus({ configured: false }); setSummary({ configured: false }); setMessage("Signed out. Local terminal use remains available."); };
  const saveKey = async () => { if (!key.trim()) return; await provisionOpenRouterKey(key.trim()); setKey(""); setMessage("OpenRouter key saved securely. It will never be returned to this browser."); };
  const removeKey = async () => { await removeOpenRouterKey(); setMessage("OpenRouter key removed. The next available source will be used."); };
  const testKey = async () => { const result = await testOpenRouterKey(); setKeyCheck(`${result.valid ? "Verified" : "Rejected"} at ${new Date(result.checked_at).toLocaleString()}`); };
  return <section data-testid="settings-account" className="max-w-2xl space-y-5">
    <header><p className="text-[11px] font-semibold uppercase tracking-[0.2em] text-wc-text-muted">Account</p><h2 className="mt-1 text-xl font-semibold">Subscription and private keys</h2><p className="mt-2 text-sm text-wc-text-faint">Web Console is complete without an account. Sign in only when you want Vrooli-provided inference.</p></header>
    <div className="rounded-2xl border border-wc-default bg-wc-surface-input/40 p-4"><div className="flex items-center justify-between"><div><h3 className="font-medium">Vrooli subscription</h3><div className="mt-2 flex items-center gap-2"><PlanBadge plan={(summary.plan_tier as "free" | "solo" | "pro" | "studio" | "business") ?? "free"} size="sm" /><PendingSyncBadge pending={summary.pending_sync ?? 0} /></div><p className="text-sm text-wc-text-faint">Status: {summary.status ?? (status.configured ? "connected" : "signed out")}</p><p data-testid="account-plan" className="text-sm text-wc-text-faint">Plan: {summary.plan_tier ?? "—"}</p><p data-testid="account-credits" className="text-sm text-wc-text-faint">Credits: {summary.credits == null ? "—" : JSON.stringify(summary.credits)}</p><p data-testid="account-pending-sync" className="text-sm text-wc-text-faint">Pending sync: {summary.pending_sync ?? 0}</p></div>{status.configured ? <button className="rounded-lg border px-3 py-2 text-sm" onClick={() => void signOut()}>Sign out</button> : <button className="rounded-lg bg-wc-accent px-3 py-2 text-sm text-white" onClick={() => void signIn()}>Sign in</button>}</div>{!status.configured && <input data-testid="account-refresh-token" value={token} onChange={(event) => setToken(event.target.value)} placeholder="Paste a refresh token if prompted" type="password" className="mt-4 w-full rounded-lg border bg-transparent px-3 py-2 text-sm" autoComplete="off" />}</div>
    <div className="rounded-2xl border border-wc-default bg-wc-surface-input/40 p-4"><h3 className="font-medium">Bring your own OpenRouter key</h3><p className="mt-1 text-sm text-wc-text-faint">Used before the Vrooli subscription source. The key is stored by the credential authority, not browser storage.</p><div className="mt-3 flex gap-2"><input data-testid="openrouter-key-input" value={key} onChange={(event) => setKey(event.target.value)} placeholder="sk-or-…" type="password" className="min-w-0 flex-1 rounded-lg border bg-transparent px-3 py-2 text-sm" autoComplete="off" /><button data-testid="openrouter-key-save" className="rounded-lg border px-3 py-2 text-sm" onClick={() => void saveKey()}>Save key</button><button data-testid="openrouter-key-test" className="rounded-lg border px-3 py-2 text-sm" onClick={() => void testKey()}>Test key</button><button data-testid="openrouter-key-remove" className="rounded-lg border px-3 py-2 text-sm" onClick={() => void removeKey()}>Remove</button></div>{keyCheck && <p data-testid="openrouter-key-check" className="mt-2 text-xs text-wc-text-faint">{keyCheck}</p>}<p className="mt-2 text-xs text-wc-text-faint">Resolution order: Ollama → your OpenRouter key → Vrooli subscription.</p></div>
    {message && <p role="status" className="text-sm text-wc-text-muted">{message}</p>}
  </section>;
}
