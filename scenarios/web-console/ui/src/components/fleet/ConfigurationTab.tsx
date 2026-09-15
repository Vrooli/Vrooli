import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { AlertTriangle, CheckCircle2, CircleAlert, Loader2 } from "lucide-react";
import { ErrorState } from "@vrooli/react-component-library/ErrorState/1";
import { FormField } from "@vrooli/react-component-library/FormField/1";
import { Input } from "@vrooli/react-component-library/Input/1";
import { PasswordInput } from "@vrooli/react-component-library/PasswordInput/2";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { GeneratedForm } from "@vrooli/react-component-library/GeneratedForm/1";
import { VerdictSummary } from "@vrooli/react-component-library/VerdictSummary/1";
import { toGeneratedFields, type OperatorInput } from "@vrooli/react-component-library/ValidationAdapter/1";
import { Button } from "../ui/button";
import { strings } from "../../consts/strings";
import { machineIssues } from "../machines/MachineList";
import {
  answerSecret,
  createCredentialGrant,
  getConfiguration,
  getConfigurationApplyStatus,
  listCredentialGrants,
  reapplyConfiguration,
  resolveConfiguration,
  revokeCredentialGrant,
  type ConfigurationQuestion,
  type Machine,
  type MachineConfigurationDetail,
} from "../../api/machines";
import type { CredentialGrant } from "@vrooli/proto-types/vrooli-bridge/v1/credentialgrant/credentialgrant_pb";
import { ReadinessState, type GetReadinessResponse } from "@vrooli/proto-types/vrooli-onboarding/v1/readiness/readiness_pb";
import type { InstallOutcome } from "../../api/capabilities";
import { summarizeApplyRun, type ApplyRunSummary } from "./applyRunSummary";

/**
 * A machine's desired state — the panel formerly reached by `Configure`.
 *
 * Three things changed beyond moving it into a tab.
 *
 * The failure is no longer body copy. A bridge 502 arrives as a transport
 * string naming a node id, a verb and a governed-catalog method; that is the
 * right text for an engineer and the wrong altitude for the person looking at
 * the screen. `ErrorState` now takes a `detail`, so the sentence stays on top
 * and the dump sits one disclosure below it.
 *
 * Drift is rows with a verb rather than a paragraph of internal keys.
 * `managed-connection` and `ssh.management` are names the bridge uses, not
 * names anyone reads, and each row carries the action that clears it.
 *
 * Held credentials read as a list of what is held, with adding behind a
 * secondary action. Before, two unlabelled placeholder boxes and a filled
 * primary sat where the list should have been — the most obscure action on the
 * screen carrying its strongest emphasis.
 */

/** Bridge names are addresses, not labels. Give the ones we know a real name. */
const DRIFT_LABELS: Record<string, string> = {
  "managed-connection": "Connection profile",
  "ssh.management": "SSH management",
};

function driftLabel(name: string): string {
  return DRIFT_LABELS[name] ?? name;
}

function answerValue(value: unknown): string {
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean" || typeof value === "bigint") {
    return value.toString();
  }
  return "";
}

function readinessStateName(state: ReadinessState): string {
  return ReadinessState[state];
}

/** Codes such as "helper_not_installed: …" lead the machine's detail; the prose after it is the explanation. */
function nodeFeatureDetail(detail?: string): string {
  return (detail ?? "").replace(/^[a-z0-9_]+:\s*/, "");
}

/**
 * A re-apply the operator started, from the request to the run's outcome.
 *
 * It renders at the top of the panel. It used to be one line of text below the
 * credentials and questions — off-screen on a phone — and polling gave up
 * after three minutes while runs routinely take longer, so a press of Fix
 * appeared to do nothing at all.
 */
type ApplyActivity =
  | { phase: "starting"; startedAt: number }
  | { phase: "running" | "done" | "timeout"; startedAt: number; summary: ApplyRunSummary }
  | { phase: "error"; startedAt: number; message: string };

const APPLY_POLL_LIMIT_MS = 20 * 60 * 1000;

function applyPollDelay(elapsedMs: number): number {
  return elapsedMs < 30_000 ? 2_000 : 5_000;
}

function ApplyActivityPanel({ activity, onDismiss }: { activity: ApplyActivity; onDismiss: () => void }) {
  const { t } = useTranslation();
  const busy = activity.phase === "starting" || activity.phase === "running";
  let title: string;
  let tone: "neutral" | "success" | "warning" | "danger";
  switch (activity.phase) {
    case "starting":
      title = t(strings.machines.applyStarting);
      tone = "neutral";
      break;
    case "error":
      title = t(strings.machines.applyError);
      tone = "danger";
      break;
    case "running":
      title = t(strings.machines.applyRunning, { finished: activity.summary.finished, total: activity.summary.total });
      tone = "neutral";
      break;
    case "timeout":
      title = t(strings.machines.applyTimedOut, { minutes: Math.round(APPLY_POLL_LIMIT_MS / 60_000) });
      tone = "warning";
      break;
    case "done": {
      const { outcome, failed, total } = activity.summary;
      const titles: Record<string, string> = {
        applied: t(strings.machines.applyApplied),
        "already-satisfied": t(strings.machines.applyAlreadySatisfied),
        partial: t(strings.machines.applyPartial, { failed: failed.length, total }),
        incomplete: t(strings.machines.applyIncomplete),
        failed: t(strings.machines.applyFailed),
        cancelled: t(strings.machines.applyCancelled),
      };
      title = titles[outcome] ?? t(strings.machines.applyIndeterminate);
      tone = outcome === "applied" || outcome === "already-satisfied" ? "success" : outcome === "failed" || outcome === "cancelled" ? "danger" : "warning";
      break;
    }
  }
  const toneClass = {
    neutral: "border-wc-default bg-wc-surface-input text-wc-text-primary",
    success: "border-emerald-400/30 bg-emerald-400/10 text-emerald-100",
    warning: "border-amber-400/30 bg-amber-400/10 text-amber-100",
    danger: "border-rose-400/30 bg-rose-400/10 text-rose-100",
  }[tone];
  const Icon = busy ? Loader2 : tone === "success" ? CheckCircle2 : tone === "danger" ? CircleAlert : AlertTriangle;
  const failed = activity.phase === "done" || activity.phase === "timeout" ? activity.summary.failed : [];
  return (
    <section
      role="status"
      aria-live="polite"
      data-testid="machine-configuration-apply"
      data-apply-phase={activity.phase}
      data-apply-outcome={activity.phase === "done" ? activity.summary.outcome : undefined}
      className={`rounded-xl border p-3 ${toneClass}`}
    >
      <div className="flex items-start gap-2">
        <Icon className={`mt-0.5 h-4 w-4 shrink-0 ${busy ? "animate-spin" : ""}`} aria-hidden />
        <p className="min-w-0 flex-1 text-xs font-medium">{title}</p>
        {!busy && (
          <button type="button" onClick={onDismiss} className="shrink-0 text-[11px] underline-offset-2 opacity-80 hover:underline">
            {t(strings.machines.applyDismiss)}
          </button>
        )}
      </div>
      {activity.phase === "error" && <p className="mt-1 break-words text-[11px] opacity-90">{activity.message}</p>}
      {failed.length > 0 && (
        <ul className="mt-2 space-y-1.5">
          {failed.map((step) => (
            <li key={step.id} data-testid={`machine-configuration-apply-failed-${step.id}`} className="break-words text-[11px]">
              <span className="font-mono">{step.name}</span> — {step.reason}
              {step.remediation && <span className="block opacity-75">{step.remediation}</span>}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

export function ConfigurationTab({
  machine,
  onInstallCapability,
}: {
  machine: Machine;
  onInstallCapability?: (capabilityID: string, target: Machine["target"]) => Promise<InstallOutcome>;
}) {
  const { t } = useTranslation();
  const [questions, setQuestions] = useState<ConfigurationQuestion[]>([]);
  const [secretValues, setSecretValues] = useState<Record<string, string>>({});
  const [status, setStatus] = useState("");
  const [loadFailure, setLoadFailure] = useState("");
  const [refreshing, setRefreshing] = useState(false);
  const [detail, setDetail] = useState<MachineConfigurationDetail | null>(null);
  const [readiness, setReadiness] = useState<GetReadinessResponse | null>(null);
  const [grants, setGrants] = useState<CredentialGrant[]>([]);
  const [reapplying, setReapplying] = useState(false);
  const [addingCredential, setAddingCredential] = useState(false);
  const [grantIdentity, setGrantIdentity] = useState("");
  const [grantField, setGrantField] = useState("");
  const [granting, setGranting] = useState(false);
  const [apply, setApply] = useState<ApplyActivity | null>(null);
  const [installs, setInstalls] = useState<Record<string, InstallOutcome | "installing">>({});
  const mounted = useRef(true);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);

  const issues = machineIssues(machine);
  // A missing procedure on a reachable machine is version skew: the machine
  // runs an older Vrooli than this console. Re-apply runs the same old build,
  // so it is withheld; the API supplies the one update command that can work.
  const targetOnboardingIncompatible =
    loadFailure.includes("target_onboarding_incompatible") || loadFailure.includes("target_onboarding_contract_mismatch");
  const updateCommand = targetOnboardingIncompatible ? (/`([^`]+)`/.exec(loadFailure)?.[1] ?? "") : "";

  const loadConfiguration = useCallback(async () => {
    try {
      const [result, grantResult] = await Promise.all([
        getConfiguration(machine.target.id),
        listCredentialGrants(machine.target.id),
      ]);
      setQuestions(result.questions);
      setDetail(result.detail);
      setReadiness(result.readiness);
      setGrants(grantResult.grants);
      setLoadFailure("");
    } catch (error: unknown) {
      setLoadFailure(error instanceof Error ? error.message : String(error));
    }
  }, [machine.target.id]);

  useEffect(() => {
    void loadConfiguration();
  }, [loadConfiguration]);

  const refresh = () => {
    setRefreshing(true);
    void loadConfiguration().finally(() => {
      setRefreshing(false);
    });
  };

  const secrets = questions.filter((question) => question.kind === "secret");
  const regular = questions.filter((question) => question.kind !== "secret");
  const adapted = regular.map((question) => ({
    id: question.id,
    kind: question.kind,
    label: question.title,
    description: question.description,
    required: question.required,
    defaultValue: question.default,
    options: question.options?.map((value) => ({ value, label: value })),
    candidates: question.candidates?.map((candidate) => ({
      label: candidate.label,
      value: candidate.id,
      status: candidate.status,
      risk: candidate.risk,
      remediation: candidate.remediation,
    })),
    validation: question.validation,
  }));
  const fields = toGeneratedFields(adapted as OperatorInput[]) as never[];

  const submit = async (values: Record<string, unknown>) => {
    setStatus("Submitting answers through the sealed target path…");
    try {
      const nodeId = machine.target.node_id || machine.target.id;
      await Promise.all(
        secrets.map((question) =>
          answerSecret({
            nodeId,
            logicalId: question.owner || question.id.split(":")[0] || question.id,
            field: question.input_id || question.id.split(":")[1] || "value",
            value: secretValues[question.id] ?? "",
          }),
        ),
      );
      if (regular.length > 0) {
        await resolveConfiguration(
          machine.target.id,
          regular.map((question) => ({ request_id: question.id, value: answerValue(values[question.id]) })),
          readiness?.configurationRevision ?? "",
        );
      }
      setSecretValues({});
      setStatus("Answers accepted. Refresh the machine to verify readiness and drift.");
    } catch (error: unknown) {
      setStatus(error instanceof Error ? error.message : "The machine rejected the configuration answers.");
    }
  };

  const trackApply = async (runId: string, startedAt: number) => {
    for (;;) {
      if (!mounted.current) return;
      const current = await getConfigurationApplyStatus(machine.target.id, runId);
      if (!mounted.current) return;
      const summary = summarizeApplyRun(current.status, current.steps);
      if (!summary.active) {
        setApply({ phase: "done", startedAt, summary });
        await loadConfiguration();
        return;
      }
      const elapsed = Date.now() - startedAt;
      if (elapsed > APPLY_POLL_LIMIT_MS) {
        setApply({ phase: "timeout", startedAt, summary });
        return;
      }
      setApply({ phase: "running", startedAt, summary });
      await new Promise((resolve) => window.setTimeout(resolve, applyPollDelay(elapsed)));
    }
  };

  const reapply = () => {
    const startedAt = Date.now();
    setReapplying(true);
    setApply({ phase: "starting", startedAt });
    void reapplyConfiguration(machine.target.id)
      .then(async (result) => {
        const runId = result.result?.run?.runId;
        if (!runId) throw new Error("The machine accepted the re-apply but returned no run to follow.");
        // An unchanged plan returns the run that already applied it, which may
        // have finished long ago; the first read shows that outcome directly.
        await trackApply(runId, startedAt);
      })
      .catch((error: unknown) => {
        if (mounted.current) {
          setApply({ phase: "error", startedAt, message: error instanceof Error ? error.message : String(error) });
        }
      })
      .finally(() => {
        if (mounted.current) setReapplying(false);
      });
  };

  const installAgent = (capabilityID: string) => {
    if (!onInstallCapability) return;
    setInstalls((current) => ({ ...current, [capabilityID]: "installing" }));
    void onInstallCapability(capabilityID, machine.target)
      .catch((error: unknown): InstallOutcome => ({
        status: "failed",
        message: error instanceof Error ? error.message : String(error),
      }))
      .then((outcome) => {
        if (mounted.current) setInstalls((current) => ({ ...current, [capabilityID]: outcome }));
      });
  };

  return (
    <div className="space-y-3" data-testid="machine-configuration-panel">
      {apply && <ApplyActivityPanel activity={apply} onDismiss={() => { setApply(null); }} />}

      {status && (
        <p className="text-xs text-wc-text-muted" role="status" data-testid="machine-configuration-status">
          {status}
        </p>
      )}

      {loadFailure && (
        <ErrorState
          title={targetOnboardingIncompatible ? t(strings.machines.configIncompatibleTitle) : t(strings.machines.configUnavailableTitle)}
          message={targetOnboardingIncompatible
            ? (
              <>
                {t(strings.machines.configIncompatibleBody)}
                {updateCommand && (
                  <code
                    data-testid="machine-configuration-update-command"
                    className="mt-2 block select-all break-all rounded-md bg-wc-surface-input px-2 py-1 text-left font-mono text-[12px]"
                  >
                    {updateCommand}
                  </code>
                )}
              </>
            )
            : "The Bridge path could not retrieve configuration questions. Check the technical detail below and refresh after the target onboarding service is healthy; re-apply may not fix a missing backend route."}
          detail={loadFailure}
          detailLabel={t(strings.machines.technicalDetail)}
          actions={
            <div className="flex flex-wrap gap-2">
              <Button
                size="sm"
                variant="outline"
                data-testid="machine-configuration-refresh"
                pending={refreshing}
                pendingLabel={t(strings.machines.refreshing)}
                onClick={refresh}
              >
                {t(strings.machines.refresh)}
              </Button>
              {!targetOnboardingIncompatible && (
                <Button
                  size="sm"
                  variant="outline"
                  data-testid="machine-configuration-reapply"
                  pending={reapplying}
                  pendingLabel={t(strings.machines.reapplying)}
                  onClick={reapply}
                >
                  {t(strings.machines.reapply)}
                </Button>
              )}
            </div>
          }
        />
      )}

      {detail && (
        <section className="rounded-xl border border-wc-default p-3">
          <div className="flex items-center justify-between gap-3">
            <h3 className="text-[11px] font-semibold uppercase tracking-[0.14em] text-wc-text-faint">
              {t(strings.machines.profileHeading)}
            </h3>
            {!loadFailure && (
              <Button
                size="sm"
                variant="outline"
                data-testid="machine-configuration-reapply"
                pending={reapplying}
                pendingLabel={t(strings.machines.reapplying)}
                onClick={reapply}
              >
                {t(strings.machines.reapply)}
              </Button>
            )}
          </div>
          <dl className="mt-2 grid gap-1 text-xs sm:grid-cols-[auto_1fr] sm:gap-x-4">
            <dt className="text-wc-text-faint">{t(strings.machines.profileDesired)}</dt>
            <dd className="font-mono text-[11px] text-wc-text-primary">
              {detail.machine?.desiredProfileId || t(strings.machines.profileNotRecorded)}
              {detail.machine?.desiredProfileVersion ? ` (${detail.machine.desiredProfileVersion})` : ""}
            </dd>
            <dt className="text-wc-text-faint">{t(strings.machines.profileApplied)}</dt>
            <dd
              className={`font-mono text-[11px] ${detail.machine?.appliedProfileId ? "text-wc-text-primary" : "text-amber-200"}`}
            >
              {detail.machine?.appliedProfileId || t(strings.machines.profileNotRecorded)}
              {detail.machine?.appliedProfileVersion ? ` (${detail.machine.appliedProfileVersion})` : ""}
            </dd>
          </dl>
          {readiness && (
            <div className="mt-3">
              <VerdictSummary
                pass={readiness.status === ReadinessState.READY ? 1 : 0}
                fail={readiness.status === ReadinessState.READY ? 0 : Math.max(1, readiness.blockers.length)}
                unmeasured={0}
              />
              <p className="mt-2 text-xs text-wc-text-faint">
                {readiness.blockers[0]?.reason ?? readiness.degraded[0]?.reason ?? `Readiness state: ${readinessStateName(readiness.status)}`}
              </p>
            </div>
          )}
        </section>
      )}

      {issues.count > 0 && (
        <section className="rounded-xl border border-wc-default p-3" data-testid="machine-configuration-drift">
          <div className="flex items-center justify-between">
            <h3 className="text-[11px] font-semibold uppercase tracking-[0.14em] text-wc-text-faint">
              {t(strings.machines.driftHeading)}
            </h3>
            <StatusBadge tone="warning">{issues.count}</StatusBadge>
          </div>
          <ul className="mt-2 divide-y divide-wc-default">
            {/* Each row carries the action that can clear it, or says plainly
                that none exists here. Every row used to be a "Fix" that ran
                the same re-apply, which cannot install an agent or grant
                Bridge SSH trust, so most Fix presses changed nothing. */}
            {issues.drift.map((item) => {
              const reapplies = item.kind === "profile" || item.kind === "selection";
              return (
                <li
                  key={`${item.kind}:${item.name}`}
                  data-testid={`machine-drift-${item.kind}-${item.name}`}
                  className="flex items-center gap-3 py-2.5"
                >
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-xs text-wc-text-primary">{driftLabel(item.name)}</span>
                    <span className="block break-words text-[11px] text-wc-text-faint">{item.reason}</span>
                  </span>
                  {reapplies ? (
                    <Button
                      size="sm"
                      variant="outline"
                      data-testid="machine-drift-reapply"
                      pending={reapplying}
                      pendingLabel={t(strings.machines.reapplying)}
                      onClick={reapply}
                    >
                      {t(strings.machines.reapply)}
                    </Button>
                  ) : (
                    <span className="shrink-0 text-[11px] text-wc-text-faint">{t(strings.machines.driftNotFixableHere)}</span>
                  )}
                </li>
              );
            })}
            {issues.missingCapabilities.map((fact) => {
              const id = fact.key.slice("capability:".length);
              const state = installs[id];
              const outcome = typeof state === "object" ? state : undefined;
              const settled = outcome?.status === "installed" || outcome?.status === "not_applicable";
              return (
                <li key={fact.key} data-testid={`machine-drift-capability-${id}`} className="flex items-center gap-3 py-2.5">
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-xs text-wc-text-primary">{fact.label || id}</span>
                    <span
                      data-testid={`machine-drift-capability-${id}-status`}
                      className={`block break-words text-[11px] ${outcome && outcome.status !== "installed" ? "text-rose-200" : "text-wc-text-faint"}`}
                    >
                      {outcome
                        ? outcome.status === "installed"
                          ? (outcome.message ?? t(strings.launcher.agentInstalled))
                          : (outcome.message ?? t(strings.launcher.installFailed))
                        : t(strings.machines.capabilityNotInstalled)}
                    </span>
                  </span>
                  {onInstallCapability && !settled && (
                    <Button
                      size="sm"
                      variant="outline"
                      data-testid={`machine-drift-install-${id}`}
                      pending={state === "installing"}
                      pendingLabel={t(strings.machines.driftInstalling)}
                      onClick={() => { installAgent(id); }}
                    >
                      {outcome ? t(strings.launcher.installRetry) : t(strings.machines.driftInstall)}
                    </Button>
                  )}
                </li>
              );
            })}
            {issues.unavailableNodeFeatures.map((fact) => (
              <li
                key={fact.key}
                data-testid={`machine-drift-node-${fact.key.slice("node_capability:".length)}`}
                className="flex items-start gap-3 py-2.5"
              >
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-xs text-wc-text-primary">{fact.label || fact.key}</span>
                  <span className="block break-words text-[11px] text-wc-text-faint">{nodeFeatureDetail(fact.detail)}</span>
                </span>
                <span className="shrink-0 text-[11px] text-wc-text-faint">{t(strings.machines.driftNotFixableHere)}</span>
              </li>
            ))}
          </ul>
        </section>
      )}

      <section className="rounded-xl border border-wc-default p-3" data-testid="machine-credential-grants">
        <h3 className="text-[11px] font-semibold uppercase tracking-[0.14em] text-wc-text-faint">
          {t(strings.machines.credentialsHeading)}
        </h3>
        <p className="mt-1 text-[11px] text-wc-text-faint">{t(strings.machines.credentialsBody)}</p>

        {grants.length === 0 ? (
          <p className="mt-2 text-xs text-wc-text-muted">{t(strings.machines.credentialsEmpty)}</p>
        ) : (
          <ul className="mt-2 divide-y divide-wc-default">
            {grants.map((grant) => {
              const received = grant.receiptAccepted && grant.ackedGeneration >= grant.generation;
              return (
                <li key={grant.id} className="flex items-center gap-3 py-2.5">
                  <span className="min-w-0 flex-1">
                    <span className="block truncate font-mono text-[11px] text-wc-text-primary">
                      {grant.logicalId}:{grant.field}
                    </span>
                    <span className="block truncate text-[11px] text-wc-text-faint">
                      generation {grant.generation.toString()}
                      {received && grant.receiptAt
                        ? ` · ${new Date(Number(grant.receiptAt.seconds) * 1000).toLocaleString()}`
                        : ""}
                    </span>
                  </span>
                  <StatusBadge tone={received ? "success" : grant.receiptReason ? "danger" : "neutral"}>
                    {received
                      ? t(strings.machines.credentialReceived)
                      : grant.receiptReason
                        ? t(strings.machines.credentialRefused)
                        : t(strings.machines.credentialPending)}
                  </StatusBadge>
                  <Button
                    size="sm"
                    variant="danger"
                    shape="square"
                    aria-label={t(strings.machines.credentialRevoke, { name: `${grant.logicalId}:${grant.field}` })}
                    onClick={() => {
                      void revokeCredentialGrant(grant.id)
                        .then(() => {
                          setGrants((current) => current.filter((item) => item.id !== grant.id));
                        })
                        .catch((error: unknown) => {
                          setStatus(
                            error instanceof Error ? error.message : "The credential grant could not be revoked.",
                          );
                        });
                    }}
                  >
                    {t(strings.machines.forget)}
                  </Button>
                </li>
              );
            })}
          </ul>
        )}

        {addingCredential ? (
          <form
            className="mt-3 space-y-3"
            onSubmit={(event) => {
              event.preventDefault();
              setGranting(true);
              void createCredentialGrant({
                nodeId: machine.target.node_id || machine.target.id,
                logicalId: grantIdentity,
                field: grantField,
                class: "user_prompt",
                retention: "durable",
              })
                .then((grant) => {
                  setGrants((current) => [...current, grant]);
                  setGrantIdentity("");
                  setGrantField("");
                  setAddingCredential(false);
                  setStatus("Grant created; Bridge will push the sealed value when the authority has it.");
                })
                .catch((error: unknown) => {
                  setStatus(error instanceof Error ? error.message : "The credential grant could not be created.");
                })
                .finally(() => {
                  setGranting(false);
                });
            }}
          >
            <div className="grid gap-3 sm:grid-cols-2">
              <FormField
                label={t(strings.machines.credentialIdentity)}
                required
                control={
                  <Input
                    data-testid="machine-credential-identity"
                    placeholder="namespace/name"
                    value={grantIdentity}
                    onChange={(event) => {
                      setGrantIdentity(event.target.value);
                    }}
                  />
                }
              />
              <FormField
                label={t(strings.machines.credentialField)}
                required
                control={
                  <Input
                    data-testid="machine-credential-field"
                    placeholder="value"
                    value={grantField}
                    onChange={(event) => {
                      setGrantField(event.target.value);
                    }}
                  />
                }
              />
            </div>
            <div className="flex justify-end gap-2">
              <Button size="sm" variant="outline" onClick={() => { setAddingCredential(false); }}>
                {t(strings.machines.cancel)}
              </Button>
              <Button size="sm" type="submit" pending={granting} disabled={!grantIdentity || !grantField}>
                {t(strings.machines.credentialGrantSubmit)}
              </Button>
            </div>
          </form>
        ) : (
          <Button
            size="sm"
            variant="outline"
            className="mt-3"
            data-testid="machine-credential-add"
            onClick={() => { setAddingCredential(true); }}
          >
            {t(strings.machines.credentialAdd)}
          </Button>
        )}
      </section>

      {questions.length > 0 && (
        <section className="rounded-xl border border-wc-default p-3">
          <h3 className="text-[11px] font-semibold uppercase tracking-[0.14em] text-wc-text-faint">
            Outstanding questions
          </h3>
          <div className="mt-3 space-y-3">
            {secrets.map((question) => (
              <PasswordInput
                key={question.id}
                name={question.id}
                label={question.title}
                description={question.description}
                required={question.required}
                autoComplete="new-password"
                // A value pushed to a node is never read back here, so offering
                // to reveal it would promise something this surface cannot do.
                revealable={false}
                value={secretValues[question.id] ?? ""}
                onValueChange={(value) => {
                  setSecretValues((current) => ({ ...current, [question.id]: value }));
                }}
              />
            ))}
            {regular.length > 0 && (
              <GeneratedForm fields={fields} onSubmit={submit} submitLabel="Submit answers" />
            )}
            {secrets.length > 0 && regular.length === 0 && (
              <div className="flex justify-end">
                <Button size="sm" onClick={() => { void submit({}); }}>
                  Submit answers
                </Button>
              </div>
            )}
          </div>
        </section>
      )}
    </div>
  );
}

export default ConfigurationTab;
