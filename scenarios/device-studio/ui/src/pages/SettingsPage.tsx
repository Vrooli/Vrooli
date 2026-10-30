import { useEffect, useState } from "react";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { Select } from "@vrooli/react-component-library/Select/1";
import { SettingsList } from "@vrooli/react-component-library/SettingsList/1";
import { Switch } from "@vrooli/react-component-library/Switch/1";
import { Button } from "@vrooli/react-component-library/Button/2";
import { IntegrationsOverview, type IntegrationItem, type IntegrationStatus } from "@vrooli/react-component-library/IntegrationsOverview/1";
import { Cpu, ShieldCheck } from "lucide-react";

import { HealthCard } from "../features/health/HealthCard";
import { fetchOptionalCapabilities, type CapabilityAvailability } from "../api/capabilities";
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

const C = strings.pages.settings.capabilities;
const A = strings.pages.settings.account;

/** The optional apps Device Studio builds on, in display order. */
const OPTIONAL_SERVICES = [
  { id: "ai-gateway", copy: C.items.aiGateway, provider: "AI Gateway", icon: <Cpu aria-hidden="true" /> },
  { id: "workspace-sandbox", copy: C.items.workspaceSandbox, provider: "Workspace Sandbox", icon: <ShieldCheck aria-hidden="true" /> },
] as const;


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
  const [creditBalance, setCreditBalance] = useState<DeviceStudioCreditBalance | null>(null);
  const [balanceBusy, setBalanceBusy] = useState(false);
  const [optionalCapabilities, setOptionalCapabilities] = useState<CapabilityAvailability[] | null>(null);
  const [capabilityCheckFailed, setCapabilityCheckFailed] = useState(false);

  async function refreshCreditBalance() {
    if (!desktopAuth?.getCreditBalance) {
      setCreditBalance({ state: "unavailable" });
      return;
    }
    setBalanceBusy(true);
    try {
      const next = await desktopAuth.getCreditBalance();
      setCreditBalance(next.state === "available" || creditBalance?.state !== "available"
        ? next : { ...creditBalance, state: "stale" });
    } catch {
      setCreditBalance(creditBalance?.state === "available" ? { ...creditBalance, state: "stale" } : { state: "unavailable" });
    }
    finally { setBalanceBusy(false); }
  }

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

  useEffect(() => { void refreshCreditBalance(); }, [desktopAuth]);

  const [capabilityCheckCount, setCapabilityCheckCount] = useState(0);
  const [checkingCapabilities, setCheckingCapabilities] = useState(true);
  useEffect(() => {
    let active = true;
    setCheckingCapabilities(true);
    void fetchOptionalCapabilities().then((states) => {
      if (!active) return;
      setOptionalCapabilities(states);
      setCapabilityCheckFailed(false);
    }).catch(() => {
      if (active) setCapabilityCheckFailed(true);
    }).finally(() => {
      if (active) setCheckingCapabilities(false);
    });
    return () => { active = false; };
  }, [capabilityCheckCount]);

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

  const balanceAvailable = creditBalance !== null && (creditBalance.state === "available" || creditBalance.state === "stale");
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

  const optionalServiceItem = (service: (typeof OPTIONAL_SERVICES)[number], states: CapabilityAvailability[] | null): IntegrationItem => {
    const state = states?.find((candidate) => candidate.id === service.id);
    const status: IntegrationStatus = state?.status === "available" ? "ready" : state?.status === "unavailable" ? "off" : states === null ? "checking" : "unknown";
    const name = t(service.copy.name);
    return {
      id: service.id,
      name,
      subtitle: t(service.copy.subtitle),
      provider: service.provider,
      icon: service.icon,
      status,
      problem: status === "off" ? { title: t(C.problemOff, { name }), description: t(service.copy.problem) } : undefined,
      features: [{ id: service.id, label: t(service.copy.feature), available: status !== "off" }],
      facts: [
        { label: t(C.facts.id), value: service.id, monospace: true },
        ...(state?.message ? [{ label: t(C.facts.health), value: state.message }] : []),
        ...(state?.checkedAt ? [{ label: t(C.facts.checked), value: new Date(state.checkedAt).toLocaleString(getCurrentLocale()) }] : []),
        ...(status === "off" && state?.operatorCommand ? [{ label: t(C.facts.command), value: state.operatorCommand, monospace: true }] : []),
      ],
    };
  };

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
      <SettingsList variant="auto" data-testid={selectors.settingsPage.accountPanel}>
        <SettingsList.Group label={t(A.heading)}>
          <SettingsList.Row
            label={t(A.statusLabel)}
            hint={<>
              <span data-testid={selectors.settingsPage.accountStatus} aria-live="polite" className="block">{desktopAuth ? accountStateLabel : t(A.desktopOnly)}</span>
              {accountNotice
                ? <span role="status" aria-live="polite" className="block">{accountNotice}</span>
                : accountState?.state !== "linked" && <span className="block">{t(A.description)}</span>}
            </>}
          >
            {/* Outside the desktop app there is nothing to connect, so no dead controls are shown. */}
            {desktopAuth && <div className="flex flex-wrap justify-end gap-space-xs">
              <Button size="sm" variant={accountState?.state === "linked" ? "outline" : "primary"} data-testid={selectors.settingsPage.accountConnect} disabled={accountBusy || accountState === null} onClick={() => void connectAccount()}>
                {t(accountState?.state === "linked" ? A.switch : A.connect)}
              </Button>
              {accountState?.state === "linked" && <Button size="sm" variant="outline" data-testid={selectors.settingsPage.accountDisconnect} disabled={accountBusy} onClick={() => void disconnectAccount()}>
                {t(A.disconnect)}
              </Button>}
            </div>}
          </SettingsList.Row>
          {accountState?.state === "linked" && <>
            <SettingsList.Row label={t(A.businessAccount)}><span className="text-sm text-muted-foreground">{accountState.businessAccountId}</span></SettingsList.Row>
            <SettingsList.Row label={t(A.principal)}><span className="text-sm text-muted-foreground">{accountState.localPrincipal}</span></SettingsList.Row>
            <SettingsList.Row label={t(A.scopes)}><span className="text-sm text-muted-foreground">{accountState.scopes.length ? accountState.scopes.join(", ") : t(A.noScopes)}</span></SettingsList.Row>
          </>}
        </SettingsList.Group>
      </SettingsList>
      <SettingsList variant="auto" data-testid="settings-credit-session">
        <SettingsList.Group label={t(A.creditHeading)}>
          <SettingsList.Row
            label={t(A.creditStatusLabel)}
            hint={<>
              <span className="block">{t(desktopAuth ? A.creditDescription : A.desktopOnlyCredits)}</span>
              {desktopAuth && creditStatus && creditStatus.state !== "shared" && creditStatus.state !== "not_shared" && <span className="block">{creditStateLabel}</span>}
              {creditNotice && <span role="status" aria-live="polite" className="block">{creditNotice}</span>}
            </>}
          >
            {desktopAuth && <div className="flex flex-wrap items-center justify-end gap-space-xs">
              {creditStatus?.state === "signed_out" && desktopAuth.signIn
                ? <Button size="sm" variant="primary" disabled={creditBusy} onClick={() => void signInForCredits()}>{t(A.creditSignIn)}</Button>
                : <Switch
                    data-testid="settings-credit-share"
                    aria-label={t(A.creditStatusLabel)}
                    checked={creditStatus?.state === "shared"}
                    disabled={!desktopAuth.setCreditSessionSharing || creditBusy || creditStatus === null || creditStatus.state === "signed_out"}
                    onCheckedChange={(checked) => void setCreditSharing(checked)}
                  />}
              {desktopAuth.setCreditSessionSharing && (creditStatus?.state === "signed_out" || creditStatus?.state === "unavailable") && <Button size="sm" variant="ghost" disabled={creditBusy} onClick={() => void setCreditSharing(false)}>{t(A.creditStop)}</Button>}
            </div>}
          </SettingsList.Row>
          {desktopAuth && <SettingsList.Row
            label={t(A.balanceLabel)}
            hint={<span data-testid="settings-credit-balance" aria-live="polite">
              <span className="block">{balanceAvailable
                ? t(A.balanceValue, { amount: creditBalance.balanceCredits, label: creditBalance.displayCreditsLabel })
                : creditBalance === null ? t(A.balanceChecking) : creditBalance.state === "signed_out" ? t(A.balanceSignedOut) : t(A.balanceUnavailable)}</span>
              {creditBalance?.state === "stale" && <span role="status" className="block">{t(A.balanceStale)}</span>}
              {balanceAvailable && creditBalance.state === "available" && creditBalance.fetchedAt && <span className="block">{t(A.balanceChecked, { time: new Date(creditBalance.fetchedAt).toLocaleString(currentLocale, { dateStyle: "medium", timeStyle: "short" }) })}</span>}
            </span>}
          >
            {desktopAuth.getCreditBalance && <Button size="sm" variant="outline" disabled={balanceBusy} onClick={() => void refreshCreditBalance()}>
              {balanceBusy ? t(A.balanceRefreshing) : t(A.balanceRefresh)}
            </Button>}
          </SettingsList.Row>}
        </SettingsList.Group>
      </SettingsList>
      {balanceAvailable && <details data-testid="settings-credit-balance-details" className="-mt-space-sm px-space-md text-sm">
        <summary className="min-h-11 cursor-pointer py-space-sm text-muted-foreground">{t(A.balanceDetails)}</summary>
        <dl className="grid grid-cols-[auto_1fr] gap-x-space-md gap-y-space-xs pb-space-sm">
          <dt className="text-muted-foreground">{t(A.balanceSource)}</dt><dd>{creditBalance.source}</dd>
          <dt className="text-muted-foreground">{t(A.balanceMultiplier)}</dt><dd>{creditBalance.displayCreditsMultiplier}</dd>
          {creditBalance.fetchedAt && <><dt className="text-muted-foreground">{t(A.balanceFetched)}</dt><dd>{new Date(creditBalance.fetchedAt).toLocaleString(currentLocale, { dateStyle: "medium", timeStyle: "short" })}</dd></>}
          {creditBalance.walletUpdatedAt && <><dt className="text-muted-foreground">{t(A.balanceWalletUpdated)}</dt><dd>{new Date(creditBalance.walletUpdatedAt).toLocaleString(currentLocale, { dateStyle: "medium", timeStyle: "short" })}</dd></>}
        </dl>
      </details>}
      <section data-testid="settings-optional-capabilities" aria-label={t(C.heading)}>
        <IntegrationsOverview
          header={<SettingsList.Intro title={t(C.heading)} description={t(C.description)} />}
          groups={[{ id: "apps", label: t(C.groupApps), items: OPTIONAL_SERVICES.map((service) => optionalServiceItem(service, optionalCapabilities)) }]}
          checkedAt={optionalCapabilities?.find((state) => state.checkedAt)?.checkedAt}
          loading={optionalCapabilities === null && !capabilityCheckFailed}
          error={capabilityCheckFailed ? t(C.checkFailed) : null}
          refreshing={checkingCapabilities && optionalCapabilities !== null}
          onRefresh={() => setCapabilityCheckCount((count) => count + 1)}
          locale={getCurrentLocale()}
        />
      </section>
      <details className="flex flex-col gap-space-sm">
        <summary className="min-h-11 cursor-pointer py-space-sm font-medium">{t(strings.pages.settings.diagnostics)}</summary>
        <HealthCard />
      </details>
    </section>
  );
}
