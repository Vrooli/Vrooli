// DOC: docs/concepts/ARCHITECTURE.md#file-map
// [REQ:P0-003a] Session List Management
// [REQ:P1-001a] Session Policy Overview
import { useState, useEffect, useCallback, useRef } from "react";
import { ArrowLeft, Terminal, Trash2, RefreshCw, Clock, Timer } from "lucide-react";
import { Button } from "../components/ui/button";
import { cn } from "../lib/classnames";
import {
  type SessionInfo,
  type PolicyMode,
  listSessions,
  deleteSession,
  updateSessionPolicy,
} from "../lib/api";
import { getShellName, formatSessionTime, truncateId } from "../lib/format";
import { POLICY_OPTIONS, policyKey } from "../consts/policy-options";
import { useCountdown } from "../hooks/useCountdown";

interface SessionsPageProps {
  onBack: () => void;
}

function PolicyCountdown({ createdAt, mode, duration }: { createdAt: string; mode: PolicyMode; duration?: string }) {
  const remaining = useCountdown(createdAt, mode, duration);
  if (!remaining) return null;
  return (
    <span className="flex items-center gap-1 text-xs text-wc-text-faint">
      <Clock className="h-3 w-3" /> {remaining}
    </span>
  );
}

export default function SessionsPage({ onBack }: SessionsPageProps) {
  const [sessions, setSessions] = useState<SessionInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null);

  // Cancel stale refreshes: if a newer refresh starts before the previous
  // one completes, the older one is silently discarded so the UI always
  // shows the result of the latest request (no race condition).
  const refreshGenRef = useRef(0);

  const refresh = useCallback(async () => {
    const gen = ++refreshGenRef.current;
    setLoading(true);
    try {
      const data = await listSessions();
      if (gen !== refreshGenRef.current) return; // stale response
      setSessions(data);
    } catch {
      if (gen !== refreshGenRef.current) return;
      // Silent - show empty state
    } finally {
      if (gen === refreshGenRef.current) setLoading(false);
    }
  }, []);

  useEffect(() => { refresh(); }, [refresh]);

  const handleDelete = useCallback(async (id: string) => {
    if (confirmDelete !== id) {
      setConfirmDelete(id);
      return;
    }
    try {
      await deleteSession(id);
      setSessions((prev) => prev.filter((s) => s.id !== id));
      setConfirmDelete(null);
    } catch {
      // Silent failure
    }
  }, [confirmDelete]);

  const handlePolicyChange = useCallback(async (id: string, mode: PolicyMode, duration?: string) => {
    try {
      await updateSessionPolicy(id, { mode, duration });
      await refresh();
    } catch {
      // Silent failure
    }
  }, [refresh]);

  return (
    <div className="flex h-screen flex-col bg-wc-surface-base text-wc-text-primary">
      <div className="flex items-center justify-between border-b border-wc-default px-4 py-3">
        <div className="flex items-center gap-3">
          <Button
            data-testid="sessions-back"
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            onClick={onBack}
          >
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <h1 className="text-lg font-semibold">Sessions</h1>
          <span className="text-sm text-wc-text-faint">({sessions.length})</span>
        </div>
        <Button
          data-testid="sessions-refresh"
          variant="ghost"
          size="icon"
          className="h-8 w-8"
          onClick={refresh}
          disabled={loading}
        >
          <RefreshCw className={cn("h-4 w-4", loading && "animate-spin")} />
        </Button>
      </div>

      <div className="flex-1 overflow-y-auto">
        {loading && sessions.length === 0 ? (
          <div className="flex items-center justify-center py-12 text-sm text-wc-text-faint">
            Loading sessions...
          </div>
        ) : sessions.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 text-sm text-wc-text-faint">
            <Terminal className="h-8 w-8 mb-2 opacity-50" />
            No active sessions
          </div>
        ) : (
          <div className="divide-y divide-white/5">
            {sessions.map((session) => (
              <div
                key={session.id}
                data-testid={`session-row-${session.id}`}
                className="group flex items-center gap-4 px-4 py-3 hover:bg-white/5"
              >
                <Terminal className="h-5 w-5 shrink-0 text-wc-accent" />
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium text-wc-text-secondary">
                      {getShellName(session.shell)}
                    </span>
                    <span className="text-xs text-wc-text-faint font-mono">
                      {truncateId(session.id)}
                    </span>
                  </div>
                  <div className="flex items-center gap-3 mt-1">
                    <span className="text-xs text-wc-text-faint">
                      {formatSessionTime(session.created_at)}
                    </span>
                    <span className="text-xs text-wc-text-faint">
                      {session.cols}×{session.rows}
                    </span>
                    <PolicyCountdown
                      createdAt={session.created_at}
                      mode={session.policy.mode}
                      duration={session.policy.duration}
                    />
                  </div>
                </div>

                <div className="flex items-center gap-2">
                  <div className="flex items-center gap-1.5">
                    <Timer className="h-3 w-3 text-wc-text-faint" />
                    <select
                      data-testid={`session-policy-${session.id}`}
                      className="h-6 rounded border border-wc-default bg-wc-surface px-1.5 text-xs text-wc-text-secondary focus:border-wc-accent focus:outline-none"
                      value={policyKey(session.policy.mode, session.policy.duration)}
                      onChange={(e) => {
                        const val = e.target.value;
                        if (val === "never") {
                          handlePolicyChange(session.id, "never");
                        } else {
                          const dur = val.split(":")[1];
                          handlePolicyChange(session.id, "preset", dur);
                        }
                      }}
                    >
                      {POLICY_OPTIONS.map((opt) => (
                        <option
                          key={policyKey(opt.mode, opt.duration)}
                          value={policyKey(opt.mode, opt.duration)}
                        >
                          {opt.label}
                        </option>
                      ))}
                    </select>
                  </div>
                  <Button
                    data-testid={`session-delete-${session.id}`}
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7 opacity-0 transition group-hover:opacity-100"
                    onClick={() => handleDelete(session.id)}
                    title={confirmDelete === session.id ? "Click again to confirm" : "Terminate session"}
                  >
                    <Trash2
                      className={cn("h-3.5 w-3.5", confirmDelete === session.id ? "text-wc-error-detail" : "text-wc-text-muted")}
                    />
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
