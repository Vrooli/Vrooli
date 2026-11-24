import { FormEvent, useState } from "react";
import { useMutation } from "@tanstack/react-query";

import { searchInvariantCandidates } from "../../api/invariantCandidates";

const UNKNOWN_LAYERS = ["Type", "Database", "Test", "Runtime"] as const;

export function InvariantCandidatePanel() {
  const [scenario, setScenario] = useState("code-facts");
  const [path, setPath] = useState("api");
  const [query, setQuery] = useState("request limit cache constraint");
  const search = useMutation({ mutationFn: () => searchInvariantCandidates(query.trim(), scenario.trim(), path.trim(), 10) });
  const response = search.data;

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (query.trim() && scenario.trim() && path.trim()) search.mutate();
  }

  return (
    <section className="rounded-panel border border-app-border bg-app-surface p-4" aria-labelledby="invariant-candidate-title">
      <h2 id="invariant-candidate-title" className="text-xl font-semibold">Invariant candidate evidence</h2>
      <p className="mt-1 text-sm text-app-muted-foreground">
        Search up to 10 source patterns in one selected scenario and relative directory. Candidate location is not proof that an invariant is actively enforced or verified.
      </p>
      <form className="mt-4 grid gap-3 lg:grid-cols-[180px_220px_1fr_auto]" onSubmit={submit}>
        <label className="flex flex-col gap-1 text-sm">
          Scenario slug
          <input aria-label="Scenario slug" value={scenario} onChange={(event) => setScenario(event.target.value)} className="rounded-md border border-app-border bg-app-background px-3 py-2" />
        </label>
        <label className="flex flex-col gap-1 text-sm">
          Relative source path
          <input aria-label="Relative source path" value={path} onChange={(event) => setPath(event.target.value)} className="rounded-md border border-app-border bg-app-background px-3 py-2" />
        </label>
        <label className="flex flex-col gap-1 text-sm">
          Candidate question
          <input aria-label="Candidate question" value={query} onChange={(event) => setQuery(event.target.value)} className="rounded-md border border-app-border bg-app-background px-3 py-2" />
        </label>
        <button type="submit" disabled={search.isPending || !query.trim() || !scenario.trim() || !path.trim()} className="rounded-md bg-app-primary px-4 py-2 text-sm font-semibold text-app-primary-foreground lg:self-end">
          {search.isPending ? "Searching…" : "Search candidates"}
        </button>
      </form>

      {search.error && <p role="alert" className="mt-3 text-sm text-app-danger">Candidate search failed: {search.error.message}</p>}
      {response && <div className="mt-4 space-y-3">
        <p className="text-xs text-app-muted-foreground">Scope: scenario {response.scenario}; {response.scope}; result limit 10.</p>
        <p className="rounded-md border border-app-border p-2 text-xs text-app-muted-foreground">
          Test Genie verification identity: UNKNOWN. This candidate API has no durable requirement ID, run ID, or observed source identity joined to a candidate. Requirement status and test-shaped source are not a run receipt.
        </p>
        {response.results.length === 0 && <p>No candidate pattern matched in this bounded source scan. This does not establish that an invariant is absent.</p>}
        {response.results.map((candidate) => <article key={candidate.id} className="rounded-md border border-app-border p-3">
          <h3 className="font-semibold">{candidate.title}</h3>
          <p className="mt-1 text-sm">{candidate.text}</p>
          <p className="mt-1 text-xs text-app-muted-foreground">Candidate mechanism: {candidate.evidenceKind} · {candidate.attestation}</p>
          <p className="mt-1 break-all font-mono text-xs text-app-muted-foreground">Candidate identity: record {candidate.id} · source observation {candidate.scenario}/{candidate.path}:{candidate.line} · file SHA-256 {candidate.sha256}</p>
          <p className="mt-1 text-xs text-app-muted-foreground">Verification identity: UNKNOWN · no exact Test Genie requirement/run receipt or matching observed source identity is attached to this candidate.</p>
          <dl className="mt-3 grid gap-1 text-xs sm:grid-cols-2">
            {UNKNOWN_LAYERS.map((layer) => <div key={layer} className="flex gap-2"><dt>{layer} enforcement:</dt><dd className="font-semibold">UNKNOWN</dd></div>)}
          </dl>
          <p className="mt-2 text-xs text-app-muted-foreground">This source location is a candidate only; it does not prove active enforcement, and test-shaped source does not prove a passing verification.</p>
        </article>)}
        {response.degradations.map((item, index) => <p key={`${item.code}:${index}`} role="status" className="text-sm text-app-muted-foreground">Partial source evidence · {item.code}: {item.message}</p>)}
      </div>}
    </section>
  );
}
