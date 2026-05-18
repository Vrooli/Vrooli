import { useEffect, useState, type ReactNode } from "react";
import { Outlet, useLocation, useNavigate } from "react-router-dom";
import { selectors } from "../../consts/selectors";
import { useIsMobile, useAppViewport, useGlobalKeydown } from "../hooks";
import { ROUTES } from "../../routes.generated";
import { Sidebar } from "./Sidebar";
import { TopHeader } from "./TopHeader";
import { MobileBottomNav } from "./MobileBottomNav";
import { Sheet } from "../ui/primitives/Sheet";
import { ErrorBoundary } from "../ui/composites/ErrorBoundary";
import { cn } from "../lib/utils";

/**
 * Application shell.
 *
 * Desktop: persistent sidebar + sticky top header + scrollable outlet.
 * Mobile: sticky top header + scrollable outlet + fixed bottom nav + a
 * hamburger sheet revealing the same nav targets as the desktop sidebar.
 *
 * All shortcuts route through `useGlobalKeydown` per react-coherence §0.5.
 */
export function AppShell(): ReactNode {
  const isMobile = useIsMobile();
  useAppViewport();
  const navigate = useNavigate();
  const location = useLocation();
  const [mobileNavOpen, setMobileNavOpen] = useState(false);

  // Close the mobile sheet whenever the route changes.
  useEffect(() => {
    setMobileNavOpen(false);
  }, [location.pathname]);

  // Global keyboard shortcuts (chord-based). One central registration per
  // react-coherence §0.5; surfaces do NOT register their own document
  // keydown listeners.
  useGlobalKeydown((sequence, event) => {
    // Ignore modified keys for navigation chords (cmd/ctrl/alt should not
    // hijack browser behavior).
    if (event.metaKey || event.ctrlKey || event.altKey) return false;
    switch (sequence) {
      case "g g":
        void navigate(ROUTES.goldensIndex);
        return true;
      case "g s":
        void navigate(ROUTES.skillsIndex);
        return true;
      case "g m":
        void navigate(ROUTES.manifestsIndex);
        return true;
      case "g .":
        void navigate(ROUTES.settings);
        return true;
      default:
        return false;
    }
  });

  return (
    <div
      data-testid={selectors.nav.appShell}
      data-viewport={isMobile ? "mobile" : "desktop"}
      className="flex h-[var(--app-height,100vh)] w-full overflow-hidden bg-app-background text-app-foreground"
    >
      {/* Desktop sidebar */}
      {isMobile ? null : <Sidebar />}

      <div className="flex min-w-0 flex-1 flex-col">
        <TopHeader onMenuToggle={isMobile ? () => setMobileNavOpen(true) : undefined} />
        <main
          className={cn(
            "flex-1 overflow-auto px-4 py-4 sm:px-6",
            isMobile ? "pb-24" : "pb-6",
          )}
        >
          <ErrorBoundary>
            <Outlet />
          </ErrorBoundary>
        </main>
      </div>

      {/* Mobile bottom nav */}
      {isMobile ? <MobileBottomNav /> : null}

      {/* Mobile hamburger sheet — duplicates the sidebar nav for full-label
          access (the bottom nav uses icons only above the safe-area). */}
      {isMobile ? (
        <Sheet open={mobileNavOpen} onOpenChange={setMobileNavOpen} side="left">
          <div className="-m-6">
            <Sidebar />
          </div>
        </Sheet>
      ) : null}
    </div>
  );
}
