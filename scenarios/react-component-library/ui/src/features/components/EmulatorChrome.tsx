import { type ReactNode, useId } from "react";
import { Eye, MonitorSmartphone, Palette, RotateCcw, Undo2, ZoomIn } from "lucide-react";

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

interface EmulatorToolbarProps {
  emulator: DeviceEmulationValue;
  filters?: DeviceFiltersValue;
}

interface EmulatorViewportProps {
  emulator: DeviceEmulationValue;
  filters?: DeviceFiltersValue;
  children: ReactNode;
}

const controlClass =
  "h-9 rounded-md border border-[#31405f] bg-[#080d1b] px-3 text-sm text-slate-100 shadow-inner shadow-black/20";
const iconButtonClass =
  "inline-flex h-9 w-9 items-center justify-center rounded-md border border-[#31405f] bg-[#11182b] text-slate-100 transition hover:bg-[#18233b] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#6ea2ff]/60";
const groupedControlClass =
  "inline-flex h-9 items-center overflow-hidden rounded-md border border-[#31405f] bg-[#080d1b] text-sm text-slate-100";
const groupedIconClass =
  "inline-flex h-full w-9 items-center justify-center border-r border-[#31405f] bg-[#11182b] text-slate-100";

export function EmulatorToolbar({ emulator, filters }: EmulatorToolbarProps) {
  const { t } = useTranslation();
  const idPrefix = useId();
  const presetId = `${idPrefix}-preset`;
  const colorSchemeId = `${idPrefix}-color-scheme`;
  const visionFilterId = `${idPrefix}-vision-filter`;
  const blurId = `${idPrefix}-blur`;
  const widthInputId = `${idPrefix}-width`;
  const heightInputId = `${idPrefix}-height`;
  const zoomId = `${idPrefix}-zoom`;

  return (
    <div
      data-testid={selectors.components.emulator.toolbar}
      className="flex min-w-0 flex-wrap items-center gap-2 bg-[#0f172a] px-3 py-2 text-slate-300"
      role="group"
      aria-label={t(strings.components.emulator.toolbarLabel)}
    >
      <button
        type="button"
        className={`${iconButtonClass} border-[#45608f] bg-[#17213a] text-white`}
        aria-label={t(strings.components.emulator.viewportEnabled)}
        title={t(strings.components.emulator.viewportEnabled)}
      >
        <MonitorSmartphone aria-hidden className="h-4 w-4" />
      </button>
      <label className="min-w-0" htmlFor={presetId}>
        <select
          id={presetId}
          data-testid={selectors.components.emulator.presetSelect}
          value={emulator.presetId}
          onChange={(e) => emulator.setPreset(e.target.value as DevicePresetId)}
          className={`${controlClass} rcl-device-select w-36`}
          aria-label={t(strings.components.emulator.presetLabel)}
        >
          {emulator.presets.map((p) => (
            <option key={p.id} value={p.id}>
              {p.label}
            </option>
          ))}
        </select>
      </label>

      <div
        data-testid={selectors.components.emulator.dimensions}
        className="inline-flex h-9 items-center gap-2 font-mono text-sm text-slate-100"
        aria-live="polite"
      >
        <label htmlFor={widthInputId} className="sr-only">
          {t(strings.components.emulator.widthLabel)}
        </label>
        <input
          id={widthInputId}
          type="number"
          inputMode="numeric"
          min={1}
          value={Math.round(emulator.displayWidth)}
          onChange={(e) => emulator.setDimension("width", Number(e.target.value))}
          disabled={!emulator.isResponsive}
          className="h-9 w-[4.6rem] rounded-md border border-[#31405f] bg-[#11182b] px-2 text-right text-slate-100 disabled:opacity-70"
        />
        <span className="text-slate-300" aria-hidden>
          {t(strings.components.emulator.dimensionSeparator)}
        </span>
        <label htmlFor={heightInputId} className="sr-only">
          {t(strings.components.emulator.heightLabel)}
        </label>
        <input
          id={heightInputId}
          type="number"
          inputMode="numeric"
          min={1}
          value={Math.round(emulator.displayHeight)}
          onChange={(e) => emulator.setDimension("height", Number(e.target.value))}
          disabled={!emulator.isResponsive}
          className="h-9 w-[4.6rem] rounded-md border border-[#31405f] bg-[#11182b] px-2 text-right text-slate-100 disabled:opacity-70"
        />
      </div>

      <label className={groupedControlClass} htmlFor={zoomId}>
        <span className={groupedIconClass}>
          <ZoomIn aria-hidden className="h-4 w-4" />
        </span>
        <select
          id={zoomId}
          data-testid={selectors.components.emulator.zoomValue}
          value={emulator.zoom}
          onChange={(e) => emulator.setZoom(Number(e.target.value))}
          className="rcl-device-select h-full w-[5.4rem] border-0 bg-[#11182b] px-3 text-sm text-slate-100"
          aria-label={t(strings.components.emulator.zoomLabel)}
        >
          {emulator.zoomLevels.map((level) => (
            <option key={level} value={level}>
              {t(strings.components.emulator.zoomValue, {
                percent: Math.round(level * 100),
              })}
            </option>
          ))}
        </select>
      </label>

      {filters && (
        <>
          <label className={groupedControlClass} htmlFor={colorSchemeId}>
            <span className={groupedIconClass}>
              <Palette aria-hidden className="h-4 w-4" />
            </span>
            <select
              id={colorSchemeId}
              data-testid={selectors.components.emulator.colorSchemeSelect}
              value={filters.colorScheme}
              onChange={(e) => filters.setColorScheme(e.target.value as ColorScheme)}
              className="rcl-device-select h-full w-[6.6rem] border-0 bg-[#11182b] px-3 text-sm text-slate-100"
              aria-label={t(strings.components.emulator.colorSchemeLabel)}
            >
              <option value="system">{t(strings.components.emulator.colorScheme.system)}</option>
              <option value="light">{t(strings.components.emulator.colorScheme.light)}</option>
              <option value="dark">{t(strings.components.emulator.colorScheme.dark)}</option>
            </select>
          </label>
          <label className={groupedControlClass} htmlFor={visionFilterId}>
            <span className={groupedIconClass}>
              <Eye aria-hidden className="h-4 w-4" />
            </span>
            <select
              id={visionFilterId}
              data-testid={selectors.components.emulator.visionFilterSelect}
              value={filters.visionFilter}
              onChange={(e) => filters.setVisionFilter(e.target.value as VisionFilter)}
              className="rcl-device-select h-full w-[8.4rem] border-0 bg-[#11182b] px-3 text-sm text-slate-100"
              aria-label={t(strings.components.emulator.visionFilterLabel)}
            >
              <option value="none">{t(strings.components.emulator.visionFilter.normal)}</option>
              <option value="grayscale">{t(strings.components.emulator.visionFilter.grayscale)}</option>
              <option value="protanopia">{t(strings.components.emulator.visionFilter.protanopia)}</option>
              <option value="deuteranopia">{t(strings.components.emulator.visionFilter.deuteranopia)}</option>
              <option value="tritanopia">{t(strings.components.emulator.visionFilter.tritanopia)}</option>
            </select>
          </label>
          <label className="inline-flex h-9 items-center gap-2 text-sm" htmlFor={blurId}>
            <span className="text-slate-400">{t(strings.components.emulator.blurLabel)}</span>
            <input
              id={blurId}
              data-testid={selectors.components.emulator.blurSlider}
              type="range"
              min={filters.blurMin}
              max={filters.blurMax}
              step={1}
              value={filters.blurPx}
              onChange={(e) => filters.setBlurPx(Number(e.target.value))}
              aria-label={t(strings.components.emulator.blurLabel)}
              className="h-6 w-24 accent-app-primary"
            />
            <span
              data-testid={selectors.components.emulator.blurValue}
              className="w-9 font-mono text-slate-300"
            >
              {t(strings.components.emulator.blurValue, { px: filters.blurPx })}
            </span>
          </label>
        </>
      )}

      <button
        type="button"
        data-testid={selectors.components.emulator.rotate}
        onClick={emulator.rotate}
        aria-pressed={emulator.isRotated}
        aria-label={t(strings.components.emulator.rotate)}
        title={t(strings.components.emulator.rotate)}
        className={iconButtonClass}
      >
        <RotateCcw aria-hidden className="h-4 w-4" />
      </button>
      <button
        type="button"
        data-testid={selectors.components.emulator.reset}
        onClick={emulator.reset}
        aria-label={t(strings.components.emulator.reset)}
        title={t(strings.components.emulator.reset)}
        className={iconButtonClass}
      >
        <Undo2 aria-hidden className="h-4 w-4" />
      </button>
    </div>
  );
}

export function EmulatorViewport({ emulator, filters, children }: EmulatorViewportProps) {
  return (
    <div className="h-full w-full overflow-auto bg-[#05070d]">
      {filters && <DeviceVisionFilterDefs />}
      <div
        style={{
          width: emulator.scaledWidth,
          height: emulator.scaledHeight,
        }}
        className="relative"
      >
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
  return (
    <div
      data-testid={selectors.components.emulator.root}
      className="flex h-full flex-col"
    >
      <div className="border-b border-white/10 bg-[#111827] px-3 py-2">
        <EmulatorToolbar emulator={emulator} filters={filters} />
      </div>
      <EmulatorViewport emulator={emulator} filters={filters}>
        {children}
      </EmulatorViewport>
    </div>
  );
}
