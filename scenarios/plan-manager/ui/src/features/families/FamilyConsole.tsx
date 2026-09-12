import { useEffect, useState } from "react";
import { fromJson } from "@bufbuild/protobuf";
import {
  FamilyEdgeSchema,
  ReviewDecision,
  type FamilyEdge,
  type GetFrontierResponse,
  type PlanFamily,
} from "@vrooli/proto-types/plan-manager/v1/families/families_pb";

import { createFamily, getFamily, getFrontier, listFamilies, proposeGraph, reviewGraph } from "../../api/families";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Textarea } from "../../components/ui/textarea";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { errorMessage } from "../../lib/errorMessage";
import { useTranslation } from "../../i18n";
import { InvestigationIncidentHistory } from "../investigation/InvestigationIncidentHistory";

const enumLabel = (value: number, prefix: string) => `${prefix} ${value}`;

export function FamilyConsole() {
  const { t } = useTranslation();
  const [families, setFamilies] = useState<PlanFamily[]>([]);
  const [family, setFamily] = useState<PlanFamily | null>(null);
  const [frontier, setFrontier] = useState<GetFrontierResponse | null>(null);
  const [slug, setSlug] = useState("");
  const [outcome, setOutcome] = useState("");
  const [context, setContext] = useState("");
  const [reviewer, setReviewer] = useState("");
  const [rationale, setRationale] = useState("");
  const [corrections, setCorrections] = useState("[]");
  const [error, setError] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);

  const refresh = async (id: string) => {
    const [nextFamily, nextFrontier] = await Promise.all([getFamily(id), getFrontier(id)]);
    setFamily(nextFamily);
    setFrontier(nextFrontier);
  };
  useEffect(() => { void listFamilies().then(setFamilies).catch(setError); }, []);
  const run = (action: () => Promise<void>) => {
    setBusy(true); setError(null);
    void action().catch(setError).finally(() => setBusy(false));
  };
  const parseCorrections = (): FamilyEdge[] => {
    const raw = JSON.parse(corrections) as unknown;
    if (!Array.isArray(raw)) throw new Error(t(strings.pages.families.correctionInvalid));
    return raw.map((value) => fromJson(FamilyEdgeSchema, value));
  };
  const review = (decision: ReviewDecision) => run(async () => {
    if (!family) return;
    const correctedEdges = decision === ReviewDecision.CORRECTED ? parseCorrections() : [];
    const next = await reviewGraph({ family, decision, reviewer, rationale, correctedEdges });
    await refresh(next.familyId);
  });

  return <div className="flex flex-col gap-6">
    <form data-testid={selectors.families.createForm} className="grid gap-3 rounded-control border border-app-border bg-app-surface p-4 md:grid-cols-2" onSubmit={(event) => {
      event.preventDefault(); run(async () => { const created = await createFamily({ slug, outcome, sharedContext: context, maximumParallelPlans: 2 }); setFamilies((current) => [...current, created]); await refresh(created.familyId); });
    }}>
      <label className="text-sm">{t(strings.pages.families.slug)}<Input value={slug} onChange={(e) => setSlug(e.target.value)} required /></label>
      <label className="text-sm">{t(strings.pages.families.outcome)}<Input value={outcome} onChange={(e) => setOutcome(e.target.value)} required /></label>
      <label className="text-sm md:col-span-2">{t(strings.pages.families.sharedContext)}<Textarea value={context} onChange={(e) => setContext(e.target.value)} /></label>
      <Button disabled={busy} type="submit">{t(strings.pages.families.create)}</Button>
    </form>

    <label className="text-sm">{t(strings.pages.families.select)}
      <select data-testid={selectors.families.select} className="ml-2 rounded-control border border-app-border bg-app-surface px-3 py-2" value={family?.familyId ?? ""} onChange={(e) => run(() => refresh(e.target.value))}>
        <option value="">{t(strings.pages.families.selectPlaceholder)}</option>
        {families.map((item) => <option key={item.familyId} value={item.familyId}>{item.slug}</option>)}
      </select>
    </label>
    {error ? <p role="alert" className="text-sm text-app-danger">{errorMessage(error, t)}</p> : null}

    {family ? <section data-testid={selectors.families.detail} aria-labelledby="family-detail-heading" className="flex flex-col gap-5">
      <header><h3 id="family-detail-heading" className="text-xl font-semibold">{family.slug}</h3><p>{family.outcome}</p><p className="text-sm text-app-muted-foreground">{family.sharedContext}</p></header>
      <p data-testid={selectors.families.launchState} className={frontier?.launchable ? "text-app-success" : "text-app-warning"}>{frontier?.launchable ? t(strings.pages.families.launchable) : t(strings.pages.families.notLaunchable)}</p>
      {frontier?.diagnostics.map((diagnostic) => <p key={diagnostic} className="text-sm text-app-warning">{diagnostic}</p>)}

      <div className="overflow-x-auto"><table data-testid={selectors.families.membersTable} className="w-full text-left text-sm"><caption className="mb-2 text-left font-semibold">{t(strings.pages.families.members)}</caption><thead><tr><th>{t(strings.pages.families.plan)}</th><th>{t(strings.pages.families.role)}</th><th>{t(strings.pages.families.state)}</th><th>{t(strings.pages.families.activity)}</th></tr></thead><tbody>{family.members.map((member) => <tr key={member.planId}><td>{member.planId}</td><td>{enumLabel(member.role,"role")}</td><td>{enumLabel(member.state,"state")}</td><td>{member.detail || member.executionId}</td></tr>)}</tbody></table></div>
      <div className="overflow-x-auto"><table data-testid={selectors.families.claimsTable} className="w-full text-left text-sm"><caption className="mb-2 text-left font-semibold">{t(strings.pages.families.claims)}</caption><thead><tr><th>{t(strings.pages.families.plan)}</th><th>{t(strings.pages.families.resource)}</th><th>{t(strings.pages.families.access)}</th><th>{t(strings.pages.families.evidence)}</th></tr></thead><tbody>{family.claims.map((claim) => <tr key={claim.claimId}><td>{claim.planId}</td><td><code>{claim.resource}</code></td><td>{enumLabel(claim.access,"access")}</td><td>{claim.resolved ? claim.source : t(strings.pages.families.unknownInteraction)}</td></tr>)}</tbody></table></div>
      <section><h4 className="font-semibold">{t(strings.pages.families.proposal)} {family.graph ? family.graph.revision.toString() : "—"}</h4>{family.graph?.edges.map((edge, index) => <article data-testid={selectors.families.edge({ index })} key={`${edge.fromPlanId}-${edge.toPlanId}-${index}`} className="my-2 rounded-control border border-app-border p-3"><p><code>{edge.fromPlanId}</code> → <code>{edge.toPlanId}</code></p><p className="text-sm">{edge.reason || t(strings.pages.families.noReason)}</p><p className="text-xs text-app-muted-foreground">{enumLabel(edge.provenance,"provenance")}; {edge.claimIds.join(", ")}</p></article>)}{!family.graph ? <Button disabled={busy} onClick={() => run(async () => { const next=await proposeGraph(family); await refresh(next.familyId); })}>{t(strings.pages.families.propose)}</Button> : null}</section>
      <section data-testid={selectors.families.review}><h4 className="font-semibold">{t(strings.pages.families.reviewed)} {family.review?.graphRevision.toString() ?? "—"}</h4>{family.review ? <p>{enumLabel(family.review.decision,"decision")}: {family.review.rationale}</p> : <div className="grid gap-2"><label>{t(strings.pages.families.reviewer)}<Input value={reviewer} onChange={(e)=>setReviewer(e.target.value)} /></label><label>{t(strings.pages.families.rationale)}<Textarea value={rationale} onChange={(e)=>setRationale(e.target.value)} /></label><label>{t(strings.pages.families.corrections)}<Textarea value={corrections} onChange={(e)=>setCorrections(e.target.value)} /></label><div className="flex flex-wrap gap-2"><Button disabled={busy || !family.graph || !reviewer || !rationale} onClick={()=>review(ReviewDecision.APPROVED)}>{t(strings.pages.families.approve)}</Button><Button disabled={busy || !family.graph || !reviewer || !rationale} variant="outline" onClick={()=>review(ReviewDecision.CORRECTED)}>{t(strings.pages.families.correct)}</Button><Button disabled={busy || !family.graph || !reviewer || !rationale} variant="outline" onClick={()=>review(ReviewDecision.REJECTED)}>{t(strings.pages.families.reject)}</Button></div></div>}</section>
      <ol data-testid={selectors.families.frontier} className="list-decimal pl-5">{frontier?.batches.map((batch) => <li key={batch.ordinal}>{batch.planIds.join(", ")}</li>)}</ol>
      <InvestigationIncidentHistory familyId={family.familyId} />
    </section> : null}
  </div>;
}
