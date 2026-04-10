import { LayoutGrid, LayoutList } from "lucide-react";
import { useWakeLockStatus } from "../../stores/useWakeLockStatus";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import { Button } from "../ui/button";
import { SettingsCard, SettingsRow, SettingsSectionIntro, SettingsToggle } from "./primitives";

const STATUS_HINTS: Record<string, string> = {
  active: "Screen is being kept awake.",
  off: "Prevent the device from dimming or locking while the console is open.",
  unsupported: "Your browser or connection doesn't support screen wake lock. A video-based fallback is being used instead.",
  denied: "Screen wake lock was denied — your device may be in power-save mode.",
  released: "Re-acquiring screen wake lock\u2026",
};

const STATUS_HINT_CLASSES: Record<string, string | undefined> = {
  active: "text-wc-accent",
  denied: "text-wc-error",
  released: "text-yellow-500",
};

const DEFAULT_HINT = "Prevent the device from dimming or locking while the console is open.";

export default function WorkspaceSection() {
  const isMinimapVisible = useWorkspaceStore((state) => state.isMinimapVisible);
  const setMinimapVisible = useWorkspaceStore((state) => state.setMinimapVisible);
  const displayMode = useWorkspaceStore((state) => state.displayMode);
  const setDisplayMode = useWorkspaceStore((state) => state.setDisplayMode);
  const toolbarLayout = useWorkspaceStore((state) => state.toolbarLayout);
  const setToolbarLayout = useWorkspaceStore((state) => state.setToolbarLayout);
  const keepScreenAwake = useWorkspaceStore((state) => state.keepScreenAwake);
  const setKeepScreenAwake = useWorkspaceStore((state) => state.setKeepScreenAwake);
  const wakeLockStatus = useWakeLockStatus((s) => s.status);

  const wakeLockHint = keepScreenAwake
    ? (STATUS_HINTS[wakeLockStatus] ?? DEFAULT_HINT)
    : DEFAULT_HINT;
  const wakeLockHintClass = keepScreenAwake
    ? STATUS_HINT_CLASSES[wakeLockStatus]
    : undefined;

  return (
    <div className="space-y-4">
      <SettingsSectionIntro
        eyebrow="Layout"
        title="Workspace behavior"
        description="Tune the overall terminal layout and how the mobile controls use limited screen space."
      />

      <SettingsCard className="space-y-4">
        <SettingsRow
          label="Pane layout"
          hint="Choose the main workspace presentation."
          control={(
            <div className="flex items-center gap-2">
              <Button
                data-testid="display-mode-grid"
                variant={displayMode === "grid" ? "default" : "outline"}
                size="sm"
                className="h-8 px-3"
                onClick={() => setDisplayMode("grid")}
              >
                <LayoutGrid className="mr-1 h-3.5 w-3.5" />
                Grid
              </Button>
              <Button
                data-testid="display-mode-tabs"
                variant={displayMode === "tabs" ? "default" : "outline"}
                size="sm"
                className="h-8 px-3"
                onClick={() => setDisplayMode("tabs")}
              >
                <LayoutList className="mr-1 h-3.5 w-3.5" />
                Tabs
              </Button>
            </div>
          )}
        />

        <SettingsRow
          label="Mobile toolbar"
          hint="Compact keeps a tighter footprint. Expanded shows more controls at once."
          control={(
            <div className="flex items-center gap-2">
              <Button
                data-testid="toolbar-layout-compact"
                variant={toolbarLayout === "compact" ? "default" : "outline"}
                size="sm"
                className="h-8 px-3"
                onClick={() => setToolbarLayout("compact")}
              >
                Compact
              </Button>
              <Button
                data-testid="toolbar-layout-expanded"
                variant={toolbarLayout === "expanded" ? "default" : "outline"}
                size="sm"
                className="h-8 px-3"
                onClick={() => setToolbarLayout("expanded")}
              >
                Expanded
              </Button>
            </div>
          )}
        />

        {displayMode === "grid" && (
          <SettingsRow
            label="Minimap"
            hint="Keep the grid overview visible while navigating larger layouts."
            control={(
              <SettingsToggle
                testId="minimap-toggle"
                checked={isMinimapVisible}
                onClick={() => setMinimapVisible(!isMinimapVisible)}
              />
            )}
          />
        )}
        <SettingsRow
          label="Keep screen awake"
          hint={wakeLockHint}
          hintClassName={wakeLockHintClass}
          control={(
            <SettingsToggle
              testId="keep-screen-awake-toggle"
              checked={keepScreenAwake}
              onClick={() => setKeepScreenAwake(!keepScreenAwake)}
            />
          )}
        />
      </SettingsCard>
    </div>
  );
}
