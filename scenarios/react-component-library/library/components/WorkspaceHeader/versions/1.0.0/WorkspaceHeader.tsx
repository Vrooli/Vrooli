/**
 * @libraryId react-component-library:WorkspaceHeader
 * @version 1.0.0
 * @status released
 * @deps {"react":"^18"}
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

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
      <style
        data-rcl-workspace-header-styles
        dangerouslySetInnerHTML={{ __html: workspaceHeaderStyles }}
      />
      <div className="rcl-workspace-header__row">
        {leading ? (
          <div className="rcl-workspace-header__leading">{leading}</div>
        ) : null}
        <div className="rcl-workspace-header__copy">
          <h1 className="rcl-workspace-header__title">{title}</h1>
          {description ? (
            <p className="rcl-workspace-header__description">{description}</p>
          ) : null}
        </div>
        {primaryAction || actions ? (
          <div className="rcl-workspace-header__actions">
            {primaryAction}
            {actions}
          </div>
        ) : null}
      </div>
      {children ? (
        <div className="rcl-workspace-header__children">{children}</div>
      ) : null}
    </Element>
  );
});
