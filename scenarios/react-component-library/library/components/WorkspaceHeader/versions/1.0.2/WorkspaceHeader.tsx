/**
 * @libraryId react-component-library:WorkspaceHeader
 * @displayName Workspace Header
 * @description A responsive page header with navigation, context, and action slots for operational workspaces.
 * @version 1.0.2
 * @tags ["layout","header","workspace","navigation"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import type { ReactNode } from "react";
import { workspaceHeaderStyles } from "./styles";

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
