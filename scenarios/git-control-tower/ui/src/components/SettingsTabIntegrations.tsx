import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import {
  IntegrationsOverview,
  type IntegrationGroup,
  type IntegrationItem,
} from "@vrooli/react-component-library/IntegrationsOverview/1.1.1";
import { SettingsList } from "@vrooli/react-component-library/SettingsList/1";
import { Button } from "@vrooli/react-component-library/Button/2";
import { useCapabilities } from "../lib/hooks";
import { useCredentials, useDeleteCredential, useTestCredential } from "../lib/hooks-settings";
import { queryKeys } from "../lib/hooks-query-keys";
import type { Credential } from "../lib/api";

interface SettingsTabIntegrationsProps {
  isMobile: boolean;
  repoId?: string | null;
}

function dependencyDetails(dependency: {
  name: string; description: string; status: string; message?: string;
  operatorCommand?: string; uses?: {name: string}[];
}): Partial<IntegrationItem> {
  return {
    ...(dependency.status === "unavailable" ? { problem: {
      title: `${dependency.uses?.[0]?.name ?? dependency.name} is off`,
      description: dependency.message || dependency.description,
    }} : {}),
    detail: dependency.operatorCommand
      ? <code className="block whitespace-pre-wrap break-words text-sm">{dependency.operatorCommand}</code>
      : undefined,
  };
}

function initialsOf(name: string): string {
  const words = name.split(/[\s.-]+/).filter(Boolean);
  const [first = "", second = ""] = words;
  return (second ? first.charAt(0) + second.charAt(0) : name.slice(0, 2)).toUpperCase();
}

function formatTime(value: string | undefined): string | undefined {
  if (!value) return undefined;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return undefined;
  return date.toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
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
      dependencies={caps}
      decorateDependency={dependencyDetails}
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
