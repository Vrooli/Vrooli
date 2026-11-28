/**
 * Chromium's touch compatibility click inside an isolated HTML frame can
 * subtract the frame's offset from the viewport origin, so a tap lands up to
 * the frame's top offset away from the control the person touched. The only
 * reliable workaround is to keep that frame flush with the viewport origin,
 * which in the file viewer means the drawer's chrome moves below the preview.
 *
 * WebKit (every iOS browser, including Chrome and Firefox there) and Gecko do
 * not have that defect, and there the shared viewer's normal layout — location
 * row and tool row above the preview — is both correct and safer. So this
 * flag is set only where the defect exists: a Blink engine (the only one that
 * exposes `navigator.userAgentData`) on a coarse pointer.
 *
 * This is engine capability, not device identity: `data-wc-touch-html-frame-origin`
 * may only gate the drawer geometry in `styles.css`. Nothing else should read it.
 */
export const TOUCH_HTML_FRAME_ORIGIN_ATTRIBUTE = "wcTouchHtmlFrameOrigin";

export interface TouchHtmlFrameOriginEnvironment {
  /** Blink exposes `navigator.userAgentData`; WebKit and Gecko do not. */
  chromium?: boolean;
  /** `window.matchMedia`, injectable for tests. */
  matchMedia?: (query: string) => MediaQueryList;
  /** Defaults to the live document element. */
  root?: HTMLElement;
}

export function isBlinkEnvironment(nav: Navigator | undefined = typeof navigator === "undefined" ? undefined : navigator): boolean {
  return Boolean(nav && "userAgentData" in nav);
}

/**
 * Adds (or removes) the attribute that pins the touch HTML preview to the
 * viewport origin. Returns a disposer so tests and any future unmount path can
 * take the flag away again.
 */
export function installTouchHtmlFrameOrigin(environment: TouchHtmlFrameOriginEnvironment = {}): () => void {
  const root = environment.root ?? (typeof document === "undefined" ? undefined : document.documentElement);
  const chromium = environment.chromium ?? isBlinkEnvironment();
  if (!root || !chromium) return () => {};
  const matchMedia = environment.matchMedia ?? (typeof window === "undefined" ? undefined : window.matchMedia.bind(window));
  if (!matchMedia) return () => {};

  const query = matchMedia("(pointer: coarse)");
  const apply = () => {
    if (query.matches) root.dataset[TOUCH_HTML_FRAME_ORIGIN_ATTRIBUTE] = "true";
    else delete root.dataset[TOUCH_HTML_FRAME_ORIGIN_ATTRIBUTE];
  };
  apply();
  query.addEventListener("change", apply);
  return () => {
    query.removeEventListener("change", apply);
    delete root.dataset[TOUCH_HTML_FRAME_ORIGIN_ATTRIBUTE];
  };
}
