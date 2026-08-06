/**
 * @libraryId react-component-library:PageFrame
 * @version 1.0.0
 * @status released
 * @deps {"react":"^18"}
 */
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
export function PageFrame({ regions = {}, fixture, children, title = "Preview page" }: PageFrameProps) {
  const navigation = regions.navigation;
  const content = regions.content ?? children;
  return <div data-testid="page-frame" data-frame-asset="navigation.page" className="min-h-full bg-app-background text-app-foreground">
    <a href="#page-frame-content" className="sr-only focus:not-sr-only">Skip to content</a>
    <div data-frame-region="navigation" className="min-h-full">{navigation}</div>
    <main id="page-frame-content" data-frame-region="content" aria-label={title} className="min-w-0 p-4">
      {content}
      {fixture?.asset ? <output data-frame-fixture={fixture.asset} className="sr-only">{fixture.asset}</output> : null}
    </main>
  </div>;
}
