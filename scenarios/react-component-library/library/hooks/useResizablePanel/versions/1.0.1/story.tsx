import { useMemo, useRef } from "react";

import { useResizablePanel, type ResizeStorage } from "./useResizablePanel";

const shell = {
  display: "flex",
  inlineSize: "min(100%, 560px)",
  blockSize: 200,
  border: "var(--border-hairline, 1px) solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, .75rem)",
  background: "var(--color-surface-muted, #f1f5f9)",
  overflow: "hidden",
};

const separator = {
  position: "absolute" as const,
  insetBlock: 0,
  insetInlineEnd: -6,
  inlineSize: 12,
  cursor: "col-resize",
  background: "transparent",
};

function Rig({ storage }: { storage?: ResizeStorage }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const { separatorProps, panelProps, size } = useResizablePanel({
    containerRef,
    panelRef,
    min: 160,
    max: 380,
    defaultSize: 240,
    panelName: "Sidebar",
    storage,
    storageKey: storage ? "story.sidebar.width" : undefined,
  });

  return (
    <div ref={containerRef} style={shell}>
      <div
        ref={panelRef}
        {...panelProps}
        style={{
          ...panelProps.style,
          position: "relative",
          flex: "0 0 auto",
          padding: "var(--space-sm, 16px)",
          background: "var(--color-surface, #fff)",
          color: "var(--color-foreground, #0f172a)",
        }}
      >
        <p style={{ margin: 0 }}>Sidebar</p>
        <p style={{ margin: "var(--space-2xs, 8px) 0 0" }}>{`${size} pixels`}</p>
        <div
          {...separatorProps}
          data-testid="hooks.use-resizable-panel"
          style={{ ...separator, ...separatorProps.style }}
        />
      </div>
      <div style={{ flex: "1 1 auto", padding: "var(--space-sm, 16px)" }}>
        <p style={{ margin: 0 }}>Workspace</p>
      </div>
    </div>
  );
}

export function Default() {
  return <Rig />;
}

export function Persisted() {
  const storage = useMemo<ResizeStorage>(() => {
    const values = new Map<string, string>([["story.sidebar.width", "300"]]);
    return {
      get: (key) => values.get(key) ?? null,
      set: (key, value) => {
        values.set(key, value);
      },
    };
  }, []);
  return <Rig storage={storage} />;
}
