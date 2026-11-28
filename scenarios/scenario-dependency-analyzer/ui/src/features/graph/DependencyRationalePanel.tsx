import { FormEvent, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { Card, CardContent, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1.3.1";

import { searchDependencyRationale } from "../../api/dependencyRationale";

interface DependencyRationalePanelProps {
  scenario: string;
}

const fields = [
  ["Description", "description"],
  ["Purpose", "purpose"],
  ["Runtime-only rationale", "runtimeOnlyRationale"],
  ["Degraded behavior", "degradedBehavior"],
] as const;

export function DependencyRationalePanel({ scenario }: DependencyRationalePanelProps) {
  const [query, setQuery] = useState("dependency purpose runtime-only rationale degraded behavior");
  const search = useMutation({ mutationFn: () => searchDependencyRationale(query.trim(), scenario, 10) });

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (query.trim()) search.mutate();
  }

  return (
    <Card className="glass border border-border/40" aria-labelledby="dependency-rationale-heading">
      <CardHeader>
        <CardTitle id="dependency-rationale-heading" className="text-base">Authored dependency rationale</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3 text-sm">
        <p className="text-xs text-muted-foreground">
          One selected scenario scope · results show consumer → dependency declarations from `.vrooli/service.json`. This is separate from detected graph edges and package approval rationale. It does not prove wiring, reachability, health, or runtime use.
        </p>
        <form className="space-y-2" onSubmit={submit}>
          <label htmlFor="dependency-rationale-query" className="text-xs font-medium text-foreground">Rationale query</label>
          <input id="dependency-rationale-query" value={query} onChange={(event) => setQuery(event.target.value)} maxLength={512} className="w-full rounded-md border border-border/60 bg-background/70 px-3 py-2 text-xs" />
          <button type="submit" disabled={search.isPending || !query.trim()} className="rounded-md bg-primary px-3 py-2 text-xs font-semibold text-primary-foreground">
            {search.isPending ? "Searching…" : "Search this scenario"}
          </button>
        </form>
        {search.error && <p role="alert" className="text-xs text-destructive">Rationale search failed: {search.error.message}</p>}
        {search.data && <div className="space-y-3">
          <p className="text-[11px] text-muted-foreground">Scan: {search.data.scan.basis} · complete {String(search.data.scan.complete)} · runtime status known {String(search.data.scan.runtimeStatusKnown)}</p>
          {search.data.results.length === 0 && <p className="rounded-md border border-border/40 bg-background/40 p-2 text-xs text-muted-foreground">No declaration rationale matched this query in the selected scenario. This is not evidence that the graph edge or dependency is absent.</p>}
          {search.data.results.map((result) => <article key={result.id} className="space-y-2 rounded-md border border-border/50 bg-background/40 p-3">
            <h3 className="font-medium text-foreground">{result.consumer} → {result.dependency}</h3>
            <p className="text-[11px] text-muted-foreground">Manifest declaration · {result.required ? "required" : "optional"} · {result.enabled ? "enabled" : "disabled"} · {result.runtimeOnly ? "runtime-only" : "not marked runtime-only"}</p>
            <dl className="space-y-2 text-xs">
              {fields.map(([label, key]) => {
                const value = result.evidence[key];
                return value ? <div key={key}><dt className="font-semibold text-foreground">{label}</dt><dd className="whitespace-pre-wrap text-muted-foreground">{value}</dd></div> : null;
              })}
            </dl>
            <p className="break-all border-t border-border/30 pt-2 font-mono text-[10px] text-muted-foreground">
              Owner {result.source.owner} · {result.source.basis} · {result.source.path} · {result.source.fingerprint} · observed {result.source.observedAt}
            </p>
          </article>)}
          {search.data.degradations.map((item, index) => <p key={`${item.code}:${index}`} role="status" className="text-xs text-muted-foreground">Partial scan · {item.scenario || scenario} · {item.code}: {item.reason}</p>)}
          {search.data.scan.scenariosFailed > 0 && <p role="status" className="text-xs text-muted-foreground">{search.data.scan.scenariosFailed} manifest(s) could not be read; this scenario’s evidence may be incomplete.</p>}
        </div>}
      </CardContent>
    </Card>
  );
}
