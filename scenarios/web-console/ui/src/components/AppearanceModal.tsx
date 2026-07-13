import { useCallback } from "react";
import { CopyCheck } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { useWorkspaceSync } from "../hooks/useWorkspaceSync";
import { DEFAULT_THEME_ID } from "../consts/config";
import { strings } from "../consts/strings";
import { Button } from "./ui/button";
import { DrawerShell } from "./DrawerShell";
import HeaderColorPicker from "./appearance/HeaderColorPicker";
import ThemePicker from "./appearance/ThemePicker";
import FontSizeStepper from "./appearance/FontSizeStepper";

export default function AppearanceModal() {
  const { t } = useTranslation();
  const appearanceModalPane = useWorkspaceStore((s) => s.appearanceModalPane);
  const setAppearanceModalPane = useWorkspaceStore((s) => s.setAppearanceModalPane);
  const panes = useWorkspaceStore((s) => s.panes);
  const setPaneColor = useWorkspaceStore((s) => s.setPaneColor);
  const setPaneTheme = useWorkspaceStore((s) => s.setPaneTheme);
  const setPaneFontSize = useWorkspaceStore((s) => s.setPaneFontSize);
  const applyAppearanceToAll = useWorkspaceStore((s) => s.applyAppearanceToAll);
  const paneCount = useWorkspaceStore((s) => s.panes.length);
  const { syncPaneUpdate } = useWorkspaceSync();

  const pane = panes.find((p) => p.sessionId === appearanceModalPane);

  const close = useCallback(() => setAppearanceModalPane(null), [setAppearanceModalPane]);

  if (!appearanceModalPane || !pane) return null;

  const sessionId = pane.sessionId;
  const currentColor = pane.headerColor;
  const currentThemeId = pane.themeId ?? DEFAULT_THEME_ID;
  const currentFontSize = pane.fontSize ?? 14;

  return (
    <DrawerShell
      open
      onClose={close}
      size="compact"
      closeAriaLabel={t(strings.appearance.closeAriaLabel)}
      title={t(strings.appearance.title)}
      panelTestId="appearance-modal"
    >
      <div className="h-full space-y-5 overflow-y-auto p-4">
        <HeaderColorPicker
          currentColor={currentColor}
          onSelectColor={(color) => {
            setPaneColor(sessionId, color);
            syncPaneUpdate(sessionId, { header_color: color });
          }}
          testIdPrefix="appearance"
        />
        <ThemePicker
          currentThemeId={currentThemeId}
          onSelectTheme={(themeId) => {
            setPaneTheme(sessionId, themeId);
            syncPaneUpdate(sessionId, { theme_id: themeId });
          }}
          testIdPrefix="appearance"
        />
        <FontSizeStepper
          currentSize={currentFontSize}
          onChangeSize={(size) => {
            setPaneFontSize(sessionId, size);
            syncPaneUpdate(sessionId, { font_size: size });
          }}
          testIdPrefix="appearance"
        />

        {paneCount > 1 && (
          <div className="border-t border-wc-default pt-2">
            <Button
              data-testid="appearance-apply-all"
              variant="outline"
              className="w-full"
              onClick={() => {
                applyAppearanceToAll(sessionId);
                for (const target of panes) {
                  syncPaneUpdate(target.sessionId, {
                    header_color: currentColor,
                    theme_id: currentThemeId,
                    font_size: currentFontSize,
                  });
                }
              }}
            >
              <CopyCheck className="h-4 w-4 me-2" />
              {t(strings.appearance.applyToAll)}
            </Button>
          </div>
        )}
      </div>
    </DrawerShell>
  );
}
