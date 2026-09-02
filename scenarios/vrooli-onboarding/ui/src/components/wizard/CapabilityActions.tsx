import { useMemo, useRef, useState } from "react";
import { applyCapability, previewCapability } from "../../lib/api";
import type { CapabilityInput, CapabilityPreview, CapabilityResult, CapabilityStatus } from "../../types";
import { Button } from "@vrooli/react-component-library/Button/2";

interface CapabilityActionsProps {
  statuses: CapabilityStatus[];
  onRefresh: () => void;
}

type InputValue = string | boolean;

export function CapabilityActions({ statuses, onRefresh }: CapabilityActionsProps) {
  const [values, setValues] = useState<Record<string, Record<string, InputValue>>>({});
  const [confirmations, setConfirmations] = useState<Record<string, boolean>>({});
  const [previews, setPreviews] = useState<Record<string, CapabilityPreview | undefined>>({});
  const [results, setResults] = useState<Record<string, CapabilityResult | undefined>>({});
  const [busy, setBusy] = useState<string | null>(null);
  const [error, setError] = useState<Record<string, string | undefined>>({});
  const secretValues = useRef<Record<string, Record<string, string>>>({});
  const [, refreshInputs] = useState(0);

  const visible = useMemo(() => statuses, [statuses]);

  if (visible.length === 0) return null;

  return <section className="mt-6" data-testid="capability-actions" aria-label="Operator capabilities">
    <h2 className="text-lg font-medium">Operator capabilities</h2>
    <p className="mt-1 text-sm text-muted">Each provider declares its own inputs, safeguards, and evidence. Secret answers stay in memory for this action and are never shown in status or evidence.</p>
    <div className="mt-3 space-y-4">
      {visible.map((status) => <CapabilityCard
        key={status.descriptor.id}
        status={status}
        values={values[status.descriptor.id] ?? {}}
        secretValues={secretValues.current[status.descriptor.id] ?? {}}
        confirmed={confirmations[status.descriptor.id] ?? false}
        preview={previews[status.descriptor.id]}
        result={results[status.descriptor.id]}
        busy={busy === status.descriptor.id}
        error={error[status.descriptor.id]}
        onValue={(id, value, secret) => {
          if (secret) {
            const current = secretValues.current[status.descriptor.id] ?? {};
            current[id] = String(value);
            secretValues.current[status.descriptor.id] = current;
            refreshInputs((revision) => revision + 1);
            return;
          }
          setValues((current) => ({ ...current, [status.descriptor.id]: { ...current[status.descriptor.id], [id]: value } }));
        }}
        onConfirm={(value) => setConfirmations((current) => ({ ...current, [status.descriptor.id]: value }))}
        onPreview={async () => {
          setBusy(status.descriptor.id);
          setError((current) => ({ ...current, [status.descriptor.id]: undefined }));
          try {
            const preview = await previewCapability(makeRequest(status, values[status.descriptor.id] ?? {}, secretValues.current[status.descriptor.id] ?? {}, false));
            setPreviews((current) => ({ ...current, [status.descriptor.id]: preview }));
            setResults((current) => ({ ...current, [status.descriptor.id]: undefined }));
          } catch {
            setError((current) => ({ ...current, [status.descriptor.id]: "The capability preview failed. Correct the reported input or host condition and try again." }));
          } finally {
            setBusy(null);
          }
        }}
        onApply={async () => {
          setBusy(status.descriptor.id);
          setError((current) => ({ ...current, [status.descriptor.id]: undefined }));
          try {
            const result = await applyCapability(makeRequest(status, values[status.descriptor.id] ?? {}, secretValues.current[status.descriptor.id] ?? {}, confirmations[status.descriptor.id] ?? false));
            setResults((current) => ({ ...current, [status.descriptor.id]: result }));
            if (result.state === "ready" || result.state === "degraded") {
              setValues((current) => ({ ...current, [status.descriptor.id]: {} }));
              delete secretValues.current[status.descriptor.id];
              refreshInputs((revision) => revision + 1);
              setConfirmations((current) => ({ ...current, [status.descriptor.id]: false }));
              setPreviews((current) => ({ ...current, [status.descriptor.id]: undefined }));
              onRefresh();
            }
          } catch {
            setError((current) => ({ ...current, [status.descriptor.id]: "The capability could not be applied. No success is reported until its provider returns verified evidence." }));
          } finally {
            setBusy(null);
          }
        }}
      />)}
    </div>
  </section>;
}

function CapabilityCard({
  status,
  values,
  secretValues,
  confirmed,
  preview,
  result,
  busy,
  error,
  onValue,
  onConfirm,
  onPreview,
  onApply,
}: {
  status: CapabilityStatus;
  values: Record<string, InputValue>;
  secretValues: Record<string, string>;
  confirmed: boolean;
  preview?: CapabilityPreview;
  result?: CapabilityResult;
  busy: boolean;
  error?: string;
  onValue: (id: string, value: InputValue, secret: boolean) => void;
  onConfirm: (value: boolean) => void;
  onPreview: () => Promise<void>;
  onApply: () => Promise<void>;
}) {
  const descriptor = status.descriptor;
  const hasAction = (descriptor.inputs ?? []).length > 0;
  const missing = new Set(status.missing_inputs ?? []);
  const canPreview = descriptor.inputs?.filter((input) => input.required && input.kind !== "confirmation").every((input) => hasInput(input, values, secretValues, missing)) ?? false;
  const canApply = Boolean(preview && confirmed && canPreview);

  return <article className="rounded-lg border border-muted bg-surface-muted p-4" data-testid={`capability-card-${descriptor.id}`}>
    <div className="flex flex-wrap items-start justify-between gap-2">
      <div>
        <h3 className="font-medium">{descriptor.title}</h3>
        <p className="mt-1 text-xs text-muted">{descriptor.owner} · {status.state}</p>
      </div>
      <span className="rounded-full border border-muted px-2 py-1 text-xs text-muted">{descriptor.id}</span>
    </div>
    {descriptor.description && <p className="mt-2 text-sm text-foreground">{descriptor.description}</p>}
    {descriptor.risk && <p className="mt-2 text-xs text-warning">Risk: {descriptor.risk}</p>}
    {status.remediation && <p className="mt-2 text-xs text-primary-soft">Next: {status.remediation}</p>}
    {(status.evidence ?? []).length > 0 && <EvidenceList evidence={status.evidence ?? []} />}
    {hasAction && <div className="mt-3 space-y-3">
      {(descriptor.inputs ?? []).map((input) => <CapabilityInput key={input.id} input={input} value={input.kind === "secret" ? secretValues[input.id] : values[input.id]} missing={missing.has(input.id)} onValue={(value) => onValue(input.id, value, input.kind === "secret")} />)}
    </div>}
    {hasAction && descriptor.policy.requires_confirmation && <label className="mt-3 flex items-start gap-2 text-sm"><input data-testid={`capability-confirm-${descriptor.id}`} type="checkbox" checked={confirmed} onChange={(event) => onConfirm(event.target.checked)} className="mt-1 h-4 w-4" /><span>I reviewed the provider preview and authorize its declared mutations. This confirmation is required for every apply.</span></label>}
    {error && <p className="mt-3 text-sm text-danger" role="alert">{error}</p>}
    {preview && <div className="mt-3 rounded-md border border-primary-soft/30 bg-primary-soft/10 p-3 text-sm" data-testid={`capability-preview-${descriptor.id}`}><p className="font-medium">Review preview</p><ul className="mt-1 list-disc pl-5">{(preview.mutations ?? []).map((mutation) => <li key={mutation.id}>{mutation.summary}{mutation.reversible ? " · reversible" : ""}</li>)}</ul>{preview.remediation && <p className="mt-2 text-xs text-muted">{preview.remediation}</p>}</div>}
    {result && <div className={`mt-3 rounded-md border p-3 text-sm ${result.state === "ready" ? "border-primary-soft/30 bg-primary-soft/10" : "border-warning/30 bg-warning-surface"}`} data-testid={`capability-result-${descriptor.id}`} role="status"><p className="font-medium">{result.outcome} · {result.state}</p>{result.remediation && <p className="mt-1 text-xs text-muted">{result.remediation}</p>}{result.evidence && <EvidenceList evidence={result.evidence} />}</div>}
    {hasAction && <div className="mt-3 flex flex-wrap gap-2">
      <Button type="button" variant="secondary" disabled={busy || !canPreview} onClick={() => { void onPreview(); }}>{busy ? "Working…" : "Preview"}</Button>
      <Button type="button" disabled={busy || !canApply} onClick={() => { void onApply(); }}>Apply reviewed capability</Button>
    </div>}
  </article>;
}

function CapabilityInput({ input, value, missing, onValue }: { input: CapabilityInput; value?: InputValue; missing: boolean; onValue: (value: InputValue) => void }) {
  if (input.kind === "confirmation") return null;
  const label = <span className="font-medium">{input.label}{input.required ? " · required" : ""}{missing ? " · needed" : ""}</span>;
  if (input.kind === "boolean") return <label className="flex items-start gap-2 text-sm"><input type="checkbox" checked={value === true} onChange={(event) => onValue(event.target.checked)} className="mt-1 h-4 w-4" />{label}</label>;
  const options = input.options ?? [];
  const candidates = input.candidates ?? [];
  const selectOptions = options.length > 0 ? options.map((option) => ({ value: option, label: option })) : candidates.map((candidate) => ({ value: candidate.id, label: candidate.label || candidate.location || candidate.id }));
  return <label className="block text-sm"><span>{label}</span>{input.description && <span className="mt-1 block text-xs text-muted">{input.description}</span>}{input.validation && <span className="mt-1 block text-xs text-muted">Validation: {input.validation}</span>}{selectOptions.length > 0 ? <select value={typeof value === "string" ? value : ""} onChange={(event) => onValue(event.target.value)} className="mt-2 min-h-11 w-full rounded-md border border-muted bg-surface px-3 py-2" aria-label={input.label}><option value="">Choose…</option>{selectOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}</select> : <input type={input.kind === "secret" ? "password" : "text"} autoComplete="off" value={typeof value === "string" ? value : ""} placeholder={input.default ?? ""} onChange={(event) => onValue(event.target.value)} className="mt-2 min-h-11 w-full rounded-md border border-muted bg-surface px-3 py-2" aria-label={input.label} />}</label>;
}

function EvidenceList({ evidence }: { evidence: Array<{ kind: string; artifact_identity: string; verified: boolean; coverage?: string[]; remediation?: string }> }) {
  return <div className="mt-2" data-testid="capability-evidence"><p className="text-xs font-medium">Evidence</p><ul className="mt-1 space-y-1 text-xs text-muted">{evidence.map((item) => <li key={`${item.kind}-${item.artifact_identity}`}>{item.kind} · {item.verified ? "verified" : "not verified"}{item.coverage ? ` · ${item.coverage.length} covered` : ""}{item.remediation ? ` · ${item.remediation}` : ""}</li>)}</ul></div>;
}

function hasInput(input: CapabilityInput, values: Record<string, InputValue>, secretValues: Record<string, string>, missing: Set<string>) {
  if (!input.required) return true;
  const value = input.kind === "secret" ? secretValues[input.id] : values[input.id] ?? input.default;
  if (missing.has(input.id) && (value === undefined || value === "") && !input.default) return false;
  return typeof value === "boolean" ? true : Boolean(value && value.trim());
}

function makeRequest(status: CapabilityStatus, values: Record<string, InputValue>, secretValues: Record<string, string>, confirm: boolean) {
  const inputs: Record<string, unknown> = {};
  for (const input of status.descriptor.inputs ?? []) {
    const value = input.kind === "secret" ? secretValues[input.id] : values[input.id] ?? input.default;
    if (value !== undefined && value !== "") inputs[input.id] = value;
  }
  return { capability_id: status.descriptor.id, confirm, inputs };
}
