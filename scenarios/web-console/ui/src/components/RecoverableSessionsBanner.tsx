import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";

import {
  dismissRecoverableSession,
  listRecoverableSessions,
  recoverSession,
  type RecoverableSession,
  type RecoverResult,
} from "../api/sessions";
import { strings } from "../consts/strings";

export interface RecoverableSessionsBannerProps {
  // Called after a successful recovery so the workspace can attach the new
  // pane. The new session id is the one returned by the API.
  onRecovered?: (result: RecoverResult) => void;
}

// RecoverableSessionsBanner surfaces awaiting_recovery rows from the API and
// invokes the recover endpoint on click. It hides when the list is empty so
// the normal workspace UI is unchanged after a clean restart.
//
// See: scenarios/web-console/docs/guides/SESSION_RECOVERY.md
export default function RecoverableSessionsBanner(props: RecoverableSessionsBannerProps) {
  const { onRecovered } = props;
  const { t } = useTranslation();
  const [rows, setRows] = useState<RecoverableSession[] | null>(null);
  const [busy, setBusy] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    try {
      const list = await listRecoverableSessions();
      setRows(list);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setRows([]);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const handleRecover = useCallback(
    async (id: string) => {
      setBusy(id);
      setError(null);
      try {
        const result = await recoverSession(id);
        if (onRecovered) onRecovered(result);
        await refresh();
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      } finally {
        setBusy(null);
      }
    },
    [onRecovered, refresh],
  );

  const handleDismiss = useCallback(
    async (id: string) => {
      setBusy(id);
      setError(null);
      try {
        await dismissRecoverableSession(id);
        await refresh();
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      } finally {
        setBusy(null);
      }
    },
    [refresh],
  );

  if (rows === null) return null;
  if (rows.length === 0) return null;

  return (
    <div
      data-testid="recoverable-sessions-banner"
      className="wc-stable-theme border-b border-amber-700/40 bg-amber-900/20 text-sm text-amber-100"
    >
      <div className="py-2 ps-[max(0.75rem,var(--wc-safe-left,0px))] pe-[max(0.75rem,var(--wc-safe-right,0px))] font-medium">
        {t(strings.recoverableSessions.heading, { count: rows.length })}
      </div>
      <ul className="divide-y divide-amber-700/30">
        {rows.map((row) => (
          <li
            key={row.id}
            data-testid={`recoverable-row-${row.id}`}
            className="flex items-center gap-2 py-2 ps-[max(0.75rem,var(--wc-safe-left,0px))] pe-[max(0.75rem,var(--wc-safe-right,0px))]"
          >
            <span className="font-mono text-xs">{row.id.slice(0, 8)}</span>
            <span className="text-xs">
              {t(strings.recoverableSessions.agentLabel, {
                agent: row.agent_type ?? t(strings.recoverableSessions.agentNone),
              })}
            </span>
            {row.cwd ? (
              <span className="truncate text-xs opacity-70">
                {t(strings.recoverableSessions.cwdLabel, { cwd: row.cwd })}
              </span>
            ) : null}
            <div className="ml-auto flex gap-2">
              <button
                type="button"
                disabled={!row.recoverable || busy === row.id}
                onClick={() => handleRecover(row.id)}
                className="rounded border border-amber-400/50 px-2 py-0.5 text-xs hover:bg-amber-700/30 disabled:opacity-50"
                title={
                  row.recoverable
                    ? t(strings.recoverableSessions.reattachTitle)
                    : row.not_recoverable_reason
                }
                data-testid={`recoverable-row-${row.id}-recover`}
              >
                {t(strings.recoverableSessions.reattach)}
              </button>
              <button
                type="button"
                disabled={busy === row.id}
                onClick={() => handleDismiss(row.id)}
                className="rounded border border-amber-400/30 px-2 py-0.5 text-xs hover:bg-amber-700/20 disabled:opacity-50"
                title={t(strings.recoverableSessions.dismissTitle)}
                data-testid={`recoverable-row-${row.id}-dismiss`}
              >
                {t(strings.recoverableSessions.dismiss)}
              </button>
            </div>
          </li>
        ))}
      </ul>
      {error ? (
        <div className="py-2 ps-[max(0.75rem,var(--wc-safe-left,0px))] pe-[max(0.75rem,var(--wc-safe-right,0px))] text-xs text-red-300" role="alert">
          {error}
        </div>
      ) : null}
    </div>
  );
}
