/**
 * @libraryId react-component-library:PageHeader
 * @displayName PageHeader
 * @description The unframed title region every routed page opens with: optional eyebrow, leading element, title at level one or two, description, and right-aligned actions that wrap under the copy on a phone. Compact density for headers inside panes.
 * @version 2.0.1
 * @tags []
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:PageHeader */
import type { ReactNode } from "react";

import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";

/**
 * 2.0.0 — one page header for every routed page.
 *
 * 1.x rendered a fixed `h1` with a title and an optional description and
 * nothing else, so scenarios that needed an eyebrow, a leading element or a
 * heading level below `h1` wrote their own. This version takes all of those
 * and stays unframed: the header is a row of type at the top of a page, not a
 * card, so pages keep the shell's ground behind them.
 */
export interface PageHeaderProps {
  title: ReactNode;
  /** Short line above the title: a section name, a breadcrumb, a back link. */
  eyebrow?: ReactNode;
  description?: ReactNode;
  /** Sits before the copy: an avatar, a mark, a status glyph. */
  leading?: ReactNode;
  /** Right-aligned controls; wraps under the copy on a phone. */
  actions?: ReactNode;
  /** Heading level. A page's own header is `1`; a section inside a page is `2`. */
  level?: 1 | 2;
  /** Compact removes the bottom rule and tightens the vertical rhythm for headers inside panes. */
  density?: "comfortable" | "compact";
  /** Test id for the header landmark. */
  testId?: string;
  /** Id placed on the heading so a page region can point `aria-labelledby` at it. */
  headingId?: string;
  className?: string;
}

export const pageHeaderStyles = `
[data-rcl-page-header] { display: flex; align-items: flex-start; justify-content: space-between; gap: var(--space-sm); min-inline-size: 0; padding-block: var(--space-2xs) var(--space-sm); border-block-end: var(--border-hairline) solid var(--color-border); }
[data-rcl-page-header][data-density="compact"] { padding-block: 0 var(--space-2xs); border-block-end: 0; }
[data-rcl-page-header-lead] { display: flex; min-inline-size: 0; flex: 1; gap: var(--space-xs); align-items: flex-start; }
[data-rcl-page-header-leading] { display: grid; flex: none; place-items: center; }
[data-rcl-page-header-copy] { display: grid; min-inline-size: 0; gap: var(--space-3xs); }
[data-rcl-page-header-eyebrow] { color: var(--color-muted-foreground); font: var(--text-caption); letter-spacing: 0.06em; text-transform: uppercase; }
[data-rcl-page-header-eyebrow] a { color: var(--color-primary); text-decoration: none; }
[data-rcl-page-header-title] { margin: 0; color: var(--color-foreground); font: var(--text-title); letter-spacing: -0.015em; overflow-wrap: anywhere; text-wrap: balance; }
[data-rcl-page-header][data-level="2"] [data-rcl-page-header-title] { font: var(--text-heading); letter-spacing: -0.01em; }
[data-rcl-page-header-description] { max-inline-size: 64ch; margin: 0; color: var(--color-muted-foreground); font: var(--text-body-sm); }
[data-rcl-page-header-actions] { display: flex; flex: none; flex-wrap: wrap; align-items: center; justify-content: flex-end; gap: var(--space-2xs); }
@media (max-width: 47.999rem) { [data-rcl-page-header] { flex-direction: column; } [data-rcl-page-header-actions] { inline-size: 100%; justify-content: flex-start; } }
@media (forced-colors: active) { [data-rcl-page-header] { border-block-end-color: CanvasText; } }
`;

export const PageHeader = withClassName(function PageHeader({
  title,
  eyebrow,
  description,
  leading,
  actions,
  level = 1,
  density = "comfortable",
  testId = "navigation.page-header",
  headingId,
  className,
}: PageHeaderProps) {
  const Heading = level === 1 ? "h1" : "h2";
  return (
    <>
      <StyleSheet name="page-header-2-0-1" css={pageHeaderStyles} />
      <header
        data-rcl-page-header=""
        data-level={level}
        data-density={density}
        data-testid={testId}
        className={className}
      >
        <div data-rcl-page-header-lead="">
          {leading ? <div data-rcl-page-header-leading="">{leading}</div> : null}
          <div data-rcl-page-header-copy="">
            {eyebrow ? <div data-rcl-page-header-eyebrow="">{eyebrow}</div> : null}
            <Heading data-rcl-page-header-title="" id={headingId}>
              {title}
            </Heading>
            {description ? <p data-rcl-page-header-description="">{description}</p> : null}
          </div>
        </div>
        {actions ? <div data-rcl-page-header-actions="">{actions}</div> : null}
      </header>
    </>
  );
});
