/**
 * ExportStylizationSidebar - Left sidebar with stylization controls for export dialog.
 *
 * Provides Visual, Cursor, and Branding tabs for customizing the export appearance.
 * Only visible when stylization mode is set to "stylized".
 *
 * Reuses the same section components as the Record page's PreviewSettingsPanel,
 * excluding the Stream/Quality tab (not relevant for export).
 *
 * Features:
 * - Resizable width with persistence
 * - Shows message when stylization doesn't apply to selected format
 *
 * @module export/components
 */

import { useState } from "react";
import { Eye, MousePointer2, Sparkles, AlertCircle } from "lucide-react";
import clsx from "clsx";
import { useSettingsStore } from "@stores/settingsStore";
import {
  VisualSection,
  CursorSection,
  BrandingSection,
} from "@/domains/preview-settings/sections";
import { useExportSidebarResize } from "./useExportSidebarResize";

// =============================================================================
// Types
// =============================================================================

export type ExportSidebarSection = "visual" | "cursor" | "branding";

interface SectionConfig {
  id: ExportSidebarSection;
  label: string;
  icon: React.ReactNode;
}

const SIDEBAR_SECTIONS: SectionConfig[] = [
  { id: "visual", label: "Visual", icon: <Eye size={16} /> },
  { id: "cursor", label: "Cursor", icon: <MousePointer2 size={16} /> },
  { id: "branding", label: "Branding", icon: <Sparkles size={16} /> },
];

// =============================================================================
// Main Component
// =============================================================================

export interface ExportStylizationSidebarProps {
  className?: string;
  /** Whether stylization applies to the currently selected format */
  stylizationApplies?: boolean;
}

/**
 * Left sidebar for export stylization controls.
 * Contains tabs for Visual, Cursor, and Branding settings.
 */
export function ExportStylizationSidebar({
  className,
  stylizationApplies = true,
}: ExportStylizationSidebarProps) {
  const [activeSection, setActiveSection] = useState<ExportSidebarSection>("visual");

  // Get replay settings from the Zustand store
  const { replay, setReplaySetting } = useSettingsStore();

  // Get resize functionality
  const { width, isResizing, handleResizeStart } = useExportSidebarResize();

  return (
    <div
      className={clsx(
        "relative border-r border-gray-800 flex flex-col bg-slate-950/40 shrink-0",
        className,
      )}
      style={{ width: `${width}px`, minWidth: `${width}px` }}
    >
      {/* Resize Handle - on right edge */}
      <div
        className={clsx(
          "absolute top-0 right-0 w-1 h-full cursor-col-resize hover:bg-blue-500/50 active:bg-blue-500 transition-colors z-10",
          isResizing && "bg-blue-500",
        )}
        onMouseDown={handleResizeStart}
        role="separator"
        aria-orientation="vertical"
        aria-label="Resize sidebar"
        aria-valuenow={width}
      />

      {/* Tab Navigation */}
      <nav className="flex border-b border-gray-800">
        {SIDEBAR_SECTIONS.map((section) => {
          const isActive = activeSection === section.id;
          return (
            <button
              key={section.id}
              type="button"
              onClick={() => setActiveSection(section.id)}
              className={clsx(
                "flex-1 flex items-center justify-center gap-1.5 px-3 py-2.5 text-xs font-medium transition-colors",
                isActive
                  ? "text-blue-400 bg-blue-900/20 border-b-2 border-blue-500"
                  : "text-gray-400 hover:text-gray-200 hover:bg-gray-800/50",
              )}
            >
              <span className={isActive ? "text-blue-400" : ""}>{section.icon}</span>
              <span>{section.label}</span>
            </button>
          );
        })}
      </nav>

      {/* Section Content */}
      <div className="flex-1 overflow-y-auto p-4">
        {/* Warning when stylization doesn't apply */}
        {!stylizationApplies && (
          <div className="mb-4 p-3 rounded-lg border border-amber-500/30 bg-amber-500/10">
            <div className="flex items-start gap-2">
              <AlertCircle size={16} className="text-amber-400 shrink-0 mt-0.5" />
              <div className="text-xs text-amber-200">
                <p className="font-medium">Stylization not applicable</p>
                <p className="text-amber-300/80 mt-1">
                  The selected format (JSON Package) exports raw data. Stylization settings will
                  apply when you select a video format (MP4, GIF).
                </p>
              </div>
            </div>
          </div>
        )}

        {activeSection === "visual" && (
          <VisualSection
            presentation={replay.presentation}
            background={replay.background}
            chromeTheme={replay.chromeTheme}
            browserScale={replay.browserScale}
            deviceFrameTheme={replay.deviceFrameTheme}
            onPresentationChange={(v) => setReplaySetting("presentation", v)}
            onBackgroundChange={(v) => setReplaySetting("background", v)}
            onChromeThemeChange={(v) => setReplaySetting("chromeTheme", v)}
            onBrowserScaleChange={(v) => setReplaySetting("browserScale", v)}
            onDeviceFrameThemeChange={(v) => setReplaySetting("deviceFrameTheme", v)}
          />
        )}
        {activeSection === "cursor" && (
          <CursorSection
            cursorTheme={replay.cursorTheme}
            cursorInitialPosition={replay.cursorInitialPosition}
            cursorClickAnimation={replay.cursorClickAnimation}
            cursorScale={replay.cursorScale}
            cursorSpeedProfile={replay.cursorSpeedProfile}
            cursorPathStyle={replay.cursorPathStyle}
            onCursorThemeChange={(v) => setReplaySetting("cursorTheme", v)}
            onCursorInitialPositionChange={(v) => setReplaySetting("cursorInitialPosition", v)}
            onCursorClickAnimationChange={(v) => setReplaySetting("cursorClickAnimation", v)}
            onCursorScaleChange={(v) => setReplaySetting("cursorScale", v)}
            onCursorSpeedProfileChange={(v) => setReplaySetting("cursorSpeedProfile", v)}
            onCursorPathStyleChange={(v) => setReplaySetting("cursorPathStyle", v)}
          />
        )}
        {activeSection === "branding" && <BrandingSection />}
      </div>
    </div>
  );
}

export default ExportStylizationSidebar;
