import { Bot, BookOpen, Mic, Search, Sparkles } from "lucide-react";
import { useMemo, useState, type ReactNode } from "react";
import {
  IntegrationsOverview,
  type IntegrationGroup,
  type IntegrationItem,
  type IntegrationStatus as OverviewStatus,
} from "@vrooli/react-component-library/IntegrationsOverview/1";
import { SettingsList } from "@vrooli/react-component-library/SettingsList/1";
import type { IntegrationStatus } from "@vrooli/proto-types/portal/v1/integrations/integrations_pb";

import { BehaviorMode, BehaviorOverride, IntegrationState, integrationStateLabel } from "../../api/integrations";
import {
  evaluateDeploymentProfile,
  loadDeploymentProfileConfig,
  saveDeploymentProfileConfig,
  type DeploymentProfile,
} from "./deploymentProfiles";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { getCurrentLocale, useTranslation } from "../../i18n";
import { useIntegrationStatus } from "./useIntegrationStatus";
import { Button } from "../../components/ui/button";

const S = strings.integrations.settings;

const MODE_KEY = {
  [BehaviorMode.OFF]: strings.integrations.mode.off,
  [BehaviorMode.PASSIVE]: strings.integrations.mode.passive,
  [BehaviorMode.FULL]: strings.integrations.mode.full,
  [BehaviorMode.UNSPECIFIED]: strings.integrations.mode.unknown,
} as const satisfies Record<BehaviorMode, string>;

const OVERRIDE_OPTIONS = [
  { value: BehaviorOverride.AUTO, label: S.overrideAuto },
  { value: BehaviorOverride.FORCE_OFF, label: S.overrideForceOff },
  { value: BehaviorOverride.FORCE_PASSIVE, label: S.overrideForcePassive },
] as const;

/** Plain-language copy per integration id; `scenario` items can be started with `vrooli scenario start`. */
const ITEM_COPY: Record<string, { copy: (typeof S.items)[keyof typeof S.items]; provider: string; icon: ReactNode; scenario: boolean }> = {
  openrouter: { copy: S.items.openrouter, provider: "OpenRouter", icon: <Sparkles aria-hidden="true" />, scenario: false },
  "agent-manager": { copy: S.items.agentManager, provider: "Agent Manager", icon: <Bot aria-hidden="true" />, scenario: true },
  "search-hub": { copy: S.items.searchHub, provider: "Search Hub", icon: <Search aria-hidden="true" />, scenario: true },
  "prompt-manager": { copy: S.items.promptManager, provider: "Prompt Manager", icon: <BookOpen aria-hidden="true" />, scenario: true },
  "audio-tools": { copy: S.items.audioTools, provider: "Audio Tools", icon: <Mic aria-hidden="true" />, scenario: true },
};

function overviewStatus(state: IntegrationState): OverviewStatus {
  switch (state) {
    case IntegrationState.AVAILABLE:
      return "ready";
    case IntegrationState.DEGRADED:
      return "limited";
    case IntegrationState.UNAVAILABLE:
      return "off";
    default:
      return "unknown";
  }
}

function humanize(slug: string): string {
  const words = slug.replace(/[-_]+/g, " ").trim();
  return words.charAt(0).toUpperCase() + words.slice(1);
}

function formatTime(value: string | undefined, locale: string): string | undefined {
  if (!value) return undefined;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return undefined;
  return date.toLocaleString(locale, { dateStyle: "medium", timeStyle: "short" });
}

export function IntegrationsPanel() {
  const { t } = useTranslation();
  const locale = getCurrentLocale();
  const { status, loading, error, refresh, setOverride } = useIntegrationStatus();
  const integrations = useMemo(() => status?.integrations ?? [], [status?.integrations]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
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
      setProfileError(t(S.profileInvalidEndpoint));
      setProfileSaved(false);
    }
  };

  const searchMode = (
    <SettingsList>
      <SettingsList.Group>
        <SettingsList.Row
          label={t(S.searchModeLabel)}
          hint={t(S.searchModeHint, { mode: t(MODE_KEY[status?.activeMode ?? BehaviorMode.UNSPECIFIED]) })}
        >
          <select
            data-testid={selectors.integrations.overrideSelect}
            value={status?.override ?? BehaviorOverride.AUTO}
            disabled={loading}
            onChange={(event) => void setOverride(Number(event.target.value))}
            className="rounded-control border border-app-border bg-app-surface px-3 py-2 text-sm text-app-foreground"
          >
            {OVERRIDE_OPTIONS.map((option) => (
              <option key={option.value} value={option.value}>
                {t(option.label)}
              </option>
            ))}
          </select>
        </SettingsList.Row>
      </SettingsList.Group>
    </SettingsList>
  );

  const toItem = (integration: IntegrationStatus): IntegrationItem => {
    const known = ITEM_COPY[integration.id];
    const name = known ? t(known.copy.name) : humanize(integration.displayName || integration.id);
    const state = overviewStatus(integration.state);
    const stats = integration.stats;
    const lastOk = formatTime(stats?.lastOkAt, locale);
    const features = known
      ? Object.entries(known.copy.features).map(([id, key]) => ({ id, label: t(key), available: state !== "off" }))
      : undefined;
    return {
      id: integration.id,
      name,
      subtitle: known ? t(known.copy.subtitle) : undefined,
      provider: known?.provider ?? humanize(integration.id),
      icon: known?.icon,
      initials: known ? undefined : name.slice(0, 2).toUpperCase(),
      status: state,
      statusLabel: state === "limited" ? t(S.statusLimited) : undefined,
      problem: state === "off"
        ? { title: t(S.problemOff, { name }), description: known ? t(known.copy.problem) : t(S.problemGeneric, { name }) }
        : state === "limited"
          ? { title: t(S.problemLimited, { name }), description: t(S.problemLimitedDescription) }
          : undefined,
      features,
      facts: [
        { label: t(S.facts.id), value: integration.id, monospace: true },
        ...(integration.reason ? [{ label: t(S.facts.health), value: integration.reason }] : []),
        ...(stats?.sampleCount ? [
          { label: t(S.facts.latency), value: t(S.facts.latencyValue, { ms: Math.round(stats.latencyP95Ms) }) },
          { label: t(S.facts.samples), value: String(stats.sampleCount) },
        ] : []),
        ...(lastOk ? [{ label: t(S.facts.lastOk), value: lastOk }] : []),
        ...(state === "off" && known?.scenario
          ? [{ label: t(S.facts.command), value: `vrooli scenario start ${integration.id}`, monospace: true }]
          : []),
      ],
      detail: integration.id === "search-hub" ? searchMode : undefined,
    };
  };

  const models = integrations.filter((integration) => integration.id === "openrouter").map(toItem);
  const apps = integrations.filter((integration) => integration.id !== "openrouter").map(toItem);
  const groups: IntegrationGroup[] = [
    ...(models.length ? [{ id: "models", label: t(S.groups.models), items: models }] : []),
    { id: "apps", label: t(S.groups.apps), items: apps },
  ];

  return (
    <section data-testid={selectors.integrations.settingsPanel} className="flex flex-col gap-6">
      <IntegrationsOverview
        header={<SettingsList.Intro title={t(S.title)} description={t(S.description)} />}
        groups={groups}
        checkedAt={status?.evaluatedAt}
        loading={loading && !status}
        error={error || null}
        refreshing={loading && Boolean(status)}
        onRefresh={() => void refresh()}
        selectedId={selectedId}
        onSelectedIdChange={setSelectedId}
        locale={locale}
      />

      {selectedId === null ? (
        <SettingsList data-testid={selectors.integrations.deploymentProfilePanel}>
          <SettingsList.Group label={t(S.profileGroup)}>
            <SettingsList.Row label={t(S.profileLabel)} hint={t(S.profileDescription)}>
              <select
                data-testid={selectors.integrations.deploymentProfileSelect}
                value={profile.profile}
                onChange={(event) => {
                  setProfile((current) => ({ ...current, profile: event.target.value as DeploymentProfile }));
                  setProfileSaved(false);
                }}
                className="rounded-control border border-app-border bg-app-surface px-3 py-2 text-sm text-app-foreground"
              >
                <option data-testid={selectors.integrations.deploymentProfileOption({ id: "client" })} value="client">
                  {t(S.profileClient)}
                </option>
                <option data-testid={selectors.integrations.deploymentProfileOption({ id: "local-control" })} value="local-control">
                  {t(S.profileLocalControl)}
                </option>
                <option data-testid={selectors.integrations.deploymentProfileOption({ id: "automation" })} value="automation">
                  {t(S.profileAutomation)}
                </option>
              </select>
            </SettingsList.Row>
            <SettingsList.Row label={t(S.endpointLabel)} control="wide">
              <input
                data-testid={selectors.integrations.deploymentEndpointInput}
                value={endpoint}
                onChange={(event) => {
                  setEndpoint(event.target.value);
                  setProfileSaved(false);
                }}
                placeholder={t(S.endpointPlaceholder)}
                inputMode="url"
                autoComplete="off"
                className="w-full rounded-control border border-app-border bg-app-surface px-3 py-2 text-sm text-app-foreground"
              />
            </SettingsList.Row>
            <SettingsList.Row
              label={
                <span data-testid={selectors.integrations.deploymentProfileStatus}>
                  {profileStatus.ready
                    ? t(S.profileReady)
                    : t(S.missingCapabilities, { capabilities: profileStatus.missingCapabilities.join(", ") })}
                </span>
              }
              hint={profileError ? <span className="text-app-danger">{profileError}</span> : profileSaved ? t(S.profileSaved) : undefined}
            >
              <Button data-testid={selectors.integrations.deploymentProfileSave} type="button" onClick={saveProfile}>
                {t(S.saveProfile)}
              </Button>
            </SettingsList.Row>
          </SettingsList.Group>
        </SettingsList>
      ) : null}
    </section>
  );
}
