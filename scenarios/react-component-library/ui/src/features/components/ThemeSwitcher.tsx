/** @vrooliComponentSource forms.select */
import { Eye, Palette } from "lucide-react";
import { useEffect } from "react";
import { useQuery } from "@tanstack/react-query";

import { Button } from "@vrooli/react-component-library/Button/2";
import { Select } from "@vrooli/react-component-library/Select/1";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { type ColorScheme, type DeviceFiltersValue } from "../../hooks/useDeviceFilters";
import { AnchoredMenu } from "./AnchoredMenu";
import { componentsClient } from "../../api/components";

export type PreviewKit = string;

interface Props {
  previewReady: boolean;
  colorScheme: ColorScheme;
  setColorScheme: (scheme: ColorScheme) => void;
  kit: PreviewKit;
  setKit: (kit: PreviewKit) => void;
  filters: Pick<
    DeviceFiltersValue,
    "visionFilter" | "setVisionFilter" | "blurPx" | "setBlurPx" | "blurMin" | "blurMax"
  >;
  compactOnMobile?: boolean;
}

const MODE_OPTIONS = [
  {
    value: "system",
    label: strings.components.themeSwitcher.mode.system,
    testid: selectors.components.themeSwitcher.modeSystem,
  },
  {
    value: "light",
    label: strings.components.themeSwitcher.mode.light,
    testid: selectors.components.themeSwitcher.modeLight,
  },
  {
    value: "dark",
    label: strings.components.themeSwitcher.mode.dark,
    testid: selectors.components.themeSwitcher.modeDark,
  },
] as const satisfies ReadonlyArray<{ value: ColorScheme; label: string; testid: string }>;

/** Owns preview-only axes: kit selection, light/dark mode, and visual checks. */
export function ThemeSwitcher({
  previewReady,
  colorScheme,
  setColorScheme,
  kit,
  setKit,
  filters,
  compactOnMobile = false,
}: Props) {
  const { t } = useTranslation();
  const activeMode = MODE_OPTIONS.find((option) => option.value === colorScheme);
  const { data: designStyles } = useQuery({
    queryKey: ["design-styles"],
    queryFn: () => componentsClient.listDesignStyles({}),
    staleTime: 60_000,
  });
  const previewKits = designStyles?.styles ?? [];

  useEffect(() => {
    if (previewKits.length > 0 && !previewKits.some((style) => style.id === kit)) {
      setKit(previewKits[0]!.id);
    }
  }, [kit, previewKits, setKit]);

  return (
    <div data-testid={selectors.components.themeSwitcher.root} className="min-w-0 text-xs">
      <AnchoredMenu
        label={t(strings.components.themeSwitcher.appearanceLabel)}
        summary={t(activeMode?.label ?? strings.components.themeSwitcher.mode.system)}
        icon={<Palette aria-hidden className="h-icon-sm w-icon-sm" />}
        triggerTestId={selectors.components.themeSwitcher.appearanceToggle}
        panelTestId={selectors.components.themeSwitcher.appearancePanel}
        compactOnMobile={compactOnMobile}
      >
        <p className="mb-space-xs text-xs leading-5 text-app-muted-foreground">
          {t(strings.components.themeSwitcher.appearanceDescription)}
        </p>
        <section
          aria-labelledby="rcl-appearance-mode"
          className="border-b border-app-border pb-space-xs"
        >
          <h3 id="rcl-appearance-mode" className="text-sm font-semibold text-app-foreground">
            {t(strings.components.themeSwitcher.modeLabel)}
          </h3>
          <p className="mt-space-3xs text-xs text-app-muted-foreground">
            {t(strings.components.themeSwitcher.modeDescription)}
          </p>
          <div
            data-testid={selectors.components.themeSwitcher.mode}
            role="group"
            aria-label={t(strings.components.themeSwitcher.modeLabel)}
            className="mt-space-2xs inline-flex overflow-hidden rounded-md border border-app-border"
          >
            {MODE_OPTIONS.map((option) => (
              <Button
                key={option.value}
                type="button"
                data-testid={option.testid}
                variant={colorScheme === option.value ? "primary" : "secondary"}
                aria-pressed={colorScheme === option.value}
                className="h-touch min-h-touch rounded-none border-0 px-space-xs text-xs"
                onClick={() => setColorScheme(option.value)}
              >
                {t(option.label)}
              </Button>
            ))}
          </div>
        </section>

        <section
          aria-labelledby="rcl-appearance-kit"
          className="border-b border-app-border py-space-xs"
        >
          <h3 id="rcl-appearance-kit" className="text-sm font-semibold text-app-foreground">
            {t(strings.components.themeSwitcher.kitLabel)}
          </h3>
          <p className="mt-space-3xs text-xs text-app-muted-foreground">
            {t(strings.components.themeSwitcher.kitDescription)}
          </p>
          <label className="mt-space-2xs block">
            <span className="sr-only">{t(strings.components.themeSwitcher.kitLabel)}</span>
            <select
              data-testid={selectors.components.themeSwitcher.kitSelect}
              value={kit}
              onChange={(event) => setKit(event.target.value)}
              className="h-control-sm min-h-control-sm w-full rounded-md border border-app-border bg-app-surface px-space-2xs text-xs text-app-foreground"
            >
              {previewKits.map((option) => (
                <option key={option.id} value={option.id}>
                  {option.name || option.id}
                </option>
              ))}
            </select>
          </label>
        </section>

        <section aria-labelledby="rcl-appearance-vision" className="pt-space-xs">
          <h3
            id="rcl-appearance-vision"
            className="flex items-center gap-space-2xs text-sm font-semibold text-app-foreground"
          >
            <Eye aria-hidden className="h-icon-sm w-icon-sm" />
            {t(strings.components.themeSwitcher.visualLabel)}
          </h3>
          <p className="mt-space-3xs text-xs text-app-muted-foreground">
            {t(strings.components.themeSwitcher.visualDescription)}
          </p>
          <label
            className="mt-space-2xs block text-xs text-app-muted-foreground"
            htmlFor="rcl-appearance-vision-filter"
          >
            {t(strings.components.emulator.visionFilterLabel)}
          </label>
          <Select
            id="rcl-appearance-vision-filter"
            data-testid={selectors.components.themeSwitcher.visionFilterSelect}
            value={filters.visionFilter}
            onChange={(event) =>
              filters.setVisionFilter(event.target.value as DeviceFiltersValue["visionFilter"])
            }
            className="mt-space-3xs h-control-sm min-h-control-sm text-xs"
            options={[
              { value: "none", label: t(strings.components.emulator.visionFilter.normal) },
              { value: "grayscale", label: t(strings.components.emulator.visionFilter.grayscale) },
              {
                value: "protanopia",
                label: t(strings.components.emulator.visionFilter.protanopia),
              },
              {
                value: "deuteranopia",
                label: t(strings.components.emulator.visionFilter.deuteranopia),
              },
              {
                value: "tritanopia",
                label: t(strings.components.emulator.visionFilter.tritanopia),
              },
            ]}
          />
          <label
            className="mt-space-xs flex items-center gap-space-2xs text-xs text-app-muted-foreground"
            htmlFor="rcl-appearance-blur"
          >
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
              className="h-icon-lg min-h-0 flex-1 rounded-none border-0 bg-transparent p-0 accent-app-primary"
            />
            <span className="w-control-sm font-mono">
              {t(strings.components.emulator.blurValue, { px: filters.blurPx })}
            </span>
          </label>
        </section>
        {!previewReady && (
          <span className="sr-only">{t(strings.components.themeSwitcher.loading)}</span>
        )}
        <span className="sr-only" aria-live="polite">
          {t(strings.components.themeSwitcher.applied, { name: kit })}
        </span>
      </AnchoredMenu>
    </div>
  );
}
