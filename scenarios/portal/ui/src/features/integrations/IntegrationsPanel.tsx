import { BehaviorOverride, IntegrationState, integrationStateLabel } from "../../api/integrations";
import {
  evaluateDeploymentProfile,
  loadDeploymentProfileConfig,
  saveDeploymentProfileConfig,
  type DeploymentProfile,
} from "./deploymentProfiles";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { useIntegrationStatus } from "./useIntegrationStatus";
import { useMemo, useState } from "react";
import { Button } from "../../components/ui/button";

const STATE_KEY = {
  [IntegrationState.AVAILABLE]: strings.integrations.state.available,
  [IntegrationState.DEGRADED]: strings.integrations.state.degraded,
  [IntegrationState.UNAVAILABLE]: strings.integrations.state.unavailable,
  [IntegrationState.UNKNOWN]: strings.integrations.state.unknown,
  [IntegrationState.UNSPECIFIED]: strings.integrations.state.unspecified,
} as const satisfies Record<IntegrationState, string>;

const OVERRIDE_OPTIONS = [
  { value: BehaviorOverride.AUTO, label: strings.integrations.settings.overrideAuto },
  { value: BehaviorOverride.FORCE_OFF, label: strings.integrations.settings.overrideForceOff },
  { value: BehaviorOverride.FORCE_PASSIVE, label: strings.integrations.settings.overrideForcePassive },
] as const;

export function IntegrationsPanel() {
  const { t } = useTranslation();
  const { status, loading, error, setOverride } = useIntegrationStatus();
  const integrations = useMemo(() => status?.integrations ?? [], [status?.integrations]);
  const [profile, setProfile] = useState(() => loadDeploymentProfileConfig());
  const [endpoint, setEndpoint] = useState(profile.endpoint);
  const [profileError, setProfileError] = useState("");
  const [profileSaved, setProfileSaved] = useState(false);
  const profileStatus = useMemo(
    () => evaluateDeploymentProfile(profile, integrations.map((item) => ({ id: item.id, state: integrationStateLabel(item.state) }))),
    [integrations, profile],
  );

  const saveProfile = () => {
    try {
      const saved = saveDeploymentProfileConfig({ profile: profile.profile, endpoint });
      setProfile(saved);
      setEndpoint(saved.endpoint);
      setProfileError("");
      setProfileSaved(true);
    } catch {
      setProfileError(t(strings.integrations.settings.profileInvalidEndpoint));
      setProfileSaved(false);
    }
  };

  return (
    <section
      data-testid={selectors.integrations.settingsPanel}
      aria-labelledby="integrations-heading"
      className="flex flex-col gap-4"
    >
      <div className="flex flex-col gap-1">
        <h3 id="integrations-heading" className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.integrations.settings.title)}
        </h3>
        <p className="max-w-3xl text-sm text-app-muted-foreground">
          {t(strings.integrations.settings.description)}
        </p>
      </div>

      <div
        data-testid={selectors.integrations.deploymentProfilePanel}
        className="flex max-w-3xl flex-col gap-3 rounded-control border border-app-border p-4"
      >
        <div className="flex flex-col gap-1">
          <h4 className="text-sm font-semibold text-app-foreground">
            {t(strings.integrations.settings.profileTitle)}
          </h4>
          <p className="text-sm text-app-muted-foreground">
            {t(strings.integrations.settings.profileDescription)}
          </p>
        </div>
        <label className="flex max-w-sm flex-col gap-2 text-sm font-medium text-app-foreground">
          <span>{t(strings.integrations.settings.profileLabel)}</span>
          <select
            data-testid={selectors.integrations.deploymentProfileSelect}
            value={profile.profile}
            onChange={(event) => {
              setProfile((current) => ({ ...current, profile: event.target.value as DeploymentProfile }));
              setProfileSaved(false);
            }}
            className="rounded-control border border-app-border bg-app-surface px-3 py-2 text-app-foreground"
          >
            <option data-testid={selectors.integrations.deploymentProfileOption({ id: "client" })} value="client">
              {t(strings.integrations.settings.profileClient)}
            </option>
            <option data-testid={selectors.integrations.deploymentProfileOption({ id: "local-control" })} value="local-control">
              {t(strings.integrations.settings.profileLocalControl)}
            </option>
            <option data-testid={selectors.integrations.deploymentProfileOption({ id: "automation" })} value="automation">
              {t(strings.integrations.settings.profileAutomation)}
            </option>
          </select>
        </label>
        <label className="flex max-w-xl flex-col gap-2 text-sm font-medium text-app-foreground">
          <span>{t(strings.integrations.settings.endpointLabel)}</span>
          <input
            data-testid={selectors.integrations.deploymentEndpointInput}
            value={endpoint}
            onChange={(event) => {
              setEndpoint(event.target.value);
              setProfileSaved(false);
            }}
            placeholder={t(strings.integrations.settings.endpointPlaceholder)}
            inputMode="url"
            autoComplete="off"
            className="rounded-control border border-app-border bg-app-surface px-3 py-2 text-app-foreground"
          />
        </label>
        <div className="flex flex-wrap items-center gap-3">
          <Button data-testid={selectors.integrations.deploymentProfileSave} type="button" onClick={saveProfile}>
            {t(strings.integrations.settings.saveProfile)}
          </Button>
          <span
            data-testid={selectors.integrations.deploymentProfileStatus}
            className="text-sm text-app-muted-foreground"
          >
            {profileStatus.ready
              ? t(strings.integrations.settings.profileReady)
              : t(strings.integrations.settings.missingCapabilities, {
                  capabilities: profileStatus.missingCapabilities.join(", "),
                })}
          </span>
          {profileSaved ? <span className="text-sm text-app-success">{t(strings.integrations.settings.profileSaved)}</span> : null}
        </div>
        {profileError ? <p className="text-sm text-app-danger">{profileError}</p> : null}
      </div>

      <label className="flex max-w-sm flex-col gap-2 text-sm font-medium text-app-foreground">
        <span>{t(strings.integrations.settings.overrideLabel)}</span>
        <select
          data-testid={selectors.integrations.overrideSelect}
          value={status?.override ?? BehaviorOverride.AUTO}
          disabled={loading}
          onChange={(event) => void setOverride(Number(event.target.value))}
          className="rounded-control border border-app-border bg-app-surface px-3 py-2 text-app-foreground"
        >
          {OVERRIDE_OPTIONS.map((option) => (
            <option key={option.value} value={option.value}>
              {t(option.label)}
            </option>
          ))}
        </select>
      </label>

      {error ? (
        <p className="text-sm text-app-danger">{t(strings.integrations.settings.error)}</p>
      ) : null}

      {integrations.length === 0 && !loading ? (
        <p className="text-sm text-app-muted-foreground">{t(strings.integrations.settings.empty)}</p>
      ) : (
        <div className="overflow-x-auto">
          <table
            data-testid={selectors.integrations.statusTable}
            className="min-w-full border-collapse text-left text-sm"
          >
            <thead className="border-b border-app-border text-xs uppercase text-app-muted-foreground">
              <tr>
                <th className="py-2 pr-4">{t(strings.integrations.settings.nameHeader)}</th>
                <th className="py-2 pr-4">{t(strings.integrations.settings.stateHeader)}</th>
                <th className="py-2 pr-4">{t(strings.integrations.settings.latencyHeader)}</th>
                <th className="py-2 pr-4">{t(strings.integrations.settings.samplesHeader)}</th>
                <th className="py-2 pr-4">{t(strings.integrations.settings.reasonHeader)}</th>
              </tr>
            </thead>
            <tbody>
              {integrations.map((integration) => (
                <tr
                  key={integration.id}
                  data-testid={selectors.integrations.statusRow({ id: integration.id })}
                  className="border-b border-app-border/70"
                >
                  <td className="py-2 pr-4 font-medium text-app-foreground">{integration.displayName}</td>
                  <td className="py-2 pr-4 text-app-muted-foreground">{t(STATE_KEY[integration.state])}</td>
                  <td className="py-2 pr-4 text-app-muted-foreground">
                    {Math.round(integration.stats?.latencyP95Ms ?? 0)}
                  </td>
                  <td className="py-2 pr-4 text-app-muted-foreground">
                    {String(integration.stats?.sampleCount ?? 0n)}
                  </td>
                  <td className="py-2 pr-4 text-app-muted-foreground">{integration.reason}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}
