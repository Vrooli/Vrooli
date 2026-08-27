import { useRef } from "react";

import { SidebarShell } from "./SidebarShell";

const shell = {
  display: "flex",
  inlineSize: "min(100%, 640px)",
  blockSize: 280,
  border: "var(--border-hairline, 1px) solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, .75rem)",
  background: "var(--color-surface-muted, #f1f5f9)",
  overflow: "hidden",
  position: "relative" as const,
};

const nav = (
  <nav aria-label="Sections" style={{ display: "grid", gap: "var(--space-2xs, 8px)", padding: "var(--space-sm, 16px)" }}>
    <a href="#catalog">Catalog</a>
    <a href="#coverage">Coverage</a>
    <a href="#settings">Settings</a>
  </nav>
);

function Rig({
  mode = "persistent",
  mobileOpen = false,
  legacy = false,
}: {
  mode?: "persistent" | "responsive" | "overlay";
  mobileOpen?: boolean;
  legacy?: boolean;
}) {
  const containerRef = useRef<HTMLDivElement>(null);
  return (
    <div ref={containerRef} style={shell}>
      <SidebarShell
        mode={mode}
        mobileOpen={mobileOpen}
        onMobileClose={() => undefined}
        mobileLabel="Navigation drawer"
        desktopLabel="Navigation"
        closeLabel="Close navigation"
        mobileHeader={<span>Component Library</span>}
        {...(legacy
          ? { width: 240, resizeHandleProps: { className: "legacy" } }
          : {
              resizable: {
                containerRef,
                min: 160,
                max: 380,
                defaultSize: 240,
                adjacentMin: 120,
                panelName: "Navigation",
              },
            })}
      >
        {nav}
      </SidebarShell>
      <main style={{ flex: "1 1 auto", minInlineSize: 0, padding: "var(--space-sm, 16px)" }}>
        <p style={{ margin: 0 }}>Workspace</p>
      </main>
    </div>
  );
}

export function Default() {
  return <Rig />;
}

export function KeyboardResize() {
  return <Rig />;
}

export function DrawerOpen() {
  return <Rig mode="responsive" mobileOpen />;
}

export function LegacyHandle() {
  return <Rig legacy />;
}
