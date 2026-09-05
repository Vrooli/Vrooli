import type { ReactNode } from "react";

import type { AppRoute } from "../app/routeDefinitions";
import { selectors } from "../consts/selectors";
import { BottomNav } from "./BottomNav";
import { Sidebar } from "./Sidebar";
import { TopBar } from "./TopBar";

export function AppShell({
  activeRoute,
  children,
  onNavigate
}: {
  activeRoute: AppRoute;
  children: ReactNode;
  onNavigate: (routeKey: AppRoute) => void;
}) {
  return (
    <div
      className="flex min-h-full flex-col bg-transparent text-foreground"
      data-testid={selectors.layout.shell}
    >
      <TopBar />
      <div className="flex min-h-0 flex-1">
        <Sidebar activeRoute={activeRoute} onNavigate={onNavigate} />
        <main
          aria-label="Main content"
          className="min-w-0 flex-1 overflow-auto px-6 pb-24 pt-6 sm:px-10 md:pb-10"
          data-testid={selectors.layout.main}
        >
          <div className="flex flex-col gap-6">{children}</div>
        </main>
      </div>
      <BottomNav activeRoute={activeRoute} onNavigate={onNavigate} />
    </div>
  );
}
