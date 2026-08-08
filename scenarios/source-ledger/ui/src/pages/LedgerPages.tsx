import { useEffect, useState } from "react";
import { Link, useParams, useSearchParams } from "react-router-dom";
import { buildApiUrl } from "@vrooli/api-base";

import { API_BASE } from "../api/client";
import { selectors } from "../consts/selectors";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";

type Scope = { id: string; label: string; frontierTarget: number; wakeBudget: number; maxEntryLines: number; facets?: Facet[] };
type Facet = { id: string; label: string; retentionPolicy?: string; residentBudget?: number };
type Entry = { id: string; body: string; facetId: string; kind: string };
type Frontier = { eligibleCount: number; target: number; nodes: { id: string; entryId: string; facetId: string; depth: number }[] };
type RecallHit = { entryId: string; facetId: string; text: string; score: number };

async function post<T>(path: string, body: unknown): Promise<T> {
  const response = await fetch(buildApiUrl(path, { baseUrl: API_BASE }), {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(body),
    cache: "no-store",
  });
  if (!response.ok) throw new Error(`Source Ledger request failed (${response.status})`);
  return (await response.json()) as T;
}

function useAsync<T>(load: () => Promise<T>, deps: readonly unknown[]) {
  const [state, setState] = useState<{ data?: T; error?: string }>({});
  useEffect(() => {
    let live = true;
    void load().then((data) => live && setState({ data })).catch((error: unknown) => live && setState({ error: String(error) }));
    return () => { live = false; };
    // The caller controls the request identity through deps.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);
  return state;
}

export function LedgerDashboardPage() {
  const scopes = useAsync(() => post<{ scopes: Scope[] }>("/vrooli.source_ledger.v1.scopes.ScopesService/ListScopes", {}), []);
  const [query, setQuery] = useState("");
  return <section data-testid={selectors.pages.dashboard} className="flex min-w-0 flex-col gap-5" aria-labelledby="ledger-heading">
    <div><h2 id="ledger-heading" className="text-2xl font-semibold">Ledger corpora</h2><p className="text-app-muted-foreground">Every scope is one bounded view over the shared append-only journal.</p></div>
    {scopes.error && <p className="text-app-danger">{scopes.error}</p>}
    <div className="grid min-w-0 gap-4 md:grid-cols-2">
      {scopes.data?.scopes.map((scope) => <ScopeCard key={scope.id} scope={scope} />)}
    </div>
    <Card className="min-w-0"><CardHeader><CardTitle>Cross-scope search</CardTitle></CardHeader><CardContent className="flex min-w-0 flex-wrap gap-2"><input aria-label="Search every scope" className="min-w-0 flex-1 rounded-control border border-app-border bg-app-background px-3 py-2" value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Search by meaning" /><Link to={`/search?q=${encodeURIComponent(query)}`}><Button disabled={!query.trim()}>Search</Button></Link></CardContent></Card>
  </section>;
}

function ScopeCard({ scope }: { scope: Scope }) {
  const entries = useAsync(() => post<{ entries: Entry[] }>("/vrooli.source_ledger.v1.journal.JournalService/ListEntries", { scope: scope.id, limit: 500 }), [scope.id]);
  const frontier = useAsync(() => post<Frontier>("/vrooli.source_ledger.v1.forest.ForestService/GetFrontier", { scope: scope.id, limit: 1 }), [scope.id]);
  return <Card className="min-w-0"><CardHeader><CardTitle><Link className="break-words hover:underline" to={`/scopes/${encodeURIComponent(scope.id)}`}>{scope.label || scope.id}</Link></CardTitle><p className="break-words text-xs text-app-muted-foreground">{scope.id}</p></CardHeader><CardContent className="grid min-w-0 grid-cols-2 gap-3 text-sm"><Metric label="Entries" value={entries.data?.entries.length ?? "…"} /><Metric label="Frontier" value={frontier.data ? `${frontier.data.eligibleCount}/${frontier.data.target}` : "…"} /><Metric label="Wake budget" value={scope.wakeBudget} /><Metric label="Max lines" value={scope.maxEntryLines} /></CardContent></Card>;
}

function Metric({ label, value }: { label: string; value: string | number }) { return <div><div className="text-xs uppercase text-app-muted-foreground">{label}</div><div className="text-xl font-semibold">{value}</div></div>; }

export function ScopeDetailPage() {
  const { scope: encoded = "" } = useParams();
  const scope = encoded.startsWith(":") ? "" : decodeURIComponent(encoded);
  const entries = useAsync(() => scope ? post<{ entries: Entry[] }>("/vrooli.source_ledger.v1.journal.JournalService/ListEntries", { scope, limit: 500 }) : Promise.resolve({ entries: [] }), [scope]);
  const frontier = useAsync(() => scope ? post<Frontier>("/vrooli.source_ledger.v1.forest.ForestService/GetFrontier", { scope, limit: 50 }) : Promise.resolve({ eligibleCount: 0, target: 0, nodes: [] }), [scope]);
  const facets = useAsync(() => scope ? post<{ facets: Facet[] }>("/vrooli.source_ledger.v1.facets.FacetsService/ListFacets", { scope }) : Promise.resolve({ facets: [] }), [scope]);
  if (!scope) return <UnavailableScopePage />;
  return <section className="flex min-w-0 flex-col gap-5" aria-labelledby="scope-heading"><div className="flex min-w-0 flex-wrap items-center justify-between gap-3"><div className="min-w-0"><h2 id="scope-heading" className="break-words text-2xl font-semibold">{scope}</h2><p className="text-app-muted-foreground">Journal timeline, frontier explorer, and facet review queue.</p></div><Link to={`/scopes/${encodeURIComponent(scope)}/vocabulary`}><Button variant="secondary">Edit vocabulary</Button></Link></div><div className="grid min-w-0 gap-4 lg:grid-cols-3"><Card className="min-w-0"><CardHeader><CardTitle>Journal timeline</CardTitle></CardHeader><CardContent><ul className="space-y-2">{entries.data?.entries.slice(-20).reverse().map((entry) => <li key={entry.id} className="min-w-0 border-b border-app-border pb-2"><div className="break-words text-xs text-app-muted-foreground">{entry.facetId} · {entry.kind}</div><p className="break-words line-clamp-3 text-sm">{entry.body}</p></li>)}</ul></CardContent></Card><Card className="min-w-0"><CardHeader><CardTitle>Frontier explorer</CardTitle></CardHeader><CardContent><p className="mb-3 text-sm text-app-muted-foreground">{frontier.data?.eligibleCount ?? "…"} eligible nodes</p><ul className="space-y-2">{frontier.data?.nodes.map((node) => <li key={node.id} className="break-words rounded-control border border-app-border p-2 text-sm">{node.facetId} · depth {node.depth}<div className="break-words text-xs text-app-muted-foreground">{node.entryId || node.id}</div></li>)}</ul></CardContent></Card><Card className="min-w-0"><CardHeader><CardTitle>Facet review queue</CardTitle></CardHeader><CardContent><ul className="space-y-2">{facets.data?.facets.map((facet) => <li key={facet.id} className="flex min-w-0 justify-between gap-2 text-sm"><span className="break-words">{facet.label || facet.id}</span><span className="break-words text-right text-app-muted-foreground">{facet.retentionPolicy || "retain"}</span></li>)}</ul></CardContent></Card></div></section>;
}

export function VocabularyPage() {
  const { scope: encoded = "" } = useParams();
  const scope = encoded.startsWith(":") ? "" : decodeURIComponent(encoded);
  const facets = useAsync(() => scope ? post<{ facets: Facet[] }>("/vrooli.source_ledger.v1.facets.FacetsService/ListFacets", { scope }) : Promise.resolve({ facets: [] }), [scope]);
  const [saved, setSaved] = useState(false);
  if (!scope) return <UnavailableScopePage />;
  return <section className="flex min-w-0 flex-col gap-5" aria-labelledby="vocabulary-heading"><div className="min-w-0"><h2 id="vocabulary-heading" className="break-words text-2xl font-semibold">Vocabulary · {scope}</h2><p className="text-app-muted-foreground">Review facet retention and residency policy for this corpus.</p></div><Card className="min-w-0"><CardContent className="space-y-3">{facets.data?.facets.map((facet) => <label key={facet.id} className="grid min-w-0 gap-1"><span className="break-words text-sm font-medium">{facet.label || facet.id}</span><input defaultValue={facet.retentionPolicy || "retain"} aria-label={`${facet.id} retention policy`} className="min-w-0 w-full rounded-control border border-app-border bg-app-background px-3 py-2" /></label>)}<Button onClick={() => setSaved(true)}>{saved ? "Draft saved" : "Save vocabulary draft"}</Button></CardContent></Card></section>;
}

function UnavailableScopePage() {
  return <section className="flex min-w-0 flex-col gap-4" aria-labelledby="scope-unavailable-heading"><h2 id="scope-unavailable-heading" className="text-2xl font-semibold">Scope unavailable</h2><p className="text-app-muted-foreground">Choose a registered scope from the ledger dashboard before opening its detail or vocabulary view.</p><Link to="/"><Button variant="secondary">Back to ledger</Button></Link></section>;
}

export function SearchPage() {
  const [params] = useSearchParams();
  const [query, setQuery] = useState(params.get("q") || "");
  const scopes = useAsync(() => post<{ scopes: Scope[] }>("/vrooli.source_ledger.v1.scopes.ScopesService/ListScopes", {}), []);
  const results = useAsync(async () => { if (!query.trim() || !scopes.data) return [] as (RecallHit & { scope: string })[]; const grouped = await Promise.all(scopes.data.scopes.map(async (scope) => { const response = await post<{ hits: RecallHit[] }>("/vrooli.source_ledger.v1.recall.RecallService/Recall", { scope: scope.id, query, limit: 10 }); return response.hits.map((hit) => ({ ...hit, scope: scope.id })); })); return grouped.flat(); }, [query, scopes.data]);
  return <section className="flex min-w-0 flex-col gap-5" aria-labelledby="search-heading"><div className="min-w-0"><h2 id="search-heading" className="text-2xl font-semibold">Cross-scope search</h2><p className="text-app-muted-foreground">Results are labelled with the corpus that owns each entry.</p></div><div className="flex min-w-0 flex-wrap gap-2"><input aria-label="Search every scope" className="min-w-0 flex-1 rounded-control border border-app-border bg-app-background px-3 py-2" value={query} onChange={(e) => setQuery(e.target.value)} /><Button onClick={() => setQuery(query.trim())}>Search</Button></div><Card className="min-w-0"><CardContent><ul className="space-y-3">{results.data?.map((hit) => <li key={`${hit.scope}:${hit.entryId}`} className="min-w-0 border-b border-app-border pb-3"><div className="break-words text-xs font-semibold uppercase text-app-muted-foreground">{hit.scope} · {hit.facetId}</div><p className="break-words">{hit.text}</p><div className="text-xs text-app-muted-foreground">score {hit.score.toFixed(3)}</div></li>)}</ul></CardContent></Card></section>;
}
