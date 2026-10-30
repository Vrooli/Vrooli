import { useState } from "react";
import { Link } from "react-router-dom";
import { Bug } from "lucide-react";
import { Card } from "@vrooli/react-component-library/Card/1";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { Table } from "@vrooli/react-component-library/Table/1";
import { ButtonGroup } from "@vrooli/react-component-library/ButtonGroup/1";
import { Toggle } from "@vrooli/react-component-library/Toggle/1";
import type { UsageInterventions, UsageToolFinding, UsageToolGuess, UsageToolSub } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { DataState } from "../../components/feedback/DataState";
import { formatCode, formatCount, formatPercent, formatTokens } from "../../lib/format";
import { useFrictionActions, useUsageFriction } from "./api";

const STATUSES = ["open", "reported", "dismissed"] as const;
type FrictionStatus = (typeof STATUSES)[number];

const BUTTON = "inline-flex whitespace-nowrap min-h-11 items-center justify-center rounded-md border border-border px-3 text-sm font-medium hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50 sm:min-h-8";
const CELL = "max-md:!block max-md:!border-0 max-md:!px-0 max-md:!py-0.5";

function Kpi({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="text-xl font-bold tabular-nums">{value}</div>
    </div>
  );
}

type Pending = "report" | "dismiss" | undefined;

/** What a row's actions need: a tool finding and the start of the window it was listed in. */
interface Actionable {
  fingerprint: string;
  since: string;
  status: string;
  bugId: string;
  owner: string;
}

/** The row's actions: one button each, confirmed inline (no dialog) and disabled while the call is out. */
function RowActions({ item }: { item: Actionable }) {
  const { report, dismiss } = useFrictionActions();
  const [confirming, setConfirming] = useState<Pending>();
  const busy = report.isPending || dismiss.isPending;
  const error = (report.error ?? dismiss.error) as Error | null;
  if (item.status === "reported") return <StatusBadge tone="success">Reported · {item.bugId || "filed"}</StatusBadge>;
  if (item.status === "dismissed") return <StatusBadge tone="neutral">Dismissed</StatusBadge>;
  const run = (action: "report" | "dismiss") => (action === "report" ? report : dismiss).mutate({ fingerprint: item.fingerprint, since: item.since }, { onSettled: () => setConfirming(undefined) });
  if (confirming) {
    return (
      <div className="flex flex-wrap items-center gap-2" role="group" aria-label="Confirm">
        <span className="text-sm">{confirming === "report" ? `File a bug with ${item.owner || "its owner"}?` : "Dismiss this problem?"}</span>
        <button type="button" disabled={busy} onClick={() => run(confirming)} className={`${BUTTON} bg-primary text-primary-foreground hover:bg-primary/90`}>{busy ? "Working…" : "Confirm"}</button>
        <button type="button" disabled={busy} onClick={() => setConfirming(undefined)} className={BUTTON}>Cancel</button>
      </div>
    );
  }
  return (
    <div className="flex flex-col items-start gap-1 md:items-end">
      <div className="flex gap-2">
        <button type="button" onClick={() => setConfirming("report")} className={`${BUTTON} bg-primary text-primary-foreground hover:bg-primary/90`}>Report bug</button>
        <button type="button" onClick={() => setConfirming("dismiss")} className={BUTTON}>Dismiss</button>
      </div>
      {error ? <span role="alert" className="text-xs text-destructive">Could not complete: {error.message}</span> : null}
    </div>
  );
}

/** Column labels return inline on a phone, where the header row is hidden. */
function Label({ children }: { children: string }) {
  return <span className="mr-2 text-xs text-muted-foreground md:hidden">{children}</span>;
}

/** What a finding or action adds to its sentence: the lines a root cause rolls up, the guessed commands with the alias to add, and printed against read tokens. */
export function FrictionDetail({ rootCause, subs, guesses, printedTokens = 0n, readTokens = 0n }: { rootCause: string; subs: UsageToolSub[]; guesses: UsageToolGuess[]; printedTokens?: bigint; readTokens?: bigint }) {
  const rolledUp = subs.length > 1;
  if (!rolledUp && guesses.length === 0 && !(printedTokens > 0n && readTokens > 0n)) return null;
  return (
    <div className="mt-1 flex flex-col gap-1 text-xs text-muted-foreground" data-testid="usage-friction-detail">
      {rolledUp ? (
        <div>
          Root cause {formatCode(rootCause)} rolls up {subs.length} lines:
          <ul className="m-0 list-none p-0" aria-label="Rolled-up lines">
            {subs.map((sub) => <li key={`${sub.kind}/${sub.signature}`} className="break-words">{sub.kind} in {formatCount(sub.calls)} calls · <code>{sub.signature}</code></li>)}
          </ul>
        </div>
      ) : null}
      {guesses.length > 0 ? (
        <ul className="m-0 list-none p-0" aria-label="Guessed commands">
          {guesses.map((guess) => (
            <li key={guess.token} className="break-words">
              guessed <code>{guess.token}</code> ×{formatCount(guess.calls)}: {guess.suggestion ? <>add alias <code>{guess.token}</code> → <code>{guess.suggestion}</code></> : "fix the help"}
            </li>
          ))}
        </ul>
      ) : null}
      {printedTokens > 0n && readTokens > 0n ? <div>Largest call printed ~{formatTokens(printedTokens)} tokens; the model read {formatTokens(readTokens)}</div> : null}
    </div>
  );
}

function ToolRow({ finding, since }: { finding: UsageToolFinding; since: string }) {
  return (
    <tr data-testid="usage-tool-finding" className="max-md:block max-md:border-b max-md:border-border max-md:py-2">
      <td className={CELL}>
        <b className="break-words">{finding.description}</b>
        <div className="text-xs text-muted-foreground">{formatCode(finding.kind)}</div>
        <FrictionDetail rootCause={finding.rootCause} subs={finding.subs} guesses={finding.guesses} printedTokens={finding.printedTokens} readTokens={finding.readTokens} />
        {finding.exampleCommands.length > 0 ? (
          <ul className="mt-1 flex flex-col gap-0.5" aria-label="Example commands">
            {finding.exampleCommands.map((example) => <li key={example}><code className="break-all text-xs">{example}</code></li>)}
          </ul>
        ) : null}
      </td>
      <td className={CELL}><Label>Owner</Label><code className="break-words text-sm">{finding.owner}</code></td>
      <td className={`${CELL} whitespace-nowrap tabular-nums md:text-right`}><Label>Seen</Label>{formatCount(finding.calls)} calls · {formatCount(finding.runs)} {finding.runs === 1n ? "run" : "runs"}</td>
      <td className={`${CELL} tabular-nums md:text-right`}>
        <Label>Cost</Label>
        {/* One line per kind of harm, each over the base the API divided it by: a roll-up's failures and notices are shares of different totals. */}
        <ul className="m-0 list-none p-0 text-xs" aria-label="Cost by kind of harm">
          {finding.shares.map((share) => <li key={share.kind} className="break-words">{share.sentence}</li>)}
        </ul>
      </td>
      <td className={`${CELL} md:text-right`}><RowActions item={{ fingerprint: finding.fingerprint, since, status: finding.status, bugId: finding.bugId, owner: finding.owner }} /></td>
    </tr>
  );
}

/** Operator messages the classifier put in a friction category that passed the accuracy gate, or why none is shown. */
function Interventions({ section }: { section: UsageInterventions }) {
  const summary = `${formatCount(section.operatorMessages)} operator messages, ${formatPercent(section.coverage, 0)} classified, over ${section.agentHours.toFixed(1)} agent-hours`;
  return (
    <Card className="flex flex-col gap-3 p-4" data-testid="usage-interventions">
      <div>
        <h3 className="text-base font-semibold">Operator interventions</h3>
        <p className="text-sm text-muted-foreground" data-testid="usage-interventions-coverage">{summary}.</p>
      </div>
      {section.categories.length === 0 ? (
        <p className="text-sm" role="status" data-testid="usage-interventions-empty">None shown: {section.unshownReason}. This is not "no friction".</p>
      ) : (
        <>
          <Table>
            <table>
              <thead className="max-md:sr-only">
                <tr><th scope="col">Category</th><th scope="col">Owner</th><th scope="col" className="md:text-right">Interventions</th><th scope="col" className="md:text-right">Agent work before</th></tr>
              </thead>
              <tbody>
                {section.categories.map((category) => (
                  <tr key={category.category} data-testid="usage-intervention-category" className="max-md:block max-md:border-b max-md:border-border max-md:py-2">
                    <td className={CELL}>
                      <b>{category.pooled ? "Operator friction (category unproven)" : formatCode(category.category)}</b>
                      {category.repeats > 0n ? <div className="text-xs text-muted-foreground">{formatCount(category.repeats)} repeat reports</div> : null}
                    </td>
                    <td className={CELL}>
                      <Label>Owner</Label>
                      {category.pooled ? (
                        <ul className="m-0 list-none p-0 text-xs text-muted-foreground" aria-label="Hints, unproven">
                          {category.hints.map((hint) => <li key={hint.category}>{formatCount(hint.count)} look like {formatCode(hint.category)} (unproven): {hint.owner}</li>)}
                        </ul>
                      ) : category.owner}
                    </td>
                    <td className={`${CELL} whitespace-nowrap tabular-nums md:text-right`}><Label>Interventions</Label>{formatCount(category.interventions)} · {category.perAgentHour.toFixed(2)} per agent-hour</td>
                    <td className={`${CELL} tabular-nums md:text-right`}><Label>Agent work before</Label>{formatCount(category.workRequests)} requests · {formatCount(category.workToolCalls)} tool calls · {formatTokens(category.workTokens)} tokens</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </Table>
          <ul className="flex flex-col gap-1" aria-label="Example messages">
            {section.examples.map((example) => (
              <li key={example.eventId} className="text-sm" data-testid="usage-intervention-example">
                <span className="text-muted-foreground">{formatCode(example.category)}{example.pooled ? " (unproven)" : ""}: </span>“{example.text}” <Link to={`/runs/${encodeURIComponent(example.runId)}`} className="underline">open run</Link>
                {example.repeatOfEventId ? <span className="text-muted-foreground"> · repeats an earlier report</span> : null}
              </li>
            ))}
          </ul>
        </>
      )}
    </Card>
  );
}

function Header() {
  return (
    <thead className="max-md:sr-only">
      <tr><th scope="col">What keeps going wrong</th><th scope="col">Owner</th><th scope="col" className="md:text-right">Seen</th><th scope="col" className="md:text-right">Cost</th><th scope="col"><span className="sr-only">Actions</span></th></tr>
    </thead>
  );
}

/** Usage › Friction: owner-level tool problems in plain words, each with an owner, a cost, and Report bug or Dismiss, beside what the profiles could see; then the classified operator interventions the accuracy gate admits. */
export function UsageFriction() {
  const [status, setStatus] = useState<FrictionStatus>("open");
  const friction = useUsageFriction();
  const findings = (friction.data?.toolFindings ?? []).filter((finding) => finding.status === status);
  const coverage = friction.data?.coverage;
  const interventions = friction.data?.interventions;
  const below = friction.data?.belowFloor ?? 0n;
  return (
    <div className="flex flex-col gap-4" data-testid="usage-friction">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div className="grid grid-cols-2 gap-x-8 gap-y-3 sm:grid-cols-4">
          <Kpi label="Open tool problems" value={formatCount(friction.data?.open)} />
          <Kpi label="Operator interventions" value={!interventions?.shown ? "not shown" : formatCount(interventions.shown)} />
          <Kpi label="Outcome known" value={coverage ? formatPercent(coverage.outcomeKnownShare, 0) : "—"} />
          <Kpi label="Reported to owners" value={formatCount(friction.data?.reported)} />
        </div>
        <ButtonGroup label="Status" collapse="never" className="flex-wrap">
          {STATUSES.map((value) => <Toggle key={value} size="lg" pressed={status === value} onPressedChange={() => setStatus(value)}>{formatCode(value)}</Toggle>)}
        </ButtonGroup>
      </div>
      {coverage ? <p className="text-sm text-muted-foreground" data-testid="usage-friction-coverage">Seen since {friction.data?.windowStart.slice(0, 10)}: {coverage.sentence}.</p> : null}
      {below > 0n ? <p className="text-sm text-muted-foreground" data-testid="usage-friction-below">{formatCount(below)} more {below === 1n ? "finding scores" : "findings score"} under the materiality floor ({formatPercent(friction.data?.materialityScore ?? 0, 0)}): counted, not listed.</p> : null}
      <Card className="p-0">
        <DataState
          subject="Friction"
          loading={friction.isLoading}
          error={friction.error}
          isEmpty={findings.length === 0}
          onRetry={() => void friction.refetch()}
          empty={<EmptyState icon={<Bug aria-hidden="true" />} title={`No ${status} problems`} description={coverage ? `None met its threshold in what was seen: ${coverage.sentence}. Unknown outcomes cannot show failures.` : "Nothing was seen in the window, so there is no telling whether tools caused friction."} />}
        >
          <Table>
            <table>
              <Header />
              <tbody>
                {findings.map((finding) => <ToolRow key={finding.fingerprint} finding={finding} since={friction.data?.windowStart ?? ""} />)}
              </tbody>
            </table>
          </Table>
        </DataState>
      </Card>
      {interventions ? <Interventions section={interventions} /> : null}
      <p className="text-sm text-muted-foreground">Tool problems rank by √(their share of the window's harm in their own unit × the share of runs that show it): failed calls, command time or context tax. Report bug files one to its owner through the report-bug path with example commands and evidence runs, and marks the row Reported. Operator interventions are shown per category only once the classifier has passed its accuracy gate for it.</p>
    </div>
  );
}
