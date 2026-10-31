import { useState, type ReactNode } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Bot, Camera, ClipboardCheck, FlaskConical, ShieldCheck, Sparkles } from "lucide-react";
import {
  IntegrationsOverview,
  type IntegrationGroup,
  type IntegrationItem,
  type IntegrationStatus,
} from "@vrooli/react-component-library/IntegrationsOverview/1";
import { SettingsList } from "@vrooli/react-component-library/SettingsList/1";
import { Button } from "@vrooli/react-component-library/Button/2";
import { useCapabilities } from "../lib/hooks";
import { useCredentials, useDeleteCredential, useTestCredential } from "../lib/hooks-settings";
import { queryKeys } from "../lib/hooks-query-keys";
import type { CapabilityState, Credential } from "../lib/api";

interface SettingsTabIntegrationsProps {
  isMobile: boolean;
  repoId?: string | null;
}

/** Plain-language copy for the apps Git Control Tower builds on; anything else falls back to its slug. */
const APP_COPY: Record<string, { name: string; subtitle: string; provider: string; icon: ReactNode }> = {
  "agent-manager": {
    name: "Agents",
    subtitle: "Run agents and approve their changes",
    provider: "Agent Manager",
    icon: <Bot aria-hidden="true" />,
  },
  "browser-automation-studio": {
    name: "Visual capture",
    subtitle: "Screenshots of app screens over time",
    provider: "Browser Automation Studio",
    icon: <Camera aria-hidden="true" />,
  },
  "scenario-auditor": {
    name: "Standards checks",
    subtitle: "Rule findings in reviews",
    provider: "Scenario Auditor",
    icon: <ShieldCheck aria-hidden="true" />,
  },
  "test-genie": {
    name: "Tests",
    subtitle: "Run tests and see their results",
    provider: "Test Genie",
    icon: <FlaskConical aria-hidden="true" />,
  },
  "tidiness-manager": {
    name: "Code quality",
    subtitle: "Lint, type and tidiness scores",
    provider: "Tidiness Manager",
    icon: <Sparkles aria-hidden="true" />,
  },
  "workspace-sandbox": {
    name: "Approved changes",
    subtitle: "Track approved changes and preview commits",
    provider: "Workspace Sandbox",
    icon: <ClipboardCheck aria-hidden="true" />,
  },
};

function humanize(slug: string): string {
  const words = slug.replace(/[-_]+/g, " ").trim();
  return words.charAt(0).toUpperCase() + words.slice(1);
}

function initialsOf(name: string): string {
  const words = name.split(/[\s.-]+/).filter(Boolean);
  const [first = "", second = ""] = words;
  return (second ? first.charAt(0) + second.charAt(0) : name.slice(0, 2)).toUpperCase();
}

function capabilityStatus(cap: CapabilityState): IntegrationStatus {
  if (cap.status === "available") return "ready";
  if (cap.status === "unavailable") return "off";
  return "checking";
}

function formatTime(value: string | undefined): string | undefined {
  if (!value) return undefined;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return undefined;
  return date.toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
}

function capabilityItem(cap: CapabilityState): IntegrationItem {
  const slug = cap.dependencySlug || cap.id;
  const copy = APP_COPY[slug];
  const name = copy?.name ?? humanize(cap.name || slug);
  const provider = copy?.provider ?? humanize(slug);
  const status = capabilityStatus(cap);
  const checked = formatTime(cap.checkedAt);
  return {
    id: cap.id,
    name,
    subtitle: copy?.subtitle ?? cap.description,
    provider,
    icon: copy?.icon,
    initials: copy ? undefined : initialsOf(name),
    status,
    problem: status === "off"
      ? {
          title: `${name} is off`,
          description: cap.features.length
            ? `${cap.features.join(", ")} won't work until ${provider} is running.`
            : `This won't work until ${provider} is running.`,
        }
      : undefined,
    features: cap.features.map((feature) => ({ id: feature, label: feature, available: status !== "off" })),
    facts: [
      { label: "ID", value: slug, monospace: true },
      ...(cap.message ? [{ label: "Health", value: cap.message }] : []),
      ...(checked ? [{ label: "Last checked", value: checked }] : []),
      ...(status === "off" && cap.dependencyKind === "scenario"
        ? [{ label: "Start it with", value: `vrooli scenario start ${slug}`, monospace: true }]
        : []),
    ],
  };
}

function credentialHost(credential: Credential): string {
  if (credential.type === "ssh") {
    const match = /@([^:/]+)[:/]/.exec(credential.url);
    return match?.[1] ?? "SSH";
  }
  try {
    return new URL(credential.url).hostname || "HTTPS";
  } catch {
    return "HTTPS";
  }
}

function CredentialActions({ credential, repoId, onMessage }: {
  credential: Credential;
  repoId?: string | null;
  onMessage: (message: IntegrationItem["message"]) => void;
}) {
  const testMutation = useTestCredential(repoId);
  const deleteMutation = useDeleteCredential(repoId);
  const busy = testMutation.isPending || deleteMutation.isPending;

  return (
    <div className="flex flex-wrap items-center gap-2">
      <Button type="button" variant="secondary" size="sm" disabled={busy} onClick={async () => {
        onMessage(undefined);
        try {
          const result = await testMutation.mutateAsync({ remote: credential.remote, use_stored: true });
          onMessage(result.authorized
            ? { text: "Connection verified.", tone: "success" }
            : { text: result.error || "Connection test failed.", tone: "danger" });
        } catch (error) {
          onMessage({ text: error instanceof Error ? error.message : "Connection test failed.", tone: "danger" });
        }
      }}>{testMutation.isPending ? "Testing…" : "Test connection"}</Button>
      <Button type="button" variant="secondary" size="sm" disabled={busy} onClick={async () => {
        if (!window.confirm("Remove this repository credential?")) return;
        onMessage(undefined);
        try {
          await deleteMutation.mutateAsync(credential.id);
          onMessage({ text: "Credential removed.", tone: "success" });
        } catch (error) {
          onMessage({ text: error instanceof Error ? error.message : "Credential could not be removed.", tone: "danger" });
        }
      }}>{deleteMutation.isPending ? "Removing…" : "Remove"}</Button>
    </div>
  );
}

export function SettingsTabIntegrations({ repoId }: SettingsTabIntegrationsProps) {
  const queryClient = useQueryClient();
  const capabilities = useCapabilities();
  const credentialsQuery = useCredentials(repoId);
  const [messages, setMessages] = useState<Record<string, IntegrationItem["message"]>>({});

  // The API lists origin even when nothing is stored for it; Git then uses the computer's own login.
  const credentials = credentialsQuery.data?.credentials ?? [];
  const unmanaged = credentials.filter((credential) => !credential.is_configured);
  const accountItems: IntegrationItem[] = credentials.filter((credential) => credential.is_configured).map((credential) => {
    const host = credentialHost(credential);
    const kind = credential.type.toUpperCase();
    return {
      id: `credential-${credential.id}`,
      name: host,
      subtitle: [credential.remote, kind, credential.username].filter(Boolean).join(" · "),
      initials: initialsOf(host),
      status: "ready",
      statusLabel: "Connected",
      facts: [
        { label: "Remote", value: credential.remote, monospace: true },
        { label: "URL", value: credential.url, monospace: true },
        { label: "Type", value: kind },
        ...(formatTime(credential.updated_at) ? [{ label: "Updated", value: formatTime(credential.updated_at) }] : []),
      ],
      message: messages[credential.id],
      detail: (
        <CredentialActions
          credential={credential}
          repoId={repoId}
          onMessage={(message) => setMessages((current) => ({ ...current, [credential.id]: message }))}
        />
      ),
    };
  });
  const unmanagedNote = unmanaged.length
    ? `${unmanaged.map((credential) => credential.remote).join(", ")} ${unmanaged.length === 1 ? "uses" : "use"} this computer's own Git login. Save a token or SSH key in the Credentials tab to manage it here.`
    : undefined;

  const caps = capabilities.data?.capabilities ?? [];
  const apps = caps.filter((cap) => cap.dependencyKind === "scenario").map(capabilityItem);
  const services = caps.filter((cap) => cap.dependencyKind !== "scenario").map(capabilityItem);

  const groups: IntegrationGroup[] = [
    {
      id: "accounts",
      label: "Repository accounts",
      items: accountItems,
      footer: accountItems.length ? unmanagedNote : undefined,
      emptyText: credentialsQuery.isLoading
        ? "Checking repository credentials…"
        : credentialsQuery.isError
          ? "Repository credentials can't be loaded right now."
          : unmanagedNote ?? "No repository credentials yet. Add HTTPS tokens and SSH keys in the Credentials tab.",
    },
    { id: "apps", label: "Powered by other apps", items: apps },
    ...(services.length ? [{ id: "services", label: "Local services", items: services }] : []),
  ];

  return (
    <IntegrationsOverview
      header={
        <SettingsList.Intro
          title={<span className="sr-only">Integrations</span>}
          description="What Git Control Tower uses, and whether each part is working."
        />
      }
      groups={groups}
      checkedAt={capabilities.data?.timestamp}
      loading={capabilities.isLoading}
      error={capabilities.isError ? `Integrations can't be checked right now: ${capabilities.error.message}` : null}
      refreshing={capabilities.isFetching || credentialsQuery.isFetching}
      onRefresh={() => {
        void queryClient.invalidateQueries({ queryKey: queryKeys.capabilities });
        void queryClient.invalidateQueries({ queryKey: queryKeys.credentials(repoId) });
      }}
    />
  );
}
