/**
 * @vrooliComponentSource react-component-library:EmptyState
 * @vrooliComponentVersion 1.2.0
 * @vrooliComponentAdoption 08245e9a-4333-4e0a-ae3c-f6178d1d06e1
 * @vrooliComponentAppliedAt 2026-08-11T00:47:52Z
 * @vrooliComponentSourceSha256 f84b7d3f4e8ed390ceda158e2da04b05dcb3450eae8c07c530bdf3f1f1905a73
 * @vrooliComponentDriftHash b4b7f3cd5620c94a90860ef0610ddc51276ef145243000cd0df28de21d232a4f
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
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

export function EmptyState({ title, description, icon, action, className }: EmptyStateProps) {
  const titleId = `rcl-empty-state-${useId().replace(/:/g, "")}-title`;
  return (
    <>
      <style data-rcl-empty-state-styles dangerouslySetInnerHTML={{ __html: emptyStateStyles }} />
      <section data-rcl-empty-state className={className} aria-labelledby={titleId}>
        {icon ? (
          <div data-rcl-empty-state-icon aria-hidden="true">
            {icon}
          </div>
        ) : null}
        <div data-rcl-empty-state-copy>
          <h2 id={titleId} data-rcl-empty-state-title>
            {title}
          </h2>
          {description ? <p data-rcl-empty-state-description>{description}</p> : null}
        </div>
        {action ? <div data-rcl-empty-state-action>{action}</div> : null}
      </section>
    </>
  );
}
