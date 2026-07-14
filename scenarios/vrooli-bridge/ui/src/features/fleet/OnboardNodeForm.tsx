import { useState } from "react";
import { CheckCircle2, Loader2, Rocket, XCircle } from "lucide-react";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import {
  OnboardingState,
  OnboardingStepStatus,
  type OnboardingOp,
  type OnboardingStepEvent,
} from "../../api/onboard";
import {
  isTerminalOnboarding,
  useOnboardingQuery,
  useStartOnboardingMutation,
} from "./queries";

const DEFAULT_REVISION = "@cp";

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
  return (
    <li
      data-testid={selectors.fleet.onboardStep({ step: event.stepId })}
      className="flex items-baseline justify-between gap-2 text-xs"
    >
      <span className="font-mono text-app-foreground">{event.stepId}</span>
      <span className={failed ? "text-app-danger" : "text-app-muted-foreground"}>
        {t(STEP_STATUS_LABEL[event.status])}
        {event.detail ? ` — ${event.detail}` : ""}
      </span>
    </li>
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
 * One-shot onboarding surface: the owner points the control plane at a raw SSH
 * host and it drives the host from bare OS to a paired, ONLINE fleet agent as a
 * durable, server-owned op. The SSH password is masked, held only in ephemeral
 * component state, sent once in the StartOnboarding request body, and cleared
 * the moment the request settles — it is never written to browser storage or
 * any persisted field. Live step states are read back from the durable op via
 * GetOnboarding (polled until terminal), so a reload simply re-attaches.
 */
export function OnboardNodeForm() {
  const { t } = useTranslation();
  const start = useStartOnboardingMutation();

  const [host, setHost] = useState("");
  const [user, setUser] = useState("");
  const [port, setPort] = useState("");
  const [nodeName, setNodeName] = useState("");
  const [password, setPassword] = useState("");
  const [capabilities, setCapabilities] = useState("");
  const [revision, setRevision] = useState(DEFAULT_REVISION);
  const [opId, setOpId] = useState<string | null>(null);

  const onboarding = useOnboardingQuery(opId);
  const op = onboarding.data?.op ?? null;
  const events = onboarding.data?.events ?? [];
  const active = opId !== null && op !== null && !isTerminalOnboarding(op.state);
  const submitting = start.isPending || active;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const trimmedHost = host.trim();
    if (!trimmedHost || submitting) return;
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
      },
      {
        onSuccess: (resp) => setOpId(resp.opId),
        // Clear the secret the moment the request settles (success OR failure) so
        // it never lingers in component state beyond the request.
        onSettled: () => setPassword(""),
      },
    );
  };

  return (
    <section
      aria-labelledby="fleet-onboard-heading"
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <h3 id="fleet-onboard-heading" className="text-sm font-semibold text-app-foreground">
        {t(strings.fleet.onboard.heading)}
      </h3>
      <p className="mt-1 text-xs text-app-muted-foreground">{t(strings.fleet.onboard.description)}</p>

      <form
        data-testid={selectors.fleet.onboard.form}
        onSubmit={handleSubmit}
        className="mt-3 grid gap-3 sm:grid-cols-2"
      >
        <Field
          id="fleet-onboard-host-input"
          testId={selectors.fleet.onboard.host}
          label={t(strings.fleet.onboard.hostLabel)}
          value={host}
          onChange={setHost}
          placeholder={t(strings.fleet.onboard.hostPlaceholder)}
          disabled={submitting}
        />
        <Field
          id="fleet-onboard-user-input"
          testId={selectors.fleet.onboard.user}
          label={t(strings.fleet.onboard.userLabel)}
          value={user}
          onChange={setUser}
          placeholder={t(strings.fleet.onboard.userPlaceholder)}
          disabled={submitting}
        />
        <Field
          id="fleet-onboard-port-input"
          testId={selectors.fleet.onboard.port}
          label={t(strings.fleet.onboard.portLabel)}
          value={port}
          onChange={setPort}
          placeholder={t(strings.fleet.onboard.portPlaceholder)}
          type="number"
          disabled={submitting}
        />
        <Field
          id="fleet-onboard-name-input"
          testId={selectors.fleet.onboard.name}
          label={t(strings.fleet.onboard.nameLabel)}
          value={nodeName}
          onChange={setNodeName}
          placeholder={t(strings.fleet.onboard.namePlaceholder)}
          disabled={submitting}
        />
        <Field
          id="fleet-onboard-password-input"
          testId={selectors.fleet.onboard.password}
          label={t(strings.fleet.onboard.passwordLabel)}
          value={password}
          onChange={setPassword}
          type="password"
          disabled={submitting}
          help={t(strings.fleet.onboard.passwordHelp)}
        />
        <Field
          id="fleet-onboard-capabilities-input"
          testId={selectors.fleet.onboard.capabilities}
          label={t(strings.fleet.onboard.capabilitiesLabel)}
          value={capabilities}
          onChange={setCapabilities}
          placeholder={t(strings.fleet.onboard.capabilitiesPlaceholder)}
          disabled={submitting}
          help={t(strings.fleet.onboard.capabilitiesHelp)}
        />
        <Field
          id="fleet-onboard-revision-input"
          testId={selectors.fleet.onboard.revision}
          label={t(strings.fleet.onboard.revisionLabel)}
          value={revision}
          onChange={setRevision}
          disabled={submitting}
          help={t(strings.fleet.onboard.revisionHelp)}
        />
        <div className="flex items-end">
          <Button
            type="submit"
            size="sm"
            data-testid={selectors.fleet.onboard.submit}
            disabled={submitting || host.trim().length === 0}
          >
            <Rocket aria-hidden="true" className="mr-2 h-4 w-4" />
            {submitting ? t(strings.fleet.onboard.submitting) : t(strings.fleet.onboard.submit)}
          </Button>
        </div>
      </form>

      {start.error && (
        <p
          data-testid={selectors.fleet.onboard.error}
          role="alert"
          className="mt-3 text-sm text-app-danger"
        >
          {errorMessage(start.error, t)}
        </p>
      )}

      {op && (
        <div
          data-testid={selectors.fleet.onboard.progress}
          className="mt-3 flex flex-col gap-2 rounded-panel border border-app-border bg-app-background p-3"
        >
          <p className="flex items-center gap-2 text-xs font-semibold text-app-foreground">
            {active && <Loader2 aria-hidden="true" className="h-3.5 w-3.5 animate-spin" />}
            {t(strings.fleet.onboard.progressHeading)}: {t(STATE_LABEL[op.state])}
          </p>
          {events.length === 0 ? (
            <p className="text-xs text-app-muted-foreground">{t(strings.fleet.onboard.waiting)}</p>
          ) : (
            <ul data-testid={selectors.fleet.onboard.steps} className="flex flex-col gap-1">
              {events.map((ev) => (
                <StepRow key={`${ev.sequence}-${ev.stepId}`} event={ev} />
              ))}
            </ul>
          )}
          {isTerminalOnboarding(op.state) && <TerminalBanner op={op} />}
        </div>
      )}
    </section>
  );
}
