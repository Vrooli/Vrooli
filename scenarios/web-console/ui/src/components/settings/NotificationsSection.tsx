import { useMemo } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { Bell, BellOff, ExternalLink, RefreshCw } from "lucide-react";
import { strings } from "../../consts/strings";
import {
  fetchNotificationCatalog,
  fetchScenarioLink,
  updateNotificationPreference,
  NotificationsUnavailableError,
  WEB_CONSOLE_SOURCE,
  type NotificationCategory,
  type NotificationPreference,
  type NotificationSource,
} from "../../api/notifications";
import { SettingsList } from "@vrooli/react-component-library/SettingsList/1";
import { Avatar } from "@vrooli/react-component-library/Avatar/1";
import { SettingsToggle, SettingsIntro } from "./primitives";
import { AppNotificationSettings, type AppNotificationStrings } from "@vrooli/react-component-library/AppNotificationSettings/1";

const URGENCY_LADDER = ["low", "normal", "high", "critical"] as const;

const CATALOG_QUERY_KEY = ["notification-catalog", WEB_CONSOLE_SOURCE] as const;
const HUB_LINK_QUERY_KEY = ["scenario-link", "notification-hub"] as const;

/**
 * A source may declare an empty category list — it only offers a coarse,
 * source-wide on/off. We synthesise a single pseudo-category with the empty
 * string id so the row maps onto the source-wide preference row the hub stores.
 */
function categoriesFor(source: NotificationSource, allLabel: string): NotificationCategory[] {
  if (source.categories.length > 0) return source.categories;
  return [
    {
      category: "",
      display_name: allLabel,
      description: "",
      declared: source.declared,
      default_enabled: true,
      default_min_urgency: "normal",
    },
  ];
}

/** A category the source never declared has no display name; show its key as words instead of a blank row. */
function categoryLabel(category: NotificationCategory, allLabel: string): string {
  if (category.display_name) return category.display_name;
  const words = category.category.replace(/[._-]+/g, " ").trim();
  return words ? words.charAt(0).toUpperCase() + words.slice(1) : allLabel;
}

/** Resolve the effective preference for a (source, category), falling back to declared defaults. */
function effectivePreference(
  source: NotificationSource,
  category: NotificationCategory,
  preferences: NotificationPreference[],
): NotificationPreference {
  const existing = preferences.find(
    (p) =>
      p.source_scenario === source.source_scenario &&
      p.category === category.category &&
      // Session-scoped overrides (scope_key set) are not the global row; they
      // are managed per-session from the session menu, not this settings list.
      !p.scope_key,
  );
  return (
    existing ?? {
      source_scenario: source.source_scenario,
      category: category.category,
      enabled: category.default_enabled,
      min_urgency: category.default_min_urgency || "normal",
      channel_override: "",
    }
  );
}

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
  const queryClient = useQueryClient();

  const catalog = useQuery({
    queryKey: CATALOG_QUERY_KEY,
    queryFn: () => fetchNotificationCatalog(WEB_CONSOLE_SOURCE),
    enabled: open,
    retry: (failureCount, error) =>
      !(error instanceof NotificationsUnavailableError) && failureCount < 2,
  });

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

  const mutation = useMutation({
    mutationFn: updateNotificationPreference,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: CATALOG_QUERY_KEY });
    },
  });

  const preferences = useMemo(() => catalog.data?.preferences ?? [], [catalog.data]);
  const sources = catalog.data?.sources ?? [];

  const unavailable = catalog.error instanceof NotificationsUnavailableError;
  const allLabel = t(strings.settings.notificationsSection.allCategory);

  const refetch = () => {
    void catalog.refetch();
  };

  return (
    <SettingsList>
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
        sources.map((source) => (
          <SettingsList.Group
            key={source.source_scenario}
            label={
              <span className="flex items-center gap-2.5">
                <Avatar name={source.display_name} size="xs" shape="rounded" />
                <span className="normal-case tracking-normal">{source.display_name}</span>
              </span>
            }
          >
            {categoriesFor(source, allLabel).map((category) => {
              const pref = effectivePreference(source, category, preferences);
              return (
                <SettingsList.Row
                  key={`${source.source_scenario}:${category.category}`}
                  label={categoryLabel(category, allLabel)}
                  hint={
                    <div className="grid gap-2">
                      {category.description ? <span>{category.description}</span> : null}
                      {pref.enabled ? (
                        <label className="flex items-center gap-2 text-xs text-wc-text-secondary">
                          {t(strings.settings.notificationsSection.minUrgencyLabel)}
                          <select
                            data-testid={`notification-urgency-${source.source_scenario}-${category.category}`}
                            className="h-8 rounded-md border border-wc-default bg-wc-surface-input px-2 text-sm text-wc-text-primary"
                            disabled={mutation.isPending}
                            value={pref.min_urgency || "normal"}
                            onChange={(e) => {
                              mutation.mutate({ ...pref, min_urgency: e.target.value });
                            }}
                          >
                            {URGENCY_LADDER.map((level) => (
                              <option key={level} value={level}>
                                {t(strings.settings.notificationsSection.urgency[level])}
                              </option>
                            ))}
                          </select>
                        </label>
                      ) : null}
                    </div>
                  }
                  control="compact"
                >
                  <SettingsToggle
                    testId={`notification-enable-${source.source_scenario}-${category.category}`}
                    ariaLabel={categoryLabel(category, allLabel)}
                    checked={pref.enabled}
                    disabled={mutation.isPending}
                    onCheckedChange={(enabled) => {
                      mutation.mutate({ ...pref, enabled });
                    }}
                  />
                </SettingsList.Row>
              );
            })}
          </SettingsList.Group>
        ))
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
