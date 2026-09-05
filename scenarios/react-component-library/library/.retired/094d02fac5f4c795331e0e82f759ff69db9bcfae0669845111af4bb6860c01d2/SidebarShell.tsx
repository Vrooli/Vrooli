/**
 * @libraryId react-component-library:SidebarShell
 * @displayName Sidebar Shell
 * @description Responsive sidebar parent that owns desktop resizing and mobile full-width safe-area drawer behavior.
 * @version 2.4.0
 * @tags ["layout","navigation","responsive"]
 * @deps {"react":"^18","lucide-react":"^0.424.0","react-component-library:useEscapeKey":"^1.0.0","react-component-library:useResizablePanel":"^1.0.0","react-component-library:ResizeHandle":"^1.0.0","react-component-library:useSwipe":"^2.0.2"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
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
import { useEscapeKey } from "@vrooli/react-component-library/useEscapeKey/1.0.0";
import {
  useResizablePanel,
  type ResizeStorage,
} from "@vrooli/react-component-library/useResizablePanel/1.0.0";
import {
  ResizeHandle,
  useResizeStrings,
} from "@vrooli/react-component-library/ResizeHandle/1.0.0";
import { useSwipe } from "@vrooli/react-component-library/useSwipe/2.0.2";
import { sidebarShellStyles, sidebarShellResizeStyles } from "./styles";

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
  className?: string;
  panelClassName?: string;
  contentClassName?: string;
  backdropClassName?: string;
}

const cn = (...inputs: Array<string | undefined>) =>
  inputs.filter(Boolean).join(" ");

/** Movement, in px, before a drag is assigned to an axis. */
const AXIS_SLOP = 8;

/** Travel, in px, that commits an opening drag on release. */
const EDGE_OPEN_THRESHOLD = 64;

/** Travel per ms that commits an opening drag regardless of distance. */
const EDGE_OPEN_VELOCITY = 0.4;

function assignRef<T>(ref: Ref<T> | undefined, value: T | null) {
  if (typeof ref === "function") ref(value);
  else if (ref && typeof ref === "object")
    (ref as { current: T | null }).current = value;
}

export const SidebarShell = forwardRef<HTMLDivElement, SidebarShellProps>(
  function SidebarShell(
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
    const dismissTowardStart = useRef(true);

    const markDragging = useCallback((dragging: boolean) => {
      const panel = panelRef.current;
      if (panel) panel.dataset.dragging = dragging ? "true" : "false";
    }, []);

    const writeOffset = useCallback(
      (pixels: number) => {
        const panel = panelRef.current;
        if (panel) {
          const signed = dismissTowardStart.current ? -pixels : pixels;
          panel.style.transform =
            pixels > 0 ? `translateX(${String(signed)}px)` : "";
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
        const signed = dismissTowardStart.current
          ? -dragExtent.current
          : dragExtent.current;
        panel.style.transform = `translateX(${String(signed)}px)`;
      }
    }, [markDragging]);

    const swipe = useSwipe({
      // Resolved per gesture below; a start-edge drawer dismisses toward the
      // inline start, which is left in LTR and right in RTL.
      direction: "left",
      threshold: 96,
      velocity: 0.5,
      onCommit: () => {
        writeDismissed();
        onMobileClose();
      },
      onCancel: () => {
        writeOffset(0);
      },
    });

    const swipeRight = useSwipe({
      direction: "right",
      threshold: 96,
      velocity: 0.5,
      onCommit: () => {
        writeDismissed();
        onMobileClose();
      },
      onCancel: () => {
        writeOffset(0);
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
      const signed = dismissTowardStart.current ? -remaining : remaining;
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
        dismissTowardStart.current =
          getComputedStyle(panel).direction !== "rtl";
        // The panel is laid out while closed — hidden and translated away, not
        // removed — so it measures correctly before it has ever been shown.
        dragExtent.current =
          panel.getBoundingClientRect().width || panel.offsetWidth;
        edgeGesture.current = {
          originX: event.clientX,
          originY: event.clientY,
          startedAt: performance.now(),
          axis: "undecided",
        };

        const travelOf = (clientX: number) => {
          const dx = clientX - (edgeGesture.current?.originX ?? 0);
          return Math.max(0, dismissTowardStart.current ? dx : -dx);
        };

        const onMove = (moveEvent: PointerEvent) => {
          const gesture = edgeGesture.current;
          if (!gesture) return;
          const dx = moveEvent.clientX - gesture.originX;
          const dy = moveEvent.clientY - gesture.originY;
          if (gesture.axis === "undecided") {
            if (Math.abs(dx) < AXIS_SLOP && Math.abs(dy) < AXIS_SLOP) return;
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
            travelled >= EDGE_OPEN_THRESHOLD ||
            travelled / elapsed >= EDGE_OPEN_VELOCITY;
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
    const handlersFor = () => (dismissTowardStart.current ? swipe : swipeRight);

    const dragProps = swipeEnabled
      ? {
          onPointerDown: (event: ReactPointerEvent<HTMLDivElement>) => {
            const panel = panelRef.current;
            if (!panel) return;
            dismissTowardStart.current =
              getComputedStyle(panel).direction !== "rtl";
            dragOriginX.current = event.clientX;
            dragOriginY.current = event.clientY;
            dragAxis.current = "undecided";
            dragExtent.current = panel.getBoundingClientRect().width;
            armCloseSafetyNet();
            handlersFor().onPointerDown(event);
          },
          onPointerMove: (event: ReactPointerEvent<HTMLDivElement>) => {
            const dx = event.clientX - dragOriginX.current;
            const dy = event.clientY - dragOriginY.current;
            // Decide once, on the first movement that clears the slop radius,
            // which axis owns the gesture. Without this the drawer only dragged
            // where there was nothing to scroll: inside the nav list any
            // vertical component started a scroll, and that scroll cancelled the
            // pointer stream mid-gesture.
            if (dragAxis.current === "undecided") {
              if (Math.abs(dx) < AXIS_SLOP && Math.abs(dy) < AXIS_SLOP) return;
              dragAxis.current =
                Math.abs(dx) > Math.abs(dy) ? "inline" : "block";
              // Promote only once the drag is ours, so a plain scroll never pays
              // for a compositor layer it will not use.
              if (dragAxis.current === "inline") markDragging(true);
            }
            if (dragAxis.current !== "inline") return;
            handlersFor().onPointerMove(event);
            writeOffset(Math.max(0, dismissTowardStart.current ? -dx : dx));
          },
          onPointerUp: (event: ReactPointerEvent<HTMLDivElement>) => {
            const claimed = dragAxis.current === "inline";
            dragAxis.current = "undecided";
            if (!claimed) return;
            handlersFor().onPointerUp(event);
          },
          onPointerCancel: (event: ReactPointerEvent<HTMLDivElement>) => {
            dragAxis.current = "undecided";
            handlersFor().onPointerCancel(event);
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
          name="rcl-sidebar-shell-2-4-0"
          css={sidebarShellStyles + sidebarShellResizeStyles}
        />
        {edgeOpenEnabled ? (
          <div
            data-testid={`${testId}-edge`}
            data-rcl-sidebar-edge=""
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
          data-open={mobileOpen ? "true" : "false"}
          data-swipe={isDialogOpen && swipeEnabled ? "true" : undefined}
          {...(isDialogOpen ? dragProps : {})}
          data-collapsed={resizable && resize.isCollapsed ? "true" : "false"}
          role={isDialogOpen ? "dialog" : "complementary"}
          aria-modal={isDialogOpen ? "true" : undefined}
          aria-label={
            isDialogOpen ? mobileLabel : (desktopLabel ?? mobileLabel)
          }
          style={style}
          className={cn(className, panelClassName)}
        >
          {!isPersistent && mobileChrome === "shell" && (
            <div className="rcl-sidebar-shell__header">
              <div className="rcl-sidebar-shell__header-content">
                {mobileHeader}
              </div>
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
          <div className={cn("rcl-sidebar-shell__content", contentClassName)}>
            {children}
          </div>
          {resizable ? (
            <ResizeHandle
              separatorProps={resize.separatorProps}
              testId={`${testId}-resize-handle`}
            />
          ) : resizeHandleProps ? (
            <div
              data-testid={`${testId}-resize-handle`}
              {...resizeHandleProps}
              className={cn(
                "rcl-sidebar-shell__resize",
                resizeHandleProps.className,
              )}
            />
          ) : null}
        </div>
      </>
    );
  },
);
