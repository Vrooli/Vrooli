import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { SettingsList } from "@vrooli/react-component-library/SettingsList/1";
import { Input } from "@vrooli/react-component-library/Input/1";
import { Select } from "@vrooli/react-component-library/Select/1";
import { Button } from "@vrooli/react-component-library/Button/2";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation, type Locale } from "../i18n";
import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTheme, type ThemeChoice } from "../theme/ThemeProvider";
import { fetchAvailability, fetchPlanningProfile, replaceAvailability, updatePlanningProfile } from "../api/workspace";
import { createFixtureConnection, disconnectConnection, fetchConnections, syncConnection } from "../api/integrations";
import { create } from "@bufbuild/protobuf";
import { AvailabilityExceptionSchema, AvailabilityWindowSchema, type AvailabilityException, type AvailabilityWindow } from "@vrooli/proto-types/personal-planner/v1/workspace/workspace_pb";
import { DEFAULT_TRANSITION_HOURS, OBSERVATORY_DAY_START_KEY, OBSERVATORY_NIGHT_START_KEY } from "../theme/observatoryAppearance";

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
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  const currentLocale = getCurrentLocale();
  const { choice, setTheme } = useTheme();
  const [artFree, setArtFree] = usePreference("planner.art-free");
  const [reducedScenery, setReducedScenery] = usePreference("planner.reduced-scenery");
  const [subduedNight, setSubduedNight] = usePreference("planner.subdued-night");
  const [autoDayStart, setAutoDayStart] = useStoredPreference(OBSERVATORY_DAY_START_KEY, DEFAULT_TRANSITION_HOURS.dayStart);
  const [autoNightStart, setAutoNightStart] = useStoredPreference(OBSERVATORY_NIGHT_START_KEY, DEFAULT_TRANSITION_HOURS.nightStart);
  const profile = useQuery({ queryKey: ["planning-profile"], queryFn: fetchPlanningProfile });
  const availability = useQuery({ queryKey: ["planning-availability"], queryFn: fetchAvailability });
  const connections = useQuery({ queryKey: ["calendar-connections"], queryFn: fetchConnections });
  const [timezone, setTimezone] = useState("");
  const [weekStart, setWeekStart] = useState("monday");
  const [dailyCapacityMinutes, setDailyCapacityMinutes] = useState(480);
  const [reserveMinutes, setReserveMinutes] = useState(60);
  const [focusSessionMinutes, setFocusSessionMinutes] = useState(45);
  const [windows, setWindows] = useState<AvailabilityWindow[]>([]);
  const [exception, setException] = useState<AvailabilityException>(() => create(AvailabilityExceptionSchema, { date: "", startMinute: 720, endMinute: 780, kind: "protected", reason: "" }));
  useEffect(() => {
    if (!profile.data) return;
    setTimezone(profile.data.timezone);
    setWeekStart(profile.data.weekStart);
    setDailyCapacityMinutes(profile.data.dailyCapacityMinutes);
    setReserveMinutes(profile.data.reserveMinutes);
    setFocusSessionMinutes(profile.data.focusSessionMinutes);
  }, [profile.data]);
  useEffect(() => { if (availability.data) setWindows(availability.data.windows); }, [availability.data]);
  const profileMutation = useMutation({
    mutationFn: () => updatePlanningProfile(profile.data!, { timezone, weekStart, dailyCapacityMinutes, reserveMinutes, focusSessionMinutes }),
    onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ["planning-profile"] }); },
  });
  const availabilityMutation = useMutation({
    mutationFn: () => replaceAvailability({ windows, exceptions: exception.date ? [...(availability.data?.exceptions ?? []), exception] : (availability.data?.exceptions ?? []), expectedRevision: availability.data?.revision ?? profile.data?.revision ?? 0n }),
    onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ["planning-availability"] }); await queryClient.invalidateQueries({ queryKey: ["planning-profile"] }); },
  });
  const fixtureMutation = useMutation({ mutationFn: () => createFixtureConnection(), onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ["calendar-connections"] }); } });
  const syncMutation = useMutation({ mutationFn: syncConnection, onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ["calendar-connections"] }); } });
  const disconnectMutation = useMutation({ mutationFn: disconnectConnection, onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ["calendar-connections"] }); } });
  const setWindow = (weekday: number, field: "startMinute" | "endMinute", value: number) => setWindows((current) => { const existing = current.find((item) => item.weekday === weekday); const next = existing ? create(AvailabilityWindowSchema, { ...existing, [field]: value }) : create(AvailabilityWindowSchema, { id: `weekday-${weekday}`, weekday, startMinute: 540, endMinute: 1020, timezone: timezone || "UTC", [field]: value }); return [...current.filter((item) => item.weekday !== weekday), next].sort((a, b) => a.weekday - b.weekday); });

  return (
    <section data-testid={selectors.pages.settings} aria-labelledby="settings-heading" className="settings-page flex flex-col gap-space-md">
      <PageHeader headingId="settings-heading" title={t(strings.pages.settings.title)} description={t(strings.pages.settings.description)} />
      <SettingsList variant="auto" density="compact" className="settings-list">
        <SettingsList.Group label={t(strings.pages.settings.preferences)}>
          <SettingsList.Row label={t(strings.pages.settings.themeHeading)} hint={t(strings.pages.settings.themeHint)}>
            <div className="settings-choice-group" role="radiogroup" aria-label={t(strings.theme.switcherLabel)} data-testid={selectors.settingsPage.themeSelect}>
              {THEME_CHOICES.map((themeChoice) => (
                <label key={themeChoice} className="settings-choice">
                  <input type="radio" name="theme-choice" value={themeChoice} checked={choice === themeChoice} aria-checked={choice === themeChoice} data-testid={selectors.settingsPage.themeOption({ choice: themeChoice })} onChange={() => setTheme(themeChoice)} />
                  <span>{t(THEME_LABEL_KEY[themeChoice])}</span>
                </label>
              ))}
            </div>
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
          <SettingsList.Row label={t(strings.pages.settings.sceneryHeading)} hint={t(strings.pages.settings.sceneryHint)}>
            <div className="settings-toggle-list">
              <label><input type="checkbox" checked={artFree} onChange={(event) => setArtFree(event.target.checked)} /> {t(strings.pages.settings.artFree)}</label>
              <label><input type="checkbox" checked={reducedScenery} onChange={(event) => setReducedScenery(event.target.checked)} /> {t(strings.pages.settings.reducedScenery)}</label>
              <label><input type="checkbox" checked={subduedNight} onChange={(event) => setSubduedNight(event.target.checked)} /> {t(strings.pages.settings.subduedNight)}</label>
            </div>
          </SettingsList.Row>
          <SettingsList.Row label={t(strings.pages.settings.transitionHeading)} hint={t(strings.pages.settings.transitionHint)}>
            <div className="settings-transition-grid">
              <label>{t(strings.pages.settings.dayBegins)}<Input type="time" value={autoDayStart} onChange={(event) => setAutoDayStart(event.target.value)} /></label>
              <label>{t(strings.pages.settings.nightBegins)}<Input type="time" value={autoNightStart} onChange={(event) => setAutoNightStart(event.target.value)} /></label>
            </div>
          </SettingsList.Row>
          <SettingsList.Row label={t(strings.pages.settings.planningHeading)} hint={t(strings.pages.settings.planningHint)}>
            {profile.isError && <p role="alert">{t(strings.pages.settings.planningUnavailable)}</p>}
            {profile.isLoading && <p role="status">{t(strings.pages.settings.planningLoading)}</p>}
            {profile.data && <form className="settings-profile-form" onSubmit={(event) => { event.preventDefault(); profileMutation.mutate(); }}>
              <label>{t(strings.pages.settings.timezone)}<Input value={timezone} onChange={(event) => setTimezone(event.target.value)} placeholder="America/New_York" required /></label>
              <label>{t(strings.pages.settings.weekStart)}<Select aria-label={t(strings.pages.settings.weekStart)} value={weekStart} onChange={(event) => setWeekStart(event.target.value)} options={[{ value: "monday", label: t(strings.pages.settings.monday) }, { value: "sunday", label: t(strings.pages.settings.sunday) }]} /></label>
              <label>{t(strings.pages.settings.dailyCapacity)}<Input type="number" min="0" max="1440" value={dailyCapacityMinutes} onChange={(event) => setDailyCapacityMinutes(Number(event.target.value))} /></label>
              <label>{t(strings.pages.settings.reserve)}<Input type="number" min="0" max={dailyCapacityMinutes} value={reserveMinutes} onChange={(event) => setReserveMinutes(Number(event.target.value))} /></label>
              <label>{t(strings.pages.settings.focusSession)}<Input type="number" min="5" max="240" value={focusSessionMinutes} onChange={(event) => setFocusSessionMinutes(Number(event.target.value))} /></label>
              <Button className="secondary-action" variant="secondary" type="submit" disabled={profileMutation.isPending} pending={profileMutation.isPending} pendingLabel={t(strings.pages.settings.savePlanning)}>{t(strings.pages.settings.savePlanning)}</Button>
              {profileMutation.isError && <p role="alert">{t(strings.pages.settings.planningSaveError)}</p>}
              {profileMutation.isSuccess && <p role="status">{t(strings.pages.settings.planningSaved)}</p>}
            </form>}
          </SettingsList.Row>
          <SettingsList.Row label={t(strings.pages.settings.availabilityHeading)} hint={t(strings.pages.settings.availabilityHint)}>
            {availability.isLoading && <p role="status">{t(strings.pages.settings.availabilityLoading)}</p>}
            {availability.data && <form className="settings-availability-form" onSubmit={(event) => { event.preventDefault(); availabilityMutation.mutate(); }}>
              <div className="availability-grid" aria-label={t(strings.pages.settings.availabilityHeading)}>
                {[1, 2, 3, 4, 5, 6, 7].map((weekday) => { const item = windows.find((window) => window.weekday === weekday); return <label key={weekday}><span>{t(strings.pages.settings[`weekday${weekday}` as keyof typeof strings.pages.settings] as never)}</span><Input aria-label={`${weekday} start`} type="time" value={minuteToTime(item?.startMinute ?? 540)} onChange={(event) => setWindow(weekday, "startMinute", timeToMinute(event.target.value))} /><Input aria-label={`${weekday} end`} type="time" value={minuteToTime(item?.endMinute ?? 1020)} onChange={(event) => setWindow(weekday, "endMinute", timeToMinute(event.target.value))} /></label>; })}
              </div>
              <div className="availability-exception"><strong>{t(strings.pages.settings.protectedTime)}</strong><Input type="date" value={exception.date} onChange={(event) => setException({ ...exception, date: event.target.value })} /><Input placeholder={t(strings.pages.settings.protectedReason)} value={exception.reason} onChange={(event) => setException({ ...exception, reason: event.target.value })} /></div>
              <Button className="secondary-action" variant="secondary" type="submit" disabled={availabilityMutation.isPending} pending={availabilityMutation.isPending} pendingLabel={t(strings.pages.settings.saveAvailability)}>{t(strings.pages.settings.saveAvailability)}</Button>
              {availabilityMutation.isError && <p role="alert">{t(strings.pages.settings.availabilitySaveError)}</p>}
              {availabilityMutation.isSuccess && <p role="status">{t(strings.pages.settings.availabilitySaved)}</p>}
            </form>}
          </SettingsList.Row>
          <SettingsList.Row label={t(strings.pages.settings.integrationsHeading)} hint={t(strings.pages.settings.integrationsHint)}>
            {connections.isLoading && <p role="status">{t(strings.pages.settings.integrationsLoading)}</p>}
            {connections.isError && <p role="alert">{t(strings.pages.settings.connectionError)}</p>}
            {connections.data?.length === 0 && <p className="settings-inline-note">{t(strings.pages.settings.integrationsEmpty)}</p>}
            {connections.data?.map((connection) => <article className="integration-connection" key={connection.id}>
              <div><strong>{connection.displayName}</strong><span>{connection.sourceKind === "fixture" ? t(strings.pages.settings.connectionFixture) : connection.provider} · {t(strings.pages.settings.connectionReadOnly)} · {connection.status}</span><small>{t(strings.pages.settings.connectionEvents, { count: connection.importedEventCount })} · {t(strings.pages.settings.connectionBusy, { count: Number(connection.busyMinutes) })}</small>{connection.healthMessage && <small>{connection.healthMessage}</small>}</div>
                <div className="integration-actions"><Button className="secondary-action" variant="secondary" type="button" onClick={() => syncMutation.mutate(connection)} disabled={syncMutation.isPending || connection.status === "disconnected"} pending={syncMutation.isPending} pendingLabel={t(strings.pages.settings.syncConnection)}>{t(strings.pages.settings.syncConnection)}</Button><Button className="quiet-action" variant="ghost" type="button" onClick={() => disconnectMutation.mutate(connection)} disabled={disconnectMutation.isPending || connection.status === "disconnected"}>{t(strings.pages.settings.disconnectConnection)}</Button></div>
            </article>)}
            <div className="integration-fixture-callout"><p>{t(strings.pages.settings.fixtureConnectionHelp)}</p><Button className="secondary-action" variant="secondary" type="button" onClick={() => fixtureMutation.mutate()} disabled={fixtureMutation.isPending} pending={fixtureMutation.isPending} pendingLabel={t(strings.pages.settings.fixtureConnection)}>{t(strings.pages.settings.fixtureConnection)}</Button></div>
            {(fixtureMutation.isSuccess || syncMutation.isSuccess || disconnectMutation.isSuccess) && <p role="status">{fixtureMutation.isSuccess ? t(strings.pages.settings.connectionCreated) : syncMutation.isSuccess ? t(strings.pages.settings.connectionSynced) : t(strings.pages.settings.connectionDisconnected)}</p>}
            {(fixtureMutation.isError || syncMutation.isError || disconnectMutation.isError) && <p role="alert">{t(strings.pages.settings.connectionError)}</p>}
            <p className="settings-story-note">{t(strings.pages.settings.integrationBoundary)}</p>
          </SettingsList.Row>
        </SettingsList.Group>
      </SettingsList>
      <section className="settings-story" aria-labelledby="settings-story-heading">
        <span className="card-kicker">{t(strings.pages.settings.storyKicker)}</span>
        <h2 id="settings-story-heading">{t(strings.pages.settings.storyTitle)}</h2>
        <p>{t(strings.pages.settings.storyIntro)}</p>
        <div className="settings-story-grid">
          <div><strong>{t(strings.pages.settings.storyIncludedTitle)}</strong><span>{t(strings.pages.settings.storyIncluded)}</span></div>
          <div><strong>{t(strings.pages.settings.storyPrivateTitle)}</strong><span>{t(strings.pages.settings.storyPrivate)}</span></div>
          <div><strong>{t(strings.pages.settings.storyFutureTitle)}</strong><span>{t(strings.pages.settings.storyFuture)}</span></div>
        </div>
        <p className="settings-story-note">{t(strings.pages.settings.storyNote)}</p>
      </section>
    </section>
  );
}

function minuteToTime(value: number): string { return `${String(Math.floor(value / 60)).padStart(2, "0")}:${String(value % 60).padStart(2, "0")}`; }
function timeToMinute(value: string): number { const parts = value.split(":").map(Number); return (parts[0] ?? 0) * 60 + (parts[1] ?? 0); }

function usePreference(key: string): [boolean, (value: boolean) => void] {
  const [value, setValue] = useState(() => typeof window !== "undefined" && window.localStorage.getItem(key) === "true");
  useEffect(() => {
    window.localStorage.setItem(key, String(value));
  }, [key, value]);
  return [value, setValue];
}

function useStoredPreference(key: string, fallback: string): [string, (value: string) => void] {
  const [value, setValue] = useState(() => typeof window !== "undefined" ? window.localStorage.getItem(key) ?? fallback : fallback);
  useEffect(() => { window.localStorage.setItem(key, value); }, [key, value]);
  return [value, setValue];
}
