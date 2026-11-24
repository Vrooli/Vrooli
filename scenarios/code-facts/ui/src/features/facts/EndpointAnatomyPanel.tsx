import { FormEvent, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { searchEndpointAnatomy } from "../../api/endpointAnatomy";

export function EndpointAnatomyPanel() {
  const [scenario,setScenario]=useState("code-facts");
  const [query,setQuery]=useState("persistence patterns");
  const search=useMutation({mutationFn:()=>searchEndpointAnatomy(query.trim(),scenario.trim())});
  function submit(event:FormEvent<HTMLFormElement>){event.preventDefault();if(query.trim()&&scenario.trim())search.mutate()}
  const response=search.data;
  return <section className="rounded-panel border border-app-border bg-app-surface p-4" aria-labelledby="endpoint-anatomy-title">
    <h2 id="endpoint-anatomy-title" className="text-xl font-semibold">API surface anatomy</h2>
    <p className="mt-1 text-sm text-app-muted-foreground">Joins HTTP JSON declarations and Connect procedure declarations to bounded local registration evidence. Connect source chains do not prove that the module is mounted at runtime or that a request succeeds.</p>
    <form className="mt-4 grid gap-3 md:grid-cols-[220px_1fr_auto]" onSubmit={submit}>
      <label className="flex flex-col gap-1 text-sm">Scenario slug<input aria-label="Endpoint scenario" value={scenario} onChange={e=>setScenario(e.target.value)} className="rounded-md border border-app-border bg-app-background px-3 py-2" /></label>
      <label className="flex flex-col gap-1 text-sm">Endpoint query<input aria-label="Endpoint query" value={query} onChange={e=>setQuery(e.target.value)} className="rounded-md border border-app-border bg-app-background px-3 py-2" /></label>
      <button type="submit" disabled={search.isPending} className="rounded-md bg-app-primary px-4 py-2 text-sm font-semibold text-app-primary-foreground md:self-end">{search.isPending?"Searching…":"Search API surface"}</button>
    </form>
    {search.error&&<p role="alert" className="mt-3 text-sm text-app-danger">{search.error.message}</p>}
    {response&&<div className="mt-4 space-y-3"><p className="text-xs text-app-muted-foreground">{response.scope}</p>
      {response.results.length===0&&<p>No declared endpoint matched within this source scope.</p>}
      {response.results.map(row=><article key={row.id} className="rounded-md border border-app-border p-3"><h3 className="font-semibold">{row.title} · {row.state}</h3><p className="text-sm">{row.summary} · {row.transport === "connect" ? `Connect ${row.connect_service||"unknown"}.${row.connect_method||"unknown"}` : `HTTP JSON handler ${row.handler||"unknown"}`}</p>
        {row.transport === "connect" ? <><p className="mt-2 text-xs font-semibold">Connect source chain · runtime mount remains unverified</p>{row.connect_chain?.length ? <dl className="mt-1 grid gap-2 text-xs sm:grid-cols-2">{row.connect_chain.map((source,i)=><div key={`${source.path}:${source.line}:${i}`} className="rounded border border-app-border p-2"><dt className="font-semibold">{source.basis}</dt><dd>{source.path}:{source.line}</dd><dd className="break-all">sha256:{source.sha256}</dd></div>)}</dl> : <p className="text-sm">No supported generated route provenance was established.</p>}</> : row.state === "ROUTE_AMBIGUOUS" ? <><p className="mt-2 text-sm">Multiple source registrations match this route ({row.route_candidate_total ?? row.route_candidates?.length ?? 0} total). No handler was selected; runtime dispatch precedence is unknown. {row.route_candidates_omitted ? `${row.route_candidates_omitted} candidate branches omitted by the display cap.` : ""}</p><div className="mt-2 grid gap-2 sm:grid-cols-2">{row.route_candidates?.map((candidate,i)=><article key={`${candidate.registration.path}:${candidate.registration.line}:${i}`} className="rounded border border-app-border p-2 text-xs"><h4 className="font-semibold">Candidate handler {candidate.handler}</h4><p>Registration · {candidate.registration.path}:{candidate.registration.line}</p><p className="break-all">sha256:{candidate.registration.sha256||"unknown"}</p><p>Definition · {candidate.handler_source.path||"unknown"}:{candidate.handler_source.line||"?"}</p><p className="break-all">sha256:{candidate.handler_source.sha256||"unknown"}</p></article>)}</div></> : <dl className="mt-2 grid gap-2 text-xs sm:grid-cols-3">{[row.declaration,row.registration,row.handler_source].map((source,i)=><div key={i} className="rounded border border-app-border p-2"><dt className="font-semibold">{source.basis||"unavailable"}</dt><dd>{source.path||"unknown"}:{source.line||"?"}</dd><dd className="break-all">sha256:{source.sha256||"unknown"}</dd></div>)}</dl>}
        <ul className="mt-2 list-disc pl-5 text-xs text-app-muted-foreground">{row.limits.map(limit=><li key={limit}>{limit}</li>)}</ul>
      </article>)}
      {response.degradations.map(item=><p role="status" key={item} className="text-sm text-app-muted-foreground">{item}</p>)}
    </div>}
  </section>;
}
