import { Button } from "@vrooli/react-component-library/Button/2";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "../i18n";
import { useTheme, type ThemeChoice } from "../theme/ThemeProvider";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { fetchBoard } from "../api/offers";
import { useQuery } from "@tanstack/react-query";

const THEME_CHOICES: readonly ThemeChoice[] = ["light", "dark", "system"];
const THEME_LABEL_KEYS = {
  light: strings.theme.choice.light,
  dark: strings.theme.choice.dark,
  system: strings.theme.choice.system,
};

/**
 * Settings page. Surfaces the locale and theme selectors as a real page (in
 * addition to the compact controls in the top bar). Add scenario-specific
 * preferences here as they're needed.
 */
export function SettingsPage() {
  const { t } = useTranslation();
  const currentLocale = getCurrentLocale();
  const { choice, setTheme } = useTheme();
  const board = useQuery({ queryKey: ["offer-board-settings"], queryFn: fetchBoard, retry: false });
  const ledgerIssues = board.data?.availability.filter((item) => item.source.includes("money-ledger")) ?? [];
  const ledgerConnected = !board.isError && Boolean(board.data?.position) && ledgerIssues.length === 0;
  const ledgerReason = board.error instanceof Error
    ? board.error.message
    : ledgerIssues.map((item) => `${item.source}: ${item.reason}`).join("; ");
  return (
    <ExperienceSurface
      surfaceId="settings"
      state="ready"
      data-testid={selectors.pages.settings}
      aria-labelledby="settings-heading"
      className="flex flex-col gap-6"
    >
      <h2 id="settings-heading" className="text-2xl font-semibold">
        {t(strings.pages.settings.title)}
      </h2>

      <section data-testid="settings-evaluation-schedule" role="group" aria-label={t(strings.pages.dashboard.firedTriggers)} className="rounded-md border p-4">
        <h3 className="font-semibold">{t(strings.pages.dashboard.firedTriggers)}</h3>
        <p>{t(strings.pages.dashboard.evaluationNotRun)}</p>
      </section>
      <p data-testid="settings-ledger-connection" role="status" aria-label={t(strings.pages.dashboard.ledgerPosture)}>{ledgerConnected ? t(strings.pages.settings.ledgerConnected) : t(strings.pages.settings.ledgerDisconnected)}</p>
      <p data-testid="settings-ledger-connection-reason" role={ledgerConnected ? "note" : "alert"} className={ledgerConnected ? "sr-only" : undefined}>{ledgerConnected ? t(strings.pages.dashboard.postureBasis) : t(strings.pages.settings.ledgerConnectionReason, { source: t(strings.pages.dashboard.ledgerPosture), reason: ledgerReason || t(strings.pages.dashboard.postureUnavailable) })}</p>

      <div className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.settings.themeHeading)}
        </h3>
        <div role="radiogroup" aria-label={t(strings.theme.switcherLabel)} data-testid="settings-theme" className="flex flex-wrap gap-2">
          {THEME_CHOICES.map((c) => (
            <Button
              key={c}
              type="button"
              variant={choice === c ? "primary" : "secondary"}
              size="sm"
              role="radio"
              aria-checked={choice === c}
              onClick={() => setTheme(c)}
              data-testid={selectors.settingsPage.themeOption({ choice: c })}
            >
              {t(THEME_LABEL_KEYS[c])}
            </Button>
          ))}
        </div>
      </div>

      <div className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.settings.localeHeading)}
        </h3>
        <div role="radiogroup" aria-label={t(strings.locale.switcherLabel)} data-testid="settings-locale" className="flex flex-wrap gap-2">
          {SUPPORTED_LOCALES.map((lng) => (
            <Button
              key={lng}
              type="button"
              variant={currentLocale === lng ? "primary" : "secondary"}
              size="sm"
              role="radio"
              aria-checked={currentLocale === lng}
              onClick={() => void setLocale(lng)}
              data-testid={selectors.settingsPage.localeOption({ code: lng })}
            >
              {getLocaleConfig(lng).nativeLabel}
            </Button>
          ))}
        </div>
      </div>
    </ExperienceSurface>
  );
}
