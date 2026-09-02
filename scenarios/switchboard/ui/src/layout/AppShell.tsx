import { Outlet } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { BottomNav } from "./BottomNav";
import { Sidebar } from "./Sidebar";
import { TopBar } from "./TopBar";

/**
 * Responsive app shell: a fixed-height column (top bar, then sidebar beside a
 * scrolling main) so pages that need a pinned composer can fill the viewport,
 * and ordinary pages simply scroll inside `<main>`. Below `md` the sidebar
 * yields to a pinned bottom nav and the main area reserves room for it.
 */
export function AppShell() {
  const { t } = useTranslation();
  return (
    <div data-testid={selectors.layout.shell} className="flex h-dvh flex-col bg-app-background text-app-foreground">
      <TopBar />
      <div className="flex min-h-0 flex-1">
        <Sidebar />
        <main
          data-testid={selectors.layout.main}
          aria-label={t(strings.console.mainContent)}
          className="min-w-0 flex-1 overflow-auto px-4 pb-[calc(6rem+var(--safe-area-inset-bottom))] pt-4 md:px-8 md:py-6 md:pb-6"
        >
          <div className="mx-auto h-full w-full max-w-[1400px]">
            <Outlet />
          </div>
        </main>
      </div>
      <BottomNav />
    </div>
  );
}
