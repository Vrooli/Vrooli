import { useEffect, useState } from "react";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { Select } from "@vrooli/react-component-library/Select/1";
import { SettingsList } from "@vrooli/react-component-library/SettingsList/1";

import { HealthCard } from "../features/health/HealthCard";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation, type Locale } from "../i18n";
import { useTheme, type ThemeChoice } from "../theme/ThemeProvider";

const THEME_CHOICES: readonly ThemeChoice[] = ["light", "dark", "system"];
// Literal references so the strings lint can see every catalog key in use.
const THEME_LABEL_KEY: Record<ThemeChoice, (typeof strings.theme.choice)[ThemeChoice]> = {
  light: strings.theme.choice.light,
  dark: strings.theme.choice.dark,
  system: strings.theme.choice.system,
};

/**
 * Settings owns every preference; nothing else in the shell duplicates one.
 * Appearance and language are the two every scenario has. Add the rest of
 * yours as rows in the same list, grouped by what they change.
 */
export function SettingsPage() {
  const { t } = useTranslation();
  const currentLocale = getCurrentLocale();
  const { choice, setTheme } = useTheme();
  const desktopAuth = typeof window === "undefined" ? undefined : window.desktop?.auth;
  const [accountState, setAccountState] = useState<DeviceStudioDesktopLinkStatus | null>(null);
  const [accountBusy, setAccountBusy] = useState(false);
  const [accountNotice, setAccountNotice] = useState<string | null>(null);
  const [creditStatus, setCreditStatus] = useState<DeviceStudioCreditSessionStatus | null>(null);
  const [creditBusy, setCreditBusy] = useState(false);
  const [creditNotice, setCreditNotice] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    if (!desktopAuth) {
      setAccountState({ state: "unavailable", scopes: [] });
      return () => { active = false; };
    }
    void desktopAuth.getDesktopLinkStatus({ resource: "device-studio" })
      .then((status) => { if (active) setAccountState(status); })
      .catch(() => { if (active) setAccountState({ state: "unavailable", scopes: [] }); });
    return () => { active = false; };
  }, [desktopAuth]);

  useEffect(() => {
    let active = true;
    if (!desktopAuth?.getCreditSessionShareStatus) {
      setCreditStatus({ state: "unavailable" });
      return () => { active = false; };
    }
    void desktopAuth.getCreditSessionShareStatus()
      .then((status) => { if (active) setCreditStatus(status); })
      .catch(() => { if (active) setCreditStatus({ state: "unavailable" }); });
    return () => { active = false; };
  }, [desktopAuth]);

  const accountStateLabel = accountState
    ? t(strings.pages.settings.account.states[accountState.state])
    : t(strings.pages.settings.account.checking);
  const creditStateLabel = creditStatus
    ? t(strings.pages.settings.account.creditStates[creditStatus.state])
    : t(strings.pages.settings.account.checking);

  async function setCreditSharing(enabled: boolean) {
    if (!desktopAuth?.setCreditSessionSharing) return;
    setCreditBusy(true);
    setCreditNotice(null);
    try {
      const status = await desktopAuth.setCreditSessionSharing(enabled);
      setCreditStatus(status);
      if (enabled && status.state === "shared") return;
      if (enabled && status.state === "signed_out") {
        setCreditNotice(t(strings.pages.settings.account.creditStates.signed_out));
      } else if (!enabled && status.state === "not_shared" && status.localCleared) {
        setCreditNotice(t(strings.pages.settings.account.creditRemoteUncertain));
      } else {
        setCreditNotice(t(strings.pages.settings.account.creditError));
      }
    } catch {
      setCreditStatus({ state: "unavailable" });
      setCreditNotice(t(strings.pages.settings.account.creditError));
    } finally {
      setCreditBusy(false);
    }
  }

  async function signInForCredits() {
    if (!desktopAuth?.signIn) return;
    setCreditBusy(true);
    setCreditNotice(null);
    try {
      await desktopAuth.signIn();
      setCreditNotice(t(strings.pages.settings.account.creditSignInStarted));
    } catch {
      setCreditNotice(t(strings.pages.settings.account.creditError));
    } finally {
      setCreditBusy(false);
    }
  }

  async function connectAccount() {
    if (!desktopAuth) return;
    setAccountBusy(true);
    setAccountNotice(null);
    try {
      if (accountState?.state === "linked") {
        const unlink = await desktopAuth.disconnectDesktop({ resource: "device-studio" });
        if (!unlink.localCleared || unlink.remoteRevocation !== "confirmed") {
          setAccountState({ state: "unavailable", scopes: [] });
          setAccountNotice(t(strings.pages.settings.account.remoteUncertain));
          return;
        }
        setAccountState({ state: "unlinked", scopes: [] });
      }
      const status = await desktopAuth.connectDesktop({
        resource: "device-studio",
        audience: "scenario:device-studio",
        scopes: ["device-studio:read"],
      });
      setAccountState(status);
    } catch {
      setAccountNotice(t(strings.pages.settings.account.error));
    } finally {
      setAccountBusy(false);
    }
  }

  async function disconnectAccount() {
    if (!desktopAuth) return;
    setAccountBusy(true);
    setAccountNotice(null);
    try {
      const result = await desktopAuth.disconnectDesktop({ resource: "device-studio" });
      setAccountState({ state: result.localCleared && result.remoteRevocation === "confirmed" ? "unlinked" : "unavailable", scopes: [] });
      setAccountNotice(t(!result.localCleared
        ? strings.pages.settings.account.error
        : result.remoteRevocation === "confirmed"
        ? strings.pages.settings.account.remoteConfirmed
        : strings.pages.settings.account.remoteUncertain));
    } catch {
      setAccountNotice(t(strings.pages.settings.account.error));
    } finally {
      setAccountBusy(false);
    }
  }

  return (
    <section data-testid={selectors.pages.settings} aria-labelledby="settings-heading" className="flex flex-col gap-space-md">
      <PageHeader headingId="settings-heading" title={t(strings.pages.settings.title)} description={t(strings.pages.settings.description)} />
      <SettingsList variant="auto">
        <SettingsList.Group label={t(strings.pages.settings.preferences)}>
          <SettingsList.Row label={t(strings.pages.settings.themeHeading)} hint={t(strings.pages.settings.themeHint)}>
            <Select
              aria-label={t(strings.theme.switcherLabel)}
              data-testid={selectors.settingsPage.themeSelect}
              value={choice}
              onChange={(event) => setTheme(event.target.value as ThemeChoice)}
              options={THEME_CHOICES.map((c) => ({ value: c, label: t(THEME_LABEL_KEY[c]) }))}
            />
          </SettingsList.Row>
          <SettingsList.Row label={t(strings.pages.settings.localeHeading)} hint={t(strings.pages.settings.localeHint)}>
            <Select
              aria-label={t(strings.locale.switcherLabel)}
              data-testid={selectors.settingsPage.localeSelect}
              value={currentLocale}
              onChange={(event) => void setLocale(event.target.value as Locale)}
              options={SUPPORTED_LOCALES.map((lng) => ({ value: lng, label: getLocaleConfig(lng).nativeLabel }))}
            />
          </SettingsList.Row>
        </SettingsList.Group>
      </SettingsList>
      <section data-testid={selectors.settingsPage.accountPanel} aria-labelledby="settings-account-heading" className="flex flex-col gap-space-sm rounded-lg border border-border-subtle p-space-md">
        <div>
          <h2 id="settings-account-heading" className="text-lg font-semibold">{t(strings.pages.settings.account.heading)}</h2>
          <p className="text-sm text-muted-foreground">{t(strings.pages.settings.account.description)}</p>
        </div>
        <p data-testid={selectors.settingsPage.accountStatus} aria-live="polite">
          <span className="font-medium">{t(strings.pages.settings.account.statusLabel)}: </span>{accountStateLabel}
        </p>
        {accountState?.state === "linked" && <dl className="grid gap-space-xs text-sm">
          <div><dt className="font-medium">{t(strings.pages.settings.account.businessAccount)}</dt><dd>{accountState.businessAccountId}</dd></div>
          <div><dt className="font-medium">{t(strings.pages.settings.account.principal)}</dt><dd>{accountState.localPrincipal}</dd></div>
          <div><dt className="font-medium">{t(strings.pages.settings.account.scopes)}</dt><dd>{accountState.scopes.length ? accountState.scopes.join(", ") : t(strings.pages.settings.account.noScopes)}</dd></div>
        </dl>}
        <p className="text-sm">{t(strings.pages.settings.account.localOnly)}</p>
        {accountNotice && <p role="status" aria-live="polite">{accountNotice}</p>}
        <div className="flex flex-wrap gap-space-sm">
          <button type="button" data-testid={selectors.settingsPage.accountConnect} disabled={!desktopAuth || accountBusy || accountState === null} onClick={() => void connectAccount()}>
            {t(accountState?.state === "linked" ? strings.pages.settings.account.switch : strings.pages.settings.account.connect)}
          </button>
          {accountState?.state === "linked" && <button type="button" data-testid={selectors.settingsPage.accountDisconnect} disabled={!desktopAuth || accountBusy} onClick={() => void disconnectAccount()}>
            {t(strings.pages.settings.account.disconnect)}
          </button>}
        </div>
      </section>
      <section data-testid="settings-credit-session" aria-labelledby="settings-credit-heading" className="flex flex-col gap-space-sm rounded-lg border border-border-subtle p-space-md">
        <div>
          <h2 id="settings-credit-heading" className="text-lg font-semibold">{t(strings.pages.settings.account.creditHeading)}</h2>
          <p className="text-sm text-muted-foreground">{t(strings.pages.settings.account.creditDescription)}</p>
        </div>
        <p aria-live="polite"><span className="font-medium">{t(strings.pages.settings.account.creditStatusLabel)}: </span>{creditStateLabel}</p>
        {creditNotice && <p role="status" aria-live="polite">{creditNotice}</p>}
        <div className="flex flex-wrap gap-space-sm">
          {creditStatus?.state === "signed_out" && <button type="button" disabled={!desktopAuth?.signIn || creditBusy} onClick={() => void signInForCredits()}>{t(strings.pages.settings.account.creditSignIn)}</button>}
          {creditStatus?.state !== "shared" && <button type="button" disabled={!desktopAuth?.setCreditSessionSharing || creditBusy || creditStatus === null} onClick={() => void setCreditSharing(true)}>{t(strings.pages.settings.account.creditShare)}</button>}
          {(creditStatus?.state === "shared" || creditStatus?.state === "signed_out" || creditStatus?.state === "unavailable") && <button type="button" disabled={!desktopAuth?.setCreditSessionSharing || creditBusy} onClick={() => void setCreditSharing(false)}>{t(strings.pages.settings.account.creditStop)}</button>}
        </div>
      </section>
      <details className="flex flex-col gap-space-sm">
        <summary className="min-h-11 cursor-pointer py-space-sm font-medium">{t(strings.pages.settings.diagnostics)}</summary>
        <HealthCard />
      </details>
    </section>
  );
}
