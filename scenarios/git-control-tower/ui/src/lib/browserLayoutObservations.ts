import { browserObservationClient } from "./connect";

const standalone = () => (navigator as Navigator & { standalone?: boolean }).standalone === true || window.matchMedia("(display-mode: standalone)").matches;
const rect = (element: Element | null) => {
  if (!element) return null;
  const { x, y, width, height, top, right, bottom, left } = element.getBoundingClientRect();
  const style = getComputedStyle(element);
  return { x, y, width, height, top, right, bottom, left, background: style.backgroundColor,
    paddingTop: style.paddingTop, paddingBottom: style.paddingBottom, position: style.position };
};

/** Read actual client geometry. Contains no repository content or native pixels. */
export function readBrowserLayout(probe: HTMLElement) {
  const visual = window.visualViewport;
  const style = getComputedStyle(probe);
  const overlays = Array.from(document.querySelectorAll('[data-rcl-full-page-drawer][data-state="open"], [data-rcl-bottom-sheet][data-state="open"], [role="dialog"][aria-modal="true"]'));
  const nav = document.querySelector('[data-testid="mobile-nav"]');
  return {
    expectations: { fullScreenCanvas: true, headerMaxHeight: 48 },
    keyboardVisible: document.documentElement.dataset.gctKeyboardVisible === "true",
    viewport: { width: innerWidth, height: innerHeight },
    screen: { width: screen.width, height: screen.height, availableWidth: screen.availWidth, availableHeight: screen.availHeight },
    outer: { width: outerWidth, height: outerHeight },
    visual: visual ? { width: visual.width, height: visual.height, offsetTop: visual.offsetTop, offsetLeft: visual.offsetLeft, scale: visual.scale } : null,
    dpr: devicePixelRatio,
    document: { clientHeight: document.documentElement.clientHeight, scrollHeight: document.documentElement.scrollHeight },
    scroll: { x: scrollX, y: scrollY },
    safeArea: { top: parseFloat(style.paddingTop), right: parseFloat(style.paddingRight), bottom: parseFloat(style.paddingBottom), left: parseFloat(style.paddingLeft), effectiveBottom: parseFloat(getComputedStyle(probe.children[2] as Element).height), maxBottom: parseFloat(getComputedStyle(probe.children[3] as Element).paddingBottom) },
    cssViewport: { dynamicHeight: probe.children[0]?.getBoundingClientRect().height, largeHeight: probe.children[1]?.getBoundingClientRect().height },
    fixedFrame: rect(probe),
    html: rect(document.documentElement), body: rect(document.body), shell: rect(document.querySelector('.gct-mobile-shell')),
    header: rect(document.querySelector('[data-testid="mobile-header"]')),
    statusFill: rect(document.querySelector('[data-testid="status-bar-fill-strip"]')),
    nav: rect(nav), track: rect(nav?.querySelector('[data-rcl-bottom-nav-track]') ?? null),
    controls: Array.from(nav?.querySelectorAll('[data-rcl-bottom-nav-item]') ?? []).map(element => { const box = element.getBoundingClientRect(); const hit = document.elementFromPoint(box.x + box.width / 2, box.y + box.height / 2); return { ...rect(element), centerHit: Boolean(hit && element.contains(hit)), children: Array.from(element.querySelectorAll('[data-rcl-bottom-nav-icon],[data-rcl-bottom-nav-label]')).map(rect) }; }),
    overlays: overlays.map(element => ({ id: element.getAttribute('data-testid'), frame: rect(element), panel: rect(element.querySelector('.rcl-full-page-drawer__panel, .rcl-bottom-sheet__panel, .rcl-responsive-dialog__panel') ?? element) })),
    bottomElement: (() => { const element = document.elementFromPoint(innerWidth / 2, Math.max(0, (visual?.height ?? innerHeight) - 1)); return { tag: element?.tagName, testId: element?.getAttribute('data-testid'), background: element ? getComputedStyle(element).backgroundColor : null }; })(),
    meta: { viewport: document.querySelector('meta[name="viewport"]')?.getAttribute('content'), statusStyle: document.querySelector('meta[name="apple-mobile-web-app-status-bar-style"]')?.getAttribute('content'), themeColor: document.querySelector('meta[name="theme-color"]')?.getAttribute('content') },
  };
}

/** Automatically retain changed mobile layout states, at most once per second. */
export function startBrowserLayoutObservations(): () => void {
  const probe = document.createElement('div');
  probe.setAttribute('aria-hidden', 'true');
  probe.style.cssText = 'position:fixed;inset:0;visibility:hidden;pointer-events:none;box-sizing:border-box;padding:env(safe-area-inset-top,0px) env(safe-area-inset-right,0px) env(safe-area-inset-bottom,0px) env(safe-area-inset-left,0px);';
  for (const unit of ['100dvh', '100lvh', 'var(--rcl-safe-bottom,env(safe-area-inset-bottom,0px))', '0px']) { const child = document.createElement('span'); child.style.cssText = 'position:absolute;width:0;padding-bottom:env(safe-area-max-inset-bottom,0px);height:' + unit; probe.append(child); }
  document.body.append(probe);
  const clientId = typeof crypto.randomUUID === 'function' ? crypto.randomUUID() : Math.random().toString(36).slice(2);
  // Production entry chunk identifies what THIS client loaded, including a stale tab.
  const buildId = Array.from(document.scripts).map(script => script.getAttribute('src')).find(src => src?.includes('/assets/')) ?? import.meta.url;
  let timer: ReturnType<typeof setTimeout> | undefined;
  let last = '';
  let disposed = false;
  let failures = 0;
  let sending = false;
  const report = async () => {
    timer = undefined;
    if (disposed || sending || document.visibilityState === 'hidden' || innerWidth > 767) return;
    sending = true;
    try {
      const layoutJson = JSON.stringify(readBrowserLayout(probe));
      if (layoutJson === last) return;
      await browserObservationClient.recordBrowserObservation({ observation: { scenario: 'git-control-tower', clientId, buildId: buildId.slice(-200), userAgent: navigator.userAgent.slice(0,1000), standalone: standalone(), embedded: window.top !== window.self, layoutJson } }, { timeoutMs: 6000 });
      if (!disposed) { last = layoutJson; failures = 0; }
    } catch {
      // Evidence transport must never interrupt repository work. A later layout
      // transition retries; no claim of successful capture is made locally.
      failures += 1;
      if (!disposed && failures < 3 && !timer) timer = setTimeout(() => { void report(); }, 10000);
    } finally { sending = false; }
  };
  const schedule = () => { if (!timer && !disposed) timer = setTimeout(() => { void report(); }, 1000); };
  const observer = new MutationObserver(schedule);
  observer.observe(document.body, { childList: true, subtree: true, attributes: true, attributeFilter: ['data-state', 'data-expanded', 'aria-modal'] });
  window.addEventListener('resize', schedule);
  window.addEventListener('orientationchange', schedule);
  window.visualViewport?.addEventListener('resize', schedule);
  window.visualViewport?.addEventListener('scroll', schedule);
  document.addEventListener('visibilitychange', schedule);
  schedule();
  return () => { disposed = true; observer.disconnect(); if (timer) clearTimeout(timer); probe.remove(); window.removeEventListener('resize', schedule); window.removeEventListener('orientationchange', schedule); window.visualViewport?.removeEventListener('resize', schedule); window.visualViewport?.removeEventListener('scroll', schedule); document.removeEventListener('visibilitychange', schedule); };
}
