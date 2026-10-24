import { LayoutList, Terminal } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import { strings } from "../../consts/strings";
import { Button } from "../ui/button";
import HeaderColorPicker from "../appearance/HeaderColorPicker";
import ThemePicker from "../appearance/ThemePicker";
import FontSizeStepper from "../appearance/FontSizeStepper";

import { SettingsList } from "@vrooli/react-component-library/SettingsList/1";

export default function NewPaneDefaultsSection() {
  const { t } = useTranslation();
  const defaultHeaderColor = useWorkspaceStore((state) => state.defaultHeaderColor);
  const defaultThemeId = useWorkspaceStore((state) => state.defaultThemeId);
  const defaultFontSize = useWorkspaceStore((state) => state.defaultFontSize);
  const plusButtonBehavior = useWorkspaceStore((state) => state.plusButtonBehavior);
  const setDefaultHeaderColor = useWorkspaceStore((state) => state.setDefaultHeaderColor);
  const setDefaultThemeId = useWorkspaceStore((state) => state.setDefaultThemeId);
  const setDefaultFontSize = useWorkspaceStore((state) => state.setDefaultFontSize);
  const setPlusButtonBehavior = useWorkspaceStore((state) => state.setPlusButtonBehavior);

  return (
    <SettingsList>
      <SettingsList.Intro
        eyebrow={t(strings.settings.newPaneDefaultsSection.eyebrow)}
        title={t(strings.settings.newPaneDefaultsSection.title)}
        description={t(strings.settings.newPaneDefaultsSection.description)}
      />

      <SettingsList.Group>
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
        <SettingsList.Row
          label={t(strings.settings.newPaneDefaultsSection.plusButtonLabel)}
          hint={t(strings.settings.newPaneDefaultsSection.plusButtonHint)} control="compact">{(
            <div className="flex items-center gap-2">
              <Button
                data-testid="plus-behavior-launcher"
                variant={plusButtonBehavior === "launcher" ? "default" : "outline"}
                size="sm"
                className="h-8 px-3"
                onClick={() => setPlusButtonBehavior("launcher")}
              >
                <LayoutList className="me-1 h-3.5 w-3.5" />
                {t(strings.settings.newPaneDefaultsSection.launcher)}
              </Button>
              <Button
                data-testid="plus-behavior-new-terminal"
                variant={plusButtonBehavior === "new-terminal" ? "default" : "outline"}
                size="sm"
                className="h-8 px-3"
                onClick={() => setPlusButtonBehavior("new-terminal")}
              >
                <Terminal className="me-1 h-3.5 w-3.5" />
                {t(strings.settings.newPaneDefaultsSection.emptyTerminal)}
              </Button>
            </div>
          )}</SettingsList.Row>
      </SettingsList.Group>
    </SettingsList>
  );
}
