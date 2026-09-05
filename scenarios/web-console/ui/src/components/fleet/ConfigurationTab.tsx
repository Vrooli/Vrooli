import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
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

export function ConfigurationTab({ machine }: { machine: Machine }) {
  const { t } = useTranslation();
  const [questions, setQuestions] = useState<ConfigurationQuestion[]>([]);
  const [secretValues, setSecretValues] = useState<Record<string, string>>({});
  const [status, setStatus] = useState("");
  const [loadFailure, setLoadFailure] = useState("");
  const [detail, setDetail] = useState<MachineConfigurationDetail | null>(null);
  const [grants, setGrants] = useState<CredentialGrant[]>([]);
  const [reapplying, setReapplying] = useState(false);
  const [addingCredential, setAddingCredential] = useState(false);
  const [grantIdentity, setGrantIdentity] = useState("");
  const [grantField, setGrantField] = useState("");
  const [granting, setGranting] = useState(false);

  const issues = machineIssues(machine);

  useEffect(() => {
    void Promise.all([getConfiguration(machine.target.id), listCredentialGrants(machine.target.id)])
      .then(([result, grantResult]) => {
        setQuestions(result.questions);
        setDetail(result.detail);
        setGrants(grantResult.grants);
        setLoadFailure("");
      })
      .catch((error: unknown) => {
        setLoadFailure(error instanceof Error ? error.message : String(error));
      });
  }, [machine.target.id]);

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
          regular.map((question) => ({ request_id: question.id, value: String(values[question.id] ?? "") })),
        );
      }
      setSecretValues({});
      setStatus("Answers accepted. Refresh the machine to verify readiness and drift.");
    } catch (error: unknown) {
      setStatus(error instanceof Error ? error.message : "The machine rejected the configuration answers.");
    }
  };

  const trackApply = async (runId: string) => {
    for (let attempt = 0; attempt < 180; attempt += 1) {
      const current = await getConfigurationApplyStatus(machine.target.id, runId);
      const run = current.result as { status?: string; items?: Array<{ name?: string; outcome?: string }> };
      setStatus(`Re-apply ${run.status ?? "running"}`);
      if (run.status && !["pending", "applying"].includes(run.status)) {
        const refreshed = await getConfiguration(machine.target.id);
        setDetail(refreshed.detail);
        setQuestions(refreshed.questions);
        setStatus(`Re-apply ${run.status}; configuration evidence refreshed.`);
        return;
      }
      await new Promise((resolve) => window.setTimeout(resolve, 1000));
    }
    setStatus("Re-apply is still running; the durable run remains available for refresh.");
  };

  const reapply = () => {
    setReapplying(true);
    setStatus("Re-applying the desired configuration…");
    void reapplyConfiguration(machine.target.id)
      .then(async (result) => {
        const runId = (result.result as { run_id?: string })?.run_id;
        if (runId) await trackApply(runId);
        else setStatus("Re-apply returned no durable run id.");
      })
      .catch((error: unknown) =>
        setStatus(error instanceof Error ? error.message : "The configuration could not be re-applied."),
      )
      .finally(() => setReapplying(false));
  };

  return (
    <div className="space-y-3" data-testid="machine-configuration-panel">
      {loadFailure && (
        <ErrorState
          title={t(strings.machines.configUnavailableTitle)}
          message="Its bridge agent could not answer. Re-apply the profile to bring it back into line."
          detail={loadFailure}
          detailLabel={t(strings.machines.technicalDetail)}
          actions={
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
          {detail.readiness && (
            <div className="mt-3">
              <VerdictSummary
                pass={detail.readiness.ready ? 1 : 0}
                fail={detail.readiness.ready ? 0 : (detail.readiness.reasons?.length ?? 1)}
                unmeasured={0}
              />
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
            {issues.drift.map((item) => (
              <li key={`${item.kind}:${item.name}`} className="flex items-center gap-3 py-2.5">
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-xs text-wc-text-primary">{driftLabel(item.name)}</span>
                  <span className="block truncate text-[11px] text-wc-text-faint">{item.reason}</span>
                </span>
                <Button size="sm" variant="outline" shape="square" disabled={reapplying} onClick={reapply}>
                  {t(strings.machines.driftFix)}
                </Button>
              </li>
            ))}
            {issues.missingCapabilities.map((fact) => (
              <li key={fact.key} className="flex items-center gap-3 py-2.5">
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-xs text-wc-text-primary">
                    {fact.label || fact.key.slice("capability:".length)}
                  </span>
                  <span className="block truncate text-[11px] text-wc-text-faint">
                    Not reported by this machine
                  </span>
                </span>
                <Button size="sm" variant="outline" shape="square" disabled={reapplying} onClick={reapply}>
                  {t(strings.machines.driftFix)}
                </Button>
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
                        .then(() => setGrants((current) => current.filter((item) => item.id !== grant.id)))
                        .catch((error: unknown) =>
                          setStatus(
                            error instanceof Error ? error.message : "The credential grant could not be revoked.",
                          ),
                        );
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
                .catch((error: unknown) =>
                  setStatus(error instanceof Error ? error.message : "The credential grant could not be created."),
                )
                .finally(() => setGranting(false));
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
                    onChange={(event) => setGrantIdentity(event.target.value)}
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
                    onChange={(event) => setGrantField(event.target.value)}
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
                onValueChange={(value) =>
                  setSecretValues((current) => ({ ...current, [question.id]: value }))
                }
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

      {status && (
        <p className="text-xs text-wc-text-muted" role="status" data-testid="machine-configuration-status">
          {status}
        </p>
      )}
    </div>
  );
}

export default ConfigurationTab;
