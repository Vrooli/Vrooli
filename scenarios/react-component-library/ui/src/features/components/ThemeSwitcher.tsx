import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { themesClient, type Theme } from "../../api/themes";
import { errorMessage } from "../../lib/errorMessage";
import { cn } from "../../lib/utils";

interface Props {
  /** Posts to every live preview frame, not only the first gallery item. */
  postToFrames: (message: unknown) => void;
  /** Set true once the harness has posted "preview-ready"; we re-apply
   *  the active theme on every reload so it survives save-driven nav. */
  previewReady: boolean;
  /** Re-applies a chosen override after the app bridge changes theme. */
  appResolvedTheme: "light" | "dark";
  className?: string;
}

/**
 * ThemeSwitcher — TH-003 surface.
 *
 * Lists built-in themes + lets the user load a scenario's
 * DESIGN.md-derived theme. The chosen theme's normalized CSS-custom-
 * property tokens are posted to the preview harness via
 * `{type:"rcl-theme-apply", themeId, tokens}`. The harness child sets
 * each token on :root so component CSS variables resolve to the new
 * values immediately.
 */
export function ThemeSwitcher({ postToFrames, previewReady, appResolvedTheme, className }: Props) {
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

  // Re-apply the active theme on every (re)load — the harness JS state
  // is wiped when the iframe reloads after a save, so we resend.
  useEffect(() => {
    if (!previewReady) return;
    // Empty selection is deliberately "Follow app": clear any previous
    // overrides and leave resolved-theme ownership with ComponentEditor.
    if (!activeTheme) {
      postToFrames({ type: "rcl-theme-apply", themeId: "", tokens: {} });
      return;
    }
    const resolvedTheme = activeTheme.id === "dark" ? "dark" : "light";
    postToFrames({ type: "rcl-resolved-theme", theme: resolvedTheme });
    postToFrames({
      type: "rcl-theme-apply",
      themeId: activeTheme.id,
      tokens: { ...activeTheme.tokens },
    });
  }, [previewReady, activeTheme, appResolvedTheme, postToFrames]);

  const queryError = builtinThemeQuery.error ?? scenarioThemeQuery.error;
  const loading = builtinThemeQuery.isLoading || scenarioThemeQuery.isLoading;

  return (
    <div
      data-testid={selectors.components.themeSwitcher.root}
      className={cn("flex min-w-0 flex-wrap items-center gap-1.5 text-xs text-app-muted-foreground", className)}
    >
      <label className="flex items-center gap-1.5">
        <span className="text-app-muted-foreground">
          {t(strings.components.themeSwitcher.label)}
        </span>
        <select
          data-testid={selectors.components.themeSwitcher.select}
          value={selection}
          onChange={(e) => setSelection(e.target.value)}
          className="h-7 max-w-32 rounded-md border border-app-border bg-app-surface px-2 text-xs text-app-foreground"
        >
          <option value="">{t(strings.components.themeSwitcher.noneOption)}</option>
          <optgroup label={t(strings.components.themeSwitcher.builtinOptionGroup)}>
            {(builtinsQuery.data?.themes ?? []).map((th) => (
              <option key={th.id} value={`builtin:${th.id}`}>
                {th.name || th.id}
              </option>
            ))}
          </optgroup>
        </select>
      </label>

      <label className="flex items-center gap-1.5">
        <span className="text-app-muted-foreground">
          {t(strings.components.themeSwitcher.scenarioInputLabel)}
        </span>
        <Input
          data-testid={selectors.components.themeSwitcher.scenarioInput}
          value={scenarioId}
          onChange={(e) => setScenarioId(e.target.value)}
          placeholder={t(strings.components.themeSwitcher.scenarioInputPlaceholder)}
          className="h-7 w-36 rounded-md text-xs"
        />
        <Button
          data-testid={selectors.components.themeSwitcher.scenarioApply}
          variant="secondary"
          className="h-7 rounded-md px-2 text-xs"
          disabled={!scenarioId.trim()}
          onClick={() => setSelection(`scenario:${scenarioId.trim()}`)}
        >
          {t(strings.components.themeSwitcher.scenarioApply)}
        </Button>
      </label>

      {loading && (
        <span
          data-testid={selectors.components.themeSwitcher.status}
          className="text-app-muted-foreground"
        >
          {t(strings.components.themeSwitcher.loading)}
        </span>
      )}
      {activeTheme && !loading && (
        <span
          data-testid={selectors.components.themeSwitcher.status}
          className="text-app-success"
        >
          {t(strings.components.themeSwitcher.applied, {
            name: activeTheme.name || activeTheme.id,
          })}
        </span>
      )}
      {queryError && (
        <span
          data-testid={selectors.components.themeSwitcher.error}
          className="text-app-danger"
        >
          {errorMessage(queryError, t)}
        </span>
      )}
    </div>
  );
}
