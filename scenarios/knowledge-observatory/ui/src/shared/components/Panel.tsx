// DOC: docs/concepts/ARCHITECTURE.md#ui-surface
import type { ReactNode } from "react";
import { cn } from "../lib/utils";

export type PanelProps = {
  children: ReactNode;
  className?: string;
  testId?: string;
};

export function Panel({ children, className, testId }: PanelProps) {
  return (
    <section className={cn("ko-panel ko-section", className)} data-testid={testId}>
      {children}
    </section>
  );
}

export type PanelHeaderProps = {
  title: string;
  description?: string;
  icon?: ReactNode;
  className?: string;
  titleClassName?: string;
};

export function PanelHeader({
  title,
  description,
  icon,
  className,
  titleClassName,
}: PanelHeaderProps) {
  return (
    <div className={cn("ko-section-header mb-2", className)}>
      {icon ? <span className="ko-section-icon">{icon}</span> : null}
      <div>
        <h2 className={cn("ko-text-lg font-semibold", titleClassName)}>{title}</h2>
        {description ? <p className="ko-section-description">{description}</p> : null}
      </div>
    </div>
  );
}
