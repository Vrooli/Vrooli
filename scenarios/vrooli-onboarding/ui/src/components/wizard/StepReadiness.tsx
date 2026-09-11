import { useQuery } from "@tanstack/react-query";
import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { CheckCircle2, CircleAlert, Loader2, Save, Search } from "lucide-react";
import { Code, ConnectError } from "@connectrpc/connect";
import { fetchCapabilities, type CapabilityStatus } from "../../api/capabilities";
import { fetchCredentials, provisionCredential, type CredentialListItem } from "../../api/credentials";
import { cancelApply, fetchApplyPlan, fetchApplyRun, reviewApply, startApply } from "../../api/apply";
import { acknowledgeDegraded, fetchReadiness } from "../../api/readiness";
import { fetchOperatorInputs, resolveOperatorInputs } from "../../api/operatorinputs";
import { loginAuthenticator } from "../../api/auth";
import { pollApplyRun } from "../../lib/applyRun";
import { ApplyRunState, ApplyStepState } from "@vrooli/proto-types/vrooli-onboarding/v1/apply/apply_pb";
import type { GetApplyRunResponse } from "@vrooli/proto-types/vrooli-onboarding/v1/apply/apply_pb";
import type { CompletionBlocker, ReadinessItem, ReadinessResponse } from "../../api/readiness";
import { create } from "@bufbuild/protobuf";
import { OperatorInputKind, type OperatorInputRequest } from "@vrooli/proto-types/setup/v1/operator_input_pb";
import { AnswerSchema, type Answer } from "@vrooli/proto-types/vrooli-onboarding/v1/operatorinputs/operatorinputs_pb";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Alert } from "@vrooli/react-component-library/Alert/1";
import { AlertDialog } from "@vrooli/react-component-library/AlertDialog/2";
import { Checkbox } from "@vrooli/react-component-library/Checkbox/1";
import { ApplyPlanDisclosure } from "./ApplyPlanDisclosure";
import { CapabilityActions } from "./CapabilityActions";
import { GeneratedForm } from "@vrooli/react-component-library/GeneratedForm/1";
import { toGeneratedFields, type OperatorInput } from "@vrooli/react-component-library/ValidationAdapter/1";
import type { GeneratedField } from "@vrooli/react-component-library/GeneratedForm/1";
import { PlanSummary } from "@vrooli/react-component-library/PlanSummary/0";
import { RunLadder } from "@vrooli/react-component-library/RunLadder/0";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { Timeline } from "@vrooli/react-component-library/Timeline/1";
import { Tabs } from "@vrooli/react-component-library/Tabs/1";
import { SetupTask, type SetupTaskStatus } from "@vrooli/react-component-library/SetupTask/0";
import { Input } from "@vrooli/react-component-library/Input/1";
import { InputGroup } from "@vrooli/react-component-library/InputGroup";
import { Select } from "@vrooli/react-component-library/Select/1";
import { PasswordInput as SecureValueInput } from "@vrooli/react-component-library/PasswordInput/2";
import { i18n } from "../../i18n";

type ReadinessSurfaceProps = { title: "Credentials" | "Apply" | "Validation"; target?: string };

type ProvisionError = {
  kind: "operator" | "generic";
  logicalID: string;
  field: string;
  recoveryURL?: string;
  authSource?: string;
  authProviders?: string[];
};

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
  const [credentialDetailsOpen, setCredentialDetailsOpen] = useState(false);
  const [credentialConfigTab, setCredentialConfigTab] = useState("provider");
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["readiness", target],
    queryFn: () => fetchReadiness(target),
    staleTime: 15_000,
    retry: false,
    refetchOnWindowFocus: false,
  });
  const { data: credentialInventory, isLoading: credentialsLoading, error: credentialsError, refetch: refetchCredentials } = useQuery({
    queryKey: ["credential-inventory", target],
    queryFn: () => fetchCredentials(target),
    enabled: title === "Credentials",
    staleTime: 60_000,
    retry: false,
    refetchOnWindowFocus: false,
  });
  // Configuration work is intentionally lazy. The credential list is useful
  // on first paint; provider actions and target questions are only needed
  // after the operator opens the setup workflow.
  const { data: capabilities, isLoading: capabilitiesLoading, refetch: refetchCapabilities } = useQuery({
    queryKey: ["capabilities", target],
    queryFn: () => fetchCapabilities(target),
    enabled: title === "Credentials" && credentialDetailsOpen,
    staleTime: 15_000,
    retry: false,
    refetchOnWindowFocus: false,
  });
  const { data: operatorInputs, isLoading: operatorInputsLoading } = useQuery({
    queryKey: ["operator-inputs", target],
    queryFn: () => fetchOperatorInputs(target),
    enabled: title === "Credentials" && credentialDetailsOpen,
    staleTime: 15_000,
    retry: false,
    refetchOnWindowFocus: false,
  });
  const { data: plan } = useQuery({ queryKey: ["apply-plan", target], queryFn: () => fetchApplyPlan(target), enabled: title === "Apply" });
  useLayoutEffect(() => {
    if (title !== "Apply") return;
    document.querySelector<HTMLButtonElement>('[data-testid="plan-summary"] button')?.setAttribute("data-testid", "apply-confirm");
  }, [title, data?.status, plan?.items.length]);
  const [values, setValues] = useState<Record<string, string>>({});
  const [provisioning, setProvisioning] = useState<string | null>(null);
  const [provisionError, setProvisionError] = useState<ProvisionError | null>(null);
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
      await Promise.all([refetch(), refetchCredentials()]);
    } catch (error) {
      setProvisionError({
        kind: error instanceof ConnectError && error.code === Code.Unauthenticated ? "operator" : "generic",
        logicalID,
        field,
        recoveryURL: error instanceof ConnectError ? error.metadata.get("Vrooli-Auth-Recovery-Url") ?? undefined : undefined,
        authSource: error instanceof ConnectError ? error.metadata.get("Vrooli-Auth-Source") ?? undefined : undefined,
        authProviders: error instanceof ConnectError ? (error.metadata.get("Vrooli-Auth-Providers") ?? "").split(",").map((value) => value.trim()).filter(Boolean) : undefined,
      });
    } finally {
      setProvisioning(null);
    }
  };

  const retryProvision = () => {
    if (!provisionError) return;
    void provision(provisionError.logicalID, provisionError.field);
  };

  const openAuthRecovery = () => {
    if (typeof window === "undefined") return;
    // Cloudflare's direct `/cdn-cgi/access/login` URL is only valid when the
    // edge generated it for a protected request. Replaying the current
    // protected page lets Cloudflare create that signed login handoff and
    // preserves the exact setup route for the post-login redirect.
    if (provisionError?.authSource === "cloudflare_access") {
      window.location.assign(window.location.href);
      return;
    }
    if (provisionError?.recoveryURL) {
      window.location.assign(provisionError.recoveryURL);
      return;
    }
    window.location.assign(window.location.href);
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
  const readinessCredentials = new Map((data?.credentials ?? []).map((credential) => [`${credential.logical_id}/${credential.field}`, credential]));
  const credentials = (credentialInventory?.credentials ?? data?.credentials ?? []).map((credential) => {
    const readinessCredential = readinessCredentials.get(`${credential.logical_id}/${credential.field}`);
    return readinessCredential ? { ...credential, status: readinessCredential.status, detail: readinessCredential.detail, consumer_scope: readinessCredential.consumer_scope, evidence_status: readinessCredential.evidence_status, evidence_detail: readinessCredential.evidence_detail, evidence_next_action: readinessCredential.evidence_next_action, evidence_credential_version: readinessCredential.evidence_credential_version, provider_state: readinessCredential.provider_state, provider_detail: readinessCredential.provider_detail } : credential;
  });
  const credentialCount = credentials.length;
  const outstandingCredentialCount = credentials.filter(credentialNeedsAttention).length;
  const hasCredentialInventory = credentialInventory !== undefined || data !== undefined;
  // Readiness and credential inventory are intentionally separate requests.
  // If readiness wins the race, its contextual credential projection is enough
  // to render the actual rows while the richer inventory catches up. This
  // keeps the page stable when a native-store or source-inventory probe is
  // slow, without hiding that the richer request is still in flight.
  const credentialListLoading = credentialsLoading && credentialInventory === undefined && data === undefined;
  const credentialListError = Boolean(credentialsError) && credentialInventory === undefined && data === undefined;
  const credentialTone: "info" | "success" | "warning" | "danger" = error ? "danger" : isLoading ? "info" : data?.status === "ready" ? "success" : "warning";
  const credentialStatus = error
    ? i18n.t("onboarding.readiness.error")
    : isLoading
      ? i18n.t("onboarding.readiness.loading")
      : data?.status === "ready"
        ? i18n.t("onboarding.readiness.ready")
        : i18n.t("onboarding.readiness.actionRequired", { status: data?.status ?? "unknown" });
  return <div data-testid="step-readiness" className={`readiness-surface readiness-surface--${title.toLowerCase()}`}>
    <p className="surface-eyebrow">{title === "Apply" ? i18n.t("onboarding.readiness.eyebrow.apply") : title === "Validation" ? i18n.t("onboarding.readiness.eyebrow.validation") : i18n.t("onboarding.readiness.eyebrow.credentials")}</p>
    <h1 className="text-xl font-semibold sm:text-2xl">{heading}</h1>
    <p className="mt-2 text-sm text-muted">{intro}</p>
    {isLoading && title !== "Credentials" && <p className="mt-6 flex items-center gap-2 text-muted" role="status"><Loader2 className="h-4 w-4 animate-spin" />{i18n.t("onboarding.readiness.loading")}</p>}
    {error && title !== "Credentials" && <p className="mt-6 text-danger" role="alert">{i18n.t("onboarding.readiness.error")}</p>}
    {applyError && <p className="mt-4 text-sm text-danger" role="alert">{applyError}</p>}
    {applyReconnecting && <p className="mt-4 text-sm text-muted" role="status" data-testid="apply-reconnecting">{i18n.t("onboarding.readiness.reconnecting")}</p>}
    {applyState && <div className="mt-4 rounded-lg border border-muted bg-surface-muted p-3 text-sm" data-testid="apply-state" role="status">
      <p className="font-medium">{cancelling ? i18n.t("onboarding.readiness.cancellationRequested") : applyRunLabel(applyState.status)}</p>
      <p className="mt-1 text-xs text-muted">{i18n.t("onboarding.readiness.applyRunId")}: <code>{applyState.run_id}</code></p>
      {isApplyRunActive(applyState.status) && <Button data-testid="apply-cancel" type="button" variant="secondary" disabled={cancelling} onClick={() => { void cancel(); }}>{cancelling ? i18n.t("onboarding.readiness.cancelling") : i18n.t("onboarding.readiness.cancelApply")}</Button>}
    </div>}
    {title === "Credentials" && <>
      {provisionError && <CredentialRecoveryDialog error={provisionError} onDismiss={() => setProvisionError(null)} onOpenRecovery={openAuthRecovery} onRetry={retryProvision} />}
      <section className="credential-overview" data-testid="credential-overview" aria-labelledby="credential-overview-title">
        <div className="credential-overview__header">
          <div className="credential-overview__heading">
            <p className="surface-eyebrow">{i18n.t("onboarding.readiness.eyebrow.credentials")}</p>
            <h2 id="credential-overview-title">{i18n.t("onboarding.readiness.credentialSetup")}</h2>
            <p>{isLoading ? i18n.t("onboarding.readiness.credentialLoadingSummary") : data?.status === "ready" ? i18n.t("onboarding.readiness.credentialReadySummary") : i18n.t("onboarding.readiness.credentialAttentionSummary")}</p>
          </div>
          <StatusBadge tone={credentialTone}>{credentialStatus}</StatusBadge>
        </div>
        <div className="credential-overview__facts" aria-label={i18n.t("onboarding.readiness.credentialInputs")}>
          <div><strong>{hasCredentialInventory ? credentialCount : "—"}</strong><span>{i18n.t("onboarding.readiness.credentialInputs")}</span></div>
          <div><strong data-testid="credential-outstanding-count">{hasCredentialInventory ? outstandingCredentialCount : "—"}</strong><span>{i18n.t("onboarding.readiness.requiredStatus")}</span></div>
        </div>
        <div className="credential-overview__footer">
          <Button type="button" onClick={() => setCredentialDetailsOpen(true)}>{i18n.t("onboarding.readiness.reviewCredentialSetup")}</Button>
          {isLoading && <span className="credential-overview__loading" role="status"><Loader2 className="h-4 w-4 animate-spin" />{i18n.t("onboarding.readiness.loading")}</span>}
        </div>
        {error && <Alert tone="danger" title={i18n.t("onboarding.readiness.credentialSetup")} description={i18n.t("onboarding.readiness.credentialReadinessError")} className="credential-overview__alert" />}
      </section>
      <CredentialList
        credentials={credentials}
        checkedAt={data?.checked_at}
        loading={credentialListLoading}
        error={credentialListError}
        values={values}
        provisioning={provisioning}
        onValueChange={(key, value) => setValues((current) => ({ ...current, [key]: value }))}
        onProvision={(logicalID, field) => { void provision(logicalID, field); }}
      />
      <ResponsiveDialog
        open={credentialDetailsOpen}
        onOpenChange={setCredentialDetailsOpen}
        title={i18n.t("onboarding.readiness.credentialDetailsTitle")}
        ariaLabel={i18n.t("onboarding.readiness.credentialDetailsTitle")}
        closeLabel={i18n.t("onboarding.shell.close")}
        size="lg"
        contentPadding="comfortable"
        testId="credential-details-dialog"
      >
        <CredentialConfiguration
          data={data}
          error={Boolean(error)}
          capabilities={capabilities}
          capabilitiesLoading={capabilitiesLoading}
          operatorInputs={operatorInputs}
          operatorInputsLoading={operatorInputsLoading}
          activeTab={credentialConfigTab}
          onTabChange={setCredentialConfigTab}
          target={target}
          onRefresh={() => { void Promise.all([refetchCapabilities(), refetch()]); }}
        />
      </ResponsiveDialog>
    </>}
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
      {data && <>
      {title !== "Apply" && title !== "Credentials" && <div className="mt-6 flex items-center gap-2" role="status" data-testid="readiness-summary">{data.status === "ready" ? <CheckCircle2 className="h-5 w-5 text-primary" /> : <CircleAlert className="h-5 w-5 text-warning" />}<span className={data.status === "ready" ? "text-primary" : "text-warning"}>{data.status === "ready" ? i18n.t("onboarding.readiness.ready") : i18n.t("onboarding.readiness.actionRequired", { status: data.status })}</span></div>}
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
      {title === "Validation" && <><ReadinessGroup title={i18n.t("onboarding.readiness.hostRequirements")} items={data.hosts} /><ReadinessGroup title={i18n.t("onboarding.readiness.integrations")} items={data.integrations.filter((item) => item.category === "integration")} /><ReadinessGroup title={i18n.t("onboarding.readiness.systemChecks")} items={data.integrations.filter((item) => item.category === "system")} /></>}
    </>}
  </div>;
}

type Credential = CredentialListItem & { consumer_scope?: string; evidence_status?: string; evidence_detail?: string; evidence_next_action?: string; evidence_credential_version?: string; provider_state?: string; provider_detail?: string };

function credentialNeedsAttention(credential: Credential) {
  if (!credential.required || credential.status !== "configured") return credential.required && credential.status !== "configured";
  return ["unverified", "unknown", "pending", "verification_unknown"].includes(credential.evidence_status ?? "");
}

function formatCredentialCondition(condition?: string) {
  const value = condition?.trim().replace(/[_-]+/g, " ");
  if (!value) return i18n.t("onboarding.readiness.loading");
  return value.charAt(0).toUpperCase() + value.slice(1);
}

function CredentialList({
  credentials,
  checkedAt,
  loading,
  error,
  values,
  provisioning,
  onValueChange,
  onProvision,
}: {
  credentials: Credential[];
  checkedAt?: string;
  loading: boolean;
  error: boolean;
  values: Record<string, string>;
  provisioning: string | null;
  onValueChange: (key: string, value: string) => void;
  onProvision: (logicalID: string, field: string) => void;
}) {
  const [search, setSearch] = useState("");
  const [filter, setFilter] = useState("all");
  const [sort, setSort] = useState("attention");
  const checkedDate = checkedAt ? new Date(checkedAt) : undefined;
  const formattedCheckedAt = checkedDate && !Number.isNaN(checkedDate.getTime())
    ? new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(checkedDate)
    : undefined;
  const visibleCredentials = useMemo(() => {
    const query = search.trim().toLowerCase();
    const filtered = credentials.filter((credential) => {
      const stored = credential.status === "configured";
      const attention = credentialNeedsAttention(credential);
      const matchesFilter = filter === "all"
        || (filter === "attention" && attention)
        || (filter === "configured" && stored)
        || (filter === "optional" && !credential.required);
      if (!matchesFilter) return false;
      if (!query) return true;
      return [credential.label, credential.logical_id, credential.field, credential.resource, credential.description, credential.detail]
        .filter(Boolean)
        .some((value) => value?.toLowerCase().includes(query));
    });
    return filtered.sort((left, right) => {
      if (sort === "name") return (left.label || left.field).localeCompare(right.label || right.field);
      if (sort === "resource") return left.resource.localeCompare(right.resource) || (left.label || left.field).localeCompare(right.label || right.field);
      return Number(credentialNeedsAttention(right)) - Number(credentialNeedsAttention(left))
        || Number(right.required) - Number(left.required)
        || (left.label || left.field).localeCompare(right.label || right.field);
    });
  }, [credentials, filter, search, sort]);
  return <section className="credential-list" data-testid="credential-list" aria-labelledby="credential-list-title">
    <div className="credential-list__heading">
      <div>
        <p className="surface-eyebrow">{i18n.t("onboarding.readiness.credentialPurpose")}</p>
        <h2 id="credential-list-title">{i18n.t("onboarding.readiness.credentialInputs")}</h2>
        <p>{i18n.t("onboarding.readiness.credentialListDescription")}</p>
      </div>
      {!loading && !error && <StatusBadge tone={credentials.some(credentialNeedsAttention) ? "warning" : "success"}>{credentials.length}</StatusBadge>}
    </div>
    {!loading && !error && credentials.length > 0 && <div className="credential-list__controls" aria-label={i18n.t("onboarding.readiness.credentialControls")}>
      <InputGroup className="credential-list__search" size="lg" shape="rounded" testId="credential-search-group">
        <InputGroup.Adornment side="leading"><Search aria-hidden="true" /></InputGroup.Adornment>
        <InputGroup.Field>
          <Input type="search" data-testid="credential-search" aria-label={i18n.t("onboarding.readiness.credentialSearch")} placeholder={i18n.t("onboarding.readiness.credentialSearchPlaceholder")} value={search} onChange={(event) => setSearch(event.target.value)} />
        </InputGroup.Field>
      </InputGroup>
      <label className="credential-list__select"><span>{i18n.t("onboarding.readiness.credentialFilter")}</span><Select data-testid="credential-filter" aria-label={i18n.t("onboarding.readiness.credentialFilter")} value={filter} onValueChange={setFilter} options={[{ value: "all", label: i18n.t("onboarding.readiness.credentialFilterAll") }, { value: "attention", label: i18n.t("onboarding.readiness.credentialFilterAttention") }, { value: "configured", label: i18n.t("onboarding.readiness.credentialFilterConfigured") }, { value: "optional", label: i18n.t("onboarding.readiness.credentialFilterOptional") }]} /></label>
      <label className="credential-list__select"><span>{i18n.t("onboarding.readiness.credentialSort")}</span><Select data-testid="credential-sort" aria-label={i18n.t("onboarding.readiness.credentialSort")} value={sort} onValueChange={setSort} options={[{ value: "attention", label: i18n.t("onboarding.readiness.credentialSortAttention") }, { value: "name", label: i18n.t("onboarding.readiness.credentialSortName") }, { value: "resource", label: i18n.t("onboarding.readiness.credentialSortResource") }]} /></label>
    </div>}
    {loading && <ul className="credential-list__items" aria-label={i18n.t("onboarding.readiness.credentialLoading")} role="status">
      {["one", "two", "three"].map((key) => <li key={key} className="credential-skeleton" aria-hidden="true"><span /><span /><span /></li>)}
    </ul>}
    {!loading && error && <Alert tone="danger" title={i18n.t("onboarding.readiness.credentialSetup")} description={i18n.t("onboarding.readiness.credentialReadinessError")} />}
    {!loading && !error && credentials.length === 0 && <p className="credential-list__empty">{i18n.t("onboarding.readiness.credentialNoInputs")}</p>}
    {!loading && !error && credentials.length > 0 && <ul className="credential-list__items">{visibleCredentials.map((credential) => {
      const key = `${credential.logical_id}/${credential.field}`;
      const componentSupplied = credential.provisioning === "derived" || credential.provisioning === "generated";
      const stored = credential.status === "configured";
      const verified = credential.evidence_status === "verified";
      const canProvision = !componentSupplied && !stored;
      const checking = credential.status === "pending";
      const deferred = credential.status === "deferred";
      const evidenceLabel = deferred
        ? i18n.t("onboarding.readiness.credentialDeferred")
        : checking
        ? i18n.t("onboarding.readiness.credentialChecking")
        : verified
        ? stored
          ? `${i18n.t("onboarding.readiness.credentialStored")} · ${i18n.t("onboarding.readiness.credentialVerified")}`
          : i18n.t("onboarding.readiness.credentialVerified")
        : stored
          ? `${i18n.t("onboarding.readiness.credentialStored")} · ${i18n.t("onboarding.readiness.credentialVerificationPending")}`
          : i18n.t("onboarding.readiness.credentialVerificationUnavailable");
      const taskStatus: SetupTaskStatus = checking
        ? "checking"
        : credential.status === "unsupported"
          ? "unavailable"
          : !credential.required && !stored
            ? "optional"
            : verified
              ? "ready"
              : "needs_attention";
      return <li key={key} className="credential-list__item">
        <div data-testid="credential-status">
          <SetupTask
            title={credential.label || credential.field}
            purpose={credential.description}
            target={credential.logical_id}
            account={credential.resource}
            consumerScope={credential.consumer_scope}
            checkedAt={formattedCheckedAt}
            nextAction={credential.evidence_next_action}
            status={taskStatus}
            statusLabel={evidenceLabel}
            testId="credential-card"
            guidance={<a data-testid="credential-obtain-link" className="credential-list__link" href={credential.obtain_url || "/setup/credentials#credential-guidance"}>{credential.obtain_url ? i18n.t("onboarding.readiness.obtain") : i18n.t("onboarding.readiness.credentialGuidance")}</a>}
            actions={canProvision ? <InputGroup className="credential-list__form" size="lg" shape="rounded" testId="credential-entry-group">
              <InputGroup.Field>
                <Input data-testid="credential-input" type="password" aria-label={i18n.t("onboarding.readiness.valueFor", { label: credential.label || credential.field })} autoComplete="off" value={values[key] ?? ""} onChange={(event) => onValueChange(key, event.target.value)} className="min-w-0 flex-1" />
              </InputGroup.Field>
              <InputGroup.Segment side="trailing" emphasis="solid" aria-label={provisioning === key ? i18n.t("onboarding.readiness.saving") : i18n.t("onboarding.readiness.saveSecurely")} title={provisioning === key ? i18n.t("onboarding.readiness.saving") : i18n.t("onboarding.readiness.saveSecurely")} testId="credential-save" onClick={() => onProvision(credential.logical_id, credential.field)} disabled={!values[key]?.trim() || provisioning === key}>
                {provisioning === key ? <Loader2 className="credential-list__save-icon credential-list__save-icon--loading" aria-hidden="true" /> : <Save className="credential-list__save-icon" aria-hidden="true" />}
                <span className="sr-only">{provisioning === key ? i18n.t("onboarding.readiness.saving") : i18n.t("onboarding.readiness.saveSecurely")}</span>
              </InputGroup.Segment>
            </InputGroup> : undefined}
          >
            {credential.provisioning === "derived" && <p className="credential-list__item-copy">{i18n.t("onboarding.readiness.derived", { source: credential.derived_from || "the owning component" })}</p>}
            {credential.provisioning === "generated" && <p className="credential-list__item-copy">{i18n.t("onboarding.readiness.generated")}</p>}
            {credential.detail && <p className="credential-list__item-copy">{credential.detail}</p>}
            {credential.provider_detail && <p className="credential-list__item-copy" role="note">{credential.provider_detail}</p>}
            {credential.evidence_detail && <p className="credential-list__item-copy" role="note">{credential.evidence_detail}</p>}
          </SetupTask>
        </div>
      </li>;
    })}</ul>}
    {!loading && !error && credentials.length > 0 && visibleCredentials.length === 0 && <p className="credential-list__empty" role="status">{i18n.t("onboarding.readiness.credentialNoMatches")}</p>}
  </section>;
}

function CredentialRecoveryDialog({
  error,
  onDismiss,
  onOpenRecovery,
  onRetry,
}: {
  error: ProvisionError;
  onDismiss: () => void;
  onOpenRecovery: () => void;
  onRetry: () => void;
}) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [authError, setAuthError] = useState<string | null>(null);
  const [authBusy, setAuthBusy] = useState(false);
  const isOperator = error.kind === "operator";
  const usesAuthenticator = error.authSource === "scenario_authenticator" || error.authProviders?.includes("scenario_authenticator") === true;
  const recoveryLabel = error.recoveryURL
    ? usesAuthenticator
      ? i18n.t("onboarding.readiness.openAuthenticator")
      : i18n.t("onboarding.readiness.openAccessSignIn")
    : i18n.t("onboarding.readiness.refreshSession");
  const confirm = async () => {
    if (!usesAuthenticator) {
      onOpenRecovery();
      return;
    }
    if (!email.trim() || !password) {
      setAuthError(i18n.t("onboarding.readiness.authenticatorRequiredFields"));
      return;
    }
    setAuthBusy(true);
    setAuthError(null);
    try {
      await loginAuthenticator(email, password);
      setPassword("");
      onRetry();
      onDismiss();
    } catch (cause) {
      setAuthError(cause instanceof ConnectError && cause.code === Code.Unauthenticated
        ? i18n.t("onboarding.readiness.authenticatorInvalidCredentials")
        : i18n.t("onboarding.readiness.authenticatorUnavailable"));
    } finally {
      setAuthBusy(false);
    }
  };

  return <AlertDialog
    open
    title={isOperator ? i18n.t("onboarding.readiness.operatorRequiredTitle") : i18n.t("onboarding.readiness.provisioningErrorTitle")}
    description={isOperator
      ? usesAuthenticator
        ? i18n.t("onboarding.readiness.authenticatorDialogDescription")
        : i18n.t("onboarding.readiness.operatorRequiredDialogDescription")
      : i18n.t("onboarding.readiness.provisioningErrorDialogDescription")}
    status="error"
    errorMessage={authError ?? (isOperator && !usesAuthenticator ? i18n.t("onboarding.readiness.operatorAccessRecovery") : undefined)}
    cancelLabel={i18n.t("onboarding.readiness.closeRecovery")}
    confirmLabel={usesAuthenticator ? i18n.t("onboarding.readiness.signInAndRetry") : recoveryLabel}
    busy={authBusy}
    busyLabel={i18n.t("onboarding.readiness.signingIn")}
    closeLabel={i18n.t("onboarding.readiness.credentialRecoveryDialog")}
    onCancel={onDismiss}
    onConfirm={confirm}
    testIdPrefix="credential-recovery"
  >
    {usesAuthenticator ? <form className="credential-recovery__form" onSubmit={(event) => { event.preventDefault(); void confirm(); }}>
      <label className="credential-recovery__field">
        <span>{i18n.t("onboarding.readiness.authenticatorEmail")}</span>
        <Input autoComplete="username" type="email" value={email} onChange={(event) => setEmail(event.target.value)} />
      </label>
      <label className="credential-recovery__field">
        <span>{i18n.t("onboarding.readiness.authenticatorPassword")}</span>
        <SecureValueInput autoComplete="current-password" value={password} onValueChange={setPassword} revealLabel={i18n.t("onboarding.readiness.revealPassword")} concealLabel={i18n.t("onboarding.readiness.concealPassword")} />
      </label>
      <Button type="submit" size="sm" variant="secondary" disabled={authBusy} data-testid="credential-recovery-retry">
        {i18n.t("onboarding.readiness.retryProvisioning")}
      </Button>
    </form> : <Button type="button" size="sm" variant="secondary" onClick={onRetry} data-testid="credential-recovery-retry">
      {i18n.t("onboarding.readiness.retryProvisioning")}
    </Button>}
  </AlertDialog>;
}

function CredentialConfiguration({
  data,
  error,
  capabilities,
  capabilitiesLoading,
  operatorInputs,
  operatorInputsLoading,
  activeTab,
  onTabChange,
  target,
  onRefresh,
}: {
  data?: ReadinessResponse;
  error: boolean;
  capabilities?: { capabilities: CapabilityStatus[] };
  capabilitiesLoading: boolean;
  operatorInputs?: { requests: OperatorInputRequest[] };
  operatorInputsLoading: boolean;
  activeTab: string;
  onTabChange: (tab: string) => void;
  target: string;
  onRefresh: () => void;
}) {
  const tabs = [
    { id: "provider", label: i18n.t("onboarding.readiness.credentialConfigProvider") },
    { id: "actions", label: i18n.t("onboarding.readiness.credentialActions") },
    { id: "questions", label: i18n.t("onboarding.readiness.credentialQuestions") },
  ];
  const provider = data?.credential_diagnosis?.provider;
  return <div className="credential-config" data-testid="credential-configuration">
    <p className="credential-details__intro">{i18n.t("onboarding.readiness.credentialConfigurationDescription")}</p>
    <div className="credential-config__mobile-tabs"><Tabs ariaLabel={i18n.t("onboarding.readiness.credentialConfigurationSections")} items={tabs} active={activeTab} onChange={onTabChange} density="compact" variant="segmented" testId="credential-config-tabs" /></div>
    <div className="credential-config__layout">
      <nav className="credential-config__sidebar" aria-label={i18n.t("onboarding.readiness.credentialConfigurationSections")}>
        {tabs.map((tab) => <button key={tab.id} type="button" className={activeTab === tab.id ? "is-active" : ""} aria-current={activeTab === tab.id ? "page" : undefined} onClick={() => onTabChange(tab.id)}>{tab.label}</button>)}
      </nav>
      <section className="credential-config__panel" role="tabpanel" aria-label={tabs.find((tab) => tab.id === activeTab)?.label}>
        {activeTab === "provider" && <>
          <div className="credential-config__panel-heading"><p className="surface-eyebrow">{i18n.t("onboarding.readiness.credentialConfigProvider")}</p><h3>{i18n.t("onboarding.readiness.providerDiagnosis")}</h3><p>{i18n.t("onboarding.readiness.credentialConfigProviderDescription")}</p></div>
          {error && <Alert tone="danger" title={i18n.t("onboarding.readiness.credentialSetup")} description={i18n.t("onboarding.readiness.credentialReadinessError")} />}
          {provider ? <div className="credential-config__provider-card">
            <div className="credential-config__provider-header">
              <div>
                <p className="credential-config__provider-label">{i18n.t("onboarding.readiness.providerDiagnosis")}</p>
                <h4>{provider.backend || i18n.t("onboarding.readiness.credentialConfigProvider")}</h4>
              </div>
              <StatusBadge tone={provider.condition === "available" ? "success" : "warning"}>{formatCredentialCondition(provider.condition)}</StatusBadge>
            </div>
            {provider.explanation && <p className="credential-config__provider-copy">{provider.explanation}</p>}
            {(provider.fix || provider.write_fix) && <div className="credential-config__provider-next" role="note"><strong>{i18n.t("onboarding.readiness.next")}</strong><span>{provider.fix || provider.write_fix}</span></div>}
            {(provider.write_condition || provider.write_explanation) && <dl className="credential-config__provider-facts"><div><dt>{i18n.t("onboarding.readiness.writeReachability")}</dt><dd>{provider.write_condition || provider.write_explanation}</dd></div></dl>}
          </div> : <p className="credential-config__empty">{i18n.t("onboarding.readiness.credentialConfigProviderEmpty")}</p>}
        </>}
        {activeTab === "actions" && <>
          <div className="credential-config__panel-heading"><p className="surface-eyebrow">{i18n.t("onboarding.readiness.credentialActions")}</p><h3>{i18n.t("onboarding.readiness.credentialActionsTitle")}</h3><p>{i18n.t("onboarding.readiness.credentialActionsDescription")}</p></div>
          {capabilitiesLoading ? <ConfigurationSkeleton /> : capabilities ? <CapabilityActions target={target} statuses={capabilities.capabilities} onRefresh={onRefresh} /> : <p className="credential-config__empty">{i18n.t("onboarding.readiness.credentialConfigProviderEmpty")}</p>}
        </>}
        {activeTab === "questions" && <>
          <div className="credential-config__panel-heading"><p className="surface-eyebrow">{i18n.t("onboarding.readiness.credentialQuestions")}</p><h3>{i18n.t("onboarding.readiness.credentialQuestionsTitle")}</h3><p>{i18n.t("onboarding.readiness.credentialQuestionsDescription")}</p></div>
          {operatorInputsLoading ? <ConfigurationSkeleton /> : operatorInputs ? <SchemaQuestionSet target={target} requests={operatorInputs.requests} /> : <p className="credential-config__empty">{i18n.t("onboarding.readiness.credentialConfigProviderEmpty")}</p>}
        </>}
      </section>
    </div>
  </div>;
}

function ConfigurationSkeleton() {
  return <div className="credential-config__skeleton" role="status" aria-label={i18n.t("onboarding.readiness.credentialLoading")}>
    <span />
    <span />
    <span />
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
