/**
 * @libraryId react-component-library:SidebarShell
 * @displayName Sidebar Shell
 * @description Responsive sidebar parent that owns desktop resizing and mobile full-width safe-area drawer behavior.
 * @version 2.0.0
 * @tags ["layout","navigation","responsive"]
 * @deps {"react":"^18","lucide-react":"^0.424.0","react-component-library:useEscapeKey":"^1.0.0","react-component-library:useResizablePanel":"^1.0.0","react-component-library:ResizeHandle":"^1.0.0"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 *
 * Breaking changes from 1.3.2:
 *  - Test ids follow the library's dotted convention and derive from one
 *    `testId` prop, so an adopting scenario keeps its own selector registry
 *    without re-implementing the element. 1.3.x named two of its four parts
 *    dotted and two kebab.
 *  - `resizable` replaces `resizeHandleProps`. The shell composes
 *    useResizablePanel and ResizeHandle itself; the caller no longer supplies a
 *    hand-built handle, a width, or a handle-width constant. `resizeHandleProps`
 *    and `width` remain as a deprecated escape hatch for one release.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import {
  forwardRef,
  useCallback,
  useRef,
  type CSSProperties,
  type HTMLAttributes,
  type ReactNode,
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
  className?: string;
  panelClassName?: string;
  contentClassName?: string;
  backdropClassName?: string;
}

const cn = (...inputs: Array<string | undefined>) => inputs.filter(Boolean).join(" ");

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

  const style: CSSProperties = resizable
    ? { ...resize.panelProps.style }
    : width
      ? { width }
      : {};

  return (
    <>
      <StyleSheet name="rcl-sidebar-shell-2-0-0" css={sidebarShellStyles + sidebarShellResizeStyles} />
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
