import { Outlet } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { TopBar } from "./TopBar";
import { Sidebar } from "./Sidebar";
import { BottomNav } from "./BottomNav";

/**
 * Responsive app shell. A header row over a (sidebar | content) main row;
 * mobile collapses the sidebar to a pinned bottom nav. The `<main>` is a flex
 * column with no intrinsic padding so the signature full-screen Transfer split
 * (Receive top / Send bottom) can fill it edge-to-edge; secondary pages add
 * their own padding.
 */
export function AppShell() {
  const { t } = useTranslation();
  return (
    <div
      data-testid={selectors.layout.shell}
      className="flex h-screen flex-col bg-app-background text-app-foreground"
    >
      <TopBar />
      <div className="flex min-h-0 flex-1">
        <Sidebar />
        <main
          data-testid={selectors.layout.main}
          aria-label={t(strings.layout.mainLabel)}
          className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden"
        >
          <Outlet />
        </main>
      </div>
      <BottomNav />
    </div>
  );
}
