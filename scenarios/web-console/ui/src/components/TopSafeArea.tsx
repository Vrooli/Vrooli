import type { ReactNode, Ref } from "react";
import { StatusBarFill } from "@vrooli/react-component-library/ChromeTheme";

import { cn } from "../lib/classnames";

interface TopSafeAreaProps {
  children: ReactNode;
  containerRef?: Ref<HTMLDivElement>;
  className?: string;
  fillClassName?: string;
  testId?: string;
  enabled?: boolean;
}

/**
 * Owns the iOS/PWA top safe area without placing content inside it.
 *
 * The first child surface starts below the status-bar/notch region, while the
 * fill strip gives nav modes full control over the status area's color.
 *
 * The strip itself is the library's `StatusBarFill`, and its colour comes from
 * the `ChromeTheme` service rather than from a class threaded through props:
 * the resting terminal-derived tint is that service's base, and an active
 * banner contributes over it. That direction is deliberate — the colour arrives
 * from the one place that knows what is actually on screen, instead of being
 * recomputed by every caller from conditions that may already have been
 * dismissed.
 *
 * The container is also the page's **top edge bar** for iOS 26. WebKit
 * hit-tests the middle of the top edge, walks up to the nearest fixed or
 * sticky box, and paints that box's single opaque colour over the status bar
 * only when the box spans the width but not the height of the viewport;
 * otherwise it draws its blurred scroll-edge effect over the top of the page
 * (`LocalFrameView::fixedContainerEdges`). The fixed full-screen `body` fails
 * that test, so this box is sticky: in a non-scrolling column it lays out
 * exactly as before, and it gives WebKit a qualifying bar whose colour is the
 * status strip's. Keep it free of backdrop filters and of a second background
 * colour on the path from the strip up — either one restores the blur.
 * See docs/guides/ADAPTIVE_CHROME.md.
 */
export default function TopSafeArea({
  children,
  containerRef,
  className,
  fillClassName,
  testId,
  enabled = true,
}: TopSafeAreaProps) {
  return (
    <div
      ref={containerRef}
      data-testid={testId}
      data-wc-top-edge-bar=""
      className={cn("sticky top-0 z-[var(--layer-sticky)] flex shrink-0 flex-col", className)}
    >
      {enabled && (
        <StatusBarFill
          testId={testId ? `${testId}-fill` : undefined}
          className={fillClassName}
        />
      )}
      {children}
    </div>
  );
}
