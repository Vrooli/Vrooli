/**
 * @libraryId react-component-library:ResizableSidebar
 * @displayName ResizableSidebar
 * @description A sidebar preset over the shared resize primitive: navigation-shaped defaults, pointer and keyboard resizing, persisted width.
 * @version 2.0.0
 * @tags ["navigation","responsive","token-bound","accessibility"]
 * @deps {"react":"^18","react-component-library:Resizable":"^2.0.0"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 *
 * 1.0.3 was a placeholder `<aside>` with hardcoded min and max inline sizes and
 * no resize behavior at all. This version is deliberately thin: it is a preset,
 * not a second shell. A sidebar that also needs drawer behavior should use
 * SidebarShell, which owns both.
 */
/** @vrooliComponentSource react-component-library:ResizableSidebar */
import type { ReactNode } from "react";

import { Resizable } from "@vrooli/react-component-library/Resizable/2.0.0";
import type { ResizeStorage } from "@vrooli/react-component-library/useResizablePanel/1.0.0";

export interface ResizableSidebarProps {
  children?: ReactNode;
  /** The region beside the sidebar. */
  content?: ReactNode;
  label?: string;
  min?: number;
  max?: number;
  defaultSize?: number;
  /** Space the content region must keep. */
  contentMin?: number;
  collapseBelow?: number;
  storage?: ResizeStorage;
  storageKey?: string;
  disabled?: boolean;
  onCommit?: (size: number) => void;
  className?: string;
  testId?: string;
}

export function ResizableSidebar({
  children,
  content,
  label = "Sidebar",
  min = 260,
  max = 480,
  defaultSize = 320,
  contentMin = 320,
  collapseBelow,
  storage,
  storageKey,
  disabled,
  onCommit,
  className,
  testId = "navigation.resizable-sidebar",
}: ResizableSidebarProps) {
  return (
    <Resizable
      axis="inline"
      edge="end"
      min={min}
      max={max}
      defaultSize={defaultSize}
      adjacentMin={contentMin}
      collapseBelow={collapseBelow}
      storage={storage}
      storageKey={storageKey}
      panelName={label}
      disabled={disabled}
      onCommit={onCommit}
      className={className}
      testId={testId}
      adjacent={content}
    >
      <aside aria-label={label} data-resizable-sidebar style={{ blockSize: "100%" }}>
        {children}
      </aside>
    </Resizable>
  );
}
