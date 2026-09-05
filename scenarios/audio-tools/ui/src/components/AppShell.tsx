import { useState } from "react";
import { Outlet } from "react-router-dom";
import { TopBar } from "./shell/TopBar";
import { Sidebar } from "./shell/Sidebar";
import { MobileNav } from "./shell/MobileNav";
import { SettingsDrawer } from "./shell/SettingsDrawer";
import { useKeyboardShortcuts } from "../hooks/useKeyboardShortcuts";
import { useTranslation } from "../i18n";
import { strings } from "../consts/strings";

/**
 * Application shell. Full-bleed layout (no centered card) with a sticky top
 * bar, persistent sidebar on desktop, and a bottom tab nav on mobile. Pages
 * render through React Router's `<Outlet />`.
 */
export function AppShell() {
  const { t } = useTranslation();
  const [settingsOpen, setSettingsOpen] = useState(false);

  useKeyboardShortcuts([
    {
      key: ",",
      ctrlOrMeta: true,
      handler: () => setSettingsOpen((prev) => !prev),
      allowInInputs: true,
    },
    {
      key: "Escape",
      handler: () => setSettingsOpen(false),
      allowInInputs: true,
    },
  ]);

  return (
    <div className="flex min-h-full flex-col bg-app-background text-app-foreground">
      <TopBar onOpenSettings={() => setSettingsOpen(true)} />
      <div className="flex flex-1">
        <Sidebar />
        <main
          id="main"
          className="flex min-w-0 flex-1 flex-col"
          aria-label={t(strings.shell.mainContent)}
        >
          <div className="flex-1 overflow-x-hidden p-4 md:p-6">
            <Outlet />
          </div>
        </main>
      </div>
      <MobileNav />
      <SettingsDrawer open={settingsOpen} onClose={() => setSettingsOpen(false)} />
    </div>
  );
}

export default AppShell;
