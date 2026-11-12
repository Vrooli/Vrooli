import { Alert } from "@vrooli/react-component-library/Alert/1";
import { Checkbox } from "@vrooli/react-component-library/Checkbox/1";
import { CopyIconButton } from "@vrooli/react-component-library/CopyIconButton/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { useId, type ReactNode } from "react";

import { blockersOf, type Blocker, type BlockerChoice, type Gate, type PreflightResponse, type StateExcerpt, type TopicVerdict } from "../../api/launch";
import { PathText } from "../../components/PathText";
import { Section } from "../../components/Section";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { ReadSets } from "../commission/ReadingPanel";
import { useTranslation } from "../../i18n";

const GATE_LABEL = {
  charter_bytes: strings.preflight.gates.charter_bytes,
  stance_chars: strings.preflight.gates.stance_chars,
  stance_ending: strings.preflight.gates.stance_ending,
  kernel_chars: strings.preflight.gates.kernel_chars,
  paths_exist: strings.preflight.gates.paths_exist,
  missions: strings.preflight.gates.missions,
} as const;

/** The size gates, each with its value against its limit (limits.json). */
export function Gates({ gates }: { gates: Gate[] }) {
  const { t } = useTranslation();
  return (
    <ul className="rd-gates">
      {gates.map((gate) => (
        <li key={gate.name} className={gate.ok ? "" : "rd-gate-fail"}>
          <StatusBadge tone={gate.ok ? "success" : "danger"}>{t(gate.ok ? strings.preflight.pass : strings.preflight.fail)}</StatusBadge>
          <span className="min-w-0 flex-1">{gate.name in GATE_LABEL ? t(GATE_LABEL[gate.name as keyof typeof GATE_LABEL]) : gate.name}</span>
          <span className="rd-num">{gate.name === "stance_ending" ? null : t(strings.preflight.ofLimit, { value: gate.value || "?", limit: gate.limit })}</span>
        </li>
      ))}
    </ul>
  );
}

/** What the operator decides about earlier commissions, on the builder and on a built packet's page alike. */
export interface Decide {
  blockers: Blocker[];
  choices: Record<string, BlockerChoice | "">;
  onChoose: (commissionId: string, choice: BlockerChoice | "") => void;
  /** The builder only: a built packet's read-first is fixed in its cards. */
  readFirst?: Record<string, string>;
  onReadFirst?: (topic: string, path: string | null) => void;
}

const STATE_WORDS = {
  live: strings.preflight.stateLive,
  paused: strings.preflight.statePaused,
  limited: strings.preflight.stateLimited,
  unknown: strings.preflight.stateUnknown,
} as const;

function stateWords(state: string) {
  return state in STATE_WORDS ? STATE_WORDS[state as keyof typeof STATE_WORDS] : strings.preflight.stateOther;
}

const CHOICE_LABEL = {
  alongside: strings.preflight.choiceAlongside,
  supersede: strings.preflight.choiceSupersede,
} as const;

/**
 * Each earlier commission still holding a chosen topic, once, with what this
 * launch does about it: the desk's recommendation is already selected and
 * the operator changes it here, never on another page.
 */
export function Blockers({ blockers, choices, onChoose }: Pick<Decide, "blockers" | "choices" | "onChoose">) {
  const { t } = useTranslation();
  const name = useId();
  return (
    <ul className="rd-rows">
      {blockers.map((b) => {
        const chosen = choices[b.commissionId] ?? "";
        return (
          <li key={b.commissionId} className="rd-row" data-testid={selectors.launch.blocker}>
            <span className="rd-id font-semibold">{b.commissionId}</span>
            <span className="text-body-sm">{t(stateWords(b.state), { state: b.state, topics: b.topics.join(", ") })}</span>
            <fieldset className="rd-choice">
              <legend className="sr-only">{t(strings.preflight.choiceLegend, { id: b.commissionId })}</legend>
              {b.choices.map((choice) => (
                <label key={choice}>
                  <input type="radio" name={`${name}-${b.commissionId}`} checked={chosen === choice} onChange={() => onChoose(b.commissionId, choice)} />
                  <span>
                    {t(CHOICE_LABEL[choice])}
                    {b.recommended === choice ? ` · ${t(strings.preflight.recommended)}` : ""}
                  </span>
                </label>
              ))}
              <label>
                <input type="radio" name={`${name}-${b.commissionId}`} checked={chosen === ""} onChange={() => onChoose(b.commissionId, "")} />
                <span>{t(strings.preflight.choiceHold)}</span>
              </label>
            </fieldset>
            {b.reason ? <span className="text-caption text-app-muted-foreground">{b.recommended ? t(strings.preflight.recommendedBecause, { reason: b.reason }) : b.reason}</span> : null}
            {chosen === "" ? <span className="text-caption text-app-danger">{t(strings.preflight.holdNote)}</span> : null}
          </li>
        );
      })}
    </ul>
  );
}

/**
 * The earlier commissions with nothing to decide, folded, and the newest
 * checkpoint an owner should read first, offered already ticked.
 */
export function Earlier({ verdicts, blockers, decide }: { verdicts: TopicVerdict[]; blockers: Blocker[]; decide?: Decide }) {
  const { t } = useTranslation();
  const blocking = new Set(blockers.map((b) => b.commissionId));
  const quiet = verdicts.filter((v) => !blocking.has(v.commissionId));
  const offers = decide?.onReadFirst ? quiet.filter((v) => v.readFirst) : [];
  if (verdicts.length === 0) return <p className="text-body-sm">{t(strings.preflight.noEarlier)}</p>;
  return (
    <>
      {offers.map((v) => (
        <Checkbox
          key={`${v.topic}/${v.commissionId}`}
          label={t(strings.preflight.readFirstOn, { topic: v.topic })}
          description={<PathText path={v.readFirst} />}
          checked={decide?.readFirst?.[v.topic] === v.readFirst}
          onCheckedChange={(on) => decide?.onReadFirst?.(v.topic, on ? v.readFirst : null)}
        />
      ))}
      {quiet.length > 0 ? (
        <details className="rd-fold">
          <summary>{t(strings.preflight.nothingToDo, { count: new Set(quiet.map((v) => v.commissionId)).size })}</summary>
          <ul className="rd-list">
            {quiet.map((v) => (
              <li key={`${v.topic}/${v.commissionId}`} className="text-body-sm">
                <span className="rd-id">{v.commissionId}</span> · {v.topic}: {v.message}
              </li>
            ))}
          </ul>
        </details>
      ) : null}
    </>
  );
}

/** One message to paste, with its own copy button, so a packet can always be started by hand. */
export function MessageText({ label, text, hint }: { label: string; text: string; hint?: string }) {
  const { t } = useTranslation();
  return (
    <div className="rd-message">
      <div className="flex items-center justify-between gap-space-xs">
        <span className="font-semibold">{label}</span>
        <CopyIconButton value={text} aria-label={t(strings.launch.copyMessage, { label })} copiedLabel={t(strings.common.copied)} />
      </div>
      {hint ? <span className="text-caption text-app-muted-foreground">{hint}</span> : null}
      <pre className="rd-exact">{text}</pre>
      <span className="text-caption text-app-muted-foreground">{t(strings.launch.chars, { count: text.length })}</span>
    </div>
  );
}

/**
 * The whole preflight: errors with what to cut, what this launch decides
 * about earlier commissions, any other reason it cannot start in the API's
 * words (which name no command) with `extra` beside it (e.g. "Relaunch as a
 * new commission"), then the gates. `self` is the packet being launched: a line
 * about it is left to the never-started choice when the API offers one.
 */
export function PreflightResult({ result, decide, extra, self }: { result: PreflightResponse; decide?: Decide; extra?: ReactNode; self?: string }) {
  const { t } = useTranslation();
  const packet = result.packet;
  const blockers = decide?.blockers ?? blockersOf(result);
  const named = new Set(blockers.map((b) => b.commissionId));
  if (self && result.neverStarted?.assertable) named.add(self);
  const other = result.blockers.filter((line) => ![...named].some((id) => line.includes(`${id} `) || line.endsWith(id)));
  return (
    <div className="flex flex-col gap-space-md">
      {packet?.errors.map((e, n) => (
        <Alert key={n} tone="danger" title={e.code === "gate_overflow" ? t(strings.preflight.overflow) : t(strings.preflight.error)} description={e.message} />
      ))}
      {other.length > 0 ? (
        <Alert tone="danger" title={t(strings.preflight.blocked)} description={<ul className="rd-list">{other.map((b) => <li key={b}>{b}</li>)}</ul>} actions={extra} />
      ) : null}
      {blockers.length > 0 && decide ? (
        <Section id="blockers" title={t(strings.preflight.blockersTitle, { count: blockers.length })} note={t(strings.preflight.blockersNote)}>
          <Blockers blockers={blockers} choices={decide.choices} onChoose={decide.onChoose} />
        </Section>
      ) : null}
      <Section id="verdicts" title={t(strings.preflight.verdictsTitle)}>
        <Earlier verdicts={result.topics} blockers={blockers} decide={decide} />
      </Section>
      {result.packetState ? <PacketState excerpt={result.packetState} /> : null}
      {result.ownerReading.length > 0 ? (
        <details className="rd-fold" open={result.ownerReading.some((set) => set.flags.length > 0)}>
          <summary>{t(strings.preflight.ownerReading)}</summary>
          <p className="text-caption text-app-muted-foreground">{t(strings.preflight.ownerReadingNote)}</p>
          <ReadSets sets={result.ownerReading} />
        </details>
      ) : null}
      {packet && packet.gates.length > 0 ? (
        <details className="rd-fold" open={packet.gates.some((g) => !g.ok)}>
          <summary>{t(strings.preflight.gatesTitle)}</summary>
          <Gates gates={packet.gates} />
        </details>
      ) : null}
    </div>
  );
}

/** The start of an existing packet's own STATE.md, verbatim, or why there is none. Never blocking. */
function PacketState({ excerpt }: { excerpt: StateExcerpt }) {
  const { t } = useTranslation();
  return (
    <Section id="packet-state" title={t(strings.preflight.packetState)} note={t(strings.preflight.packetStateNote)}>
      {excerpt.text ? <pre className="rd-exact">{excerpt.text}</pre> : <p className="text-body-sm">{excerpt.reason || t(strings.common.unknown)}</p>}
      {excerpt.truncated ? <p className="text-caption text-app-muted-foreground">{t(strings.preflight.packetStateTruncated)}</p> : null}
      {excerpt.path ? <PathText path={excerpt.path} /> : null}
    </Section>
  );
}

/** What the launch does about earlier commissions, named in the one confirmation. */
export function ChoiceSummary({ choices }: { choices: Record<string, BlockerChoice | ""> }) {
  const { t } = useTranslation();
  const decided = Object.entries(choices).filter(([, choice]) => choice);
  if (decided.length === 0) return null;
  return (
    <ul className="rd-list text-body-sm">
      {decided.map(([id, choice]) => (
        <li key={id}>{t(choice === "supersede" ? strings.launch.willSupersede : strings.launch.willRunAlongside, { id })}</li>
      ))}
    </ul>
  );
}
