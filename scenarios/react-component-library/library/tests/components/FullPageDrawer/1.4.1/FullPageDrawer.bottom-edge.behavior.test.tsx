import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";

import { renderWithProviders } from "../../../../../ui/src/test-utils";
import { FullPageDrawer } from "@vrooli/react-component-library/FullPageDrawer/1.4.1";
import { BottomSheet } from "@vrooli/react-component-library/BottomSheet/1.3.1";
import {
  ViewportEnvironmentProvider,
  type ViewportEnvironmentSnapshot,
} from "@vrooli/react-component-library/useViewportEnvironment/1.1.0";

const installedShortViewport: ViewportEnvironmentSnapshot = {
  layoutWidth: 393, layoutHeight: 793, visibleWidth: 393, visibleHeight: 793,
  offsetLeft: 0, offsetTop: 0, scale: 1, keyboardInset: 0, keyboardVisible: false,
  reachesScreenBottom: false,
};

const injectedCss = () => [...document.head.querySelectorAll("style")].map((node) => node.textContent ?? "").join("\n");

describe("bottom-anchored overlays and the screen's bottom edge", () => {
  it("a drawer reserves no home-indicator inset when the viewport says the app does not reach the bottom", () => {
    renderWithProviders(
      <ViewportEnvironmentProvider value={installedShortViewport}>
        <FullPageDrawer open title="Compose" closeLabel="Close" testId="drawer">body</FullPageDrawer>
      </ViewportEnvironmentProvider>,
    );
    const root = screen.getByTestId("drawer").closest<HTMLElement>("[data-rcl-full-page-drawer]");
    expect(root?.style.getPropertyValue("--rcl-safe-bottom")).toBe("0px");
  });

  it("the drawer and the sheet colour the page canvas beneath them while open on a phone", () => {
    renderWithProviders(
      <>
        <FullPageDrawer open title="Compose" closeLabel="Close" testId="drawer">body</FullPageDrawer>
        <BottomSheet open title="Pick" closeLabel="Close" testId="sheet">body</BottomSheet>
      </>,
    );
    const css = injectedCss();
    expect(css).toMatch(/@media not all and \(min-width: 48rem\)\s*\{\s*html:has\(\[data-rcl-full-page-drawer\]\[data-state="open"\]\)\s*\{\s*background-color:\s*var\(--color-surface-raised\)/);
    expect(css).toMatch(/@media not all and \(min-width: 48rem\)\s*\{\s*html:has\(\[data-rcl-bottom-sheet\]\[data-state="open"\]\)\s*\{\s*background-color:\s*var\(--color-surface-raised\)/);
  });
});
