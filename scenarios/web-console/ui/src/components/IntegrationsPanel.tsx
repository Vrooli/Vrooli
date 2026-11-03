// DOC: docs/internal/SEAMS.md#capability-registry-seam-api
// DOC: docs/internal/SEAMS.md#connected-scenarios-registry-seam-api-ui
import { useMemo, useState, type ReactNode } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ExternalLink } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { TFunction } from "i18next";
import { useCapabilities } from "../hooks/useCapabilities";
import { strings } from "../consts/strings";
import { runCapabilityAction, type CapabilityActionResponse, type CapabilityState } from "../api/capabilities";
import { fetchScenarioLink } from "../api/notifications";
import {
  getCommercialContext,
  getSubscriptionSummary,
  getConnections,
  isUnauthenticatedResponse,
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
} from "@vrooli/react-component-library/IntegrationsOverview/1.1.1";

interface IntegrationsPanelProps {
  open: boolean;
  /** Shown above the summary on the overview, not on a detail view. */
  header?: ReactNode;
}

const P = strings.integrationsPanel;

function initialsOf(name: string): string {
  const words = name.replace(/[^A-Za-z0-9 ]+/g, " ").trim().split(/\s+/);
  return ((words[0]?.[0] ?? "") + (words[1]?.[0] ?? words[0]?.[1] ?? "")).toUpperCase();
}

function humanize(slug: string): string {
  const words = slug.replace(/[-_.]+/g, " ").trim();
  return words.charAt(0).toUpperCase() + words.slice(1);
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

function dependencyDetails(
  cap: CapabilityState,
  pendingId: string | undefined,
  outcome: ActionOutcome | undefined,
  run: (cap: CapabilityState) => void,
  t: TFunction,
): Partial<IntegrationItem> {
  const unavailable = cap.status === "unavailable";
  const missing = unavailable && cap.dependencyKind === "resource" && cap.actionKind === "operator_command";
  const canRun = unavailable && (cap.dependencyKind === "scenario"
    ? cap.actionKind === "scenario_start" || cap.actionKind === "scenario_restart"
    : cap.actionKind === "operator_command");
  return {
    ...(unavailable && !missing ? { problem: { title: t(P.genericProblemTitle, { name: humanize(cap.name) }), description: cap.message || cap.description } } : {}),
    ...(missing ? {status: "missing" as const} : {}),
    action: canRun ? { label: cap.actionLabel || t(P.startAction), pendingLabel: t(P.actionRunning), pending: pendingId === cap.id, onRun: () => run(cap) } : undefined,
    inlineAction: cap.dependencyKind === "resource",
    detail: cap.dependencyKind === "scenario" && cap.dependencySlug ? <OpenScenarioLink slug={cap.dependencySlug} name={humanize(cap.dependencySlug)} /> : undefined,
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

interface OptionalComponent {
  entry: { id: string; owner: string; version: string; size: number; standalone_url?: string; provides: { id: string }[] };
}
interface ComponentStatus { catalog_error?: string; available: OptionalComponent[]; installed: { signed: OptionalComponent; state: string; reason?: string }[] }

function OptionalDependencies({ open }: { open: boolean }) {
  const queryClient = useQueryClient();
  const bridge = window.desktop?.components;
  const [outcome, setOutcome] = useState<string>();
  const [removing, setRemoving] = useState<string>();
  const catalog = useQuery({ queryKey: ["desktop-components"], queryFn: async () => await bridge!.list() as ComponentStatus, enabled: open && Boolean(bridge), retry: false });
  const refresh = () => { void queryClient.invalidateQueries({ queryKey: ["desktop-components"] }); };
  const install = useMutation({
    mutationFn: (id: string) => bridge!.install(id),
    onSuccess: () => { setOutcome("Ready for the next time you open the app."); refresh(); },
    onError: () => setOutcome("The dependency could not be added or updated. Your current installation is unchanged."),
  });
  const remove = useMutation({
    mutationFn: ({ id, keepData }: { id: string; keepData: boolean }) => bridge!.remove(id, keepData),
    onSuccess: (_, { keepData }) => { setRemoving(undefined); setOutcome(keepData ? "Dependency removed. Its saved data was kept." : "Dependency and its saved data removed."); refresh(); },
    onError: () => setOutcome("The dependency could not be removed. Please try again."),
  });
  if (!bridge || (!catalog.isError && !catalog.data?.available.length && !catalog.data?.installed.length)) return null;
  const entries = [...(catalog.data?.available ?? [])];
  for (const row of catalog.data?.installed ?? []) if (!entries.some(item => item.entry.id === row.signed.entry.id)) entries.push(row.signed);
  return <section aria-label="Optional dependencies" className="mt-4 rounded-xl border border-wc-border p-4">
    <h3 className="m-0 text-base font-semibold">Optional dependencies</h3>
    <p className="mt-1 text-sm text-wc-text-muted">Add the features you need. Added or updated dependencies start when you next open the app.</p>
    {(catalog.isError || catalog.data?.catalog_error) && <p role="alert">{catalog.data?.catalog_error ?? "The dependency catalog is unavailable."}</p>}
    {entries.map((component) => {
      const installed = catalog.data?.installed.find((row) => row.signed.entry.id === component.entry.id);
      const available = catalog.data?.available.some(row => row.entry.id === component.entry.id);
      const update = installed && installed.signed.entry.version !== component.entry.version;
      const busy = install.isPending || remove.isPending;
      return <div key={component.entry.id} data-testid={`optional-component-${component.entry.id}`} className="flex flex-wrap items-center justify-between gap-3 border-t border-wc-border py-3">
        <div><strong>{humanize(component.entry.owner)}</strong><p className="m-0 text-sm text-wc-text-muted">Version {installed?.signed.entry.version ?? component.entry.version} · {(component.entry.size / 1048576).toFixed(1)} MB</p><p className="m-0 text-sm">{component.entry.provides.map((feature) => humanize(feature.id)).join(", ")}</p>{installed?.reason && <p className="m-0 text-sm">{installed.reason}</p>}</div>
        <div className="flex flex-wrap items-center gap-2">
          {installed ? <><span role="status" className="text-sm">{installed.state === "pending_next_start" ? "Added · starts next time" : humanize(installed.state)}</span>
            {available && (update || installed.state === "needs_update") && <Button type="button" size="sm" disabled={busy} onClick={() => install.mutate(component.entry.id)}>{install.isPending ? "Updating…" : `Update to ${component.entry.version}`}</Button>}
            <Button type="button" size="sm" disabled={busy} onClick={() => setRemoving(component.entry.id)}>Remove</Button>
          </> : <Button type="button" size="sm" disabled={busy} onClick={() => install.mutate(component.entry.id)}>{install.isPending ? "Adding…" : "Add to this app"}</Button>}
          {component.entry.standalone_url && <a className="text-sm underline underline-offset-4" href={component.entry.standalone_url} target="_blank" rel="noopener noreferrer">Get the full app</a>}
        </div>
        {removing === component.entry.id && <div role="group" aria-label="Remove dependency" className="w-full rounded-lg border border-wc-border p-3">
          <p className="mt-0 text-sm">Remove {humanize(component.entry.owner)} from this app? Choose what happens to its saved data.</p>
          <div className="flex flex-wrap gap-2">
            <Button size="sm" disabled={busy} onClick={() => remove.mutate({ id: component.entry.id, keepData: true })}>Remove and keep data</Button>
            <Button size="sm" disabled={busy} onClick={() => remove.mutate({ id: component.entry.id, keepData: false })}>Remove and delete data</Button>
            <Button size="sm" disabled={busy} onClick={() => setRemoving(undefined)}>Cancel</Button>
          </div>
        </div>}
      </div>;
    })}
    {outcome && <p role="status" className="text-sm">{outcome}</p>}
  </section>;
}

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
  const subscription = useQuery({
    queryKey: ["subscription-summary"], queryFn: getSubscriptionSummary,
    enabled: open && Boolean(getWebAccessToken()), staleTime: 30_000, retry: false,
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
    const accounts = (connections.data?.connections ?? []).map((connection) => accountItem(connection, t, locale));
    const result: IntegrationGroup[] = [
      {
        id: "accounts",
        label: t(P.groups.accounts),
        items: accounts,
        width: "half",
        emptyText: connections.isError
          ? (isUnauthenticatedResponse(connections.error) ? t(P.accountsEmpty) : t(P.accountsUnavailable))
          : connections.isLoading ? t(P.accountsLoading) : t(P.accountsEmpty),
      },
      { id: "funding", label: t(P.modelSource), items: [{
        id: "vrooli-credits", name: t(P.creditsName),
        subtitle: subscription.isError ? t(P.creditsUnavailable)
          : subscription.isLoading ? t(P.creditsChecking)
          : subscription.data?.configured ? t(P.creditsConfigured)
          : t(P.creditsSignIn),
        status: subscription.isLoading ? "checking" : "unknown",
        selectable: false,
      }] },
    ];
    return result;
    // `run` closes over the mutation, which is stable for this render.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [connections.data, connections.error, connections.isError, connections.isLoading, t, locale, subscription.data, subscription.isError, subscription.isLoading]);

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
      <OptionalDependencies open={open} />
      <IntegrationsOverview
        header={header}
        groups={groups}
        dependencies={caps}
        decorateDependency={(dependency) => {
          const cap = caps.find((item) => item.id === dependency.id);
          return cap ? dependencyDetails(cap, pendingId, outcome?.capabilityId === cap.id ? outcome : undefined, run, t) : {};
        }}
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
