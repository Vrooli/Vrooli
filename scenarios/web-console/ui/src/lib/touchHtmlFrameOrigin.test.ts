import { afterEach, describe, expect, it, vi } from "vitest";
import { installTouchHtmlFrameOrigin, isBlinkEnvironment, TOUCH_HTML_FRAME_ORIGIN_ATTRIBUTE } from "./touchHtmlFrameOrigin";

type MockMediaQueryList = MediaQueryList & { setMatches: (matches: boolean) => void };

function installMatchMedia(initialMatches: boolean) {
  const listeners = new Set<(event: MediaQueryListEvent) => void>();
  const query = {
    media: "(pointer: coarse)",
    matches: initialMatches,
    onchange: null,
    addEventListener: vi.fn((_type: string, listener: (event: MediaQueryListEvent) => void) => { listeners.add(listener); }),
    removeEventListener: vi.fn((_type: string, listener: (event: MediaQueryListEvent) => void) => { listeners.delete(listener); }),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
    setMatches(matches: boolean) {
      (query as unknown as { matches: boolean }).matches = matches;
      const event = { matches, media: query.media } as MediaQueryListEvent;
      for (const listener of listeners) listener(event);
    },
  } as unknown as MockMediaQueryList;
  return { query, matchMedia: vi.fn(() => query) };
}

function root() {
  const element = document.createElement("div");
  document.body.append(element);
  return element;
}

afterEach(() => {
  vi.unstubAllGlobals();
  document.body.replaceChildren();
});

describe("touchHtmlFrameOrigin", () => {
  it("flags a coarse Blink device so the HTML preview stays at the viewport origin", () => {
    const { matchMedia } = installMatchMedia(true);
    const element = root();

    installTouchHtmlFrameOrigin({ chromium: true, matchMedia, root: element });

    expect(element.dataset[TOUCH_HTML_FRAME_ORIGIN_ATTRIBUTE]).toBe("true");
  });

  it("leaves WebKit and Gecko on the viewer's normal chrome-on-top layout", () => {
    const { matchMedia } = installMatchMedia(true);
    const element = root();

    installTouchHtmlFrameOrigin({ chromium: false, matchMedia, root: element });

    expect(element.dataset[TOUCH_HTML_FRAME_ORIGIN_ATTRIBUTE]).toBeUndefined();
    expect(matchMedia).not.toHaveBeenCalled();
  });

  it("does nothing for a fine pointer on Blink, and follows later pointer changes", () => {
    const { matchMedia, query } = installMatchMedia(false);
    const element = root();
    const dispose = installTouchHtmlFrameOrigin({ chromium: true, matchMedia, root: element });

    expect(element.dataset[TOUCH_HTML_FRAME_ORIGIN_ATTRIBUTE]).toBeUndefined();

    query.setMatches(true);
    expect(element.dataset[TOUCH_HTML_FRAME_ORIGIN_ATTRIBUTE]).toBe("true");

    query.setMatches(false);
    expect(element.dataset[TOUCH_HTML_FRAME_ORIGIN_ATTRIBUTE]).toBeUndefined();

    dispose();
    expect(query.removeEventListener).toHaveBeenCalled();
  });

  it("removes the flag on dispose", () => {
    const { matchMedia } = installMatchMedia(true);
    const element = root();
    const dispose = installTouchHtmlFrameOrigin({ chromium: true, matchMedia, root: element });

    dispose();

    expect(element.dataset[TOUCH_HTML_FRAME_ORIGIN_ATTRIBUTE]).toBeUndefined();
  });

  it("recognizes Blink by the userAgentData capability, not a UA string", () => {
    vi.stubGlobal("navigator", { userAgentData: {} });
    expect(isBlinkEnvironment()).toBe(true);

    vi.stubGlobal("navigator", { userAgent: "Mozilla/5.0 (iPhone) Version/17.0 Mobile Safari/604.1" });
    expect(isBlinkEnvironment()).toBe(false);
  });
});
