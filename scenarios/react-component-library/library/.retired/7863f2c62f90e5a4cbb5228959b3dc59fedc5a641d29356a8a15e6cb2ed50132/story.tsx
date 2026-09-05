import { useRef, type ReactNode } from "react";

import { useResizablePanel } from "@vrooli/react-component-library/useResizablePanel/1.0.0";
import { ResizeHandle, useResizeStrings } from "./ResizeHandle";

const shell = {
  display: "flex",
  inlineSize: "min(100%, 640px)",
  blockSize: 260,
  minInlineSize: 0,
  border: "var(--border-hairline, 1px) solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, 0.5rem)",
  background: "var(--color-surface-muted, #f1f5f9)",
  overflow: "hidden",
};

const panelSurface = {
  position: "relative" as const,
  flex: "0 0 auto",
  minInlineSize: 0,
  padding: "var(--space-sm, 16px)",
  background: "var(--color-surface, #ffffff)",
  color: "var(--color-foreground, #0f172a)",
};

const region = {
  flex: "1 1 auto",
  minInlineSize: 0,
  padding: "var(--space-sm, 16px)",
  color: "var(--color-muted-foreground, #64748b)",
  font: "var(--text-body, 400 var(--text-body-size) / var(--text-body-line) var(--font-sans))",
};

interface RigProps {
  axis?: "inline" | "block";
  edge?: "start" | "end";
  min?: number;
  max?: number;
  defaultSize?: number;
  snapPoints?: readonly number[];
  collapseBelow?: number;
  panelName?: string;
  dir?: "ltr" | "rtl";
  children?: ReactNode;
}

function Rig({
  axis = "inline",
  edge = "end",
  min = 160,
  max = 380,
  defaultSize = 240,
  snapPoints,
  collapseBelow,
  panelName = "Sessions",
  dir,
}: RigProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const strings = useResizeStrings();
  const { separatorProps, panelProps, size, isCollapsed } = useResizablePanel({
    containerRef,
    panelRef,
    axis,
    edge,
    min,
    max,
    defaultSize,
    snapPoints,
    collapseBelow,
    panelName,
    label: strings.label(panelName),
    formatValueText: strings.valueText,
  });

  return (
    <div
      ref={containerRef}
      dir={dir}
      style={{ ...shell, flexDirection: axis === "inline" ? "row" : "column" }}
    >
      <div
        ref={panelRef}
        {...panelProps}
        style={{ ...panelSurface, ...panelProps.style }}
      >
        <p
          style={{
            margin: 0,
            font: "var(--text-title, 700 var(--text-title-size) / var(--text-title-line) var(--font-sans))",
          }}
        >
          {panelName}
        </p>
        <p
          style={{
            margin: "var(--space-2xs, 8px) 0 0",
            font: "var(--text-body, 400 var(--text-body-size) / var(--text-body-line) var(--font-sans))",
          }}
        >
          {isCollapsed ? "Collapsed" : `${size} pixels`}
        </p>
        <ResizeHandle separatorProps={separatorProps} />
      </div>
      <div style={region}>
        <p style={{ margin: 0 }}>Workspace</p>
      </div>
    </div>
  );
}

export function Default() {
  return <Rig />;
}

export function AxisInline() {
  return <Rig axis="inline" />;
}

export function AxisBlock() {
  return (
    <Rig axis="block" min={80} max={200} defaultSize={120} panelName="Output" />
  );
}

export function Keyboard() {
  return <Rig />;
}

export function AtMinimum() {
  return <Rig />;
}

export function Snapped() {
  return <Rig snapPoints={[200, 300]} />;
}

export function Collapsed() {
  return <Rig collapseBelow={180} />;
}

export function Rtl() {
  return <Rig dir="rtl" />;
}

export function ForcedColors() {
  return <Rig />;
}
