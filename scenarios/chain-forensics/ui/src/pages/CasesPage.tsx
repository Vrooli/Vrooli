import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { ExperienceSurface, type ExperienceSurfaceState } from "@vrooli/react-component-library/ExperienceSurface/1";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { casesClient } from "../api/cases";
import { selectors } from "../consts/selectors";

export function CasesPage() {
  const [selected, setSelected] = useState("");
  const qc = useQueryClient();
  const list = useQuery({ queryKey:["forensic-cases"], queryFn:()=>casesClient.listCases({limit:100}) });
  const detail = useQuery({ queryKey:["forensic-case",selected], queryFn:()=>casesClient.getCase({id:selected}), enabled:Boolean(selected) });
  const labels = useQuery({ queryKey:["forensic-labels",selected], queryFn:()=>casesClient.listAnalystLabels({caseId:selected}), enabled:Boolean(selected) });
  const packet = useMutation({ mutationFn:()=>casesClient.exportPrivatePacket({id:selected}), onSuccess:(result)=>{
    const blob = new Blob([new TextDecoder().decode(result.packetJson)],{type:"application/json"}); const url=URL.createObjectURL(blob); const a=document.createElement("a"); a.href=url; a.download=`${selected}-private-packet.json`; a.click(); URL.revokeObjectURL(url);
  }});
  const importCase = useMutation({ mutationFn:async(file:File)=>{
    const input=JSON.parse(await file.text());
    const normalize=(v:unknown):unknown=>{if(Array.isArray(v))return v.map(normalize);if(v&&typeof v==="object")return Object.fromEntries(Object.entries(v).map(([k,x])=>[k,typeof x==="string"&&k==="chain"?`CHAIN_${x.toUpperCase()}`:typeof x==="string"&&k==="kind"?`EVIDENCE_KIND_${x.toUpperCase()}`:typeof x==="string"&&k==="grade"?`EVIDENCE_GRADE_${x.toUpperCase()}`:normalize(x)]));return v;};
    const normalized=normalize(input) as Record<string,unknown>;
    const response=await casesClient.importCase({case:normalized as never}); return response.case?.id;
  },onSuccess:(id)=>{void qc.invalidateQueries({queryKey:["forensic-cases"]});if(id)setSelected(id);}});
  const state:ExperienceSurfaceState=list.isLoading?"loading":list.error?"error":list.data?.cases.length===0?"empty":"ready";
  return <section className="flex flex-col gap-space-md" data-testid={selectors.pages.cases}>
    <PageHeader headingId="forensic-cases-heading" title="Chain forensics" description="Review sourced chain facts, preserve attribution gaps, and export a private packet locally." />
    <Card><CardHeader className="flex-row items-center justify-between gap-3"><CardTitle as="h2">Case file</CardTitle><label className="text-sm">Import evidence JSON<input className="ml-2 min-h-11 rounded-md border border-app-border px-3 py-2" type="file" accept="application/json,.json" onChange={(e)=>{const file=e.target.files?.[0];if(file)importCase.mutate(file);}} /></label></CardHeader>
      <CardContent><ExperienceSurface surfaceId="forensic-cases-list" state={state} statusMessage={list.error instanceof Error?list.error.message:undefined}>
        {list.data?.cases.length===0?<EmptyState title="No cases yet" />:<div className="grid gap-2">{list.data?.cases.map((item)=><button key={item.id} className={`rounded-md border p-3 text-left ${selected===item.id?"border-primary":"border-app-border"}`} onClick={()=>setSelected(item.id)}><span className="block font-medium">{item.title}</span><span className="text-sm text-app-muted-foreground">{item.id} · {item.sourceCount} sources · {item.occurrenceCount} occurrences · {item.edgeCount} flow edges · {item.private?"private":"not private"}</span></button>)}</div>}
      </ExperienceSurface></CardContent></Card>
    {detail.data?.case&&<div className="grid gap-space-md xl:grid-cols-2">
      <Card><CardHeader className="flex-row items-center justify-between"><CardTitle as="h2">{detail.data.case.title}</CardTitle><Button size="sm" disabled={!detail.data.case.private||packet.isPending} onClick={()=>packet.mutate()}>Export private packet</Button></CardHeader><CardContent className="space-y-4">
        <p>{detail.data.case.summary}</p><div><h3 className="font-semibold">Sources</h3><ul className="list-disc pl-5">{detail.data.case.sources.map(s=><li key={s.id}><a href={s.url} target="_blank" rel="noreferrer" className="underline">{s.name}</a> · {s.retrievedAt} · SHA-256 {s.sha256.slice(0,12)}…</li>)}</ul></div>
        <div><h3 className="font-semibold">Open gaps and counterexamples</h3><ul className="list-disc pl-5">{[...detail.data.case.gaps,...detail.data.case.counterexamples,...detail.data.case.sourceDisagreements].map((g,i)=><li key={i}>{g}</li>)}</ul></div>
        <div><h3 className="font-semibold">Separate analyst labels</h3>{labels.data?.labels.length? <ul className="list-disc pl-5">{labels.data.labels.map(label=><li key={label.id}><strong>{label.label}</strong> for {label.subject} · {label.grade}: {label.counterExplanation} Gap: {label.unresolvedGap}</li>)}</ul>:<p className="text-sm text-app-muted-foreground">No analyst labels stored for this case.</p>}</div>
        <p className="text-sm text-app-muted-foreground">Packets remain on this machine. This interface has no publish, tip-submission, accusation, or contact action.</p>
      </CardContent></Card>
      <Card><CardHeader><CardTitle as="h2">Observed occurrences and flow</CardTitle></CardHeader><CardContent className="max-h-[38rem] space-y-3 overflow-auto">
        <h3 className="font-semibold">Chain observations</h3>{detail.data.case.occurrences.map(o=><div key={o.id} className="rounded-md border p-3"><div className="font-mono text-xs">{o.chain} · {o.transactionId} · event {o.eventIndex}</div><div>{o.from||"?"} → {o.to||"?"} · {o.atomicQuantity} atomic units of {o.asset} · scale {o.scale}</div><div className="text-xs text-app-muted-foreground">{o.kind} · source {o.sourceId}</div></div>)}
        <h3 className="pt-3 font-semibold">Supported links</h3>{detail.data.case.edges.map(edge=>{const from=detail.data.case!.occurrences.find(o=>o.id===edge.fromOccurrence);const to=detail.data.case!.occurrences.find(o=>o.id===edge.toOccurrence);return <div key={edge.id} className="rounded-md border p-3"><div className="font-mono text-xs">{from?.from||from?.to||"?"} → {to?.to||to?.from||"?"}</div><div>{from?.transactionId} → {to?.transactionId}</div><div className="text-sm">{edge.grade}: {edge.rule}</div>{edge.caveats.map((c,i)=><p key={i} className="text-sm text-app-muted-foreground">Caveat: {c}</p>)}</div>})}
      </CardContent></Card>
    </div>}
    {(importCase.error||packet.error||detail.error)&&<p role="alert" className="text-app-danger">{(importCase.error||packet.error||detail.error) instanceof Error?(importCase.error||packet.error||detail.error)?.message:"Request failed"}</p>}
  </section>;
}
