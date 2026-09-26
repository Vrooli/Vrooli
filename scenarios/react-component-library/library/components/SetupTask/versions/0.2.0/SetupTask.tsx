/**
 * @libraryId react-component-library:SetupTask
 * @displayName Setup Task
 * @description Provider-neutral responsive setup task surface with status, target/account context, guidance, body, and owner actions.
 * @version 0.2.0
 * @tags ["setup","credentials","onboarding","settings"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import type { ReactNode } from "react";
import { StatusBadge, type StatusTone } from "@vrooli/react-component-library/StatusBadge/1";
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";

export type SetupTaskStatus =
  | "ready"
  | "checking"
  | "needs_attention"
  | "optional"
  | "unavailable"
  | "unknown";

export interface SetupTaskProps {
  title: string;
  purpose?: string;
  target?: string;
  account?: string;
  consumerScope?: string;
  checkedAt?: string;
  nextAction?: ReactNode;
  status?: SetupTaskStatus;
  statusLabel?: string;
  guidance?: ReactNode;
  children?: ReactNode;
  actions?: ReactNode;
  testId?: string;
}

const statusLabels: Record<SetupTaskStatus, string> = {
  ready: "Ready",
  checking: "Checking",
  needs_attention: "Needs attention",
  optional: "Optional",
  unavailable: "Unavailable",
  unknown: "Unknown",
};

const statusTones: Record<SetupTaskStatus, StatusTone> = {
  ready: "success",
  checking: "info",
  needs_attention: "warning",
  optional: "neutral",
  unavailable: "danger",
  unknown: "neutral",
};

const css = `
[data-rcl-setup-task] { box-sizing: border-box; display: grid; min-inline-size: 0; gap: var(--space-md); padding: var(--space-lg); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-card); background: var(--color-surface-raised); color: var(--color-foreground); box-shadow: var(--elev-subtle); }
[data-rcl-setup-task] *, [data-rcl-setup-task] *::before, [data-rcl-setup-task] *::after { box-sizing: border-box; }
[data-rcl-setup-task] [data-rcl-setup-task-header] { display: flex; min-inline-size: 0; align-items: flex-start; justify-content: space-between; gap: var(--space-md); }
[data-rcl-setup-task] [data-rcl-setup-task-heading] { min-inline-size: 0; }
[data-rcl-setup-task] [data-rcl-setup-task-eyebrow] { margin: 0 0 var(--space-2xs); color: var(--color-muted-foreground); font: var(--text-caption); letter-spacing: .05em; text-transform: uppercase; }
[data-rcl-setup-task] h3 { margin: 0; color: var(--color-foreground); font: var(--text-heading-sm); overflow-wrap: anywhere; }
[data-rcl-setup-task] [data-rcl-setup-task-purpose] { max-inline-size: 68ch; margin: var(--space-xs) 0 0; color: var(--color-muted-foreground); font: var(--text-body-sm); line-height: 1.5; overflow-wrap: anywhere; }
[data-rcl-setup-task] [data-rcl-setup-task-meta] { display: flex; min-inline-size: 0; flex-wrap: wrap; gap: var(--space-xs) var(--space-lg); margin: 0; padding-block: var(--space-sm); border-block: var(--border-hairline) solid var(--color-border); }
[data-rcl-setup-task] [data-rcl-setup-task-meta] div { min-inline-size: 8rem; }
[data-rcl-setup-task] [data-rcl-setup-task-meta] dt { color: var(--color-muted-foreground); font: var(--text-caption); }
[data-rcl-setup-task] [data-rcl-setup-task-meta] dd { margin: var(--space-2xs) 0 0; color: var(--color-foreground); font: var(--text-body-sm); overflow-wrap: anywhere; }
[data-rcl-setup-task] [data-rcl-setup-task-next-action] { display: grid; gap: var(--space-2xs); padding: var(--space-sm) var(--space-md); border-radius: var(--radius-control); background: var(--color-surface-muted); color: var(--color-foreground); font: var(--text-body-sm); }
[data-rcl-setup-task] [data-rcl-setup-task-next-action-label] { color: var(--color-muted-foreground); font: var(--text-caption); }
[data-rcl-setup-task] [data-rcl-setup-task-guidance] { display: grid; gap: var(--space-xs); padding: var(--space-sm) var(--space-md); border-inline-start: 3px solid var(--color-primary); border-radius: var(--radius-control); background: var(--color-surface-muted); color: var(--color-foreground); font: var(--text-body-sm); }
[data-rcl-setup-task] [data-rcl-setup-task-body] { min-inline-size: 0; }
[data-rcl-setup-task] [data-rcl-setup-task-actions] { display: flex; min-inline-size: 0; flex-wrap: wrap; align-items: center; gap: var(--space-sm); }
@media (max-width: 40rem) { [data-rcl-setup-task] { padding: var(--space-md); } [data-rcl-setup-task] [data-rcl-setup-task-header] { flex-direction: column; } [data-rcl-setup-task] [data-rcl-setup-task-meta] { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); } [data-rcl-setup-task] [data-rcl-setup-task-meta] div { min-inline-size: 0; } }
`;

/** Provider-neutral setup-task chrome. Owners supply guidance, controls, and lifecycle operations. */
export function SetupTask({
  title,
  purpose,
  target,
  account,
  consumerScope,
  checkedAt,
  nextAction,
  status = "unknown",
  statusLabel,
  guidance,
  children,
  actions,
  testId = "setup-task",
}: SetupTaskProps) {
  return (
    <article data-rcl-setup-task data-testid={testId} aria-label={title}>
      <StyleSheet libraryId="react-component-library:SetupTask" version="0.2.0-draft.1" css={css} />
      <header data-rcl-setup-task-header>
        <div data-rcl-setup-task-heading>
          <p data-rcl-setup-task-eyebrow>Setup task</p>
          <h3>{title}</h3>
          {purpose ? <p data-rcl-setup-task-purpose>{purpose}</p> : null}
        </div>
        <StatusBadge role="status" tone={statusTones[status]}>
          {statusLabel || statusLabels[status]}
        </StatusBadge>
      </header>
      {target || account || consumerScope || checkedAt ? (
        <dl data-rcl-setup-task-meta>
          {target ? (
            <div>
              <dt>Target</dt>
              <dd>{target}</dd>
            </div>
          ) : null}
          {account ? (
            <div>
              <dt>Account</dt>
              <dd>{account}</dd>
            </div>
          ) : null}
          {consumerScope ? (
            <div>
              <dt>Consumers</dt>
              <dd>{consumerScope}</dd>
            </div>
          ) : null}
          {checkedAt ? (
            <div>
              <dt>Checked</dt>
              <dd>{checkedAt}</dd>
            </div>
          ) : null}
        </dl>
      ) : null}
      {guidance ? <div data-rcl-setup-task-guidance>{guidance}</div> : null}
      {nextAction ? (
        <div data-rcl-setup-task-next-action>
          <span data-rcl-setup-task-next-action-label>Next action</span>
          <div>{nextAction}</div>
        </div>
      ) : null}
      {children ? <div data-rcl-setup-task-body>{children}</div> : null}
      {actions ? <footer data-rcl-setup-task-actions>{actions}</footer> : null}
    </article>
  );
}

export default SetupTask;
