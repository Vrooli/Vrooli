import { useMemo, useState } from "react";
import { ShieldAlert, XCircle } from "lucide-react";
import { MandateStatus } from "@vrooli/proto-types/treasury/v1/mandate/mandate_pb";

import { cancelStandingMandate, getScenarioFrozen, listOperatorMandates, setScenarioFrozen, type Mandate } from "../api/controls";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Input } from "../components/ui/input";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "../i18n";
import { useTheme, type ThemeChoice } from "../theme/ThemeProvider";

const THEME_CHOICES: readonly ThemeChoice[] = ["light", "dark", "system"];

/**
 * Settings page. Surfaces the locale and theme selectors as a real page (in
 * addition to the compact controls in the top bar). Add scenario-specific
 * preferences here as they're needed.
 */
export function SettingsPage() {
  const { t } = useTranslation();
  const currentLocale = getCurrentLocale();
  const { choice, setTheme } = useTheme();
  const [operatorToken, setOperatorToken] = useState("");
  const [mandates, setMandates] = useState<Mandate[]>([]);
  const [selectedBook, setSelectedBook] = useState("");
  const [scenarioFrozen, setScenarioFrozenState] = useState(false);
  const [controlsReady, setControlsReady] = useState(false);
  const [controlsBusy, setControlsBusy] = useState(false);
  const [controlsError, setControlsError] = useState("");
  const [announcement, setAnnouncement] = useState("");
  const books = useMemo(() => [...new Set(mandates.map((value) => value.bookId).filter(Boolean))].sort(), [mandates]);
  const standing = mandates.filter((value) => value.recurrenceSeconds > 0n && (!selectedBook || value.bookId === selectedBook));

  async function openControls() {
    setControlsBusy(true);
    setControlsError("");
    try {
      const [values, frozen] = await Promise.all([listOperatorMandates(operatorToken), getScenarioFrozen(operatorToken)]);
      setMandates(values);
      setScenarioFrozenState(frozen);
      setSelectedBook((current) => current || values.find((value) => value.bookId)?.bookId || "");
      setControlsReady(true);
    } catch {
      setControlsError(t(strings.controls.authError));
    } finally {
      setControlsBusy(false);
    }
  }

  async function toggleGlobalFreeze() {
    setControlsBusy(true);
    setControlsError("");
    try {
      const frozen = await setScenarioFrozen(operatorToken, !scenarioFrozen);
      setScenarioFrozenState(frozen);
      setAnnouncement(t(frozen ? strings.controls.frozenAnnouncement : strings.controls.unfrozenAnnouncement));
    } catch {
      setControlsError(t(strings.controls.actionError));
    } finally {
      setControlsBusy(false);
    }
  }

  async function cancel(value: Mandate) {
    setControlsBusy(true);
    setControlsError("");
    try {
      const cancelled = await cancelStandingMandate(operatorToken, value.id);
      setMandates((current) => current.map((item) => item.id === cancelled.id ? cancelled : item));
      setAnnouncement(t(strings.controls.cancelledAnnouncement, { mandate: value.id }));
    } catch {
      setControlsError(t(strings.controls.actionError));
    } finally {
      setControlsBusy(false);
    }
  }

  return (
    <section
      data-testid={selectors.pages.settings}
      aria-labelledby="settings-heading"
      className="flex flex-col gap-6"
    >
      <h2 id="settings-heading" className="text-2xl font-semibold">
        {t(strings.pages.settings.title)}
      </h2>

      <div className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.settings.themeHeading)}
        </h3>
        <div role="radiogroup" aria-label={t(strings.theme.switcherLabel)} className="flex flex-wrap gap-2">
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
              {t(strings.theme.choice[c])}
            </Button>
          ))}
        </div>
      </div>

      <div className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.settings.localeHeading)}
        </h3>
        <div role="radiogroup" aria-label={t(strings.locale.switcherLabel)} className="flex flex-wrap gap-2">
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

      <Card className="max-w-4xl" data-testid={selectors.controls.panel}>
        <CardHeader>
          <CardTitle>{t(strings.controls.title)}</CardTitle>
          <p className="text-sm text-app-muted-foreground">{t(strings.controls.description)}</p>
        </CardHeader>
        <CardContent className="grid gap-5">
          <div className="grid gap-2 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-end">
            <div className="space-y-2">
              <label htmlFor="controls-operator-token" className="text-sm font-medium">{t(strings.controls.tokenLabel)}</label>
              <Input id="controls-operator-token" type="password" autoComplete="current-password" value={operatorToken} onChange={(event) => setOperatorToken(event.target.value)} data-testid={selectors.controls.tokenInput} />
            </div>
            <Button type="button" disabled={!operatorToken || controlsBusy} onClick={() => void openControls()} data-testid={selectors.controls.openButton}>{t(strings.controls.open)}</Button>
          </div>

          {controlsReady && <>
            <div className="rounded-control border border-app-danger/40 bg-app-danger/5 p-4">
              <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <h3 className="flex items-center gap-2 font-semibold"><ShieldAlert aria-hidden="true" className="size-5 text-app-danger" />{t(strings.controls.killSwitchTitle)}</h3>
                  <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.controls.killSwitchHelp)}</p>
                </div>
                <Button type="button" variant={scenarioFrozen ? "secondary" : "danger"} disabled={controlsBusy} onClick={() => void toggleGlobalFreeze()} data-testid={selectors.controls.freezeAllButton}>
                  {scenarioFrozen ? t(strings.controls.unfreezeAll) : t(strings.controls.freezeAll)}
                </Button>
              </div>
            </div>

            <div className="grid gap-3">
              <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
                <div>
                  <h3 className="font-semibold">{t(strings.controls.standingTitle)}</h3>
                  <p className="text-sm text-app-muted-foreground">{t(strings.controls.standingHelp)}</p>
                </div>
                <label className="grid gap-1 text-sm font-medium">
                  {t(strings.controls.bookLabel)}
                  <select className="min-h-10 rounded-control border border-app-border bg-app-surface px-3" value={selectedBook} onChange={(event) => setSelectedBook(event.target.value)} data-testid={selectors.controls.bookSelect}>
                    {books.map((book) => <option key={book} value={book}>{book}</option>)}
                  </select>
                </label>
              </div>
              {standing.length === 0 ? <p className="rounded-control border border-app-border p-4 text-sm text-app-muted-foreground">{t(strings.controls.noStanding)}</p> :
                <ul className="grid gap-3">
                  {standing.map((value) => {
                    const cancelled = value.status === MandateStatus.REVOKED || Boolean(value.cancelledAt);
                    return <li key={value.id} className="flex flex-col gap-3 rounded-control border border-app-border p-4 sm:flex-row sm:items-center sm:justify-between" data-testid={selectors.controls.standingItem({ id: value.id })}>
                      <div><p className="font-medium">{value.id}</p><p className="text-sm text-app-muted-foreground">{t(strings.controls.nextCharge)}: {formatTimestamp(value.nextChargeAt)}</p></div>
                      <Button type="button" variant="danger" disabled={cancelled || controlsBusy} onClick={() => void cancel(value)} aria-label={t(strings.controls.cancelAria, { mandate: value.id })}>
                        <XCircle aria-hidden="true" className="size-4" />{cancelled ? t(strings.controls.cancelled) : t(strings.controls.cancel)}
                      </Button>
                    </li>;
                  })}
                </ul>}
            </div>
          </>}
          {controlsError && <p role="alert" className="text-sm font-medium text-app-danger">{controlsError}</p>}
          <p role="status" aria-live="polite" className="sr-only">{announcement}</p>
        </CardContent>
      </Card>
    </section>
  );
}

function formatTimestamp(value: { seconds: bigint } | undefined): string {
  if (!value) return "—";
  return new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(Number(value.seconds) * 1000);
}
