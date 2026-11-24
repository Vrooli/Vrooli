import { FormEvent, useState } from "react";
import { useMutation } from "@tanstack/react-query";

import { searchFeatureFlows } from "../../api/featureFlows";

export function FeatureFlowPanel() {
  const [scenario, setScenario] = useState("code-facts");
  const [path, setPath] = useState("api");
  const [query, setQuery] = useState("SearchComposition");
  const search = useMutation({
    mutationFn: (request: { query: string; scenario: string; path: string }) =>
      searchFeatureFlows(request.query, request.scenario, request.path, 5),
  });
  const response = search.data;

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const request = { query: query.trim(), scenario: scenario.trim(), path: path.trim() };
    if (request.query && request.scenario && request.path) search.mutate(request);
  }

  return (
    <section className="rounded-panel border border-app-border bg-app-surface p-4" aria-labelledby="feature-flow-title">
      <h2 id="feature-flow-title" className="text-xl font-semibold">Feature flow source evidence</h2>
      <p className="mt-1 text-sm text-app-muted-foreground">
        Up to 5 results and 3 source call-site edges in one selected scenario subtree. Each edge shows its caller location and graph identity, plus a resolved target location when available. Static edges are partial evidence: they do not prove runtime entry, dynamic dispatch, execution conditions, complete flow, or reachability.
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
          Flow question
          <input aria-label="Flow question" value={query} onChange={(event) => setQuery(event.target.value)} className="rounded-md border border-app-border bg-app-background px-3 py-2" />
        </label>
        <button type="submit" disabled={search.isPending || !query.trim() || !scenario.trim() || !path.trim()} className="rounded-md bg-app-primary px-4 py-2 text-sm font-semibold text-app-primary-foreground lg:self-end">
          {search.isPending ? "Searching…" : "Search source flows"}
        </button>
      </form>

      {search.error && <p role="alert" className="mt-3 text-sm text-app-danger">Feature-flow search failed: {search.error.message}</p>}
      {response && <div className="mt-4 space-y-3">
        <p data-testid="feature-flow-scope" className="text-xs text-app-muted-foreground">Scope: scenario {response.scenario} · subtree {response.path} · query {response.query} · result limit 5.</p>
        {response.results.length === 0 && <p>No matching source flow was found in this bounded graph. This is not proof that no runtime path exists.</p>}
        {response.results.map((hit) => <article key={hit.id} className="rounded-md border border-app-border p-3">
          <h3 className="font-semibold">{hit.title}</h3>
          <p className="mt-1 text-sm">{hit.text}</p>
          <p className="mt-1 text-xs text-app-muted-foreground">Source graph: {hit.graph_hash} · attestation: {hit.attestation}</p>
          <ul className="mt-2 space-y-1 text-xs font-mono text-app-muted-foreground">
            {hit.steps.map((step, index) => <li key={`${step.path}:${step.line}:${index}`}>
              Caller source {step.path}:{step.line} · graph {step.graph_hash || hit.graph_hash} · {step.from} calls {step.call}{step.to ? ` → ${step.to} · target source ${step.to_path || "path unavailable"} · target graph ${step.to_graph_hash || "identity unavailable"} (${step.resolution})` : ` → unresolved callee; no target path (${step.resolution})`}
            </li>)}
          </ul>
        </article>)}
        {response.degradations.map((item) => <p key={item.code} role="status" className="text-sm text-app-muted-foreground">Partial source evidence · {item.code}: {item.message}</p>)}
      </div>}
    </section>
  );
}
