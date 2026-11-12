import { useQueryClient } from "@tanstack/react-query";
import { Alert } from "@vrooli/react-component-library/Alert/1";
import { Button } from "@vrooli/react-component-library/Button/2";
import { LoadingState } from "@vrooli/react-component-library/LoadingState/1";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { Checkbox } from "@vrooli/react-component-library/Checkbox/1";
import { useEffect, useState } from "react";
import { Link, useLocation, useNavigate, useParams, useSearchParams } from "react-router-dom";

import { CommissionState, useCommission } from "../api/activity";
import { blockersOf, ControlAction, launchApi, useLaunchMessages, useLaunchOptions, useLaunchRecord, usePreflightPrepared, type BlockerChoice, type LaunchRecord } from "../api/launch";
import { AccessWords } from "../components/AccessWords";
import { Confirm } from "../components/Confirm";
import { PathText } from "../components/PathText";
import { QueryError, QueryView } from "../components/QueryView";
import { Section } from "../components/Section";
import { StateBadge } from "../components/StateBadge";
import { When } from "../components/When";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { withRecommendations } from "../lib/choices";
import { toDate } from "../lib/format";
import { ChoiceSummary, MessageText, PreflightResult } from "./launch/Preflight";

/** The automatic launch steps in order (SYSTEM.md §5; launcher.go). */
const STEPS = ["created", "composer_ready", "stance_sent", "ready_seen", "goal_sent", "goal_verified"] as const;
const STEP_LABEL = {
  created: strings.launchStatus.steps.created,
  composer_ready: strings.launchStatus.steps.composer_ready,
  stance_sent: strings.launchStatus.steps.stance_sent,
  ready_seen: strings.launchStatus.steps.ready_seen,
  goal_sent: strings.launchStatus.steps.goal_sent,
  goal_verified: strings.launchStatus.steps.goal_verified,
} as const;

/**
 * One packet's launch: a prepared packet is checked and launched as it is
 * (or started by hand with its two messages); a launch in progress shows each
 * step; a failed start offers discard and relaunch from the same packet.
 */
export function LaunchStatusPage() {
  const { t } = useTranslation();
  const { id = "" } = useParams();
  const [params] = useSearchParams();
  const location = useLocation();
  const launchError = (location.state as { launchError?: string } | null)?.launchError;
  const commission = useCommission(id);
  const record = useLaunchRecord(id);
  const c = commission.data?.commission;
  const launch = record.data?.launch;
  const notStarted = !launch?.running && (launch?.steps.length ?? 0) === 0 && !c?.session;
  // A packet the desk cannot place (changed after it was built) is still offered: the check says whether it may start.
  const prepared = notStarted && (c?.state === CommissionState.PREPARED || c?.state === CommissionState.UNKNOWN);
  const uncertain = prepared && c.state === CommissionState.UNKNOWN;
  const finished = launch && !launch.failedStep && !launch.running && launch.steps.some((s) => s.step === "goal_verified");
  return (
    <div data-testid={selectors.pages.launchStatus} className="rd-page">
      <Link to="/launch" className="rd-back">{t(strings.launchStatus.back)}</Link>
      <PageHeader
        headingId="launch-status-heading"
        title={c?.title?.value || id}
        eyebrow={<span className="rd-id">{id}</span>}
        description={c ? <StateBadge state={c.state} stalled={c.stalled} /> : null}
      />
      {launchError ? <Alert tone="danger" title={t(strings.launchStatus.launchCallFailed)} description={launchError} /> : null}
      {commission.isError ? <QueryError error={commission.error} onRetry={() => void commission.refetch()} /> : null}
      {record.isPending || commission.isPending ? <LoadingState label={t(strings.common.loading)} /> : null}
      {launch ? <Progress launch={launch} /> : null}
      {launch?.failedStep ? <Failed id={id} launch={launch} /> : null}
      {finished ? (
        <Alert
          tone="success"
          title={t(strings.launchStatus.launched)}
          actions={<Link to={`/commissions/${encodeURIComponent(id)}`} className="rd-more">{t(strings.launchStatus.openCommission)}</Link>}
        />
      ) : null}
      {uncertain ? (
        <Alert tone="warning" title={t(strings.launchStatus.uncertainTitle)} description={c.stateEvidence.map((e) => e.reason).filter(Boolean).join(" ") || c.stateRule} />
      ) : null}
      {prepared ? <PreparedLaunch id={id} manualFirst={params.get("manual") === "1"} uncertain={uncertain} /> : null}
      {c && !prepared && !launch?.running && !launch?.failedStep && !finished ? (
        <p className="text-body-sm">
          {t(strings.launchStatus.nothingHere)}{" "}
          <Link to={`/commissions/${encodeURIComponent(id)}`} className="rd-more">{t(strings.launchStatus.openCommission)}</Link>
        </p>
      ) : null}
    </div>
  );
}

function Progress({ launch }: { launch: LaunchRecord }) {
  const { t } = useTranslation();
  if (launch.steps.length === 0 && !launch.running) return null;
  const done = new Map(launch.steps.map((s) => [s.step, toDate(s.at)]));
  const next = STEPS.find((s) => !done.has(s));
  return (
    <Section id="progress" title={t(launch.running ? strings.launchStatus.running : strings.launchStatus.steps.title)}>
      <ol className="rd-steps">
        {STEPS.map((step) => {
          const at = done.get(step);
          const state = done.has(step) ? "done" : launch.failedStep === step ? "failed" : launch.running && step === next ? "now" : "todo";
          return (
            <li key={step} data-state={state}>
              <span className="rd-step-dot" aria-hidden="true" />
              <span className="min-w-0 flex-1">{t(STEP_LABEL[step])}</span>
              <span className="text-caption text-app-muted-foreground">
                {at ? <When date={at} /> : t(STATE_WORD[state])}
              </span>
            </li>
          );
        })}
      </ol>
      {launch.webConsoleSessionId ? (
        <p className="text-body-sm">
          {t(strings.launchStatus.session)} <span className="rd-id">{launch.webConsoleSessionId}</span>
        </p>
      ) : null}
      {launch.path ? <PathText path={launch.path} /> : null}
    </Section>
  );
}

const STATE_WORD = {
  done: strings.launchStatus.stepDone,
  failed: strings.launchStatus.stepFailed,
  now: strings.launchStatus.stepNow,
  todo: strings.launchStatus.stepTodo,
} as const;

/** A failed start: discard it with a reason, or relaunch from the same packet. */
function Failed({ id, launch }: { id: string; launch: LaunchRecord }) {
  const { t } = useTranslation();
  const client = useQueryClient();
  const [open, setOpen] = useState<"discard" | null>(null);
  const discarded = toDate(launch.discardedAt);
  return (
    <Section id="failed" title={t(strings.launchStatus.failedTitle)}>
      <Alert tone="danger" title={t(strings.launch.failedAt, { step: launch.failedStep, reason: launch.failedReason })} />
      {discarded ? (
        <p className="text-body-sm">
          {t(strings.commission.discarded, { reason: launch.discardedReason })} <When date={discarded} />
        </p>
      ) : null}
      <div className="rd-actions">
        {!discarded ? <Button variant="danger" onClick={() => setOpen("discard")}>{t(strings.control.labels.discard)}</Button> : null}
        <Relaunch id={id} />
      </div>
      <Confirm
        open={open === "discard"}
        destructive
        title={t(strings.control.discardTitle)}
        description={t(launch.webConsoleSessionId ? strings.control.discardOpen : strings.control.discard)}
        confirmLabel={t(strings.control.labels.discard)}
        reasonLabel={t(strings.control.discardReason)}
        onClose={() => setOpen(null)}
        onConfirm={async (reason) => {
          await launchApi.control(id, ControlAction.DISCARD, reason);
          await client.invalidateQueries({ queryKey: ["launch", "record", id] });
        }}
      />
    </Section>
  );
}

/**
 * A built packet launched as it is: its check with what to do about earlier
 * commissions (the desk's recommendation selected), then one Launch, with the
 * manual start (the two messages, a read) beside it (I3). A packet the desk
 * cannot place (`unknown`) launches only on the operator's word that it never
 * started, and only when the desk found no session or goal for it; its two
 * messages stay hidden until then, since pasting them could start a second
 * supervisor (rulings V1, P1). "Relaunch as a new commission" stays offered.
 */
function PreparedLaunch({ id, manualFirst, uncertain }: { id: string; manualFirst: boolean; uncertain: boolean }) {
  const { t } = useTranslation();
  const client = useQueryClient();
  const options = useLaunchOptions();
  const [choices, setChoices] = useState<Record<string, BlockerChoice | "">>({});
  const [asserted, setAsserted] = useState(false);
  const decisions = { choices, assertNeverStarted: uncertain && asserted };
  const check = usePreflightPrepared(id, decisions);
  const texts = useLaunchMessages(id, !uncertain || asserted);
  const [open, setOpen] = useState(false);
  const result = check.data;
  const blockers = result ? blockersOf(result) : [];
  useEffect(() => {
    if (result) setChoices((prev) => withRecommendations(blockersOf(result), prev) ?? prev);
  }, [result]);
  const automatic = options.data?.automaticAvailable === true;
  const launchable = result?.launchable === true && !check.isFetching;
  const never = result?.neverStarted;
  return (
    <>
      {uncertain ? (
        never?.assertable ? (
          <Section id="never-started" title={t(strings.launchStatus.neverStartedTitle)} note={t(strings.launchStatus.neverStartedNote)}>
            <Checkbox label={t(strings.launchStatus.neverStarted)} description={never.reason} checked={asserted} onCheckedChange={setAsserted} />
          </Section>
        ) : (
          <Alert tone="info" title={t(strings.launchStatus.uncertainManualTitle)} description={never?.reason || t(strings.launchStatus.uncertainManual)} actions={<Relaunch id={id} />} />
        )
      ) : null}
      <Section id="check" title={t(strings.launchStatus.check)}>
        <QueryView query={check}>
          {(r) => <PreflightResult result={r} self={id} decide={{ blockers, choices, onChoose: (cid, choice) => setChoices((prev) => ({ ...prev, [cid]: choice })) }} extra={<Relaunch id={id} />} />}
        </QueryView>
      </Section>
      {automatic && !manualFirst ? (
        <div className="rd-launch-bar">
          <Button variant="primary" disabled={!launchable} onClick={() => setOpen(true)} data-testid={selectors.launch.launchButton}>
            {t(strings.launch.launch)}
          </Button>
        </div>
      ) : options.data && !automatic ? (
        <Alert tone="info" title={t(strings.launch.manualModeTitle)} description={t(strings.launch.manualMode, { reason: options.data.automaticReason || t(strings.common.unknown) })} />
      ) : null}
      {uncertain && !asserted ? null : (
        <Section id="manual" title={t(strings.launchStatus.manualTitle)} note={t(strings.launchStatus.manualNote)}>
          <QueryView query={texts}>
            {(m) => (
              <>
                <ol className="rd-list text-body-sm">
                  <li>{t(strings.launchStatus.manual1)}</li>
                  <li>{t(strings.launchStatus.manual2)}</li>
                  <li>{t(strings.launchStatus.manual3)}</li>
                  <li>{t(strings.launchStatus.manual4)}</li>
                </ol>
                <MessageText label={t(strings.launch.stance)} text={m.stance} />
                <MessageText label={t(strings.launch.kernelGoal)} text={`/goal ${m.kernel}`} hint={t(strings.launch.kernelHint)} />
                {m.charterPath ? (
                  <p className="text-body-sm">
                    {t(strings.launchStatus.charterAt)} <PathText path={m.charterPath} />
                  </p>
                ) : null}
              </>
            )}
          </QueryView>
        </Section>
      )}
      <Confirm
        open={open}
        title={t(strings.launch.confirmTitle)}
        description={t(strings.launchStatus.confirm, { id })}
        confirmLabel={t(strings.launch.launch)}
        onClose={() => setOpen(false)}
        onConfirm={async () => {
          await launchApi.launch(id, decisions);
          await client.invalidateQueries({ queryKey: ["launch", "record", id] });
          await client.invalidateQueries({ queryKey: ["activity"] });
        }}
      >
        <ol className="rd-list text-body-sm">
          <li>{t(strings.launch.confirmStep2)}</li>
          <li>{t(strings.launch.confirmStep3)}</li>
          <li>{t(strings.launch.confirmStep4)}</li>
        </ol>
        <ChoiceSummary choices={choices} />
        {decisions.assertNeverStarted ? <p className="text-body-sm">{t(strings.launchStatus.assertRecorded)}</p> : null}
        {options.data ? (
          <p className="text-body-sm">
            <span className="font-semibold">{t(strings.launch.access)}</span> <AccessWords said={options.data.webConsoleAccess} />
          </p>
        ) : null}
      </Confirm>
    </>
  );
}

/**
 * "Relaunch as a new commission": builds a new packet from the same choices
 * that supersedes this one, after one confirmation, and opens it.
 */
function Relaunch({ id }: { id: string }) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button variant="primary" onClick={() => setOpen(true)}>{t(strings.launchStatus.relaunchNew)}</Button>
      <Confirm
        open={open}
        title={t(strings.launchStatus.relaunchTitle)}
        description={t(strings.launchStatus.relaunchDescription, { id })}
        confirmLabel={t(strings.launchStatus.relaunchBuild)}
        onClose={() => setOpen(false)}
        onConfirm={async () => {
          const res = await launchApi.relaunch(id);
          const next = res.packet?.commissionId;
          if (!res.packet?.ok || !next) throw new Error(res.packet?.errors.map((e) => e.message).join("; ") || t(strings.launch.buildFailed));
          navigate(`/launch/${encodeURIComponent(next)}`);
        }}
      />
    </>
  );
}
