/**
 * SettingsDrawer - Gear icon opens a drawer with Settings and Prompts tabs.
 *
 * Wraps existing SettingsPage and PromptsPage content in a floating panel.
 */

import { lazy, Suspense, useState } from "react";
import { FloatingPanel } from "../../../components/ui/floating-panel";
import { cn } from "../../../lib/utils";

const SettingsPage = lazy(() =>
  import("../../../pages/SettingsPage").then((m) => ({ default: m.SettingsPage })),
);
const PromptsPage = lazy(() =>
  import("../../../pages/PromptsPage").then((m) => ({ default: m.PromptsPage })),
);

type DrawerTab = "settings" | "prompts";

interface SettingsDrawerProps {
  isOpen: boolean;
  onClose: () => void;
}

export function SettingsDrawer({ isOpen, onClose }: SettingsDrawerProps) {
  const [activeTab, setActiveTab] = useState<DrawerTab>("settings");

  return (
    <FloatingPanel
      isOpen={isOpen}
      onClose={onClose}
      title="Settings"
      testId="settings-drawer"
    >
      {/* Tab bar */}
      <div className="flex border-b border-slate-700/50 -mx-4 px-4 mb-4">
        {(["settings", "prompts"] as const).map((tab) => (
          <button
            key={tab}
            type="button"
            onClick={() => setActiveTab(tab)}
            className={cn(
              "px-4 py-2 text-sm font-medium border-b-2 transition-colors",
              activeTab === tab
                ? "border-cyan-400 text-cyan-400"
                : "border-transparent text-slate-400 hover:text-slate-200",
            )}
            data-testid={`settings-drawer-tab-${tab}`}
          >
            {tab === "settings" ? "Settings" : "Prompts"}
          </button>
        ))}
      </div>

      {/* Tab content */}
      <Suspense
        fallback={
          <div className="flex items-center justify-center py-12 text-sm text-slate-400">
            Loading...
          </div>
        }
      >
        <div className={activeTab === "settings" ? "block" : "hidden"}>
          <SettingsPage />
        </div>
        <div className={activeTab === "prompts" ? "block" : "hidden"}>
          <PromptsPage />
        </div>
      </Suspense>
    </FloatingPanel>
  );
}
