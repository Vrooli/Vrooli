/** @vrooliComponentSource react-component-library:PageHeader */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import type { ReactNode } from "react";

const pageHeaderStyles = `
[data-rcl-page-header] { display: flex; align-items: end; justify-content: space-between; gap: var(--space-sm); border-block-end: var(--border-hairline) solid var(--color-border); padding-block: var(--space-xs) var(--space-sm); }
[data-rcl-page-header-copy] { min-inline-size: 0; }
[data-rcl-page-header] h1 { margin: 0; color: var(--color-foreground); font: var(--text-heading-lg); letter-spacing: var(--tracking-tight); }
[data-rcl-page-header] p { max-inline-size: 52ch; margin: var(--space-3xs) 0 0; color: var(--color-muted-foreground); font: var(--text-body-sm); }
[data-rcl-page-header-actions] { display: flex; flex-wrap: wrap; align-items: center; justify-content: end; gap: var(--space-2xs); }
@media (max-width: 48rem) { [data-rcl-page-header] { align-items: start; flex-direction: column; } [data-rcl-page-header-actions] { inline-size: 100%; justify-content: start; } }
`;

export function PageHeader({
  title = "Page",
  description,
  actions,
}: {
  title?: string;
  description?: string;
  actions?: ReactNode;
}) {
  return (
    <>
      <StyleSheet name="pageheader-1-0-0-1" css={pageHeaderStyles} />
      <header data-rcl-page-header="">
        <div data-rcl-page-header-copy="">
          <h1>{title}</h1>
          {description ? <p>{description}</p> : null}
        </div>
        {actions ? <div data-rcl-page-header-actions="">{actions}</div> : null}
      </header>
    </>
  );
}
