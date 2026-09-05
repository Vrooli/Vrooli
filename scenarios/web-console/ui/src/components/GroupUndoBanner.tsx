import { useEffect, useState } from "react";
import { RotateCcw, X } from "lucide-react";
import { useTranslation } from "react-i18next";

import { strings } from "../consts/strings";
import { UNDO_WINDOW_MS } from "../hooks/useGroupActions";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { IconButton } from "@vrooli/react-component-library/IconButton";

// [REQ:P0-014f] Group Auto-Close With Undo

interface GroupUndoBannerProps {
  onUndo: () => Promise<boolean>;
  onDismiss: () => void;
}

/**
 * The reversal window after a group closes.
 *
 * Written here rather than adopted from the component library's UndoBanner:
 * that component takes no props and reads its records from an UndoManager
 * context this app does not mount, and wiring that provider would put a
 * library-owned service in charge of a state machine this scenario needs to
 * own — the snapshot has to survive a FAILED replay, which is the case that
 * matters most and the one a generic manager would discard.
 */
export default function GroupUndoBanner({ onUndo, onDismiss }: GroupUndoBannerProps) {
  const { t } = useTranslation();
  const snapshot = useWorkspaceStore((s) => s.closedGroupUndo);
  const [restoring, setRestoring] = useState(false);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    if (!snapshot) return undefined;
    setFailed(false);
    const timer = setTimeout(onDismiss, UNDO_WINDOW_MS);
    return () => { clearTimeout(timer); };
  }, [snapshot, onDismiss]);

  if (!snapshot) return null;

  const roleCount = snapshot.roles.length;
  const message = roleCount > 0
    ? t(strings.undo.groupClosedWithRoles, { name: snapshot.group.name, count: roleCount })
    : t(strings.undo.groupClosed, { name: snapshot.group.name });

  return (
    <div
      data-testid="group-undo-banner"
      role="status"
      className="pointer-events-auto fixed inset-x-4 bottom-4 z-40 mx-auto flex max-w-md items-center gap-3 rounded-xl border border-wc-default bg-wc-surface-raised px-4 py-3 shadow-xl sm:inset-x-auto sm:start-1/2 sm:-translate-x-1/2"
    >
      <span
        className="h-2.5 w-2.5 shrink-0 rounded-full"
        style={{ backgroundColor: snapshot.group.color }}
        aria-hidden
      />
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm text-wc-text-primary">{message}</p>
        {failed && (
          <p className="text-xs text-rose-300">{t(strings.undo.restoreFailed)}</p>
        )}
      </div>
      <button
        type="button"
        data-testid="group-undo-action"
        disabled={restoring}
        onClick={() => {
          setRestoring(true);
          void onUndo()
            .then((ok) => { setFailed(!ok); })
            .finally(() => { setRestoring(false); });
        }}
        className="inline-flex min-h-11 shrink-0 items-center gap-1.5 rounded-lg px-3 text-sm font-medium text-wc-accent transition hover:bg-wc-surface-input disabled:opacity-60"
      >
        <RotateCcw className="h-4 w-4" aria-hidden />
        {t(strings.undo.undo)}
      </button>
      <IconButton
        data-testid="group-undo-dismiss"
        onClick={onDismiss}
        aria-label={t(strings.undo.dismiss)}
        size="sm"
        className="shrink-0"
      >
        <X aria-hidden />
      </IconButton>
    </div>
  );
}
