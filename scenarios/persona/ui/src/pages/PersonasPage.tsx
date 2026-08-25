import { useState } from "react";
import { Link } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { createPersona, listPersonas } from "../api/persona";
import { Button } from "@vrooli/react-component-library/Button/1.2.0";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { Input } from "../components/ui/input";
import { selectors } from "../consts/selectors";
import { PersonaKind, PersonaStatus } from "@vrooli/proto-types/persona/v1/personas/personas_pb";

export function PersonasPage() {
  const queryClient = useQueryClient();
  const [kind, setKind] = useState<PersonaKind>(PersonaKind.PERSONAL);
  const [subjectId, setSubjectId] = useState("");
  const [subjectName, setSubjectName] = useState("");
  const [basisType, setBasisType] = useState("operator-authorisation");
  const [displayName, setDisplayName] = useState("");
  const [identifierType, setIdentifierType] = useState("passport");
  const [identifierValue, setIdentifierValue] = useState("");
  const personasQuery = useQuery({ queryKey: ["personas", "all"], queryFn: () => listPersonas(true) });
  const mutation = useMutation({
    mutationFn: () => createPersona({ kind, subjectId, subjectName, basisType, displayName: displayName || subjectName, identifierType, identifierValue }),
    onSuccess: () => {
      setSubjectId(""); setSubjectName(""); setDisplayName(""); setIdentifierValue("");
      void queryClient.invalidateQueries({ queryKey: ["personas"] });
    },
  });

  return <section data-testid={selectors.pages.personas} aria-labelledby="personas-heading" className="flex flex-col gap-6">
    <div><p className="text-sm font-semibold uppercase tracking-[0.18em] text-app-primary">Identity registry</p><h2 id="personas-heading" className="mt-2 text-3xl font-semibold">Personas</h2><p className="mt-2 max-w-2xl text-app-muted-foreground">A persona is selected deliberately. Its legal basis is required at creation and remains immutable thereafter.</p></div>
    <div className="grid gap-6 xl:grid-cols-[1fr_360px]">
      <Card><CardHeader><CardTitle>Configured identities</CardTitle><CardDescription>{personasQuery.isPending ? "Loading records…" : `${personasQuery.data?.length ?? 0} retained record(s)`}</CardDescription></CardHeader><CardContent>
        {personasQuery.isError ? <p role="alert" className="text-sm text-app-danger">Unable to load personas. Retry from the page.</p> : null}
        {!personasQuery.isPending && !personasQuery.data?.length ? <p className="text-sm text-app-muted-foreground">No persona exists yet. Create the first one with a declared legal basis.</p> : null}
        <div className="grid gap-3">
          {personasQuery.data?.map((persona) => <Link key={persona.id} to={`/personas/${persona.id}`} className="rounded-panel border border-app-border p-4 transition hover:border-app-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50">
            <div className="flex items-start justify-between gap-3"><div><p className="font-semibold">{persona.displayName || persona.legalBasis?.subjectName || "Unnamed persona"}</p><p className="mt-1 text-sm text-app-muted-foreground">{persona.kind === PersonaKind.BUSINESS ? "Business" : "Personal"} · {persona.status === PersonaStatus.ARCHIVED ? "Archived" : "Active"}</p></div><span aria-label={persona.status === PersonaStatus.ARCHIVED ? "Archived" : "Active"} className="rounded-full border border-app-border px-2 py-1 text-xs">{persona.status === PersonaStatus.ARCHIVED ? "Archived" : "Active"}</span></div>
            <p className="mt-3 text-xs text-app-muted-foreground">Legal basis: {persona.legalBasis?.basisType || "Not declared"}</p>
          </Link>)}
        </div>
      </CardContent></Card>
      <Card><CardHeader><CardTitle>Create a persona</CardTitle><CardDescription>Every field supports a durable authorization decision.</CardDescription></CardHeader><CardContent>
        <form className="flex flex-col gap-4" onSubmit={(event) => { event.preventDefault(); mutation.mutate(); }}>
          <label className="text-sm font-medium">Kind<select value={kind} onChange={(event) => { const nextKind = Number(event.target.value) as PersonaKind; setKind(nextKind); setIdentifierType(nextKind === PersonaKind.BUSINESS ? "business_registration" : "passport"); }} className="mt-1 min-h-11 w-full rounded-control border border-app-border bg-app-surface px-3"><option value={PersonaKind.PERSONAL}>Personal</option><option value={PersonaKind.BUSINESS}>Business</option></select></label>
          <label className="text-sm font-medium">Subject ID<Input required value={subjectId} onChange={(event) => setSubjectId(event.target.value)} placeholder="legal-subject-001" /></label>
          <label className="text-sm font-medium">Subject name<Input required value={subjectName} onChange={(event) => setSubjectName(event.target.value)} placeholder="Named human or entity" /></label>
          <label className="text-sm font-medium">Basis type<Input required value={basisType} onChange={(event) => setBasisType(event.target.value)} /></label>
          <label className="text-sm font-medium">Identifier type<select value={identifierType} onChange={(event) => setIdentifierType(event.target.value)} className="mt-1 min-h-11 w-full rounded-control border border-app-border bg-app-surface px-3"><option value="passport">Passport</option><option value="national_id">National ID</option><option value="government_id">Government ID</option><option value="business_registration">Business registration</option><option value="tax_id">Tax ID</option><option value="duns">D-U-N-S</option></select></label>
          <label className="text-sm font-medium">Identifier value<Input required value={identifierValue} onChange={(event) => setIdentifierValue(event.target.value)} placeholder="Reference held by the legal subject" /></label>
          <label className="text-sm font-medium">Display name<Input value={displayName} onChange={(event) => setDisplayName(event.target.value)} placeholder="Optional operator label" /></label>
          {mutation.isError ? <p role="alert" className="text-sm text-app-danger">Creation was refused. Check the legal basis and try again.</p> : null}
          {mutation.isSuccess ? <p role="status" className="text-sm text-app-success">Persona created and retained.</p> : null}
          <Button type="submit" disabled={mutation.isPending}>{mutation.isPending ? "Creating…" : "Create persona"}</Button>
        </form>
      </CardContent></Card>
    </div>
  </section>;
}
