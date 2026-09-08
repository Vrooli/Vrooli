import { ChatAccountProvider } from "../features/chat/ChatAccount";
import { ContextRetentionProvider, ContextRetentionActions } from "../features/companion/ContextRetention";
import { DesktopRecoveryPanel } from "../features/surfaces/DesktopSessionPanel";
import { DesktopSessionsProvider, useDesktopSession } from "../features/surfaces/useDesktopSession";
import { AgentTasksProvider } from "../features/chat/useAgentTask";
import { CompanionPresentation, CompanionToolbar } from "../features/companion/CompanionPresentation";
import { Outlet } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { BottomNav } from "./BottomNav";
import { Sidebar } from "./Sidebar";
import { TopBar } from "./TopBar";

/**
 * Responsive app shell. CSS grid with a header row and a (sidebar | content)
 * main row; mobile collapses the sidebar to a pinned bottom nav.
 *
 * This is the real shell — replaces the centered single-card placeholder.
 * Page content renders into the `<Outlet />`; routes are configured in
 * `app/routes.tsx`.
 */
export function AppShell() {
  const { t } = useTranslation();

  return (
    <CompanionPresentation><DesktopSessionsProvider><RetainedContextShell><div
      data-testid={selectors.layout.shell}
      className="flex min-h-full flex-col bg-app-background text-app-foreground"
    >
      <CompanionToolbar />
      <div className="companion-chrome"><TopBar /></div>
      <div className="flex min-h-0 flex-1">
        <div className="companion-chrome"><Sidebar /></div>
        <main
          data-testid={selectors.layout.main}
          aria-label={t(strings.layout.mainLabel)}
          className="companion-content min-w-0 flex-1 overflow-auto p-6"
        >
          <DesktopRecoveryPanel />
          <ContextRetentionActions />
          <Outlet />
        </main>
      </div>
      <div className="companion-chrome"><BottomNav /></div>
    </div></RetainedContextShell></DesktopSessionsProvider></CompanionPresentation>
  );
}

function RetainedContextShell({children}:{children:React.ReactNode}){const {account}=useDesktopSession();return <ChatAccountProvider account={account}><AgentTasksProvider><ContextRetentionProvider account={account}>{children}</ContextRetentionProvider></AgentTasksProvider></ChatAccountProvider>;}
