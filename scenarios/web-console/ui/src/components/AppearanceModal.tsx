import { useCallback } from "react";
import { X, GripHorizontal, CopyCheck } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { useDraggablePosition } from "../hooks/useDraggablePosition";
import { DEFAULT_THEME_ID } from "../consts/config";
import { strings } from "../consts/strings";
import { Button } from "./ui/button";
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

  const pane = panes.find((p) => p.sessionId === appearanceModalPane);

  const { elementRef, floatingStyle, pointerHandlers, handleClickCapture } =
    useDraggablePosition({
      isActive: appearanceModalPane !== null,
      storageKey: "wc-appearance-pos",
      defaultPosition: () => {
        if (typeof window === "undefined") return { x: 100, y: 100 };
        return {
          x: Math.max(12, (window.innerWidth - 384) / 2),
          y: Math.max(12, window.innerHeight * 0.15),
        };
      },
    });

  const close = useCallback(() => setAppearanceModalPane(null), [setAppearanceModalPane]);

  if (!appearanceModalPane || !pane) return null;

  const sessionId = pane.sessionId;
  const currentColor = pane.headerColor;
  const currentThemeId = pane.themeId ?? DEFAULT_THEME_ID;
  const currentFontSize = pane.fontSize ?? 14;

  return (
    <>
      {/* Backdrop */}
      <div
        data-testid="appearance-backdrop"
        className="fixed inset-0 z-40 bg-wc-backdrop"
        onClick={close}
      />

      {/* Modal */}
      <div
        ref={(node) => { elementRef.current = node; }}
        data-testid="appearance-modal"
        className="fixed left-0 top-0 z-50 w-96 max-w-[calc(100vw-24px)] max-h-[80vh] overflow-hidden rounded-lg border border-wc-default bg-wc-surface-raised shadow-2xl flex flex-col"
        style={floatingStyle}
        onPointerDown={(e) => {
          const target = e.target as HTMLElement | null;
          const isOnHandle = Boolean(target?.closest("[data-drag-handle]"));
          const isOnControl = Boolean(target?.closest("button, a, input, textarea, select"));
          if (isOnHandle && !isOnControl) {
            pointerHandlers.onPointerDown(e);
          }
        }}
        onPointerMove={pointerHandlers.onPointerMove}
        onPointerUp={pointerHandlers.onPointerUp}
        onPointerCancel={pointerHandlers.onPointerCancel}
        onClickCapture={handleClickCapture}
      >
        {/* Drag handle header */}
        <div
          data-drag-handle
          className="flex items-center justify-between px-4 py-2 border-b border-wc-default cursor-grab active:cursor-grabbing select-none touch-none"
        >
          <div className="flex items-center gap-2">
            <GripHorizontal className="h-4 w-4 text-wc-text-faint" />
            <h2 className="text-sm font-semibold text-wc-text-primary">
              {t(strings.appearance.title)}
            </h2>
          </div>
          <Button
            data-testid="appearance-close"
            variant="ghost"
            size="icon"
            className="h-6 w-6"
            onClick={close}
          >
            <X className="h-3.5 w-3.5" />
          </Button>
        </div>

        {/* Scrollable content */}
        <div className="flex-1 overflow-y-auto p-4 space-y-5">
          <HeaderColorPicker
            currentColor={currentColor}
            onSelectColor={(color) => setPaneColor(sessionId, color)}
            testIdPrefix="appearance"
          />
          <ThemePicker
            currentThemeId={currentThemeId}
            onSelectTheme={(themeId) => setPaneTheme(sessionId, themeId)}
            testIdPrefix="appearance"
          />
          <FontSizeStepper
            currentSize={currentFontSize}
            onChangeSize={(size) => setPaneFontSize(sessionId, size)}
            testIdPrefix="appearance"
          />

          {paneCount > 1 && (
            <div className="pt-2 border-t border-wc-default">
              <Button
                data-testid="appearance-apply-all"
                variant="outline"
                className="w-full"
                onClick={() => applyAppearanceToAll(sessionId)}
              >
                <CopyCheck className="h-4 w-4 mr-2" />
                {t(strings.appearance.applyToAll)}
              </Button>
            </div>
          )}
        </div>
      </div>
    </>
  );
}
