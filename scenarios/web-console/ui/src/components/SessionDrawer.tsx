// DOC: docs/concepts/ARCHITECTURE.md#file-map
// DOC: docs/internal/SEAMS.md#1-entry--presentation
import { useState, useCallback, useEffect, useRef } from "react";
import { X, Trash2, Terminal, Clock, Timer, AlertCircle } from "lucide-react";
import { type SessionInfo, type PolicyMode, updateSessionPolicy, toErrorInfo } from "../lib/api";
import { getShellName, formatSessionTime, truncateId } from "../lib/format";
import { pluralize } from "../lib/pluralize";
import { POLICY_OPTIONS, policyKey } from "../consts/policy-options";
import { useCountdown } from "../hooks/useCountdown";
import { cn } from "../lib/classnames";
import { Button } from "./ui/button";

import ProviderHealthPanel from "./ProviderHealthPanel";

// [REQ:P0-008a] Drawer Layout Component
// [REQ:P0-008b] Session Status and Controls
// [REQ:P1-001b] Policy Configuration UI
// [REQ:P1-003b] Provider Health Dashboard

interface SessionDrawerProps {
  open: boolean;
  onClose: () => void;
  sessions: Array<{ session: SessionInfo }>;
  onDeleteSession: (id: string) => void;
  onSelectSession?: (id: string) => void;
}

// [REQ:P1-001b] Policy Configuration UI - per-session policy row
function SessionPolicyControl({
  session,
  onPolicyChange,
}: {
  session: SessionInfo;
  onPolicyChange: (id: string, mode: PolicyMode, duration?: string) => void;
}) {
  const currentKey = policyKey(session.policy.mode, session.policy.duration);
  const countdown = useCountdown(
    session.created_at,
    session.policy.mode,
    session.policy.duration,
  );

  return (
    <div className="mt-1 flex items-center gap-1.5">
      <Timer className="h-3 w-3 shrink-0 text-wc-text-faint" />
      <select
        data-testid={`policy-select-${session.id}`}
        className="h-5 rounded border border-wc-default bg-wc-surface px-1 text-xs text-wc-text-secondary focus:border-wc-accent focus:outline-none"
        value={currentKey}
        onChange={(e) => {
          const val = e.target.value;
          if (val === "never") {
            onPolicyChange(session.id, "never");
          } else {
            const [mode, dur] = val.split(":") as [PolicyMode, string];
            onPolicyChange(session.id, mode, dur);
          }
        }}
      >
        {POLICY_OPTIONS.map((opt) => (
          <option key={policyKey(opt.mode, opt.duration)} value={policyKey(opt.mode, opt.duration)}>
            {opt.label}
          </option>
        ))}
      </select>
      {countdown && (
        <span
          data-testid={`policy-countdown-${session.id}`}
          className="flex items-center gap-0.5 text-xs text-wc-text-faint"
        >
          <Clock className="h-2.5 w-2.5" />
          {countdown}
        </span>
      )}
    </div>
  );
}

export default function SessionDrawer({
  open,
  onClose,
  sessions,
  onDeleteSession,
  onSelectSession,
}: SessionDrawerProps) {
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null);
  const [policyError, setPolicyError] = useState<string | null>(null);
  const policyErrorTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Auto-dismiss policy error after 5 seconds
  useEffect(() => {
    return () => {
      if (policyErrorTimer.current) clearTimeout(policyErrorTimer.current);
    };
  }, []);

  const handleDelete = useCallback(
    (id: string) => {
      if (confirmDelete === id) {
        onDeleteSession(id);
        setConfirmDelete(null);
      } else {
        setConfirmDelete(id);
      }
    },
    [confirmDelete, onDeleteSession],
  );

  // [REQ:P1-001b] Policy Configuration UI - update handler
  const handlePolicyChange = useCallback(
    async (sessionId: string, mode: PolicyMode, duration?: string) => {
      try {
        await updateSessionPolicy(sessionId, { mode, duration });
        setPolicyError(null);
      } catch (err) {
        const info = toErrorInfo(err);
        setPolicyError(info.recovery || info.message);
        if (policyErrorTimer.current) clearTimeout(policyErrorTimer.current);
        policyErrorTimer.current = setTimeout(() => setPolicyError(null), 5000);
      }
    },
    [],
  );

  return (
    <>
      {/* Backdrop */}
      {open && (
        <div
          data-testid="drawer-backdrop"
          className="fixed inset-0 z-40 bg-wc-backdrop transition-opacity"
          onClick={onClose}
        />
      )}

      {/* Drawer panel */}
      <div
        data-testid="session-drawer"
        className={cn("fixed inset-y-0 left-0 z-50 flex w-72 flex-col border-r border-wc-default bg-wc-surface-raised shadow-xl transition-transform duration-200", open ? "translate-x-0" : "-translate-x-full")}
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-wc-default px-4 py-3">
          <h2 className="text-sm font-semibold uppercase tracking-wider text-wc-text-muted">
            Sessions
          </h2>
          <Button
            data-testid="drawer-close"
            variant="ghost"
            size="icon"
            className="h-7 w-7"
            onClick={onClose}
          >
            <X className="h-4 w-4" />
          </Button>
        </div>

        {/* Policy error feedback */}
        {policyError && (
          <div
            data-testid="policy-error"
            className="mx-4 mt-2 flex items-start gap-2 rounded border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-300"
          >
            <AlertCircle className="mt-0.5 h-3 w-3 shrink-0" />
            <span>{policyError}</span>
            <button
              className="ml-auto shrink-0 text-red-400 hover:text-red-200"
              onClick={() => setPolicyError(null)}
            >
              <X className="h-3 w-3" />
            </button>
          </div>
        )}

        {/* Session list */}
        <div className="flex-1 overflow-y-auto">
          {sessions.length === 0 ? (
            <div className="p-4 text-center text-sm text-wc-text-faint">
              No active sessions
            </div>
          ) : (
            <ul className="divide-y divide-white/5">
              {sessions.map(({ session }) => (
                <li
                  key={session.id}
                  data-testid={`drawer-session-${session.id}`}
                  className="group px-4 py-3 hover:bg-white/5"
                >
                  <div className="flex items-center gap-2">
                    <Terminal className="h-4 w-4 shrink-0 text-wc-accent" />
                    <button
                      className="min-w-0 flex-1 text-left"
                      onClick={() => onSelectSession?.(session.id)}
                    >
                      <div className="truncate text-sm font-medium text-wc-text-secondary">
                        {getShellName(session.shell)}
                      </div>
                      <div className="truncate text-xs text-wc-text-faint">
                        {truncateId(session.id)} &middot;{" "}
                        {formatSessionTime(session.created_at)}
                      </div>
                    </button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7 shrink-0 opacity-0 transition group-hover:opacity-100"
                      onClick={() => handleDelete(session.id)}
                      title={
                        confirmDelete === session.id
                          ? "Click again to confirm"
                          : "Terminate session"
                      }
                    >
                      <Trash2
                        className={cn("h-3.5 w-3.5", confirmDelete === session.id ? "text-wc-error-detail" : "text-wc-text-muted")}
                      />
                    </Button>
                  </div>
                  <SessionPolicyControl
                    session={session}
                    onPolicyChange={handlePolicyChange}
                  />
                </li>
              ))}
            </ul>
          )}
        </div>

        {/* Provider health panel - [REQ:P1-003b] */}
        <div className="border-t border-wc-default px-4 py-3">
          <ProviderHealthPanel open={open} />
        </div>

        {/* Footer stats */}
        <div className="border-t border-wc-default px-4 py-2 text-xs text-wc-text-faint">
          {sessions.length} active {pluralize(sessions.length, "session")}
        </div>
      </div>
    </>
  );
}
