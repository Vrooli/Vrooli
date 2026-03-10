import { useCallback } from "react";
import { X, GripHorizontal, Plus, Minus } from "lucide-react";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { useDraggablePosition } from "../hooks/useDraggablePosition";
import { HEADER_COLORS, TERMINAL_THEMES, DEFAULT_THEME_ID } from "../consts/config";
import { FONT_SIZE_MIN, FONT_SIZE_MAX } from "../lib/fontSizeUtils";
import { cn } from "../lib/classnames";
import { Button } from "./ui/button";

export default function AppearanceModal() {
  const appearanceModalPane = useWorkspaceStore((s) => s.appearanceModalPane);
  const setAppearanceModalPane = useWorkspaceStore((s) => s.setAppearanceModalPane);
  const panes = useWorkspaceStore((s) => s.panes);
  const setPaneColor = useWorkspaceStore((s) => s.setPaneColor);
  const setPaneTheme = useWorkspaceStore((s) => s.setPaneTheme);
  const setPaneFontSize = useWorkspaceStore((s) => s.setPaneFontSize);

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
              Appearance
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
          {/* Section 1: Header Color */}
          <section>
            <h3 className="text-xs font-semibold uppercase tracking-wider text-wc-text-muted mb-2">
              Header Color
            </h3>
            <div className="flex flex-wrap gap-1.5">
              {/* Transparent option */}
              <button
                type="button"
                data-testid="appearance-header-color-transparent"
                className={cn(
                  "h-6 w-6 rounded-full border",
                  currentColor === "transparent" ? "border-wc-accent ring-1 ring-wc-accent" : "border-wc-default",
                )}
                style={{ background: "rgb(var(--wc-surface-input))" }}
                onClick={() => setPaneColor(sessionId, "transparent")}
                title="No color"
              />
              {HEADER_COLORS.map((color) => (
                <button
                  key={color}
                  type="button"
                  data-testid={`appearance-header-color-${color}`}
                  className={cn(
                    "h-6 w-6 rounded-full border",
                    currentColor === color ? "border-wc-accent ring-1 ring-wc-accent" : "border-wc-default",
                  )}
                  style={{ backgroundColor: color }}
                  onClick={() => setPaneColor(sessionId, color)}
                  title={color}
                />
              ))}
            </div>
          </section>

          {/* Section 2: Terminal Theme */}
          <section>
            <h3 className="text-xs font-semibold uppercase tracking-wider text-wc-text-muted mb-2">
              Terminal Theme
            </h3>
            <div className="grid grid-cols-2 gap-2">
              {Object.values(TERMINAL_THEMES).map((theme) => (
                <button
                  key={theme.id}
                  type="button"
                  data-testid={`appearance-theme-${theme.id}`}
                  className={cn(
                    "rounded-lg border p-2 text-left transition-colors",
                    currentThemeId === theme.id
                      ? "border-wc-accent ring-1 ring-wc-accent"
                      : "border-wc-default hover:border-wc-text-faint",
                  )}
                  onClick={() => setPaneTheme(sessionId, theme.id)}
                >
                  <div
                    className="rounded px-2 py-1.5 mb-1.5 font-mono text-[10px] leading-tight"
                    style={{ backgroundColor: theme.colors.background, color: theme.colors.foreground }}
                  >
                    <span>$ hello world</span>
                    <span
                      className="inline-block ml-0.5 h-2.5 w-1 align-middle rounded-sm"
                      style={{ backgroundColor: theme.colors.cursor }}
                    />
                  </div>
                  <span className="text-xs text-wc-text-secondary">{theme.label}</span>
                </button>
              ))}
            </div>
          </section>

          {/* Section 3: Font Size */}
          <section>
            <h3 className="text-xs font-semibold uppercase tracking-wider text-wc-text-muted mb-2">
              Font Size
            </h3>
            <div className="flex items-center gap-1.5">
              <Button
                data-testid="appearance-font-decrease"
                variant="outline"
                size="icon"
                className="h-7 w-7"
                disabled={currentFontSize <= FONT_SIZE_MIN}
                onClick={() => setPaneFontSize(sessionId, currentFontSize - 1)}
              >
                <Minus className="h-3 w-3" />
              </Button>
              <span
                data-testid="appearance-font-value"
                className="w-8 text-center text-sm font-mono text-wc-text-primary"
              >
                {currentFontSize}
              </span>
              <Button
                data-testid="appearance-font-increase"
                variant="outline"
                size="icon"
                className="h-7 w-7"
                disabled={currentFontSize >= FONT_SIZE_MAX}
                onClick={() => setPaneFontSize(sessionId, currentFontSize + 1)}
              >
                <Plus className="h-3 w-3" />
              </Button>
            </div>
          </section>
        </div>
      </div>
    </>
  );
}
