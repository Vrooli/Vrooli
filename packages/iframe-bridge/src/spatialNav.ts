/**
 * Spatial navigation engine — 2-D directional focus management for gamepad
 * and keyboard-driven UIs.
 *
 * Given the currently-focused element and a direction (up / down / left / right),
 * finds the geometrically nearest focusable element, moves focus to it, and
 * manages focus groups with three modes:
 *
 *   `spatial`      — default: D-pad/stick navigates between focusable children.
 *   `passthrough`  — raw input forwarded to the component (graphs, canvases, …).
 *   `grid`         — children treated as a grid for predictable row/col nav.
 *
 * Zero runtime dependencies. Framework-agnostic.
 */

import { injectSpatialStyles, removeSpatialStyles } from './spatialNavStyles.js';

// ---------------------------------------------------------------------------
// Public types
// ---------------------------------------------------------------------------

export type Direction = 'up' | 'down' | 'left' | 'right';
export type FocusGroupMode = 'spatial' | 'passthrough' | 'grid' | 'modal';

export interface FocusGroupOptions {
  /**
   * If `true`, directional navigation wraps around (last → first, etc.).
   * Only applies to `spatial` and `grid` modes.
   */
  wrap?: boolean;
}

export interface SpatialNavOptions {
  /**
   * CSS selector used to discover focusable elements.
   * Default matches the standard interactive-element set.
   */
  focusableSelector?: string;
  /** Whether to inject the default focus-ring CSS. Default `true`. */
  injectDefaultFocusStyle?: boolean;
  /** Root element to scan for focusable elements. Default `document.body`. */
  rootElement?: HTMLElement;
  /**
   * Injectable seam — override `getBoundingClientRect` for testing.
   * Receives the element and should return a DOMRect-like object.
   */
  getBoundingClientRect?: (el: HTMLElement) => DOMRect;
  /**
   * Injectable seam — override visibility check for testing.
   * jsdom does not compute layout (`offsetParent` is always null), so tests
   * should pass `() => true` here.
   */
  isVisible?: (el: HTMLElement) => boolean;
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

/**
 * Standard focusable-element selector.
 * Reused from the useFocusTrap pattern in app-monitor.
 */
export const FOCUSABLE_SELECTOR =
  'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])';

/** Data attributes used as the public contract. */
const ATTR_ACTIVE = 'data-spatial-active';
const ATTR_FOCUS = 'data-spatial-focus';
const ATTR_GROUP = 'data-spatial-group';

/**
 * Overlap tolerance in pixels.  When filtering candidates by direction we
 * allow this much overlap on the primary axis so that elements that are
 * *almost* aligned are still reachable.
 */
const OVERLAP_TOLERANCE_PX = 4;

/**
 * Extra padding (px) when scrolling a focused element into view.
 * Prevents the element from sitting flush against the container edge.
 */
const SCROLL_PADDING_PX = 16;

// ---------------------------------------------------------------------------
// Internal geometry helpers
// ---------------------------------------------------------------------------

interface Rect {
  top: number;
  right: number;
  bottom: number;
  left: number;
  width: number;
  height: number;
}

function centerX(r: Rect): number {
  return r.left + r.width / 2;
}
function centerY(r: Rect): number {
  return r.top + r.height / 2;
}

/**
 * Returns `true` if `candidate` is in the given `direction` relative to
 * `current`.  Uses a generous overlap tolerance so that partially-aligned
 * elements are still considered.
 */
function isInDirection(current: Rect, candidate: Rect, direction: Direction): boolean {
  switch (direction) {
    case 'right':
      return candidate.left >= current.right - OVERLAP_TOLERANCE_PX;
    case 'left':
      return candidate.right <= current.left + OVERLAP_TOLERANCE_PX;
    case 'down':
      return candidate.top >= current.bottom - OVERLAP_TOLERANCE_PX;
    case 'up':
      return candidate.bottom <= current.top + OVERLAP_TOLERANCE_PX;
  }
}

/**
 * Scores how "close" a candidate is to the current element along a direction.
 * Lower is better.  Cross-axis distance is weighted 2× to strongly prefer
 * aligned elements.
 */
function scoreCandidate(current: Rect, candidate: Rect, direction: Direction): number {
  let primaryDist: number;
  let crossDist: number;

  switch (direction) {
    case 'right':
      primaryDist = candidate.left - current.right;
      crossDist = Math.abs(centerY(candidate) - centerY(current));
      break;
    case 'left':
      primaryDist = current.left - candidate.right;
      crossDist = Math.abs(centerY(candidate) - centerY(current));
      break;
    case 'down':
      primaryDist = candidate.top - current.bottom;
      crossDist = Math.abs(centerX(candidate) - centerX(current));
      break;
    case 'up':
      primaryDist = current.top - candidate.bottom;
      crossDist = Math.abs(centerX(candidate) - centerX(current));
      break;
  }

  // Clamp primary to 0 (overlap tolerance can produce negatives).
  if (primaryDist < 0) primaryDist = 0;

  return primaryDist + crossDist * 2;
}

// ---------------------------------------------------------------------------
// Focus group registry entry
// ---------------------------------------------------------------------------

interface FocusGroupEntry {
  element: HTMLElement;
  mode: FocusGroupMode;
  options: FocusGroupOptions;
}

// ---------------------------------------------------------------------------
// SpatialNavManager
// ---------------------------------------------------------------------------

export class SpatialNavManager {
  // Options (normalised)
  private readonly focusableSelector: string;
  private readonly injectDefaultFocusStyle: boolean;
  private readonly rootElement: HTMLElement;
  private readonly getRect: (el: HTMLElement) => DOMRect;
  private readonly checkVisible: (el: HTMLElement) => boolean;

  // State
  private active = false;
  private currentFocused: HTMLElement | null = null;
  private groups: FocusGroupEntry[] = [];
  private styleElement: HTMLStyleElement | null = null;
  /** Guard to prevent onNativeFocus from interfering during programmatic focus. */
  private focusingProgrammatically = false;
  /**
   * Modal scope stack.  When non-empty, spatial navigation is constrained to
   * the top element (e.g., a dialog).  Supports nesting (dialog opens dialog).
   */
  private scopeStack: HTMLElement[] = [];

  // Bound listeners
  private readonly onMouseMove: () => void;
  private readonly onTouchStart: () => void;
  private readonly onNativeFocus: (e: FocusEvent) => void;

  constructor(options?: SpatialNavOptions) {
    this.focusableSelector = options?.focusableSelector ?? FOCUSABLE_SELECTOR;
    this.injectDefaultFocusStyle = options?.injectDefaultFocusStyle ?? true;
    this.rootElement = options?.rootElement ?? document.body;
    this.getRect =
      options?.getBoundingClientRect ??
      ((el: HTMLElement) => el.getBoundingClientRect());
    this.checkVisible = options?.isVisible ?? ((el: HTMLElement) => this.defaultIsVisible(el));

    // Auto-exit spatial mode on mouse / touch.
    this.onMouseMove = () => {
      if (this.active) this.exitSpatialMode();
    };
    this.onTouchStart = () => {
      if (this.active) this.exitSpatialMode();
    };

    // Keep track of focus changes made outside spatial nav (e.g., Tab key).
    // Skips when we're the ones calling focus() to avoid double-processing.
    this.onNativeFocus = (e: FocusEvent) => {
      if (!this.active || this.focusingProgrammatically) return;
      const target = e.target;
      if (target instanceof HTMLElement) {
        this.setFocusRing(target);
      }
    };

    if (typeof window !== 'undefined') {
      window.addEventListener('mousemove', this.onMouseMove);
      window.addEventListener('touchstart', this.onTouchStart);
      document.addEventListener('focus', this.onNativeFocus, true);
    }
  }

  // -----------------------------------------------------------------------
  // Mode management
  // -----------------------------------------------------------------------

  /** Enter spatial navigation mode. */
  enterSpatialMode(): void {
    if (this.active) return;
    this.active = true;

    document.documentElement.setAttribute(ATTR_ACTIVE, '');

    if (this.injectDefaultFocusStyle && !this.styleElement) {
      this.styleElement = injectSpatialStyles();
    }

    // Focus the previously focused element, or the first focusable one.
    const target = this.currentFocused ?? this.findFirstFocusable();
    if (target) {
      this.focusElement(target);
    }
  }

  /** Exit spatial navigation mode. */
  exitSpatialMode(): void {
    if (!this.active) return;
    this.active = false;

    document.documentElement.removeAttribute(ATTR_ACTIVE);
    this.clearFocusRing();
  }

  /** Whether spatial mode is currently active. */
  isActive(): boolean {
    return this.active;
  }

  // -----------------------------------------------------------------------
  // Focus movement
  // -----------------------------------------------------------------------

  /**
   * Move focus in the given direction.
   * Returns `true` if focus was moved, `false` if no candidate was found
   * or the focused element is inside a passthrough group.
   */
  moveFocus(direction: Direction): boolean {
    if (!this.active) return false;

    // If we're inside a passthrough group, don't intercept.
    if (this.currentFocused && this.isInsidePassthroughGroup(this.currentFocused)) {
      return false;
    }

    const current = this.currentFocused;
    if (!current) {
      // Nothing focused — focus first element.
      const first = this.findFirstFocusable();
      if (first) {
        this.focusElement(first);
        return true;
      }
      return false;
    }

    // Determine the active focus group (innermost containing group).
    const group = this.findContainingGroup(current);

    let next: HTMLElement | null = null;
    if (group?.mode === 'grid') {
      next = this.findNextInGrid(current, direction, group);
    } else {
      next = this.findNextFocusable(current, direction, group);
    }

    if (next) {
      this.focusElement(next);
      return true;
    }

    return false;
  }

  /** Simulate a click on the currently focused element. */
  selectFocused(): void {
    if (this.currentFocused) {
      this.currentFocused.click();
    }
  }

  /** Navigate back — dispatches a popstate-like action. */
  goBack(): void {
    if (typeof window !== 'undefined' && window.history.length > 1) {
      window.history.back();
    }
  }

  // -----------------------------------------------------------------------
  // Focus group management
  // -----------------------------------------------------------------------

  /**
   * Register a focus group.  Returns a dispose function that unregisters it.
   */
  registerGroup(
    element: HTMLElement,
    mode: FocusGroupMode,
    options?: FocusGroupOptions,
  ): () => void {
    const entry: FocusGroupEntry = { element, mode, options: options ?? {} };
    this.groups.push(entry);
    element.setAttribute(ATTR_GROUP, mode);

    return () => {
      const idx = this.groups.indexOf(entry);
      if (idx >= 0) this.groups.splice(idx, 1);
      element.removeAttribute(ATTR_GROUP);
    };
  }

  /**
   * Cycle focus to the next (or previous) top-level focus group.
   * Used by bumper buttons (LB/RB) to jump between major UI sections.
   */
  cycleFocusGroup(direction: 'next' | 'prev'): boolean {
    if (this.groups.length === 0) return false;

    const currentGroup = this.currentFocused
      ? this.findContainingGroup(this.currentFocused)
      : null;

    let currentIdx = currentGroup ? this.groups.indexOf(currentGroup) : -1;
    if (currentIdx < 0) currentIdx = direction === 'next' ? -1 : this.groups.length;

    const step = direction === 'next' ? 1 : -1;
    const len = this.groups.length;

    // Walk groups in the given direction, looking for one with focusable children.
    for (let i = 1; i <= len; i++) {
      const idx = ((currentIdx + i * step) % len + len) % len;
      const group = this.groups[idx];
      const candidates = this.getFocusableElements(group.element);
      if (candidates.length > 0) {
        this.focusElement(candidates[0]);
        return true;
      }
    }

    return false;
  }

  // -----------------------------------------------------------------------
  // Modal scope management
  // -----------------------------------------------------------------------

  /**
   * Push a modal scope — all spatial navigation is constrained to within
   * `element` until `popScope()` is called.  Supports nesting.
   * Automatically focuses the first focusable element inside the scope.
   */
  pushScope(element: HTMLElement): void {
    this.scopeStack.push(element);

    // Focus the first focusable element inside the new scope.
    if (this.active) {
      const candidates = this.getFocusableElements(element);
      if (candidates.length > 0) {
        this.focusElement(candidates[0]);
      }
    }
  }

  /**
   * Pop the current modal scope, restoring the previous one (or root).
   */
  popScope(): void {
    this.scopeStack.pop();
  }

  /**
   * The currently active scope element, or `undefined` if no modal scope.
   */
  get activeScope(): HTMLElement | undefined {
    return this.scopeStack[this.scopeStack.length - 1];
  }

  // -----------------------------------------------------------------------
  // Cleanup
  // -----------------------------------------------------------------------

  dispose(): void {
    this.exitSpatialMode();

    if (typeof window !== 'undefined') {
      window.removeEventListener('mousemove', this.onMouseMove);
      window.removeEventListener('touchstart', this.onTouchStart);
      document.removeEventListener('focus', this.onNativeFocus, true);
    }

    if (this.styleElement) {
      removeSpatialStyles(this.styleElement);
      this.styleElement = null;
    }

    this.groups = [];
    this.currentFocused = null;
  }

  // -----------------------------------------------------------------------
  // Private: focus helpers
  // -----------------------------------------------------------------------

  private focusElement(el: HTMLElement): void {
    this.focusingProgrammatically = true;
    try {
      el.focus({ preventScroll: true });
      this.setFocusRing(el);
      this.scrollIntoViewIfNeeded(el);
    } finally {
      this.focusingProgrammatically = false;
    }
  }

  private setFocusRing(el: HTMLElement): void {
    // Clear ALL stale focus rings — prevents dual-selection when rapid
    // D-pad presses or native focus events leave orphaned attributes.
    const stale = this.rootElement.querySelectorAll<HTMLElement>(`[${ATTR_FOCUS}]`);
    for (const node of stale) {
      if (node !== el) node.removeAttribute(ATTR_FOCUS);
    }
    el.setAttribute(ATTR_FOCUS, 'true');
    this.currentFocused = el;
  }

  private clearFocusRing(): void {
    if (this.currentFocused) {
      this.currentFocused.removeAttribute(ATTR_FOCUS);
    }
  }

  /**
   * Scroll the nearest scrollable ancestor so `el` is visible.
   * Uses `container.scrollTo()` — NEVER `scrollIntoView` (iframe-unsafe).
   * Adds padding so the element doesn't sit flush against the container edge.
   */
  private scrollIntoViewIfNeeded(el: HTMLElement): void {
    const container = this.findScrollableAncestor(el);
    if (!container) return;

    const elRect = this.getRect(el);
    const containerRect = this.getRect(container);

    let scrollTop = container.scrollTop;
    let scrollLeft = container.scrollLeft;
    let changed = false;

    // Vertical — include padding so the element isn't flush against the edge
    if (elRect.top < containerRect.top + SCROLL_PADDING_PX) {
      scrollTop -= (containerRect.top + SCROLL_PADDING_PX) - elRect.top;
      changed = true;
    } else if (elRect.bottom > containerRect.bottom - SCROLL_PADDING_PX) {
      scrollTop += elRect.bottom - (containerRect.bottom - SCROLL_PADDING_PX);
      changed = true;
    }

    // Horizontal
    if (elRect.left < containerRect.left + SCROLL_PADDING_PX) {
      scrollLeft -= (containerRect.left + SCROLL_PADDING_PX) - elRect.left;
      changed = true;
    } else if (elRect.right > containerRect.right - SCROLL_PADDING_PX) {
      scrollLeft += elRect.right - (containerRect.right - SCROLL_PADDING_PX);
      changed = true;
    }

    if (changed) {
      container.scrollTo({ top: scrollTop, left: scrollLeft, behavior: 'smooth' });
    }
  }

  private findScrollableAncestor(el: HTMLElement): HTMLElement | null {
    let node = el.parentElement;
    while (node && node !== document.body) {
      const style = getComputedStyle(node);
      // Skip `display: contents` elements (e.g., SpatialGroup wrappers) —
      // they don't generate a box and can't be scroll containers.
      if (style.display !== 'contents') {
        const overflowY = style.overflowY;
        const overflowX = style.overflowX;
        if (
          overflowY === 'auto' || overflowY === 'scroll' ||
          overflowX === 'auto' || overflowX === 'scroll'
        ) {
          return node;
        }
      }
      node = node.parentElement;
    }
    return null;
  }

  // -----------------------------------------------------------------------
  // Private: candidate discovery
  // -----------------------------------------------------------------------

  private getFocusableElements(container?: HTMLElement): HTMLElement[] {
    // When a modal scope is active, constrain to within it — ignoring
    // both the provided container and the root element.
    const scope = this.activeScope;
    const root = scope ?? container ?? this.rootElement;

    // If a scope is active and a container was requested, only use the
    // container if it's inside the scope (otherwise we'd escape the modal).
    const searchRoot = scope && container && scope.contains(container)
      ? container
      : root;

    const nodes = searchRoot.querySelectorAll<HTMLElement>(this.focusableSelector);
    const result: HTMLElement[] = [];

    for (const node of nodes) {
      if (this.isVisible(node)) {
        result.push(node);
      }
    }

    return result;
  }

  private isVisible(el: HTMLElement): boolean {
    return this.checkVisible(el);
  }

  private defaultIsVisible(el: HTMLElement): boolean {
    // offsetParent is null for hidden elements (display: none, or not in DOM).
    // Exceptions: <body>, position:fixed, and elements inside display:contents ancestors.
    if (el.offsetParent === null && el !== document.body) {
      const style = getComputedStyle(el);
      if (style.position !== 'fixed' && style.display !== 'none') {
        // Walk ancestors to check if any use display:contents — those have
        // null offsetParent on descendants but are still visible.
        let ancestor = el.parentElement;
        let hasContentsAncestor = false;
        while (ancestor && ancestor !== document.body) {
          const ancestorDisplay = getComputedStyle(ancestor).display;
          if (ancestorDisplay === 'contents') {
            hasContentsAncestor = true;
            break;
          }
          if (ancestorDisplay === 'none') return false;
          ancestor = ancestor.parentElement;
        }
        if (!hasContentsAncestor) return false;
      } else if (style.display === 'none') {
        return false;
      }
    }
    if (el.getAttribute('aria-hidden') === 'true') return false;
    return true;
  }

  private findFirstFocusable(): HTMLElement | null {
    const candidates = this.getFocusableElements();
    return candidates[0] ?? null;
  }

  // -----------------------------------------------------------------------
  // Private: spatial scoring
  // -----------------------------------------------------------------------

  private findNextFocusable(
    current: HTMLElement,
    direction: Direction,
    group: FocusGroupEntry | null,
  ): HTMLElement | null {
    const candidates = this.getFocusableElements(group?.element);
    const currentRect = this.getRect(current);

    let best: HTMLElement | null = null;
    let bestScore = Infinity;

    for (const candidate of candidates) {
      if (candidate === current) continue;

      const candidateRect = this.getRect(candidate);
      if (!isInDirection(currentRect, candidateRect, direction)) continue;

      const s = scoreCandidate(currentRect, candidateRect, direction);
      if (s < bestScore) {
        bestScore = s;
        best = candidate;
      }
    }

    return best;
  }

  // -----------------------------------------------------------------------
  // Private: grid navigation
  // -----------------------------------------------------------------------

  private findNextInGrid(
    current: HTMLElement,
    direction: Direction,
    group: FocusGroupEntry,
  ): HTMLElement | null {
    const candidates = this.getFocusableElements(group.element);
    if (candidates.length === 0) return null;

    // Build row/col index from element positions.
    const rects = candidates.map((el) => ({ el, rect: this.getRect(el) }));

    // Sort by top then left to establish visual order.
    rects.sort((a, b) => a.rect.top - b.rect.top || a.rect.left - b.rect.left);

    // Determine rows: elements sharing approximately the same top value.
    const ROW_TOLERANCE = 8;
    const rows: { el: HTMLElement; rect: Rect }[][] = [];
    let currentRow: { el: HTMLElement; rect: Rect }[] = [];
    let rowTop = -Infinity;

    for (const item of rects) {
      if (item.rect.top - rowTop > ROW_TOLERANCE) {
        if (currentRow.length > 0) rows.push(currentRow);
        currentRow = [item];
        rowTop = item.rect.top;
      } else {
        currentRow.push(item);
      }
    }
    if (currentRow.length > 0) rows.push(currentRow);

    // Find current position.
    let curRow = -1;
    let curCol = -1;
    for (let r = 0; r < rows.length; r++) {
      for (let c = 0; c < rows[r].length; c++) {
        if (rows[r][c].el === current) {
          curRow = r;
          curCol = c;
          break;
        }
      }
      if (curRow >= 0) break;
    }

    if (curRow < 0) {
      // Current element not found in grid — fall back to spatial scoring.
      return this.findNextFocusable(current, direction, group);
    }

    const wrap = group.options.wrap ?? false;
    let nextRow = curRow;
    let nextCol = curCol;

    switch (direction) {
      case 'right':
        nextCol++;
        if (nextCol >= rows[nextRow].length) {
          if (wrap) nextCol = 0;
          else return null;
        }
        break;
      case 'left':
        nextCol--;
        if (nextCol < 0) {
          if (wrap) nextCol = rows[nextRow].length - 1;
          else return null;
        }
        break;
      case 'down':
        nextRow++;
        if (nextRow >= rows.length) {
          if (wrap) nextRow = 0;
          else return null;
        }
        // Clamp column to row length.
        if (nextCol >= rows[nextRow].length) nextCol = rows[nextRow].length - 1;
        break;
      case 'up':
        nextRow--;
        if (nextRow < 0) {
          if (wrap) nextRow = rows.length - 1;
          else return null;
        }
        if (nextCol >= rows[nextRow].length) nextCol = rows[nextRow].length - 1;
        break;
    }

    return rows[nextRow][nextCol].el;
  }

  // -----------------------------------------------------------------------
  // Private: focus group helpers
  // -----------------------------------------------------------------------

  /**
   * Find the innermost registered focus group containing the element.
   * When a modal scope is active, only groups within the scope are considered.
   */
  private findContainingGroup(el: HTMLElement): FocusGroupEntry | null {
    const scope = this.activeScope;
    let best: FocusGroupEntry | null = null;

    for (const group of this.groups) {
      // Skip groups outside the active modal scope.
      if (scope && !scope.contains(group.element)) continue;

      if (group.element.contains(el)) {
        // Pick the innermost (most deeply nested) group.
        if (!best || best.element.contains(group.element)) {
          best = group;
        }
      }
    }

    return best;
  }

  /**
   * Check whether `el` is inside a passthrough focus group.
   */
  private isInsidePassthroughGroup(el: HTMLElement): boolean {
    const group = this.findContainingGroup(el);
    return group?.mode === 'passthrough';
  }
}
