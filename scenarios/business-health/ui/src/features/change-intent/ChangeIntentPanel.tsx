import { FormEvent, useState } from "react";
import { useMutation } from "@tanstack/react-query";

import { assessChangeIntent } from "../../api/changeIntent";

export function ChangeIntentPanel() {
  const [intent, setIntent] = useState("");
  const assessment = useMutation({ mutationFn: () => assessChangeIntent(intent.trim(), 10) });

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (intent.trim()) assessment.mutate();
  }

  return (
    <section className="rounded-panel border border-app-border bg-app-surface p-4" aria-labelledby="change-intent-title">
      <h3 id="change-intent-title" className="text-lg font-semibold">Change intent traceability</h3>
      <p className="mt-1 text-sm text-app-muted-foreground">
        Relates one explicit statement to this scenario’s checked-in requirements and exact source references. This is a bounded traceability view, not a planner or a verification result.
      </p>
      <form className="mt-3 grid gap-2" onSubmit={submit}>
        <label htmlFor="change-intent-input" className="text-sm font-medium">Change-intent statement · business-health</label>
        <textarea id="change-intent-input" value={intent} maxLength={4096} onChange={(event) => setIntent(event.target.value)} rows={3} className="rounded-md border border-app-border bg-app-background px-3 py-2 text-sm" />
        <button type="submit" disabled={assessment.isPending || !intent.trim()} className="w-fit rounded-md bg-app-primary px-4 py-2 text-sm font-semibold text-app-primary-foreground">
          {assessment.isPending ? "Assessing…" : "Trace intent"}
        </button>
      </form>
      {assessment.error && <p role="alert" className="mt-3 text-sm text-app-danger">Assessment failed: {assessment.error.message}</p>}
      {assessment.data && <div className="mt-4 space-y-3">
        <p className="text-xs text-app-muted-foreground">{assessment.data.scope} · at most 10 candidates</p>
        {assessment.data.results.length === 0 && <p>No requirement candidate matched this bounded source view.</p>}
        {assessment.data.results.map((result) => <article key={result.requirement_id} className="rounded-md border border-app-border p-3">
          <h4 className="font-semibold">{result.requirement_id} · {result.title}</h4>
          <p className="mt-1 text-sm">Scenario: <strong>{result.scenario}</strong> · Relation: <strong>{result.relation.toUpperCase()}</strong> · {result.relation_basis} · declared status {result.declared_status}</p>
          <p className="mt-1 text-xs font-mono text-app-muted-foreground">Requirement: {result.requirement_source.path}:{result.requirement_source.line} · SHA-256 {result.requirement_source.sha256}</p>
          {result.implementation_anchors.length === 0 && <p className="mt-1 text-xs text-app-muted-foreground">No tagged source reference was found in the bounded scan.</p>}
          {result.implementation_anchors.map((anchor, index) => <p key={`${anchor.path}:${anchor.line}:${index}`} className="mt-1 break-all text-xs font-mono text-app-muted-foreground">{anchor.path}:{anchor.line} · {anchor.kind} · {anchor.attestation} · SHA-256 {anchor.sha256}</p>)}
        </article>)}
        {assessment.data.limitations.map((limitation) => <p key={limitation} role="note" className="text-xs text-app-muted-foreground">Limit: {limitation}</p>)}
        {assessment.data.degradations.map((degradation, index) => <p key={`${degradation}:${index}`} role="status" className="text-xs text-app-muted-foreground">Partial scan: {degradation}</p>)}
      </div>}
    </section>
  );
}
