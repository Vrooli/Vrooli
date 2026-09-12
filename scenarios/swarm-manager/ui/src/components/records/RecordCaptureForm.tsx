/**
 * RecordCaptureForm — progressive record intake and private-draft repair.
 *
 * Unlike the legacy strict record form, every submission is useful: complete
 * inputs publish and incomplete inputs are preserved as a private draft with
 * the server's exact repair guidance.
 */

import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import {
  ALL_RECORD_KINDS,
  ALL_RECORD_OUTCOMES,
  type RecordCaptureInput,
  type RecordCaptureMetadata,
  type RecordCaptureResult,
  type RecordItem,
} from "../../types";

interface RecordCaptureFormProps {
  record?: RecordItem;
  onSubmit: (input: RecordCaptureInput) => Promise<RecordCaptureResult>;
}

interface FormState {
  kind: string;
  scenario: string;
  trigger: string;
  approach: string;
  evidence: string;
  ruledOut: string;
  outcome: string;
}

function formState(record?: RecordItem): FormState {
  const raw = record?.capture?.raw;
  return {
    kind: raw?.kind ?? record?.kind ?? "",
    scenario: raw?.scenario ?? record?.scenario ?? "",
    trigger: raw?.trigger ?? record?.trigger ?? "",
    approach: raw?.approach ?? record?.approach ?? "",
    evidence: raw?.evidence ?? record?.evidence ?? "",
    ruledOut: record?.ruledOut.join("\n") ?? "",
    outcome: raw?.outcome ?? record?.outcome ?? "",
  };
}

function draftResult(record?: RecordItem): RecordCaptureResult | null {
  if (!record?.draft) return null;
  const capture: RecordCaptureMetadata = record.capture ?? {};
  return {
    disposition: "draft",
    record,
    accepted: capture.accepted ?? {},
    needs: capture.needs ?? [],
    invalid: capture.invalid ?? [],
    warnings: capture.warnings ?? ["Draft saved privately; it is not searchable or published."],
    nextAction: [],
  };
}

export function RecordCaptureForm({ record, onSubmit }: RecordCaptureFormProps) {
  const [values, setValues] = useState(() => formState(record));
  const [result, setResult] = useState<RecordCaptureResult | null>(() => draftResult(record));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // A successful repair replaces the draft in-place. Keep a direct navigation
  // or query refetch from leaving stale form values behind.
  useEffect(() => {
    setValues(formState(record));
    setResult(draftResult(record));
  }, [record]);

  const update = (field: keyof FormState, value: string) => setValues((current) => ({ ...current, [field]: value }));

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const next = await onSubmit({
        kind: values.kind.trim(),
        scenario: values.scenario.trim(),
        trigger: values.trigger.trim(),
        approach: values.approach.trim(),
        evidence: values.evidence.trim() || undefined,
        ruledOut: values.ruledOut.split("\n").map((value) => value.trim()).filter(Boolean),
        outcome: values.outcome.trim(),
      });
      setResult(next);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const isRepair = Boolean(record?.draft);
  return (
    <form onSubmit={submit} className="flex flex-col gap-3" data-testid="record-capture-form">
      <p className="text-sm text-slate-400">
        {isRepair
          ? "Repair this private draft. It will publish automatically once all required fields are valid."
          : "Capture what you know now. Incomplete entries are saved privately with guided repair instead of being lost."}
      </p>

      <div className="grid gap-3 md:grid-cols-2">
        <label className="flex flex-col gap-1 text-sm text-slate-300">
          Kind
          <select className="rounded border border-slate-700 bg-slate-900 px-3 py-2 text-slate-100" value={values.kind} onChange={(event) => update("kind", event.target.value)} data-testid="record-capture-kind">
            <option value="">Choose when known</option>
            {ALL_RECORD_KINDS.map((kind) => <option key={kind} value={kind}>{kind}</option>)}
          </select>
        </label>
        <label className="flex flex-col gap-1 text-sm text-slate-300">
          Outcome
          <select className="rounded border border-slate-700 bg-slate-900 px-3 py-2 text-slate-100" value={values.outcome} onChange={(event) => update("outcome", event.target.value)} data-testid="record-capture-outcome">
            <option value="">Choose when known</option>
            {ALL_RECORD_OUTCOMES.map((outcome) => <option key={outcome} value={outcome}>{outcome}</option>)}
          </select>
        </label>
      </div>
      <label className="flex flex-col gap-1 text-sm text-slate-300">
        Scenario
        <input className="rounded border border-slate-700 bg-slate-900 px-3 py-2 text-slate-100" value={values.scenario} onChange={(event) => update("scenario", event.target.value)} placeholder="e.g. swarm-manager" data-testid="record-capture-scenario" />
      </label>
      <label className="flex flex-col gap-1 text-sm text-slate-300">
        Trigger or goal
        <textarea className="min-h-20 rounded border border-slate-700 bg-slate-900 px-3 py-2 text-slate-100" value={values.trigger} onChange={(event) => update("trigger", event.target.value)} data-testid="record-capture-trigger" />
      </label>
      <label className="flex flex-col gap-1 text-sm text-slate-300">
        Approach
        <textarea className="min-h-24 rounded border border-slate-700 bg-slate-900 px-3 py-2 text-slate-100" value={values.approach} onChange={(event) => update("approach", event.target.value)} data-testid="record-capture-approach" />
      </label>
      <label className="flex flex-col gap-1 text-sm text-slate-300">
        Evidence or validation notes
        <textarea className="min-h-16 rounded border border-slate-700 bg-slate-900 px-3 py-2 text-slate-100" value={values.evidence} onChange={(event) => update("evidence", event.target.value)} data-testid="record-capture-evidence" />
      </label>
      <label className="flex flex-col gap-1 text-sm text-slate-300">
        Ruled out (one per line)
        <textarea className="min-h-16 rounded border border-slate-700 bg-slate-900 px-3 py-2 text-slate-100" value={values.ruledOut} onChange={(event) => update("ruledOut", event.target.value)} data-testid="record-capture-ruled-out" />
      </label>

      {error ? <div className="rounded border border-red-700 bg-red-950/40 p-3 text-sm text-red-200">{error}</div> : null}
      {result ? <CaptureDisposition result={result} /> : null}

      <button type="submit" disabled={busy} className="self-start rounded bg-emerald-600 px-3 py-2 text-sm font-medium text-white disabled:opacity-50" data-testid="record-capture-submit">
        {busy ? "Saving…" : isRepair ? "Save repair" : "Capture record"}
      </button>
    </form>
  );
}

function CaptureDisposition({ result }: { result: RecordCaptureResult }) {
  const published = result.disposition === "published";
  return (
    <div className={`rounded border p-3 text-sm ${published ? "border-emerald-700 bg-emerald-950/40 text-emerald-100" : "border-amber-700 bg-amber-950/40 text-amber-100"}`} data-testid="record-capture-disposition">
      <p className="font-medium">{published ? "Published" : "Private draft saved"}</p>
      <p className="mt-1">{published ? "This record is now part of searchable shared learning." : "This record is private until the listed needs are repaired."}</p>
      {result.needs.length > 0 ? <p className="mt-2"><span className="font-medium">Still needed:</span> {result.needs.join(", ")}</p> : null}
      {result.invalid.length > 0 ? (
        <ul className="mt-2 list-disc space-y-1 pl-5">
          {result.invalid.map((invalid) => <li key={`${invalid.field}:${invalid.value}`}><span className="font-medium">{invalid.field}:</span> {invalid.message}</li>)}
        </ul>
      ) : null}
      {result.warnings.map((warning) => <p key={warning} className="mt-2 text-xs">{warning}</p>)}
      {result.record.id ? <Link className="mt-3 inline-block underline underline-offset-4" to={`/records/${result.record.id}`} data-testid="record-capture-open">{published ? "Open published record" : "Open draft and repair"}</Link> : null}
    </div>
  );
}
