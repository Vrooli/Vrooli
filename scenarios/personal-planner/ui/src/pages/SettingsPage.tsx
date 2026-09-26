import { AdaptivePageHeader } from "../components/AdaptivePageHeader";
import { SettingsList } from "@vrooli/react-component-library/SettingsList/1";
import { Input } from "@vrooli/react-component-library/Input/1";
import { Select } from "@vrooli/react-component-library/Select/1";
import { Button } from "@vrooli/react-component-library/Button/2";
import { RadioGroup } from "@vrooli/react-component-library/RadioGroup/1";
import { Switch } from "@vrooli/react-component-library/Switch/1";
import { FormField } from "@vrooli/react-component-library/FormField/1";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation, type Locale } from "../i18n";
import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTheme, type ThemeChoice } from "../theme/ThemeProvider";
import { fetchAvailability, fetchPlanningProfile, replaceAvailability, updatePlanningProfile } from "../api/workspace";
import { fetchReminderPreferences, saveReminderPreferences } from "../api/review";
import { createFixtureConnection, disconnectConnection, fetchConnections, syncConnection } from "../api/integrations";
import { create } from "@bufbuild/protobuf";
import { AvailabilityExceptionSchema, AvailabilityWindowSchema, type AvailabilityException, type AvailabilityWindow } from "@vrooli/proto-types/personal-planner/v1/workspace/workspace_pb";
import { DEFAULT_TRANSITION_HOURS, OBSERVATORY_DAY_START_KEY, OBSERVATORY_NIGHT_START_KEY, OBSERVATORY_PREFERENCES_EVENT, OBSERVATORY_SCENERY_EVENT } from "../theme/observatoryAppearance";
import { HealthCard } from "../components/HealthCard";
import { Settings2 } from "lucide-react";
import { ObservatoryScene } from "../components/ObservatoryScene";
import { useBreakpoint } from "../hooks/useBreakpoint";

const THEME_CHOICES: readonly ThemeChoice[] = ["auto", "day", "night"];
// Literal references so the strings lint can see every catalog key in use.
const THEME_LABEL_KEY: Record<ThemeChoice, (typeof strings.theme.choice)[ThemeChoice]> = {
  auto: strings.theme.choice.auto,
  day: strings.theme.choice.day,
  night: strings.theme.choice.night,
};

/**
 * Settings owns every preference; nothing else in the shell duplicates one.
 * Appearance and language are the two every scenario has. Add the rest of
 * yours as rows in the same list, grouped by what they change.
 */
export function SettingsPage() {
  const queryClient = useQueryClient();
  const { isMobile } = useBreakpoint();
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
  const reminderPreferences = useQuery({ queryKey: ["reminder-preferences"], queryFn: fetchReminderPreferences });
  const [timezone, setTimezone] = useState("");
  const [weekStart, setWeekStart] = useState("monday");
  const [dailyCapacityMinutes, setDailyCapacityMinutes] = useState(480);
  const [reserveMinutes, setReserveMinutes] = useState(60);
  const [focusSessionMinutes, setFocusSessionMinutes] = useState(45);
  const [remindersEnabled, setRemindersEnabled] = useState(true);
  const [quietStartMinutes, setQuietStartMinutes] = useState(1320);
  const [quietEndMinutes, setQuietEndMinutes] = useState(420);
  const [leadMinutes, setLeadMinutes] = useState(60);
  const [windows, setWindows] = useState<AvailabilityWindow[]>([]);
  const [exception, setException] = useState<AvailabilityException>(() => create(AvailabilityExceptionSchema, { date: "", startMinute: 720, endMinute: 780, kind: "protected", reason: "" }));
  const [mobileSection, setMobileSection] = useState("preferences");
  useEffect(() => {
    if (!profile.data) return;
    setTimezone(profile.data.timezone);
    setWeekStart(profile.data.weekStart);
    setDailyCapacityMinutes(profile.data.dailyCapacityMinutes);
    setReserveMinutes(profile.data.reserveMinutes);
    setFocusSessionMinutes(profile.data.focusSessionMinutes);
  }, [profile.data]);
  useEffect(() => { if (availability.data) setWindows(availability.data.windows); }, [availability.data]);
  useEffect(() => {
    if (!reminderPreferences.data) return;
    setRemindersEnabled(reminderPreferences.data.enabled);
    setQuietStartMinutes(reminderPreferences.data.quietStartMinutes);
    setQuietEndMinutes(reminderPreferences.data.quietEndMinutes);
    setLeadMinutes(reminderPreferences.data.leadMinutes);
  }, [reminderPreferences.data]);
  const profileMutation = useMutation({
    mutationFn: () => profile.data ? updatePlanningProfile(profile.data, { timezone, weekStart, dailyCapacityMinutes, reserveMinutes, focusSessionMinutes }) : Promise.reject(new Error("Planning profile is not loaded")),
    onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ["planning-profile"] }); },
  });
  const availabilityMutation = useMutation({
    mutationFn: () => replaceAvailability({ windows, exceptions: exception.date ? [...(availability.data?.exceptions ?? []), exception] : (availability.data?.exceptions ?? []), expectedRevision: availability.data?.revision ?? profile.data?.revision ?? 0n }),
    onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ["planning-availability"] }); await queryClient.invalidateQueries({ queryKey: ["planning-profile"] }); },
  });
  const fixtureMutation = useMutation({ mutationFn: () => createFixtureConnection(), onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ["calendar-connections"] }); } });
  const syncMutation = useMutation({ mutationFn: syncConnection, onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ["calendar-connections"] }); } });
  const disconnectMutation = useMutation({ mutationFn: disconnectConnection, onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ["calendar-connections"] }); } });
  const reminderMutation = useMutation({
    mutationFn: () => saveReminderPreferences({ enabled: remindersEnabled, quietStartMinutes, quietEndMinutes, leadMinutes }),
    onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ["reminder-preferences"] }); },
  });
  const setWindow = (weekday: number, field: "startMinute" | "endMinute", value: number) => setWindows((current) => { const existing = current.find((item) => item.weekday === weekday); const next = existing ? create(AvailabilityWindowSchema, { ...existing, [field]: value }) : create(AvailabilityWindowSchema, { id: `weekday-${weekday}`, weekday, startMinute: 540, endMinute: 1020, timezone: timezone || "UTC", [field]: value }); return [...current.filter((item) => item.weekday !== weekday), next].sort((a, b) => a.weekday - b.weekday); });

  return (
    <ObservatoryScene kind="settings"><section data-testid={selectors.pages.settings} aria-labelledby="settings-heading" className="planner-surface settings-page flex flex-col gap-space-md">
      <AdaptivePageHeader className="planner-page-header" headingId="settings-heading" eyebrow="The observatory desk" title={t(strings.pages.settings.title)} description={t(strings.pages.settings.description)} leading={<span className="planner-page-mark" aria-hidden="true"><Settings2 size={21} /></span>} />
      {isMobile && <section className="settings-mobile-intro" aria-label="Mobile settings guidance"><span className="card-kicker">TUNE ONE INSTRUMENT AT A TIME</span><p>Start with the setting that changes today’s plan. The controls below stay grouped by instrument instead of becoming one long form.</p></section>}
      {isMobile && <div className="settings-mobile-nav" role="navigation" aria-label="Settings sections">{[{ id: "preferences", label: t(strings.pages.settings.preferences) }, { id: "planning", label: t(strings.pages.settings.planningHeading) }, { id: "reminders", label: "Reminders" }, { id: "availability", label: t(strings.pages.settings.availabilityHeading) }, { id: "integrations", label: t(strings.pages.settings.integrationsHeading) }, { id: "health", label: t(strings.health.title) }].map((section) => <Button key={section.id} type="button" variant={mobileSection === section.id ? "primary" : "secondary"} aria-selected={mobileSection === section.id} onClick={() => setMobileSection(section.id)}>{section.label}</Button>)}</div>}
      <SettingsList variant="auto" density="compact" className="settings-list">
        <SettingsList.Group className={isMobile && mobileSection !== "preferences" ? "settings-mobile-hidden" : undefined} label={t(strings.pages.settings.preferences)}>
          <SettingsList.Row label={t(strings.pages.settings.themeHeading)} hint={t(strings.pages.settings.themeHint)}>
            <div data-testid={selectors.settingsPage.themeSelect}>
              <RadioGroup
                label={t(strings.theme.switcherLabel)}
                name="theme-choice"
                orientation="horizontal"
                variant="card"
                value={choice}
                onValueChange={(value) => setTheme(value as ThemeChoice)}
                options={THEME_CHOICES.map((themeChoice) => ({ value: themeChoice, label: t(THEME_LABEL_KEY[themeChoice]), testId: selectors.settingsPage.themeOption({ choice: themeChoice }) }))}
              />
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
              <Switch label={t(strings.pages.settings.artFree)} checked={artFree} onCheckedChange={setArtFree} />
              <Switch label={t(strings.pages.settings.reducedScenery)} checked={reducedScenery} onCheckedChange={setReducedScenery} />
              <Switch label={t(strings.pages.settings.subduedNight)} checked={subduedNight} onCheckedChange={setSubduedNight} />
            </div>
          </SettingsList.Row>
          <SettingsList.Row control="wide" label={t(strings.pages.settings.transitionHeading)} hint={t(strings.pages.settings.transitionHint)}>
            <div className="settings-transition-grid">
              <FormField label={t(strings.pages.settings.dayBegins)} control={<Input type="time" value={autoDayStart} onChange={(event) => setAutoDayStart(event.target.value)} />} />
              <FormField label={t(strings.pages.settings.nightBegins)} control={<Input type="time" value={autoNightStart} onChange={(event) => setAutoNightStart(event.target.value)} />} />
            </div>
          </SettingsList.Row>
        </SettingsList.Group>
        <SettingsList.Group className={isMobile && mobileSection !== "planning" ? "settings-mobile-hidden" : undefined} label={t(strings.pages.settings.planningHeading)}>
          <SettingsList.Row className="settings-form-row" control="wide" label={t(strings.pages.settings.planningHeading)} hint={t(strings.pages.settings.planningHint)}>
            {profile.isError && <p role="alert">{t(strings.pages.settings.planningUnavailable)}</p>}
            {profile.isLoading && <p role="status">{t(strings.pages.settings.planningLoading)}</p>}
            {profile.data && <form className="settings-profile-form" onSubmit={(event) => { event.preventDefault(); profileMutation.mutate(); }}>
              <FormField label={t(strings.pages.settings.timezone)} required control={<Input aria-label={t(strings.pages.settings.timezone)} value={timezone} onChange={(event) => setTimezone(event.target.value)} placeholder="America/New_York" />} />
              <FormField label={t(strings.pages.settings.weekStart)} control={<Select aria-label={t(strings.pages.settings.weekStart)} value={weekStart} onChange={(event) => setWeekStart(event.target.value)} options={[{ value: "monday", label: t(strings.pages.settings.monday) }, { value: "sunday", label: t(strings.pages.settings.sunday) }]} />} />
              <FormField label={t(strings.pages.settings.dailyCapacity)} control={<Input type="number" min="0" max="1440" value={dailyCapacityMinutes} onChange={(event) => setDailyCapacityMinutes(Number(event.target.value))} />} />
              <FormField label={t(strings.pages.settings.reserve)} control={<Input type="number" min="0" max={dailyCapacityMinutes} value={reserveMinutes} onChange={(event) => setReserveMinutes(Number(event.target.value))} />} />
              <FormField label={t(strings.pages.settings.focusSession)} control={<Input type="number" min="5" max="240" value={focusSessionMinutes} onChange={(event) => setFocusSessionMinutes(Number(event.target.value))} />} />
              <Button className="secondary-action" variant="secondary" type="submit" disabled={profileMutation.isPending} pending={profileMutation.isPending} pendingLabel={t(strings.pages.settings.savePlanning)}>{t(strings.pages.settings.savePlanning)}</Button>
              {profileMutation.isError && <p role="alert">{t(strings.pages.settings.planningSaveError)}</p>}
              {profileMutation.isSuccess && <p role="status">{t(strings.pages.settings.planningSaved)}</p>}
            </form>}
          </SettingsList.Row>
        </SettingsList.Group>
        <SettingsList.Group className={isMobile && mobileSection !== "reminders" ? "settings-mobile-hidden" : undefined} label="Reminders">
          <SettingsList.Row className="settings-form-row" control="wide" label="Protect your attention" hint="In-app nudges pause during quiet hours and use your chosen lead time.">
            {reminderPreferences.isLoading && <p role="status">Loading reminder preferences…</p>}
            {reminderPreferences.data && <form className="settings-profile-form" onSubmit={(event) => { event.preventDefault(); reminderMutation.mutate(); }}>
              <Switch label="Show reminders" checked={remindersEnabled} onCheckedChange={setRemindersEnabled} />
              <div className="settings-transition-grid">
                <FormField label="Quiet hours start" control={<Input type="time" aria-label="Quiet hours start" value={minuteToTime(quietStartMinutes)} onChange={(event) => setQuietStartMinutes(timeToMinute(event.target.value))} />} />
                <FormField label="Quiet hours end" control={<Input type="time" aria-label="Quiet hours end" value={minuteToTime(quietEndMinutes)} onChange={(event) => setQuietEndMinutes(timeToMinute(event.target.value))} />} />
              </div>
              <FormField label="Lead time (minutes)" control={<Input type="number" min="0" max="240" value={leadMinutes} onChange={(event) => setLeadMinutes(Number(event.target.value))} />} />
              <Button className="secondary-action" variant="secondary" type="submit" disabled={reminderMutation.isPending} pending={reminderMutation.isPending} pendingLabel="Saving reminders…">Save reminder preferences</Button>
              {reminderMutation.isError && <p role="alert">Reminder preferences could not be saved.</p>}
              {reminderMutation.isSuccess && <p role="status">Reminder preferences saved.</p>}
            </form>}
          </SettingsList.Row>
        </SettingsList.Group>
        <SettingsList.Group className={isMobile && mobileSection !== "availability" ? "settings-mobile-hidden" : undefined} label={t(strings.pages.settings.availabilityHeading)}>
          <SettingsList.Row className="settings-form-row" control="wide" label={t(strings.pages.settings.availabilityHeading)} hint={t(strings.pages.settings.availabilityHint)}>
            {availability.isLoading && <p role="status">{t(strings.pages.settings.availabilityLoading)}</p>}
            {availability.data && <form className="settings-availability-form" onSubmit={(event) => { event.preventDefault(); availabilityMutation.mutate(); }}>
              <div className="availability-grid" aria-label={t(strings.pages.settings.availabilityHeading)}>
                <div className="availability-grid-heading" aria-hidden="true"><span>Day</span><span>Starts</span><span>Ends</span></div>
                {[1, 2, 3, 4, 5, 6, 7].map((weekday) => { const item = windows.find((window) => window.weekday === weekday); return <div className="availability-day" key={weekday}><strong>{t(strings.pages.settings[`weekday${weekday}` as keyof typeof strings.pages.settings] as never)}</strong><FormField label="Starts" optionalLabel="" control={<Input aria-label={`${weekday} start`} type="time" value={minuteToTime(item?.startMinute ?? 540)} onChange={(event) => setWindow(weekday, "startMinute", timeToMinute(event.target.value))} />} /><FormField label="Ends" optionalLabel="" control={<Input aria-label={`${weekday} end`} type="time" value={minuteToTime(item?.endMinute ?? 1020)} onChange={(event) => setWindow(weekday, "endMinute", timeToMinute(event.target.value))} />} /></div>; })}
              </div>
              <div className="availability-exception"><strong>{t(strings.pages.settings.protectedTime)}</strong><FormField label="Date" optionalLabel="" control={<Input type="date" value={exception.date} onChange={(event) => setException({ ...exception, date: event.target.value })} />} /><FormField label="Reason" optionalLabel="" control={<Input placeholder={t(strings.pages.settings.protectedReason)} value={exception.reason} onChange={(event) => setException({ ...exception, reason: event.target.value })} />} /></div>
              <Button className="secondary-action" variant="secondary" type="submit" disabled={availabilityMutation.isPending} pending={availabilityMutation.isPending} pendingLabel={t(strings.pages.settings.saveAvailability)}>{t(strings.pages.settings.saveAvailability)}</Button>
              {availabilityMutation.isError && <p role="alert">{t(strings.pages.settings.availabilitySaveError)}</p>}
              {availabilityMutation.isSuccess && <p role="status">{t(strings.pages.settings.availabilitySaved)}</p>}
            </form>}
          </SettingsList.Row>
        </SettingsList.Group>
        <SettingsList.Group className={isMobile && mobileSection !== "integrations" ? "settings-mobile-hidden" : undefined} label={t(strings.pages.settings.integrationsHeading)}>
          <SettingsList.Row className="settings-form-row" control="wide" label={t(strings.pages.settings.integrationsHeading)} hint={t(strings.pages.settings.integrationsHint)}>
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
        <SettingsList.Group className={isMobile && mobileSection !== "health" ? "settings-mobile-hidden" : undefined} label={t(strings.health.title)}>
          <SettingsList.Row
            className="settings-health-row"
            control="wide"
            label={t(strings.health.title)}
            hint={t(strings.health.description)}
          >
            <div className="settings-health-card">
              <HealthCard />
            </div>
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
    </section></ObservatoryScene>
  );
}

function minuteToTime(value: number): string { return `${String(Math.floor(value / 60)).padStart(2, "0")}:${String(value % 60).padStart(2, "0")}`; }
function timeToMinute(value: string): number { const parts = value.split(":").map(Number); return (parts[0] ?? 0) * 60 + (parts[1] ?? 0); }

function usePreference(key: string): [boolean, (value: boolean) => void] {
  const [value, setValue] = useState(() => typeof window !== "undefined" && window.localStorage.getItem(key) === "true");
  useEffect(() => {
    window.localStorage.setItem(key, String(value));
    window.dispatchEvent(new Event(OBSERVATORY_SCENERY_EVENT));
  }, [key, value]);
  return [value, setValue];
}

function useStoredPreference(key: string, fallback: string): [string, (value: string) => void] {
  const [value, setValue] = useState(() => typeof window !== "undefined" ? window.localStorage.getItem(key) ?? fallback : fallback);
  useEffect(() => {
    window.localStorage.setItem(key, value);
    window.dispatchEvent(new Event(OBSERVATORY_PREFERENCES_EVENT));
  }, [key, value]);
  return [value, setValue];
}
