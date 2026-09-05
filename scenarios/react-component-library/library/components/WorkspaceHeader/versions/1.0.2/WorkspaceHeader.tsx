/**
 * @libraryId react-component-library:WorkspaceHeader
 * @displayName Workspace Header
 * @version 1.0.2
 * @tags ["layout","header","workspace","navigation"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import type { ReactNode } from "react";
export const workspaceHeaderStyles = `
[data-rcl-workspace-header] { inline-size: 100%; min-inline-size: 0; flex-shrink: 0; overflow: hidden; border-block-end: var(--border-hairline) solid var(--color-border); background: var(--color-surface); color: var(--color-foreground); }
[data-rcl-workspace-header] .rcl-workspace-header__row { display: flex; min-block-size: var(--tap-target-min); min-inline-size: 0; align-items: center; gap: var(--space-xs); padding: var(--space-2xs) var(--space-xs); }
[data-rcl-workspace-header] .rcl-workspace-header__leading { display: flex; min-inline-size: 0; flex-shrink: 0; align-items: center; }
[data-rcl-workspace-header] .rcl-workspace-header__copy { min-inline-size: 0; flex: 1; }
[data-rcl-workspace-header] .rcl-workspace-header__title { overflow: hidden; margin: 0; color: var(--color-foreground); font-size: var(--text-heading-size); font-weight: 700; line-height: var(--text-heading-line); letter-spacing: -0.01em; text-overflow: ellipsis; white-space: nowrap; }
[data-rcl-workspace-header] .rcl-workspace-header__description { overflow: hidden; margin: var(--space-3xs) 0 0; color: var(--color-muted-foreground); font-size: var(--text-caption-size); line-height: var(--text-caption-line); text-overflow: ellipsis; white-space: nowrap; }
[data-rcl-workspace-header] .rcl-workspace-header__actions { display: flex; min-inline-size: 0; flex-shrink: 0; align-items: center; gap: var(--space-2xs); }
[data-rcl-workspace-header] .rcl-workspace-header__actions button { min-block-size: var(--tap-target-min); border: var(--border-hairline) solid var(--color-primary); border-radius: var(--radius-control); background: var(--color-primary); color: var(--color-primary-foreground); padding-inline: var(--space-sm); font: 600 var(--text-body-sm-size)/var(--text-body-sm-line) var(--font-sans); cursor: pointer; transition: background-color var(--dur-quick) var(--ease-standard), border-color var(--dur-quick) var(--ease-standard), transform var(--dur-quick) var(--ease-standard); }
[data-rcl-workspace-header] .rcl-workspace-header__actions button:hover { filter: brightness(0.96); }
[data-rcl-workspace-header] .rcl-workspace-header__actions button:active { transform: translateY(1px); }
[data-rcl-workspace-header] .rcl-workspace-header__children { min-inline-size: 0; border-block-start: var(--border-hairline) solid var(--color-border); padding-inline: var(--space-xs); }
@media (min-width: 40rem) { [data-rcl-workspace-header] .rcl-workspace-header__row { padding-inline: var(--space-sm); } [data-rcl-workspace-header] .rcl-workspace-header__children { padding-inline: var(--space-sm); } }
@media (max-width: 30rem) { [data-rcl-workspace-header] .rcl-workspace-header__row { flex-wrap: wrap; align-items: flex-start; } [data-rcl-workspace-header] .rcl-workspace-header__copy { flex-basis: calc(100% - var(--space-xs)); } [data-rcl-workspace-header] .rcl-workspace-header__actions { inline-size: 100%; justify-content: flex-start; } }
`;
export interface WorkspaceHeaderProps {
  title: ReactNode;
  description?: ReactNode;
  leading?: ReactNode;
  primaryAction?: ReactNode;
  actions?: ReactNode;
  children?: ReactNode;
  className?: string;
  as?: "header" | "div";
}

/** A structural header: callers own navigation state and action behavior. */
export const WorkspaceHeader = withClassName(function WorkspaceHeader({
  title,
  description,
  leading,
  primaryAction,
  actions,
  children,
  className,
  as: Element = "header",
}: WorkspaceHeaderProps) {
  return (
    <Element
      data-testid="workspace-header"
      data-rcl-workspace-header
      className={["rcl-workspace-header", className].filter(Boolean).join(" ")}
    >
      <StyleSheet name="workspace-header-1-0-1" css={workspaceHeaderStyles} />
      <div className="rcl-workspace-header__row">
        {leading ? <div className="rcl-workspace-header__leading">{leading}</div> : null}
        <div className="rcl-workspace-header__copy">
          <h1 className="rcl-workspace-header__title">{title}</h1>
          {description ? <p className="rcl-workspace-header__description">{description}</p> : null}
        </div>
        {primaryAction || actions ? (
          <div className="rcl-workspace-header__actions">
            {primaryAction}
            {actions}
          </div>
        ) : null}
      </div>
      {children ? <div className="rcl-workspace-header__children">{children}</div> : null}
    </Element>
  );
});
