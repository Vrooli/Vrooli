/**
 * AppShell — full-width operational shell.
 *
 * Desktop (≥ md): resizable left sidebar + main content. The sidebar is
 * driven by `useResizablePanel` so widths persist across reloads.
 * Mobile (< md): top header, bottom navigation, and a slide-in drawer
 * that hosts the same nav + component list. The dark/light theme is mirrored
 * on `<html>` by the surrounding `<ThemeProvider>`.
 *
 * Replaces the starter centered-card layout: no `max-w-xl`, no eyebrow,
 * no card wrapping page-level content.
 */
import { type ReactNode, useCallback, useRef, useState } from "react";
import { Outlet } from "react-router-dom";

import { useIsMobile } from "../hooks/useMediaQuery";
import { useResizablePanel } from "../hooks/useResizablePanel";
import { HealthPill } from "./HealthPill";
import { MobileDrawer } from "./MobileDrawer";
import { MobileHeader } from "./MobileHeader";
import { MobileNav } from "./MobileNav";
import { Sidebar } from "./Sidebar";
import { SidebarComponentList } from "./SidebarComponentList";
import { ThemeToggle } from "./ThemeToggle";

const SIDEBAR_STORAGE = "react-component-library.sidebar.width.v1";

interface Props {
  children?: ReactNode;
}

export function AppShell({ children }: Props) {
  const shellRef = useRef<HTMLDivElement>(null);
  const sidebarRef = useRef<HTMLElement>(null);
  const isMobile = useIsMobile();
  const [drawerOpen, setDrawerOpen] = useState(false);

  const { size: sidebarWidth, resizeHandleProps } = useResizablePanel({
    containerRef: shellRef,
    targetRef: sidebarRef,
    minSize: 260,
    maxSize: 480,
    defaultSize: 300,
    adjacentMinSize: 420,
    handleWidth: 6,
    storageKey: SIDEBAR_STORAGE,
  });

  const closeDrawer = useCallback(() => setDrawerOpen(false), []);
  const openDrawer = useCallback(() => setDrawerOpen(true), []);

  const headerSlot = (
    <div className="flex items-center gap-1">
      <HealthPill />
      <ThemeToggle />
    </div>
  );

  return (
    <div
      ref={shellRef}
      data-testid="app-shell"
      className="flex min-h-screen w-full bg-app-background text-app-foreground"
    >
      <Sidebar
        ref={sidebarRef}
        width={isMobile ? undefined : sidebarWidth}
        resizeHandleProps={isMobile ? undefined : resizeHandleProps}
        headerSlot={headerSlot}
        inventorySlot={<SidebarComponentList />}
      />

      <div className="flex min-w-0 flex-1 flex-col">
        <MobileHeader onOpenDrawer={openDrawer} />
        <main
          data-testid="app-main"
          className="pb-safe min-w-0 flex-1 px-4 py-4 pb-20 md:px-8 md:py-6 md:pb-8"
        >
          {children ?? <Outlet />}
        </main>
        <MobileNav />
      </div>

      <MobileDrawer open={drawerOpen} onClose={closeDrawer}>
        <SidebarComponentList onNavigate={closeDrawer} />
      </MobileDrawer>
    </div>
  );
}
