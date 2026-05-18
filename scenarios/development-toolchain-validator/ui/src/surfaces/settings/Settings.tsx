import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import {
  SUPPORTED_LOCALES,
  getCurrentLocale,
  getLocaleConfig,
  setLocale,
  useTranslation,
} from "../../i18n";
import { Card, CardHeader, CardTitle } from "../../shared/ui/primitives/Card";
import { PanelHeader } from "../../shared/ui/composites/PanelHeader";
import {
  applyPreferencesToDocument,
  usePreferencesStore,
  type Density,
  type Theme,
} from "../../shared/stores/preferencesStore";

/**
 * Settings surface.
 *
 * Theme, locale, density, sidebar-collapsed controls. The skill-catalog
 * sync and template-watcher sections are placeholder-only — the backend
 * APIs that power them haven't shipped (plan §scope explicitly defers
 * these).
 */
export function Settings() {
  const { t } = useTranslation();
  const theme = usePreferencesStore((s) => s.theme);
  const setTheme = usePreferencesStore((s) => s.setTheme);
  const density = usePreferencesStore((s) => s.density);
  const setDensity = usePreferencesStore((s) => s.setDensity);
  const sidebarCollapsed = usePreferencesStore((s) => s.sidebarCollapsed);
  const setSidebarCollapsed = usePreferencesStore((s) => s.setSidebarCollapsed);

  const currentLocale = getCurrentLocale();

  // After any preference change that affects `<html>` data-attributes, mirror
  // the new state immediately so tokens.css picks it up without a reload.
  const applyAfterMutate = (mutator: () => void) => {
    mutator();
    applyPreferencesToDocument(usePreferencesStore.getState());
  };

  return (
    <section
      data-testid={selectors.settings.surface}
      className="flex max-w-2xl flex-col gap-6"
    >
      <PanelHeader title={t(strings.settings.title)} description={t(strings.settings.subtitle)} />

      <Card surface="raised">
        <CardHeader>
          <CardTitle>{t(strings.settings.themeHeading)}</CardTitle>
        </CardHeader>
        <div className="mt-3 flex gap-2">
          {(
            [
              { value: "dark" as const, label: t(strings.settings.themeDarkLabel), testId: selectors.settings.themeDark },
              { value: "light" as const, label: t(strings.settings.themeLightLabel), testId: selectors.settings.themeLight },
            ] satisfies readonly { value: Theme; label: string; testId: string }[]
          ).map(({ value, label, testId }) => {
            const isActive = theme === value;
            return (
              <button
                key={value}
                type="button"
                data-testid={testId}
                aria-pressed={isActive}
                onClick={() => applyAfterMutate(() => setTheme(value))}
                className={
                  isActive
                    ? "rounded-control bg-app-accent px-3 py-1.5 text-sm font-medium text-app-primary-foreground"
                    : "rounded-control border border-app-border px-3 py-1.5 text-sm text-app-foreground hover:bg-app-surface-muted"
                }
              >
                {label}
              </button>
            );
          })}
        </div>
      </Card>

      <Card surface="raised">
        <CardHeader>
          <CardTitle>{t(strings.settings.localeHeading)}</CardTitle>
        </CardHeader>
        <div className="mt-3 flex gap-2">
          {SUPPORTED_LOCALES.map((lng) => {
            const isActive = currentLocale === lng;
            return (
              <button
                key={lng}
                type="button"
                data-testid={selectors.locale.toggle({ code: lng })}
                aria-pressed={isActive}
                onClick={() => void setLocale(lng)}
                className={
                  isActive
                    ? "rounded-control bg-app-accent px-3 py-1.5 text-sm font-medium text-app-primary-foreground"
                    : "rounded-control border border-app-border px-3 py-1.5 text-sm text-app-foreground hover:bg-app-surface-muted"
                }
              >
                {getLocaleConfig(lng).nativeLabel}
              </button>
            );
          })}
        </div>
      </Card>

      <Card surface="raised">
        <CardHeader>
          <CardTitle>{t(strings.settings.densityHeading)}</CardTitle>
        </CardHeader>
        <div className="mt-3 flex gap-2">
          {(
            [
              {
                value: "comfortable" as const,
                label: t(strings.settings.densityComfortableLabel),
                testId: selectors.settings.densityComfortable,
              },
              {
                value: "compact" as const,
                label: t(strings.settings.densityCompactLabel),
                testId: selectors.settings.densityCompact,
              },
            ] satisfies readonly { value: Density; label: string; testId: string }[]
          ).map(({ value, label, testId }) => {
            const isActive = density === value;
            return (
              <button
                key={value}
                type="button"
                data-testid={testId}
                aria-pressed={isActive}
                onClick={() => applyAfterMutate(() => setDensity(value))}
                className={
                  isActive
                    ? "rounded-control bg-app-accent px-3 py-1.5 text-sm font-medium text-app-primary-foreground"
                    : "rounded-control border border-app-border px-3 py-1.5 text-sm text-app-foreground hover:bg-app-surface-muted"
                }
              >
                {label}
              </button>
            );
          })}
        </div>
      </Card>

      <Card surface="raised">
        <CardHeader>
          <CardTitle>{t(strings.settings.sidebarHeading)}</CardTitle>
        </CardHeader>
        <label className="mt-3 flex items-center gap-2 text-sm text-app-foreground">
          <input
            type="checkbox"
            data-testid={selectors.settings.sidebarCollapsed}
            checked={sidebarCollapsed}
            onChange={(e) => setSidebarCollapsed(e.target.checked)}
            className="h-4 w-4 rounded border-app-border bg-app-surface-input"
          />
          {t(strings.settings.sidebarCollapsed)}
        </label>
      </Card>

      {/* TODO: skill-catalog sync — backend Connect-RPC service not shipped. */}
      <Card surface="muted">
        <CardHeader>
          <CardTitle>{t(strings.settings.catalogSyncHeading)}</CardTitle>
        </CardHeader>
        <p
          data-testid={selectors.settings.catalogSyncPending}
          className="mt-2 text-xs text-app-muted-foreground"
        >
          {t(strings.settings.catalogSyncPending)}
        </p>
      </Card>

      {/* TODO: template-watcher status — backend Connect-RPC service not shipped. */}
      <Card surface="muted">
        <CardHeader>
          <CardTitle>{t(strings.settings.watcherHeading)}</CardTitle>
        </CardHeader>
        <p
          data-testid={selectors.settings.watcherPending}
          className="mt-2 text-xs text-app-muted-foreground"
        >
          {t(strings.settings.watcherPending)}
        </p>
      </Card>
    </section>
  );
}
