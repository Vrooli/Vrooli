import { useQuery } from "@tanstack/react-query";
import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { CheckCircle2, CircleAlert, Loader2 } from "lucide-react";
import { fetchCapabilities } from "../../api/capabilities";
import { provisionCredential } from "../../api/credentials";
import { cancelApply, fetchApplyPlan, fetchApplyRun, reviewApply, startApply } from "../../api/apply";
import { acknowledgeDegraded, fetchReadiness } from "../../api/readiness";
import { fetchOperatorInputs, resolveOperatorInputs } from "../../api/operatorinputs";
import { pollApplyRun } from "../../lib/applyRun";
import { ApplyRunState, ApplyStepState } from "@vrooli/proto-types/vrooli-onboarding/v1/apply/apply_pb";
import type { GetApplyRunResponse } from "@vrooli/proto-types/vrooli-onboarding/v1/apply/apply_pb";
import type { CompletionBlocker, ReadinessItem } from "../../api/readiness";
import { create } from "@bufbuild/protobuf";
import { OperatorInputKind, type OperatorInputRequest } from "@vrooli/proto-types/setup/v1/operator_input_pb";
import { AnswerSchema, type Answer } from "@vrooli/proto-types/vrooli-onboarding/v1/operatorinputs/operatorinputs_pb";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Checkbox } from "@vrooli/react-component-library/Checkbox/1";
import { ApplyPlanDisclosure } from "./ApplyPlanDisclosure";
import { CapabilityActions } from "./CapabilityActions";
import { GeneratedForm } from "@vrooli/react-component-library/GeneratedForm/1";
import { PasswordInput as SecureValueInput } from "@vrooli/react-component-library/PasswordInput/2";
import { toGeneratedFields, type OperatorInput } from "@vrooli/react-component-library/ValidationAdapter/1";
import type { GeneratedField } from "@vrooli/react-component-library/GeneratedForm/1";
import { PlanSummary } from "@vrooli/react-component-library/PlanSummary/0";
import { RunLadder } from "@vrooli/react-component-library/RunLadder/0";
import { Timeline } from "@vrooli/react-component-library/Timeline/1";
import { i18n } from "../../i18n";

type ReadinessSurfaceProps = { title: "Credentials" | "Apply" | "Validation"; target?: string };

function applyRunStateName(state: ApplyRunState): string {
  return ApplyRunState[state]?.toLowerCase().replace(/_/g, " ") ?? "unknown";
}

function applyRunStorageKey(target: string) {
  return `vrooli-onboarding.apply-run.${target}`;
}

function isApplyRunActive(status: ApplyRunState) {
  return status === ApplyRunState.PENDING || status === ApplyRunState.APPLYING;
}

function applyRunLabel(status: ApplyRunState) {
  switch (status) {
    case ApplyRunState.APPLIED:
    case ApplyRunState.ALREADY_SATISFIED:
      return i18n.t("onboarding.readiness.applied");
    case ApplyRunState.PARTIALLY_APPLIED:
      return i18n.t("onboarding.readiness.partiallyApplied");
    case ApplyRunState.CANCELLED:
      return i18n.t("onboarding.readiness.cancelled");
    case ApplyRunState.INDETERMINATE:
      return i18n.t("onboarding.readiness.indeterminate");
    case ApplyRunState.FAILED:
      return i18n.t("onboarding.readiness.failed");
    default:
      return i18n.t("onboarding.readiness.running");
  }
}

type ApplyRunView = {
  run_id: string;
  status: ApplyRunState;
  status_name: string;
  items: Array<{ name: string; outcome: string; error?: string }>;
};

function applyRunView(current: GetApplyRunResponse): ApplyRunView {
  return {
    run_id: current.runId,
    status: current.status,
    status_name: current.legacyStatus || applyRunStateName(current.status),
    items: current.steps.map((item) => ({
      name: item.name,
      outcome: item.legacyOutcome || ApplyStepState[item.state]?.toLowerCase().replace(/_/g, " ") || "unknown",
      error: item.error,
    })),
  };
}

export function StepCredentials({ target = "local" }: { target?: string }) {
  return <ReadinessSurface title={"Credentials"} target={target} />;
}

export function StepReady({ target = "local" }: { target?: string }) {
  return <ReadinessSurface title={"Validation"} target={target} />;
}

export function ReadinessSurface({ title, target = "local" }: ReadinessSurfaceProps) {
  const { data, isLoading, error, refetch } = useQuery({ queryKey: ["readiness", target], queryFn: () => fetchReadiness(target) });
  const { data: capabilities, refetch: refetchCapabilities } = useQuery({ queryKey: ["capabilities", target], queryFn: () => fetchCapabilities(target), enabled: title === "Credentials" });
  const { data: operatorInputs } = useQuery({ queryKey: ["operator-inputs", target], queryFn: () => fetchOperatorInputs(target), enabled: title === "Credentials" });
  const { data: plan } = useQuery({ queryKey: ["apply-plan", target], queryFn: () => fetchApplyPlan(target), enabled: title === "Apply" });
  useLayoutEffect(() => {
    if (title !== "Apply") return;
    document.querySelector<HTMLButtonElement>('[data-testid="plan-summary"] button')?.setAttribute("data-testid", "apply-confirm");
  }, [title, data?.status, plan?.items.length]);
  const [values, setValues] = useState<Record<string, string>>({});
  const [provisioning, setProvisioning] = useState<string | null>(null);
  const [provisionError, setProvisionError] = useState<string | null>(null);
  const [applyState, setApplyState] = useState<ApplyRunView | null>(null);
  const [applyError, setApplyError] = useState<string | null>(null);
  const [applyReconnecting, setApplyReconnecting] = useState(false);
  const [cancelling, setCancelling] = useState(false);
  const [acknowledging, setAcknowledging] = useState(false);
  const [acknowledgeError, setAcknowledgeError] = useState<string | null>(null);
  const applyController = useRef<AbortController | null>(null);
  const observationGeneration = useRef(0);
  const idempotencyKey = useRef<string>(globalThis.crypto?.randomUUID?.() ?? `onboarding-${Date.now()}-${Math.random().toString(36).slice(2)}`);

  useEffect(() => () => {
    observationGeneration.current += 1;
    applyController.current?.abort();
  }, []);

  const observeApplyRun = async (accepted: GetApplyRunResponse) => {
    if (title !== "Apply") return;
    const generation = observationGeneration.current;
    const update = (current: GetApplyRunResponse) => {
      if (generation !== observationGeneration.current) return;
      window.localStorage.setItem(applyRunStorageKey(target), current.runId);
      setApplyState(applyRunView(current));
    };
    update(accepted);
    const settled = await pollApplyRun(accepted, {
      fetchStatus: (runID) => fetchApplyRun(runID, target),
      onUpdate: update,
      onConnectionChange: (connected) => {
        if (generation === observationGeneration.current) setApplyReconnecting(!connected);
      },
      wait: (ms) => new Promise((resolve) => window.setTimeout(resolve, ms)),
      now: () => Date.now(),
    });
    if (generation !== observationGeneration.current) return;
    setApplyState(applyRunView(settled));
    setApplyReconnecting(false);
  };

  useEffect(() => {
    if (title !== "Apply") return;
    const storedRunID = window.localStorage.getItem(applyRunStorageKey(target));
    if (!storedRunID) return;
    let cancelled = false;
    void fetchApplyRun(storedRunID, target)
      .then((run) => {
        if (!cancelled) void observeApplyRun(run);
      })
      .catch(() => {
        if (!cancelled) setApplyError(i18n.t("onboarding.readiness.applyError"));
      });
    return () => {
      cancelled = true;
      observationGeneration.current += 1;
    };
  }, [target, title]);

  const provision = async (logicalID: string, field: string) => {
    const key = `${logicalID}/${field}`;
    const value = values[key]?.trim() ?? "";
    if (!value) return;
    setProvisioning(key);
    setProvisionError(null);
    try {
      await provisionCredential({ logical_id: logicalID, field, value }, target);
      setValues((current) => ({ ...current, [key]: "" }));
      await refetch();
    } catch {
      setProvisionError(i18n.t("onboarding.readiness.provisioningError"));
    } finally {
      setProvisioning(null);
    }
  };

  const apply = async () => {
    if (applyState && isApplyRunActive(applyState.status)) return;
    setApplyError(null);
    applyController.current?.abort();
    const controller = new AbortController();
    applyController.current = controller;
    try {
      if (!plan?.plan_id || !plan.plan_digest || !plan.revision) throw new Error("apply plan is not reviewable");
      const review = await reviewApply({ target, plan_id: plan.plan_id, plan_digest: plan.plan_digest, expected_revision: plan.revision });
      const accepted = await startApply({
        target,
        plan_id: review.planId,
        plan_digest: review.planDigest,
        expected_revision: review.revision,
        consent_receipt_id: review.consentReceiptId,
        idempotency_key: idempotencyKey.current,
      });
      if (!accepted.run) throw new Error("apply response did not include a run");
      await observeApplyRun(accepted.run);
      setApplyReconnecting(false);
      await refetch();
    } catch {
      setApplyReconnecting(false);
      setApplyError(i18n.t("onboarding.readiness.applyError"));
    }
  };

  const cancel = async () => {
    if (!applyState || !isApplyRunActive(applyState.status)) return;
    setCancelling(true);
    setApplyError(null);
    try {
      const response = await cancelApply(applyState.run_id, target);
      if (response.run) setApplyState(applyRunView(response.run));
    } catch {
      setApplyError(i18n.t("onboarding.readiness.cancelError"));
    } finally {
      setCancelling(false);
    }
  };

  const acceptDegraded = async () => {
    const digest = data?.degraded_digest;
    if (!digest) return;
    setAcknowledging(true);
    setAcknowledgeError(null);
    try {
      await acknowledgeDegraded(digest, target);
      await refetch();
    } catch {
      setAcknowledgeError(i18n.t("onboarding.readiness.acknowledgementError"));
    } finally {
      setAcknowledging(false);
    }
  };

  const heading = title === "Apply" ? i18n.t("onboarding.readiness.heading.apply") : title === "Validation" ? i18n.t("onboarding.readiness.heading.validation") : i18n.t("onboarding.readiness.heading.credentials");
  const intro = title === "Apply"
    ? i18n.t("onboarding.readiness.intro.apply")
    : title === "Validation"
      ? i18n.t("onboarding.readiness.intro.validation")
      : i18n.t("onboarding.readiness.intro.credentials");
  return <div data-testid="step-readiness" className={`readiness-surface readiness-surface--${title.toLowerCase()}`}>
    <p className="surface-eyebrow">{title === "Apply" ? i18n.t("onboarding.readiness.eyebrow.apply") : title === "Validation" ? i18n.t("onboarding.readiness.eyebrow.validation") : i18n.t("onboarding.readiness.eyebrow.credentials")}</p>
    <h1 className="text-xl font-semibold sm:text-2xl">{heading}</h1>
    <p className="mt-2 text-sm text-muted">{intro}</p>
    {isLoading && <p className="mt-6 flex items-center gap-2 text-muted" role="status"><Loader2 className="h-4 w-4 animate-spin" />{i18n.t("onboarding.readiness.loading")}</p>}
    {error && <p className="mt-6 text-danger" role="alert">{i18n.t("onboarding.readiness.error")}</p>}
    {provisionError && <p className="mt-4 text-sm text-danger" role="alert">{provisionError}</p>}
    {applyError && <p className="mt-4 text-sm text-danger" role="alert">{applyError}</p>}
    {applyReconnecting && <p className="mt-4 text-sm text-muted" role="status" data-testid="apply-reconnecting">{i18n.t("onboarding.readiness.reconnecting")}</p>}
    {applyState && <div className="mt-4 rounded-lg border border-muted bg-surface-muted p-3 text-sm" data-testid="apply-state" role="status">
      <p className="font-medium">{cancelling ? i18n.t("onboarding.readiness.cancellationRequested") : applyRunLabel(applyState.status)}</p>
      <p className="mt-1 text-xs text-muted">{i18n.t("onboarding.readiness.applyRunId")}: <code>{applyState.run_id}</code></p>
      {isApplyRunActive(applyState.status) && <Button data-testid="apply-cancel" type="button" variant="secondary" disabled={cancelling} onClick={() => { void cancel(); }}>{cancelling ? i18n.t("onboarding.readiness.cancelling") : i18n.t("onboarding.readiness.cancelApply")}</Button>}
    </div>}
    {title === "Credentials" && capabilities && <CapabilityActions statuses={capabilities.capabilities} onRefresh={() => { void Promise.all([refetchCapabilities(), refetch()]); }} />}
    {title === "Credentials" && operatorInputs && <SchemaQuestionSet target={target} requests={operatorInputs.requests} />}
    {title === "Credentials" && data?.credential_diagnosis?.provider && <div data-testid="backend-diagnosis" role="status" className="mt-4 rounded-lg border border-warning/30 bg-warning-surface p-3 text-sm"><p className="font-medium text-warning">{i18n.t("onboarding.readiness.providerDiagnosis")}</p><p className="mt-1 text-foreground">{data.credential_diagnosis.provider.condition} · {data.credential_diagnosis.provider.backend}</p>{data.credential_diagnosis.provider.explanation && <p className="mt-1 text-xs text-muted">{data.credential_diagnosis.provider.explanation}</p>}{data.credential_diagnosis.provider.fix && <p className="mt-1 text-xs text-primary-soft">{i18n.t("onboarding.readiness.next")} {data.credential_diagnosis.provider.fix}</p>}{data.credential_diagnosis.provider.write_condition && <p className="mt-1 text-xs text-muted">{i18n.t("onboarding.readiness.writeReachability")} {data.credential_diagnosis.provider.write_condition}. {data.credential_diagnosis.provider.write_fix}</p>}</div>}
    {title === "Apply" && <section className="mt-4" aria-label={i18n.t("onboarding.readiness.applyPlan")}>
      <ApplyPlanDisclosure items={plan?.items ?? []} />
      <PlanSummary
        kicker={i18n.t("onboarding.readiness.proposedSetup")}
        title={i18n.t("onboarding.readiness.planTitle")}
        note={i18n.t("onboarding.readiness.planNote")}
        facts={[
          { value: String(plan?.items.length ?? 0), label: i18n.t("onboarding.readiness.items") },
          { value: String(plan?.items.filter((item) => item.required).length ?? 0), label: i18n.t("onboarding.readiness.required") },
        ]}
        items={(plan?.items ?? []).map((item) => ({ label: item.name, implied: !item.required }))}
        onAccept={() => { void apply(); }}
        acceptLabel={i18n.t("onboarding.readiness.applySelection")}
      />
    </section>}
    {title === "Credentials" && !data && <div data-testid="credential-card" role="group" className="mt-4 list-none rounded-lg border border-muted bg-surface-muted p-3 text-sm"><p data-testid="credential-purpose" role="note" className="font-medium">{i18n.t("onboarding.readiness.declaredCredentials")}</p><span data-testid="credential-status" role="status" className="ml-2 text-muted">{i18n.t("onboarding.readiness.loadingDescriptors")}</span><a data-testid="credential-obtain-link" className="mt-1 inline-flex min-h-11 items-center rounded px-2 text-xs text-primary-soft underline" href="/setup/credentials#credential-guidance">{i18n.t("onboarding.readiness.credentialGuidance")}</a><div className="mt-3 flex flex-col gap-2 sm:flex-row"><SecureValueInput testId="credential-input" aria-label={i18n.t("onboarding.readiness.credentialValue")} autoComplete="off" disabled revealable={false} className="min-w-0 flex-1" /><Button data-testid="credential-save" type="button" disabled className="rounded-md bg-primary px-3 py-2 text-sm font-medium text-on-primary">{i18n.t("onboarding.readiness.saveSecurely")}</Button></div></div>}
      {data && <>
      {title !== "Apply" && <div className="mt-6 flex items-center gap-2" role="status" data-testid="readiness-summary">{data.status === "ready" ? <CheckCircle2 className="h-5 w-5 text-primary" /> : <CircleAlert className="h-5 w-5 text-warning" />}<span className={data.status === "ready" ? "text-primary" : "text-warning"}>{data.status === "ready" ? i18n.t("onboarding.readiness.ready") : i18n.t("onboarding.readiness.actionRequired", { status: data.status })}</span></div>}
      {title === "Validation" && <section className="mt-5 rounded-xl border border-muted bg-surface-muted p-4" aria-label={i18n.t("onboarding.readiness.readinessTimeline")} data-testid="readiness-timeline">
        <h2 className="mb-3 text-sm font-semibold">{i18n.t("onboarding.readiness.setupProgress")}</h2>
        <Timeline events={[{ label: i18n.t("onboarding.readiness.selectionReviewed"), detail: i18n.t("onboarding.readiness.selectionDetail") }, { label: i18n.t("onboarding.readiness.installValidated"), detail: data.status === "ready" ? i18n.t("onboarding.readiness.checksReady") : i18n.t("onboarding.readiness.reviewActions") }]} />
      </section>}
      {applyState && <>
        <div data-testid="apply-progress" role="progressbar" aria-label={i18n.t("onboarding.readiness.applyProgress")}>
          <RunLadder
            runId={applyState.run_id}
            reconnecting={applyReconnecting}
            resumeNote={i18n.t("onboarding.readiness.safeToClose")}
            phases={[{ id: "apply", label: i18n.t("onboarding.readiness.applyPhase"), state: applyState.status === ApplyRunState.FAILED || applyState.status === ApplyRunState.PARTIALLY_APPLIED || applyState.status === ApplyRunState.CANCELLED || applyState.status === ApplyRunState.INDETERMINATE ? "failed" : applyState.status === ApplyRunState.APPLIED || applyState.status === ApplyRunState.ALREADY_SATISFIED ? "complete" : "active" }]}
            items={applyState.items.map((item) => ({ name: item.name, outcome: item.outcome === "success" || item.outcome === "succeeded" || item.outcome === "applied" || item.outcome === "already satisfied" ? "succeeded" : item.outcome === "failed" || item.outcome === "blocked" ? "failed" : item.outcome === "applying" || item.outcome === "running" ? "running" : "pending", error: item.error }))}
          />
        </div>
        <div data-testid="apply-report" role="table" aria-label={i18n.t("onboarding.readiness.applyReport")} />
      </>}
      {applyState && [ApplyRunState.PARTIALLY_APPLIED, ApplyRunState.FAILED, ApplyRunState.CANCELLED].includes(applyState.status) && <><p data-testid="skipped-note" role="note" className="mt-3 text-sm text-warning">{i18n.t("onboarding.readiness.skipped")}</p><Button data-testid="retry" type="button" variant="secondary" onClick={() => { void apply(); }}>{i18n.t("onboarding.readiness.applyAgain")}</Button></>}
      {title === "Validation" && <>
        <BlockerList title={i18n.t("onboarding.readiness.blocking")} testID="readiness-blockers" items={data.blockers ?? []} tone="danger" />
        <BlockerList title={i18n.t("onboarding.readiness.degraded")} testID="readiness-degraded" items={data.degraded ?? []} tone="warning" />
        {acknowledgeError && <p className="mt-3 text-sm text-danger" role="alert">{acknowledgeError}</p>}
        <div className="mt-4 flex flex-wrap gap-3">
          <Button data-testid="recheck" type="button" variant="secondary" onClick={() => { void refetch(); }}>{i18n.t("onboarding.readiness.recheck")}</Button>
          {(data.blockers ?? []).length === 0 && (data.degraded ?? []).length > 0 && !data.degraded_acknowledged && <Button data-testid="readiness-continue-degraded" type="button" variant="secondary" disabled={acknowledging} onClick={() => { void acceptDegraded(); }}>{acknowledging ? i18n.t("onboarding.readiness.recording") : i18n.t("onboarding.readiness.acceptDegraded")}</Button>}
          {(data.degraded ?? []).length > 0 && data.degraded_acknowledged && <p role="status" className="text-sm text-warning">{i18n.t("onboarding.readiness.degradedAccepted")}</p>}
        </div>
        {(data.blockers ?? []).length > 0 && <p data-testid="finish-blocked" role="note" className="mt-3 text-sm text-danger">{i18n.t("onboarding.readiness.finishBlocked")}</p>}
      </>}
      {title === "Credentials" && <ul className="mt-4 space-y-2">{data.credentials.map((credential) => {
        const key = `${credential.logical_id}/${credential.field}`;
        const componentSupplied = credential.provisioning === "derived" || credential.provisioning === "generated";
        const canProvision = title === "Credentials" && !componentSupplied && credential.status !== "configured";
        return <li key={key} data-testid="credential-card" className="rounded-lg border border-muted bg-surface-muted p-3 text-sm"><p data-testid="credential-purpose" role="note" className="font-medium">{credential.label || credential.field}</p><span data-testid="credential-status" role="status" className="ml-2 text-muted">{credential.required ? i18n.t("onboarding.readiness.requiredStatus") : i18n.t("onboarding.readiness.optionalStatus")} · {credential.status}</span>{credential.provisioning === "derived" && <p className="mt-1 text-xs text-muted">{i18n.t("onboarding.readiness.derived", { source: credential.derived_from || "the owning component" })}</p>}{credential.provisioning === "generated" && <p className="mt-1 text-xs text-muted">{i18n.t("onboarding.readiness.generated")}</p>}{credential.description && <p className="mt-1 text-xs text-muted">{credential.description}</p>}<a data-testid="credential-obtain-link" className="mt-1 inline-flex min-h-11 items-center rounded px-2 text-xs text-primary-soft underline" href={credential.obtain_url || "/setup/credentials#credential-guidance"}>{credential.obtain_url ? i18n.t("onboarding.readiness.obtain") : i18n.t("onboarding.readiness.credentialGuidance")}</a>{credential.detail && <p className="mt-1 text-xs text-muted">{credential.detail}</p>}{canProvision && <div className="mt-3 flex flex-col gap-2 sm:flex-row"><SecureValueInput testId="credential-input" aria-label={i18n.t("onboarding.readiness.valueFor", { label: credential.label || credential.field })} revealable={false} autoComplete="off" value={values[key] ?? ""} onValueChange={(value) => setValues((current) => ({ ...current, [key]: value }))} className="min-w-0 flex-1" /><Button data-testid="credential-save" type="button" disabled={!values[key]?.trim() || provisioning === key} onClick={() => { void provision(credential.logical_id, credential.field); }} className="rounded-md bg-primary px-3 py-2 text-sm font-medium text-on-primary disabled:opacity-50">{provisioning === key ? i18n.t("onboarding.readiness.saving") : i18n.t("onboarding.readiness.saveSecurely")}</Button></div>}</li>;
      })}</ul>}
      {title === "Validation" && <><ReadinessGroup title={i18n.t("onboarding.readiness.hostRequirements")} items={data.hosts} /><ReadinessGroup title={i18n.t("onboarding.readiness.integrations")} items={data.integrations.filter((item) => item.category === "integration")} /><ReadinessGroup title={i18n.t("onboarding.readiness.systemChecks")} items={data.integrations.filter((item) => item.category === "system")} /></>}
    </>}
  </div>;
}

function SchemaQuestionSet({ target, requests }: { target: string; requests: OperatorInputRequest[] }) {
  const [secretValues, setSecretValues] = useState<Record<string, string>>({});
  const [declined, setDeclined] = useState<Record<string, boolean>>({});
  const [message, setMessage] = useState<string | null>(null);
  const secrets = requests.filter((request) => inputKindName(request.kind) === "secret");
  const regular = requests.filter((request) => inputKindName(request.kind) !== "secret");
  const adapted = requests.map((request) => ({
    id: request.id,
    kind: inputKindName(request.kind),
    label: request.title,
    description: request.description,
    required: request.required,
    defaultValue: request.defaultValue,
    options: request.options?.map((value) => ({ value, label: value })),
    candidates: request.candidates?.map((candidate) => ({ label: candidate.label, value: candidate.id, status: candidate.status, risk: candidate.risk, remediation: candidate.remediation })),
    validation: request.validation,
  }));
  const fields = toGeneratedFields(adapted.filter((request) => request.kind !== "secret") as OperatorInput[]) as GeneratedField[];
  const submit = async (values: Record<string, unknown>) => {
    const answers = requests.map((request) => {
      const answer = create(AnswerSchema, { requestId: request.id, value: String(inputKindName(request.kind) === "secret" ? secretValues[request.id] ?? "" : values[request.id] ?? "") });
      // The additive declined field is present in the governed generated
      // schema. Keep this cast compatible with an older projected node_modules
      // copy while the source package refreshes.
      (answer as unknown as { declined?: boolean }).declined = declined[request.id] === true;
      return answer;
    }) as unknown as Answer[];
    try {
      await resolveOperatorInputs(answers as Answer[], target);
      setMessage(i18n.t("onboarding.readiness.answersSubmitted"));
    } catch {
      setMessage(i18n.t("onboarding.readiness.answersRejected"));
    }
  };
  if (requests.length === 0) return null;
  return <section className="mt-4 rounded-lg border border-muted bg-surface-muted p-4" data-testid="target-question-set" aria-label={i18n.t("onboarding.readiness.outstandingQuestions", { target })}>
    <h2 className="text-lg font-medium">{i18n.t("onboarding.readiness.outstandingQuestions", { target: "" }).replace(/\s+$/, "")} <code>{target}</code></h2>
    <p className="mt-1 text-sm text-muted">{i18n.t("onboarding.readiness.schemaDescription")}</p>
    {secrets.map((request) => <SecureValueInput key={request.id} name={request.id} label={request.title} value={secretValues[request.id] ?? ""} onValueChange={(value: string) => setSecretValues((current) => ({ ...current, [request.id]: value }))} autoComplete="new-password" revealable={false} />)}
    {regular.length > 0 && <GeneratedForm mode="uncontrolled" fields={fields} onSubmit={submit} submitLabel={i18n.t("onboarding.readiness.submitAnswers")} />}
    {requests.filter((request) => (request as unknown as { declinable?: boolean }).declinable).map((request) => <Checkbox key={`decline-${request.id}`} className="mt-3" checked={declined[request.id] === true} onCheckedChange={(checked) => setDeclined((current) => ({ ...current, [request.id]: checked }))} label={<>{i18n.t("onboarding.capabilities.decline")} — {request.title}{declined[request.id] ? ` · ${i18n.t("onboarding.capabilities.declined")}` : ""}</>} />)}
    {secrets.length > 0 && <Button type="button" className="mt-3" onClick={() => { void submit({}); }}>{i18n.t("onboarding.readiness.submitSecretAnswers")}</Button>}
    {message && <p className="mt-3 text-sm" role="status" data-testid="target-question-status">{message}</p>}
  </section>;
}

function inputKindName(kind: OperatorInputKind): "secret" | "choice" | "confirm" | "path" | "enum" | "boolean" | "duration" | "confirmation" {
  switch (kind) {
    case OperatorInputKind.SECRET: return "secret";
    case OperatorInputKind.CHOICE: return "choice";
    case OperatorInputKind.CONFIRM: return "confirm";
    case OperatorInputKind.PATH: return "path";
    case OperatorInputKind.ENUM: return "enum";
    case OperatorInputKind.BOOLEAN: return "boolean";
    case OperatorInputKind.DURATION: return "duration";
    case OperatorInputKind.CONFIRMATION: return "confirmation";
    default: return "choice";
  }
}

/**
 * BlockerList shows the named reasons configuration is not complete. It renders
 * nothing when the list is empty, so a clean verdict stays quiet.
 */
function BlockerList({ title, testID, items, tone }: { title: string; testID: string; items: CompletionBlocker[]; tone: "danger" | "warning" }) {
  if (items.length === 0) return null;
  const border = tone === "danger" ? "border-danger/40" : "border-warning/40";
  const text = tone === "danger" ? "text-danger" : "text-warning";
  return <section className="mt-4" aria-label={title}>
    <h2 className={`text-lg font-medium ${text}`}>{title}</h2>
    <ul data-testid={testID} role="list" className={`mt-2 space-y-2 rounded-lg border ${border} p-3`}>
      {items.map((item) => <li key={`${item.kind}-${item.name}`} data-testid={`${testID}-item`} className="text-sm">
        <span className="font-medium">{item.kind}: {item.name}</span>
        <p className="mt-1 text-xs text-muted">{item.reason}</p>
        <p className="mt-1 text-xs text-primary-soft">{i18n.t("onboarding.readiness.nextAction", { remediation: item.remediation })}</p>
      </li>)}
    </ul>
  </section>;
}

function ReadinessGroup({ title, items }: { title: string; items: ReadinessItem[] }) {
  return <section className="mt-6"><h2 className="text-lg font-medium">{title}</h2>{items.length === 0 ? <p className="mt-2 text-sm text-muted">{i18n.t("onboarding.readiness.noRequirements")}</p> : <ul className="mt-3 space-y-2">{items.map((item) => <li key={`${item.kind ?? "item"}-${item.name}`} data-testid="readiness-item" className="rounded-lg border border-muted bg-surface-muted p-3 text-sm"><span className="font-medium">{item.name}</span><span className="ml-2 text-muted">{item.kind ? `${item.kind} · ` : ""}{item.status}{item.required ? ` · ${i18n.t("onboarding.readiness.requiredStatus").toLowerCase()}` : ""}</span>{item.detail && <p className="mt-1 text-xs text-muted">{item.detail}</p>}{item.remediation && <p data-testid="remediation" className="mt-1 text-xs text-primary-soft">{i18n.t("onboarding.readiness.nextAction", { remediation: item.remediation })}</p>}</li>)}</ul>}</section>;
}
