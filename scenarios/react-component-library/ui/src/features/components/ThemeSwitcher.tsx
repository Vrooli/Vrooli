/** @vrooliComponentSource forms.select */
import { Eye, Palette } from "lucide-react";

import { Button } from "../../components/Button";
import { Select } from "../../components/Select";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { type ColorScheme, type DeviceFiltersValue } from "../../hooks/useDeviceFilters";
import { AnchoredMenu } from "./AnchoredMenu";

export const PREVIEW_KITS = [
  { value: "vrooli-default", label: strings.components.themeSwitcher.kitOptions.default },
  { value: "vrooli-command-display", label: strings.components.themeSwitcher.kitOptions.commandDisplay },
  { value: "vrooli-conversion-landing", label: strings.components.themeSwitcher.kitOptions.conversionLanding },
] as const;

export type PreviewKit = (typeof PREVIEW_KITS)[number]["value"];

interface Props {
  previewReady: boolean;
  colorScheme: ColorScheme;
  setColorScheme: (scheme: ColorScheme) => void;
  kit: PreviewKit;
  setKit: (kit: PreviewKit) => void;
  filters: Pick<DeviceFiltersValue, "visionFilter" | "setVisionFilter" | "blurPx" | "setBlurPx" | "blurMin" | "blurMax">;
}

const MODE_OPTIONS = [
  { value: "system", label: strings.components.themeSwitcher.mode.system, testid: selectors.components.themeSwitcher.modeSystem },
  { value: "light", label: strings.components.themeSwitcher.mode.light, testid: selectors.components.themeSwitcher.modeLight },
  { value: "dark", label: strings.components.themeSwitcher.mode.dark, testid: selectors.components.themeSwitcher.modeDark },
] as const satisfies ReadonlyArray<{ value: ColorScheme; label: string; testid: string }>;

/** Owns preview-only axes: kit selection, light/dark mode, and visual checks. */
export function ThemeSwitcher({
  previewReady,
  colorScheme,
  setColorScheme,
  kit,
  setKit,
  filters,
}: Props) {
  const { t } = useTranslation();
  const activeMode = MODE_OPTIONS.find((option) => option.value === colorScheme);

  return (
    <div data-testid={selectors.components.themeSwitcher.root} className="min-w-0 text-xs">
      <AnchoredMenu
        label={t(strings.components.themeSwitcher.appearanceLabel)}
        summary={t(activeMode?.label ?? strings.components.themeSwitcher.mode.system)}
        icon={<Palette aria-hidden className="h-4 w-4" />}
        triggerTestId={selectors.components.themeSwitcher.appearanceToggle}
        panelTestId={selectors.components.themeSwitcher.appearancePanel}
      >
        <p className="mb-space-xs text-xs leading-5 text-app-muted-foreground">
          {t(strings.components.themeSwitcher.appearanceDescription)}
        </p>
        <section aria-labelledby="rcl-appearance-mode" className="border-b border-app-border pb-space-xs">
          <h3 id="rcl-appearance-mode" className="text-sm font-semibold text-app-foreground">
            {t(strings.components.themeSwitcher.modeLabel)}
          </h3>
          <p className="mt-space-3xs text-xs text-app-muted-foreground">{t(strings.components.themeSwitcher.modeDescription)}</p>
          <div data-testid={selectors.components.themeSwitcher.mode} role="group" aria-label={t(strings.components.themeSwitcher.modeLabel)} className="mt-space-2xs inline-flex overflow-hidden rounded-md border border-app-border">
            {MODE_OPTIONS.map((option) => (
              <Button
                key={option.value}
                type="button"
                data-testid={option.testid}
                variant={colorScheme === option.value ? "primary" : "secondary"}
                aria-pressed={colorScheme === option.value}
                className="h-11 min-h-11 rounded-none border-0 px-space-xs text-xs"
                onClick={() => setColorScheme(option.value)}
              >
                {t(option.label)}
              </Button>
            ))}
          </div>
        </section>

        <section aria-labelledby="rcl-appearance-kit" className="border-b border-app-border py-space-xs">
          <h3 id="rcl-appearance-kit" className="text-sm font-semibold text-app-foreground">
            {t(strings.components.themeSwitcher.kitLabel)}
          </h3>
          <p className="mt-space-3xs text-xs text-app-muted-foreground">{t(strings.components.themeSwitcher.kitDescription)}</p>
          <label className="mt-space-2xs block">
            <span className="sr-only">{t(strings.components.themeSwitcher.kitLabel)}</span>
            <select
              data-testid={selectors.components.themeSwitcher.kitSelect}
              value={kit}
              onChange={(event) => setKit(event.target.value as PreviewKit)}
              className="h-9 min-h-9 w-full rounded-md border border-app-border bg-app-surface px-space-2xs text-xs text-app-foreground"
            >
              {PREVIEW_KITS.map((option) => (
                <option key={option.value} value={option.value}>{t(option.label)}</option>
              ))}
            </select>
          </label>
        </section>

        <section aria-labelledby="rcl-appearance-vision" className="pt-space-xs">
          <h3 id="rcl-appearance-vision" className="flex items-center gap-space-2xs text-sm font-semibold text-app-foreground">
            <Eye aria-hidden className="h-4 w-4" />
            {t(strings.components.themeSwitcher.visualLabel)}
          </h3>
          <p className="mt-space-3xs text-xs text-app-muted-foreground">{t(strings.components.themeSwitcher.visualDescription)}</p>
          <label className="mt-space-2xs block text-xs text-app-muted-foreground" htmlFor="rcl-appearance-vision-filter">
            {t(strings.components.emulator.visionFilterLabel)}
          </label>
          <Select
            id="rcl-appearance-vision-filter"
            data-testid={selectors.components.themeSwitcher.visionFilterSelect}
            value={filters.visionFilter}
            onChange={(event) => filters.setVisionFilter(event.target.value as DeviceFiltersValue["visionFilter"])}
            className="mt-space-3xs h-9 min-h-9 text-xs"
            options={[
              { value: "none", label: t(strings.components.emulator.visionFilter.normal) },
              { value: "grayscale", label: t(strings.components.emulator.visionFilter.grayscale) },
              { value: "protanopia", label: t(strings.components.emulator.visionFilter.protanopia) },
              { value: "deuteranopia", label: t(strings.components.emulator.visionFilter.deuteranopia) },
              { value: "tritanopia", label: t(strings.components.emulator.visionFilter.tritanopia) },
            ]}
          />
          <label className="mt-space-xs flex items-center gap-space-2xs text-xs text-app-muted-foreground" htmlFor="rcl-appearance-blur">
            <span>{t(strings.components.emulator.blurLabel)}</span>
            <input
              id="rcl-appearance-blur"
              data-testid={selectors.components.themeSwitcher.blurSlider}
              type="range"
              min={filters.blurMin}
              max={filters.blurMax}
              step={1}
              value={filters.blurPx}
              onChange={(event) => filters.setBlurPx(Number(event.target.value))}
              className="h-6 min-h-0 flex-1 rounded-none border-0 bg-transparent p-0 accent-app-primary"
            />
            <span className="w-9 font-mono">{t(strings.components.emulator.blurValue, { px: filters.blurPx })}</span>
          </label>
        </section>
        {!previewReady && <span className="sr-only">{t(strings.components.themeSwitcher.loading)}</span>}
        <span className="sr-only" aria-live="polite">{t(strings.components.themeSwitcher.applied, { name: kit })}</span>
      </AnchoredMenu>
    </div>
  );
}
