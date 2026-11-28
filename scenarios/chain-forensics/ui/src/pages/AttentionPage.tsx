import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { ArrowDownToLine, LockKeyhole, ShieldCheck } from "lucide-react";
import { timestampDate } from "@bufbuild/protobuf/wkt";

import { casesClient } from "../api/cases";
import { selectors } from "../consts/selectors";

type PrivatePacketCase = {
  id: string;
  title: string;
  summary: string;
  private: boolean;
  edges?: Array<{ id: string; grade: string; rule: string; caveats?: string[] }>;
  gaps?: string[];
  counterexamples?: string[];
  source_disagreements?: string[];
};
type PrivatePacket = { schema_version: string; case: PrivatePacketCase };

function parsePacket(bytes: Uint8Array): PrivatePacket | undefined {
  try {
    const parsed = JSON.parse(new TextDecoder().decode(bytes)) as PrivatePacket;
    return parsed?.case?.private ? parsed : undefined;
  } catch {
    return undefined;
  }
}

function gradeLabel(value: string): string {
  return value.replace(/^EVIDENCE_GRADE_/, "").replace(/_/g, " ").toLowerCase();
}

export function AttentionPage() {
  const [selectedCaseID, setSelectedCaseID] = useState("");
  const queryClient = useQueryClient();
  const cases = useQuery({ queryKey: ["forensic-cases"], queryFn: () => casesClient.listCases({ limit: 100 }) });
  const deliveries = useQuery({ queryKey: ["attention-deliveries"], queryFn: () => casesClient.listAttentionDeliveries({ limit: 100 }) });
  const deliveryByCase = useMemo(() => new Map((deliveries.data?.deliveries ?? []).map((item) => [item.caseId, item])), [deliveries.data]);

  const deliver = useMutation({
    mutationFn: async () => {
      const packet = await casesClient.exportPrivatePacket({ id: selectedCaseID });
      return casesClient.deliverPrivatePacketToAttention({ caseId: selectedCaseID, packetSha256: packet.sha256 });
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["attention-deliveries"] });
    },
  });

  const privateCases = (cases.data?.cases ?? []).filter((item) => item.private);
  const selected = privateCases.find((item) => item.id === selectedCaseID);

  return (
    <section data-testid={selectors.pages.attention} aria-labelledby="attention-heading" className="flex flex-col gap-space-md">
      <PageHeader headingId="attention-heading" title="Local attention" description="Review private case packets that were delivered to this machine for follow-up." />

      <Card className="border-primary/30">
        <CardContent className="flex flex-col gap-3 pt-5 sm:flex-row sm:items-start">
          <LockKeyhole className="mt-0.5 shrink-0 text-primary" aria-hidden="true" />
          <div className="min-w-0">
            <h2 className="font-semibold">Private · local-only handoff</h2>
            <p className="text-sm text-app-muted-foreground">Delivery stores a packet and receipt in Chain Forensics. It does not publish, notify, or contact anyone.</p>
          </div>
          <StatusBadge tone="info" className="sm:ml-auto">This machine</StatusBadge>
        </CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle as="h2">Deliver a current private case packet</CardTitle></CardHeader>
        <CardContent className="flex flex-col gap-3 sm:flex-row sm:items-end">
          <label className="flex min-w-0 flex-1 flex-col gap-1 text-sm font-medium" htmlFor="attention-case">Private case
            <select id="attention-case" value={selectedCaseID} onChange={(event) => setSelectedCaseID(event.target.value)} className="min-h-11 rounded-md border border-app-border bg-app-surface px-3 py-2">
              <option value="">Choose a private case</option>
              {privateCases.map((item) => <option key={item.id} value={item.id}>{item.title} · {item.id}</option>)}
            </select>
          </label>
          <Button disabled={!selected || deliver.isPending} onClick={() => deliver.mutate()} data-testid={selectors.pages.attentionDeliverButton}>
            <ArrowDownToLine aria-hidden="true" /> {deliveryByCase.has(selectedCaseID) ? "Refresh local packet" : "Deliver to local attention"}
          </Button>
        </CardContent>
        <CardContent className="pt-0 text-sm text-app-muted-foreground">The current case version is packetized and bound to its SHA-256 hash. A later case revision needs a new delivery.</CardContent>
      </Card>

      {(deliver.error || deliveries.error || cases.error) && <p role="alert" className="text-app-danger">{(deliver.error || deliveries.error || cases.error) instanceof Error ? (deliver.error || deliveries.error || cases.error)?.message : "Unable to load local attention records."}</p>}

      <section aria-labelledby="delivered-heading" className="space-y-3">
        <div className="flex items-center gap-2"><ShieldCheck className="text-primary" aria-hidden="true" /><h2 id="delivered-heading" className="text-xl font-semibold">Delivered packets</h2></div>
        {deliveries.isLoading && <p role="status">Loading local delivery records…</p>}
        {!deliveries.isLoading && !deliveries.error && deliveries.data?.deliveries.length === 0 && <EmptyState title="No packets delivered" description="Choose an existing private case above to place its current packet in this local review queue." />}
        <div className="grid gap-3">
          {deliveries.data?.deliveries.map((receipt) => {
            const packet = parsePacket(receipt.packetJson);
            if (!packet) return <Card key={`${receipt.caseId}-${receipt.packetSha256}`}><CardContent className="pt-5"><p className="font-semibold">Unreadable local packet</p><p className="break-all font-mono text-xs">{receipt.caseId} · {receipt.packetSha256}</p></CardContent></Card>;
            const c = packet.case;
            const unresolved = [...(c.gaps ?? []), ...(c.counterexamples ?? []), ...(c.source_disagreements ?? [])];
            return (
              <Card key={`${receipt.caseId}-${receipt.packetSha256}`} data-testid={selectors.pages.attentionDelivery}>
                <CardHeader className="gap-2 sm:flex-row sm:items-start sm:justify-between">
                  <div className="min-w-0"><CardTitle as="h3">{c.title}</CardTitle><p className="mt-1 text-sm text-app-muted-foreground">{c.id} · Delivered {receipt.deliveredAt ? timestampDate(receipt.deliveredAt).toLocaleString() : "time unavailable"}</p></div>
                  <StatusBadge tone="success">Private local packet</StatusBadge>
                </CardHeader>
                <CardContent className="space-y-4">
                  <p className="break-words">{c.summary}</p>
                  <div className="flex flex-wrap gap-2 text-sm"><span>{c.edges?.length ?? 0} graded links</span><span aria-hidden="true">·</span><span>{unresolved.length} unresolved gaps / counterpoints</span></div>
                  <section aria-label="Evidence grades" className="space-y-2">
                    <h4 className="font-semibold">Evidence grades</h4>
                    {c.edges?.length ? <ul className="space-y-2">{c.edges.map((edge) => <li key={edge.id} className="rounded-md border border-app-border p-3"><div className="flex flex-wrap items-center gap-2"><StatusBadge tone={["unresolved", "hypothesis"].includes(edge.grade.toLowerCase().replace(/^evidence_grade_/, "")) ? "warning" : "neutral"}>{gradeLabel(edge.grade)}</StatusBadge><span>{edge.rule}</span></div>{edge.caveats?.map((caveat) => <p key={caveat} className="mt-1 break-words text-sm text-app-muted-foreground">Caveat: {caveat}</p>)}</li>)}</ul> : <p className="text-sm text-app-muted-foreground">No flow links are included in this packet.</p>}
                  </section>
                  <section aria-label="Unresolved gaps and counterpoints" className="space-y-2">
                    <h4 className="font-semibold">Unresolved gaps and counterpoints</h4>
                    {unresolved.length ? <ul className="list-disc space-y-1 pl-5">{unresolved.map((gap, index) => <li key={`${index}-${gap}`} className="break-words">{gap}</li>)}</ul> : <p className="text-sm text-app-muted-foreground">No unresolved gaps are recorded in this packet.</p>}
                  </section>
                  <p className="break-all border-t border-app-border pt-3 font-mono text-xs text-app-muted-foreground">Packet SHA-256 · {receipt.packetSha256}</p>
                </CardContent>
              </Card>
            );
          })}
        </div>
      </section>
    </section>
  );
}
