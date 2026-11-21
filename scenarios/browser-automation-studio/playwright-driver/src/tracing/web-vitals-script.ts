/**
 * Web-Vitals Observer Init Script
 *
 * A self-contained PerformanceObserver bundle injected into every document
 * (via context.addInitScript) BEFORE the page's own scripts run, so it
 * captures paint/LCP/CLS/long-task/navigation metrics from the very first
 * frame. It writes into a single global object that the driver reads back
 * at trace stop.
 *
 * DESIGN — why inline, not the `web-vitals` npm package:
 * - Zero new dependency in the driver bundle (the package would add a
 *   runtime dep that must clear SDA governance for a ~2KB observer we can
 *   hand-write).
 * - The page under test is arbitrary and uncooperative; we cannot import a
 *   module into it. addInitScript injects raw source, so a string is the
 *   only shape that works for ANY url (Tier 0).
 * - We need raw entries (not the package's "report once" semantics) so the
 *   performance-health analyzer can compute its own deltas.
 *
 * Tier 1 (⚛ React component marks) rides along the CDP trace's
 * blink.user_timing category untouched — this observer does not need to
 * handle it; it only adds the page-cooperation-free Tier 0 vitals.
 *
 * The global is `window.__vrooliWebVitals`. It is intentionally simple,
 * JSON-serializable, and append-only.
 */

/** Key the observer writes the collected metrics under, on `window`. */
export const WEB_VITALS_GLOBAL = '__vrooliWebVitals';

/**
 * The init-script source. Kept as a string so it can be handed to
 * `context.addInitScript({ content })` verbatim. It must be standalone
 * (no closures over driver state) and defensive (a missing
 * PerformanceObserver entry type must never throw).
 */
export const WEB_VITALS_INIT_SCRIPT = `(() => {
  if (window.${WEB_VITALS_GLOBAL}) { return; }
  var store = {
    startedAt: Date.now(),
    paint: [],
    lcp: null,
    cls: { value: 0, entries: 0 },
    longTasks: [],
    fcp: null,
    navigation: null,
  };
  window.${WEB_VITALS_GLOBAL} = store;

  function safeObserve(type, cb, buffered) {
    try {
      var po = new PerformanceObserver(function (list) {
        var entries = list.getEntries();
        for (var i = 0; i < entries.length; i++) { cb(entries[i]); }
      });
      po.observe({ type: type, buffered: buffered !== false });
    } catch (e) {
      // Entry type unsupported in this browser/page — Tier 0 degrades to
      // whatever metrics ARE available; never throw.
    }
  }

  safeObserve('paint', function (e) {
    store.paint.push({ name: e.name, startTime: e.startTime });
    if (e.name === 'first-contentful-paint') {
      store.fcp = e.startTime;
    }
  });

  safeObserve('largest-contentful-paint', function (e) {
    store.lcp = { value: e.startTime, size: e.size, renderTime: e.renderTime, loadTime: e.loadTime };
  });

  safeObserve('layout-shift', function (e) {
    if (!e.hadRecentInput) {
      store.cls.value += e.value;
      store.cls.entries += 1;
    }
  });

  safeObserve('longtask', function (e) {
    store.longTasks.push({ startTime: e.startTime, duration: e.duration, name: e.name });
  });

  safeObserve('navigation', function (e) {
    store.navigation = {
      domContentLoaded: e.domContentLoadedEventEnd,
      loadEventEnd: e.loadEventEnd,
      responseEnd: e.responseEnd,
      domInteractive: e.domInteractive,
      type: e.type,
    };
  });
})();`;
