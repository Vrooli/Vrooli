import { CirclePlay, MoreVertical, Send } from "lucide-react";
import { useTranslation } from "react-i18next";

import { strings } from "../consts/strings";
import { cn } from "../lib/classnames";
import type { RoleMeta, TabGroupMeta } from "../stores/useWorkspaceStore";

// [REQ:P0-014e] Waiting Roles

interface RoleRowProps {
  role: RoleMeta;
  group: TabGroupMeta;
  /** Start the role's command. The row becomes a pane once the session exists. */
  onStart: (role: RoleMeta) => void;
  /** Open the handoff composer aimed at this role. Omitted when it is the only member. */
  onHandoff?: (role: RoleMeta) => void;
  /** Open the role's overflow menu (rename, edit prompt, delete). */
  onOpenMenu: (role: RoleMeta, position: { x: number; y: number }) => void;
  /** Last member of its group, so the row closes the block's border. */
  isLastInGroup: boolean;
  /** Compact presentation for the horizontal tab strip. */
  variant?: "sidebar" | "tab";
}

/**
 * A role that has not started.
 *
 * The dashed border is the whole message: this is a position the operator has
 * reserved, not a session that is running. A solid row would read as a
 * terminal that is merely quiet, which is the one impression this must not
 * give — the operator needs to know nothing is happening here yet.
 */
export default function RoleRow({
  role,
  group,
  onStart,
  onHandoff,
  onOpenMenu,
  isLastInGroup,
  variant = "sidebar",
}: RoleRowProps) {
  const { t } = useTranslation();
  const startLabel = t(strings.roles.startRole, { label: role.label });

  if (variant === "tab") {
    return (
      <button
        type="button"
        data-testid={`tab-waiting-role-${role.id}`}
        onClick={() => { onStart(role); }}
        title={role.command || startLabel}
        aria-label={startLabel}
        className="group flex h-full shrink-0 items-center gap-1.5 border-r border-dashed border-wc-default px-3 text-xs text-wc-text-muted transition-colors hover:bg-wc-surface-input hover:text-wc-text-primary"
        style={{ borderLeft: `2px dashed ${group.color}` }}
      >
        <CirclePlay className="h-3.5 w-3.5 shrink-0" aria-hidden />
        <span className="max-w-[110px] truncate">{role.label}</span>
        <span className="rounded bg-wc-surface-input px-1 text-[10px] uppercase tracking-wide">
          {t(strings.roles.waiting)}
        </span>
      </button>
    );
  }

  return (
    <div
      data-testid={`sidebar-waiting-role-${role.id}`}
      className={cn(
        "group relative mb-1 flex w-full items-center gap-2 border border-s border-dashed border-wc-default bg-wc-surface-base/25 px-2 py-2 text-start",
        isLastInGroup ? "mb-2 rounded-b" : "rounded-none",
      )}
      style={{ borderLeftColor: group.color, borderLeftStyle: "solid" }}
    >
      <button
        type="button"
        data-testid={`sidebar-waiting-role-start-${role.id}`}
        onClick={() => { onStart(role); }}
        aria-label={startLabel}
        title={role.command || startLabel}
        className="flex min-w-0 flex-1 items-center gap-2 text-start"
      >
        <CirclePlay className="h-4 w-4 shrink-0 text-wc-text-faint transition group-hover:text-wc-accent" aria-hidden />
        <span className="min-w-0 flex-1">
          <span className="block truncate text-xs font-medium text-wc-text-secondary">{role.label}</span>
          <span className="block truncate text-[11px] text-wc-text-faint">
            {role.command || t(strings.roles.noCommand)}
          </span>
        </span>
      </button>

      <span className="shrink-0 rounded bg-wc-surface-input px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-wc-text-faint">
        {t(strings.roles.waiting)}
      </span>

      {onHandoff && (
        <button
          type="button"
          data-testid={`sidebar-waiting-role-handoff-${role.id}`}
          onClick={() => { onHandoff(role); }}
          aria-label={t(strings.handoff.handOffToRole, { label: role.label })}
          title={t(strings.handoff.handOffToRole, { label: role.label })}
          className="shrink-0 rounded p-1 text-wc-text-faint transition hover:bg-wc-surface-input hover:text-wc-text-primary"
        >
          <Send className="h-3.5 w-3.5" aria-hidden />
        </button>
      )}

      <button
        type="button"
        data-testid={`sidebar-waiting-role-menu-${role.id}`}
        onClick={(event) => {
          event.stopPropagation();
          const rect = event.currentTarget.getBoundingClientRect();
          onOpenMenu(role, { x: rect.left, y: rect.bottom });
        }}
        aria-label={t(strings.roles.roleOptions, { label: role.label })}
        title={t(strings.roles.roleOptions, { label: role.label })}
        className="shrink-0 rounded p-1 text-wc-text-faint transition hover:bg-wc-surface-input hover:text-wc-text-primary"
      >
        <MoreVertical className="h-3.5 w-3.5" aria-hidden />
      </button>
    </div>
  );
}
