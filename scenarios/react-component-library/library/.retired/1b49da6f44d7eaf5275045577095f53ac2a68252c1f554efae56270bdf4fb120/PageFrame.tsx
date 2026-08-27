/**
 * @libraryId react-component-library:PageFrame
 * @displayName Page Frame
 * @description A route-content boundary that gives navigation and page specimens a real responsive document context.
 * @version 1.0.1
 * @tags ["layout","navigation","frame"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import type { ReactNode } from "react";

export interface PageFrameProps {
  regions?: Record<string, ReactNode>;
  fixture?: { asset?: string; dataShapes?: string[] };
  children?: ReactNode;
  title?: string;
}

// PageFrame is intentionally small: the frame owns document landmarks and
// region placement, while the framed asset owns its own behavior and props.
// This gives preview a real composition boundary without adding a second
// slot vocabulary to the catalog.
export const PageFrame = withClassName(function PageFrame({
  regions = {},
  fixture,
  children,
  title = "Preview page",
}: PageFrameProps) {
  const navigation = regions.navigation;
  const content = regions.content ?? children;
  return (
    <>
      <style
        data-rcl-page-frame-styles
        dangerouslySetInnerHTML={{
          __html: `
        [data-rcl-page-frame] { min-height: 100vh; display: grid; grid-template-columns: minmax(14rem, 18rem) minmax(0, 1fr); background: var(--color-background); color: var(--color-foreground); }
        [data-rcl-page-frame-nav] { min-width: 0; padding: var(--space-md); border-inline-end: var(--border-hairline) solid var(--color-border); background: var(--color-surface); }
        [data-rcl-page-frame-content] { min-width: 0; padding: clamp(var(--space-md), 4vw, var(--space-2xl)); }
        .rcl-page-frame-skip { position: fixed; inset-block-start: var(--space-sm); inset-inline-start: var(--space-sm); z-index: 2; transform: translateY(-180%); padding: var(--space-xs) var(--space-sm); border-radius: var(--radius-control); background: var(--color-primary); color: var(--color-primary-foreground); font-weight: 700; text-decoration: none; transition: transform 160ms ease; }
        .rcl-page-frame-skip:focus-visible { transform: translateY(0); outline: var(--focus-ring-width) solid var(--focus-ring-color); outline-offset: var(--focus-ring-offset); }
        @media (prefers-reduced-motion: reduce) { .rcl-page-frame-skip { transition: none; } }
        @media (max-width: 720px) { [data-rcl-page-frame] { grid-template-columns: 1fr; } [data-rcl-page-frame-nav] { border-inline-end: 0; border-block-end: var(--border-hairline) solid var(--color-border); padding: var(--space-sm); } [data-rcl-page-frame-content] { padding: var(--space-sm); } }
      `,
        }}
      />
      <div data-testid="page-frame" data-frame-asset="navigation.page" data-rcl-page-frame>
        <a href="#page-frame-content" className="rcl-page-frame-skip">
          Skip to content
        </a>
        <div data-frame-region="navigation" data-rcl-page-frame-nav>
          {navigation}
        </div>
        <main
          id="page-frame-content"
          data-frame-region="content"
          aria-label={title}
          data-rcl-page-frame-content
        >
          {content}
          {fixture?.asset ? (
            <output data-frame-fixture={fixture.asset} className="sr-only">
              {fixture.asset}
            </output>
          ) : null}
        </main>
      </div>
    </>
  );
});
