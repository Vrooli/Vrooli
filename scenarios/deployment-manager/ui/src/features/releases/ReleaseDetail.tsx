import { useQuery } from "@tanstack/react-query";
import { ArrowLeft, Copy, Loader2, ShieldCheck } from "lucide-react";
import { Link, useParams, useSearchParams } from "react-router-dom";
import { Badge } from "../../components/ui/badge";
import { Button } from "../../components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../components/ui/card";
import { getRelease, getReleaseDossier, getReleaseOperation, type Release, type ReleaseHealth, type ReleaseOperation } from "../../lib/api";
import { getErrorMessage } from "../../lib/utils";

function CopyableValue({ label, value }: { label: string; value?: string }) {
  if (!value) return null;
  return (
    <div className="flex items-center gap-2 text-sm">
      <span className="text-slate-400">{label}:</span>
      <code className="break-all text-cyan-200">{value}</code>
      <Button
        aria-label={`Copy ${label}`}
        className="h-7 px-2"
        variant="ghost"
        onClick={() => void navigator.clipboard?.writeText(value)}
      >
        <Copy className="h-3.5 w-3.5" />
      </Button>
    </div>
  );
}

function IdentityCard({ release }: { release: Release }) {
  const candidate = release.candidate;
  const destination = release.destination_revision;
  const review = release.review_binding;
  return (
    <Card>
      <CardHeader>
        <CardTitle>Release identity</CardTitle>
        <CardDescription>Every execution and receipt below must resolve to these exact durable identities.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-2">
        <CopyableValue label="Release ID" value={release.id} />
        <CopyableValue label="Owner deployment" value={release.deployment_id} />
        <CopyableValue label="Candidate ID" value={release.candidate_id ?? candidate?.candidate_id} />
        <CopyableValue label="Destination revision" value={release.destination_revision_id ?? destination?.destination_revision_id} />
        <CopyableValue label="Review binding" value={review?.review_id ?? release.readiness_review_key} />
        <CopyableValue label="Artifact manifest" value={release.artifact_digest} />
        <div className="grid gap-2 pt-2 text-sm text-slate-300 md:grid-cols-3">
          <span>Source: <code>{candidate?.source_revision ?? release.git_commit_hash}</code></span>
          <span>Channel: <code>{destination?.channel ?? release.channel}</code></span>
          <span>Authorization epoch: <code>{review?.authorization_epoch ?? release.authorization_epoch ?? "unbound"}</code></span>
        </div>
      </CardContent>
    </Card>
  );
}

function CandidateCard({ release }: { release: Release }) {
  const candidate = release.candidate;
  if (!candidate) {
    return <Card><CardHeader><CardTitle>Candidate evidence</CardTitle></CardHeader><CardContent className="text-sm text-amber-200">Candidate identity is unavailable. The release cannot be treated as fully reviewable.</CardContent></Card>;
  }
  return (
    <Card>
      <CardHeader><CardTitle>Candidate evidence</CardTitle><CardDescription>Final signed artifacts and build inputs bound to this release.</CardDescription></CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-2 text-sm text-slate-300 md:grid-cols-2">
          <span>Profile revision: <code>{candidate.profile_revision}</code></span>
          <span>Dependency lock: <code>{candidate.dependency_lock_digest}</code></span>
          <span>Policy digest: <code>{candidate.policy_digest}</code></span>
          <span>Artifacts: {candidate.artifacts.length}</span>
        </div>
        <div className="rounded border border-cyan-500/20 bg-cyan-500/5 p-3 text-sm">
          <p className="font-medium text-cyan-100">Operational ownership</p>
          <p className="mt-1 text-slate-400">These authorities are part of the reviewed candidate declaration.</p>
          <div className="mt-3 grid gap-2 text-slate-300 md:grid-cols-2">
            {[
              ["Support", candidate.capability_declaration?.support_owner],
              ["Incident", candidate.capability_declaration?.incident_owner],
              ["Customer contact", candidate.capability_declaration?.customer_contact],
              ["Release", candidate.capability_declaration?.release_authority],
              ["Rollback", candidate.capability_declaration?.rollback_authority],
              ["Degraded mode", candidate.capability_declaration?.degraded_mode_authority],
            ].map(([label, value]) => <span key={label}>{label}: <code>{value || "unavailable"}</code></span>)}
          </div>
        </div>
        <div className="space-y-2">
          {candidate.artifacts.map((artifact) => (
            <div className="rounded border border-white/10 bg-white/5 p-3 text-sm" key={artifact.target.id}>
              <div className="flex flex-wrap items-center justify-between gap-2"><span className="font-medium">{artifact.target.id}</span><Badge>{artifact.target.os} · {artifact.target.architecture} · {artifact.target.format}</Badge></div>
              <p className="mt-2 break-all text-slate-300">Bytes: <code>{artifact.digest}</code></p>
              <p className="break-all text-slate-400">Immutable ref: <code>{artifact.immutable_ref}</code></p>
              <p className="break-all text-slate-400">Signer: <code>{artifact.signer_ref}</code> · signature <code>{artifact.signature_digest}</code></p>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

function ReceiptCard({ release }: { release: Release }) {
  const publicationReceipts = release.publication_receipts ?? [];
  const updateReceipts = release.client_update_receipts ?? [];
  const recoveryReceipts = release.recovery_receipts ?? [];
  return (
    <Card>
      <CardHeader><CardTitle className="flex items-center gap-2"><ShieldCheck className="h-5 w-5" /> External effect receipts</CardTitle><CardDescription>Durable producer evidence is separate from the release status.</CardDescription></CardHeader>
      <CardContent className="space-y-3">
        {publicationReceipts.length === 0 && updateReceipts.length === 0 && recoveryReceipts.length === 0 && <p className="text-sm text-amber-200">No external effect receipt has been recorded.</p>}
        {publicationReceipts.map((receipt) => <div className="rounded border border-emerald-500/20 bg-emerald-500/5 p-3 text-sm" key={`${receipt.target_id}-${receipt.external_receipt}`}><div className="flex flex-wrap justify-between gap-2"><span>{receipt.target_id} · {receipt.producer}</span><Badge variant="success">{receipt.outcome}</Badge></div><p className="mt-1 break-all text-slate-300">Object: <code>{receipt.destination_object}</code></p><p className="break-all text-slate-400">Receipt: <code>{receipt.external_receipt}</code> · digest <code>{receipt.artifact_digest}</code></p></div>)}
        {updateReceipts.map((receipt) => <div className="rounded border border-cyan-500/20 bg-cyan-500/5 p-3 text-sm" key={`${receipt.target_id}-${receipt.external_receipt}`}><div className="flex flex-wrap justify-between gap-2"><span>{receipt.target_id} · installed client</span><Badge variant="success">{receipt.outcome}</Badge></div><p className="mt-1 break-all text-slate-300">Predecessor: <code>{receipt.predecessor_ref}</code> · successor <code>{receipt.successor_digest}</code></p><p className="break-all text-slate-400">Receipt: <code>{receipt.external_receipt}</code> · version {receipt.verified_version}</p></div>)}
        {recoveryReceipts.map((receipt) => <div className="rounded border border-amber-500/20 bg-amber-500/5 p-3 text-sm" key={receipt.external_receipt}><div className="flex flex-wrap justify-between gap-2"><span>{receipt.action} · owner recovery</span><Badge variant={receipt.health === "stopped" ? "success" : "warning"}>{receipt.outcome}</Badge></div><p className="mt-1 break-all text-slate-300">Deployment: <code>{receipt.deployment_id}</code> · health <code>{receipt.health}</code></p><p className="break-all text-slate-400">Receipt: <code>{receipt.external_receipt}</code> · observed {receipt.observed_at}</p></div>)}
      </CardContent>
    </Card>
  );
}

function HealthCard({ health, missingProof }: { health?: ReleaseHealth; missingProof: string[] }) {
  if (!health) {
    return <Card><CardHeader><CardTitle>Current release standing</CardTitle></CardHeader><CardContent className="text-sm text-amber-200">Canonical health is unavailable. Treat publication and client state as unverified until the dossier can be retrieved.</CardContent></Card>;
  }
  return (
    <Card aria-live="polite">
      <CardHeader><CardTitle>Current release standing</CardTitle><CardDescription>Health and blockers come from the canonical release dossier.</CardDescription></CardHeader>
      <CardContent className="space-y-3 text-sm">
        <div className="flex flex-wrap gap-2"><Badge variant={health.status === "healthy" ? "success" : health.status === "attention" ? "destructive" : "warning"}>Health: {health.status}</Badge><Badge variant={health.publication_verified ? "success" : "warning"}>Public path: {health.publication_verified ? "verified" : "unverified"}</Badge><Badge variant={health.client_updates_healthy ? "success" : "warning"}>Clients: {health.client_updates_healthy ? "healthy" : "unverified"}</Badge></div>
        {health.recovery_standing && <p className="text-slate-300">Recovery standing: <code>{health.recovery_standing}</code></p>}
        <div className="grid gap-2 text-slate-300 md:grid-cols-2"><p>Supported recovery controls: <code>{health.supported_controls?.join(", ") || "none reported"}</code></p><p>Unavailable controls: <code>{health.unsupported_controls?.join(", ") || "none reported"}</code></p></div>
        {missingProof.length > 0 && <div><p className="font-medium text-amber-200">Missing proof</p><ul className="list-disc space-y-1 pl-5 text-amber-100">{missingProof.map((proof) => <li key={proof}>{proof}</li>)}</ul></div>}
        {health.alerts.length > 0 && <div className="space-y-2"><p className="font-medium text-red-200">Required attention</p>{health.alerts.map((alert) => <div className="rounded border border-red-500/20 bg-red-500/5 p-2" key={`${alert.code}-${alert.target ?? "release"}`}><p className="text-red-100">{alert.message}</p><p className="text-slate-300">Next action: {alert.next_action}</p></div>)}</div>}
        {missingProof.length === 0 && health.alerts.length === 0 && <p className="text-emerald-200">No current dossier blockers were reported.</p>}
      </CardContent>
    </Card>
  );
}

function OperationCard({ operation }: { operation?: ReleaseOperation }) {
  if (!operation) return null;
  const terminal = ["completed", "failed", "ambiguous", "cancelled"].includes(operation.status.toLowerCase());
  return (
    <Card aria-live="polite">
      <CardHeader>
        <CardTitle>Release operation</CardTitle>
        <CardDescription>This durable operation can be revisited after a lost response or reconnect.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-2 text-sm text-slate-300">
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant={operation.status === "completed" ? "success" : operation.status === "failed" || operation.status === "ambiguous" ? "destructive" : "warning"}>{operation.status}</Badge>
          {operation.active_stage && <span>Current stage: <code>{operation.active_stage}</code></span>}
          {terminal && <span className="text-slate-400">Terminal standing recorded.</span>}
        </div>
        <div className="grid gap-2 md:grid-cols-2">
          <span>Operation: <code className="break-all">{operation.operation_id}</code></span>
          <span>Idempotency key: <code className="break-all">{operation.idempotency_key || "unavailable"}</code></span>
        </div>
        {operation.error && <p className="text-red-200">Required attention: {operation.error}</p>}
        <p className="text-xs text-slate-400">The page is bound to this operation identity; refresh or reconnect does not create a second execution.</p>
      </CardContent>
    </Card>
  );
}

export function ReleaseDetail() {
  const { id } = useParams<{ id: string }>();
  const [searchParams] = useSearchParams();
  const query = useQuery({ queryKey: ["release", id], queryFn: () => getRelease(id ?? ""), enabled: Boolean(id) });
  const dossierQuery = useQuery({ queryKey: ["release-dossier", id], queryFn: () => getReleaseDossier(id ?? ""), enabled: Boolean(id) });
  const operationId = searchParams.get("operation_id") ?? query.data?.operation_id ?? "";
  const operationQuery = useQuery({ queryKey: ["release-operation", operationId], queryFn: () => getReleaseOperation(operationId), enabled: Boolean(operationId) });
  if (query.isLoading) return <Loader2 className="h-8 w-8 animate-spin text-cyan-400" />;
  if (query.error) return <p className="rounded border border-red-500/30 bg-red-500/10 p-3 text-red-200">{getErrorMessage(query.error)}</p>;
  const release = query.data;
  if (!release) return <p className="text-slate-400">Release record was not returned.</p>;
  return (
    <div className="space-y-6" data-testid="release-detail">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div><Link className="inline-flex items-center gap-1 text-sm text-cyan-300 hover:text-cyan-200" to="/releases"><ArrowLeft className="h-4 w-4" /> All releases</Link><h1 className="mt-3 text-3xl font-bold">{release.release_version} · {release.channel}</h1><p className="mt-1 text-slate-400">Candidate workspace for the exact release execution.</p></div>
        <Badge variant={release.status === "published" ? "success" : release.status === "failed" || release.status === "verify_failed" ? "destructive" : "warning"}>{release.status}</Badge>
      </div>
      <IdentityCard release={release} />
      <CandidateCard release={release} />
      <OperationCard operation={operationQuery.data} />
      {operationQuery.error && <p role="alert" className="rounded border border-red-500/30 bg-red-500/10 p-3 text-red-200">Operation standing unavailable: {getErrorMessage(operationQuery.error)}</p>}
      <HealthCard health={dossierQuery.data?.health} missingProof={dossierQuery.data?.missing_proof ?? []} />
      <Card><CardHeader><CardTitle>Destination and review</CardTitle></CardHeader><CardContent className="grid gap-2 text-sm text-slate-300 md:grid-cols-2"><span>Destination kind: <code>{release.destination_revision?.kind ?? "unavailable"}</code></span><span>Destination: <code>{release.destination_revision?.destination_id ?? "unavailable"}</code></span><span>Review channel: <code>{release.review_binding?.channel ?? release.channel}</code></span><span>Targets: <code>{release.review_binding?.targets?.join(", ") ?? release.platforms?.map((platform) => platform.platform).join(", ") ?? "unavailable"}</code></span><span className="break-all">Evidence digest: <code>{release.review_binding?.evidence_set_digest ?? "unavailable"}</code></span><span className="break-all">Configuration digest: <code>{release.destination_revision?.configuration_digest ?? "unavailable"}</code></span></CardContent></Card>
      <ReceiptCard release={release} />
      <Card><CardHeader><CardTitle>Platform execution</CardTitle><CardDescription>Connection, publication, and verification are shown per target.</CardDescription></CardHeader><CardContent className="space-y-2">{release.platforms?.map((platform) => <div className="flex flex-wrap items-center justify-between gap-2 rounded border border-white/10 bg-white/5 px-3 py-2 text-sm" key={platform.platform}><span>{platform.platform}</span><Badge variant={platform.status === "published" ? "success" : platform.status === "failed" || platform.status === "verify_failed" ? "destructive" : "warning"}>{platform.status}</Badge>{platform.error && <span className="text-red-200">{platform.error}</span>}</div>) ?? <p className="text-sm text-slate-400">No platform execution rows.</p>}</CardContent></Card>
    </div>
  );
}
