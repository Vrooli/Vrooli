import { Link, useParams } from "react-router-dom";
import { useMutation, useQuery } from "@tanstack/react-query";

import { checkPersonaHealth, getPersona, listChannels, listHandoffs, retrieveCode } from "../api/persona";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { selectors } from "../consts/selectors";
import { HandoffState } from "@vrooli/proto-types/persona/v1/handoffs/handoffs_pb";
import { PersonaKind, PersonaStatus } from "@vrooli/proto-types/persona/v1/personas/personas_pb";

export function PersonaDetailPage() {
  const { personaId = "" } = useParams();
  const personaQuery = useQuery({ queryKey: ["persona", personaId], queryFn: () => getPersona(personaId), enabled: Boolean(personaId) });
  const healthQuery = useQuery({ queryKey: ["persona-health", personaId], queryFn: () => checkPersonaHealth(personaId), enabled: Boolean(personaId) });
  const handoffsQuery = useQuery({ queryKey: ["handoffs", personaId], queryFn: () => listHandoffs(personaId), enabled: Boolean(personaId) });
  const channelsQuery = useQuery({ queryKey: ["channels", personaId], queryFn: () => listChannels(personaId), enabled: Boolean(personaId) });
  const codeMutation = useMutation({ mutationFn: (channelId: string) => retrieveCode(personaId, channelId, "persona-verification") });
  const persona = personaQuery.data;

  if (personaQuery.isPending) return <p>Loading persona…</p>;
  if (personaQuery.isError || !persona) return <p role="alert">This persona could not be loaded. <Link className="underline" to="/personas">Return to the registry.</Link></p>;

  return <section data-testid={selectors.pages.personaDetail} aria-labelledby="persona-detail-heading" className="flex flex-col gap-6">
    <div><Link to="/personas" className="text-sm font-medium text-app-primary underline-offset-4 hover:underline">← All personas</Link><div className="mt-3 flex flex-wrap items-end justify-between gap-4"><div><p className="text-sm font-semibold uppercase tracking-[0.18em] text-app-primary">{persona.kind === PersonaKind.BUSINESS ? "Business persona" : "Personal persona"}</p><h2 id="persona-detail-heading" className="mt-2 text-3xl font-semibold">{persona.displayName || persona.legalBasis?.subjectName}</h2></div><span className="rounded-full border border-app-border px-3 py-1 text-sm">{persona.status === PersonaStatus.ARCHIVED ? "Archived" : "Active"}</span></div></div>
    <div className="grid gap-4 lg:grid-cols-3">
      <Card><CardHeader><CardTitle>Legal basis</CardTitle><CardDescription>Immutable after creation</CardDescription></CardHeader><CardContent className="space-y-3 text-sm"><Detail label="Subject" value={persona.legalBasis?.subjectName || "—"} /><Detail label="Subject ID" value={persona.legalBasis?.subjectId || "—"} /><Detail label="Basis" value={persona.legalBasis?.basisType || "—"} /></CardContent></Card>
      <Card><CardHeader><CardTitle>Health</CardTitle><CardDescription>Dependencies and staleness</CardDescription></CardHeader><CardContent>{healthQuery.isPending ? <p className="text-sm text-app-muted-foreground">Checking…</p> : healthQuery.isError ? <p role="alert" className="text-sm text-app-danger">Health is unavailable.</p> : healthQuery.data?.length ? <ul className="space-y-2 text-sm">{healthQuery.data.map((finding) => <li key={finding.code} className={finding.blocking ? "text-app-danger" : "text-app-muted-foreground"}><span className="font-semibold">{finding.code}</span> {finding.message}</li>)}</ul> : <p className="text-sm text-app-success">No blocking findings.</p>}</CardContent></Card>
      <Card><CardHeader><CardTitle>Human queue</CardTitle><CardDescription>Handoffs attached to this identity</CardDescription></CardHeader><CardContent>{handoffsQuery.data?.length ? <ul className="space-y-2 text-sm">{handoffsQuery.data.slice(0, 4).map((handoff) => <li key={handoff.id}><Link className="font-medium text-app-primary underline-offset-4 hover:underline" to={`/handoffs/${handoff.id}`}>{handoff.title}</Link><span className="ml-2 text-app-muted-foreground">{handoff.state === HandoffState.AWAITING_HUMAN ? "waiting" : "resolved"}</span></li>)}</ul> : <p className="text-sm text-app-muted-foreground">No handoffs recorded.</p>}</CardContent></Card>
    </div>
    <Card><CardHeader><CardTitle>Controlled routes</CardTitle><CardDescription>References only; provider credentials remain outside this scenario.</CardDescription></CardHeader><CardContent>{channelsQuery.isPending ? <p className="text-sm text-app-muted-foreground">Checking routes…</p> : channelsQuery.data?.length ? <div className="space-y-3">{channelsQuery.data.map((channel) => <div key={channel.id} className="flex flex-wrap items-center justify-between gap-3 rounded-panel border border-app-border p-3"><div><p className="font-medium">{channel.address}</p><p className="text-sm text-app-muted-foreground">{channel.adapter} · {channel.enabled ? "Enabled" : "Disabled"}</p></div><button type="button" className="min-h-11 rounded-control border border-app-border px-3 text-sm font-semibold hover:border-app-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50" onClick={() => codeMutation.mutate(channel.id)} disabled={codeMutation.isPending}>{codeMutation.isPending ? "Retrieving…" : "Retrieve code"}</button></div>)}<p role="status" aria-live="polite" className="text-sm text-app-muted-foreground">{codeMutation.data ? `One-time code ${codeMutation.data.code}; expires ${codeMutation.data.expiresAt ? new Date(Number(codeMutation.data.expiresAt.seconds) * 1000).toLocaleString() : "at the provider deadline"}.` : "Retrieved one-time codes and their expiry are announced here."}</p>{codeMutation.isError ? <p role="alert" className="text-sm text-app-danger">The named route refused retrieval; no fallback route was used.</p> : null}</div> : <p className="text-sm text-app-muted-foreground">No controlled route is configured yet.</p>}</CardContent></Card>
    <Card><CardHeader><CardTitle>Record boundary</CardTitle><CardDescription>What persona keeps, and what it deliberately does not.</CardDescription></CardHeader><CardContent className="grid gap-3 text-sm md:grid-cols-2"><Boundary title="Kept here" body="Legal basis, controlled-channel references, document bindings, account linkage and append-only evidence." /><Boundary title="Never returned here" body="Mailbox secrets, provider credentials, document bytes, card details and money movement." /></CardContent></Card>
  </section>;
}

function Detail({ label, value }: { label: string; value: string }) { return <div><dt className="text-xs uppercase tracking-wide text-app-muted-foreground">{label}</dt><dd className="mt-1 break-words font-medium">{value}</dd></div>; }
function Boundary({ title, body }: { title: string; body: string }) { return <div className="rounded-panel border border-app-border p-3"><p className="font-semibold">{title}</p><p className="mt-1 text-app-muted-foreground">{body}</p></div>; }
