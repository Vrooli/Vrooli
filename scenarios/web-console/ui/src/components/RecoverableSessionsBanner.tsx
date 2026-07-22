import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

import { dismissRecoverableSession, listRecoverableSessions, recoverSession, type RecoverableSession, type RecoverResult } from "../api/sessions";
import { strings } from "../consts/strings";
import { cn } from "../lib/classnames";
import { DrawerShell } from "./DrawerShell";

export interface RecoverableSessionsBannerProps {
  onRecovered?: (result: RecoverResult) => void;
  topSafe?: boolean;
}

export default function RecoverableSessionsBanner({ onRecovered, topSafe = false }: RecoverableSessionsBannerProps) {
  const { t } = useTranslation();
  const [rows, setRows] = useState<RecoverableSession[] | null>(null);
  const [busy, setBusy] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [open, setOpen] = useState(false);
  // Retain the key for this banner's lifetime so an ambiguous response can be
  // retried without creating a second replacement session.
  const recoveryKeys = useRef(new Map<string, string>());

  const refresh = useCallback(async () => {
    try { setRows(await listRecoverableSessions()); }
    catch (err) { setError(err instanceof Error ? err.message : String(err)); setRows([]); }
  }, []);
  useEffect(() => { void refresh(); }, [refresh]);

  const recover = useCallback(async (id: string) => {
    setBusy(id); setError(null);
    const key = recoveryKeys.current.get(id) ?? crypto.randomUUID();
    recoveryKeys.current.set(id, key);
    try { const result = await recoverSession(id, key); onRecovered?.(result); await refresh(); }
    catch (err) { setError(err instanceof Error ? err.message : String(err)); }
    finally { setBusy(null); }
  }, [onRecovered, refresh]);
  const dismiss = useCallback(async (id: string) => {
    setBusy(id); setError(null);
    try { await dismissRecoverableSession(id); await refresh(); }
    catch (err) { setError(err instanceof Error ? err.message : String(err)); }
    finally { setBusy(null); }
  }, [refresh]);
  const recoverAll = useCallback(async () => {
    if (!rows) return;
    for (const row of rows) if (row.recoverable) await recover(row.id);
  }, [recover, rows]);
  const dismissAll = useCallback(async () => {
    if (!rows) return;
    for (const row of rows) await dismiss(row.id);
  }, [dismiss, rows]);

  if (!rows?.length) return null;
  return <>
    <div data-testid="recoverable-sessions-banner" className={cn("wc-stable-theme flex flex-wrap items-center gap-2 border-b border-amber-700/40 bg-amber-900/20 px-3 py-2 text-sm text-amber-100", topSafe && "pt-[var(--wc-safe-top,0px)]")}>
      <span className="min-w-0 flex-1 font-medium">{t(strings.recoverableSessions.heading, { count: rows.length })}</span>
      <button type="button" onClick={() => void recoverAll()} disabled={busy !== null || !rows.some(r => r.recoverable)} className="rounded border border-amber-400/50 px-2 py-1 text-xs disabled:opacity-50">{t(strings.recoverableSessions.reattachAll)}</button>
      <button type="button" onClick={() => setOpen(true)} className="rounded border border-amber-400/50 px-2 py-1 text-xs">{t(strings.recoverableSessions.view)}</button>
      <button type="button" onClick={() => void dismissAll()} disabled={busy !== null} className="rounded border border-amber-400/30 px-2 py-1 text-xs disabled:opacity-50">{t(strings.recoverableSessions.dismissAll)}</button>
      {error ? <span role="alert" className="basis-full text-xs text-red-300">{error}</span> : null}
    </div>
    <DrawerShell open={open} onClose={() => setOpen(false)} title={t(strings.recoverableSessions.heading, { count: rows.length })} panelTestId="recoverable-sessions-drawer" size="full">
      <ul className="h-full overflow-y-auto divide-y divide-wc-default p-3">
        {rows.map(row => <li key={row.id} data-testid={`recoverable-row-${row.id}`} className="flex items-center gap-2 py-3">
          <span aria-hidden className="h-3 w-3 rounded-full border border-white/30" style={{ backgroundColor: row.header_color || "transparent" }} />
          <div className="min-w-0 flex-1"><div className="truncate font-medium">{row.pane_name || row.id.slice(0, 8)}</div><div className="truncate text-xs text-wc-text-muted">{row.agent_type ?? t(strings.recoverableSessions.agentNone)}{row.cwd ? ` · ${row.cwd}` : ""}{row.group_name ? ` · ${row.group_name}` : ""}</div></div>
          <button type="button" data-testid={`recoverable-row-${row.id}-recover`} disabled={!row.recoverable || busy !== null} title={row.recoverable ? t(strings.recoverableSessions.reattachTitle) : row.not_recoverable_reason} onClick={() => void recover(row.id)} className="rounded border px-2 py-1 text-xs disabled:opacity-50">{t(strings.recoverableSessions.reattach)}</button>
          <button type="button" data-testid={`recoverable-row-${row.id}-dismiss`} disabled={busy !== null} onClick={() => void dismiss(row.id)} className="rounded border px-2 py-1 text-xs disabled:opacity-50">{t(strings.recoverableSessions.dismiss)}</button>
        </li>)}
      </ul>
    </DrawerShell>
  </>;
}
