import { useEffect, useRef, useState } from "react";
import { CheckCircle2, ChevronRight, Eye, EyeOff, Loader2, Rocket, XCircle } from "lucide-react";
import { timestampDate } from "@bufbuild/protobuf/wkt";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { formatDate } from "../../i18n/format";
import { errorMessage } from "../../lib/errorMessage";
import {
  OnboardingState,
  OnboardingStepStatus,
  SourceMode,
  type OnboardingOp,
  type OnboardingStepEvent,
} from "../../api/onboard";
import { isTerminalOnboarding, useOnboardingQuery, useStartOnboardingMutation } from "./queries";

const DEFAULT_REVISION = "@cp";

// The wizard's three plain-language steps. Kept as an ordered tuple so the
// stepper, the "Step N of 3" indicator, and the step-body switch all agree.
const STEP_LABELS = [
  strings.fleet.onboard.stepConnect,
  strings.fleet.onboard.stepUnlock,
  strings.fleet.onboard.stepReview,
] as const;
const LAST_STEP = STEP_LABELS.length - 1;

// State/step-status enum → i18n key. Records (not switches) so an added enum
// value is a compile error here, keeping the label map exhaustive.
// `as const satisfies` preserves each value's literal key-path type (what the
// typed `t()` demands) while still checking the map is exhaustive over the enum.
const STATE_LABEL = {
  [OnboardingState.UNSPECIFIED]: strings.fleet.onboard.state.unspecified,
  [OnboardingState.PENDING]: strings.fleet.onboard.state.pending,
  [OnboardingState.SSH_SETUP]: strings.fleet.onboard.state.sshSetup,
  [OnboardingState.PUSHING_SCRIPT]: strings.fleet.onboard.state.pushingScript,
  [OnboardingState.BOOTSTRAPPING]: strings.fleet.onboard.state.bootstrapping,
  [OnboardingState.VERIFYING]: strings.fleet.onboard.state.verifying,
  [OnboardingState.SUCCEEDED]: strings.fleet.onboard.state.succeeded,
  [OnboardingState.FAILED]: strings.fleet.onboard.state.failed,
  [OnboardingState.CANCELLED]: strings.fleet.onboard.state.cancelled,
} as const satisfies Record<OnboardingState, string>;

const STEP_STATUS_LABEL = {
  [OnboardingStepStatus.UNSPECIFIED]: strings.fleet.onboard.stepStatus.unspecified,
  [OnboardingStepStatus.STARTED]: strings.fleet.onboard.stepStatus.started,
  [OnboardingStepStatus.OK]: strings.fleet.onboard.stepStatus.ok,
  [OnboardingStepStatus.SKIPPED]: strings.fleet.onboard.stepStatus.skipped,
  [OnboardingStepStatus.FAILED]: strings.fleet.onboard.stepStatus.failed,
} as const satisfies Record<OnboardingStepStatus, string>;

// Failure taxonomy code (wire string from op.failureReason) → i18n key. Mirrors
// the API's onboard.FailureReason vocabulary; an unmapped/empty code falls back
// to the honest generic message rather than pretending to know the cause.
const FAILURE_KEY = {
  ssh_setup_failed: strings.fleet.onboard.failure.ssh_setup_failed,
  script_push_failed: strings.fleet.onboard.failure.script_push_failed,
  pairing_issue_failed: strings.fleet.onboard.failure.pairing_issue_failed,
  bootstrap_usage_error: strings.fleet.onboard.failure.bootstrap_usage_error,
  unsupported_platform: strings.fleet.onboard.failure.unsupported_platform,
  pairing_failed: strings.fleet.onboard.failure.pairing_failed,
  bootstrap_failed: strings.fleet.onboard.failure.bootstrap_failed,
  verify_online_failed: strings.fleet.onboard.failure.verify_online_failed,
  interrupted_by_restart: strings.fleet.onboard.failure.interrupted_by_restart,
  internal_error: strings.fleet.onboard.failure.internal_error,
} as const;

type FailureMsgKey =
  | (typeof FAILURE_KEY)[keyof typeof FAILURE_KEY]
  | typeof strings.fleet.onboard.failure.generic;

// Resolve the i18n key for an op's failure reason, defaulting to the honest
// generic message for an unmapped/empty code. The Record cast re-opens the
// literal map to arbitrary string lookups so an unknown wire code yields
// `undefined` (→ generic) rather than a false always-defined type.
function failureStringKey(reason: string): FailureMsgKey {
  return (FAILURE_KEY as Record<string, FailureMsgKey>)[reason] ?? strings.fleet.onboard.failure.generic;
}

function splitCapabilities(raw: string): string[] {
  return raw
    .split(",")
    .map((s) => s.trim())
    .filter((s) => s.length > 0);
}

/** One step row in the live progress list. */
function StepRow({ event }: { event: OnboardingStepEvent }) {
  const { t } = useTranslation();
  const failed = event.status === OnboardingStepStatus.FAILED;
  // Wall-clock time each step was emitted, so the timeline reads as a sequence of
  // moments (and a stalled step is visible as a gap). Absent until the server
  // stamps it; a missing timestamp simply renders no time rather than a bogus one.
  const stampedAt = event.emittedAt ? formatDate(timestampDate(event.emittedAt), { timeStyle: "medium" }) : "";
  return (
    <li
      data-testid={selectors.fleet.onboardStep({ step: event.stepId })}
      className="flex items-baseline gap-2 text-xs"
    >
      <span className="shrink-0 font-mono text-[0.6rem] tabular-nums text-app-muted-foreground">{stampedAt}</span>
      <span className="font-mono text-app-foreground">{event.stepId}</span>
      <span className={`ms-auto text-end ${failed ? "text-app-danger" : "text-app-muted-foreground"}`}>
        {t(STEP_STATUS_LABEL[event.status])}
        {event.detail ? ` — ${event.detail}` : ""}
      </span>
    </li>
  );
}

/**
 * Collapsible panel showing the raw node-side output captured when onboarding
 * failed (op.failureDetail) — the concrete cause (e.g. the `make setup` error)
 * behind the plain-language failure reason. Uses a native <details> so it is
 * keyboard-accessible and needs no extra state; collapsed by default so the
 * banner stays scannable, expandable for the operator (or a support session)
 * that needs the specifics. The output is the machine's own diagnostics, never
 * secret material.
 */
function FailureOutput({ detail }: { detail: string }) {
  const { t } = useTranslation();
  return (
    <details
      data-testid={selectors.fleet.onboard.failureOutput}
      className="mt-1 rounded-control border border-app-danger/30 bg-app-background/60"
    >
      <summary
        data-testid={selectors.fleet.onboard.failureOutputToggle}
        className="cursor-pointer select-none px-2 py-1 text-xs font-medium text-app-foreground marker:text-app-muted-foreground"
      >
        {t(strings.fleet.onboard.failureOutputHeading)}
      </summary>
      <pre className="max-h-64 overflow-auto whitespace-pre-wrap break-words px-2 pb-2 pt-1 font-mono text-[0.65rem] leading-relaxed text-app-muted-foreground">
        {detail}
      </pre>
    </details>
  );
}

/** Terminal success/failure banner for a finished op. */
function TerminalBanner({ op }: { op: OnboardingOp }) {
  const { t } = useTranslation();
  if (op.state === OnboardingState.SUCCEEDED) {
    return (
      <div
        data-testid={selectors.fleet.onboard.success}
        role="status"
        className="flex flex-col gap-1 rounded-panel border border-app-success/40 bg-app-success/10 p-3"
      >
        <p className="flex items-center gap-2 text-sm font-semibold text-app-foreground">
          <CheckCircle2 aria-hidden="true" className="h-4 w-4 text-app-success" />
          {t(strings.fleet.onboard.successHeading)}
        </p>
        <p className="text-xs text-app-muted-foreground">
          {t(strings.fleet.onboard.successBody, { node: op.nodeName || op.nodeId })}
        </p>
        <p className="text-xs text-app-muted-foreground">{t(strings.fleet.onboard.viewNode)}</p>
      </div>
    );
  }
  if (op.state === OnboardingState.FAILED) {
    return (
      <div
        data-testid={selectors.fleet.onboard.failure}
        role="alert"
        className="flex flex-col gap-1 rounded-panel border border-app-danger/40 bg-app-danger/10 p-3"
      >
        <p className="flex items-center gap-2 text-sm font-semibold text-app-foreground">
          <XCircle aria-hidden="true" className="h-4 w-4 text-app-danger" />
          {t(strings.fleet.onboard.failureHeading)}
        </p>
        <p className="text-xs text-app-foreground">{t(failureStringKey(op.failureReason))}</p>
        {op.failureDetail ? <FailureOutput detail={op.failureDetail} /> : null}
        <p className="text-xs text-app-muted-foreground">{t(strings.fleet.onboard.retryHint)}</p>
      </div>
    );
  }
  // CANCELLED (or any other terminal-but-not-success) — surface the state label.
  return (
    <div
      data-testid={selectors.fleet.onboard.failure}
      role="alert"
      className="rounded-panel border border-app-border bg-app-background p-3 text-xs text-app-muted-foreground"
    >
      {t(STATE_LABEL[op.state])}
    </div>
  );
}

/** A labeled text input field. */
function Field({
  id,
  label,
  value,
  onChange,
  placeholder,
  type,
  testId,
  disabled,
  help,
}: {
  id: string;
  label: string;
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  type?: string;
  testId: string;
  disabled?: boolean;
  help?: string;
}) {
  return (
    <div className="flex min-w-0 flex-col gap-1">
      <label htmlFor={id} className="text-xs text-app-muted-foreground">
        {label}
      </label>
      <Input
        id={id}
        data-testid={testId}
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        disabled={disabled}
      />
      {help && <p className="text-[0.65rem] text-app-muted-foreground">{help}</p>}
    </div>
  );
}

/**
 * A labeled masked input with a show/hide affordance. The reveal toggle is a
 * standard credential-field affordance (typos in a masked field are otherwise
 * invisible); revealing only changes the input's `type`, never where the value
 * lives — it stays in ephemeral component state either way.
 */
function SecretField({
  id,
  label,
  value,
  onChange,
  testId,
  toggleTestId,
  disabled,
  help,
  showLabel,
  hideLabel,
}: {
  id: string;
  label: string;
  value: string;
  onChange: (v: string) => void;
  testId: string;
  toggleTestId: string;
  disabled?: boolean;
  help?: string;
  showLabel: string;
  hideLabel: string;
}) {
  const [revealed, setRevealed] = useState(false);
  return (
    <div className="flex min-w-0 flex-col gap-1">
      <label htmlFor={id} className="text-xs text-app-muted-foreground">
        {label}
      </label>
      <div className="relative">
        <Input
          id={id}
          data-testid={testId}
          type={revealed ? "text" : "password"}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          disabled={disabled}
          // Suppress the browser's save-password offer: this is a one-shot
          // remote-host secret that must never reach the browser's persistent
          // credential store.
          autoComplete="new-password"
          className="pe-9"
        />
        <button
          type="button"
          data-testid={toggleTestId}
          aria-label={revealed ? hideLabel : showLabel}
          aria-pressed={revealed}
          onClick={() => setRevealed((v) => !v)}
          disabled={disabled}
          className="absolute inset-y-0 end-0 flex w-9 items-center justify-center text-app-muted-foreground hover:text-app-foreground disabled:opacity-50"
        >
          {revealed ? (
            <EyeOff aria-hidden="true" className="h-4 w-4" />
          ) : (
            <Eye aria-hidden="true" className="h-4 w-4" />
          )}
        </button>
      </div>
      {help && <p className="text-[0.65rem] text-app-muted-foreground">{help}</p>}
    </div>
  );
}

/** The "Step N of 3" indicator with named steps and a current-step marker. */
function StepIndicator({ current }: { current: number }) {
  const { t } = useTranslation();
  return (
    <div className="flex flex-col gap-2">
      <p data-testid={selectors.fleet.onboard.stepIndicator} className="text-xs font-medium text-app-muted-foreground">
        {t(strings.fleet.onboard.stepIndicator, { current: current + 1, total: STEP_LABELS.length })}
      </p>
      <ol className="flex flex-wrap gap-2">
        {STEP_LABELS.map((label, i) => {
          const active = i === current;
          const done = i < current;
          return (
            <li
              key={label}
              aria-current={active ? "step" : undefined}
              className={[
                "flex items-center gap-1.5 rounded-control px-2.5 py-1 text-xs font-medium",
                active
                  ? "bg-app-primary text-app-primary-foreground"
                  : done
                    ? "bg-app-surface-muted text-app-foreground"
                    : "bg-app-surface-muted text-app-muted-foreground",
              ].join(" ")}
            >
              <span
                className={[
                  "flex h-4 w-4 items-center justify-center rounded-pill text-[0.6rem]",
                  active ? "bg-app-primary-foreground text-app-primary" : "bg-app-border text-app-foreground",
                ].join(" ")}
              >
                {i + 1}
              </span>
              {t(label)}
            </li>
          );
        })}
      </ol>
    </div>
  );
}

/**
 * Guided "Add a node" wizard: three plain-language steps (Connect → Unlock →
 * Review & start) that drive a raw SSH host from bare OS to a paired, ONLINE
 * fleet agent as a durable, server-owned op. The SSH password is masked, held
 * only in ephemeral component state, sent once in the StartOnboarding request,
 * and cleared the moment the request settles — never written to browser storage.
 *
 * The wizard is a single form so keyboard submit works: Back/Next are type=button
 * and only the final "Start" is type=submit; Enter on an earlier step advances
 * instead of starting. All field state lives in this component, so values survive
 * moving between steps. Once an op is started the wizard is replaced by the live
 * progress view (GetOnboarding polled until terminal), so a reload re-attaches.
 */
export function OnboardNodeForm({ retryTarget }: { retryTarget?: OnboardingOp | null } = {}) {
  const { t } = useTranslation();
  const start = useStartOnboardingMutation();

  const [step, setStep] = useState(0);
  const [host, setHost] = useState("");
  const [user, setUser] = useState("");
  const [port, setPort] = useState("");
  const [nodeName, setNodeName] = useState("");
  const [password, setPassword] = useState("");
  const [capabilities, setCapabilities] = useState("");
  const [revision, setRevision] = useState(DEFAULT_REVISION);
  // Default ON: the operator is handing over admin credentials, so the useful
  // default is to leave the host with working non-interactive sudo.
  const [provisionSudo, setProvisionSudo] = useState(true);
  // Dial-back URL for the node. Blank falls through to the server default:
  // $BRIDGE_CONTROL_PLANE_URL or the control plane's own derived address.
  const [controlPlaneUrl, setControlPlaneUrl] = useState("");
  const [reachabilityMode, setReachabilityMode] = useState("lan");
  // Setup profile — blank fields fall through to the node's `vrooli setup`
  // defaults (the sensible fleet default: don't reshape a node's setup unless the
  // operator asks). includeOptional defaults off (required safeguards only).
  const [setupEnvironment, setSetupEnvironment] = useState("");
  const [setupResources, setSetupResources] = useState("");
  const [setupScenarios, setSetupScenarios] = useState("");
  const [includeOptional, setIncludeOptional] = useState(false);
  // Source mode: default working-tree — ship THIS computer's current files
  // (uncommitted work included) over SSH, so onboarding your own machines needs no
  // git push. Unchecking switches to pinned (the node downloads a pushed revision),
  // which is the only mode that requires the target commit to already be pushed.
  const [workingTree, setWorkingTree] = useState(true);
  // Collapsed disclosures: port on step 1, everything expert on step 3.
  const [moreOpen, setMoreOpen] = useState(false);
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const [opId, setOpId] = useState<string | null>(null);
	const [retryUnavailable, setRetryUnavailable] = useState(false);

  // A saved failed attempt pre-fills only durable, non-secret target identity.
  // The SSH password is deliberately absent from OnboardingOp and must be
  // entered again for a retry.
  useEffect(() => {
    if (!retryTarget) return;
    setHost(retryTarget.host);
    setUser(retryTarget.user);
    setPort(retryTarget.port > 0 ? String(retryTarget.port) : "");
    setNodeName(retryTarget.nodeName);
    setRevision(retryTarget.targetRevision || DEFAULT_REVISION);
    setPassword("");
		setRetryUnavailable(false);
    setStep(0);
    setOpId(null);
  }, [retryTarget]);

  const onboarding = useOnboardingQuery(opId);
  const op = onboarding.data?.op ?? null;
  const events = onboarding.data?.events ?? [];
  const active = opId !== null && op !== null && !isTerminalOnboarding(op.state);
  // `submitting` is the brief window while StartOnboarding is in flight (the
  // wizard's Start button shows "Starting…"); once an op id lands the wizard is
  // replaced entirely by the live progress view.
  const submitting = start.isPending;
  const showProgress = opId !== null;

  // Move focus to the step heading on every step change so keyboard and screen-
  // reader users land on the new step's context, not a stale control. Skips the
  // initial mount so the wizard doesn't steal focus when the page first renders.
  const headingRef = useRef<HTMLHeadingElement>(null);
  const mountedRef = useRef(false);
  useEffect(() => {
    if (!mountedRef.current) {
      mountedRef.current = true;
      return;
    }
    headingRef.current?.focus();
  }, [step]);

  const canLeaveConnect = host.trim().length > 0;

  const goNext = () => {
    if (step === 0 && !canLeaveConnect) return;
    setStep((s) => Math.min(LAST_STEP, s + 1));
  };
  const goBack = () => setStep((s) => Math.max(0, s - 1));

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    // Enter on an earlier step advances rather than starting onboarding; only
    // the final step's "Start" actually kicks off the op.
    if (step < LAST_STEP) {
      goNext();
      return;
    }
    const trimmedHost = host.trim();
    if (!trimmedHost || submitting) return;
    if (retryTarget && (!retryTarget.machineId || !retryTarget.enrollmentAttemptId)) {
      setRetryUnavailable(true);
      return;
    }
    const parsedPort = Number.parseInt(port.trim(), 10);
    start.mutate(
      {
        host: trimmedHost,
        user: user.trim(),
        port: Number.isFinite(parsedPort) ? parsedPort : 0,
        nodeName: nodeName.trim(),
        sshPassword: password,
        capabilities: splitCapabilities(capabilities),
        targetRevision: revision.trim() || DEFAULT_REVISION,
        controlPlaneUrl: controlPlaneUrl.trim(),
        reachabilityMode,
        provisionSudo,
        setupEnvironment: setupEnvironment.trim(),
        setupResources: setupResources.trim(),
        setupScenarios: setupScenarios.trim(),
        includeOptional,
        sourceMode: workingTree ? SourceMode.WORKING_TREE : SourceMode.PINNED_REVISION,
        machineId: retryTarget?.machineId ?? "",
        retryOfEnrollmentAttemptId: retryTarget?.enrollmentAttemptId ?? "",
      },
      {
        onSuccess: (resp) => setOpId(resp.opId),
        // Clear the secret the moment the request settles (success OR failure) so
        // it never lingers in component state beyond the request.
        onSettled: () => setPassword(""),
      },
    );
  };

  // Once an op is started, the wizard is replaced by the live progress view.
  if (showProgress) {
    return (
      <section
        id="add-node"
        aria-labelledby="fleet-onboard-heading"
        className="rounded-panel border border-app-border bg-app-surface p-4"
      >
        <h3
          id="fleet-onboard-heading"
          ref={headingRef}
          tabIndex={-1}
          className="text-sm font-semibold text-app-foreground outline-none"
        >
          {t(strings.fleet.onboard.heading)}
        </h3>
        <div
          data-testid={selectors.fleet.onboard.progress}
          className="mt-3 flex flex-col gap-2 rounded-panel border border-app-border bg-app-background p-3"
        >
          <p className="flex items-center gap-2 text-sm font-semibold text-app-foreground">
            {active && <Loader2 aria-hidden="true" className="h-3.5 w-3.5 animate-spin" />}
            {t(strings.fleet.onboard.progressHeading)}
          </p>
          {op && (
            <p className="text-xs text-app-muted-foreground">{t(STATE_LABEL[op.state])}</p>
          )}
          {events.length === 0 ? (
            <p className="text-xs text-app-muted-foreground">{t(strings.fleet.onboard.waiting)}</p>
          ) : (
            <ul data-testid={selectors.fleet.onboard.steps} className="flex flex-col gap-1">
              {events.map((ev) => (
                <StepRow key={`${ev.sequence}-${ev.stepId}`} event={ev} />
              ))}
            </ul>
          )}
          {op && isTerminalOnboarding(op.state) && <TerminalBanner op={op} />}
        </div>
      </section>
    );
  }

  return (
    <section
      id="add-node"
      aria-labelledby="fleet-onboard-heading"
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <h3
        id="fleet-onboard-heading"
        ref={headingRef}
        tabIndex={-1}
        className="text-sm font-semibold text-app-foreground outline-none"
      >
        {t(strings.fleet.onboard.heading)}
      </h3>
      <p className="mt-1 text-xs text-app-muted-foreground">{t(strings.fleet.onboard.description)}</p>

      <div className="mt-4">
        <StepIndicator current={step} />
      </div>

      <form data-testid={selectors.fleet.onboard.form} onSubmit={handleSubmit} className="mt-4 flex flex-col gap-4">
        {step === 0 && (
          <div className="flex flex-col gap-3">
            <p className="text-sm text-app-foreground">{t(strings.fleet.onboard.connectIntro)}</p>
            <div className="grid gap-3 sm:grid-cols-2">
              <Field
                id="fleet-onboard-host-input"
                testId={selectors.fleet.onboard.host}
                label={t(strings.fleet.onboard.hostLabel)}
                value={host}
                onChange={setHost}
                placeholder={t(strings.fleet.onboard.hostPlaceholder)}
              />
              <Field
                id="fleet-onboard-user-input"
                testId={selectors.fleet.onboard.user}
                label={t(strings.fleet.onboard.userLabel)}
                value={user}
                onChange={setUser}
                placeholder={t(strings.fleet.onboard.userPlaceholder)}
              />
            </div>
            <p className="rounded-panel border border-app-border bg-app-background p-2 text-[0.7rem] text-app-muted-foreground">
              {t(strings.fleet.onboard.macHint)}
            </p>
            <div>
              <button
                type="button"
                data-testid={selectors.fleet.onboard.moreToggle}
                aria-expanded={moreOpen}
                onClick={() => setMoreOpen((v) => !v)}
                className="text-xs font-medium text-app-muted-foreground hover:text-app-foreground"
              >
                {t(strings.fleet.onboard.moreLabel)}
              </button>
              {moreOpen && (
                <div className="mt-2">
                  <Field
                    id="fleet-onboard-port-input"
                    testId={selectors.fleet.onboard.port}
                    label={t(strings.fleet.onboard.portLabel)}
                    value={port}
                    onChange={setPort}
                    placeholder={t(strings.fleet.onboard.portPlaceholder)}
                    type="number"
                  />
                </div>
              )}
            </div>
          </div>
        )}

        {step === 1 && (
          <div className="flex flex-col gap-3">
            <p className="text-sm text-app-foreground">{t(strings.fleet.onboard.unlockIntro)}</p>
            <SecretField
              id="fleet-onboard-password-input"
              testId={selectors.fleet.onboard.password}
              toggleTestId={selectors.fleet.onboard.passwordToggle}
              label={t(strings.fleet.onboard.passwordLabel)}
              value={password}
              onChange={setPassword}
              showLabel={t(strings.fleet.onboard.passwordShow)}
              hideLabel={t(strings.fleet.onboard.passwordHide)}
            />
            <p className="text-[0.7rem] text-app-muted-foreground">
              {t(strings.fleet.onboard.passwordOptionalNote)}
            </p>
            <div className="flex min-w-0 flex-col gap-1">
              <label className="flex items-start gap-2 text-xs text-app-foreground">
                <input
                  id="fleet-onboard-provision-sudo-input"
                  data-testid={selectors.fleet.onboard.provisionSudo}
                  type="checkbox"
                  className="mt-0.5 h-4 w-4 shrink-0"
                  checked={provisionSudo}
                  onChange={(e) => setProvisionSudo(e.target.checked)}
                />
                <span>{t(strings.fleet.onboard.provisionSudoLabel)}</span>
              </label>
              <p className="text-[0.65rem] text-app-muted-foreground">
                {t(strings.fleet.onboard.provisionSudoHelp)}
              </p>
            </div>
          </div>
        )}

        {step === 2 && (
          <div className="flex flex-col gap-3">
            <div
              className="flex flex-col gap-2 rounded-panel border border-app-border bg-app-background p-3 text-sm text-app-foreground"
            >
              <p>
                {t(strings.fleet.onboard.reviewSummary, {
                  target: `${user.trim() || t(strings.fleet.onboard.userPlaceholder)}@${host.trim()}`,
                })}
              </p>
              <p data-testid={selectors.fleet.onboard.sourceSummary} className="text-xs text-app-muted-foreground">
                {workingTree
                  ? t(strings.fleet.onboard.sourceSummaryWorkingTree)
                  : t(strings.fleet.onboard.sourceSummaryPinned, { revision: revision.trim() || DEFAULT_REVISION })}
              </p>
            </div>

            <div className="rounded-panel border border-app-border p-3">
              <button
                type="button"
                data-testid={selectors.fleet.onboard.advancedToggle}
                aria-expanded={advancedOpen}
                onClick={() => setAdvancedOpen((v) => !v)}
                className="flex w-full items-center justify-between text-xs font-semibold text-app-foreground"
              >
                <span>{t(strings.fleet.onboard.advancedHeading)}</span>
                <ChevronRight
                  aria-hidden="true"
                  className={`h-4 w-4 transition-transform ${advancedOpen ? "rotate-90" : ""}`}
                />
              </button>
              <p className="mt-1 text-[0.65rem] text-app-muted-foreground">
                {t(strings.fleet.onboard.advancedHelp)}
              </p>

              {advancedOpen && (
                <div className="mt-3 flex flex-col gap-3">
                  <div className="grid gap-3 sm:grid-cols-2">
                    <Field
                      id="fleet-onboard-name-input"
                      testId={selectors.fleet.onboard.name}
                      label={t(strings.fleet.onboard.nameLabel)}
                      value={nodeName}
                      onChange={setNodeName}
                      placeholder={t(strings.fleet.onboard.namePlaceholder)}
                    />
                    <Field
                      id="fleet-onboard-capabilities-input"
                      testId={selectors.fleet.onboard.capabilities}
                      label={t(strings.fleet.onboard.capabilitiesLabel)}
                      value={capabilities}
                      onChange={setCapabilities}
                      placeholder={t(strings.fleet.onboard.capabilitiesPlaceholder)}
                      help={t(strings.fleet.onboard.capabilitiesHelp)}
                    />
                    <Field
                      id="fleet-onboard-revision-input"
                      testId={selectors.fleet.onboard.revision}
                      label={t(strings.fleet.onboard.revisionLabel)}
                      value={revision}
                      onChange={setRevision}
                      help={t(strings.fleet.onboard.revisionHelp)}
                    />
                    <Field
                      id="fleet-onboard-control-plane-url-input"
                      testId={selectors.fleet.onboard.controlPlaneUrl}
                      label={t(strings.fleet.onboard.controlPlaneUrlLabel)}
                      value={controlPlaneUrl}
                      onChange={setControlPlaneUrl}
                      placeholder={t(strings.fleet.onboard.controlPlaneUrlPlaceholder)}
                      help={t(strings.fleet.onboard.controlPlaneUrlHelp)}
                    />
                    <label className="flex flex-col gap-1 text-xs text-app-muted-foreground">
                      <span className="font-medium text-app-foreground">{t(strings.fleet.onboard.reachabilityModeLabel)}</span>
                      <select
                        value={reachabilityMode}
                        onChange={(event) => setReachabilityMode(event.target.value)}
                        className="h-9 rounded-control border border-app-border bg-app-background px-2 text-sm text-app-foreground"
                      >
                        <option value="lan">LAN — trusted local network</option>
                        <option value="tunnel">Tunnel — off-LAN or segmented network</option>
                        <option value="manual">Manual — managed DNS or VPN</option>
                      </select>
                      <span>{t(strings.fleet.onboard.reachabilityModeHelp)}</span>
                    </label>
                    <Field
                      id="fleet-onboard-setup-environment-input"
                      testId={selectors.fleet.onboard.setupEnvironment}
                      label={t(strings.fleet.onboard.setupEnvironmentLabel)}
                      value={setupEnvironment}
                      onChange={setSetupEnvironment}
                      placeholder={t(strings.fleet.onboard.setupEnvironmentPlaceholder)}
                      help={t(strings.fleet.onboard.setupEnvironmentHelp)}
                    />
                    <Field
                      id="fleet-onboard-setup-resources-input"
                      testId={selectors.fleet.onboard.setupResources}
                      label={t(strings.fleet.onboard.setupResourcesLabel)}
                      value={setupResources}
                      onChange={setSetupResources}
                      placeholder={t(strings.fleet.onboard.setupResourcesPlaceholder)}
                      help={t(strings.fleet.onboard.setupResourcesHelp)}
                    />
                    <Field
                      id="fleet-onboard-setup-scenarios-input"
                      testId={selectors.fleet.onboard.setupScenarios}
                      label={t(strings.fleet.onboard.setupScenariosLabel)}
                      value={setupScenarios}
                      onChange={setSetupScenarios}
                      placeholder={t(strings.fleet.onboard.setupScenariosPlaceholder)}
                      help={t(strings.fleet.onboard.setupScenariosHelp)}
                    />
                  </div>
                  <label className="flex items-start gap-2 text-xs text-app-foreground">
                    <input
                      id="fleet-onboard-include-optional-input"
                      data-testid={selectors.fleet.onboard.includeOptional}
                      type="checkbox"
                      className="mt-0.5 h-4 w-4 shrink-0"
                      checked={includeOptional}
                      onChange={(e) => setIncludeOptional(e.target.checked)}
                    />
                    <span>{t(strings.fleet.onboard.includeOptionalLabel)}</span>
                  </label>
                  <p className="text-[0.65rem] text-app-muted-foreground">
                    {t(strings.fleet.onboard.includeOptionalHelp)}
                  </p>

                  <label className="flex items-start gap-2 text-xs text-app-foreground">
                    <input
                      id="fleet-onboard-source-working-tree-input"
                      data-testid={selectors.fleet.onboard.sourceWorkingTree}
                      type="checkbox"
                      className="mt-0.5 h-4 w-4 shrink-0"
                      checked={workingTree}
                      onChange={(e) => setWorkingTree(e.target.checked)}
                    />
                    <span>{t(strings.fleet.onboard.sourceModeLabel)}</span>
                  </label>
                  <p className="text-[0.65rem] text-app-muted-foreground">
                    {t(strings.fleet.onboard.sourceModeHelp)}
                  </p>
                  {!workingTree && (
                    <p
                      data-testid={selectors.fleet.onboard.sourceWarning}
                      role="note"
                      className="rounded-panel border border-app-border bg-app-background p-2 text-[0.65rem] text-app-muted-foreground"
                    >
                      {t(strings.fleet.onboard.sourceModeWarning)}
                    </p>
                  )}
                </div>
              )}
            </div>
          </div>
        )}

        {start.error && (
          <p data-testid={selectors.fleet.onboard.error} role="alert" className="text-sm text-app-danger">
            {errorMessage(start.error, t)}
          </p>
        )}
			{retryUnavailable && (
				<p data-testid={selectors.fleet.onboard.error} role="alert" className="text-sm text-app-danger">
					{t(strings.fleet.onboard.retryUnavailable)}
				</p>
			)}

        <div className="flex items-center justify-between gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            data-testid={selectors.fleet.onboard.back}
            onClick={goBack}
            disabled={step === 0}
          >
            {t(strings.fleet.onboard.back)}
          </Button>
          {step < LAST_STEP ? (
            <Button
              type="button"
              size="sm"
              data-testid={selectors.fleet.onboard.next}
              onClick={goNext}
              disabled={step === 0 && !canLeaveConnect}
            >
              {t(strings.fleet.onboard.next)}
              <ChevronRight aria-hidden="true" className="ml-1 h-4 w-4" />
            </Button>
          ) : (
            <Button
              type="submit"
              size="sm"
              data-testid={selectors.fleet.onboard.submit}
              disabled={host.trim().length === 0 || submitting}
            >
              <Rocket aria-hidden="true" className="mr-2 h-4 w-4" />
              {submitting ? t(strings.fleet.onboard.submitting) : t(strings.fleet.onboard.submit)}
            </Button>
          )}
        </div>
      </form>
    </section>
  );
}
