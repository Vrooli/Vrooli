import { describe, expect, it } from "vitest";

import { renderWithProviders } from "../../../../../ui/src/test-utils";
import { FullPageDrawer, fullPageDrawerStyles } from "@vrooli/react-component-library/FullPageDrawer/1.4.3";
import { bottomSheetStyles } from "@vrooli/react-component-library/BottomSheet/1.3.3";

// WebKit paints the area outside the page with the <html> background with the
// <body> background blended over it (LocalFrameView::documentBackgroundColor).
// A host with an opaque body therefore hides any colour set on <html> alone —
// the 1.4.1/1.3.1 defect — so the open sheet must colour both.
describe("the strip beneath an open sheet takes the sheet's surface", () => {
  it("colours both <html> and <body> while a drawer or sheet is open on a phone", () => {
    for (const [styles, root] of [
      [fullPageDrawerStyles, "data-rcl-full-page-drawer"],
      [bottomSheetStyles, "data-rcl-bottom-sheet"],
    ] as const) {
      const scope = `html:has\\(\\[${root}\\]\\[data-state="open"\\]\\)`;
      expect(styles).toMatch(new RegExp(
        `@media not all and \\(min-width: 48rem\\)\\s*\\{\\s*${scope},\\s*${scope} body\\s*\\{\\s*background-color:\\s*var\\(--color-surface-raised\\)`,
      ));
    }
  });

  it("the rule's scope matches only while a sheet is open", () => {
    const scope = 'html:has([data-rcl-full-page-drawer][data-state="open"])';
    expect(document.documentElement.matches(scope)).toBe(false);
    const { unmount } = renderWithProviders(
      <FullPageDrawer open title="Compose" closeLabel="Close" testId="drawer">body</FullPageDrawer>,
    );
    expect(document.documentElement.matches(scope)).toBe(true);
    unmount();
    expect(document.documentElement.matches(scope)).toBe(false);
  });
});
