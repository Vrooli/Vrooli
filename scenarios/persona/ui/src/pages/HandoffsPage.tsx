import { Link, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { completeHandoff, getHandoff, listAllHandoffs, listPersonas } from "../api/persona";
import { Button } from "@vrooli/react-component-library/Button/1.2.0";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { selectors } from "../consts/selectors";
import { HandoffState } from "@vrooli/proto-types/persona/v1/handoffs/handoffs_pb";

export function HandoffsPage() {
  const { handoffId } = useParams();
  return handoffId ? <HandoffDetail handoffId={handoffId} /> : <HandoffQueue />;
}

function HandoffQueue() {
  const personasQuery = useQuery({ queryKey: ["personas", "all"], queryFn: () => listPersonas(true) });
  const query = useQuery({ queryKey: ["handoffs", "all", personasQuery.data?.map((persona) => persona.id)], queryFn: () => listAllHandoffs(personasQuery.data ?? []), enabled: personasQuery.isSuccess });
  const handoffs = [...(query.data ?? [])].sort((a, b) => Number(b.createdAt?.seconds ?? 0) - Number(a.createdAt?.seconds ?? 0));
  return <section data-testid={selectors.pages.handoffs} aria-labelledby="handoffs-heading" className="flex flex-col gap-6"><div><p className="text-sm font-semibold uppercase tracking-[0.18em] text-app-primary">Human boundary</p><h2 id="handoffs-heading" className="mt-2 text-3xl font-semibold">Handoffs</h2><p className="mt-2 max-w-2xl text-app-muted-foreground">Every machine wall becomes a resumable checkpoint. The queue remains useful even when optional delivery is offline.</p></div><Card><CardHeader><CardTitle>Waiting and resolved work</CardTitle><CardDescription>{query.isPending ? "Loading queue…" : `${handoffs.length} handoff(s)`}</CardDescription></CardHeader><CardContent>{query.isError ? <p role="alert" className="text-app-danger">The queue is unavailable. No action was taken.</p> : !handoffs.length && !query.isPending ? <p className="text-sm text-app-muted-foreground">Nothing is waiting for a human.</p> : <div className="divide-y divide-app-border">{handoffs.map((handoff) => <Link key={handoff.id} to={`/handoffs/${handoff.id}`} className="flex flex-wrap items-center justify-between gap-4 py-4 first:pt-0 last:pb-0"><div><p className="font-semibold">{handoff.title || handoff.kind}</p><p className="mt-1 text-sm text-app-muted-foreground">{handoff.humanAction}</p></div><span className="rounded-full border border-app-border px-3 py-1 text-sm">{stateLabel(handoff.state)}</span></Link>)}</div>}</CardContent></Card></section>;
}

function HandoffDetail({ handoffId }: { handoffId: string }) {
  const queryClient = useQueryClient();
  const query = useQuery({ queryKey: ["handoff", handoffId], queryFn: () => getHandoff(handoffId) });
  const mutation = useMutation({ mutationFn: () => completeHandoff(handoffId), onSuccess: () => { void queryClient.invalidateQueries({ queryKey: ["handoff", handoffId] }); void queryClient.invalidateQueries({ queryKey: ["handoffs"] }); } });
  const handoff = query.data;
  if (query.isPending) return <p>Loading handoff…</p>;
  if (query.isError || !handoff) return <p role="alert">This handoff could not be loaded. <Link className="underline" to="/handoffs">Return to the queue.</Link></p>;
  return <section data-testid={selectors.pages.handoffDetail} aria-labelledby="handoff-detail-heading" className="flex flex-col gap-6"><div><Link to="/handoffs" className="text-sm font-medium text-app-primary underline-offset-4 hover:underline">← Handoff queue</Link><h2 id="handoff-detail-heading" className="mt-3 text-3xl font-semibold">{handoff.title || handoff.kind}</h2><p className="mt-2 max-w-2xl text-app-muted-foreground">{handoff.humanAction}</p></div><Card><CardHeader><CardTitle>Checkpoint</CardTitle><CardDescription>Review what is already complete before taking the human-only step.</CardDescription></CardHeader><CardContent className="space-y-5"><div className="flex flex-wrap items-center justify-between gap-4"><div><p className="text-xs uppercase tracking-wide text-app-muted-foreground">State</p><p className="mt-1 font-semibold">{stateLabel(handoff.state)}</p></div><div><p className="text-xs uppercase tracking-wide text-app-muted-foreground">Deadline</p><p className="mt-1 font-medium">{handoff.deadline ? new Date(Number(handoff.deadline.seconds) * 1000).toLocaleString() : "No deadline"}</p></div></div>{handoff.checkpoint?.completedFields.length ? <div><h3 className="font-semibold">Completed fields</h3><ul className="mt-2 divide-y divide-app-border rounded-panel border border-app-border">{handoff.checkpoint.completedFields.map((field) => <li key={field.name} className="flex justify-between gap-3 p-3 text-sm"><span>{field.name}</span><span className="text-app-muted-foreground">{field.value || "Complete"}</span></li>)}</ul></div> : null}{handoff.checkpoint?.requiredDocumentIds.length ? <div><h3 className="font-semibold">Required custody records</h3><p className="mt-1 text-sm text-app-muted-foreground">{handoff.checkpoint.requiredDocumentIds.length} document binding(s) are scoped to this handoff. Content is not displayed here.</p></div> : null}{handoff.state === HandoffState.AWAITING_HUMAN ? <Button data-testid="handoff-complete" onClick={() => mutation.mutate()} disabled={mutation.isPending}>{mutation.isPending ? "Completing…" : "Mark human step complete"}</Button> : <p className="rounded-panel border border-app-border bg-app-surface-muted p-3 text-sm">This handoff is terminal or already resumed. No further action is available.</p>}{mutation.isError ? <p role="alert" className="text-sm text-app-danger">Completion was refused. The durable state remains unchanged.</p> : null}</CardContent></Card></section>;
}

function stateLabel(state: HandoffState) { return ({ [HandoffState.OPEN]: "Open", [HandoffState.DELIVERED]: "Delivered", [HandoffState.AWAITING_HUMAN]: "Awaiting human", [HandoffState.COMPLETED]: "Completed", [HandoffState.EXPIRED]: "Expired", [HandoffState.CANCELLED]: "Cancelled", [HandoffState.RESUMED]: "Resumed" } as Record<number, string>)[state] ?? "Unknown"; }
