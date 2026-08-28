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
  <nav
    aria-label="Sections"
    style={{ display: "grid", gap: "var(--space-2xs, 8px)", padding: "var(--space-sm, 16px)" }}
  >
    <a href="#catalog">Catalog</a>
    <a href="#coverage">Coverage</a>
    <a href="#settings">Settings</a>
  </nav>
);

function Rig({
  mode = "persistent",
  mobileOpen = false,
  legacy = false,
  mobileWidth,
  swipeToClose,
  edgeToOpen = false,
  side,
}: {
  mode?: "persistent" | "responsive" | "overlay";
  mobileOpen?: boolean;
  legacy?: boolean;
  mobileWidth?: string;
  swipeToClose?: boolean;
  edgeToOpen?: boolean;
  side?: "inline-start" | "inline-end";
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
        mobileWidth={mobileWidth}
        side={side}
        {...(edgeToOpen ? { onMobileOpen: () => undefined } : {})}
        {...(swipeToClose === undefined ? {} : { swipeToClose })}
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

/**
 * The drawer presentation. It is pinned to `overlay` rather than `responsive`
 * so the story asserts the same anatomy at any harness width: `responsive`
 * resolves to the persistent panel on a desktop-sized viewport, where the
 * dialog role, backdrop and close button this story is about do not exist.
 */
export function DrawerOpen() {
  return <Rig mode="overlay" mobileOpen />;
}

export function LegacyHandle() {
  return <Rig legacy />;
}

/**
 * The drawer accepts a dismissing drag and leaves a strip of backdrop beside
 * it. `mobileWidth` is what sizes that strip: the panel takes the width given,
 * and the uncovered remainder dismisses on tap.
 */
export function SwipeDrawer() {
  return <Rig mode="overlay" mobileOpen mobileWidth="min(18rem, calc(100% - 3rem))" />;
}

/** A drawer may opt out of the gesture and rely on its close button alone. */
export function SwipeDisabled() {
  return <Rig mode="overlay" mobileOpen swipeToClose={false} />;
}

/**
 * Closed, with the opening gesture armed. The strip along the screen edge is
 * the drag target; it exists only while the drawer is closed, so it never
 * competes with the panel it opens.
 */
export function EdgeSwipeToOpen() {
  return <Rig mode="overlay" edgeToOpen />;
}

/**
 * Anchored to the inline end. The panel sits against the opposite edge, its
 * seam moves with it, and the dismissing drag reverses — one resolved value
 * drives all three, so they cannot disagree.
 */
export function EndAnchored() {
  return (
    <Rig mode="overlay" mobileOpen side="inline-end" mobileWidth="min(18rem, calc(100% - 3rem))" />
  );
}

/**
 * Right-to-left, still anchored to the inline start — which is the right edge
 * in this locale. Through 2.4.0 the panel moved here while its closed
 * transform kept pushing left, into the viewport rather than out of it.
 */
export function RightToLeft() {
  return (
    <div dir="rtl">
      <Rig mode="overlay" mobileOpen mobileWidth="min(18rem, calc(100% - 3rem))" />
    </div>
  );
}
