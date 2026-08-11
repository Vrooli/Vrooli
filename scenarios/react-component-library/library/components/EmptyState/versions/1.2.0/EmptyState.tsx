/**
 * @libraryId react-component-library:EmptyState
 * @version 1.2.0
 * @status released
 * @deps {"react":"^18"}
 */
import { useId, type ReactNode } from "react";
import { emptyStateStyles } from "./styles";

export interface EmptyStateProps {
  title: string;
  description?: string;
  icon?: ReactNode;
  action?: ReactNode;
  className?: string;
}

export function EmptyState({
  title,
  description,
  icon,
  action,
  className,
}: EmptyStateProps) {
  const titleId = `rcl-empty-state-${useId().replace(/:/g, "")}-title`;
  return (
    <>
      <style
        data-rcl-empty-state-styles
        dangerouslySetInnerHTML={{ __html: emptyStateStyles }}
      />
      <section
        data-rcl-empty-state
        className={className}
        aria-labelledby={titleId}
      >
        {icon ? (
          <div data-rcl-empty-state-icon aria-hidden="true">
            {icon}
          </div>
        ) : null}
        <div data-rcl-empty-state-copy>
          <h2 id={titleId} data-rcl-empty-state-title>
            {title}
          </h2>
          {description ? (
            <p data-rcl-empty-state-description>{description}</p>
          ) : null}
        </div>
        {action ? <div data-rcl-empty-state-action>{action}</div> : null}
      </section>
    </>
  );
}
