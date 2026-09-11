import type { ReactNode } from "react";
import { render } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

const raw = vi.hoisted(() => ({
  current: {
    layoutWidth: 393, layoutHeight: 793, visibleWidth: 393, visibleHeight: 793,
    offsetLeft: 0, offsetTop: 0, scale: 1, keyboardInset: 0, keyboardVisible: false,
  },
}));
const screenFacts = vi.hoisted(() => ({
  current: { standalone: true, screenWidth: 393, screenHeight: 852, safeTop: 59 },
}));

vi.mock("@vrooli/react-component-library/useViewportEnvironment/1", () => ({
  useViewportEnvironment: () => raw.current,
  ViewportEnvironmentProvider: ({ children }: { children: ReactNode }) => children,
}));
vi.mock("../lib/viewportCorrection", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../lib/viewportCorrection")>()),
  readScreenFacts: () => screenFacts.current,
}));

import { AppViewportProvider, VIEWPORT_EXTENDED_ATTRIBUTE } from "./AppViewportProvider";

const marked = () => document.documentElement.hasAttribute(VIEWPORT_EXTENDED_ATTRIBUTE);

describe("AppViewportProvider", () => {
  afterEach(() => {
    document.documentElement.removeAttribute(VIEWPORT_EXTENDED_ATTRIBUTE);
    raw.current = { ...raw.current, visibleHeight: 793, keyboardVisible: false, keyboardInset: 0 };
    screenFacts.current = { ...screenFacts.current, standalone: true };
  });

  // iOS paints no fixed-position content below the short layout viewport, so
  // the page must say when it has extended itself past it.
  it("marks the document while the app is extended past the reported viewport, and clears the mark when it is not", () => {
    const { rerender, unmount } = render(<AppViewportProvider><span /></AppViewportProvider>);
    expect(marked()).toBe(true);

    raw.current = { ...raw.current, visibleHeight: 450, keyboardInset: 343, keyboardVisible: true };
    rerender(<AppViewportProvider><span /></AppViewportProvider>);
    expect(marked()).toBe(false);

    raw.current = { ...raw.current, visibleHeight: 793, keyboardInset: 0, keyboardVisible: false };
    rerender(<AppViewportProvider><span /></AppViewportProvider>);
    expect(marked()).toBe(true);
    unmount();
    expect(marked()).toBe(false);
  });

  it("never marks a browser tab", () => {
    screenFacts.current = { ...screenFacts.current, standalone: false };
    render(<AppViewportProvider><span /></AppViewportProvider>);
    expect(marked()).toBe(false);
  });
});
