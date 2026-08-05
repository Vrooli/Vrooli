import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Eye, Palette } from "lucide-react";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Select } from "../../components/ui/select";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { themesClient, type Theme } from "../../api/themes";
import { type ColorScheme, type DeviceFiltersValue, type VisionFilter } from "../../hooks/useDeviceFilters";
import { errorMessage } from "../../lib/errorMessage";
import { AnchoredMenu } from "./AnchoredMenu";

interface Props {
  /** Posts to every live preview frame, not only the first gallery item. */
  postToFrames: (message: unknown) => void;
  /** Set true once the harness has posted "preview-ready"; we re-apply
   *  the active pack on every reload so it survives save-driven nav. */
  previewReady: boolean;
  /** Color-scheme mode — the single owner of light/dark. */
  colorScheme: ColorScheme;
  setColorScheme: (scheme: ColorScheme) => void;
  filters: Pick<DeviceFiltersValue, "visionFilter" | "setVisionFilter" | "blurPx" | "setBlurPx" | "blurMin" | "blurMax">;
}

/**
 * ThemeSwitcher — the preview theming control (TH-003 surface).
 *
 * Theming has two orthogonal axes, and this control makes each one a
 * single-owner surface so they can no longer fight:
 *
 *  1. **Color-scheme mode** (System / Light / Dark) — a segmented toggle
 *     bound to `useDeviceFilters().colorScheme`. ComponentEditor derives
 *     the resolved light/dark from it and is the *only* sender of
 *     `rcl-resolved-theme`. Picking a token pack must never move it.
 *  2. **Token pack** — a built-in palette or a scenario's DESIGN.md-derived
 *     palette. The chosen pack's normalized CSS-custom-property tokens are
 *     posted via `{type:"rcl-theme-apply", themeId, tokens}` (an override
 *     axis only). It never posts `rcl-resolved-theme`.
 *
 * Built-in packs are re-labelled (Slate / Midnight / High Contrast) so a
 * pack name can never be mistaken for a color-scheme mode.
 */

/** Server pack id → display-name string key. Renames the built-in packs
 *  so none collide with the color-scheme mode labels (System/Light/Dark).
 *  Unknown ids (future packs, scenario imports) keep their server name. */
function builtinPackKey(id: string) {
  switch (id) {
    case "light":
      return strings.components.themeSwitcher.packNames.slate;
    case "dark":
      return strings.components.themeSwitcher.packNames.midnight;
    case "high-contrast":
      return strings.components.themeSwitcher.packNames.highContrast;
    default:
      return null;
  }
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

export function ThemeSwitcher({
  postToFrames,
  previewReady,
  colorScheme,
  setColorScheme,
  filters,
}: Props) {
  const { t } = useTranslation();
  const [selection, setSelection] = useState("");
  const [scenarioId, setScenarioId] = useState("");
  const [activeTheme, setActiveTheme] = useState<Theme | null>(null);

  const builtinsQuery = useQuery({
    queryKey: ["themes", "builtin"],
    queryFn: () => themesClient.listBuiltinThemes({}),
    staleTime: Infinity,
  });

  const builtinThemeQuery = useQuery({
    enabled: selection.startsWith("builtin:"),
    queryKey: ["themes", "builtin", selection],
    queryFn: () =>
      themesClient.getBuiltinTheme({ id: selection.slice("builtin:".length) }),
  });

  const scenarioThemeQuery = useQuery({
    enabled: selection.startsWith("scenario:"),
    queryKey: ["themes", "scenario", selection],
    queryFn: () =>
      themesClient.getThemeFromScenario({
        scenarioId: selection.slice("scenario:".length),
      }),
  });

  useEffect(() => {
    const theme =
      builtinThemeQuery.data?.theme ?? scenarioThemeQuery.data?.theme ?? null;
    setActiveTheme(theme);
  }, [builtinThemeQuery.data, scenarioThemeQuery.data]);

  // Re-apply the active pack on every (re)load — the harness JS state is
  // wiped when the iframe reloads after a save, so we resend. The pack is a
  // token-override axis ONLY: it posts `rcl-theme-apply` and never
  // `rcl-resolved-theme`, leaving light/dark ownership with the mode toggle.
  useEffect(() => {
    if (!previewReady) return;
    postToFrames({
      type: "rcl-theme-apply",
      themeId: activeTheme?.id ?? "",
      tokens: activeTheme ? { ...activeTheme.tokens } : {},
    });
  }, [previewReady, activeTheme, postToFrames]);

  const queryError = builtinThemeQuery.error ?? scenarioThemeQuery.error;
  const loading = builtinThemeQuery.isLoading || scenarioThemeQuery.isLoading;
  const packLabelFor = (theme: Theme): string => {
    const key = builtinPackKey(theme.id);
    return key ? t(key) : theme.name || theme.id;
  };

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
        <p className="mb-3 text-xs leading-5 text-app-muted-foreground">
          {t(strings.components.themeSwitcher.appearanceDescription)}
        </p>
        <section aria-labelledby="rcl-appearance-mode" className="border-b border-app-border pb-3">
          <h3 id="rcl-appearance-mode" className="text-sm font-semibold text-app-foreground">
            {t(strings.components.themeSwitcher.modeLabel)}
          </h3>
          <p className="mt-1 text-xs text-app-muted-foreground">{t(strings.components.themeSwitcher.modeDescription)}</p>
          <div
            data-testid={selectors.components.themeSwitcher.mode}
            role="group"
            aria-label={t(strings.components.themeSwitcher.modeLabel)}
            className="mt-2 inline-flex overflow-hidden rounded-md border border-app-border"
          >
        {MODE_OPTIONS.map((option) => {
          const active = colorScheme === option.value;
          return (
            <Button
              key={option.value}
              type="button"
              data-testid={option.testid}
              variant={active ? "primary" : "secondary"}
              aria-pressed={active}
              className="h-11 min-h-11 rounded-none border-0 px-2.5 text-xs"
              onClick={() => setColorScheme(option.value)}
            >
              {t(option.label)}
            </Button>
          );
        })}
      </div>
        </section>

        <section aria-labelledby="rcl-appearance-tokens" className="border-b border-app-border py-3">
          <h3 id="rcl-appearance-tokens" className="text-sm font-semibold text-app-foreground">
            {t(strings.components.themeSwitcher.packLabel)}
          </h3>
          <p className="mt-1 text-xs text-app-muted-foreground">{t(strings.components.themeSwitcher.packDescription)}</p>
          <label className="mt-2 block">
            <span className="sr-only">{t(strings.components.themeSwitcher.packLabel)}</span>
        <select
          data-testid={selectors.components.themeSwitcher.select}
          value={selection}
          onChange={(e) => setSelection(e.target.value)}
          className="h-9 min-h-9 w-full rounded-md border border-app-border bg-app-surface px-2 text-xs text-app-foreground"
        >
          <option value="">{t(strings.components.themeSwitcher.noneOption)}</option>
          <optgroup label={t(strings.components.themeSwitcher.builtinOptionGroup)}>
            {(builtinsQuery.data?.themes ?? []).map((th) => (
              <option key={th.id} value={`builtin:${th.id}`}>
                {packLabelFor(th)}
              </option>
            ))}
          </optgroup>
        </select>
          </label>
          <h4 className="mt-3 text-xs font-medium text-app-foreground">{t(strings.components.themeSwitcher.importLabel)}</h4>
          <p className="mt-1 text-xs text-app-muted-foreground">{t(strings.components.themeSwitcher.importDescription)}</p>
          <label className="mt-2 block text-xs text-app-muted-foreground" htmlFor="rcl-theme-scenario-id">
            {t(strings.components.themeSwitcher.scenarioInputLabel)}
          </label>
          <div className="mt-1 flex gap-2">
            <Input
              id="rcl-theme-scenario-id"
              data-testid={selectors.components.themeSwitcher.scenarioInput}
              value={scenarioId}
              onChange={(e) => setScenarioId(e.target.value)}
              placeholder={t(strings.components.themeSwitcher.scenarioInputPlaceholder)}
              className="h-9 min-w-0 flex-1 rounded-md text-xs"
            />
            <Button
              type="button"
              data-testid={selectors.components.themeSwitcher.scenarioApply}
              variant="secondary"
              className="h-9 shrink-0 rounded-md px-2 text-xs"
              disabled={!scenarioId.trim()}
              onClick={() => setSelection(`scenario:${scenarioId.trim()}`)}
            >
              {t(strings.components.themeSwitcher.scenarioApply)}
            </Button>
          </div>
        </section>

        <section aria-labelledby="rcl-appearance-vision" className="pt-3">
          <h3 id="rcl-appearance-vision" className="flex items-center gap-1.5 text-sm font-semibold text-app-foreground">
            <Eye aria-hidden className="h-4 w-4" />
            {t(strings.components.themeSwitcher.visualLabel)}
          </h3>
          <p className="mt-1 text-xs text-app-muted-foreground">{t(strings.components.themeSwitcher.visualDescription)}</p>
          <label className="mt-2 block text-xs text-app-muted-foreground" htmlFor="rcl-appearance-vision-filter">
            {t(strings.components.emulator.visionFilterLabel)}
          </label>
          <Select
            id="rcl-appearance-vision-filter"
            data-testid={selectors.components.themeSwitcher.visionFilterSelect}
            value={filters.visionFilter}
            onChange={(event) => filters.setVisionFilter(event.target.value as VisionFilter)}
            className="mt-1 h-9 min-h-9 text-xs"
            options={[
              { value: "none", label: t(strings.components.emulator.visionFilter.normal) },
              { value: "grayscale", label: t(strings.components.emulator.visionFilter.grayscale) },
              { value: "protanopia", label: t(strings.components.emulator.visionFilter.protanopia) },
              { value: "deuteranopia", label: t(strings.components.emulator.visionFilter.deuteranopia) },
              { value: "tritanopia", label: t(strings.components.emulator.visionFilter.tritanopia) },
            ]}
          />
          <label className="mt-3 flex items-center gap-2 text-xs text-app-muted-foreground" htmlFor="rcl-appearance-blur">
            <span>{t(strings.components.emulator.blurLabel)}</span>
            <Input
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

        {loading && (
        <span
          data-testid={selectors.components.themeSwitcher.status}
          className="shrink-0 text-app-muted-foreground"
        >
          {t(strings.components.themeSwitcher.loading)}
        </span>
      )}
        {activeTheme && !loading && (
        <span
          data-testid={selectors.components.themeSwitcher.status}
          className="min-w-0 truncate text-app-success"
        >
          {t(strings.components.themeSwitcher.applied, {
            name: packLabelFor(activeTheme),
          })}
        </span>
      )}
        {queryError && (
        <span
          data-testid={selectors.components.themeSwitcher.error}
          className="min-w-0 truncate text-app-danger"
        >
          {errorMessage(queryError, t)}
        </span>
        )}
      </AnchoredMenu>
    </div>
  );
}
