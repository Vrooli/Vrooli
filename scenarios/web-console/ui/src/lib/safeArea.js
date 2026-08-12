let cached = null;
function parsePx(value) {
    const parsed = Number.parseFloat(value);
    return Number.isFinite(parsed) ? parsed : 0;
}
function cacheKey() {
    const vv = window.visualViewport;
    return [
        window.innerWidth,
        window.innerHeight,
        vv?.width ?? 0,
        vv?.height ?? 0,
        vv?.offsetTop ?? 0,
        vv?.offsetLeft ?? 0,
    ].join(":");
}
export function readSafeAreaInsets() {
    if (typeof window === "undefined" || typeof document === "undefined" || !document.body) {
        return { top: 0, right: 0, bottom: 0, left: 0 };
    }
    const key = cacheKey();
    if (cached?.key === key)
        return cached.insets;
    const el = document.createElement("div");
    el.style.position = "fixed";
    el.style.visibility = "hidden";
    el.style.pointerEvents = "none";
    el.style.paddingTop = "env(safe-area-inset-top)";
    el.style.paddingRight = "env(safe-area-inset-right)";
    el.style.paddingBottom = "env(safe-area-inset-bottom)";
    el.style.paddingLeft = "env(safe-area-inset-left)";
    document.body.appendChild(el);
    const styles = window.getComputedStyle(el);
    const insets = {
        top: parsePx(styles.paddingTop),
        right: parsePx(styles.paddingRight),
        bottom: parsePx(styles.paddingBottom),
        left: parsePx(styles.paddingLeft),
    };
    el.remove();
    cached = { key, insets };
    return insets;
}
