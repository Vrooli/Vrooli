import { useMemo, useState } from "react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1.3.1";
import { intakeClient } from "../api/intake";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

type IntakeForm = { artifactId: string; revisionId: string; producer: string; intent: string; rawItems: string };
type StoredArticle = { batch_id: string; item: { external_id: string; variants?: Array<{ id: string; name: string; channel: string; body: string }> }; assessment: { status: string }; receipt: { id: string } };
const EMPTY_FORM: IntakeForm = { artifactId: "", revisionId: "", producer: "", intent: "", rawItems: "[]" };
const LABEL = strings.pages.intake;

function parseItems(raw: string): string[] {
  const parsed: unknown = JSON.parse(raw);
  if (!Array.isArray(parsed) || parsed.some((item) => !item || typeof item !== "object" || Array.isArray(item))) {
    throw new Error("Article items must be a JSON array of objects.");
  }
  return parsed.map((item) => JSON.stringify(item));
}

export function IntakePage() {
  const { t } = useTranslation();
  const [form, setForm] = useState(EMPTY_FORM);
  const [approved, setApproved] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [result, setResult] = useState("");
  const [batchId, setBatchId] = useState("");
  const [storedItems, setStoredItems] = useState<StoredArticle[]>([]);
  const [selectedExternalId, setSelectedExternalId] = useState("");
  const [selectedVariantIds, setSelectedVariantIds] = useState<string[]>([]);
  const [sourceChannel, setSourceChannel] = useState("");
  const [format, setFormat] = useState("post");
  const [postTypeId, setPostTypeId] = useState("");
  const [previewKey, setPreviewKey] = useState("");
  const [operationKey, setOperationKey] = useState("");
  const [promotionAckKey, setPromotionAckKey] = useState("");
  const source = useMemo(() => ({ artifactId: form.artifactId, revisionId: form.revisionId, producer: form.producer, importIntent: form.intent }), [form]);
  const formKey = `${JSON.stringify(source)}\n${form.rawItems}`;
  const canAdmit = approved && previewKey === formKey && !busy;
  const canReconcile = canAdmit && Boolean(batchId);
  const selectedItem = storedItems.find((entry) => entry.item.external_id === selectedExternalId);
  const selectedVariants = selectedItem?.item.variants ?? [];
  const promotionBasis = JSON.stringify([selectedItem?.batch_id, selectedExternalId, selectedItem?.receipt.id, [...selectedVariantIds].sort(), sourceChannel, format, postTypeId]);
  const canPromote = !busy && promotionAckKey === promotionBasis && selectedItem?.assessment.status === "admissible" && Boolean(format && postTypeId) && (selectedVariants.length > 0 ? selectedVariantIds.length > 0 : Boolean(sourceChannel));

  function update(patch: Partial<IntakeForm>) {
    setForm((current) => ({ ...current, ...patch }));
    setApproved(false);
    setPreviewKey("");
    setResult("");
    setError("");
  }

  async function run(action: "preview" | "admit" | "list" | "reconcile") {
    setBusy(true);
    setError("");
    try {
      let response: string;
      if (action === "list") {
        const value = await intakeClient.list({ batchId });
        response = value.resultJson;
      } else {
        const rawItemsJson = parseItems(form.rawItems);
        const exactSource = { artifactId: source.artifactId, revisionId: source.revisionId, producer: source.producer, importIntent: source.importIntent };
        if (action === "preview") {
          response = (await intakeClient.preview({ source: exactSource, rawItemsJson })).resultJson;
          setPreviewKey(formKey);
          setOperationKey("");
        } else if (action === "admit") {
          if (!canAdmit) throw new Error(t(LABEL.approvalHint));
          const key = operationKey || crypto.randomUUID();
          setOperationKey(key);
          response = (await intakeClient.admit({ source: exactSource, idempotencyKey: key, rawItemsJson })).resultJson;
          const stored = JSON.parse(response) as StoredArticle[];
          const receiptBatchId = stored[0]?.batch_id;
          if (receiptBatchId) setBatchId(receiptBatchId);
          setStoredItems(stored);
          setSelectedExternalId(stored[0]?.item?.external_id ?? "");
        } else {
          if (!canReconcile) throw new Error(t(LABEL.approvalHint));
          response = (await intakeClient.reconcile({ batchId, source: exactSource, rawItemsJson })).resultJson;
        }
      }
      setResult(JSON.stringify(JSON.parse(response), null, 2));
      if (action === "list" || action === "reconcile") {
        const stored = JSON.parse(response) as StoredArticle[];
        setStoredItems(stored);
        setSelectedExternalId((current) => stored.some((entry) => entry.item.external_id === current) ? current : stored[0]?.item.external_id ?? "");
      }
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : t(LABEL.error));
    } finally {
      setBusy(false);
    }
  }

  async function promoteSelected() {
    if (!selectedItem || !canPromote) return;
    setBusy(true);
    setError("");
    try {
      const response = await intakeClient.promote({
        batchId: selectedItem.batch_id,
        externalId: selectedItem.item.external_id,
        expectedReceiptId: selectedItem.receipt.id,
        variantIds: selectedVariantIds,
        channel: selectedVariants.length === 0 ? sourceChannel : "",
        format,
        postTypeId,
        campaignId: "",
        projectId: "",
      });
      setResult(JSON.stringify(JSON.parse(response.resultJson), null, 2));
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : t(LABEL.error));
    } finally {
      setBusy(false);
    }
  }

  return <section className="mx-auto w-full max-w-5xl space-y-4 px-4 py-6 sm:px-6 lg:px-8" data-experience-surface="intake" data-testid="page-intake">
    <header>
      <h1 className="font-serif text-4xl font-medium tracking-tight">{t(LABEL.title)}</h1>
      <p className="mt-2 max-w-3xl text-sm text-app-muted-foreground">{t(LABEL.description)}</p>
    </header>
    <Card>
      <CardHeader><CardTitle>{t(LABEL.source)}</CardTitle></CardHeader>
      <CardContent className="grid gap-4 sm:grid-cols-2">
        {(["artifactId", "revisionId", "producer", "intent"] as const).map((field) => {
          const label = field === "artifactId" ? LABEL.artifact : field === "revisionId" ? LABEL.revision : field === "producer" ? LABEL.producer : LABEL.intent;
          return <label key={field} className="grid gap-1 text-sm">{t(label)}<input className="min-h-11 rounded-control border border-app-border bg-app-background px-3" value={form[field]} onChange={(event) => update({ [field]: event.target.value })} /></label>;
        })}
        <label className="grid gap-1 text-sm sm:col-span-2">{t(LABEL.items)}<textarea className="min-h-48 rounded-control border border-app-border bg-app-background p-3 font-mono text-xs" value={form.rawItems} onChange={(event) => update({ rawItems: event.target.value })} aria-describedby="intake-items-hint" /></label>
        <p id="intake-items-hint" className="text-xs text-app-muted-foreground sm:col-span-2">{t(LABEL.itemsHint)}</p>
        <label className="flex gap-3 rounded-control border border-app-border p-3 text-sm sm:col-span-2"><input className="mt-1 size-4 shrink-0" type="checkbox" checked={approved} onChange={(event) => setApproved(event.target.checked)} /><span>{t(LABEL.approval)}<span className="mt-1 block text-xs text-app-muted-foreground">{t(LABEL.approvalHint)}</span></span></label>
        <div className="flex flex-wrap gap-2 sm:col-span-2">
          <Button variant="secondary" disabled={busy} onClick={() => void run("preview")}>{busy ? t(LABEL.working) : t(LABEL.preview)}</Button>
          <Button disabled={!canAdmit} onClick={() => void run("admit")}>{t(LABEL.admit)}</Button>
        </div>
      </CardContent>
    </Card>
    <Card>
      <CardHeader><CardTitle>{t(LABEL.result)}</CardTitle></CardHeader>
      <CardContent className="space-y-3">
        <p className="text-sm text-app-muted-foreground">{previewKey === formKey ? t(LABEL.previewReady) : t(LABEL.empty)}</p>
        <div className="flex flex-wrap items-end gap-2">
          <label className="grid flex-1 gap-1 text-sm">{t(LABEL.batch)}<input className="min-h-11 rounded-control border border-app-border bg-app-background px-3" value={batchId} onChange={(event) => setBatchId(event.target.value)} /></label>
          <Button variant="secondary" disabled={busy || !batchId} onClick={() => void run("list")}>{t(LABEL.load)}</Button>
          <Button variant="secondary" disabled={!canReconcile} onClick={() => void run("reconcile")}>{t(LABEL.reconcile)}</Button>
        </div>
        {error ? <p role="alert" className="text-sm text-app-danger">{error || t(LABEL.error)}</p> : null}
        {result ? <pre className="max-h-[32rem] overflow-auto rounded-control border border-app-border bg-app-background p-3 text-xs">{result}</pre> : null}
      </CardContent>
    </Card>
    {storedItems.length ? <Card>
      <CardHeader><CardTitle>{t(LABEL.promoteTitle)}</CardTitle></CardHeader>
      <CardContent className="space-y-3">
        <p className="text-sm text-app-muted-foreground">{t(LABEL.promoteHint)}</p>
        <label className="grid gap-1 text-sm">{t(LABEL.selectItem)}<select className="min-h-11 rounded-control border border-app-border bg-app-background px-3" value={selectedExternalId} onChange={(event) => { setSelectedExternalId(event.target.value); setSelectedVariantIds([]); }}>
          {storedItems.map((entry) => <option key={entry.item.external_id} value={entry.item.external_id}>{entry.item.external_id} · {entry.assessment.status}</option>)}
        </select></label>
        {selectedVariants.length ? <fieldset className="rounded-control border border-app-border p-3"><legend className="px-1 text-sm">{t(LABEL.variants)}</legend><div className="space-y-2">{selectedVariants.map((variant) => <label key={variant.id} className="flex gap-2 text-sm"><input type="checkbox" checked={selectedVariantIds.includes(variant.id)} onChange={(event) => setSelectedVariantIds((current) => event.target.checked ? [...current, variant.id] : current.filter((id) => id !== variant.id))} /><span>{variant.name} · {variant.channel}<span className="mt-1 block whitespace-pre-wrap text-xs text-app-muted-foreground">{variant.body}</span></span></label>)}</div></fieldset> : <label className="grid gap-1 text-sm">{t(LABEL.channel)}<input className="min-h-11 rounded-control border border-app-border bg-app-background px-3" value={sourceChannel} onChange={(event) => setSourceChannel(event.target.value)} /></label>}
        <div className="grid gap-3 sm:grid-cols-2">
          <label className="grid gap-1 text-sm">{t(LABEL.format)}<input className="min-h-11 rounded-control border border-app-border bg-app-background px-3" value={format} onChange={(event) => setFormat(event.target.value)} /></label>
          <label className="grid gap-1 text-sm">{t(LABEL.postType)}<input className="min-h-11 rounded-control border border-app-border bg-app-background px-3" value={postTypeId} onChange={(event) => setPostTypeId(event.target.value)} /></label>
        </div>
        <label className="flex gap-3 rounded-control border border-app-border p-3 text-sm"><input className="mt-1 size-4 shrink-0" type="checkbox" checked={promotionAckKey === promotionBasis} onChange={(event) => setPromotionAckKey(event.target.checked ? promotionBasis : "")} /><span>{t(LABEL.promoteApproval)}<span className="mt-1 block text-xs text-app-muted-foreground">{t(LABEL.promoteApprovalHint)}</span></span></label>
        <Button disabled={!canPromote} onClick={() => void promoteSelected()}>{t(LABEL.promote)}</Button>
      </CardContent>
    </Card> : null}
  </section>;
}
