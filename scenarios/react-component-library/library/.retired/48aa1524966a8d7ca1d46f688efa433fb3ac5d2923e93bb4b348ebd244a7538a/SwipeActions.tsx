/**
 * @libraryId react-component-library:SwipeActions
 * @displayName Swipe Actions
 * @description Reveals row actions behind any content with a staged, handedness-aware drag that composes inside a swipe-to-close drawer.
 * @version 1.0.3
 * @tags ["gesture","list","mobile","accessibility"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource patterns.swipe-actions */
import { useCallback, useEffect, useId, useRef, type ReactNode } from "react";

import { cn } from "@vrooli/react-component-library/ClassMerge/1.0.2";
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import { useControllableState } from "@vrooli/react-component-library/useControllableState/1.0.0";
import { useGestureDirection } from "@vrooli/react-component-library/useHandedness/1.1.2";
import { useSwipeGesture } from "@vrooli/react-component-library/useSwipeGesture/1.0.0";
import { swipeActionsStyles } from "./styles";

export type SwipeActionTone = "neutral" | "primary" | "destructive";

export interface SwipeAction {
  /** Stable identity, also used to build the action's test id. */
  id: string;
  /** Visible text. Also the accessible name when only an icon is shown. */
  label: string;
  icon?: ReactNode;
  tone?: SwipeActionTone;
  onSelect: () => void;
}

export interface SwipeActionsProps {
  /**
   * Ordered nearest-first: the first action arms soonest.
   *
   * The same array should feed the surface's menu. A swipe is an accelerator,
   * never the only route to an action — it is invisible, undiscoverable, and
   * unreachable by keyboard, switch, and screen-reader users. Two or three
   * entries is the practical ceiling before the track stops being readable.
   */
  actions: readonly SwipeAction[];
  children: ReactNode;
  /**
   * What a past-threshold release means. `rest` holds the row open so the user
   * can choose; `commit` fires the armed action immediately.
   *
   * Prefer `rest` when the actions are heterogeneous, when one of them is
   * destructive, or when only one direction is available because an ancestor
   * drawer owns the other.
   */
  releaseMode?: "rest" | "commit";
  /** Width of a single action, in pixels. Also the first stage's threshold. */
  actionWidth?: number;
  open?: boolean;
  defaultOpen?: boolean;
  onOpenChange?: (open: boolean) => void;
  disabled?: boolean;
  /** Accessible name for the revealed group. */
  label?: string;
  className?: string;
  contentClassName?: string;
  testId?: string;
}

const DEFAULT_ACTION_WIDTH = 76;
/** A stage arms a little before its slot is fully uncovered, so the commit point
 *  is reached where the action has become readable rather than pixel-exact. */
const ARM_RATIO = 0.62;

export function SwipeActions({
  actions,
  children,
  releaseMode = "rest",
  actionWidth = DEFAULT_ACTION_WIDTH,
  open,
  defaultOpen = false,
  onOpenChange,
  disabled = false,
  label,
  className,
  contentClassName,
  testId = "patterns.swipe-actions",
}: SwipeActionsProps) {
  const rootRef = useRef<HTMLDivElement>(null);
  const faceRef = useRef<HTMLDivElement>(null);
  const groupId = useId();
  const [isOpen, setIsOpen] = useControllableState({
    value: open,
    defaultValue: defaultOpen,
    onChange: onOpenChange,
  });

  // Reveal runs away from the edge an anchored drawer sits against, so a row
  // inside a swipe-to-close sidebar can never contend with it: the two live on
  // opposite signs of the same axis and the sign alone says which was meant.
  const { direction, sign } = useGestureDirection("reveal", { elementRef: rootRef });

  const enabled = !disabled && actions.length > 0;
  const travel = actions.length * actionWidth;
  const stages = actions.map((_, index) => (index + 1) * actionWidth * ARM_RATIO);

  const openRef = useRef(isOpen);
  openRef.current = isOpen;

  const paint = useCallback((translate: number, dragging: boolean) => {
    const face = faceRef.current;
    if (!face) return;
    face.dataset.dragging = dragging ? "true" : "false";
    face.dataset.settling = dragging ? "false" : "true";
    face.style.transform = translate === 0 ? "" : `translateX(${String(translate)}px)`;
  }, []);

  // Keep the rendered offset in step with `open` whenever it changes for a
  // reason other than the gesture — a controlled parent closing the row, or a
  // sibling row opening. Without this the face can be left stranded mid-travel
  // by an inline transform that no stylesheet rule can override.
  useEffect(() => {
    paint(isOpen ? travel * sign : 0, false);
  }, [isOpen, travel, sign, paint]);

  const close = useCallback(() => {
    if (openRef.current) setIsOpen(false);
    else paint(0, false);
  }, [setIsOpen, paint]);

  const { onPointerDown, cancel } = useSwipeGesture({
    direction,
    stages,
    releaseMode,
    disabled: !enabled,
    startOffset: () => (openRef.current ? travel : 0),
    onMove: ({ translate }) => {
      paint(translate, true);
    },
    onRelease: (release) => {
      if (release.outcome === "commit") {
        const armed = actions[Math.max(0, release.stage - 1)];
        paint(0, false);
        if (openRef.current) setIsOpen(false);
        armed?.onSelect();
        return;
      }
      if (release.outcome === "rest") {
        paint(travel * sign, false);
        if (!openRef.current) setIsOpen(true);
        return;
      }
      // "return" and "abort" both land closed. An abort must never perform the
      // armed action: a browser-initiated cancel is the user changing their
      // mind, or the platform taking the gesture away, not a confirmation.
      close();
    },
  });

  const runAction = (action: SwipeAction) => {
    close();
    action.onSelect();
  };

  return (
    <>
      <StyleSheet name="rcl-swipe-actions-1-0-0" css={swipeActionsStyles} />
      <div
        ref={rootRef}
        data-testid={testId}
        data-rcl-swipe-actions=""
        data-open={isOpen ? "true" : "false"}
        data-swipe={enabled ? "true" : undefined}
        data-reveal={direction}
        // The escape hatch a swipe-enabled SidebarShell publishes for children
        // that own the inline axis. Without it every ancestor rule pins this
        // subtree to pan-y and the browser cancels the drag as a scroll.
        {...(enabled ? { "data-rcl-pan-x": "" } : {})}
        className={cn(className)}
      >
        <div
          data-rcl-swipe-actions-track=""
          data-side={direction === "right" ? "left" : "right"}
          id={groupId}
          role="group"
          aria-label={label}
          aria-hidden={isOpen ? undefined : true}
          style={{ inlineSize: travel }}
        >
          {actions.map((action) => (
            <button
              key={action.id}
              type="button"
              data-testid={`${testId}.action.${action.id}`}
              data-rcl-swipe-action=""
              data-tone={action.tone ?? "neutral"}
              style={{ inlineSize: actionWidth }}
              tabIndex={isOpen ? 0 : -1}
              aria-label={action.label}
              onClick={() => {
                runAction(action);
              }}
              // A pointer landing on an action must not also start a drag on
              // the face beneath it.
              onPointerDown={(event) => {
                event.stopPropagation();
                cancel();
              }}
            >
              {action.icon}
              <span>{action.label}</span>
            </button>
          ))}
        </div>
        <div
          ref={faceRef}
          data-testid={`${testId}.face`}
          data-rcl-swipe-actions-face=""
          onPointerDown={enabled ? onPointerDown : undefined}
          aria-describedby={isOpen ? groupId : undefined}
          className={cn(contentClassName)}
        >
          {children}
        </div>
      </div>
    </>
  );
}

export default SwipeActions;
