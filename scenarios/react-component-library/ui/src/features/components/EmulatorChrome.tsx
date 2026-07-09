import { type ReactNode, useId } from "react";
import { Eye, MonitorSmartphone, Palette, RotateCcw, Undo2, ZoomIn } from "lucide-react";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Select } from "../../components/ui/select";
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

const groupedControlClass =
  "inline-flex h-9 items-center overflow-hidden rounded-md border border-app-border bg-app-surface text-sm text-app-foreground";
const groupedIconClass =
  "inline-flex h-full w-9 items-center justify-center border-r border-app-border bg-app-surface-muted text-app-foreground";

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
      className="flex min-w-0 flex-wrap items-center gap-2 bg-app-surface px-3 py-2 text-app-muted-foreground"
      role="group"
      aria-label={t(strings.components.emulator.toolbarLabel)}
    >
      <Button
        type="button"
        size="icon"
        variant="secondary"
        className="h-9 min-h-9 w-9 min-w-9 border-app-primary/50 bg-app-primary/10 text-app-foreground hover:bg-app-surface-muted"
        aria-label={t(strings.components.emulator.viewportEnabled)}
        title={t(strings.components.emulator.viewportEnabled)}
      >
        <MonitorSmartphone aria-hidden className="h-4 w-4" />
      </Button>
      <label className="min-w-0" htmlFor={presetId}>
        <Select
          id={presetId}
          data-testid={selectors.components.emulator.presetSelect}
          value={emulator.presetId}
          onChange={(e) => emulator.setPreset(e.target.value as DevicePresetId)}
          className="rcl-device-select h-9 min-h-9 w-36 border-app-border bg-app-surface text-sm text-app-foreground shadow-inner"
          aria-label={t(strings.components.emulator.presetLabel)}
          options={emulator.presets.map((preset) => ({ value: preset.id, label: preset.label }))}
        />
      </label>

      <div
        data-testid={selectors.components.emulator.dimensions}
        className="inline-flex h-9 items-center gap-2 font-mono text-sm text-app-foreground"
        aria-live="polite"
      >
        <label htmlFor={widthInputId} className="sr-only">
          {t(strings.components.emulator.widthLabel)}
        </label>
        <Input
          id={widthInputId}
          type="number"
          inputMode="numeric"
          min={1}
          value={Math.round(emulator.displayWidth)}
          onChange={(e) => emulator.setDimension("width", Number(e.target.value))}
          disabled={!emulator.isResponsive}
          className="h-9 min-h-9 w-[4.6rem] border-app-border bg-app-surface-muted px-2 text-right text-sm text-app-foreground disabled:opacity-70"
        />
        <span className="text-app-muted-foreground" aria-hidden>
          {t(strings.components.emulator.dimensionSeparator)}
        </span>
        <label htmlFor={heightInputId} className="sr-only">
          {t(strings.components.emulator.heightLabel)}
        </label>
        <Input
          id={heightInputId}
          type="number"
          inputMode="numeric"
          min={1}
          value={Math.round(emulator.displayHeight)}
          onChange={(e) => emulator.setDimension("height", Number(e.target.value))}
          disabled={!emulator.isResponsive}
          className="h-9 min-h-9 w-[4.6rem] border-app-border bg-app-surface-muted px-2 text-right text-sm text-app-foreground disabled:opacity-70"
        />
      </div>

      <label className={groupedControlClass} htmlFor={zoomId}>
        <span className={groupedIconClass}>
          <ZoomIn aria-hidden className="h-4 w-4" />
        </span>
        <Select
          id={zoomId}
          data-testid={selectors.components.emulator.zoomValue}
          value={emulator.zoom}
          onChange={(e) => emulator.setZoom(Number(e.target.value))}
          className="rcl-device-select h-full min-h-0 w-[5.4rem] rounded-none border-0 bg-app-surface-muted px-3 text-sm text-app-foreground"
          aria-label={t(strings.components.emulator.zoomLabel)}
          options={emulator.zoomLevels.map((level) => ({
            value: String(level),
            label: t(strings.components.emulator.zoomValue, {
              percent: Math.round(level * 100),
            }),
          }))}
        />
      </label>

      {filters && (
        <>
          <label className={groupedControlClass} htmlFor={colorSchemeId}>
            <span className={groupedIconClass}>
              <Palette aria-hidden className="h-4 w-4" />
            </span>
            <Select
              id={colorSchemeId}
              data-testid={selectors.components.emulator.colorSchemeSelect}
              value={filters.colorScheme}
              onChange={(e) => filters.setColorScheme(e.target.value as ColorScheme)}
              className="rcl-device-select h-full min-h-0 w-[6.6rem] rounded-none border-0 bg-app-surface-muted px-3 text-sm text-app-foreground"
              aria-label={t(strings.components.emulator.colorSchemeLabel)}
              options={[
                { value: "system", label: t(strings.components.emulator.colorScheme.system) },
                { value: "light", label: t(strings.components.emulator.colorScheme.light) },
                { value: "dark", label: t(strings.components.emulator.colorScheme.dark) },
              ]}
            />
          </label>
          <label className={groupedControlClass} htmlFor={visionFilterId}>
            <span className={groupedIconClass}>
              <Eye aria-hidden className="h-4 w-4" />
            </span>
            <Select
              id={visionFilterId}
              data-testid={selectors.components.emulator.visionFilterSelect}
              value={filters.visionFilter}
              onChange={(e) => filters.setVisionFilter(e.target.value as VisionFilter)}
              className="rcl-device-select h-full min-h-0 w-[8.4rem] rounded-none border-0 bg-app-surface-muted px-3 text-sm text-app-foreground"
              aria-label={t(strings.components.emulator.visionFilterLabel)}
              options={[
                { value: "none", label: t(strings.components.emulator.visionFilter.normal) },
                { value: "grayscale", label: t(strings.components.emulator.visionFilter.grayscale) },
                { value: "protanopia", label: t(strings.components.emulator.visionFilter.protanopia) },
                { value: "deuteranopia", label: t(strings.components.emulator.visionFilter.deuteranopia) },
                { value: "tritanopia", label: t(strings.components.emulator.visionFilter.tritanopia) },
              ]}
            />
          </label>
          <label className="inline-flex h-9 items-center gap-2 text-sm" htmlFor={blurId}>
            <span className="text-app-muted-foreground">{t(strings.components.emulator.blurLabel)}</span>
            <Input
              id={blurId}
              data-testid={selectors.components.emulator.blurSlider}
              type="range"
              min={filters.blurMin}
              max={filters.blurMax}
              step={1}
              value={filters.blurPx}
              onChange={(e) => filters.setBlurPx(Number(e.target.value))}
              aria-label={t(strings.components.emulator.blurLabel)}
              className="h-6 min-h-0 w-24 rounded-none border-0 bg-transparent p-0 accent-app-primary"
            />
            <span
              data-testid={selectors.components.emulator.blurValue}
              className="w-9 font-mono text-app-muted-foreground"
            >
              {t(strings.components.emulator.blurValue, { px: filters.blurPx })}
            </span>
          </label>
        </>
      )}

      <Button
        type="button"
        data-testid={selectors.components.emulator.rotate}
        onClick={emulator.rotate}
        aria-pressed={emulator.isRotated}
        aria-label={t(strings.components.emulator.rotate)}
        title={t(strings.components.emulator.rotate)}
        size="icon"
        variant="secondary"
        className="h-9 min-h-9 w-9 min-w-9 border-app-border bg-app-surface-muted text-app-foreground hover:bg-app-surface-muted"
      >
        <RotateCcw aria-hidden className="h-4 w-4" />
      </Button>
      <Button
        type="button"
        data-testid={selectors.components.emulator.reset}
        onClick={emulator.reset}
        aria-label={t(strings.components.emulator.reset)}
        title={t(strings.components.emulator.reset)}
        size="icon"
        variant="secondary"
        className="h-9 min-h-9 w-9 min-w-9 border-app-border bg-app-surface-muted text-app-foreground hover:bg-app-surface-muted"
      >
        <Undo2 aria-hidden className="h-4 w-4" />
      </Button>
    </div>
  );
}

export function EmulatorViewport({ emulator, filters, children }: EmulatorViewportProps) {
  return (
    <div className="h-full w-full overflow-auto bg-app-background">
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
      <div className="border-b border-app-border bg-app-surface px-3 py-2">
        <EmulatorToolbar emulator={emulator} filters={filters} />
      </div>
      <EmulatorViewport emulator={emulator} filters={filters}>
        {children}
      </EmulatorViewport>
    </div>
  );
}
