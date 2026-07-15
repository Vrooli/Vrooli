import { type ReactNode, useEffect, useId, useRef, useState } from "react";
import { Eye, MoreHorizontal, RotateCcw, Undo2, ZoomIn } from "lucide-react";

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

/**
 * Below this measured container width (px) the toolbar folds its secondary
 * controls (vision filter, blur, rotate) into an overflow menu so the
 * primary device controls never wrap past two rows. Keyed on the measured
 * panel/container width via ResizeObserver — the preview panel is
 * user-resizable, so a viewport media query would be wrong here.
 */
export const TOOLBAR_INLINE_MIN_WIDTH = 720;

/**
 * Track the rendered width of a container element. Defaults to a wide value
 * so that, before the first measurement (and in environments without
 * ResizeObserver, e.g. jsdom), every control renders inline.
 */
function useContainerWidth<T extends HTMLElement>() {
  const ref = useRef<T | null>(null);
  const [width, setWidth] = useState<number>(Number.POSITIVE_INFINITY);

  useEffect(() => {
    const element = ref.current;
    if (!element || typeof ResizeObserver === "undefined") return undefined;
    setWidth(element.getBoundingClientRect().width);
    const observer = new ResizeObserver((entries) => {
      const entry = entries[0];
      if (entry) setWidth(entry.contentRect.width);
    });
    observer.observe(element);
    return () => observer.disconnect();
  }, []);

  return { ref, width };
}

const groupedControlClass =
  "inline-flex h-9 items-center overflow-hidden rounded-md border border-app-border bg-app-surface text-sm text-app-foreground";
const groupedIconClass =
  "inline-flex h-full w-9 items-center justify-center border-r border-app-border bg-app-surface-muted text-app-foreground";

export function EmulatorToolbar({ emulator, filters }: EmulatorToolbarProps) {
  const { t } = useTranslation();
  const idPrefix = useId();
  const presetId = `${idPrefix}-preset`;
  const visionFilterId = `${idPrefix}-vision-filter`;
  const blurId = `${idPrefix}-blur`;
  const widthInputId = `${idPrefix}-width`;
  const heightInputId = `${idPrefix}-height`;
  const zoomId = `${idPrefix}-zoom`;

  const { ref: containerRef, width } = useContainerWidth<HTMLDivElement>();
  const collapsed = width < TOOLBAR_INLINE_MIN_WIDTH;
  const [overflowOpen, setOverflowOpen] = useState(false);
  const overflowRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!collapsed && overflowOpen) setOverflowOpen(false);
  }, [collapsed, overflowOpen]);

  useEffect(() => {
    if (!overflowOpen) return undefined;
    const onPointerDown = (ev: MouseEvent) => {
      if (overflowRef.current && !overflowRef.current.contains(ev.target as Node)) {
        setOverflowOpen(false);
      }
    };
    const onKeyDown = (ev: KeyboardEvent) => {
      if (ev.key === "Escape") setOverflowOpen(false);
    };
    window.addEventListener("mousedown", onPointerDown);
    window.addEventListener("keydown", onKeyDown);
    return () => {
      window.removeEventListener("mousedown", onPointerDown);
      window.removeEventListener("keydown", onKeyDown);
    };
  }, [overflowOpen]);

  // Secondary controls (vision filter, blur, rotate). Rendered inline when
  // the toolbar is wide, or inside the overflow menu when it is narrow.
  const secondaryControls = (
    <>
      {filters && (
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
      )}
      {filters && (
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
      )}
      <Button
        type="button"
        data-testid={selectors.components.emulator.rotate}
        onClick={emulator.rotate}
        aria-pressed={emulator.isRotated}
        aria-label={t(strings.components.emulator.rotate)}
        title={t(strings.components.emulator.rotate)}
        variant="secondary"
        className="h-9 min-h-9 gap-2 border-app-border bg-app-surface-muted px-2 text-sm text-app-foreground hover:bg-app-surface-muted"
      >
        <RotateCcw aria-hidden className="h-4 w-4" />
        {collapsed && <span>{t(strings.components.emulator.rotate)}</span>}
      </Button>
    </>
  );

  return (
    <div
      ref={containerRef}
      data-testid={selectors.components.emulator.toolbar}
      className="flex w-full min-w-0 flex-wrap items-center gap-2 bg-app-surface px-3 py-2 text-app-muted-foreground"
      role="group"
      aria-label={t(strings.components.emulator.toolbarLabel)}
    >
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
        className={`${groupedControlClass} gap-1 px-2 font-mono`}
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
          className="h-7 min-h-0 w-[3.4rem] rounded-none border-0 bg-app-surface-muted px-1 text-right text-sm text-app-foreground disabled:opacity-70"
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
          className="h-7 min-h-0 w-[3.4rem] rounded-none border-0 bg-app-surface-muted px-1 text-right text-sm text-app-foreground disabled:opacity-70"
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

      {collapsed ? (
        <div ref={overflowRef} className="relative shrink-0">
          <Button
            type="button"
            data-testid={selectors.components.emulator.overflowToggle}
            variant="secondary"
            aria-expanded={overflowOpen}
            aria-label={t(strings.components.emulator.more)}
            title={t(strings.components.emulator.more)}
            className="h-9 min-h-9 gap-1.5 border-app-border bg-app-surface-muted px-2 text-sm text-app-foreground"
            onClick={() => setOverflowOpen((open) => !open)}
          >
            <MoreHorizontal aria-hidden className="h-4 w-4" />
          </Button>
          {overflowOpen && (
            <div
              data-testid={selectors.components.emulator.overflowPanel}
              className="absolute right-0 top-full z-30 mt-1 flex w-60 flex-col gap-3 rounded-md border border-app-border bg-app-surface p-3 shadow-lg"
            >
              {secondaryControls}
            </div>
          )}
        </div>
      ) : (
        secondaryControls
      )}

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
 * vision-filter / blur controls (req 05) and the viewport composes
 * `filter: url(#...) blur(Npx)` with the transform. Color-scheme mode is
 * owned by the ThemeSwitcher, not this toolbar.
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
