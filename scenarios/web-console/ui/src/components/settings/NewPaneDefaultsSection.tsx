import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import HeaderColorPicker from "../appearance/HeaderColorPicker";
import ThemePicker from "../appearance/ThemePicker";
import FontSizeStepper from "../appearance/FontSizeStepper";
import { SettingsCard, SettingsSectionIntro } from "./primitives";

export default function NewPaneDefaultsSection() {
  const defaultHeaderColor = useWorkspaceStore((state) => state.defaultHeaderColor);
  const defaultThemeId = useWorkspaceStore((state) => state.defaultThemeId);
  const defaultFontSize = useWorkspaceStore((state) => state.defaultFontSize);
  const setDefaultHeaderColor = useWorkspaceStore((state) => state.setDefaultHeaderColor);
  const setDefaultThemeId = useWorkspaceStore((state) => state.setDefaultThemeId);
  const setDefaultFontSize = useWorkspaceStore((state) => state.setDefaultFontSize);

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
      </SettingsCard>
    </div>
  );
}
