import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { AlertDialog } from "@vrooli/react-component-library/AlertDialog/2";
import { Checkbox } from "@vrooli/react-component-library/Checkbox/1";

import { strings } from "../consts/strings";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";

// [REQ:P0-014c] Group Assignment And Administration Split
// [REQ:P0-014f] Group Auto-Close With Undo

interface CloseGroupDialogProps {
  /**
   * Close the group itself: its members are released and it becomes undoable
   * for the length of the banner's window.
   */
  onCloseGroup: (groupId: string) => void;
  /** Close one session, the same way the tab menu's Close does (archive). */
  onCloseSession: (sessionId: string) => void;
}

/**
 * The one confirmation for closing a group.
 *
 * Closing a group on its own is cheap and reversible — the sessions survive
 * and the undo banner covers the group — so on that path this dialog is
 * mostly a statement of consequences. What it exists for is the second half:
 * the operator usually wants the sessions gone too, and doing that by hand,
 * one tab at a time, was the whole friction. That is a real destruction, so
 * it is opt-in, off by default, and the dialog says exactly where the
 * sessions go and what undo can and cannot bring back.
 */
export default function CloseGroupDialog({ onCloseGroup, onCloseSession }: CloseGroupDialogProps) {
  const { t } = useTranslation();
  const groupId = useWorkspaceStore((s) => s.closeGroupTarget);
  const setCloseGroupTarget = useWorkspaceStore((s) => s.setCloseGroupTarget);
  const groups = useWorkspaceStore((s) => s.groups);
  const panes = useWorkspaceStore((s) => s.panes);
  const roles = useWorkspaceStore((s) => s.roles);
  const [alsoCloseSessions, setAlsoCloseSessions] = useState(false);

  // The safe option is the default on EVERY open. A checkbox that remembered
  // "yes, destroy the sessions" would turn one deliberate choice into a
  // standing one.
  useEffect(() => {
    if (groupId) setAlsoCloseSessions(false);
  }, [groupId]);

  const group = groups.find((g) => g.id === groupId) ?? null;
  if (!group) return null;

  const memberIds = panes.filter((pane) => pane.groupId === group.id).map((pane) => pane.sessionId);
  const waitingCount = roles.filter((role) => role.groupId === group.id && !role.sessionId).length;
  const dismiss = () => { setCloseGroupTarget(null); };

  const confirm = () => {
    // The group goes first so its undo snapshot records every member while
    // they are still members. Archiving first would let auto-close fire and
    // leave this handler closing a group that no longer exists.
    onCloseGroup(group.id);
    if (alsoCloseSessions) {
      for (const sessionId of memberIds) onCloseSession(sessionId);
    }
    setCloseGroupTarget(null);
  };

  return (
    <AlertDialog
      open
      destructive={alsoCloseSessions}
      title={t(strings.groupContextMenu.closeTitle, { name: group.name })}
      description={t(strings.groupContextMenu.closeBody)}
      cancelLabel={t(strings.roles.cancel)}
      confirmLabel={t(strings.manageGroups.closeGroup)}
      onCancel={dismiss}
      onConfirm={confirm}
      testIdPrefix="close-group"
    >
      <div className="space-y-3">
        <p data-testid="close-group-summary" className="text-sm text-wc-text-muted">
          {t(strings.manageGroups.sessionCount, { count: memberIds.length })}
          {waitingCount > 0 && ` · ${t(strings.roles.waitingCount, { count: waitingCount })}`}
        </p>

        {memberIds.length > 0 && (
          <Checkbox
            data-testid="close-group-also-sessions"
            label={t(strings.groupContextMenu.closeSessionsLabel)}
            description={t(strings.groupContextMenu.closeSessionsHint)}
            checked={alsoCloseSessions}
            onCheckedChange={setAlsoCloseSessions}
          />
        )}

        {/* Which of the two things is about to happen, said in full rather
            than left to be discovered after the fact. */}
        <p data-testid="close-group-consequence" className="text-[11px] text-wc-text-faint">
          {memberIds.length === 0
            ? t(strings.groupContextMenu.closeUndoNote)
            : alsoCloseSessions
              ? t(strings.groupContextMenu.closeUndoNote)
              : `${t(strings.groupContextMenu.closeKeepHint)} ${t(strings.groupContextMenu.closeUndoNote)}`}
        </p>
      </div>
    </AlertDialog>
  );
}
