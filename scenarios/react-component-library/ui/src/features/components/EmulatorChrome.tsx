/** @vrooliComponentSource patterns.responsive-inspector-workspace */
import { type ReactNode, useId } from "react";
import { MonitorSmartphone, RotateCw, Undo2, ZoomIn } from "lucide-react";

import { Button } from "../../components/Button";
import { Input } from "../../components/Input";
import { Select } from "../../components/Select";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { type DeviceEmulationValue, type DevicePresetId } from "../../hooks/useDeviceEmulation";
import { type DeviceFiltersValue } from "../../hooks/useDeviceFilters";
import { DeviceVisionFilterDefs } from "./DeviceVisionFilterDefs";
import { AnchoredMenu } from "./AnchoredMenu";

interface EmulatorChromeProps {
  emulator: DeviceEmulationValue;
  /** Filters stay global preview state; their controls live in Appearance. */
  filters?: DeviceFiltersValue;
  children: ReactNode;
}

interface EmulatorToolbarProps {
  emulator: DeviceEmulationValue;
}

interface EmulatorViewportProps {
  emulator: DeviceEmulationValue;
  filters?: DeviceFiltersValue;
  mode?: "stage" | "gallery";
  children: ReactNode;
}

const deviceGroup = (id: DevicePresetId): "phone" | "tablet" | "desktop" | "responsive" => {
  if (id === "mobile") return "phone";
  if (id === "tablet") return "tablet";
  if (id === "responsive") return "responsive";
  return "desktop";
};

const deviceGroupLabel = (
  group: "phone" | "tablet" | "desktop",
  t: ReturnType<typeof useTranslation>["t"],
) => {
  if (group === "phone") return t(strings.components.emulator.deviceGroups.phone);
  if (group === "tablet") return t(strings.components.emulator.deviceGroups.tablet);
  return t(strings.components.emulator.deviceGroups.desktop);
};

/**
 * The viewport control is one progressive-disclosure menu instead of a row of
 * permanently visible/disabled inputs. Device and zoom state remain owned by
 * useDeviceEmulation so persisted behavior is unchanged.
 */
export function EmulatorToolbar({ emulator }: EmulatorToolbarProps) {
  const { t } = useTranslation();
  const idPrefix = useId();
  const widthInputId = `${idPrefix}-width`;
  const heightInputId = `${idPrefix}-height`;
  const zoomId = `${idPrefix}-zoom`;
  const selectedPreset = emulator.presets.find((preset) => preset.id === emulator.presetId);
  const summary = t(strings.components.emulator.summary, {
    device: selectedPreset?.label ?? emulator.presetId,
    zoom: Math.round(emulator.zoom * 100),
  });

  return (
    <div
      data-testid={selectors.components.emulator.toolbar}
      className="flex min-w-0 items-center"
      role="group"
      aria-label={t(strings.components.emulator.toolbarLabel)}
    >
      <AnchoredMenu
        label={t(strings.components.emulator.viewportLabel)}
        summary={summary}
        icon={<MonitorSmartphone aria-hidden className="h-4 w-4" />}
        triggerTestId={selectors.components.emulator.viewportToggle}
        panelTestId={selectors.components.emulator.viewportPanel}
      >
        <p className="mb-space-xs text-xs leading-5 text-app-muted-foreground">
          {t(strings.components.emulator.viewportDescription)}
        </p>
        {(["phone", "tablet", "desktop"] as const).map((group) => {
          const presets = emulator.presets.filter((preset) => deviceGroup(preset.id) === group);
          if (presets.length === 0) return null;
          return (
            <section key={group} className="border-b border-app-border py-space-xs first:pt-0">
              <h3 className="text-sm font-semibold text-app-foreground">
                {deviceGroupLabel(group, t)}
              </h3>
              <div className="mt-space-2xs grid grid-cols-1 gap-space-2xs sm:grid-cols-2">
                {presets.map((preset) => {
                  const active = preset.id === emulator.presetId;
                  return (
                    <Button
                      key={preset.id}
                      type="button"
                      data-testid={selectors.components.emulator.presetOption}
                      data-preset={preset.id}
                      variant={active ? "primary" : "secondary"}
                      aria-pressed={active}
                      className="h-auto min-h-12 min-w-11 justify-between px-space-xs py-space-2xs text-left text-xs"
                      onClick={() => emulator.setPreset(preset.id)}
                    >
                      <span>{preset.label}</span>
                      <span className="font-mono text-[11px] opacity-80">
                        {preset.width}×{preset.height}
                      </span>
                    </Button>
                  );
                })}
              </div>
            </section>
          );
        })}

        <section className="border-b border-app-border py-space-xs">
          <h3 className="text-sm font-semibold text-app-foreground">
            {t(strings.components.emulator.responsiveLabel)}
          </h3>
          <p className="mt-space-3xs text-xs text-app-muted-foreground">
            {t(strings.components.emulator.responsiveDescription)}
          </p>
          <Button
            type="button"
            data-testid={selectors.components.emulator.presetOption}
            data-preset="responsive"
            variant={emulator.isResponsive ? "primary" : "secondary"}
            aria-pressed={emulator.isResponsive}
            className="mt-space-2xs h-9 px-space-xs text-xs"
            onClick={() => emulator.setPreset("responsive")}
          >
            {t(strings.components.emulator.responsiveLabel)}
          </Button>
          {emulator.isResponsive && (
            <div
              data-testid={selectors.components.emulator.dimensions}
              className="mt-space-2xs grid grid-cols-2 gap-space-2xs"
            >
              <label className="text-xs text-app-muted-foreground" htmlFor={widthInputId}>
                {t(strings.components.emulator.widthLabel)}
                <Input
                  id={widthInputId}
                  type="number"
                  inputMode="numeric"
                  min={1}
                  value={Math.round(emulator.displayWidth)}
                  onChange={(event) => emulator.setDimension("width", Number(event.target.value))}
                  className="mt-space-3xs h-9 min-h-9 font-mono text-xs"
                />
              </label>
              <label className="text-xs text-app-muted-foreground" htmlFor={heightInputId}>
                {t(strings.components.emulator.heightLabel)}
                <Input
                  id={heightInputId}
                  type="number"
                  inputMode="numeric"
                  min={1}
                  value={Math.round(emulator.displayHeight)}
                  onChange={(event) => emulator.setDimension("height", Number(event.target.value))}
                  className="mt-space-3xs h-9 min-h-9 font-mono text-xs"
                />
              </label>
            </div>
          )}
        </section>

        <section className="pt-space-xs">
          <label className="text-xs text-app-muted-foreground" htmlFor={zoomId}>
            <span className="flex items-center gap-space-2xs">
              <ZoomIn aria-hidden className="h-3.5 w-3.5" />
              {t(strings.components.emulator.zoomLabel)}
            </span>
            <Select
              id={zoomId}
              data-testid={selectors.components.emulator.zoomValue}
              value={emulator.zoom}
              onChange={(event) => emulator.setZoom(Number(event.target.value))}
              className="mt-space-3xs h-9 min-h-9 text-xs"
              options={emulator.zoomLevels.map((level) => ({
                value: String(level),
                label: t(strings.components.emulator.zoomValue, {
                  percent: Math.round(level * 100),
                }),
              }))}
            />
          </label>
          <div className="mt-space-xs flex gap-space-2xs">
            <Button
              type="button"
              data-testid={selectors.components.emulator.rotate}
              onClick={emulator.rotate}
              aria-pressed={emulator.isRotated}
              variant="secondary"
              className="h-9 flex-1 gap-space-2xs px-space-2xs text-xs"
            >
              <RotateCw aria-hidden className="h-3.5 w-3.5" />
              {t(strings.components.emulator.rotate)}
            </Button>
            <Button
              type="button"
              data-testid={selectors.components.emulator.reset}
              onClick={emulator.reset}
              variant="secondary"
              className="h-9 flex-1 gap-space-2xs px-space-2xs text-xs"
            >
              <Undo2 aria-hidden className="h-3.5 w-3.5" />
              {t(strings.components.emulator.reset)}
            </Button>
            <Button
              type="button"
              data-testid="components-emulator-fit"
              onClick={emulator.fitToPane}
              variant="secondary"
              className="h-9 flex-1 px-space-2xs text-xs"
            >
              {t(strings.components.emulator.fit)}
            </Button>
          </div>
        </section>
      </AnchoredMenu>
    </div>
  );
}

export function EmulatorViewport({
  emulator,
  filters,
  mode = "stage",
  children,
}: EmulatorViewportProps) {
  const isStage = mode === "stage";
  return (
    <div
      data-testid={selectors.components.emulator.viewportFrame}
      data-emulator-viewport-frame
      className="h-full w-full overflow-auto bg-app-background"
    >
      {filters && <DeviceVisionFilterDefs />}
      <div
        data-testid={selectors.components.emulator.viewportCanvas}
        style={isStage ? { width: emulator.scaledWidth, height: emulator.scaledHeight } : undefined}
        className={isStage ? "relative" : "relative h-full w-full"}
      >
        <div
          data-testid={selectors.components.emulator.viewport}
          style={
            isStage
              ? {
                  width: emulator.displayWidth,
                  height: emulator.displayHeight,
                  transform: `scale(${emulator.zoom})`,
                  transformOrigin: "top left",
                  filter: filters?.filterCSS || undefined,
                }
              : { filter: filters?.filterCSS || undefined }
          }
          className={isStage ? "origin-top-left" : "h-full w-full"}
        >
          {children}
        </div>
      </div>
    </div>
  );
}

export function EmulatorChrome({ emulator, filters, children }: EmulatorChromeProps) {
  return (
    <div data-testid={selectors.components.emulator.root} className="flex h-full flex-col">
      <div className="border-b border-app-border bg-app-surface px-space-xs py-space-2xs">
        <EmulatorToolbar emulator={emulator} />
      </div>
      <EmulatorViewport emulator={emulator} filters={filters}>
        {children}
      </EmulatorViewport>
    </div>
  );
}
