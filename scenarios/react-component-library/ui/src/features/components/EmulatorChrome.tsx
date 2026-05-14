import { type ReactNode } from "react";

import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import {
  type DeviceEmulationValue,
  type DevicePresetId,
} from "../../hooks/useDeviceEmulation";
import {
  type DeviceFiltersValue,
  type ColorScheme,
  type VisionFilter,
} from "../../hooks/useDeviceFilters";
import { DeviceVisionFilterDefs } from "./DeviceVisionFilterDefs";

interface EmulatorChromeProps {
  emulator: DeviceEmulationValue;
  /**
   * Optional DevTools-style filter controls (req DV-001/002). When
   * omitted (e.g., legacy tests), only the device-emulation toolbar
   * renders; the filter chain stays empty and no SVG defs are emitted.
   */
  filters?: DeviceFiltersValue;
  children: ReactNode;
}

/**
 * EmulatorChrome wraps the preview iframe in the toolbar + viewport
 * surface specified by req VP-001..004. The toolbar drives the hook;
 * the viewport applies CSS transform: scale(zoom) to the children, so
 * zoom is purely on-screen and does not resize the iframe contents.
 *
 * When a `filters` value is supplied, the toolbar also exposes the
 * color-scheme / vision-filter / blur controls (req 05) and the
 * viewport composes `filter: url(#...) blur(Npx)` with the transform.
 */
export function EmulatorChrome({ emulator, filters, children }: EmulatorChromeProps) {
  const { t } = useTranslation();

  return (
    <div
      data-testid={selectors.components.emulator.root}
      className="flex h-full flex-col"
    >
      <div
        data-testid={selectors.components.emulator.toolbar}
        className="flex flex-wrap items-center gap-2 border-b border-white/5 bg-black/30 px-3 py-2 text-xs text-slate-300"
      >
        <label className="flex items-center gap-1.5">
          <span className="text-slate-400">
            {t(strings.components.emulator.presetLabel)}
          </span>
          <select
            data-testid={selectors.components.emulator.presetSelect}
            value={emulator.presetId}
            onChange={(e) => emulator.setPreset(e.target.value as DevicePresetId)}
            className="rounded border border-white/10 bg-black/40 px-2 py-1 text-slate-100"
          >
            {emulator.presets.map((p) => (
              <option key={p.id} value={p.id}>
                {p.label}
              </option>
            ))}
          </select>
        </label>

        <span
          data-testid={selectors.components.emulator.dimensions}
          className="font-mono text-slate-400"
        >
          {t(strings.components.emulator.dimensions, {
            width: emulator.displayWidth,
            height: emulator.displayHeight,
          })}
        </span>

        {filters && (
          <>
            <label className="flex items-center gap-1.5">
              <span className="text-slate-400">
                {t(strings.components.emulator.colorSchemeLabel)}
              </span>
              <select
                data-testid={selectors.components.emulator.colorSchemeSelect}
                value={filters.colorScheme}
                onChange={(e) => filters.setColorScheme(e.target.value as ColorScheme)}
                className="rounded border border-white/10 bg-black/40 px-2 py-1 text-slate-100"
              >
                <option value="system">{t(strings.components.emulator.colorScheme.system)}</option>
                <option value="light">{t(strings.components.emulator.colorScheme.light)}</option>
                <option value="dark">{t(strings.components.emulator.colorScheme.dark)}</option>
              </select>
            </label>
            <label className="flex items-center gap-1.5">
              <span className="text-slate-400">
                {t(strings.components.emulator.visionFilterLabel)}
              </span>
              <select
                data-testid={selectors.components.emulator.visionFilterSelect}
                value={filters.visionFilter}
                onChange={(e) => filters.setVisionFilter(e.target.value as VisionFilter)}
                className="rounded border border-white/10 bg-black/40 px-2 py-1 text-slate-100"
              >
                <option value="none">{t(strings.components.emulator.visionFilter.none)}</option>
                <option value="grayscale">{t(strings.components.emulator.visionFilter.grayscale)}</option>
                <option value="protanopia">{t(strings.components.emulator.visionFilter.protanopia)}</option>
                <option value="deuteranopia">{t(strings.components.emulator.visionFilter.deuteranopia)}</option>
                <option value="tritanopia">{t(strings.components.emulator.visionFilter.tritanopia)}</option>
              </select>
            </label>
            <label className="flex items-center gap-1.5">
              <span className="text-slate-400">
                {t(strings.components.emulator.blurLabel)}
              </span>
              <input
                data-testid={selectors.components.emulator.blurSlider}
                type="range"
                min={filters.blurMin}
                max={filters.blurMax}
                step={1}
                value={filters.blurPx}
                onChange={(e) => filters.setBlurPx(Number(e.target.value))}
                aria-label={t(strings.components.emulator.blurLabel)}
                className="h-6 w-24"
              />
              <span
                data-testid={selectors.components.emulator.blurValue}
                className="w-12 text-center font-mono text-slate-300"
              >
                {t(strings.components.emulator.blurValue, { px: filters.blurPx })}
              </span>
            </label>
          </>
        )}

        <div className="ml-auto flex items-center gap-1">
          <Button
            data-testid={selectors.components.emulator.zoomOut}
            variant="outline"
            onClick={emulator.zoomOut}
            disabled={emulator.zoom <= emulator.zoomMin}
            aria-label={t(strings.components.emulator.zoomOut)}
            className="h-7 w-7 p-0 text-xs"
          >
            −
          </Button>
          <span
            data-testid={selectors.components.emulator.zoomValue}
            className="w-12 text-center font-mono text-slate-300"
          >
            {t(strings.components.emulator.zoomValue, {
              percent: Math.round(emulator.zoom * 100),
            })}
          </span>
          <Button
            data-testid={selectors.components.emulator.zoomIn}
            variant="outline"
            onClick={emulator.zoomIn}
            disabled={emulator.zoom >= emulator.zoomMax}
            aria-label={t(strings.components.emulator.zoomIn)}
            className="h-7 w-7 p-0 text-xs"
          >
            +
          </Button>
          <Button
            data-testid={selectors.components.emulator.resetZoom}
            variant="outline"
            onClick={emulator.resetZoom}
            disabled={emulator.zoom === 1}
            className="h-7 px-2 text-xs"
          >
            {t(strings.components.emulator.resetZoom)}
          </Button>
          <Button
            data-testid={selectors.components.emulator.rotate}
            variant="outline"
            onClick={emulator.rotate}
            aria-pressed={emulator.isRotated}
            className="h-7 px-2 text-xs"
          >
            {t(strings.components.emulator.rotate)}
          </Button>
          <Button
            data-testid={selectors.components.emulator.reset}
            variant="outline"
            onClick={emulator.reset}
            className="h-7 px-2 text-xs"
          >
            {t(strings.components.emulator.reset)}
          </Button>
        </div>
      </div>

      <div className="flex-1 overflow-auto bg-slate-900/40">
        {filters && <DeviceVisionFilterDefs />}
        <div
          data-testid={selectors.components.emulator.viewport}
          style={{
            width: emulator.displayWidth,
            height: emulator.displayHeight,
            transform: `scale(${emulator.zoom})`,
            transformOrigin: "top left",
            filter: filters?.filterCSS || undefined,
          }}
          className="origin-top-left"
        >
          {children}
        </div>
      </div>
    </div>
  );
}
