/**
 * @libraryId react-component-library:SplitPane
 * @displayName SplitPane
 * @version 2.0.5
 * @tags ["layout","interaction","token-bound"]
 * @deps {"react":"^18","react-component-library:Resizable":"^2.0.0"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:SplitPane */
import type { ReactNode } from "react";

import { Resizable } from "@vrooli/react-component-library/Resizable/2";
import type { ResizeStorage } from "@vrooli/react-component-library/useResizablePanel/1";

export interface SplitPaneProps {
  primary?: ReactNode;
  secondary?: ReactNode;
  /** `horizontal` places the panes side by side, `vertical` stacks them. */
  orientation?: "horizontal" | "vertical";
  min?: number;
  max?: number;
  defaultSize?: number;
  secondaryMin?: number;
  snapPoints?: readonly number[];
  collapseBelow?: number;
  storage?: ResizeStorage;
  storageKey?: string;
  primaryName?: string;
  disabled?: boolean;
  onCommit?: (size: number) => void;
  className?: string;
  testId?: string;
}

export function SplitPane({
  primary,
  secondary,
  orientation = "horizontal",
  min = 160,
  max = 960,
  defaultSize = 320,
  secondaryMin = 160,
  snapPoints,
  collapseBelow,
  storage,
  storageKey,
  primaryName = "Primary pane",
  disabled,
  onCommit,
  className,
  testId = "manipulation.split-pane",
}: SplitPaneProps) {
  return (
    <Resizable
      axis={orientation === "horizontal" ? "inline" : "block"}
      edge="end"
      min={min}
      max={max}
      defaultSize={defaultSize}
      adjacentMin={secondaryMin}
      snapPoints={snapPoints}
      collapseBelow={collapseBelow}
      storage={storage}
      storageKey={storageKey}
      panelName={primaryName}
      disabled={disabled}
      onCommit={onCommit}
      className={className}
      testId={testId}
      adjacent={<section data-rcl-split-pane-secondary="">{secondary}</section>}
    >
      <section data-rcl-split-pane-primary="">{primary}</section>
    </Resizable>
  );
}
