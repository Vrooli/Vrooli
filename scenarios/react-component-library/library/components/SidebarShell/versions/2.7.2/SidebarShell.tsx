/**
 * @libraryId react-component-library:SidebarShell
 * @displayName Sidebar
 * @description The persistent navigation or workspace panel with sections, active state, collapse to icons, keyboard traversal, independent scrolling, and stable width.
 * @version 2.7.2
 * @tags []
 * @deps {"react":"^18","lucide-react":"^0.424.0","react-component-library:useEscapeKey":"^1.0.0","react-component-library:useResizablePanel":"^1.0.0","react-component-library:ResizeHandle":"^1.0.0","react-component-library:useSwipeGesture":"^2.0.0"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import {
  forwardRef,
  useCallback,
  useEffect,
  useRef,
  type CSSProperties,
  type HTMLAttributes,
  type ReactNode,
  type PointerEvent as ReactPointerEvent,
  type Ref,
  type RefObject,
} from "react";
import { X } from "lucide-react";
import { useEscapeKey } from "@vrooli/react-component-library/useEscapeKey/1";
import {
  useResizablePanel,
  type ResizeStorage,
} from "@vrooli/react-component-library/useResizablePanel/1";
import { ResizeHandle, useResizeStrings } from "@vrooli/react-component-library/ResizeHandle/1";
import { useSwipeGesture } from "@vrooli/react-component-library/useSwipeGesture/2";
import { resolveGestureFeel } from "@vrooli/react-component-library/GestureTokens/1";
import {
  useGestureDirection,
  type AnchorEdge,
} from "@vrooli/react-component-library/useHandedness/1";
export const sidebarShellStyles = `
[data-rcl-sidebar-shell] { min-block-size: 0; min-inline-size: 0; display: flex; flex-direction: column; background: var(--color-surface); color: var(--color-foreground); }

/* 2.5.0 — the seam between the panel and the content it sits beside follows the
   physical edge the panel is anchored to. Physical properties are paired with
   data-edge on purpose: data-edge is already a resolved side, so restating it
   logically would re-introduce the very indirection that made 2.4.0 disagree
   with itself in RTL. */
[data-rcl-sidebar-shell][data-edge="left"] { border-right: var(--border-hairline) solid var(--color-border); }
[data-rcl-sidebar-shell][data-edge="right"] { border-left: var(--border-hairline) solid var(--color-border); }

[data-rcl-sidebar-shell][data-mode="persistent"] { position: relative; z-index: auto; block-size: 100%; flex-shrink: 0; box-shadow: var(--elev-flat); }
[data-rcl-sidebar-shell][data-mode="overlay"], [data-rcl-sidebar-shell][data-mode="responsive"] { position: fixed; inset-block: 0; z-index: var(--layer-modal); block-size: 100dvh; inline-size: 100%; max-inline-size: none; padding-block: var(--rcl-safe-top, env(safe-area-inset-top)) var(--rcl-safe-bottom, env(safe-area-inset-bottom)); box-shadow: var(--elev-modal); transform: translateX(0); transition: transform var(--dur-quick) var(--ease-standard), visibility var(--dur-quick) var(--ease-standard); }

/* 2.5.0 — where the drawer rests. 2.4.0 pinned this with inset-inline-start,
   which mirrors with the locale, while the closed transform below stayed
   physical. In RTL the panel therefore moved to the right edge and its closed
   transform pushed it further into the viewport instead of out of it. */
[data-rcl-sidebar-shell][data-mode="overlay"][data-edge="left"],
[data-rcl-sidebar-shell][data-mode="responsive"][data-edge="left"] { left: 0; right: auto; }
[data-rcl-sidebar-shell][data-mode="overlay"][data-edge="right"],
[data-rcl-sidebar-shell][data-mode="responsive"][data-edge="right"] { right: 0; left: auto; }

/* The closed state travels out through the edge the panel is anchored to.
   Four attributes; the opening rule below matches that count deliberately. */
[data-rcl-sidebar-shell][data-mode="overlay"][data-open="false"][data-edge="left"],
[data-rcl-sidebar-shell][data-mode="responsive"][data-open="false"][data-edge="left"] { visibility: hidden; transform: translateX(-100%); }
[data-rcl-sidebar-shell][data-mode="overlay"][data-open="false"][data-edge="right"],
[data-rcl-sidebar-shell][data-mode="responsive"][data-open="false"][data-edge="right"] { visibility: hidden; transform: translateX(100%); }

[data-rcl-sidebar-shell][data-mode="responsive"] { display: flex; }
[data-rcl-sidebar-shell] .rcl-sidebar-shell__header { display: flex; align-items: center; justify-content: space-between; gap: var(--space-xs); min-block-size: var(--tap-target-min); border-block-end: var(--border-hairline) solid var(--color-border); padding-inline: var(--space-xs); }
[data-rcl-sidebar-shell] .rcl-sidebar-shell__header-content { min-inline-size: 0; overflow-wrap: anywhere; }
[data-rcl-sidebar-shell] .rcl-sidebar-shell__close { min-block-size: var(--tap-target-min); min-inline-size: var(--tap-target-min); flex-shrink: 0; border: 0; border-radius: var(--radius-control); background: transparent; color: var(--color-muted-foreground); cursor: pointer; }
[data-rcl-sidebar-shell] .rcl-sidebar-shell__close:hover { background: var(--color-surface-muted); color: var(--color-foreground); }
[data-rcl-sidebar-shell] .rcl-sidebar-shell__icon { inline-size: var(--space-sm); block-size: var(--space-sm); }
[data-rcl-sidebar-shell] .rcl-sidebar-shell__content { min-block-size: 0; min-inline-size: 0; flex: 1; overflow: auto; }
[data-rcl-sidebar-shell] .rcl-sidebar-shell__resize { position: absolute; inset-block: 0; inset-inline-end: calc(var(--space-3xs) * -1); z-index: var(--layer-sticky); inline-size: var(--space-xs); border: 0; background: transparent; cursor: col-resize; }
[data-rcl-sidebar-shell] .rcl-sidebar-shell__resize:hover { background: color-mix(in srgb, var(--color-primary) 25%, transparent); }
[data-rcl-sidebar-backdrop] { position: fixed; inset: 0; z-index: calc(var(--layer-modal) - 1); border: 0; background: color-mix(in srgb, var(--color-shell) 60%, transparent); cursor: default; }

/* 2.1.0 — drag-to-dismiss.
   pan-y is what lets the gesture coexist with the nav list: the browser
   keeps vertical scrolling and hands horizontal movement to the pointer
   handlers, so a drag down the list still scrolls.
   While a drag is live the transform is written every frame, so the easing
   that animates open/close must be off or it fights the finger, and the
   panel is promoted once at pointerdown rather than mid-gesture. */
[data-rcl-sidebar-shell][data-swipe="true"] { touch-action: pan-y; }
/* Every descendant, not just the shell's own content box. A consumer's own
   scrolling list is a scroll container, and a scroll container resolves
   touch-action against itself before the shell above it — so with only the
   shell restricted, a drag starting inside the list was claimed as a scroll
   and cancelled the pointer stream. That is why the gesture used to work
   only in the strip above the list, where nothing scrolls.
   A child that genuinely claims the inline axis opts out with data-rcl-pan-x;
   SwipeActions is the component that does, and the pairing is covered by that
   component's "inside a drawer" story. */
[data-rcl-sidebar-shell][data-swipe="true"] *:not([data-rcl-pan-x]):not([data-rcl-pan-x] *) { touch-action: pan-y; }
[data-rcl-sidebar-shell][data-dragging="true"] { transition: none; will-change: transform; }

/* 2.2.0 — the opening drag.
   The strip sits under the backdrop layer and only exists while the drawer is
   closed, so it never competes with the panel it opens. pan-y keeps a
   vertical flick on the page behind it scrolling the page. */
[data-rcl-sidebar-edge] { position: fixed; inset-block: 0; z-index: calc(var(--layer-modal) - 2); touch-action: pan-y; background: transparent; }
[data-rcl-sidebar-edge][data-edge="left"] { left: 0; right: auto; }
[data-rcl-sidebar-edge][data-edge="right"] { right: 0; left: auto; }

/* While an opening drag is live the panel is on screen but not yet "open", so
   it needs the visibility the closed rule takes away, and none of the easing
   that would fight the finger.
   Matched to the closed rule's specificity on purpose, and placed after it.
   The closed rule is four attributes; a three-attribute opening rule loses the
   cascade, and the panel then tracked the finger correctly and did all of it
   invisibly — the drag appeared to do nothing until release, when data-open
   flipped and the ordinary transition slid the panel in. [data-edge] is present
   to match the count, not to filter. */
[data-rcl-sidebar-shell][data-mode="overlay"][data-opening="true"][data-edge],
[data-rcl-sidebar-shell][data-mode="responsive"][data-opening="true"][data-edge] { visibility: visible; transition: none; will-change: transform; }

/* Desktop last, so its resets win every tie against the drawer rules above at
   equal specificity. A responsive shell stops being a drawer here. */
@media (min-width: 768px) {
  [data-rcl-sidebar-shell][data-mode="responsive"] { position: relative; inset: auto; left: auto; right: auto; z-index: auto; block-size: 100%; inline-size: auto; padding-block: 0; box-shadow: var(--elev-flat); transform: none; visibility: visible; }
  [data-rcl-sidebar-shell][data-mode="responsive"][data-open="false"][data-edge] { transform: none; visibility: visible; }
  [data-rcl-sidebar-shell][data-mode="responsive"] .rcl-sidebar-shell__header { display: none; }
  [data-rcl-sidebar-shell][data-mode="overlay"] { inline-size: min(22rem, 100vw); max-inline-size: 22rem; }
  [data-rcl-sidebar-backdrop][data-mode="responsive"] { display: none; }
}
`;

// Appended in 2.0.0: the shell now owns the resize affordance, so it also owns
// the rule that a drawer has no seam to drag. Below the desktop breakpoint a
// responsive shell is a full-width dialog; an overlay shell always is.
export const sidebarShellResizeStyles = `
[data-rcl-sidebar-shell][data-mode="overlay"] [data-rcl-resize-handle] { display: none; }
[data-rcl-sidebar-shell][data-mode="responsive"] [data-rcl-resize-handle] { display: none; }
@media (min-width: 768px) {
  [data-rcl-sidebar-shell][data-mode="responsive"] [data-rcl-resize-handle] { display: flex; }
}
`;
export interface SidebarShellResizeConfig {
  /** The region the sidebar shares with its adjacent content. */
  containerRef: RefObject<HTMLElement | null>;
  min: number;
  max: number;
  defaultSize: number;
  /** Space the adjacent region must keep. */
  adjacentMin?: number;
  storageKey?: string;
  storage?: ResizeStorage;
  snapPoints?: readonly number[];
  collapseBelow?: number;
  /** Names the panel in the accessible label and value text. */
  panelName?: string;
  step?: number;
  coarseStep?: number;
  disabled?: boolean;
  onCommit?: (size: number) => void;
  onCollapse?: (collapsed: boolean) => void;
}

export interface SidebarShellProps {
  children: ReactNode;
  mode?: "responsive" | "overlay" | "persistent";
  mobileOpen: boolean;
  onMobileClose: () => void;
  mobileLabel: string;
  desktopLabel?: string;
  closeLabel: string;
  mobileHeader?: ReactNode;
  /** Composes the resize behavior and affordance. Preferred over `width`. */
  resizable?: SidebarShellResizeConfig;
  /** Root test id; every part derives from it. */
  testId?: string;
  /**
   * Who owns the drawer header and its close control. `"shell"` renders the
   * header row and close button; `"content"` renders neither, for a consumer
   * whose own children already carry a close affordance. Persistent mode has no
   * header either way.
   */
  mobileChrome?: "shell" | "content";
  /** @deprecated Pass `resizable`. Removed in 3.0.0. */
  width?: number;
  /** @deprecated Pass `resizable`. Removed in 3.0.0. */
  resizeHandleProps?: HTMLAttributes<HTMLDivElement>;
  /**
   * Dismiss the drawer by dragging it toward its own edge. On by default in
   * drawer mode: a surface a thumb can open should be one a thumb can close,
   * and the close button remains the non-gesture path. Ignored in persistent
   * mode, where there is nothing to dismiss.
   */
  swipeToClose?: boolean;
  /**
   * Open the drawer by dragging in from the edge it lives on — the gesture
   * every mobile OS trains people to expect, and the counterpart to
   * `swipeToClose`. Enabled by default once `onMobileOpen` is supplied, since
   * without a way to report the open there is nothing to enable.
   */
  edgeSwipeToOpen?: boolean;
  /** Called when an edge drag asks for the drawer. Required for `edgeSwipeToOpen`. */
  onMobileOpen?: () => void;
  /**
   * Width of the invisible strip along the screen edge that starts an opening
   * drag. Wide enough to hit with a thumb, narrow enough not to swallow taps
   * meant for whatever sits underneath it.
   */
  edgeSwipeWidth?: string;
  /**
   * Inline size of the drawer in overlay mode, as any CSS length —
   * `"20rem"`, `"90%"`, `"min(22rem, calc(100% - 3rem))"`. The remainder is
   * backdrop, and the backdrop dismisses, so this is also how wide the
   * tap-to-close strip is. Omit to size the panel from `className` instead.
   * Persistent mode ignores it; that width belongs to `resizable`.
   */
  mobileWidth?: string;
  /**
   * Which logical edge the drawer is anchored to.
   *
   * Omit it and the shell follows the app's handedness preference, which is an
   * ergonomic setting rather than a locale one: moving the drawer within thumb
   * reach must not mirror the interface's text, and mirroring the text must not
   * be the only way to move the drawer. Writing direction is applied on top, so
   * `inline-start` is the left edge in English and the right edge in Arabic.
   */
  side?: AnchorEdge;
  className?: string;
  panelClassName?: string;
  contentClassName?: string;
  backdropClassName?: string;
}

const cn = (...inputs: Array<string | undefined>) => inputs.filter(Boolean).join(" ");

/** Movement, in px, before a drag is assigned to an axis. */
const gestureAxisSlop = resolveGestureFeel().axisSlop;

/** Travel, in px, that commits an opening drag on release. */
const EDGE_OPEN_THRESHOLD = 64;

/** Travel per ms that commits an opening drag regardless of distance. */
const EDGE_OPEN_VELOCITY = 0.4;

function assignRef<T>(ref: Ref<T> | undefined, value: T | null) {
  if (typeof ref === "function") ref(value);
  else if (ref && typeof ref === "object") (ref as { current: T | null }).current = value;
}

export const SidebarShell = forwardRef<HTMLDivElement, SidebarShellProps>(function SidebarShell(
  {
    children,
    mode = "responsive",
    mobileOpen,
    onMobileClose,
    mobileLabel,
    desktopLabel,
    closeLabel,
    mobileHeader,
    resizable,
    testId = "navigation.sidebar",
    mobileChrome = "shell",
    swipeToClose = true,
    edgeSwipeToOpen,
    onMobileOpen,
    edgeSwipeWidth = "1.25rem",
    mobileWidth,
    side,
    // These two fields are intentionally retained for the documented one-release
    // migration path; the latest version must still expose the replacement.
    // eslint-disable-next-line @typescript-eslint/no-deprecated
    width,
    // eslint-disable-next-line @typescript-eslint/no-deprecated
    resizeHandleProps,
    className,
    panelClassName,
    contentClassName,
    backdropClassName,
  },
  ref,
) {
  const isPersistent = mode === "persistent";
  const isDialogOpen = !isPersistent && mobileOpen;

  useEscapeKey(isDialogOpen, onMobileClose);

  const panelRef = useRef<HTMLDivElement | null>(null);
  const setPanelRef = useCallback(
    (node: HTMLDivElement | null) => {
      panelRef.current = node;
      assignRef(ref, node);
    },
    [ref],
  );

  // Drag-to-dismiss. The gesture rides the panel itself rather than a grabber:
  // a sidebar has no handle to grab, and a drawer that only closes by button
  // reads as stuck on a touch screen.
  //
  // Two things keep it honest. The offset is measured in real pixels from the
  // pointer stream, not from useSwipe's progress fraction — that fraction is
  // distance/threshold clamped to 1, so feeding it a percentage transform
  // moves the panel by a multiple of the finger and stops dead at the
  // threshold. And the writes are imperative: a state update per pointermove
  // would re-render the whole navigation tree on every frame of a drag.
  const swipeEnabled = swipeToClose && !isPersistent;
  const dragOriginX = useRef(0);
  const dragOriginY = useRef(0);
  const dragAxis = useRef<"undecided" | "inline" | "block">("undecided");
  const dragExtent = useRef(0);
  /**
   * Set for the life of a drag that began inside a descendant claiming the
   * inline axis for itself -- an open row of swipe actions, say, whose dismiss
   * travels the same direction as this drawer's.
   *
   * Declining has to last the whole gesture, not just the pointerdown. Bailing
   * only at the start would leave the move handler running against a stale
   * origin from some earlier press, and the panel would jump.
   */
  const dragDeclined = useRef(false);
  // 2.5.0 — one resolver for both halves of the component. Until now the
  // gesture read `getComputedStyle(panel).direction` privately while the
  // stylesheet resolved the edge on its own, and the two could disagree: in RTL
  // the panel sat on the right and its closed transform still pushed it left.
  // The ref exists because the pointer handlers run outside render.
  const dismiss = useGestureDirection("dismiss", { anchor: side, elementRef: panelRef });
  const dismissLeftward = useRef(dismiss.sign < 0);
  dismissLeftward.current = dismiss.sign < 0;

  const markDragging = useCallback((dragging: boolean) => {
    const panel = panelRef.current;
    if (panel) panel.dataset.dragging = dragging ? "true" : "false";
  }, []);

  const writeOffset = useCallback(
    (pixels: number) => {
      const panel = panelRef.current;
      if (panel) {
        const signed = dismissLeftward.current ? -pixels : pixels;
        panel.style.transform = pixels > 0 ? `translateX(${String(signed)}px)` : "";
      }
      markDragging(pixels > 0);
    },
    [markDragging],
  );

  // Carry the panel the rest of the way out from wherever the finger left it,
  // so the close does not begin with a jump back to the open position.
  const writeDismissed = useCallback(() => {
    const panel = panelRef.current;
    markDragging(false);
    if (panel && dragExtent.current > 0) {
      const signed = dismissLeftward.current ? -dragExtent.current : dragExtent.current;
      panel.style.transform = `translateX(${String(signed)}px)`;
    }
  }, [markDragging]);

  const swipe = useSwipeGesture({
    direction: "left",
    stages: [resolveGestureFeel().dismissThreshold],
    releaseMode: "commit",
    onMove: ({ offset }) => writeOffset(offset),
    onRelease: (release) => {
      if (release.outcome === "commit") {
        writeDismissed();
        onMobileClose();
      } else writeOffset(0);
    },
  });

  const swipeRight = useSwipeGesture({
    direction: "right",
    stages: [resolveGestureFeel().dismissThreshold],
    releaseMode: "commit",
    onMove: ({ offset }) => writeOffset(offset),
    onRelease: (release) => {
      if (release.outcome === "commit") {
        writeDismissed();
        onMobileClose();
      } else writeOffset(0);
    },
  });

  // A dismissing drag ends with an inline transform parked at the closed
  // position, and inline styles outrank the stylesheet that would otherwise
  // slide the panel back. Clearing it on every open is what keeps a
  // swipe-closed drawer re-openable: without this the next open renders the
  // backdrop over a panel still translated off-screen, which reads as "open
  // but invisible".
  useEffect(() => {
    if (mobileOpen) writeOffset(0);
  }, [mobileOpen, writeOffset]);

  // Opening drag. The panel is in the tree while closed (hidden, parked at
  // -100%), so the gesture can carry it in from the edge under the finger
  // rather than waiting for the open to commit and animating from scratch.
  const edgeOpenEnabled =
    !isPersistent &&
    !mobileOpen &&
    (edgeSwipeToOpen ?? Boolean(onMobileOpen)) &&
    Boolean(onMobileOpen);

  const writeOpening = useCallback((pixels: number) => {
    const panel = panelRef.current;
    if (!panel) return;
    const extent = dragExtent.current;
    if (extent <= 0) return;
    // Travel is measured from the closed position, so the panel sits at
    // -extent and walks toward 0 as the finger moves inward.
    const remaining = Math.max(0, extent - Math.min(pixels, extent));
    const signed = dismissLeftward.current ? -remaining : remaining;
    panel.dataset.opening = "true";
    panel.style.transform = `translateX(${String(signed)}px)`;
  }, []);

  const clearOpening = useCallback(
    (commit: boolean) => {
      const panel = panelRef.current;
      if (!panel) return;
      delete panel.dataset.opening;
      // On commit the open state takes over and the stylesheet holds the panel
      // at 0; on cancel it must fall back to the closed transform the sheet
      // already describes. Either way the inline override has to go.
      panel.style.transform = "";
      if (commit) markDragging(false);
    },
    [markDragging],
  );

  // The opening drag tracks on window listeners rather than on the strip's own
  // React handlers. The strip is barely a thumb wide, so the finger leaves it
  // on the first millimetre of travel; from then on only the window sees the
  // pointer. Element pointer capture is the usual answer, but useSwipe
  // swallows a failed setPointerCapture silently, and a capture that never
  // took looks exactly like this: nothing moves until release, when the
  // terminating event arrives on the element again and the commit fires. Going
  // straight to the window removes the failure mode instead of depending on it.
  const edgeGesture = useRef<{
    originX: number;
    originY: number;
    startedAt: number;
    axis: "undecided" | "inline" | "block";
  } | null>(null);
  const edgeListeners = useRef<(() => void) | null>(null);

  const detachEdge = useCallback(() => {
    edgeListeners.current?.();
    edgeListeners.current = null;
    edgeGesture.current = null;
  }, []);

  // A gesture in flight outlives a re-render but must not outlive the shell.
  useEffect(() => detachEdge, [detachEdge]);

  const beginEdgeGesture = useCallback(
    (event: ReactPointerEvent<HTMLDivElement>) => {
      const panel = panelRef.current;
      if (!panel) return;
      detachEdge();
      // The panel is laid out while closed — hidden and translated away, not
      // removed — so it measures correctly before it has ever been shown.
      dragExtent.current = panel.getBoundingClientRect().width || panel.offsetWidth;
      edgeGesture.current = {
        originX: event.clientX,
        originY: event.clientY,
        startedAt: performance.now(),
        axis: "undecided",
      };

      const travelOf = (clientX: number) => {
        const dx = clientX - (edgeGesture.current?.originX ?? 0);
        return Math.max(0, dismissLeftward.current ? dx : -dx);
      };

      const onMove = (moveEvent: PointerEvent) => {
        const gesture = edgeGesture.current;
        if (!gesture) return;
        const dx = moveEvent.clientX - gesture.originX;
        const dy = moveEvent.clientY - gesture.originY;
        if (gesture.axis === "undecided") {
          if (Math.abs(dx) < gestureAxisSlop && Math.abs(dy) < gestureAxisSlop) return;
          gesture.axis = Math.abs(dx) > Math.abs(dy) ? "inline" : "block";
        }
        if (gesture.axis !== "inline") return;
        // Non-passive, so this is what keeps the page behind the strip still
        // while the drawer is being pulled in.
        if (moveEvent.cancelable) moveEvent.preventDefault();
        writeOpening(travelOf(moveEvent.clientX));
      };

      const onEnd = (endEvent: PointerEvent) => {
        const gesture = edgeGesture.current;
        detachEdge();
        if (!gesture || gesture.axis !== "inline") {
          clearOpening(false);
          return;
        }
        const travelled = travelOf(endEvent.clientX);
        const elapsed = Math.max(1, performance.now() - gesture.startedAt);
        const committed =
          travelled >= EDGE_OPEN_THRESHOLD || travelled / elapsed >= EDGE_OPEN_VELOCITY;
        if (committed) {
          clearOpening(true);
          onMobileOpen?.();
        } else {
          clearOpening(false);
        }
      };

      // A cancel is the browser taking the gesture away, not the user finishing
      // it, so it must abandon the drag rather than run the commit test. Wiring
      // both to the same handler meant a browser-initiated cancel past the
      // threshold opened the drawer the user was no longer asking for.
      const onCancel = () => {
        detachEdge();
        clearOpening(false);
      };
      window.addEventListener("pointermove", onMove, { passive: false });
      window.addEventListener("pointerup", onEnd);
      window.addEventListener("pointercancel", onCancel);
      edgeListeners.current = () => {
        window.removeEventListener("pointermove", onMove);
        window.removeEventListener("pointerup", onEnd);
        window.removeEventListener("pointercancel", onCancel);
      };
    },
    [clearOpening, detachEdge, onMobileOpen, writeOpening],
  );

  const edgeProps = { onPointerDown: beginEdgeGesture };

  // If a closing drag ends with the pointer somewhere the panel's own handlers
  // cannot see — released over the backdrop, or cancelled by the browser — the
  // React pointerup never runs and the inline transform is left stranded,
  // stranding the panel with it. This releases the drag no matter where it
  // ends; the element handlers still own the commit decision when they do run.
  const closeSafetyNet = useRef<(() => void) | null>(null);

  const armCloseSafetyNet = useCallback(() => {
    closeSafetyNet.current?.();
    const release = () => {
      closeSafetyNet.current?.();
      closeSafetyNet.current = null;
      if (dragAxis.current === "inline") writeOffset(0);
      dragAxis.current = "undecided";
    };
    window.addEventListener("pointerup", release);
    window.addEventListener("pointercancel", release);
    closeSafetyNet.current = () => {
      window.removeEventListener("pointerup", release);
      window.removeEventListener("pointercancel", release);
    };
  }, [writeOffset]);

  useEffect(() => () => closeSafetyNet.current?.(), []);

  // useSwipe fixes its direction at render time, but which edge counts as
  // "away" is only known once the panel is measured, so both directions are
  // instantiated and the gesture picks one at pointerdown.
  const handlersFor = () => (dismissLeftward.current ? swipe : swipeRight);

  const dragProps = swipeEnabled
    ? {
        onPointerDown: (event: ReactPointerEvent<HTMLDivElement>) => {
          const panel = panelRef.current;
          if (!panel) return;
          // A descendant that has already claimed the inline axis owns this
          // drag. The check is on the attribute rather than on propagation
          // because a descendant cannot stop propagation without also cutting
          // off its own window-level listeners.
          const target = event.target as Element | null;
          dragDeclined.current = Boolean(target?.closest("[data-rcl-gesture-claim]"));
          if (dragDeclined.current) return;
          dragOriginX.current = event.clientX;
          dragOriginY.current = event.clientY;
          dragAxis.current = "undecided";
          dragExtent.current = panel.getBoundingClientRect().width;
          armCloseSafetyNet();
          handlersFor().onPointerDown(event);
        },
        onPointerMove: (event: ReactPointerEvent<HTMLDivElement>) => {
          if (dragDeclined.current) return;
          const dx = event.clientX - dragOriginX.current;
          const dy = event.clientY - dragOriginY.current;
          // Decide once, on the first movement that clears the slop radius,
          // which axis owns the gesture. Without this the drawer only dragged
          // where there was nothing to scroll: inside the nav list any
          // vertical component started a scroll, and that scroll cancelled the
          // pointer stream mid-gesture.
          if (dragAxis.current === "undecided") {
            if (Math.abs(dx) < gestureAxisSlop && Math.abs(dy) < gestureAxisSlop) return;
            dragAxis.current = Math.abs(dx) > Math.abs(dy) ? "inline" : "block";
            // Promote only once the drag is ours, so a plain scroll never pays
            // for a compositor layer it will not use.
            if (dragAxis.current === "inline") markDragging(true);
          }
          if (dragAxis.current !== "inline") return;
          void event;
          writeOffset(Math.max(0, dismissLeftward.current ? -dx : dx));
        },
        onPointerUp: (event: ReactPointerEvent<HTMLDivElement>) => {
          if (dragDeclined.current) {
            dragDeclined.current = false;
            return;
          }
          const claimed = dragAxis.current === "inline";
          dragAxis.current = "undecided";
          if (!claimed) return;
          void event;
        },
        onPointerCancel: () => {
          if (dragDeclined.current) {
            dragDeclined.current = false;
            return;
          }
          dragAxis.current = "undecided";
          handlersFor().cancel();
        },
      }
    : {};

  const strings = useResizeStrings();
  const panelName = resizable?.panelName ?? desktopLabel ?? mobileLabel;
  // The hook is always called — a conditional hook is not an option — but it
  // stays inert without a container to measure against.
  const inertContainer = useRef<HTMLElement | null>(null);
  const resize = useResizablePanel({
    containerRef: resizable?.containerRef ?? inertContainer,
    panelRef,
    axis: "inline",
    edge: "end",
    min: resizable?.min ?? 0,
    max: resizable?.max ?? 0,
    defaultSize: resizable?.defaultSize ?? 0,
    adjacentMin: resizable?.adjacentMin,
    step: resizable?.step,
    coarseStep: resizable?.coarseStep,
    snapPoints: resizable?.snapPoints,
    collapseBelow: resizable?.collapseBelow,
    storage: resizable?.storage,
    storageKey: resizable?.storageKey,
    panelName,
    label: strings.label(panelName),
    formatValueText: strings.valueText,
    onCommit: resizable?.onCommit,
    onCollapse: resizable?.onCollapse,
    disabled: !resizable || resizable.disabled,
  });

  // In drawer mode the panel is an overlay, so `mobileWidth` sizes it and the
  // uncovered remainder becomes the backdrop's tap-to-dismiss strip. Persistent
  // mode keeps the resize/`width` sizing; the two never apply at once.
  const style: CSSProperties =
    isDialogOpen && mobileWidth
      ? { inlineSize: mobileWidth }
      : resizable
        ? { ...resize.panelProps.style }
        : width
          ? { width }
          : {};

  return (
    <>
      <StyleSheet
        name="rcl-sidebar-shell-2-5-3"
        css={sidebarShellStyles + sidebarShellResizeStyles}
      />
      {edgeOpenEnabled ? (
        <div
          data-testid={`${testId}-edge`}
          data-rcl-sidebar-edge=""
          data-edge={dismiss.anchorEdge}
          role="presentation"
          style={{ inlineSize: edgeSwipeWidth }}
          {...edgeProps}
        />
      ) : null}
      {isDialogOpen ? (
        <button
          type="button"
          data-testid={`${testId}-backdrop`}
          data-rcl-sidebar-backdrop=""
          data-mode={mode}
          aria-label={closeLabel}
          className={backdropClassName}
          onClick={onMobileClose}
        />
      ) : null}
      <div
        ref={setPanelRef}
        {...(resizable ? { id: resize.panelProps.id } : {})}
        data-testid={testId}
        data-rcl-sidebar-shell=""
        data-mode={mode}
        // The resolved physical side. The stylesheet keys its positioning and
        // its closed transform off this one value, so the panel can no longer
        // rest on one edge while animating out through the other.
        data-edge={dismiss.anchorEdge}
        data-open={mobileOpen ? "true" : "false"}
        data-swipe={isDialogOpen && swipeEnabled ? "true" : undefined}
        {...(isDialogOpen ? dragProps : {})}
        data-collapsed={resizable && resize.isCollapsed ? "true" : "false"}
        role={isDialogOpen ? "dialog" : "complementary"}
        aria-modal={isDialogOpen ? "true" : undefined}
        aria-label={isDialogOpen ? mobileLabel : (desktopLabel ?? mobileLabel)}
        style={style}
        className={cn(className, panelClassName)}
      >
        {!isPersistent && mobileChrome === "shell" && (
          <div className="rcl-sidebar-shell__header">
            <div className="rcl-sidebar-shell__header-content">{mobileHeader}</div>
            <button
              type="button"
              data-testid={`${testId}-close`}
              aria-label={closeLabel}
              onClick={onMobileClose}
              className="rcl-sidebar-shell__close"
            >
              <X aria-hidden className="rcl-sidebar-shell__icon" />
            </button>
          </div>
        )}
        <div className={cn("rcl-sidebar-shell__content", contentClassName)}>{children}</div>
        {resizable ? (
          <ResizeHandle separatorProps={resize.separatorProps} testId={`${testId}-resize-handle`} />
        ) : resizeHandleProps ? (
          <div
            data-testid={`${testId}-resize-handle`}
            {...resizeHandleProps}
            className={cn("rcl-sidebar-shell__resize", resizeHandleProps.className)}
          />
        ) : null}
      </div>
    </>
  );
});
