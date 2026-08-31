/**
 * @libraryId react-component-library:Resizable
 * @displayName Resizable
 * @description A bounded resizable region on either logical axis, with keyboard operation, snapping, a collapse threshold and persisted size.
 * @version 2.0.3
 * @tags ["layout","interaction","token-bound"]
 * @deps {"react":"^18","react-component-library:useResizablePanel":"^1.0.0","react-component-library:ResizeHandle":"^1.0.0"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:Resizable */
import { useRef, type ReactNode } from "react";

import {
  useResizablePanel,
  type ResizeAxis,
  type ResizeEdge,
  type ResizeStorage,
} from "@vrooli/react-component-library/useResizablePanel/1";
import {
  ResizeHandle,
  useResizeStrings,
} from "@vrooli/react-component-library/ResizeHandle/1";

export interface ResizableProps {
  /** The region whose size the user controls. */
  children?: ReactNode;
  /** The region that takes the remaining space. */
  adjacent?: ReactNode;
  axis?: ResizeAxis;
  edge?: ResizeEdge;
  min: number;
  max: number;
  defaultSize: number;
  adjacentMin?: number;
  step?: number;
  coarseStep?: number;
  snapPoints?: readonly number[];
  collapseBelow?: number;
  storage?: ResizeStorage;
  storageKey?: string;
  /** Names the region in the accessible label and value text. */
  panelName?: string;
  disabled?: boolean;
  onCommit?: (size: number) => void;
  onCollapse?: (collapsed: boolean) => void;
  className?: string;
  panelClassName?: string;
  adjacentClassName?: string;
  testId?: string;
}

export function Resizable({
  children,
  adjacent,
  axis = "inline",
  edge = "end",
  min,
  max,
  defaultSize,
  adjacentMin,
  step,
  coarseStep,
  snapPoints,
  collapseBelow,
  storage,
  storageKey,
  panelName = "Panel",
  disabled,
  onCommit,
  onCollapse,
  className,
  panelClassName,
  adjacentClassName,
  testId = "manipulation.resizable",
}: ResizableProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const strings = useResizeStrings();
  const { separatorProps, panelProps, isCollapsed } = useResizablePanel({
    containerRef,
    panelRef,
    axis,
    edge,
    min,
    max,
    defaultSize,
    adjacentMin,
    step,
    coarseStep,
    snapPoints,
    collapseBelow,
    storage,
    storageKey,
    panelName,
    label: strings.label(panelName),
    formatValueText: strings.valueText,
    onCommit,
    onCollapse,
    disabled,
  });

  return (
    <div
      ref={containerRef}
      data-testid={testId}
      data-rcl-resizable=""
      data-orientation={axis === "inline" ? "horizontal" : "vertical"}
      data-axis={axis}
      data-collapsed={isCollapsed ? "true" : "false"}
      className={className}
      style={{
        display: "flex",
        flexDirection: axis === "inline" ? "row" : "column",
        minInlineSize: 0,
        minBlockSize: 0,
      }}
    >
      <div
        ref={panelRef}
        {...panelProps}
        className={panelClassName}
        style={{
          ...panelProps.style,
          position: "relative",
          flex: "0 0 auto",
          minInlineSize: 0,
          minBlockSize: 0,
        }}
      >
        {children}
        <ResizeHandle
          separatorProps={separatorProps}
          testId={`${testId}-handle`}
        />
      </div>
      <div
        data-rcl-resizable-adjacent=""
        className={adjacentClassName}
        style={{ flex: "1 1 auto", minInlineSize: 0, minBlockSize: 0 }}
      >
        {adjacent}
      </div>
    </div>
  );
}
