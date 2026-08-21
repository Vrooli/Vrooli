/**
 * @vrooliComponentSource react-component-library:WorkspaceHeader
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption a4ba3a93-b98a-411e-b945-cf4ff726f2de
 * @vrooliComponentAppliedAt 2026-08-11T00:47:44Z
 * @vrooliComponentSourceSha256 78e917f74d84fcdb855f8819291e366c96edc5b03e8db69fc0c0e971d0fe2342
 * @vrooliComponentDriftHash bb0472eaeace9ce4aa55146c776331544f66373b625e2b960517c6a50b8d8628
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import type { ReactNode } from "react";
import { workspaceHeaderStyles } from "./styles";
import { useComponentStyles } from "../../../../hooks/useComponentStyles";

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
export function WorkspaceHeader({
  title,
  description,
  leading,
  primaryAction,
  actions,
  children,
  className,
  as: Element = "header",
}: WorkspaceHeaderProps) {
  useComponentStyles("rcl-workspace-header", workspaceHeaderStyles);

  return (
    <Element
      data-testid="workspace-header"
      data-rcl-workspace-header
      className={["rcl-workspace-header", className].filter(Boolean).join(" ")}
    >
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
}
