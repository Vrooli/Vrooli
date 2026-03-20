import { LayoutList, Terminal } from "lucide-react";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import { Button } from "../ui/button";
import HeaderColorPicker from "../appearance/HeaderColorPicker";
import ThemePicker from "../appearance/ThemePicker";
import FontSizeStepper from "../appearance/FontSizeStepper";
import { SettingsCard, SettingsRow, SettingsSectionIntro } from "./primitives";

export default function NewPaneDefaultsSection() {
  const defaultHeaderColor = useWorkspaceStore((state) => state.defaultHeaderColor);
  const defaultThemeId = useWorkspaceStore((state) => state.defaultThemeId);
  const defaultFontSize = useWorkspaceStore((state) => state.defaultFontSize);
  const plusButtonBehavior = useWorkspaceStore((state) => state.plusButtonBehavior);
  const setDefaultHeaderColor = useWorkspaceStore((state) => state.setDefaultHeaderColor);
  const setDefaultThemeId = useWorkspaceStore((state) => state.setDefaultThemeId);
  const setDefaultFontSize = useWorkspaceStore((state) => state.setDefaultFontSize);
  const setPlusButtonBehavior = useWorkspaceStore((state) => state.setPlusButtonBehavior);

  return (
    <div className="space-y-4">
      <SettingsSectionIntro
        eyebrow="Appearance"
        title="New pane defaults"
        description="Set the baseline look for every new terminal pane without affecting existing sessions."
      />

      <SettingsCard className="space-y-5">
        <HeaderColorPicker
          currentColor={defaultHeaderColor}
          onSelectColor={setDefaultHeaderColor}
          testIdPrefix="defaults"
        />
        <ThemePicker
          currentThemeId={defaultThemeId}
          onSelectTheme={setDefaultThemeId}
          testIdPrefix="defaults"
        />
        <FontSizeStepper
          currentSize={defaultFontSize}
          onChangeSize={setDefaultFontSize}
          testIdPrefix="defaults"
        />
        <SettingsRow
          label="Plus button default"
          hint="Quick-tap action. The other moves to long-press."
          control={(
            <div className="flex items-center gap-2">
              <Button
                data-testid="plus-behavior-launcher"
                variant={plusButtonBehavior === "launcher" ? "default" : "outline"}
                size="sm"
                className="h-8 px-3"
                onClick={() => setPlusButtonBehavior("launcher")}
              >
                <LayoutList className="mr-1 h-3.5 w-3.5" />
                Launcher
              </Button>
              <Button
                data-testid="plus-behavior-new-terminal"
                variant={plusButtonBehavior === "new-terminal" ? "default" : "outline"}
                size="sm"
                className="h-8 px-3"
                onClick={() => setPlusButtonBehavior("new-terminal")}
              >
                <Terminal className="mr-1 h-3.5 w-3.5" />
                Empty terminal
              </Button>
            </div>
          )}
        />
      </SettingsCard>
    </div>
  );
}
