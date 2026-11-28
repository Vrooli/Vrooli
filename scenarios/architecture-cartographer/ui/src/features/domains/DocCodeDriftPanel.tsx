import { FormEvent, useState } from "react";
import { useMutation } from "@tanstack/react-query";

import { searchDocCodeDrift } from "../../api/docCodeDrift";

export function DocCodeDriftPanel({scenario}:{scenario:string}) {
  const [query,setQuery]=useState("api domain source path");
  const search=useMutation({mutationFn:()=>searchDocCodeDrift(query.trim(),`scenario:${scenario}`)});
  function submit(event:FormEvent<HTMLFormElement>){event.preventDefault();if(query.trim())search.mutate()}
  const response=search.data;
  return <section className="rounded-panel border border-app-border bg-app-surface p-4" aria-labelledby="doc-code-drift-title">
    <h3 id="doc-code-drift-title" className="text-lg font-semibold">Documentation vs API source paths</h3>
    <p className="mt-1 text-sm text-app-muted-foreground">Compares curated DOMAINS.md Source Paths with bounded Go source, then links only proven literal HTTP route → handler → direct internal call chains. Static links do not prove semantic conformance, generated Connect routing, or runtime reachability.</p>
    <form className="mt-3 flex gap-2" onSubmit={submit}><label className="sr-only" htmlFor="doc-code-drift-query">Search docs and source</label><input id="doc-code-drift-query" value={query} onChange={e=>setQuery(e.target.value)} className="min-w-0 flex-1 rounded-md border border-app-border bg-app-background px-3 py-2"/><button type="submit" disabled={search.isPending} className="rounded-md bg-app-primary px-3 py-2 text-sm font-semibold text-app-primary-foreground">{search.isPending?"Checking…":"Compare paths"}</button></form>
    {search.error&&<p role="alert" className="mt-3 text-sm text-app-danger">{search.error.message}</p>}
    {response&&<div className="mt-3 space-y-3"><p className="text-xs text-app-muted-foreground">{response.scope} · authority {response.authority}</p>
      {response.results.length===0&&<p>No source-path candidate matched this query.</p>}
      {response.results.map(row=><article key={row.id} className="rounded-md border border-app-border p-3"><h4 className="font-semibold">{row.title} · {row.classification} · {row.status}</h4><p className="text-sm">{row.claim}</p><p className="text-xs">Docs {row.documentation.path}:{row.documentation.line} · sha256:{row.documentation.sha256}</p><p className="mt-1 text-xs font-semibold">Literal API chain: {row.api_status||"UNKNOWN"}</p>
        {row.code.map((source,i)=><p key={`${source.path}:${source.line}:${i}`} className="text-xs">Code {source.path}:{source.line} · sha256:{source.sha256} · {source.basis}</p>)}
        {(row.api??[]).map((source,i)=><p key={`api:${source.path}:${source.line}:${i}`} className="text-xs">API {source.path}:{source.line} · sha256:{source.sha256} · {source.basis}</p>)}
        <ul className="mt-1 list-disc pl-5 text-xs text-app-muted-foreground">{row.limits.map(limit=><li key={limit}>{limit}</li>)}</ul>
      </article>)}
      {response.degradations.map(item=><p role="status" key={item} className="text-sm text-app-muted-foreground">{item}</p>)}
    </div>}
  </section>;
}
