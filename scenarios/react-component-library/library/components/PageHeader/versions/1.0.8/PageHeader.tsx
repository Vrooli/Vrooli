/**
 * @libraryId react-component-library:PageHeader
 * @displayName PageHeader
 * @version 1.0.8
 * @tags ["navigation","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:PageHeader */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import type { ReactNode } from "react";

const pageHeaderStyles = `
[data-rcl-page-header] { display: flex; align-items: end; justify-content: space-between; gap: var(--space-sm); border-block-end: var(--border-hairline) solid var(--color-border); padding-block: var(--space-xs) var(--space-sm); }
[data-rcl-page-header-copy] { min-inline-size: 0; }
[data-rcl-page-header] h1 { margin: 0; color: var(--color-foreground); font: var(--text-heading-lg); letter-spacing: var(--tracking-tight); }
[data-rcl-page-header] p { max-inline-size: 52ch; margin: var(--space-3xs) 0 0; color: var(--color-muted-foreground); font: var(--text-body-sm); }
[data-rcl-page-header-actions] { display: flex; flex-wrap: wrap; align-items: center; justify-content: end; gap: var(--space-2xs); }
@media (max-width: 48rem) { [data-rcl-page-header] { align-items: start; flex-direction: column; } [data-rcl-page-header-actions] { inline-size: 100%; justify-content: start; } }
`;

export const PageHeader = withClassName(function PageHeader({
  title,
  description,
  actions,
}: {
  title?: string;
  description?: string;
  actions?: ReactNode;
}) {
  const libraryStrings = useStrings();
  title = title ?? libraryStrings("navigation.page-header.page", "Page");
  return (
    <>
      <StyleSheet name="pageheader-1-0-7-1" css={pageHeaderStyles} />
      <header data-testid="navigation.page-header" data-rcl-page-header="">
        <div data-rcl-page-header-copy="">
          <h1>{title}</h1>
          {description ? <p>{description}</p> : null}
        </div>
        {actions ? <div data-rcl-page-header-actions="">{actions}</div> : null}
      </header>
    </>
  );
});
