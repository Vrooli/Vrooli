import { useState } from "react";
import { CheckCircle, AlertCircle, Circle } from "lucide-react";
import { useCapabilities } from "../lib/hooks";
import { useCredentials, useDeleteCredential, useTestCredential } from "../lib/hooks-settings";
import type { CapabilityState, CapabilityStatus, Credential } from "../lib/api";
import { StatusBadge, type StatusTone } from "@vrooli/react-component-library/StatusBadge/1.2.2";
import { IntegrationCard } from "@vrooli/react-component-library/IntegrationCard/0";
import { Button } from "@vrooli/react-component-library/Button/2";

interface SettingsTabIntegrationsProps {
  isMobile: boolean;
  repoId?: string | null;
}

function statusIcon(status: CapabilityStatus, size: string) {
  switch (status) {
    case "available":
      return <CheckCircle className={`${size} text-emerald-500 shrink-0`} />;
    case "unavailable":
      return <AlertCircle className={`${size} text-red-500 shrink-0`} />;
    default:
      return <Circle className={`${size} text-slate-600 shrink-0`} />;
  }
}

function borderColor(status: CapabilityStatus): string {
  switch (status) {
    case "available":
      return "border-l-emerald-500";
    case "unavailable":
      return "border-l-red-500";
    default:
      return "border-l-slate-700";
  }
}

function statusTone(status: CapabilityStatus): StatusTone {
  if (status === "available") return "success";
  if (status === "unavailable") return "danger";
  return "neutral";
}

function credentialProvider(url: string, type: Credential["type"]): string {
  if (type === "ssh") return "SSH provider";
  try {
    return new URL(url).hostname || "HTTPS provider";
  } catch {
    return "HTTPS provider";
  }
}

function CredentialConnectionCard({ credential, repoId }: { credential: Credential; repoId?: string | null }) {
  const testMutation = useTestCredential(repoId);
  const deleteMutation = useDeleteCredential(repoId);
  const [message, setMessage] = useState<string>();
  const status = credential.is_configured ? "connected" : "needs_attention";

  return (
    <IntegrationCard
      providerName={credentialProvider(credential.url, credential.type)}
      connectionName={`${credential.remote} ${credential.type.toUpperCase()} credential`}
      accountLabel={credential.username}
      status={status}
      bindings={repoId ? ["current repository"] : ["repository remotes"]}
      freshness={credential.updated_at ? `Updated ${credential.updated_at}` : undefined}
      nextAction={status === "connected" ? "Test or remove this credential from the actions below." : "Configure this credential in the repository credentials tab."}
      actions={(
        <div className="flex flex-wrap items-center gap-2">
          <Button type="button" variant="secondary" size="sm" disabled={testMutation.isPending || deleteMutation.isPending} onClick={async () => {
            setMessage(undefined);
            try {
              const result = await testMutation.mutateAsync({ remote: credential.remote, use_stored: true });
              setMessage(result.authorized ? "Connection verified." : result.error || "Connection test failed.");
            } catch (error) {
              setMessage(error instanceof Error ? error.message : "Connection test failed.");
            }
          }}>{testMutation.isPending ? "Testing…" : "Test connection"}</Button>
          <Button type="button" variant="secondary" size="sm" disabled={testMutation.isPending || deleteMutation.isPending} onClick={async () => {
            if (!window.confirm("Remove this repository credential?")) return;
            setMessage(undefined);
            try {
              await deleteMutation.mutateAsync(credential.id);
              setMessage("Credential removed.");
            } catch (error) {
              setMessage(error instanceof Error ? error.message : "Credential could not be removed.");
            }
          }}>{deleteMutation.isPending ? "Removing…" : "Remove"}</Button>
          {message && <span role="status" className="text-[11px] text-slate-400">{message}</span>}
        </div>
      )}
    />
  );
}

function CapabilityCard({ cap, isMobile }: { cap: CapabilityState; isMobile: boolean }) {
  const textSm = isMobile ? "text-sm" : "text-xs";
  const textXs = isMobile ? "text-xs" : "text-[11px]";
  const py = isMobile ? "py-3" : "py-2";
  const px = isMobile ? "px-4" : "px-3";
  const isUnavailable = cap.status === "unavailable";

  return (
    <div
      className={`rounded-lg border border-slate-800 ${borderColor(cap.status)} border-l-2 bg-slate-900/40 ${px} ${py} space-y-1.5`}
    >
      <div className="flex items-center gap-2">
        {statusIcon(cap.status, "h-4 w-4")}
        <span className={`${textSm} font-medium text-slate-200`}>{cap.name}</span>
        <StatusBadge tone={statusTone(cap.status)}>{cap.status}</StatusBadge>
        <span className={`${textXs} px-1.5 py-0.5 rounded bg-slate-800 text-slate-400`}>
          {cap.dependencyKind}
        </span>
      </div>

      <p className={`${textXs} text-slate-400`}>{cap.description}</p>

      {cap.message && (
        <p className={`${textXs} text-slate-500 font-mono`}>{cap.message}</p>
      )}

      {cap.features.length > 0 && (
        <div className="flex flex-wrap gap-1 mt-1">
          {cap.features.map((feature) => (
            <span
              key={feature}
              className={`${textXs} px-1.5 py-0.5 rounded-full border ${
                isUnavailable
                  ? "border-slate-800 text-slate-600 line-through"
                  : "border-slate-700 text-slate-300"
              }`}
            >
              {feature}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}

export function SettingsTabIntegrations({ isMobile, repoId }: SettingsTabIntegrationsProps) {
  const { data, isLoading, isError, error } = useCapabilities();
  const credentialsQuery = useCredentials(repoId);
  const credentials = credentialsQuery.data?.credentials ?? [];

  const textSm = isMobile ? "text-sm" : "text-xs";
  const textXs = isMobile ? "text-xs" : "text-[11px]";
  const gap = isMobile ? "gap-3" : "gap-2";

  if (isLoading) {
    return (
      <div className="space-y-4">
        <section data-testid="connected-accounts" className="space-y-2">
          <h3 className={`font-semibold text-slate-200 ${textSm}`}>Connected accounts</h3>
          {credentialsQuery.isLoading ? <p className={`${textXs} text-slate-500`}>Checking repository credentials...</p> : credentialsQuery.isError ? <p className={`${textXs} text-amber-300`}>Repository credentials are temporarily unavailable.</p> : credentials.length > 0 ? credentials.map((credential) => <CredentialConnectionCard key={credential.id} credential={credential} repoId={repoId} />) : <p className={`${textXs} text-slate-500`}>No repository credentials are connected yet.</p>}
        </section>
        <h3 className={`font-semibold text-slate-200 ${textSm}`}>Runtime dependencies</h3>
        <p className={`${textXs} text-slate-500`}>Checking integrations...</p>
      </div>
    );
  }

  if (isError) {
    return (
      <div className="space-y-4">
        <section data-testid="connected-accounts" className="space-y-2">
          <h3 className={`font-semibold text-slate-200 ${textSm}`}>Connected accounts</h3>
          {credentialsQuery.isLoading ? <p className={`${textXs} text-slate-500`}>Checking repository credentials...</p> : credentialsQuery.isError ? <p className={`${textXs} text-amber-300`}>Repository credentials are temporarily unavailable.</p> : credentials.length > 0 ? credentials.map((credential) => <CredentialConnectionCard key={credential.id} credential={credential} repoId={repoId} />) : <p className={`${textXs} text-slate-500`}>No repository credentials are connected yet.</p>}
        </section>
        <h3 className={`font-semibold text-slate-200 ${textSm}`}>Runtime dependencies</h3>
        <p className={`${textXs} text-red-400`}>Failed to load integrations: {error.message}</p>
      </div>
    );
  }

  const capabilities = data?.capabilities ?? [];
  const activeCount = capabilities.filter((c) => c.status === "available").length;

  return (
    <div className="space-y-4">
      <section data-testid="connected-accounts" className="space-y-2">
        <h3 className={`font-semibold text-slate-200 ${textSm}`}>Connected accounts</h3>
        {credentialsQuery.isLoading ? <p className={`${textXs} text-slate-500`}>Checking repository credentials...</p> : credentialsQuery.isError ? <p className={`${textXs} text-amber-300`}>Repository credentials are temporarily unavailable.</p> : credentials.length > 0 ? credentials.map((credential) => <CredentialConnectionCard key={credential.id} credential={credential} repoId={repoId} />) : <p className={`${textXs} text-slate-500`}>No repository credentials are connected yet. HTTPS credentials and SSH keys remain managed in their dedicated settings.</p>}
      </section>
      <div className="flex items-center justify-between">
        <h3 className={`font-semibold text-slate-200 ${textSm}`}>Runtime dependencies</h3>
        <span className={`${textXs} text-slate-500`}>
          {activeCount}/{capabilities.length} active
        </span>
      </div>

      {capabilities.length === 0 ? (
        <p className={`${textXs} text-slate-500`}>No runtime dependencies configured.</p>
      ) : (
        <div className={`flex flex-col ${gap}`}>
          {capabilities.map((cap) => (
            <CapabilityCard key={cap.id} cap={cap} isMobile={isMobile} />
          ))}
        </div>
      )}
    </div>
  );
}
