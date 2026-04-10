// DOC: docs/concepts/ARCHITECTURE.md#ui-surface
import type { ReactNode } from "react";
import { Button, type ButtonProps } from "../ui/button";
import { Panel } from "./Panel";

export type SectionErrorAction = {
  label: string;
  onClick: () => void;
  variant?: ButtonProps["variant"];
};

export type SectionErrorStateProps = {
  title: string;
  description: string;
  errorMessage?: string;
  actions: SectionErrorAction[];
  footer?: ReactNode;
};

export function SectionErrorState({
  title,
  description,
  errorMessage,
  actions,
  footer,
}: SectionErrorStateProps) {
  return (
    <Panel>
      <h2 className="ko-text-lg font-semibold ko-text-danger-strong">{title}</h2>
      <p className="ko-text-sm ko-muted mt-2">{description}</p>
      {errorMessage ? <p className="ko-text-xs ko-text-danger-muted mt-2">{errorMessage}</p> : null}
      <div className="mt-4 flex flex-wrap gap-2">
        {actions.map((action) => (
          <Button key={action.label} onClick={action.onClick} variant={action.variant ?? "primary"}>
            {action.label}
          </Button>
        ))}
      </div>
      {footer ? <div className="mt-4">{footer}</div> : null}
    </Panel>
  );
}
