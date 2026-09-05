import { useQuery } from "@tanstack/react-query";
import type { ReactNode } from "react";

import { Button } from "@vrooli/react-component-library/Button/2";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";

import { consoleApi, consoleKeys } from "../api/console";
import { fetchHealth } from "../api/health";
import { Page } from "../components/console/Page";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "../i18n";
import { formatDate } from "../i18n/format";
import { useTheme } from "../theme/ThemeProvider";

const THEME_CHOICES = [
  { choice: "light" as const, label: strings.theme.choice.light },
  { choice: "dark" as const, label: strings.theme.choice.dark },
  { choice: "system" as const, label: strings.theme.choice.system },
];

/**
 * Settings: appearance and language for this browser, then the facts an
 * operator needs about this install — which descriptors booted, whether
 * metering is configured, and where the API stands. Nothing here pretends to
 * be a control it is not.
 */
export function SettingsPage() {
  const { t } = useTranslation();
  const currentLocale = getCurrentLocale();
  const { choice, setTheme } = useTheme();
  const channels = useQuery({ queryKey: consoleKeys.channels, queryFn: ({ signal }) => consoleApi.channels(signal), staleTime: 60_000 });
  const health = useQuery({ queryKey: ["health"], queryFn: fetchHealth, refetchInterval: 30_000 });

  return (
    <Page testId={selectors.pages.settings} headingId="settings-heading" title={t(strings.pages.settings.title)} description={t(strings.pages.settings.description)}>
      <div className="flex max-w-3xl flex-col divide-y divide-app-border">
        <Section title={t(strings.pages.settings.themeHeading)} description={t(strings.pages.settings.themeDescription)}>
          <div data-testid="settings-theme" role="radiogroup" aria-label={t(strings.theme.switcherLabel)} className="flex flex-wrap gap-2">
            {THEME_CHOICES.map(({ choice: themeChoice, label }) => (
              <Button key={themeChoice} type="button" variant={choice === themeChoice ? "primary" : "secondary"} size="sm" role="radio" aria-checked={choice === themeChoice} onClick={() => setTheme(themeChoice)} data-testid={selectors.settingsPage.themeOption({ choice: themeChoice })}>
                {t(label)}
              </Button>
            ))}
          </div>
        </Section>

        <Section title={t(strings.pages.settings.localeHeading)} description={t(strings.pages.settings.localeDescription)}>
          <div role="radiogroup" aria-label={t(strings.locale.switcherLabel)} className="flex flex-wrap gap-2">
            {SUPPORTED_LOCALES.map((lng) => (
              <Button key={lng} type="button" variant={currentLocale === lng ? "primary" : "secondary"} size="sm" role="radio" aria-checked={currentLocale === lng} onClick={() => void setLocale(lng)} data-testid={selectors.settingsPage.localeOption({ code: lng })}>
                {getLocaleConfig(lng).nativeLabel}
              </Button>
            ))}
          </div>
        </Section>

        <Section title={t(strings.pages.settings.descriptorHeading)} description={t(strings.console.channels.descriptorLoaded)} testId="settings-descriptor-region" surfaceId="descriptor-region" state={channels.isPending ? "loading" : channels.isError ? "error" : "ready"}>
          <p data-testid="settings-descriptor-status" role="status" className="text-sm text-app-foreground">
            {channels.data ? t(strings.pages.settings.descriptorStatus, { count: channels.data.length }) : channels.isError ? t(strings.console.region.errorTitle) : t(strings.console.region.loading)}
          </p>
          {channels.data ? (
            <ul className="mt-2 flex flex-wrap gap-1.5">
              {channels.data.map((channel) => (
                <li key={channel.descriptor.id} className="inline-flex items-center gap-1.5 rounded-pill border border-app-border bg-app-surface px-2 py-0.5 font-mono text-xs">
                  <span aria-hidden="true" className="h-2 w-2 rounded-full" style={{ background: channel.descriptor.accent ?? "var(--color-accent)" }} />
                  {channel.descriptor.id}
                </li>
              ))}
            </ul>
          ) : null}
        </Section>

        <Section title={t(strings.pages.settings.trustHeading)} description={t(strings.pages.settings.trustDescription)} testId="settings-blast-radius">
          <ul className="flex flex-col gap-1.5 text-sm text-app-foreground">
            <li>{t(strings.pages.settings.trustRule1)}</li>
            <li>{t(strings.pages.settings.trustRule2)}</li>
            <li>{t(strings.pages.settings.trustRule3)}</li>
          </ul>
        </Section>

        <Section title={t(strings.pages.settings.meteringHeading)} description={t(strings.pages.settings.meteringDescription)} testId="settings-metering-region" surfaceId="metering-region" state="ready">
          <p data-testid="settings-byok-note" className="text-sm text-app-foreground">
            {t(strings.pages.settings.byokNote)}
          </p>
        </Section>

        <Section title={t(strings.pages.settings.aboutHeading)}>
          <dl className="grid gap-2 text-sm sm:grid-cols-3">
            <div>
              <dt className="text-xs text-app-muted-foreground">{t(strings.health.title)}</dt>
              <dd className="mt-0.5">
                {health.data ? (
                  <StatusBadge tone={health.data.status === "ok" || health.data.status === "healthy" ? "success" : "warning"}>{health.data.status}</StatusBadge>
                ) : health.isError ? (
                  <StatusBadge tone="danger">{t(strings.console.shell.apiUnreachable)}</StatusBadge>
                ) : (
                  <span className="text-app-muted-foreground">{t(strings.health.loading)}</span>
                )}
              </dd>
            </div>
            <div>
              <dt className="text-xs text-app-muted-foreground">{t(strings.health.serviceLabel)}</dt>
              <dd className="mt-0.5 font-mono text-sm">{health.data?.service ?? "—"}</dd>
            </div>
            <div>
              <dt className="text-xs text-app-muted-foreground">{t(strings.health.timestampLabel)}</dt>
              <dd className="mt-0.5 text-sm">{health.data ? formatDate(new Date(health.data.timestamp), { dateStyle: "medium", timeStyle: "short" }) : "—"}</dd>
            </div>
          </dl>
        </Section>
      </div>
    </Page>
  );
}

function Section({ title, description, children, testId, surfaceId, state }: { title: string; description?: string; children: ReactNode; testId?: string; surfaceId?: string; state?: "loading" | "ready" | "error" }) {
  return (
    <section data-testid={testId} data-experience-surface={surfaceId} data-experience-state={surfaceId ? state : undefined} className="grid gap-3 py-6 first:pt-0 md:grid-cols-[14rem_minmax(0,1fr)] md:gap-8">
      <div>
        <h3 className="text-sm font-semibold text-app-foreground">{title}</h3>
        {description ? <p className="mt-1 text-xs text-app-muted-foreground">{description}</p> : null}
      </div>
      <div className="min-w-0">{children}</div>
    </section>
  );
}
