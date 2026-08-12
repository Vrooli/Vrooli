/**
 * Pure, framework-free grid layout utilities for the workspace.
 *
 * Adapted from scenarios/app-monitor/ui/src/features/preview-workspace/utils/layout.ts
 * with simplifications: no className, no resolveDropIndex, no minimap, no pinned-column logic.
 */
const clampPaneCount = (count) => {
    if (!Number.isFinite(count) || count < 1)
        return 1;
    return Math.floor(count);
};
const clamp = (value, min, max) => {
    if (value < min)
        return min;
    if (value > max)
        return max;
    return value;
};
/**
 * Determines grid dimensions (columns × rows) for a given pane count.
 * Max 2 columns by default; single pane always uses 1×1.
 */
export const resolveWorkspaceLayout = (paneCount, maxColumns = 2) => {
    const count = clampPaneCount(paneCount);
    const safeCols = Math.max(1, Math.floor(maxColumns));
    if (count <= 1 || safeCols <= 1) {
        return { columns: 1, rows: count };
    }
    const columns = Math.min(2, safeCols);
    return { columns, rows: Math.ceil(count / columns) };
};
const normalizeFractions = (fractions) => {
    const safe = fractions.map((v) => (Number.isFinite(v) && v > 0 ? v : 0));
    const sum = safe.reduce((a, b) => a + b, 0);
    if (sum <= 0)
        return [];
    return safe.map((v) => v / sum);
};
/**
 * Preserves existing proportions when pane count changes.
 * Adds equal-sized new tracks; trims if count decreased.
 */
export const reconcileTrackFractions = (current, nextCount) => {
    if (!Number.isFinite(nextCount) || nextCount <= 0)
        return [1];
    const count = Math.floor(nextCount);
    if (count === 1)
        return [1];
    const normalized = normalizeFractions(current);
    if (normalized.length === count)
        return normalized;
    if (normalized.length === 0) {
        return Array.from({ length: count }, () => 1 / count);
    }
    if (normalized.length > count) {
        return normalizeFractions(normalized.slice(0, count));
    }
    const next = [...normalized];
    while (next.length < count) {
        const last = next[next.length - 1] ?? 1 / count;
        next.push(last);
    }
    return normalizeFractions(next);
};
/**
 * Compares two fraction arrays with floating-point tolerance.
 *
 * normalizeFractions divides each value by the sum.  For certain track counts
 * (6, 7, 9, …) the sum is not exactly 1.0 in IEEE 754, so re-normalizing
 * already-normalized values produces a slightly different array.  Strict
 * equality would see a difference on every pass, causing an infinite
 * reconciliation loop in the Workspace effect that persists fractions to the
 * store (React error #185 — maximum update depth exceeded).
 *
 * An epsilon of 1e-12 absorbs the ~1 ULP drift while remaining far below any
 * user-visible precision.
 */
const FRAC_EPSILON = 1e-12;
export const fractionsMatch = (a, b) => {
    if (a.length !== b.length)
        return false;
    for (let i = 0; i < a.length; i++) {
        if (Math.abs((a[i] ?? 0) - (b[i] ?? 0)) > FRAC_EPSILON)
            return false;
    }
    return true;
};
/**
 * Generates a CSS grid template string like `"minmax(0, 0.5fr) 8px minmax(0, 0.5fr)"`.
 */
export const buildGridTrackTemplate = (fractions, splitterSize) => {
    if (fractions.length <= 1)
        return "minmax(0, 1fr)";
    const safeSplitter = Number.isFinite(splitterSize) && splitterSize > 0 ? splitterSize : 6;
    const segments = [];
    fractions.forEach((fraction, index) => {
        const f = Number.isFinite(fraction) && fraction > 0 ? fraction : 1;
        segments.push(`minmax(0, ${f}fr)`);
        if (index < fractions.length - 1) {
            segments.push(`${safeSplitter}px`);
        }
    });
    return segments.join(" ");
};
/**
 * Resize algorithm that only affects the two tracks adjacent to a splitter.
 * Returns a new fractions array with only positions `index` and `index+1` changed.
 */
export const updateAdjacentFractions = ({ startValues, index, delta, containerSize, splitterCount, minTrackPx, splitterSize, }) => {
    if (index < 0 || index >= startValues.length - 1)
        return startValues;
    const safeSplitter = Number.isFinite(splitterSize) && splitterSize > 0 ? splitterSize : 8;
    const usableSize = Math.max(1, containerSize - splitterCount * safeSplitter);
    const current = startValues[index] ?? 0;
    const next = startValues[index + 1] ?? 0;
    const pairFraction = current + next;
    const pairSize = pairFraction * usableSize;
    if (pairSize <= 0)
        return startValues;
    const boundedMin = Math.max(48, Math.min(minTrackPx, pairSize / 2 - 1));
    if (!Number.isFinite(boundedMin) || boundedMin <= 0)
        return startValues;
    const currentPx = current * usableSize;
    const nextCurrentPx = clamp(currentPx + delta, boundedMin, pairSize - boundedMin);
    const nextFraction = nextCurrentPx / usableSize;
    const pairedFraction = (pairSize - nextCurrentPx) / usableSize;
    const updated = [...startValues];
    updated[index] = nextFraction;
    updated[index + 1] = pairedFraction;
    return updated;
};
