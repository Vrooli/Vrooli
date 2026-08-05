import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Link } from "react-router-dom";
import { CheckCircle2, GitCompareArrows, Globe, KeyRound, RefreshCw, RotateCcw, ShieldCheck, Trash2 } from "lucide-react";
import { Mode, type ConfigReadiness, type CredentialFieldStatus } from "@vrooli/proto-types/tunnel-manager/v1/config/config_pb";

import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { QueryState } from "../components/ui/QueryState";
import { StatusBadge, type BadgeTone } from "../components/ui/StatusBadge";
import { configClient } from "../api/config";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "../i18n";
import { errorMessage } from "../lib/errorMessage";
import type { ThemeChoice } from "../theme/themeContext";
import { THEME_CHOICE_LABEL } from "../theme/themeChoiceLabels";
import { useTheme } from "../theme/useTheme";

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

function hasReadOnlyCredentialField(fields: readonly CredentialFieldStatus[]) {
  return fields.some((field) => field.present && !field.writable);
}

function credentialNextActionKey(readiness: ConfigReadiness) {
  if (readiness.missingFields.length > 0) return strings.config.nextActionMissing;
  if (!readiness.remoteAvailable) return strings.config.nextActionUnavailable;
  if (readiness.desiredMode === Mode.REMOTE) return strings.config.nextActionRemote;
  return strings.config.nextActionReady;
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
  const [accountId, setAccountId] = useState("");
  const [tunnelId, setTunnelId] = useState("");
  const [apiToken, setApiToken] = useState("");

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
  const reconcileMutation = useMutation({
    mutationFn: () => configClient.sync({}),
    onSuccess: () => void refreshConfig(),
  });
  const saveCredentialsMutation = useMutation({
    mutationFn: () => configClient.setCloudflareCredentials({ accountId, tunnelId, apiToken }),
    onSuccess: () => {
      setApiToken("");
      void refreshConfig();
    },
  });
  const clearCredentialsMutation = useMutation({
    mutationFn: () => configClient.clearCloudflareCredentials({ fields: ["all"] }),
    onSuccess: () => {
      setAccountId("");
      setTunnelId("");
      setApiToken("");
      void refreshConfig();
    },
  });
  const publicExposureMutation = useMutation({
    mutationFn: (enabled: boolean) => configClient.setPublicExposure({ enabled }),
    onSuccess: () => void refreshConfig(),
  });

  const config = configQuery.data?.config;
  const readiness = configQuery.data?.readiness;
  const syncResult = syncMutation.data ?? dryRunMutation.data;
  const credentialFields = readiness?.credentialFields ?? [];
  const hasReadOnlyCredential = hasReadOnlyCredentialField(credentialFields);
  const credentialMutationResult = saveCredentialsMutation.data ?? clearCredentialsMutation.data;
  const actionError =
    dryRunMutation.error ??
    syncMutation.error ??
    switchModeMutation.error ??
    reconcileMutation.error ??
    saveCredentialsMutation.error ??
    clearCredentialsMutation.error ??
    publicExposureMutation.error;
  const actionPending =
    dryRunMutation.isPending ||
    syncMutation.isPending ||
    switchModeMutation.isPending ||
    reconcileMutation.isPending ||
    saveCredentialsMutation.isPending ||
    clearCredentialsMutation.isPending ||
    publicExposureMutation.isPending;
  const reconcileResult = reconcileMutation.data;

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

              <form
                data-testid={selectors.settingsPage.credentialsForm}
                className="grid gap-3 border-t border-app-border pt-4 md:grid-cols-3"
                onSubmit={(event) => {
                  event.preventDefault();
                  saveCredentialsMutation.mutate();
                }}
              >
                <div className="md:col-span-3">
                  <h4 className="text-sm font-semibold">{t(strings.config.credentialsHeading)}</h4>
                  <p className="text-sm text-app-muted-foreground">{t(strings.config.credentialsDescription)}</p>
                </div>
                <div
                  data-testid={selectors.settingsPage.credentialPolicy}
                  className="flex gap-3 rounded-control border border-app-border bg-app-surface-muted p-3 text-sm md:col-span-3"
                >
                  <ShieldCheck aria-hidden="true" className="mt-0.5 h-4 w-4 shrink-0 text-app-primary" />
                  <div className="flex flex-col gap-1">
                    <span className="font-medium">{t(strings.config.credentialPolicyTitle)}</span>
                    <span className="text-app-muted-foreground">{t(strings.config.credentialPolicyBody)}</span>
                    <span className="text-app-muted-foreground">{t(strings.config.cloudflareTokenHelp)}</span>
                  </div>
                </div>
                <p
                  data-testid={selectors.settingsPage.credentialNextAction}
                  className="rounded-control border border-app-border bg-app-background p-3 text-sm text-app-muted-foreground md:col-span-3"
                >
                  {t(credentialNextActionKey(readiness))}
                </p>
                {hasReadOnlyCredential && (
                  <p
                    data-testid={selectors.settingsPage.credentialShadowWarning}
                    className="rounded-control border border-app-warning/40 bg-app-warning/10 p-3 text-sm text-app-warning md:col-span-3"
                  >
                    {t(strings.config.credentialReadOnlyWarning)}
                  </p>
                )}
                <label className="flex flex-col gap-1 text-sm">
                  {t(strings.config.accountId)}
                  <Input
                    value={accountId}
                    onChange={(event) => setAccountId(event.target.value)}
                    data-testid={selectors.settingsPage.accountIdInput}
                    autoComplete="off"
                  />
                </label>
                <label className="flex flex-col gap-1 text-sm">
                  {t(strings.config.tunnelId)}
                  <Input
                    value={tunnelId}
                    onChange={(event) => setTunnelId(event.target.value)}
                    data-testid={selectors.settingsPage.tunnelIdInput}
                    autoComplete="off"
                  />
                </label>
                <label className="flex flex-col gap-1 text-sm">
                  {t(strings.config.apiToken)}
                  <Input
                    type="password"
                    value={apiToken}
                    onChange={(event) => setApiToken(event.target.value)}
                    data-testid={selectors.settingsPage.apiTokenInput}
                    autoComplete="off"
                  />
                </label>
                <p className="text-xs text-app-muted-foreground md:col-span-3">{t(strings.config.apiTokenHelp)}</p>
                <div className="flex flex-wrap gap-2 md:col-span-3">
                  <Button
                    type="submit"
                    disabled={actionPending || (!accountId.trim() && !tunnelId.trim() && !apiToken.trim())}
                    data-testid={selectors.settingsPage.credentialsSaveButton}
                  >
                    <KeyRound aria-hidden="true" className="mr-2 h-4 w-4" />
                    {t(strings.config.saveCredentials)}
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    disabled={actionPending}
                    data-testid={selectors.settingsPage.credentialsClearButton}
                    onClick={() => clearCredentialsMutation.mutate()}
                  >
                    <Trash2 aria-hidden="true" className="mr-2 h-4 w-4" />
                    {t(strings.config.clearCredentials)}
                  </Button>
                </div>
                {credentialFields.length > 0 && (
                  <ul
                    data-testid={selectors.settingsPage.credentialFields}
                    className="grid gap-2 text-sm md:col-span-3 md:grid-cols-3"
                    aria-label={t(strings.config.credentialFields)}
                  >
                    {credentialFields.map((field) => (
                      <CredentialFieldItem key={field.name} field={field} t={t} />
                    ))}
                  </ul>
                )}
              </form>

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

              <div className="flex flex-col gap-3 border-t border-app-border pt-4">
                <h4 className="text-sm font-semibold">{t(strings.config.reconcileHeading)}</h4>
                <p data-testid={selectors.settingsPage.switchModeNote} className="text-sm text-app-muted-foreground">
                  {t(strings.config.switchModeNote)}
                </p>
                <div className="flex flex-wrap items-center gap-3">
                  <Button
                    type="button"
                    disabled={actionPending}
                    data-testid={selectors.settingsPage.reconcileNowButton}
                    onClick={() => reconcileMutation.mutate()}
                  >
                    <RefreshCw aria-hidden="true" className="mr-2 h-4 w-4" />
                    {t(strings.config.reconcileNow)}
                  </Button>
                  <Link
                    to="/drift"
                    data-testid={selectors.settingsPage.reviewDriftLink}
                    className="inline-flex items-center gap-2 text-sm text-app-primary underline-offset-2 hover:underline"
                  >
                    <GitCompareArrows aria-hidden="true" className="h-4 w-4" />
                    {t(strings.config.reviewDrift)}
                  </Link>
                </div>
                {reconcileResult && (
                  <p data-testid={selectors.settingsPage.reconcileResult} className="text-sm text-app-muted-foreground">
                    {t(strings.config.reconcileResult, {
                      added: reconcileResult.added.length,
                      removed: reconcileResult.removed.length,
                    })}
                  </p>
                )}
              </div>

              <div
                data-testid={selectors.settingsPage.publicExposurePanel}
                className="flex flex-col gap-3 border-t border-app-border pt-4"
              >
                <div className="flex flex-col gap-1">
                  <h4 className="text-sm font-semibold">{t(strings.config.publicExposureHeading)}</h4>
                  <p className="text-sm text-app-muted-foreground">{t(strings.config.publicExposureBody)}</p>
                </div>
                <div className="flex flex-wrap items-center gap-3">
                  <StatusBadge
                    tone={config.publicExposureEnabled ? "success" : "neutral"}
                    data-testid={selectors.settingsPage.publicExposureState}
                  >
                    {config.publicExposureEnabled
                      ? t(strings.config.publicExposureOn)
                      : t(strings.config.publicExposureOff)}
                  </StatusBadge>
                  <Button
                    type="button"
                    variant={config.publicExposureEnabled ? "outline" : "default"}
                    disabled={actionPending}
                    data-testid={selectors.settingsPage.publicExposureToggle}
                    onClick={() => publicExposureMutation.mutate(!config.publicExposureEnabled)}
                  >
                    <Globe aria-hidden="true" className="mr-2 h-4 w-4" />
                    {config.publicExposureEnabled
                      ? t(strings.config.publicExposureDisable)
                      : t(strings.config.publicExposureEnable)}
                  </Button>
                  <Link
                    to="/drift"
                    data-testid={selectors.settingsPage.publicExposureStatusLink}
                    className="inline-flex items-center gap-2 text-sm text-app-primary underline-offset-2 hover:underline"
                  >
                    <GitCompareArrows aria-hidden="true" className="h-4 w-4" />
                    {t(strings.config.publicExposureViewStatus)}
                  </Link>
                </div>
              </div>

              {credentialMutationResult && (
                <p className="text-sm text-app-muted-foreground">
                  {saveCredentialsMutation.data ? t(strings.config.credentialSaved) : t(strings.config.credentialCleared)}
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

function formatCredentialField(field: CredentialFieldStatus, t: ReturnType<typeof useTranslation>["t"]) {
  return t(strings.config.credentialFieldStatus, {
    name: field.name,
    state: field.present ? t(strings.config.credentialPresent) : t(strings.config.credentialMissing),
    source: field.source || t(strings.config.none),
    writable: field.writable ? t(strings.config.credentialWritable) : t(strings.config.credentialReadOnly),
  });
}

function CredentialFieldItem({
  field,
  t,
}: {
  field: CredentialFieldStatus;
  t: ReturnType<typeof useTranslation>["t"];
}) {
  return (
    <li className="rounded-control border border-app-border bg-app-surface-muted p-3">
      <span className="block break-words font-medium text-app-foreground">{field.name}</span>
      <span className="block text-app-muted-foreground">{formatCredentialField(field, t)}</span>
    </li>
  );
}
