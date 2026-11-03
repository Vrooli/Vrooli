import { useEffect, useState } from "react";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";
import { Select } from "@vrooli/react-component-library/Select";
import { approveCodingBackup, getBusinessAccounts, getCodingBackup, removeCodingBackup, restartCodingSource, type CodingBackupStatus, type BusinessAccount } from "../api/monetization";
import { modelSelectionClient } from "../api/modelSelection";

/** Explicitly grants one aggregate cap; the server owns its expiry and keys. */
export function BackupApproval({ harness, account, model, cap, onApproved }: { harness: string; account: string; model: string; cap: number; onApproved: () => void }) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  return <div className="space-y-2 text-xs">
    <p>Use credits when this plan is used up, up to {cap.toLocaleString()} credits total over the next 24 hours. New sessions and resumable runs may switch automatically. Live terminals still ask before restarting.</p>
    <button type="button" disabled={busy || !account || !model} className="min-h-11 rounded border border-wc-default px-3 disabled:opacity-50" onClick={() => {
      setBusy(true); setError("");
      void approveCodingBackup({ harness, business_account_id: account, model, max_credits: cap, expires_at: new Date(Date.now() + 24 * 3600_000).toISOString(), automatic: true })
        .then(onApproved).catch((e: unknown) => setError(e instanceof Error ? e.message : String(e))).finally(() => setBusy(false));
    }}>{busy ? "Saving…" : "Allow backup for 24 hours"}</button>
    {error && <p role="alert">{error}</p>}
  </div>;
}

export function CodingBackup({ sessionId, onRestarted }: { sessionId: string; onRestarted?: (id: string) => void }) {
  const [status, setStatus] = useState<CodingBackupStatus | null>(null);
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [accounts, setAccounts] = useState<BusinessAccount[]>([]);
  const [account, setAccount] = useState("");
  const [models, setModels] = useState<{ value: string; label: string }[]>([]);
  const [model, setModel] = useState("");
  const [cap, setCap] = useState(5000);
  const harness = status?.limit.harness === "claude" ? "opencode" : status?.limit.harness ?? "";
  const refresh = () => getCodingBackup("", sessionId).then(setStatus).catch(() => setStatus(null));
  useEffect(() => {
    let active = true;
    const read = () => { void getCodingBackup("", sessionId).then((s) => { if (active) setStatus(s); }).catch(() => { if (active) setStatus(null); }); };
    read(); const timer = window.setInterval(read, 30_000);
    return () => { active = false; window.clearInterval(timer); };
  }, [sessionId]);
  useEffect(() => {
    if (!open || status?.consent || status?.on_credits || status?.revocation_pending) return;
    let active = true;
    void getBusinessAccounts().then((r) => { if (active) { setAccounts(r.accounts ?? []); setAccount(r.accounts?.[0]?.id ?? ""); } }).catch((e: unknown) => { if (active) setError(String(e)); });
    return () => { active = false; };
  }, [open, status?.consent, status?.on_credits, status?.revocation_pending]);
  useEffect(() => {
    setModels([]); setModel("");
    if (!open || !account || !harness) return;
    let active = true;
    void modelSelectionClient.getRoleModelChoices({ agentId: harness, targetId: "local", launchCommand: harness, businessAccountId: account }).then((r) => {
      if (!active) return;
      const options = r.choices.filter((c) => c.source === "vrooli-credits").map((c) => ({ value: c.model, label: c.displayName }));
      setModels(options); setModel(options[0]?.value ?? "");
      if (!options.length) setError(r.detail || "No model is eligible for this harness.");
    }).catch((e: unknown) => { if (active) setError(String(e)); });
    return () => { active = false; };
  }, [open, account, harness]);
  if (!status || (!status.revocation_pending && !status.on_credits && status.decision === "subscription")) return null;
  const restart = (source: "credits" | "subscription") => {
    setBusy(true); setError("");
    void restartCodingSource(sessionId, source, harness).then((created) => {
      onRestarted?.(created.id);
      setOpen(false);
    }).catch((e: unknown) => setError(String(e))).finally(() => setBusy(false));
  };
  return <>
    <button type="button" data-testid="coding-backup-status" className="min-h-7 rounded px-2 text-xs text-wc-warning" onClick={() => setOpen(true)}>{status.revocation_pending ? "Key revocation pending" : status.on_credits ? "Vrooli credits" : "Plan limit · options"}</button>
    {open && <ResponsiveDialog open avoidKeyboard onClose={() => setOpen(false)} size="sm" title={status.on_credits ? "Using Vrooli credits" : "Plan used up"} closeLabel="Close" testId="coding-backup-dialog">
      <div className="space-y-4 p-4 text-sm">
        <p>{status.limit.harness === "claude" ? "Claude Code cannot use this credit route. You can wait, use your own Anthropic credential, or start OpenCode with a summary." : "Switching starts a new session with a summary of the recent conversation. Your original session remains available."}</p>
        {status.limit.resets_at && !status.limit.resets_at.startsWith("0001-") && <p>Plan resets {new Date(status.limit.resets_at).toLocaleString()}.</p>}
        {status.revocation_pending ? <p>New backup launches are disabled. Existing keys may still work until revocation succeeds or they expire.</p> : status.on_credits ? <button type="button" disabled={busy} onClick={() => restart("subscription")} className="min-h-11 rounded border px-3">Start on my own account</button>
          : status.consent ? <><p>Approved cap: {status.consent.max_credits.toLocaleString()} credits total, until {new Date(status.consent.expires_at).toLocaleString()}.</p><button type="button" disabled={busy} onClick={() => restart("credits")} className="min-h-11 rounded border px-3">{status.limit.harness === "claude" ? "Start OpenCode on credits" : "Continue on credits"}</button></>
            : <><Select aria-label="Billing account" value={account} options={accounts.map((a) => ({ value: a.id, label: a.display_name }))} onValueChange={setAccount} /><Select aria-label="Backup model" value={model} options={models} onValueChange={setModel} /><Select aria-label="Backup credit cap" value={String(cap)} options={[1000, 5000, 10000].map((n) => ({ value: String(n), label: `${n.toLocaleString()} credits` }))} onValueChange={(n) => setCap(Number(n))} /><BackupApproval harness={harness} account={account} model={model} cap={cap} onApproved={() => { void refresh(); }} /></>}
        {(status.consent || status.revocation_pending) && <button type="button" disabled={busy} onClick={() => { void removeCodingBackup(harness).then(refresh).catch((e: unknown) => { setError(String(e)); void refresh(); }); }} className="min-h-11 px-3">{status.revocation_pending ? "Retry key revocation" : "Turn off backup and revoke its keys"}</button>}
        <button type="button" onClick={() => setOpen(false)} className="min-h-11 px-3">Wait / keep this session</button>
        {error && <p role="alert" className="text-wc-error-text">{error}</p>}
      </div>
    </ResponsiveDialog>}
  </>;
}
