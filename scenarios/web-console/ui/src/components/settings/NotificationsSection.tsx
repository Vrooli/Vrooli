import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { Bell, BellOff, ExternalLink, RefreshCw } from "lucide-react";
import { strings } from "../../consts/strings";
import {
  fetchScenarioLink,
  notificationPreferenceControlGroups,
  notificationPreferencesClient,
  NotificationsUnavailableError,
  WEB_CONSOLE_SOURCE,
} from "../../api/notifications";
import { useNotificationPreferences } from "@vrooli/api-base/notifications/react";
import { SettingsList } from "@vrooli/react-component-library/SettingsList/1";
import { SettingsIntro } from "./primitives";
import { AppNotificationSettings, type AppNotificationStrings } from "@vrooli/react-component-library/AppNotificationSettings/1";
import { NotificationPreferencesPanel } from "@vrooli/react-component-library/AppNotificationSettings/1";

const HUB_LINK_QUERY_KEY = ["scenario-link", "notification-hub"] as const;

/** A loading skeleton shaped like the populated list, so the section never flashes blank. */
function NotificationsSkeleton({ label }: { label: string }) {
  return (
    <SettingsList.Group>
      <div
        className="grid gap-5 p-1"
        data-testid="notifications-loading"
        role="status"
        aria-live="polite"
        aria-label={label}
      >
        {[0, 1].map((group) => (
          <div key={group} className="grid gap-3">
            <div className="flex items-center gap-3">
              <div className="h-9 w-9 shrink-0 animate-pulse rounded-full bg-wc-surface-input" />
              <div className="h-3 w-32 animate-pulse rounded bg-wc-surface-input" />
            </div>
            {[0, 1].map((row) => (
              <div key={row} className="flex items-center justify-between gap-4 pl-12">
                <div className="grid flex-1 gap-2">
                  <div className="h-2.5 w-40 animate-pulse rounded bg-wc-surface-input" />
                  <div className="h-2 w-3/4 animate-pulse rounded bg-wc-surface-input" />
                </div>
                <div className="h-5 w-9 shrink-0 animate-pulse rounded-full bg-wc-surface-input" />
              </div>
            ))}
          </div>
        ))}
        <span className="sr-only">{label}</span>
      </div>
    </SettingsList.Group>
  );
}

/** A composed, centred message used for both the empty and error states. */
function NotificationsMessage({
  icon,
  title,
  body,
  children,
  testId,
}: {
  icon: React.ReactNode;
  title: string;
  body: string;
  children?: React.ReactNode;
  testId: string;
}) {
  return (
    <SettingsList.Group>
      <div
        className="flex flex-col items-center gap-3 px-6 py-10 text-center"
        role="status"
        data-testid={testId}
      >
        <span className="grid h-12 w-12 place-items-center rounded-2xl bg-wc-accent/10 text-wc-accent">
          {icon}
        </span>
        <div className="grid gap-1">
          <p className="font-semibold text-wc-text-primary">{title}</p>
          <p className="mx-auto max-w-sm text-sm text-wc-text-secondary">{body}</p>
        </div>
        {children && <div className="mt-1 flex flex-col items-center gap-2">{children}</div>}
      </div>
    </SettingsList.Group>
  );
}

function SecondaryButton({
  children,
  onClick,
  testId,
  icon,
}: {
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
      className="inline-flex items-center gap-2 rounded-lg border border-wc-default bg-transparent px-3.5 py-1.5 text-sm font-medium text-wc-text-primary transition-colors hover:bg-wc-surface-input/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-wc-accent"
    >
      {icon}
      {children}
    </button>
  );
}

export default function NotificationsSection({ open }: { open: boolean }) {
  const { t } = useTranslation();
  const { catalog, save } = useNotificationPreferences(notificationPreferencesClient, [WEB_CONSOLE_SOURCE], open);

  // The link to Notification Hub is resolved server-side so it carries the
  // right shape for the operator's current origin (tunnel subdomain on mobile,
  // localhost on the same machine). A sibling subdomain is also a distinct
  // origin, which is what lets an installed Notification Hub PWA open instead of
  // a bare browser tab.
  const hubLink = useQuery({
    queryKey: HUB_LINK_QUERY_KEY,
    queryFn: () => fetchScenarioLink("notification-hub"),
    enabled: open,
    staleTime: 5 * 60_000,
    retry: false,
  });

  const preferences = useMemo(() => catalog.data?.preferences ?? [], [catalog.data]);
  const sources = catalog.data?.sources ?? [];

  const unavailable = catalog.error instanceof NotificationsUnavailableError;
  const allLabel = t(strings.settings.notificationsSection.allCategory);

  const refetch = () => {
    void catalog.refetch();
  };

  return (
    <SettingsList layout="aligned" variant="flush">
      <SettingsIntro
        eyebrow={t(strings.settings.notificationsSection.eyebrow)}
        title={t(strings.settings.notificationsSection.title)}
        description={t(strings.settings.notificationsSection.description)}
      />

      <AppNotificationsGroup />

      {unavailable ? (
        <NotificationsMessage
          testId="notifications-error"
          icon={<BellOff className="h-6 w-6" aria-hidden="true" />}
          title={t(strings.settings.notificationsSection.error.title)}
          body={t(strings.settings.notificationsSection.error.body)}
        >
          <SecondaryButton
            testId="notifications-retry"
            onClick={refetch}
            icon={
              <RefreshCw
                className={`h-4 w-4 ${catalog.isFetching ? "animate-spin" : ""}`}
                aria-hidden="true"
              />
            }
          >
            {t(strings.settings.notificationsSection.error.retry)}
          </SecondaryButton>
        </NotificationsMessage>
      ) : catalog.isLoading ? (
        <NotificationsSkeleton label={t(strings.settings.notificationsSection.loading.label)} />
      ) : sources.length === 0 ? (
        <NotificationsMessage
          testId="notifications-empty"
          icon={<Bell className="h-6 w-6" aria-hidden="true" />}
          title={t(strings.settings.notificationsSection.empty.title)}
          body={t(strings.settings.notificationsSection.empty.body)}
        >
          <SecondaryButton
            testId="notifications-refresh"
            onClick={refetch}
            icon={
              <RefreshCw
                className={`h-4 w-4 ${catalog.isFetching ? "animate-spin" : ""}`}
                aria-hidden="true"
              />
            }
          >
            {t(strings.settings.notificationsSection.empty.refresh)}
          </SecondaryButton>
          {hubLink.data ? (
            <a
              href={hubLink.data}
              target="_blank"
              rel="noopener noreferrer"
              data-testid="notifications-hub-link"
              className="inline-flex items-center gap-1.5 text-xs font-medium text-wc-accent hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-wc-accent rounded"
            >
              {t(strings.settings.notificationsSection.empty.setup)}
              <ExternalLink className="h-3 w-3" aria-hidden="true" />
            </a>
          ) : (
            <p className="text-xs text-wc-text-faint">
              {t(strings.settings.notificationsSection.empty.setup)}
            </p>
          )}
        </NotificationsMessage>
      ) : (
        <NotificationPreferencesPanel
          groups={notificationPreferenceControlGroups(sources, preferences, allLabel, (preference) => save.mutate(preference))}
          disabled={save.isPending}
          strings={{
            minimumUrgency: t(strings.settings.notificationsSection.minUrgencyLabel),
            urgency: {
              low: t(strings.settings.notificationsSection.urgency.low),
              normal: t(strings.settings.notificationsSection.urgency.normal),
              high: t(strings.settings.notificationsSection.urgency.high),
              critical: t(strings.settings.notificationsSection.urgency.critical),
            },
          }}
        />
      )}
      {sources.length > 0 && hubLink.data ? (
        <div className="flex justify-center">
          <a
            href={hubLink.data}
            target="_blank"
            rel="noopener noreferrer"
            data-testid="notifications-open-hub-link"
            className="inline-flex items-center gap-1.5 text-xs font-medium text-wc-accent hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-wc-accent rounded"
          >
            {t(strings.settings.notificationsSection.openHub)}
            <ExternalLink className="h-3 w-3" aria-hidden="true" />
          </a>
        </div>
      ) : null}
    </SettingsList>
  );
}

const APP_NOTIFICATION_STRING_KEYS = [
  "title", "loading", "loadError", "retry", "installFirst", "unsupported", "ineligible", "offDescription",
  "turnOn", "turningOn", "chooseTitle", "chooseDescription", "bestMatch", "alsoHere", "standaloneLabel",
  "standaloneDescription", "confirm", "cancel", "onDescription", "onStandaloneDescription", "turnOff",
  "turningOff", "saveError", "statusOn", "statusOff", "statusPaused", "statusUnavailable",
  "mutedDescription", "pausedDescription", "lastSeen",
] as const satisfies readonly (keyof AppNotificationStrings & keyof typeof strings.settings.notificationsSection.appNotifications)[];

/** Web Console's own app can show its notifications instead of Chimesail; categories stay in the list above. */
function AppNotificationsGroup() {
  const { t } = useTranslation();
  const translated = useMemo(
    () => Object.fromEntries(APP_NOTIFICATION_STRING_KEYS.map((key) => [key, t(strings.settings.notificationsSection.appNotifications[key])])) as Partial<AppNotificationStrings>,
    [t],
  );
  return (
    <SettingsList.Group>
      <AppNotificationSettings showCategories={false} strings={translated} />
    </SettingsList.Group>
  );
}
