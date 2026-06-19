import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckCircle2, RefreshCw, RotateCcw } from "lucide-react";
import { Mode } from "@vrooli/proto-types/tunnel-manager/v1/config/config_pb";

import { Button } from "../components/ui/button";
import { QueryState } from "../components/ui/QueryState";
import { StatusBadge, type BadgeTone } from "../components/ui/StatusBadge";
import { configClient } from "../api/config";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "../i18n";
import { errorMessage } from "../lib/errorMessage";
import { useTheme, type ThemeChoice } from "../theme/ThemeProvider";
import { THEME_CHOICE_LABEL } from "../theme/themeChoiceLabels";

const THEME_CHOICES: readonly ThemeChoice[] = ["light", "dark", "system"];
const CONFIG_QUERY_KEY = ["config-state"] as const;

function modeLabelKey(mode: Mode) {
  if (mode === Mode.REMOTE) return strings.config.mode.remote;
  if (mode === Mode.LOCAL) return strings.config.mode.local;
  return strings.config.mode.unspecified;
}

function readinessTone(syncReady: boolean, remoteAvailable: boolean): BadgeTone {
  if (syncReady && remoteAvailable) return "success";
  if (syncReady) return "info";
  return "warning";
}

function syncSummary(resp: Awaited<ReturnType<typeof configClient.sync>>, t: ReturnType<typeof useTranslation>["t"]) {
  if (resp.message) return resp.message;
  if (resp.setupRequired) {
    return t(strings.config.syncSetupRequired, { fields: resp.missingFields.join(", ") });
  }
  if (resp.noChanges) return t(strings.config.syncNoChanges);
  return t(strings.config.syncChanged, {
    added: resp.added.length,
    removed: resp.removed.length,
  });
}

/**
 * Settings page. Owns first-run setup and operator preferences: config
 * readiness, local/remote mode, ingress sync, theme, and locale.
 */
export function SettingsPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const currentLocale = getCurrentLocale();
  const { choice, setTheme } = useTheme();

  const configQuery = useQuery({ queryKey: CONFIG_QUERY_KEY, queryFn: () => configClient.getConfig({}) });
  const refreshConfig = () => queryClient.invalidateQueries({ queryKey: CONFIG_QUERY_KEY });

  const dryRunMutation = useMutation({
    mutationFn: () => configClient.sync({ dryRun: true }),
  });
  const syncMutation = useMutation({
    mutationFn: () => configClient.sync({ dryRun: false }),
    onSuccess: () => void refreshConfig(),
  });
  const switchModeMutation = useMutation({
    mutationFn: (targetMode: Mode) => configClient.switchMode({ targetMode }),
    onSuccess: () => void refreshConfig(),
  });

  const config = configQuery.data?.config;
  const readiness = configQuery.data?.readiness;
  const syncResult = syncMutation.data ?? dryRunMutation.data;
  const actionError = dryRunMutation.error ?? syncMutation.error ?? switchModeMutation.error;
  const actionPending = dryRunMutation.isPending || syncMutation.isPending || switchModeMutation.isPending;

  return (
    <section
      data-testid={selectors.pages.settings}
      aria-labelledby="settings-heading"
      className="flex flex-col gap-6"
    >
      <h2 id="settings-heading" className="text-2xl font-semibold">
        {t(strings.pages.settings.title)}
      </h2>

      <section
        data-testid={selectors.settingsPage.configPanel}
        className="flex flex-col gap-4 rounded-panel border border-app-border bg-app-surface p-4"
      >
        <div className="flex flex-col gap-1">
          <h3 className="text-lg font-semibold">{t(strings.config.setupHeading)}</h3>
          <p className="text-sm text-app-muted-foreground">{t(strings.config.setupDescription)}</p>
        </div>
        <QueryState
          isLoading={configQuery.isLoading}
          error={configQuery.error}
          loadingLabel={t(strings.config.loading)}
          errorLabel={t(strings.config.error)}
        >
          {config && readiness && (
            <>
              <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
                <ReadinessItem
                  label={t(strings.config.currentMode)}
                  value={t(modeLabelKey(config.mode))}
                  testId={selectors.settingsPage.currentMode}
                />
                <ReadinessItem
                  label={t(strings.config.remoteAvailability)}
                  value={readiness.remoteAvailable ? t(strings.config.available) : t(strings.config.unavailable)}
                  tone={readiness.remoteAvailable ? "success" : "warning"}
                  testId={selectors.settingsPage.remoteAvailable}
                />
                <ReadinessItem
                  label={t(strings.config.credentialSource)}
                  value={readiness.credentialSource || t(strings.config.none)}
                  testId={selectors.settingsPage.credentialSource}
                />
                <ReadinessItem
                  label={t(strings.config.syncReadiness)}
                  value={readiness.syncReady ? t(strings.config.ready) : t(strings.config.setupRequired)}
                  tone={readinessTone(readiness.syncReady, readiness.remoteAvailable)}
                  testId={selectors.settingsPage.syncReady}
                />
              </div>

              <dl className="grid gap-3 text-sm md:grid-cols-2">
                <Detail label={t(strings.config.tunnelId)} value={config.tunnelId || t(strings.config.notSet)} />
                <Detail label={t(strings.config.accountId)} value={config.accountId || t(strings.config.notSet)} />
                <Detail label={t(strings.config.credentialRef)} value={readiness.credentialRef || t(strings.config.notSet)} />
                <Detail label={t(strings.config.localConfigPath)} value={readiness.localConfigPath || t(strings.config.notSet)} />
                <Detail label={t(strings.config.metricsEndpoint)} value={config.promEndpoint || t(strings.config.notSet)} />
              </dl>

              {readiness.missingFields.length > 0 && (
                <p data-testid={selectors.settingsPage.missingFields} className="text-sm text-app-warning">
                  {t(strings.config.missingFields, { fields: readiness.missingFields.join(", ") })}
                </p>
              )}

              <div className="flex flex-wrap gap-2 border-t border-app-border pt-4">
                <Button
                  type="button"
                  variant={config.mode === Mode.LOCAL ? "default" : "outline"}
                  disabled={actionPending || config.mode === Mode.LOCAL}
                  data-testid={selectors.settingsPage.localModeButton}
                  onClick={() => switchModeMutation.mutate(Mode.LOCAL)}
                >
                  <RotateCcw aria-hidden="true" className="mr-2 h-4 w-4" />
                  {t(strings.config.useLocalMode)}
                </Button>
                <Button
                  type="button"
                  variant={config.mode === Mode.REMOTE ? "default" : "outline"}
                  disabled={actionPending || config.mode === Mode.REMOTE || !readiness.remoteAvailable}
                  data-testid={selectors.settingsPage.remoteModeButton}
                  onClick={() => switchModeMutation.mutate(Mode.REMOTE)}
                >
                  <CheckCircle2 aria-hidden="true" className="mr-2 h-4 w-4" />
                  {t(strings.config.useRemoteMode)}
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  disabled={actionPending}
                  data-testid={selectors.settingsPage.syncDryRunButton}
                  onClick={() => dryRunMutation.mutate()}
                >
                  <RefreshCw aria-hidden="true" className="mr-2 h-4 w-4" />
                  {t(strings.config.syncDryRun)}
                </Button>
                <Button
                  type="button"
                  disabled={actionPending || !readiness.syncReady}
                  data-testid={selectors.settingsPage.syncApplyButton}
                  onClick={() => syncMutation.mutate()}
                >
                  <CheckCircle2 aria-hidden="true" className="mr-2 h-4 w-4" />
                  {t(strings.config.syncApply)}
                </Button>
              </div>

              {syncResult && (
                <p data-testid={selectors.settingsPage.syncResult} className="text-sm text-app-muted-foreground">
                  {syncSummary(syncResult, t)}
                </p>
              )}
              {actionError && (
                <p data-testid={selectors.settingsPage.actionError} role="alert" className="text-sm text-app-danger">
                  {errorMessage(actionError, t)}
                </p>
              )}
            </>
          )}
        </QueryState>
      </section>

      <div className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.settings.themeHeading)}
        </h3>
        <div role="radiogroup" aria-label={t(strings.theme.switcherLabel)} className="flex gap-2">
          {THEME_CHOICES.map((c) => (
            <button
              key={c}
              type="button"
              role="radio"
              aria-checked={choice === c}
              onClick={() => setTheme(c)}
              data-testid={selectors.settingsPage.themeOption({ choice: c })}
              className={
                choice === c
                  ? "rounded-control bg-app-primary px-3 py-1 text-sm font-medium text-app-primary-foreground"
                  : "rounded-control border border-app-border px-3 py-1 text-sm text-app-foreground hover:bg-app-surface-muted"
              }
            >
              {t(THEME_CHOICE_LABEL[c])}
            </button>
          ))}
        </div>
      </div>

      <div className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.settings.localeHeading)}
        </h3>
        <div role="radiogroup" aria-label={t(strings.locale.switcherLabel)} className="flex gap-2">
          {SUPPORTED_LOCALES.map((lng) => (
            <button
              key={lng}
              type="button"
              role="radio"
              aria-checked={currentLocale === lng}
              onClick={() => void setLocale(lng)}
              data-testid={selectors.settingsPage.localeOption({ code: lng })}
              className={
                currentLocale === lng
                  ? "rounded-control bg-app-primary px-3 py-1 text-sm font-medium text-app-primary-foreground"
                  : "rounded-control border border-app-border px-3 py-1 text-sm text-app-foreground hover:bg-app-surface-muted"
              }
            >
              {getLocaleConfig(lng).nativeLabel}
            </button>
          ))}
        </div>
      </div>
    </section>
  );
}

function ReadinessItem({
  label,
  value,
  tone = "neutral",
  testId,
}: {
  label: string;
  value: string;
  tone?: BadgeTone;
  testId: string;
}) {
  return (
    <div className="flex flex-col gap-1 rounded-control border border-app-border bg-app-surface-muted p-3">
      <span className="text-xs uppercase text-app-muted-foreground">{label}</span>
      <StatusBadge tone={tone} data-testid={testId}>
        {value}
      </StatusBadge>
    </div>
  );
}

function Detail({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <dt className="text-xs uppercase text-app-muted-foreground">{label}</dt>
      <dd className="break-words font-medium">{value}</dd>
    </div>
  );
}
