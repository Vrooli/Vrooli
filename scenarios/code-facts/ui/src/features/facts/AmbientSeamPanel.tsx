import { FormEvent, useState } from "react";
import { useMutation } from "@tanstack/react-query";

import { searchAmbientSeams } from "../../api/ambientSeams";

export function AmbientSeamPanel() {
  const [scenario, setScenario] = useState("code-facts");
  const [query, setQuery] = useState("clock environment HTTP client logger");
  const search = useMutation({ mutationFn: () => searchAmbientSeams(query.trim(), scenario.trim()) });
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (query.trim() && scenario.trim()) search.mutate();
  }
  const response = search.data;
  return (
    <section className="rounded-panel border border-app-border bg-app-surface p-4" aria-labelledby="ambient-seam-title">
      <h2 id="ambient-seam-title" className="text-xl font-semibold">Ambient seam source evidence</h2>
      <p className="mt-1 text-sm text-app-muted-foreground">Static Go call sites only. Query matches are lexical and show which evidence fields matched. A direct AST argument containment is source syntax, not data flow or runtime wiring; test-file suffixes do not establish suite execution or test substitution, and non-test paths do not establish production binary membership.</p>
      <form className="mt-4 grid gap-3 md:grid-cols-[220px_1fr_auto]" onSubmit={submit}>
        <label className="flex flex-col gap-1 text-sm">Scenario slug<input aria-label="Scenario slug" value={scenario} onChange={(event) => setScenario(event.target.value)} className="rounded-md border border-app-border bg-app-background px-3 py-2" /></label>
        <label className="flex flex-col gap-1 text-sm">Seam query<input aria-label="Ambient seam query" value={query} onChange={(event) => setQuery(event.target.value)} className="rounded-md border border-app-border bg-app-background px-3 py-2" /></label>
        <button type="submit" disabled={search.isPending} className="rounded-md bg-app-primary px-4 py-2 text-sm font-semibold text-app-primary-foreground md:self-end">{search.isPending ? "Searching…" : "Search seams"}</button>
      </form>
      {search.error && <p role="alert" className="mt-3 text-sm text-app-danger">{search.error.message}</p>}
      {response && <div className="mt-4 space-y-3">
        <p className="text-xs text-app-muted-foreground">{response.scope}</p>
        <p className="text-xs text-app-muted-foreground">Lexical match status: {response.match_status ?? "UNKNOWN"}. Ties and unsupported query syntax remain unknown; a lexical match does not establish relevance.</p>
        {response.results.length === 0 && <p>No ambient seam matched in this bounded source scope.</p>}
        {response.results.map((hit) => <article key={hit.id} className="rounded-md border border-app-border p-3">
          <h3 className="font-semibold">{hit.title}</h3>
          <p className="mt-1 text-sm">{hit.summary}</p>
          <p className="mt-1 text-xs text-app-muted-foreground">Candidate match status: {hit.match_status ?? "UNKNOWN"}</p>
          <p className="mt-1 text-xs text-app-muted-foreground">Lexical query matches: {hit.matched_terms.join(", ")} · evidence fields: {hit.matched_fields.join(", ")}</p>
          <ul className="mt-1 space-y-1 text-xs text-app-muted-foreground">{(hit.match_evidence ?? []).map((evidence, index) => <li key={`${evidence.term}-${evidence.field}-${index}`}>{evidence.term} matched {evidence.field}={evidence.value} at {evidence.source_path}:{evidence.line} · sha256:{evidence.source_sha256}</li>)}</ul>
          <p className="mt-1 text-xs text-app-muted-foreground">{hit.source_path}:{hit.line} · sha256:{hit.source_sha256}</p>
          <p className="text-xs text-app-muted-foreground">Context {hit.context_symbol} · {hit.source_layer} · AST argument context {hit.argument_context_status}{hit.argument_callee ? `: ${hit.expression} appears within arguments to ${hit.argument_callee} at line ${hit.argument_line}` : ": no direct argument container found; aliases and data flow remain unknown"}</p>
          <p className="text-xs text-app-muted-foreground">Evidence basis: {hit.basis}</p>
          <p className="mt-1 text-xs text-app-muted-foreground">{hit.limit}</p>
        </article>)}
        {response.degradations.map((item) => <p key={item.code} role="status" className="text-sm text-app-muted-foreground">{item.code}: {item.message}</p>)}
      </div>}
    </section>
  );
}
