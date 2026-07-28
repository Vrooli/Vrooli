import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo, useState } from "react";

import { AnnotationAuthor, DispositionState } from "../../../../../../packages/proto/gen/typescript/signal-inbox/v1/triage/triage_pb";

import { retrievalClient } from "../../api/retrieval";
import { triageClient } from "../../api/triage";
import { Button } from "../../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../../components/ui/card";
import { EmptyState } from "../../components/ui/empty-state";
import { Textarea } from "../../components/ui/textarea";
import { useTriageKeys } from "../../hooks/useTriageKeys";
import { SignalClassificationControl } from "../categories/SignalClassificationControl";

const ambientKey = ["retrieval", "ambient"] as const;
const triageKey = (signalID: string) => ["triage", signalID] as const;

export function TriageQueue() {
  const client = useQueryClient();
  const [note, setNote] = useState("");
  const ambient = useQuery({ queryKey: ambientKey, queryFn: () => retrievalClient.ambient({ budget: 20 }) });
  const current = ambient.data?.results[0];
  const signalID = current?.signal?.id ?? "";
  const triage = useQuery({ queryKey: triageKey(signalID), queryFn: () => triageClient.getTriage({ signalId: signalID }), enabled: Boolean(signalID) });
  const refresh = () => {
    void client.invalidateQueries({ queryKey: ambientKey });
    if (signalID) void client.invalidateQueries({ queryKey: triageKey(signalID) });
  };
  const disposition = useMutation({ mutationFn: (state: DispositionState) => triageClient.setDisposition({ signalId: current?.signal?.id ?? "", state }), onSuccess: refresh });
  const annotation = useMutation({ mutationFn: () => triageClient.addAnnotation({ signalId: current?.signal?.id ?? "", author: AnnotationAuthor.OPERATOR, body: note }), onSuccess: () => { setNote(""); refresh(); } });
  const actions = useMemo(() => ({ accept: () => current && disposition.mutate(DispositionState.TRIAGED), drop: () => current && disposition.mutate(DispositionState.DROPPED), annotate: () => current && note.trim() && annotation.mutate() }), [annotation, current, disposition, note]);
  useTriageKeys(Boolean(current), actions);

  return <Card aria-label="Triage queue"><CardHeader><CardTitle>Triage queue</CardTitle></CardHeader><CardContent className="flex flex-col gap-3">
    {ambient.isLoading && <p>Loading unresolved signals…</p>}
    {ambient.error && <p className="text-app-danger">Could not load the triage queue.</p>}
    {!ambient.isLoading && !ambient.error && !current && <EmptyState title="No unresolved signals. The queue is clear." className="border-0 bg-transparent p-0" />}
    {current?.signal && <>
      <p className="text-sm text-app-muted-foreground">Source: {current.signal.sourceUrl || current.signal.sourceKind}</p>
      <section aria-label="Extracted content" className="rounded border border-app-border p-3"><h3 className="font-medium">Extracted content</h3><p>{current.signal.extractedContent || "No extraction yet; add a manual annotation."}</p></section>
      <section aria-label="Operator annotations" className="rounded border border-app-border p-3"><h3 className="font-medium">Operator annotations</h3>
        {triage.isLoading && <p className="text-sm text-app-muted-foreground">Loading annotations…</p>}
        {triage.error && <p className="text-sm text-app-danger">Could not load annotations.</p>}
        {!triage.isLoading && !triage.error && (triage.data?.triage?.annotations.length ?? 0) === 0 && <p className="text-sm text-app-muted-foreground">No annotations yet.</p>}
        {triage.data?.triage?.annotations.length ? <ul className="mt-2 space-y-1">{triage.data.triage.annotations.map((annotation) => <li key={annotation.id}><span className="font-medium">{annotation.author === 1 ? "Operator" : annotation.author === 2 ? "Agent" : "System"}:</span> {annotation.body}</li>)}</ul> : null}
      </section>
      {current.signal.needsAttention && <p className="text-app-danger">Needs attention: add the missing text as an annotation.</p>}
      <SignalClassificationControl signalID={current.signal.id} />
      <Textarea aria-label="Triage annotation" placeholder="Optional operator note" value={note} onChange={(event) => setNote(event.target.value)} />
      <div className="flex flex-wrap gap-2"><Button onClick={actions.accept} disabled={disposition.isPending}>Accept (A)</Button><Button variant="secondary" onClick={actions.annotate} disabled={!note.trim() || annotation.isPending}>Annotate (N)</Button><Button variant="secondary" onClick={actions.drop} disabled={disposition.isPending}>Drop from ambient (D)</Button></div>
      {(disposition.error || annotation.error) && <p className="text-app-danger">Triage update failed. The signal remains stored and searchable.</p>}
    </>}
  </CardContent></Card>;
}
