import { FormEvent, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { searchPersistencePatterns } from "../../api/persistencePatterns";

export function PersistencePatternPanel() {
  const [scenario, setScenario] = useState("code-facts");
  const [query, setQuery] = useState("catalog generations schema migration repository");
  const search = useMutation({ mutationFn: () => searchPersistencePatterns(query.trim(), scenario.trim()) });
  function submit(event: FormEvent<HTMLFormElement>) { event.preventDefault(); if (query.trim() && scenario.trim()) search.mutate(); }
  const response = search.data;
  return <section className="rounded-panel border border-app-border bg-app-surface p-4" aria-labelledby="persistence-pattern-title">
    <h2 id="persistence-pattern-title" className="text-xl font-semibold">Persistence pattern source evidence</h2>
    <p className="mt-1 text-sm text-app-muted-foreground">Static schema, migration registration and repository references joined by exact table name. This does not establish live database or migration state.</p>
    <form className="mt-4 grid gap-3 md:grid-cols-[220px_1fr_auto]" onSubmit={submit}>
      <label className="flex flex-col gap-1 text-sm">Scenario slug<input aria-label="Persistence scenario" value={scenario} onChange={(event) => setScenario(event.target.value)} className="rounded-md border border-app-border bg-app-background px-3 py-2" /></label>
      <label className="flex flex-col gap-1 text-sm">Pattern query<input aria-label="Persistence query" value={query} onChange={(event) => setQuery(event.target.value)} className="rounded-md border border-app-border bg-app-background px-3 py-2" /></label>
      <button type="submit" disabled={search.isPending} className="rounded-md bg-app-primary px-4 py-2 text-sm font-semibold text-app-primary-foreground md:self-end">{search.isPending ? "Searching…" : "Search patterns"}</button>
    </form>
    {search.error && <p role="alert" className="mt-3 text-sm text-app-danger">{search.error.message}</p>}
    {response && <div className="mt-4 space-y-3"><p className="text-xs text-app-muted-foreground">{response.scope}</p>
      {response.results.length===0 && !(response.unknowns?.length) && <p>No joined persistence pattern matched this query in the bounded source scope.</p>}
      {response.results.map((hit) => <article key={hit.id} className="rounded-md border border-app-border p-3"><h3 className="font-semibold">{hit.title}</h3><p className="text-sm">{hit.summary}</p>
        <dl className="mt-2 grid gap-2 text-xs sm:grid-cols-4">{[hit.schema,hit.schema_binding,hit.migration,hit.repository].map((evidence) => <div key={evidence.layer} className="rounded border border-app-border p-2"><dt className="font-semibold">{evidence.layer} · {evidence.status}</dt><dd>{evidence.path}:{evidence.line}</dd><dd>{evidence.basis}</dd><dd className="break-all">sha256:{evidence.sha256}</dd></div>)}</dl>
        <ul className="mt-2 list-disc pl-5 text-xs text-app-muted-foreground">{hit.limits.map((limit) => <li key={limit}>{limit}</li>)}</ul>
      </article>)}
      {(response.unknowns ?? []).map((item) => <p key={`${item.entity}:${item.reason}`} role="status" data-testid="persistence-unknown" className="rounded border border-app-border p-3 text-sm">UNKNOWN lineage for {item.entity}: {item.reason}. Schema anchor: {item.schema.path}:{item.schema.line} · sha256:{item.schema.sha256}. Exact migration/repository lineage was not established.</p>)}
      {response.degradations.map((degradation) => <p key={degradation} role="status" className="text-sm text-app-muted-foreground">{degradation}</p>)}
    </div>}
  </section>;
}
