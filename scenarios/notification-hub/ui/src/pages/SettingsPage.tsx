import { Button } from "@vrooli/react-component-library/Button/2";
import { Avatar } from "@vrooli/react-component-library/Avatar/1";
import { SettingsList } from "@vrooli/react-component-library/SettingsList/1.0.1";
import { Switch } from "@vrooli/react-component-library/Switch/1.0.13";
import { Bell, BellOff, RefreshCw } from "lucide-react";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "../i18n";
import { usePreferenceStorageUnavailable, useTheme, type ThemeChoice } from "../theme/ThemeProvider";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { fetchNotificationCatalog, updateNotificationPreference, MIN_URGENCY_CHOICES, type CatalogCategory, type CatalogSource, type NotificationPreference } from "../api/preferences";
import { useCallback, useEffect, useRef, useState } from "react";
import { useSession } from "../features/session/SessionGate";
import { SessionPanel } from "../features/session/SessionGate";
import { Link, useLocation } from "react-router-dom";
import { useQuietEditorHeader } from "../layout/AppShell";
import { preserveRoutedTestMode } from "../features/routedTestMode";
import { QuietHoursPanel } from "../features/settings/QuietHoursPanel";
import { EscalationOrderPanel } from "../features/settings/EscalationOrderPanel";

function preserveSettingsTestContext(path: string, search: string): string {
  const destination = preserveRoutedTestMode(path, search);
  const current = new URLSearchParams(search);
  if (current.get("vrooli_test_mode") !== "1") return destination;
  const target = new URL(destination, "http://notification-hub.local");
  for (const key of ["e21_fixture", "e22_fixture"] as const) {
    const value = current.get(key);
    if (value) target.searchParams.set(key, value);
  }
  return `${target.pathname}${target.search}${target.hash}`;
}

const THEME_CHOICES: readonly ThemeChoice[] = ["light", "dark", "system"];

/**
 * Settings page. Surfaces the locale and theme selectors as a real page (in
 * addition to the compact controls in the top bar). Add scenario-specific
 * preferences here as they're needed.
 */
export function SettingsPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { choice, setTheme } = useTheme();
  const preferenceStorageUnavailable = usePreferenceStorageUnavailable();
  const location = useLocation();
  const [focusedSettingsPanel, setFocusedSettingsPanel] = useState<"quiet" | "order" | null>(() => new URLSearchParams(location.search).get("panel") === "quiet" ? "quiet" : null);
  const setQuietEditorHeader = useQuietEditorHeader();
  const quietPanelWasFocused = useRef(false);
  const handleQuietFocus = useCallback((focused: boolean) => {
    setFocusedSettingsPanel(focused ? "quiet" : null);
    if (focused) {
      quietPanelWasFocused.current = true;
      return;
    }
    if (!quietPanelWasFocused.current) return;
    quietPanelWasFocused.current = false;
    const restoreFocus = () => document.querySelector<HTMLElement>(`[data-testid="${selectors.quietHours.openWeek}"]`)?.focus({ preventScroll: true });
    if (typeof window.requestAnimationFrame === "function") window.requestAnimationFrame(restoreFocus);
    else window.setTimeout(restoreFocus, 0);
  }, []);
  const handleQuietHeaderChange = useCallback((header: Parameters<NonNullable<typeof setQuietEditorHeader>>[0]) => {
    setQuietEditorHeader?.(header);
  }, [setQuietEditorHeader]);
  const handleOrderFocus = useCallback((focused: boolean) => setFocusedSettingsPanel(focused ? "order" : null), []);
  useEffect(() => {
    if (focusedSettingsPanel !== "quiet") return;
    const main = document.getElementById(`${selectors.layout.shell}-main`);
    if (main) main.scrollTop = 0;
  }, [focusedSettingsPanel]);
  const session = useSession();
  const signedIn = Boolean(session.data?.signedIn);
  const subject = session.data?.subject ?? "";
  const [signInOpen, setSignInOpen] = useState(false);
  const [sessionExpired, setSessionExpired] = useState(false);
  const [expiredSettingsSubject, setExpiredSettingsSubject] = useState("");
  useEffect(() => {
    const onExpired = () => {
      setExpiredSettingsSubject(subject);
      setSessionExpired(true);
      setSignInOpen(false);
    };
    window.addEventListener("notification-hub:session-expired", onExpired);
    return () => window.removeEventListener("notification-hub:session-expired", onExpired);
  }, [subject]);
  useEffect(() => {
    if (signedIn) setSessionExpired(false);
  }, [signedIn]);
  const requestedView = new URLSearchParams(location.search).get("view");
  const allSettingsView = requestedView === "all";
  const preferencesView = requestedView === "preferences";
  const retainingProtectedSettings = sessionExpired && Boolean(expiredSettingsSubject) && allSettingsView;

  // Catalog and recipient preference views are ordinary configured-recipient
  // reads. Preference changes remain behind the owner identity check.
  const catalog = useQuery({
    queryKey: ["notification-catalog"],
    queryFn: fetchNotificationCatalog,
    enabled: !sessionExpired && ((signedIn && !preferencesView) || allSettingsView),
  });
  const preferenceUpdate = useMutation({ mutationFn: updateNotificationPreference, onSuccess: () => void catalog.refetch() });
  const prefIndex = new Map<string, NotificationPreference>();
  for (const p of catalog.data?.preferences ?? []) prefIndex.set(`${p.source_scenario}\u0000${p.category}`, p);
  const effectivePref = (sourceScenario: string, category: string, defaultEnabled: boolean, defaultMinUrgency: string): NotificationPreference => {
    const stored = prefIndex.get(`${sourceScenario}\u0000${category}`);
    if (stored) return stored;
    return { source_scenario: sourceScenario, category, enabled: defaultEnabled, min_urgency: defaultMinUrgency || "low", channel_override: "" };
  };
  const savePref = (next: NotificationPreference) => preferenceUpdate.mutate(next);

  if (sessionExpired && !retainingProtectedSettings) {
    return <SessionExpiredRecovery draftRetained={false} signInOpen={signInOpen} onSignIn={() => setSignInOpen(true)} onBack={() => setSignInOpen(false)} onPreferences={() => { setSessionExpired(false); setSignInOpen(false); }} />;
  }

  if (((!signedIn && !session.isPending && !allSettingsView) || (preferencesView && (signedIn || session.isPending))) && !retainingProtectedSettings) {
    const ownerRequired = !preferencesView && !signInOpen;
    const preferences = preferencesView && !signInOpen;
    const signIn = signInOpen;
    return (
      <section
        data-testid={selectors.pages.settings}
        aria-label={t(ownerRequired ? strings.settings.ownerRequiredLabel : preferences ? signedIn ? strings.settings.preferencesSessionTitle : strings.settings.appearanceLanguageTitle : strings.session.title)}
        className="-mx-4 -mt-4 flex min-h-[calc(100dvh-3.5rem)] flex-col sm:-mx-8"
      >
        <div className="mx-auto w-full max-w-3xl px-4 pt-5 sm:px-8">
          <Link to={preferences || signIn ? "/settings" : "/settings?view=all"} className="text-sm font-medium underline" onClick={() => setSignInOpen(false)}>
            ← {t(strings.pages.settings.title)}
          </Link>
        </div>

        <div className="mx-auto flex w-full max-w-3xl flex-1 flex-col gap-5 px-4 py-5 sm:px-8 sm:py-8 sm:gap-6">
          {ownerRequired && <>
            <div className="flex flex-col gap-4">
              <p className="text-sm text-app-muted-foreground">{t(strings.settings.ownerRequiredLabel)}</p>
              <h1 className="text-[28px] font-bold leading-tight tracking-tight sm:text-5xl">{t(strings.settings.ownerRequiredHeading)}</h1>
              <p className="text-base leading-relaxed sm:text-2xl">{t(strings.settings.ownerRequiredBody)}</p>
            </div>
            <Link to="/settings?view=preferences" className="flex items-center justify-between gap-4 border-b border-app-border py-5 text-start">
              <span className="flex flex-col gap-1">
                <span className="text-lg font-bold sm:text-2xl">{t(strings.settings.appearanceLanguageTitle)}</span>
                <span className="text-sm text-app-muted-foreground sm:text-lg">{t(strings.settings.appearanceLanguageBody)}</span>
              </span>
              <span aria-hidden="true" className="shrink-0 text-2xl text-app-muted-foreground">›</span>
            </Link>
            <aside className="border-s-4 border-app-warning ps-4 py-3">
              <p className="text-base font-bold">{t(strings.settings.returnAfterSignInTitle)}</p>
              <p className="mt-2 text-base leading-relaxed">{t(strings.settings.returnAfterSignInBody)}</p>
            </aside>
          </>}

          {preferences && <>
            <p className="text-sm text-app-muted-foreground">
              {t(signedIn ? strings.settings.preferencesSessionIntro : strings.settings.browserOnly)}
            </p>
            <h1 className="text-3xl font-bold tracking-tight sm:text-4xl">
              {t(signedIn ? strings.settings.preferencesSessionTitle : strings.settings.appearanceLanguageTitle)}
            </h1>
            <AppearanceLanguageControls
              choice={choice}
              setTheme={setTheme}
              preferenceStorageUnavailable={preferenceStorageUnavailable}
              layout="select"
            />
            {!preferenceStorageUnavailable && <>
              <section className="flex flex-col gap-2 border-t border-app-border pt-5" aria-labelledby="browser-session-heading">
                <h2 id="browser-session-heading" className="text-sm font-semibold uppercase text-app-muted-foreground">
                  {t(strings.session.sectionTitle)}
                </h2>
                {signedIn ? <SessionPanel /> : session.isPending ? null : <p role="status" className="text-sm text-app-muted-foreground">{t(strings.session.hubSessionInactive)}</p>}
                <p className="text-sm text-app-muted-foreground">{t(strings.session.draftsBrowserOnly)}</p>
              </section>
              {!signedIn && !session.isPending && <Button className="min-h-12 w-full sm:w-fit" type="button" variant="outline" onClick={() => setSignInOpen(true)}>
                {t(strings.settings.openSignIn)}
              </Button>}
            </>}
          </>}

          {signIn && <>
            <button type="button" onClick={() => setSignInOpen(false)} className="w-fit text-sm font-medium underline">
              ← {t(strings.settings.backToOwnerRequired)}
            </button>
            <SessionPanel />
          </>}
        </div>

        {preferences && <footer className="sticky bottom-0 border-t border-app-border bg-app-surface-muted px-4 py-3 pb-[calc(0.75rem+env(safe-area-inset-bottom))] sm:px-8">
          <Link to="/settings?view=all" className="mx-auto flex min-h-12 w-full max-w-3xl items-center justify-center rounded-lg border border-app-border bg-transparent px-4 py-3 text-center font-medium hover:bg-app-surface-raised">
            {t(preferenceStorageUnavailable ? strings.settings.continueToSettings : strings.reach.backToSettings)}
          </Link>
        </footer>}

        {ownerRequired && <footer className="sticky bottom-0 flex gap-3 border-t border-app-border bg-app-surface-muted px-4 py-3 pb-[calc(0.75rem+env(safe-area-inset-bottom))] sm:px-8">
              <Button className="min-h-12 flex-1" type="button" variant="primary" onClick={() => setSignInOpen(true)}>
            {t(strings.settings.openSignIn)}
          </Button>
              <Link to="/settings?view=preferences" className="inline-flex min-h-12 flex-1 items-center justify-center rounded-lg border border-app-border px-4 py-3 text-center font-medium hover:bg-app-surface-raised">
            {t(strings.settings.openPreferences)}
              </Link>
        </footer>}
      </section>
    );
  }

  return (
    <section
      data-testid={selectors.pages.settings}
      aria-label={focusedSettingsPanel === "quiet" ? t(strings.settings.ownerSettingsTitle) : focusedSettingsPanel === "order" ? t(strings.settings.ownerSettingsTitle) : t(strings.settings.preferencesSessionTitle)}
      className={focusedSettingsPanel === "quiet" || focusedSettingsPanel === "order" ? "flex min-h-0 flex-1 flex-col" : "flex flex-col gap-6"}
    >
    <div
      hidden={sessionExpired}
      aria-hidden={sessionExpired}
      style={sessionExpired ? { display: "none" } : undefined}
      key={retainingProtectedSettings ? expiredSettingsSubject : subject}
      className={focusedSettingsPanel === "quiet" || focusedSettingsPanel === "order" ? "flex min-h-0 flex-1 flex-col" : "flex flex-col gap-6"}
    >
    <div hidden={focusedSettingsPanel !== null} className={focusedSettingsPanel === null ? "flex flex-col gap-6" : "hidden"}>
      <h2 id="settings-heading" className="text-2xl font-semibold">
        {t(strings.settings.preferencesSessionTitle)}
      </h2>
      <p className="text-sm text-app-muted-foreground">{t(strings.settings.preferencesSessionIntro)}</p>

      <AppearanceLanguageControls
        choice={choice}
        setTheme={setTheme}
        preferenceStorageUnavailable={preferenceStorageUnavailable}
      />

      <section className="flex flex-col gap-2" aria-labelledby="settings-session-heading">
        <h3 id="settings-session-heading" className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.session.sectionTitle)}
        </h3>
        <SessionPanel />
        <p className="text-sm text-app-muted-foreground">{t(strings.session.draftsBrowserOnly)}</p>
      </section>

      <section className="flex flex-col gap-2" aria-labelledby="reach-settings-heading">
        <h3 id="reach-settings-heading" className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.reach.title)}
        </h3>
        <p className="text-sm text-app-muted-foreground">{t(strings.reach.settingsSummary)}</p>
        <Link to={preserveSettingsTestContext("/settings/reach", location.search)} onClick={() => queryClient.removeQueries({ queryKey: ["recipient-devices"], exact: true })} className="inline-flex w-fit items-center rounded-lg border border-app-border px-4 py-2 font-medium hover:bg-app-surface-raised">
          {t(strings.reach.openSettings)}
        </Link>
      </section>

      {!signedIn && <section role="status" className="rounded-panel border border-app-border p-4">
        <h3 className="text-lg font-semibold">{t(strings.settings.ownerSettingsTitle)}</h3>
        <p className="text-sm text-app-muted-foreground">{t(strings.settings.ownerSettingsSignIn)}</p>
      </section>}

      <div className="flex flex-col gap-3">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">{t(strings.settings.integrationTitle)}</h3>
        <aside role="status" aria-label={`${t(strings.settings.integrationTitle)}: ${t(strings.settings.integrationLockedTitle)}`} className="rounded-panel border border-app-border bg-app-surface-raised p-4">
          <p className="font-medium">{t(strings.settings.integrationLockedTitle)}</p>
          <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.settings.integrationLockedBody)}</p>
          <p className="mt-2 text-sm text-app-muted-foreground">{t(strings.settings.integrationUnchanged)}</p>
        </aside>
      </div>

      <div className="flex flex-col gap-3">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">{t(strings.settings.notificationPreferencesTitle)}</h3>
        <p className="text-sm text-app-muted-foreground">{t(strings.settings.notificationPreferencesIntro)}</p>
        {!signedIn && <p className="text-sm text-app-muted-foreground">{t(strings.settings.notificationPreferencesSignIn)}</p>}

        {catalog.isError ? (
          <NotificationsMessage
            testId="notifications-error"
            icon={<BellOff className="h-6 w-6" aria-hidden="true" />}
            title="Can't reach notifications right now"
            body="Your preferences are safe. This is usually a brief hiccup reaching the hub — try again in a moment."
          >
            <SecondaryButton
              testId="notifications-retry"
              onClick={() => void catalog.refetch()}
              icon={<RefreshCw className={`h-4 w-4 ${catalog.isFetching ? "animate-spin" : ""}`} aria-hidden="true" />}
            >
              Retry
            </SecondaryButton>
          </NotificationsMessage>
        ) : catalog.isLoading ? (
          <NotificationsSkeleton />
        ) : (catalog.data?.sources.length ?? 0) === 0 ? (
          <NotificationsMessage
            testId="notifications-empty"
            icon={<Bell className="h-6 w-6" aria-hidden="true" />}
            title="No notifications to manage yet"
            body="Once an app you use starts sending you notifications, it'll appear here so you can fine-tune what reaches you."
          >
            <SecondaryButton
              testId="notifications-refresh"
              onClick={() => void catalog.refetch()}
              icon={<RefreshCw className={`h-4 w-4 ${catalog.isFetching ? "animate-spin" : ""}`} aria-hidden="true" />}
            >
              Refresh
            </SecondaryButton>
          </NotificationsMessage>
        ) : (
          catalog.data?.sources.map((source: CatalogSource) => {
            const sourceName = source.display_name || source.source_scenario;
            return (
              <SettingsList
                key={source.source_scenario}
                data-testid={`notification-source-${source.source_scenario}`}
                variant="auto"
                density="comfortable"
              >
                <SettingsList.Group label={(
                  <div className="flex items-center gap-2.5">
                    <Avatar name={sourceName} size="xs" shape="rounded" />
                    <h4 className="text-sm font-semibold text-app-foreground">{sourceName}</h4>
                  </div>
                )}>
                  {categoriesFor(source).map((category) => {
                    const label = category.display_name || category.category || `All ${sourceName} notifications`;
                    const pref = effectivePref(source.source_scenario, category.category, category.default_enabled, category.default_min_urgency);
                    return (
                      <SettingsList.Row
                        key={category.category}
                        label={label}
                        hint={category.description || undefined}
                        control="wide"
                      >
                        <div className="flex min-w-0 flex-col gap-2 sm:flex-row sm:items-center sm:justify-end sm:gap-4">
                          <Switch
                            checked={pref.enabled}
                            disabled={preferenceUpdate.isPending || !signedIn}
                            aria-label={t(strings.settings.notificationPreferenceEnable, { category: label })}
                            onCheckedChange={(enabled) => savePref({ ...pref, enabled })}
                          />
                          <label className="flex min-h-11 items-center justify-between gap-3 text-sm sm:justify-start">
                            <span>{t(strings.settings.notificationPreferenceMinimumUrgency)}</span>
                            <select
                              className="min-h-11 rounded-control border border-app-border bg-app-background px-3"
                              aria-label={`${label} · ${t(strings.settings.notificationPreferenceMinimumUrgency)}`}
                              value={pref.min_urgency || "low"}
                              disabled={preferenceUpdate.isPending || !signedIn}
                              onChange={(event) => savePref({ ...pref, min_urgency: event.target.value })}
                            >
                              {MIN_URGENCY_CHOICES.map((choice) => (
                                <option key={choice} value={choice}>{t(strings.settings.notificationUrgency[choice])}</option>
                              ))}
                            </select>
                          </label>
                        </div>
                      </SettingsList.Row>
                    );
                  })}
                </SettingsList.Group>
              </SettingsList>
            );
          })
        )}
        {signedIn && preferenceUpdate.isError && <p aria-live="polite" className="text-sm text-app-muted-foreground">Notification preference update failed.</p>}
      </div>
    </div>
    {(signedIn || retainingProtectedSettings) && <>
      <div hidden={focusedSettingsPanel === "order"} className={focusedSettingsPanel === "quiet" ? "flex min-h-0 flex-1 flex-col" : undefined}>
        <QuietHoursPanel
          focused={focusedSettingsPanel === "quiet"}
          onFocusedChange={handleQuietFocus}
          backToSettingsLabel={t(strings.pages.settings.title)}
          onEditorHeaderChange={setQuietEditorHeader && !sessionExpired ? handleQuietHeaderChange : undefined}
        />
      </div>
      <div hidden={focusedSettingsPanel === "quiet"} className={focusedSettingsPanel === "order" ? "flex min-h-0 flex-1 flex-col" : undefined}>
        <EscalationOrderPanel
          focused={focusedSettingsPanel === "order"}
          onFocusedChange={handleOrderFocus}
          onEditorHeaderChange={setQuietEditorHeader && !sessionExpired ? handleQuietHeaderChange : undefined}
        />
      </div>
    </>}
    </div>
    {retainingProtectedSettings && <SessionExpiredRecovery draftRetained signInOpen={signInOpen} onSignIn={() => setSignInOpen(true)} onBack={() => setSignInOpen(false)} onPreferences={() => { setSessionExpired(false); setSignInOpen(false); }} />}
    </section>
  );
}

function SessionExpiredRecovery({
  draftRetained,
  signInOpen,
  onSignIn,
  onBack,
  onPreferences,
}: {
  draftRetained: boolean;
  signInOpen: boolean;
  onSignIn: () => void;
  onBack: () => void;
  onPreferences: () => void;
}) {
  const { t } = useTranslation();
  return (
    <section
      aria-label={t(strings.settings.sessionExpiredTitle)}
      className="-mx-4 -mt-4 flex min-h-[calc(100dvh-3.5rem)] flex-col sm:-mx-8"
    >
      <div className="mx-auto flex w-full max-w-3xl flex-col gap-5 px-4 py-5 sm:px-8 sm:py-8 sm:gap-6">
        {signInOpen ? (
          <>
            <button type="button" onClick={onBack} className="w-fit text-sm font-medium underline">
              ← {t(strings.settings.sessionExpiredTitle)}
            </button>
            <SessionPanel />
          </>
        ) : (
          <>
            <p className="text-sm text-app-muted-foreground">{t(strings.settings.sessionExpiredEyebrow)}</p>
            <h1 className="text-3xl font-bold tracking-tight sm:text-4xl">{t(strings.settings.sessionExpiredTitle)}</h1>
            <aside className="border-s-4 border-app-warning ps-4 py-3">
              <p className="text-lg font-bold">{t(draftRetained ? strings.settings.sessionExpiredDraftTitle : strings.settings.sessionExpiredRecoveryTitle)}</p>
              <p className="mt-2 text-base leading-relaxed">{t(draftRetained ? strings.settings.sessionExpiredDraftBody : strings.settings.sessionExpiredRecoveryBody)}</p>
            </aside>
            <p className="text-base leading-relaxed">{t(strings.settings.sessionExpiredPrivacyBody)}</p>
            <p className="text-sm text-app-muted-foreground">{t(strings.settings.sessionExpiredOtherOwner)}</p>
          </>
        )}
      </div>
      {!signInOpen && <footer className="sticky bottom-0 flex gap-3 border-t border-app-border bg-app-surface-muted px-4 py-3 pb-[calc(0.75rem+env(safe-area-inset-bottom))] sm:px-8">
        <Button className="min-h-12 flex-1 rounded-lg" type="button" variant="primary" onClick={onSignIn}>
          {t(strings.settings.openSignIn)}
        </Button>
        <Link to="/settings?view=preferences" onClick={onPreferences} className="flex min-h-12 flex-1 items-center justify-center rounded-lg border border-app-border px-4 text-center font-medium">
          {t(strings.settings.openPreferences)}
        </Link>
      </footer>}
    </section>
  );
}

function AppearanceLanguageControls({
  choice,
  setTheme,
  preferenceStorageUnavailable,
  layout = "radio",
}: {
  choice: ThemeChoice;
  setTheme: (choice: ThemeChoice) => void;
  preferenceStorageUnavailable: boolean;
  layout?: "radio" | "select";
}) {
  const { t } = useTranslation();
  const currentLocale = getCurrentLocale();
  if (layout === "select") {
    return (
      <div className="flex flex-col gap-5">
        <div className="flex flex-col gap-2">
          <label htmlFor="settings-appearance" className="text-base font-semibold">{t(strings.pages.settings.themeHeading)}</label>
          <select
            id="settings-appearance"
            aria-label={t(strings.theme.switcherLabel)}
            className="min-h-12 w-full rounded-xl border-2 border-app-border bg-app-surface-raised px-4 text-base text-app-foreground"
            value={choice}
            onChange={(event) => setTheme(event.currentTarget.value as ThemeChoice)}
          >
            {THEME_CHOICES.map((theme) => <option key={theme} value={theme}>{t(strings.theme.choice[theme])}</option>)}
          </select>
        </div>
        <div className="flex flex-col gap-2">
          <label htmlFor="settings-language" className="text-base font-semibold">{t(strings.pages.settings.displayLanguageHeading)}</label>
          <select
            id="settings-language"
            aria-label={t(strings.locale.switcherLabel)}
            className="min-h-12 w-full rounded-xl border-2 border-app-border bg-app-surface-raised px-4 text-base text-app-foreground"
            value={currentLocale}
            onChange={(event) => void setLocale(event.currentTarget.value as (typeof SUPPORTED_LOCALES)[number])}
          >
            {SUPPORTED_LOCALES.map((locale) => <option key={locale} value={locale}>{getLocaleConfig(locale).nativeLabel}</option>)}
          </select>
        </div>
        {preferenceStorageUnavailable && (
          <>
            <aside role="status" aria-live="polite" aria-atomic="true" className="border-s-4 border-app-warning ps-4 py-3">
              <p className="font-semibold">{t(strings.preferenceStorage.title)}</p>
              <p>{t(strings.preferenceStorage.body)}</p>
            </aside>
            <p className="text-sm text-app-muted-foreground">
              {t(strings.preferenceStorage.noAccountChange)} {t(strings.preferenceStorage.otherDraftsStay)}
            </p>
          </>
        )}
      </div>
    );
  }
  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.settings.themeHeading)}
        </h3>
        <div role="radiogroup" aria-label={t(strings.theme.switcherLabel)} className="flex flex-wrap gap-2">
          {THEME_CHOICES.map((theme) => (
            <button
              key={theme}
              type="button"
              role="radio"
              aria-checked={choice === theme}
              onClick={() => setTheme(theme)}
              data-testid={selectors.settingsPage.themeOption({ choice: theme })}
              className={`min-h-11 rounded-full border px-3 text-sm font-semibold ${choice === theme ? "border-app-primary bg-app-primary text-app-primary-foreground" : "border-app-border bg-app-surface text-app-foreground"}`}
            >
              {t(strings.theme.choice[theme])}
            </button>
          ))}
        </div>
      </div>

      <div className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.settings.localeHeading)}
        </h3>
        <div role="radiogroup" aria-label={t(strings.locale.switcherLabel)} className="flex flex-wrap gap-2">
          {SUPPORTED_LOCALES.map((locale) => (
            <button
              key={locale}
              type="button"
              role="radio"
              aria-checked={currentLocale === locale}
              onClick={() => void setLocale(locale)}
              data-testid={selectors.settingsPage.localeOption({ code: locale })}
              className={`min-h-11 rounded-full border px-3 text-sm font-semibold ${currentLocale === locale ? "border-app-primary bg-app-primary text-app-primary-foreground" : "border-app-border bg-app-surface text-app-foreground"}`}
            >
              {getLocaleConfig(locale).nativeLabel}
            </button>
          ))}
        </div>
      </div>

      {preferenceStorageUnavailable && (
        <aside role="status" aria-live="polite" aria-atomic="true" className="border-s-4 border-app-warning ps-4 py-3">
          <p className="font-semibold">{t(strings.preferenceStorage.title)}</p>
          <p>{t(strings.preferenceStorage.body)}</p>
          <p className="mt-2 text-sm text-app-muted-foreground">{t(strings.preferenceStorage.noAccountChange)}</p>
          <p className="text-sm text-app-muted-foreground">{t(strings.preferenceStorage.otherDraftsStay)}</p>
        </aside>
      )}
    </div>
  );
}

/**
 * A source may declare an empty category list — it only offers a coarse,
 * source-wide on/off. We synthesise a single pseudo-category with the empty
 * string id so the row maps onto the source-wide preference the hub stores
 * (matching web-console's NotificationsSection).
 */
function categoriesFor(source: CatalogSource): CatalogCategory[] {
  if (source.categories.length > 0) return source.categories;
  return [
    {
      category: "",
      display_name: "All notifications",
      description: "",
      declared: source.declared,
      default_enabled: true,
      default_min_urgency: "low",
    },
  ];
}

/** A loading skeleton shaped like the populated list, so the section never flashes blank. */
function NotificationsSkeleton() {
  return (
    <div className="grid gap-4" data-testid="notifications-loading" role="status" aria-live="polite" aria-label="Loading notification preferences">
      {[0, 1].map((group) => (
        <div key={group} className="flex flex-col gap-3 rounded-lg border border-app-border bg-app-surface-raised px-3 py-3">
          <div className="flex items-center gap-2.5">
            <div className="h-6 w-6 shrink-0 animate-pulse rounded-md bg-app-surface-muted" />
            <div className="h-3 w-32 animate-pulse rounded bg-app-surface-muted" />
          </div>
          {[0, 1].map((row) => (
            <div key={row} className="flex items-center justify-between gap-4">
              <div className="grid flex-1 gap-2">
                <div className="h-2.5 w-40 animate-pulse rounded bg-app-surface-muted" />
                <div className="h-2 w-3/4 animate-pulse rounded bg-app-surface-muted" />
              </div>
              <div className="h-5 w-16 shrink-0 animate-pulse rounded-full bg-app-surface-muted" />
            </div>
          ))}
        </div>
      ))}
      <span className="sr-only">Loading notification preferences</span>
    </div>
  );
}

/** A composed, centred message used for both the empty and error states. */
function NotificationsMessage({ icon, title, body, children, testId }: {
  icon: React.ReactNode;
  title: string;
  body: string;
  children?: React.ReactNode;
  testId: string;
}) {
  return (
    <div className="flex flex-col items-center gap-3 rounded-lg border border-app-border bg-app-surface-raised px-6 py-10 text-center" role="status" data-testid={testId}>
      <span className="grid h-12 w-12 place-items-center rounded-2xl bg-app-primary/10 text-app-primary">{icon}</span>
      <div className="grid gap-1">
        <p className="font-semibold text-app-foreground">{title}</p>
        <p className="mx-auto max-w-sm text-sm text-app-muted-foreground">{body}</p>
      </div>
      {children && <div className="mt-1 flex flex-col items-center gap-2">{children}</div>}
    </div>
  );
}

function SecondaryButton({ children, onClick, testId, icon }: {
  children: React.ReactNode;
  onClick: () => void;
  testId?: string;
  icon?: React.ReactNode;
}) {
  return (
    <button
      type="button"
      data-testid={testId}
      onClick={onClick}
      className="inline-flex items-center gap-2 rounded-lg border border-app-border bg-transparent px-3.5 py-1.5 text-sm font-medium text-app-foreground transition-colors hover:bg-app-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary"
    >
      {icon}
      {children}
    </button>
  );
}
