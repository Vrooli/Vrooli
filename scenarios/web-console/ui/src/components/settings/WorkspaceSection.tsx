import { LayoutGrid, LayoutList } from "lucide-react";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import { Button } from "../ui/button";
import { SettingsCard, SettingsRow, SettingsSectionIntro, SettingsToggle } from "./primitives";

export default function WorkspaceSection() {
  const isMinimapVisible = useWorkspaceStore((state) => state.isMinimapVisible);
  const setMinimapVisible = useWorkspaceStore((state) => state.setMinimapVisible);
  const displayMode = useWorkspaceStore((state) => state.displayMode);
  const setDisplayMode = useWorkspaceStore((state) => state.setDisplayMode);
  const toolbarLayout = useWorkspaceStore((state) => state.toolbarLayout);
  const setToolbarLayout = useWorkspaceStore((state) => state.setToolbarLayout);
  const keepScreenAwake = useWorkspaceStore((state) => state.keepScreenAwake);
  const setKeepScreenAwake = useWorkspaceStore((state) => state.setKeepScreenAwake);

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
          hint="Prevent the device from dimming or locking while the console is open."
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
