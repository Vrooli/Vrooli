// DOC: docs/internal/SEAMS.md#capability-registry-seam-api
// DOC: docs/internal/SEAMS.md#connected-scenarios-registry-seam-api-ui
import { useMemo, useState, type ReactNode } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Bell, Boxes, Cloud, Cpu, ExternalLink, Mic, MonitorSmartphone, Plug, TerminalSquare } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { TFunction } from "i18next";
import { useCapabilities } from "../hooks/useCapabilities";
import { strings } from "../consts/strings";
import { runCapabilityAction, type CapabilityActionResponse, type CapabilityState } from "../api/capabilities";
import { fetchScenarioLink } from "../api/notifications";
import {
  getCommercialContext,
  getConnections,
  isCommercialContentVisible,
  removeOpenRouterKey,
  safeCommercialDestination,
  testOpenRouterKey,
  type ManagedConnection,
} from "../api/monetization";
import { getWebAccessToken } from "../lib/auth";
import { Button } from "@vrooli/react-component-library/Button/2";
import {
  IntegrationsOverview,
  type IntegrationGroup,
  type IntegrationItem,
  type IntegrationStatus,
  type IntegrationsOverviewStrings,
} from "@vrooli/react-component-library/IntegrationsOverview/1";

interface IntegrationsPanelProps {
  open: boolean;
  /** Shown above the summary on the overview, not on a detail view. */
  header?: ReactNode;
}

const P = strings.integrationsPanel;

/** Capabilities whose copy Web Console words for people; keys into integrationsPanel.items. */
const ITEM_COPY: Record<string, keyof typeof P.items> = {
  "audio-tools": "audioTools",
  "vrooli-bridge": "vrooliBridge",
  "notification-hub": "notificationHub",
  "integration-hub": "integrationHub",
  ollama: "ollama",
  openrouter: "openrouter",
  "session-backend-standard": "sessionStandard",
  "session-backend-persistent": "sessionPersistent",
  "terminal-local": "sessionStandard",
  "terminal-persistent": "sessionPersistent",
  "terminal-remote": "vrooliBridge",
};

const ICONS: Record<string, ReactNode> = {
  "audio-tools": <Mic />,
  "vrooli-bridge": <MonitorSmartphone />,
  "terminal-remote": <MonitorSmartphone />,
  "notification-hub": <Bell />,
  "integration-hub": <Plug />,
  ollama: <Cpu />,
  openrouter: <Cloud />,
  "session-backend-standard": <TerminalSquare />,
  "session-backend-persistent": <TerminalSquare />,
  "terminal-local": <TerminalSquare />,
  "terminal-persistent": <TerminalSquare />,
};

const AGENT_INITIALS: Record<string, string> = {
  claude: "CC",
  "claude-code": "CC",
  codex: "CX",
  opencode: "OC",
  grok: "GK",
  agy: "AG",
  antigravity: "AG",
};

const MODEL_IDS = new Set(["ollama", "openrouter"]);

type GroupId = "agents" | "models" | "apps" | "sessions" | "other";

function groupOf(cap: CapabilityState): GroupId {
  if (cap.id in AGENT_INITIALS || (cap.dependencyKind !== "scenario" && cap.actionKind === "operator_command")) return "agents";
  if (MODEL_IDS.has(cap.id)) return "models";
  if (cap.id.startsWith("session-backend-") || cap.id === "terminal-local" || cap.id === "terminal-persistent") return "sessions";
  if (cap.dependencyKind === "scenario") return "apps";
  return "other";
}

function initialsOf(name: string): string {
  const words = name.replace(/[^A-Za-z0-9 ]+/g, " ").trim().split(/\s+/);
  return ((words[0]?.[0] ?? "") + (words[1]?.[0] ?? words[0]?.[1] ?? "")).toUpperCase();
}

function camel(slug: string): string {
  return slug.replace(/[-_]+(\w)/g, (_, c: string) => c.toUpperCase());
}

function humanize(slug: string): string {
  const words = slug.replace(/[-_.]+/g, " ").trim();
  return words.charAt(0).toUpperCase() + words.slice(1);
}

function versionOf(message?: string): string | undefined {
  return message?.match(/\d+\.\d+(?:\.\d+)?/)?.[0];
}

function runnable(cap: CapabilityState): boolean {
  if (cap.dependencyKind === "scenario") return cap.actionKind === "scenario_start" || cap.actionKind === "scenario_restart";
  return cap.actionKind === "operator_command";
}

function capabilityStatus(cap: CapabilityState, group: GroupId): IntegrationStatus {
  if (cap.status === "unknown") return "checking";
  if (cap.status === "unavailable") return group === "agents" ? "missing" : "off";
  const features = Object.values(cap.featureStatus ?? {});
  return features.includes("unavailable") ? "limited" : "ready";
}

function formatTime(value: string | undefined, locale: string): string | undefined {
  if (!value) return undefined;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return undefined;
  return new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "short" }).format(date);
}

const ACCOUNT_STATUS: Record<ManagedConnection["status"], { status: IntegrationStatus; label: keyof typeof P.accountStatus }> = {
  connected: { status: "ready", label: "connected" },
  checking: { status: "checking", label: "checking" },
  needs_attention: { status: "limited", label: "needsAttention" },
  expired: { status: "limited", label: "expired" },
  insufficient_scope: { status: "limited", label: "insufficientScope" },
  offline: { status: "limited", label: "offline" },
  disconnected: { status: "off", label: "disconnected" },
  revoked: { status: "off", label: "revoked" },
  provider_unavailable: { status: "off", label: "providerUnavailable" },
  unknown: { status: "unknown", label: "unknown" },
};

function supportsCredentialActions(actions?: string[]): boolean {
  return Boolean(actions?.some((action) => action === "test" || action === "delete"));
}

function ConnectionActions({ supportedActions }: { supportedActions: string[] }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [message, setMessage] = useState<string>();
  const testMutation = useMutation({
    mutationFn: testOpenRouterKey,
    onSuccess: (result) => {
      setMessage(result.valid ? t(P.connectionVerified) : t(P.connectionTestFailed));
      void queryClient.invalidateQueries({ queryKey: ["integration-connections"] });
    },
    onError: () => { setMessage(t(P.connectionTestUnavailable)); },
  });
  const removeMutation = useMutation({
    mutationFn: removeOpenRouterKey,
    onSuccess: () => {
      setMessage(t(P.connectionRemoved));
      void queryClient.invalidateQueries({ queryKey: ["integration-connections"] });
    },
    onError: () => { setMessage(t(P.connectionRemoveFailed)); },
  });
  const busy = testMutation.isPending || removeMutation.isPending;

  return (
    <div className="flex flex-wrap items-center gap-2">
      {supportedActions.includes("test") && (
        <Button type="button" size="sm" variant="secondary" disabled={busy} onClick={() => { setMessage(undefined); testMutation.mutate(); }}>
          {testMutation.isPending ? t(P.testingConnection) : t(P.testConnection)}
        </Button>
      )}
      {supportedActions.includes("delete") && (
        <Button
          type="button"
          size="sm"
          variant="ghost"
          disabled={busy}
          onClick={() => {
            if (window.confirm(t(P.connectionRemoveConfirm))) {
              setMessage(undefined);
              removeMutation.mutate();
            }
          }}
        >
          {removeMutation.isPending ? t(P.removingConnection) : t(P.removeConnection)}
        </Button>
      )}
      {message && <span role="status" className="text-sm text-wc-text-muted">{message}</span>}
    </div>
  );
}

/** Resolved only when its detail view is open, so the overview costs no link lookups. */
function OpenScenarioLink({ slug, name }: { slug: string; name: string }) {
  const { t } = useTranslation();
  const link = useQuery({
    queryKey: ["scenario-link", slug],
    queryFn: () => fetchScenarioLink(slug),
    staleTime: 5 * 60_000,
    retry: false,
  });
  if (!link.data) return null;
  return (
    <a
      href={link.data}
      target="_blank"
      rel="noopener noreferrer"
      data-testid={`integration-open-${slug}`}
      className="inline-flex min-h-11 items-center justify-center gap-1.5 self-center rounded-md px-3 text-sm font-medium text-wc-accent hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-wc-accent"
    >
      {t(P.openApp, { name })}
      <ExternalLink className="h-4 w-4" aria-hidden="true" />
    </a>
  );
}

interface ActionOutcome {
  capabilityId: string;
  success: boolean;
  text: string;
}

function capabilityItem(
  cap: CapabilityState,
  group: GroupId,
  t: TFunction,
  locale: string,
  pendingId: string | undefined,
  outcome: ActionOutcome | undefined,
  run: (cap: CapabilityState) => void,
): IntegrationItem {
  const copyKey = ITEM_COPY[cap.id];
  const copy = copyKey ? P.items[copyKey] : undefined;
  const name = copy ? t(copy.name) : group === "agents" ? cap.name : humanize(cap.name);
  const status = capabilityStatus(cap, group);
  const version = group === "agents" ? versionOf(cap.message) : undefined;
  const subtitle = copy
    ? t(copy.subtitle)
    : version
      ? t(P.agentVersion, { version })
      : group === "agents" ? undefined : cap.description || undefined;
  const provider = copy && "provider" in copy ? t(copy.provider) : undefined;

  let problem: IntegrationItem["problem"];
  if (status === "off") {
    problem = copy
      ? { title: t(copy.problemTitle), description: t(copy.problemDescription) }
      : { title: t(P.genericProblemTitle, { name }), description: t(P.genericProblemDescription) };
  }

  const features = cap.features.map((slug) => {
    const key = camel(slug) as keyof typeof P.features;
    return {
      id: slug,
      label: key in P.features ? t(P.features[key]) : humanize(slug),
      available: cap.featureStatus?.[slug] !== "unavailable",
    };
  });
  if (status === "limited") {
    const missing = features.filter((feature) => !feature.available).map((feature) => feature.label);
    problem = {
      title: t(P.limitedTitle, { name }),
      description: t(P.limitedDescription, { features: missing.join(", ") }),
    };
  }

  // Starting or installing fixes only something that is off or absent; a
  // running service with some features down needs its provider, not a start.
  const canRun = (status === "off" || status === "missing") && runnable(cap);
  const action = canRun
    ? {
        label: group === "agents" ? t(P.agentInstall) : cap.actionLabel || t(P.startAction),
        pendingLabel: t(P.actionRunning),
        pending: pendingId === cap.id,
        onRun: () => { run(cap); },
      }
    : undefined;

  const facts = [
    { label: t(P.facts.id), value: cap.dependencySlug || cap.id, monospace: true },
    ...(cap.message ? [{ label: t(P.facts.health), value: cap.message }] : []),
    ...(cap.reasonCode ? [{ label: t(P.facts.reason), value: cap.reasonCode, monospace: true }] : []),
    ...(formatTime(cap.checkedAt, locale) ? [{ label: t(P.facts.checked), value: formatTime(cap.checkedAt, locale) }] : []),
    ...(status !== "ready" && cap.operatorCommand ? [{ label: t(P.facts.command), value: cap.operatorCommand, monospace: true }] : []),
  ];

  return {
    id: cap.id,
    name,
    subtitle: status === "missing" && action ? t(P.overview.statusMissing) : status === "missing" ? undefined : subtitle,
    provider,
    icon: ICONS[cap.id] ?? (group === "agents" ? undefined : <Boxes />),
    initials: group === "agents" ? (AGENT_INITIALS[cap.id] ?? initialsOf(cap.name)) : undefined,
    status,
    problem,
    action,
    inlineAction: group === "agents",
    features: features.length > 1 || copy ? features : [],
    facts,
    detail: cap.dependencyKind === "scenario" && cap.dependencySlug ? <OpenScenarioLink slug={cap.dependencySlug} name={provider ?? name} /> : undefined,
    message: outcome ? { text: outcome.text, tone: outcome.success ? "success" : "danger" } : undefined,
  };
}

function accountItem(connection: ManagedConnection, t: TFunction, locale: string): IntegrationItem {
  const mapped = ACCOUNT_STATUS[connection.status];
  const name = connection.connection_name || humanize(connection.provider);
  const verified = formatTime(connection.last_verified_at, locale);
  const subtitle = [connection.account_label, verified ? t(P.accountVerified, { time: verified }) : undefined].filter(Boolean).join(" · ");
  const hasProblem = mapped.status === "limited" || mapped.status === "off";
  return {
    id: `account:${connection.id}`,
    name,
    subtitle: subtitle || undefined,
    provider: humanize(connection.provider),
    initials: initialsOf(connection.provider),
    status: mapped.status,
    statusLabel: t(P.accountStatus[mapped.label]),
    problem: hasProblem ? { title: t(P.accountProblem, { name }), description: connection.next_action } : undefined,
    facts: [
      ...(connection.scopes?.length ? [{ label: t(P.facts.scopes), value: connection.scopes.join(", ") }] : []),
      ...(connection.bindings?.length ? [{ label: t(P.facts.usedBy), value: connection.bindings.join(", ") }] : []),
      ...(connection.freshness ? [{ label: t(P.facts.freshness), value: connection.freshness }] : []),
    ],
    detail:
      connection.provider.toLowerCase() === "openrouter" && supportsCredentialActions(connection.supported_actions) ? (
        <ConnectionActions supportedActions={connection.supported_actions ?? []} />
      ) : undefined,
    selectable: true,
  };
}

const OVERVIEW_STRING_KEYS = [
  "summaryAllReady", "summarySomeReady", "summaryChecking", "summaryError", "checkedAt", "refresh", "refreshing",
  "back", "providedBy", "statusReady", "statusLimited", "statusOff", "statusMissing", "statusChecking",
  "statusUnknown", "featuresTitle", "featureUnavailable", "technicalDetails", "emptyGroup",
] as const satisfies readonly (keyof IntegrationsOverviewStrings & keyof typeof P.overview)[];

export default function IntegrationsPanel({ open, header }: IntegrationsPanelProps) {
  const { t, i18n } = useTranslation();
  const locale = i18n.language;
  const capabilities = useCapabilities(open);
  const connections = useQuery({
    queryKey: ["integration-connections"],
    queryFn: getConnections,
    enabled: open,
    staleTime: 30_000,
    retry: false,
  });
  const caps = useMemo(() => capabilities.data?.capabilities ?? [], [capabilities.data]);
  const commercialCapabilityId = caps.find((cap) => cap.status === "unavailable")?.id ?? "";
  const commercial = useQuery({
    queryKey: ["commercial-context", "integrations", commercialCapabilityId],
    queryFn: () => getCommercialContext("integrations", commercialCapabilityId),
    enabled: open && Boolean(getWebAccessToken()),
    staleTime: 60_000,
    retry: false,
  });
  const queryClient = useQueryClient();
  const [pendingId, setPendingId] = useState<string>();
  const [outcome, setOutcome] = useState<ActionOutcome>();
  const [dismissed, setDismissed] = useState<Set<string>>(() => new Set());

  const action = useMutation({
    mutationFn: (cap: CapabilityState) => {
      if (!cap.actionKind) throw new Error("action kind missing");
      return runCapabilityAction(cap.id, cap.actionKind);
    },
    onMutate: (cap) => {
      setPendingId(cap.id);
      setOutcome(undefined);
    },
    onSuccess: (result: CapabilityActionResponse) => {
      setOutcome({
        capabilityId: result.capabilityId,
        success: result.success,
        text: result.success ? (result.message || t(P.actionSucceeded)) : t(P.actionFailed, { message: result.message || result.status }),
      });
      queryClient.setQueryData(["capabilities"], { capabilities: result.capabilities, timestamp: result.timestamp });
    },
    onError: (err, cap) => {
      setOutcome({ capabilityId: cap.id, success: false, text: t(P.actionFailed, { message: err instanceof Error ? err.message : String(err) }) });
    },
    onSettled: () => {
      setPendingId(undefined);
      void queryClient.invalidateQueries({ queryKey: ["capabilities"] });
    },
  });

  const run = (cap: CapabilityState) => {
    if (!action.isPending) action.mutate(cap);
  };

  const groups: IntegrationGroup[] = useMemo(() => {
    const byGroup: Record<GroupId, IntegrationItem[]> = { agents: [], models: [], apps: [], sessions: [], other: [] };
    for (const cap of caps) {
      const group = groupOf(cap);
      byGroup[group].push(
        capabilityItem(cap, group, t, locale, pendingId, outcome?.capabilityId === cap.id ? outcome : undefined, run),
      );
    }
    const accounts = (connections.data?.connections ?? []).map((connection) => accountItem(connection, t, locale));
    const result: IntegrationGroup[] = [
      {
        id: "accounts",
        label: t(P.groups.accounts),
        items: accounts,
        width: "half",
        emptyText: connections.isError ? t(P.accountsUnavailable) : connections.isLoading ? t(P.accountsLoading) : t(P.accountsEmpty),
      },
      { id: "agents", label: t(P.groups.agents), items: byGroup.agents, width: "half" },
      { id: "apps", label: t(P.groups.apps), items: byGroup.apps },
      { id: "models", label: t(P.groups.models), items: byGroup.models, width: "half" },
      { id: "sessions", label: t(P.groups.sessions), items: byGroup.sessions, width: "half" },
      { id: "other", label: t(P.groups.other), items: byGroup.other },
    ];
    return result.filter((group) => group.id === "accounts" || group.items.length > 0);
    // `run` closes over the mutation, which is stable for this render.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [caps, connections.data, connections.isError, connections.isLoading, t, locale, pendingId, outcome]);

  const overviewStrings = useMemo(
    () => Object.fromEntries(OVERVIEW_STRING_KEYS.map((key) => [key, t(P.overview[key])])) as Partial<IntegrationsOverviewStrings>,
    [t],
  );

  const recommendations = (commercial.data?.content ?? []).filter(
    (item) => isCommercialContentVisible(item, "integrations") && !dismissed.has(item.content_id),
  );
  const notices = recommendations.length > 0 ? (
    <section data-testid="commercial-context" className="grid gap-3 rounded-lg border border-wc-default bg-wc-surface-raised p-4">
      {recommendations.map((item) => {
        const destination = safeCommercialDestination(item.cta_destination);
        return (
          <div key={item.content_id} className="grid gap-1">
            <div className="flex items-start justify-between gap-3">
              <h3 className="m-0 text-sm font-semibold text-wc-text-primary">{item.title}</h3>
              {item.dismissible && (
                <Button type="button" size="sm" variant="ghost" onClick={() => { setDismissed((current) => new Set(current).add(item.content_id)); }}>
                  {t(P.dismissRecommendation)}
                </Button>
              )}
            </div>
            <p className="m-0 text-sm text-wc-text-muted">{item.description}</p>
            {destination && <a className="text-sm font-medium text-wc-accent hover:underline" href={destination}>{item.cta_label}</a>}
          </div>
        );
      })}
      {commercial.data?.stale && <p className="m-0 text-xs text-wc-text-faint">{t(P.staleCommercialContext)}</p>}
    </section>
  ) : null;

  return (
    <div data-testid="integrations-panel">
      <IntegrationsOverview
        header={header}
        groups={groups}
        notices={notices}
        checkedAt={capabilities.data?.timestamp}
        loading={capabilities.isLoading}
        error={capabilities.isError ? t(P.loadFailed, { message: capabilities.error.message }) : null}
        refreshing={capabilities.isFetching || connections.isFetching}
        onRefresh={() => {
          void queryClient.invalidateQueries({ queryKey: ["capabilities"] });
          void queryClient.invalidateQueries({ queryKey: ["integration-connections"] });
        }}
        strings={overviewStrings}
        locale={locale}
      />
    </div>
  );
}
