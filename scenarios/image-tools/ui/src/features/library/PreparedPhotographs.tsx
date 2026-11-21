import { useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { preparationClient } from "../../api/preparation";
import { blobUrl } from "../../api/client";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Textarea } from "../../components/ui/textarea";

const queryKey = ["prepared-photographs"] as const;

/** Durable source/clean-plate comparison. A review records an observed result;
 * it does not run models or turn a successful inference into automatic approval. */
export function PreparedPhotographs() {
  const { t } = useTranslation();
  const k = strings.preparation;
  const cache = useQueryClient();
  const [selected, setSelected] = useState("");
  const [reviewer, setReviewer] = useState("");
  const [rationale, setRationale] = useState("");
  const [evidence, setEvidence] = useState("");
  const [overscan, setOverscan] = useState(false);
  const reviewOperations = useRef(new Map<string, string>());
  const list = useQuery({ queryKey, queryFn: ({ signal }) => preparationClient.listPhotographs({ limit: 30 }, { signal }) });
  const current = list.data?.photographs.find((p) => p.id === selected);
  const review = useMutation({
    mutationFn: (decision: "accepted" | "rejected") => {
      const review = { decision, reviewedBy: reviewer.trim(), rationale: rationale.trim(), evidenceRefs: evidence.split("\n").map((line) => line.trim()).filter(Boolean) };
      const intent = JSON.stringify({ id: selected, review, overscanReviewed: overscan });
      let operationId = reviewOperations.current.get(intent);
      if (!operationId) {
        operationId = crypto.randomUUID();
        reviewOperations.current.set(intent, operationId);
      }
      return preparationClient.reviewPhotograph({ id: selected, review, overscanReviewed: overscan, operationId }).then((response) => ({ response, intent }));
    },
    onSuccess: async ({ response, intent }) => {
      reviewOperations.current.delete(intent);
      await cache.invalidateQueries({ queryKey });
      setSelected(response.photograph?.id ?? "");
      setRationale("");
      setEvidence("");
      setOverscan(false);
    },
  });
  const changeSelection = (id: string) => {
    setSelected(id); setRationale(""); setEvidence(""); setOverscan(false); review.reset();
  };
  const ready = Boolean(current && reviewer.trim() && rationale.trim() && evidence.trim()) && !review.isPending;

  return <section aria-labelledby="preparation-heading" className="flex flex-col gap-4">
    <div>
      <h2 id="preparation-heading" className="text-2xl font-semibold">{t(k.title)}</h2>
      <p className="mt-1 text-sm text-app-muted-foreground">{t(k.description)}</p>
    </div>
    {list.isLoading && <p role="status">{t(k.loading)}</p>}
    {list.error && <p role="alert">{errorMessage(list.error, t)}</p>}
    {list.data?.photographs.length === 0 && <p className="rounded-card border border-app-border p-6 text-app-muted-foreground">{t(k.empty)}</p>}
    <div className="flex flex-wrap gap-2" aria-label={t(k.choose)}>
      {list.data?.photographs.map((p) => <Button key={p.id} variant={p.id === selected ? "default" : "outline"} aria-pressed={p.id === selected} disabled={review.isPending} onClick={() => changeSelection(p.id)}>
        {p.label} · {t(k.status[p.manifest?.review?.decision === "accepted" ? "accepted" : p.manifest?.review?.decision === "rejected" ? "rejected" : "pending"])}
      </Button>)}
    </div>
    {current && <div className="rounded-card border border-app-border p-4 md:p-6">
      <div className="grid gap-4 md:grid-cols-2">
        <figure><figcaption className="mb-2 font-medium">{t(k.source)}</figcaption><img src={blobUrl(current.sourcePreviewRef)} alt={t(k.sourceAlt)} className="w-full rounded-control border border-app-border" /></figure>
        <figure><figcaption className="mb-2 font-medium">{t(k.cleanPlate)}</figcaption><img src={blobUrl(current.cleanPlatePreviewRef)} alt={t(k.cleanPlateAlt)} className="w-full rounded-control border border-app-border" /></figure>
      </div>
      <p className="mt-3 text-sm text-app-muted-foreground">{t(k.estimated)}</p>
      {current.repairAssessment && <aside aria-label={t(k.contextTitle)} className="mt-3 rounded-control border border-app-border bg-app-surface-muted p-3 text-sm">
        <p>{t(k.coverage, { percent: Math.round(current.repairAssessment.maskAreaFraction * 100) })}</p>
        <ul className="mt-1 list-disc ps-5">
          {current.repairAssessment.riskSignals.includes("large_mask_area") && <li>{t(k.largeMask)}</li>}
          {current.repairAssessment.riskSignals.includes("requested_context_clipped_by_source_boundary") && <li>{t(k.clippedContext)}</li>}
          {current.repairAssessment.riskSignals.includes("large_crop_downscaled_for_inference") && <li>{t(k.downscaled)}</li>}
        </ul>
      </aside>}
      {current.manifest?.review?.rationale && <blockquote className="mt-3 border-s-2 border-app-border ps-3">{current.manifest.review.rationale}<footer className="text-sm text-app-muted-foreground">{current.manifest.review.reviewedBy}</footer></blockquote>}
      <a href={blobUrl(current.bundleRef)} download className="mt-3 inline-block underline">{t(k.download)}</a>
      <form className="mt-6 grid gap-4 md:grid-cols-2" onSubmit={(event) => event.preventDefault()}>
        <label className="flex flex-col gap-1">{t(k.reviewer)}<Input value={reviewer} maxLength={128} onChange={(event) => setReviewer(event.target.value)} disabled={review.isPending} /></label>
        <label className="flex flex-col gap-1">{t(k.rationale)}<Textarea value={rationale} maxLength={2048} onChange={(event) => setRationale(event.target.value)} disabled={review.isPending} /></label>
        <label className="flex flex-col gap-1 md:col-span-2">{t(k.evidence)}<Textarea value={evidence} maxLength={8192} onChange={(event) => setEvidence(event.target.value)} disabled={review.isPending} /></label>
        <label className="flex items-start gap-2 md:col-span-2"><input type="checkbox" checked={overscan} onChange={(event) => setOverscan(event.target.checked)} disabled={review.isPending} /><span>{t(k.overscan)}</span></label>
        {review.error && <p role="alert" className="md:col-span-2">{errorMessage(review.error, t)}</p>}
        <div className="flex gap-2 md:col-span-2">
          <Button type="button" disabled={!ready} onClick={() => review.mutate("accepted")}>{t(k.accept)}</Button>
          <Button type="button" variant="outline" disabled={!ready} onClick={() => review.mutate("rejected")}>{t(k.reject)}</Button>
          {review.isPending && <span role="status">{t(k.saving)}</span>}
        </div>
      </form>
    </div>}
  </section>;
}
