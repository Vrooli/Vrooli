import { useMemo, useRef, useState } from "react";
import { applyCapability, previewCapability, type CapabilityInput, type CapabilityPreview, type CapabilityResult, type CapabilityStatus } from "../../api/capabilities";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Alert } from "@vrooli/react-component-library/Alert/1";
import { Checkbox } from "@vrooli/react-component-library/Checkbox/1";
import { Input } from "@vrooli/react-component-library/Input/1";
import { PasswordInput } from "@vrooli/react-component-library/PasswordInput/2";
import { Select } from "@vrooli/react-component-library/Select/1";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1";
import { FormSection } from "@vrooli/react-component-library/FormSection/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { i18n } from "../../i18n";

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
  const attentionCount = visible.filter((status) => status.state !== "ready").length;

  if (visible.length === 0) return null;

  return <section className="capability-actions" data-testid="capability-actions" aria-label={i18n.t("onboarding.capabilities.label")}>
    <Alert
      tone={attentionCount > 0 ? "warning" : "success"}
      title={attentionCount > 0 ? i18n.t("onboarding.capabilities.attentionTitle", { count: attentionCount }) : i18n.t("onboarding.capabilities.readyTitle")}
      description={attentionCount > 0 ? i18n.t("onboarding.capabilities.attentionDescription") : i18n.t("onboarding.capabilities.readyDescription")}
    />
    <div className="capability-actions__list">
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
            setError((current) => ({ ...current, [status.descriptor.id]: i18n.t("onboarding.capabilities.previewError") }));
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
              const { [status.descriptor.id]: _clearedSecretValues, ...remainingSecretValues } = secretValues.current;
              secretValues.current = remainingSecretValues;
              refreshInputs((revision) => revision + 1);
              setConfirmations((current) => ({ ...current, [status.descriptor.id]: false }));
              setPreviews((current) => ({ ...current, [status.descriptor.id]: undefined }));
              onRefresh();
            }
          } catch {
            setError((current) => ({ ...current, [status.descriptor.id]: i18n.t("onboarding.capabilities.applyError") }));
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
  const blocked = descriptor.disposition === "unsupported" || descriptor.disposition === "deferred" || status.state === "unsupported";
  const missing = new Set(status.missing_inputs ?? []);
  const canPreview = !blocked && (descriptor.inputs?.filter((input) => input.required && input.kind !== "confirmation").every((input) => hasInput(input, values, secretValues, missing)) ?? false);
  const canApply = Boolean(preview && confirmed && canPreview);

  const badgeTone = blocked ? "warning" : status.state === "ready" ? "success" : "neutral";
  const hasDetails = Boolean(descriptor.scope || descriptor.purpose || descriptor.sensitivity || descriptor.disposition || descriptor.provenance);
  return <Card className="capability-card" data-testid={`capability-card-${descriptor.id}`}>
    <CardHeader className="capability-card__header">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <CardTitle as="h3">{descriptor.title}</CardTitle>
          <CardDescription>{descriptor.owner} · {descriptor.id}</CardDescription>
        </div>
        <StatusBadge tone={badgeTone}>{status.state}</StatusBadge>
      </div>
    </CardHeader>
    <CardContent className="capability-card__content">
      {descriptor.description && <p className="text-sm text-foreground">{descriptor.description}</p>}
      {hasDetails && <FormSection title={i18n.t("onboarding.capabilities.details")} summary={descriptor.disposition_reason || undefined} collapsible defaultOpen className="capability-card__details">
        <dl className="grid gap-2 text-xs text-muted sm:grid-cols-2" data-testid={`capability-provenance-${descriptor.id}`}>
          {descriptor.scope && <div><dt className="font-medium">{i18n.t("onboarding.capabilities.scope")}</dt><dd>{descriptor.scope}</dd></div>}
          {descriptor.purpose && <div><dt className="font-medium">{i18n.t("onboarding.capabilities.purpose")}</dt><dd>{descriptor.purpose}</dd></div>}
          {descriptor.sensitivity && <div><dt className="font-medium">{i18n.t("onboarding.capabilities.sensitivity")}</dt><dd>{descriptor.sensitivity}</dd></div>}
          {descriptor.disposition && <div><dt className="font-medium">{i18n.t("onboarding.capabilities.disposition")}</dt><dd>{descriptor.disposition}</dd></div>}
          {descriptor.provenance?.requester && <div><dt className="font-medium">{i18n.t("onboarding.capabilities.requester")}</dt><dd>{descriptor.provenance.requester}</dd></div>}
          {descriptor.provenance?.scope && <div><dt className="font-medium">{i18n.t("onboarding.capabilities.permissionScope")}</dt><dd>{descriptor.provenance.scope}</dd></div>}
          {descriptor.provenance?.grant_source && <div><dt className="font-medium">{i18n.t("onboarding.capabilities.grantSource")}</dt><dd>{descriptor.provenance.grant_source}</dd></div>}
          {descriptor.provenance?.revocation_limit && <div><dt className="font-medium">{i18n.t("onboarding.capabilities.revocationLimit")}</dt><dd>{descriptor.provenance.revocation_limit}</dd></div>}
        </dl>
      </FormSection>}
    {descriptor.risk && <Alert tone="warning" title={i18n.t("onboarding.capabilities.risk", { risk: descriptor.risk })} />}
	    {(descriptor.disposition === "unsupported" || descriptor.disposition === "deferred" || status.state === "unsupported") && <div data-testid={`capability-blocked-${descriptor.id}`}><Alert tone="warning" title={i18n.t("onboarding.capabilities.blocked")} description={descriptor.disposition_reason || status.remediation} /></div>}
    {status.remediation && <p className="mt-2 text-xs text-primary-soft">{i18n.t("onboarding.capabilities.next", { remediation: status.remediation })}</p>}
    {(status.evidence ?? []).length > 0 && <EvidenceList evidence={status.evidence ?? []} />}
    {hasAction && !blocked && <div className="mt-3 space-y-3">
      {(descriptor.inputs ?? []).map((input) => <CapabilityInput key={input.id} input={input} value={input.kind === "secret" ? secretValues[input.id] : values[input.id]} missing={missing.has(input.id)} onValue={(value) => onValue(input.id, value, input.kind === "secret")} />)}
    </div>}
    {hasAction && !blocked && descriptor.policy.requires_confirmation && <Checkbox data-testid={`capability-confirm-${descriptor.id}`} checked={confirmed} onCheckedChange={onConfirm} label={i18n.t("onboarding.capabilities.confirmation")} className="mt-3" />}
    {error && <Alert tone="danger" title={i18n.t("onboarding.capabilities.applyError")} description={error} />}
    {preview && <div className="mt-3 rounded-md border border-primary-soft/30 bg-primary-soft/10 p-3 text-sm" data-testid={`capability-preview-${descriptor.id}`}><p className="font-medium">{i18n.t("onboarding.capabilities.review")}</p><ul className="mt-1 list-disc pl-5">{(preview.mutations ?? []).map((mutation) => <li key={mutation.id}>{mutation.summary}{mutation.reversible ? ` · ${i18n.t("onboarding.capabilities.reversible")}` : ""}</li>)}</ul>{preview.remediation && <p className="mt-2 text-xs text-muted">{preview.remediation}</p>}</div>}
    {result && <div className={`mt-3 rounded-md border p-3 text-sm ${result.state === "ready" ? "border-primary-soft/30 bg-primary-soft/10" : "border-warning/30 bg-warning-surface"}`} data-testid={`capability-result-${descriptor.id}`} role="status"><p className="font-medium">{result.outcome} · {result.state}</p>{result.remediation && <p className="mt-1 text-xs text-muted">{result.remediation}</p>}{result.evidence && <EvidenceList evidence={result.evidence} />}</div>}
    {hasAction && !blocked && <div className="mt-3 flex flex-wrap gap-2">
      <Button type="button" variant="secondary" disabled={busy || !canPreview} onClick={() => { void onPreview(); }}>{busy ? i18n.t("onboarding.capabilities.working") : i18n.t("onboarding.capabilities.preview")}</Button>
      <Button type="button" disabled={busy || !canApply} onClick={() => { void onApply(); }}>{i18n.t("onboarding.capabilities.apply")}</Button>
    </div>}
    </CardContent>
  </Card>;
}

function CapabilityInput({ input, value, missing, onValue }: { input: CapabilityInput; value?: InputValue; missing: boolean; onValue: (value: InputValue) => void }) {
  if (input.kind === "confirmation") return null;
  const label = <span className="font-medium">{input.label}{input.required ? ` · ${i18n.t("onboarding.capabilities.required")}` : ""}{missing ? ` · ${i18n.t("onboarding.capabilities.needed")}` : ""}</span>;
  if (input.kind === "boolean") return <Checkbox checked={value === true} onCheckedChange={onValue as (checked: boolean) => void} label={label} />;
  const options = input.options ?? [];
  const candidates = input.candidates ?? [];
  const selectOptions = options.length > 0 ? options.map((option) => ({ value: option, label: option })) : candidates.map((candidate) => ({ value: candidate.id, label: candidate.label || candidate.location || candidate.id }));
  return <label className="block text-sm"><span>{label}</span>{input.description && <span className="mt-1 block text-xs text-muted">{input.description}</span>}{input.validation && <span className="mt-1 block text-xs text-muted">{i18n.t("onboarding.capabilities.validation", { value: input.validation })}</span>}{selectOptions.length > 0 ? <Select value={typeof value === "string" ? value : ""} onValueChange={onValue} className="mt-2" aria-label={input.label} options={selectOptions} placeholder={i18n.t("onboarding.capabilities.choose")} /> : input.kind === "secret" ? <PasswordInput revealable={false} autoComplete="off" maxLength={input.constraints?.max_length} value={typeof value === "string" ? value : ""} placeholder={input.default ?? ""} onValueChange={onValue} className="mt-2" aria-label={input.label} /> : <Input type={input.kind === "duration" ? "text" : "text"} autoComplete="off" maxLength={input.constraints?.max_length} value={typeof value === "string" ? value : ""} placeholder={input.default ?? ""} onChange={(event) => onValue(event.target.value)} className="mt-2" aria-label={input.label} />}</label>;
}

function EvidenceList({ evidence }: { evidence: Array<{ kind: string; artifact_identity: string; verified: boolean; coverage?: string[]; remediation?: string }> }) {
  return <div className="mt-2" data-testid="capability-evidence"><p className="text-xs font-medium">{i18n.t("onboarding.capabilities.evidence")}</p><ul className="mt-1 space-y-1 text-xs text-muted">{evidence.map((item) => <li key={`${item.kind}-${item.artifact_identity}`}>{item.kind} · {item.verified ? i18n.t("onboarding.capabilities.verified") : i18n.t("onboarding.capabilities.notVerified")}{item.coverage ? ` · ${item.coverage.length} covered` : ""}{item.remediation ? ` · ${item.remediation}` : ""}</li>)}</ul></div>;
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
