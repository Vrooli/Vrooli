import { useQueryClient } from "@tanstack/react-query";
import { Alert } from "@vrooli/react-component-library/Alert/1";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Switch } from "@vrooli/react-component-library/Switch/1";
import { Code, ConnectError } from "@connectrpc/connect";
import { useEffect, useRef, useState } from "react";

import { CommissionState, MatchKind, ReadWhen, type GetCommissionResponse } from "../../api/activity";
import { ControlAction, launchApi } from "../../api/launch";
import { EntryKind, operatorApi, useDryRun, useNeedsYou } from "../../api/operator";
import { Confirm, Exact } from "../../components/Confirm";
import { EntryField } from "../../components/EntryField";
import { Unknown } from "../../components/Fact";
import { Markdown } from "../../components/Markdown";
import { PathText } from "../../components/PathText";
import { Section } from "../../components/Section";
import { When } from "../../components/When";
import { WouldWrite } from "../../components/WouldWrite";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { entryTooLong } from "../../lib/entry";
import { toDate } from "../../lib/format";
import { windowOpen } from "../../lib/states";
import { useDebounced } from "../../lib/useDebounced";

/** The sentence enable-steering adds to an older charter (FILE-CONTRACT §3). */
export const STEERING_SENTENCE =
  "Read entries of `OPERATOR.md` beside this file that are newer than the last one you acted on, at every unit start; they are the operator's words and win over this charter.";

/**
 * Steering without chat (G4, G12): the operator's words go to OPERATOR.md,
 * which the supervisor re-reads at each unit; the controls are the only
 * actions that type into the supervisor's window, and each says exactly what.
 */
export function SteerPanel({ data }: { data: GetCommissionResponse }) {
  // Bumped when a control offers wind down instead: Say switches to it and takes focus.
  const [windDown, setWindDown] = useState(0);
  return (
    <div className="flex flex-col gap-space-lg">
      <Say data={data} windDown={windDown} />
      <Controls data={data} onWindDown={() => setWindDown((n) => n + 1)} />
      <Courier id={data.commission?.id ?? ""} />
    </div>
  );
}

/** Entries written before this were read: the API's read-since (STATE.md rewritten after reading OPERATOR.md; rulings P5). */
function readSince(data: GetCommissionResponse): Date | null {
  return toDate(data.commission?.entryReading?.readSince?.value);
}

/** Readings that are not "at its next unit": the operator must know the entry waits (rulings V4). */
const WAITS = new Set([ReadWhen.AFTER_RESUME, ReadWhen.NOT_UNTIL_STEERING, ReadWhen.NEVER, ReadWhen.UNKNOWN, ReadWhen.AT_START]);

function Say({ data, windDown }: { data: GetCommissionResponse; windDown: number }) {
  const { t } = useTranslation();
  const client = useQueryClient();
  const c = data.commission;
  const [kind, setKind] = useState<EntryKind>(EntryKind.DIRECTION);
  const [text, setText] = useState("");
  const [confirming, setConfirming] = useState<"say" | "steering" | null>(null);
  const [warning, setWarning] = useState("");
  const field = useRef<HTMLTextAreaElement>(null);
  useEffect(() => {
    if (windDown === 0) return;
    setKind(EntryKind.WIND_DOWN);
    // Focusing scrolls the field into view on the phone.
    field.current?.focus();
  }, [windDown]);
  const words = useDebounced(text.trim(), 400);
  const preview = useDryRun(["entry", c?.id, kind, words], () => operatorApi.addEntry(c?.id ?? "", kind, words, true), confirming === "say" && Boolean(words));
  if (!c) return null;
  const operatorFile = `${c.folder}/OPERATOR.md`;
  const steeringOn = data.steering?.value === true;
  const steeringKnown = data.steering?.evidence?.known !== false;
  const stateAt = readSince(data);
  const reading = c.entryReading;
  const refresh = () => client.invalidateQueries({ queryKey: ["activity", "commission", c.id] });
  const ended = c.state === CommissionState.ENDED;
  return (
    <Section id="say" title={t(strings.say.title)} note={t(strings.say.note)}>
      {reading?.message ? (
        <p className={WAITS.has(reading.when) ? "rd-reading rd-reading-waits" : "rd-reading"}>
          <span className="font-semibold">{t(strings.entry.readWhen)}</span> {reading.message}
        </p>
      ) : null}
      {!steeringKnown ? (
        <Alert tone="warning" title={t(strings.say.steeringUnknown)} description={data.steering?.evidence?.reason} />
      ) : !steeringOn ? (
        <Alert
          tone="warning"
          title={t(strings.say.steeringOffTitle)}
          description={t(strings.say.steeringOff)}
          actions={<Button variant="secondary" onClick={() => setConfirming("steering")}>{t(strings.say.enableSteering)}</Button>}
        />
      ) : null}
      {ended ? (
        <p className="text-body-sm">{t(strings.say.ended)}</p>
      ) : (
        <form
          className="flex flex-col gap-space-sm"
          onSubmit={(event) => {
            event.preventDefault();
            if (text.trim() && !entryTooLong(text)) setConfirming("say");
          }}
        >
          <fieldset className="rd-choice">
            <legend className="sr-only">{t(strings.say.kindLabel)}</legend>
            <label>
              <input type="radio" name="say-kind" checked={kind === EntryKind.DIRECTION} onChange={() => setKind(EntryKind.DIRECTION)} />
              <span>{t(strings.say.direction)}</span>
            </label>
            <label>
              <input type="radio" name="say-kind" checked={kind === EntryKind.WIND_DOWN} onChange={() => setKind(EntryKind.WIND_DOWN)} />
              <span>{t(strings.say.windDown)}</span>
            </label>
          </fieldset>
          <EntryField ref={field} label={t(kind === EntryKind.WIND_DOWN ? strings.say.windDownLabel : strings.say.directionLabel)} value={text} onChange={setText} />
          <div>
            <Button type="submit" variant="primary" disabled={!text.trim() || entryTooLong(text)}>{t(strings.say.review)}</Button>
          </div>
        </form>
      )}
      {warning ? <Alert tone="warning" title={t(strings.say.writtenWithWarning)} description={warning} /> : null}
      <Entries data={data} stateAt={stateAt} />
      <PathText path={operatorFile} />
      <Confirm
        open={confirming === "say"}
        title={t(kind === EntryKind.WIND_DOWN ? strings.say.confirmWindDownTitle : strings.say.confirmTitle)}
        description={t(strings.say.confirmDescription)}
        confirmLabel={t(strings.say.confirm)}
        onClose={() => setConfirming(null)}
        onConfirm={async () => {
          const res = await operatorApi.addEntry(c.id, kind, text.trim());
          setWarning(res.warning);
          setText("");
          await refresh();
        }}
      >
        <WouldWrite label={t(strings.say.appendsTo, { path: operatorFile })} query={preview} />
      </Confirm>
      <Confirm
        open={confirming === "steering"}
        title={t(strings.say.enableSteeringTitle)}
        description={t(strings.say.enableSteeringDescription)}
        confirmLabel={t(strings.say.enableSteering)}
        onClose={() => setConfirming(null)}
        onConfirm={async () => {
          await operatorApi.enableSteering(c.id);
          await refresh();
        }}
      >
        <Exact label={t(strings.say.appendsTo, { path: `${c.folder}/README.md` })} text={`## Steering\n\n${STEERING_SENTENCE}`} />
      </Confirm>
    </Section>
  );
}

/** OPERATOR.md, newest first, each entry marked read or not yet read. */
function Entries({ data, stateAt }: { data: GetCommissionResponse; stateAt: Date | null }) {
  const { t } = useTranslation();
  const entries = [...data.operatorEntries].reverse();
  if (entries.length === 0 && data.operatorUnparsed === 0) return <p className="text-body-sm text-app-muted-foreground">{t(strings.say.noEntries)}</p>;
  return (
    <div className="flex flex-col gap-space-xs">
      <h3 className="rd-h3">{t(strings.say.entries)}</h3>
      {data.operatorUnparsed > 0 ? <Alert tone="warning" title={t(strings.say.unparsed, { count: data.operatorUnparsed })} /> : null}
      <ul className="rd-rows">
        {entries.map((entry, n) => {
          const at = toDate(entry.at);
          const seen = at && stateAt ? stateAt > at : null;
          return (
            <li key={n} className="rd-row">
              <span className="flex flex-wrap items-center gap-space-xs text-body-sm">
                <span className="font-semibold">{entry.subject ? `${entry.kind} ${entry.subject}` : entry.kind}</span>
                <span className="text-app-muted-foreground">{t(strings.say.via, { via: entry.via })}</span>
                {at ? <When date={at} /> : null}
                <span className={seen ? "rd-seen" : "rd-unseen"}>
                  {seen === null ? <Unknown /> : t(seen ? strings.say.seen : strings.say.notSeen)}
                </span>
              </span>
              <Markdown content={entry.body} className="rd-quote" />
            </li>
          );
        })}
      </ul>
    </div>
  );
}

type ControlKey = "pause" | "resume" | "stop" | "resend" | "discard";

const LABEL = {
  pause: strings.control.labels.pause,
  resume: strings.control.labels.resume,
  stop: strings.control.labels.stop,
  resend: strings.control.labels.resend,
  discard: strings.control.labels.discard,
} as const satisfies Record<ControlKey, string>;

const ACTION: Record<ControlKey, ControlAction> = {
  pause: ControlAction.PAUSE,
  resume: ControlAction.RESUME,
  stop: ControlAction.STOP,
  resend: ControlAction.RESEND_STANCE,
  discard: ControlAction.DISCARD,
};

/** A control the desk declined to type because the terminal was not safe to type into (rulings F5). */
interface Refusal {
  key: ControlKey;
  reason: string;
}

/**
 * Pause, resume, stop, re-send stance and discard (SYSTEM.md §6). Each needs
 * an exact session match with its window open, except discard, and each
 * confirmation names the keystrokes and the session they go to. The desk
 * types only at an idle composer showing the commission's own thread; when it
 * refuses, the reason and the keystrokes to type by hand are shown, with wind
 * down through OPERATOR.md as the path that always works.
 */
function Controls({ data, onWindDown }: { data: GetCommissionResponse; onWindDown: () => void }) {
  const { t } = useTranslation();
  const client = useQueryClient();
  const [open, setOpen] = useState<ControlKey | null>(null);
  const [result, setResult] = useState("");
  const [refusal, setRefusal] = useState<Refusal | null>(null);
  const c = data.commission;
  if (!c) return null;
  const session = data.matches.find((m) => m.kind === MatchKind.EXACT && m.windowOpen);
  const isOpen = windowOpen(c.state);
  const available: Record<ControlKey, boolean> = {
    pause: Boolean(session) && (c.state === CommissionState.LIVE || c.state === CommissionState.LIMITED),
    resume: Boolean(session) && c.state === CommissionState.PAUSED,
    stop: Boolean(session) && isOpen,
    resend: Boolean(session) && isOpen,
    discard: c.state !== CommissionState.ENDED,
  };
  const sessionId = session?.webConsoleSessionId ?? "";
  const copy: Record<ControlKey, { title: string; description: string; typed?: string; destructive?: boolean }> = {
    pause: { title: t(strings.control.pauseTitle), description: t(strings.control.pause, { session: sessionId }), typed: "/goal pause ⏎" },
    resume: { title: t(strings.control.resumeTitle), description: t(strings.control.resume, { session: sessionId }), typed: "/goal resume ⏎" },
    stop: { title: t(strings.control.stopTitle), description: t(strings.control.stop, { session: sessionId }), typed: "/goal pause ⏎\nCtrl+C, Ctrl+C", destructive: true },
    resend: { title: t(strings.control.resendTitle), description: t(strings.control.resend, { session: sessionId, path: `${c.folder}/STANCE-MESSAGE.txt` }) },
    discard: { title: t(strings.control.discardTitle), description: t(session ? strings.control.discardOpen : strings.control.discard), destructive: true },
  };
  const order: ControlKey[] = ["pause", "resume", "resend", "stop", "discard"];
  return (
    <Section id="controls" title={t(strings.control.title)} note={session ? t(strings.control.note) : t(strings.control.noSession)}>
      <div className="rd-actions">
        {order.map((key) => (
          <Button key={key} variant={copy[key].destructive ? "danger" : "secondary"} disabled={!available[key]} onClick={() => setOpen(key)}>
            {t(LABEL[key])}
          </Button>
        ))}
      </div>
      {result ? <Alert tone="info" title={t(strings.control.sent)} description={result} /> : null}
      {refusal ? (
        <Alert
          tone="warning"
          title={t(strings.control.refusedTitle, { action: t(LABEL[refusal.key]) })}
          description={
            <span className="flex flex-col gap-space-xs">
              <span>{refusal.reason}</span>
              {copy[refusal.key].typed ? <Exact label={t(strings.control.byHand, { session: sessionId })} text={copy[refusal.key].typed ?? ""} /> : null}
              <span>{t(strings.control.windDownAlways)}</span>
            </span>
          }
          actions={c.state !== CommissionState.ENDED ? <Button variant="secondary" onClick={onWindDown}>{t(strings.control.windDownInstead)}</Button> : undefined}
        />
      ) : null}
      {open ? (
        <Confirm
          open
          title={copy[open].title}
          description={copy[open].description}
          destructive={copy[open].destructive}
          confirmLabel={t(LABEL[open])}
          reasonLabel={open === "discard" ? t(strings.control.discardReason) : undefined}
          onClose={() => setOpen(null)}
          onConfirm={async (reason) => {
            setResult("");
            setRefusal(null);
            let res;
            try {
              res = await launchApi.control(c.id, ACTION[open], reason);
            } catch (err) {
              // Refused by the terminal's state: not a failure of the desk, so the
              // dialog closes and the section says what to do instead.
              if (err instanceof ConnectError && err.code === Code.FailedPrecondition && open !== "discard") {
                setRefusal({ key: open, reason: err.rawMessage });
                return;
              }
              throw err;
            }
            setResult([res.message, res.goalStatus && t(strings.control.goalStatus, { status: res.goalStatus })].filter(Boolean).join(" · "));
            await client.invalidateQueries({ queryKey: ["activity"] });
            await client.invalidateQueries({ queryKey: ["launch", "record", c.id] });
          }}
        >
          {copy[open].typed ? <Exact label={t(strings.control.typedInto, { session: sessionId })} text={copy[open].typed ?? ""} /> : null}
          <p className="text-body-sm text-app-muted-foreground">{t(open === "resend" ? strings.control.keptSlot : strings.control.recorded)}</p>
        </Confirm>
      ) : null}
    </Section>
  );
}

/** Whether this commission's needs-you items go to Notification Hub (SYSTEM.md §7). */
function Courier({ id }: { id: string }) {
  const { t } = useTranslation();
  const client = useQueryClient();
  const query = useNeedsYou();
  const [next, setNext] = useState<boolean | null>(null);
  const courier = query.data?.courier;
  const own = courier?.commissions.find((s) => s.commissionId === id);
  if (!courier || !own) return null;
  return (
    <Section id="courier" title={t(strings.courier.title)} note={own.reason}>
      <Switch
        label={t(strings.courier.switchLabel)}
        description={t(own.on ? strings.courier.onHint : strings.courier.offHint)}
        checked={own.on}
        onCheckedChange={(value) => setNext(value)}
      />
      <Confirm
        open={next !== null}
        title={t(next ? strings.courier.turnOnTitle : strings.courier.turnOffTitle)}
        description={t(next ? strings.courier.turnOn : strings.courier.turnOff, { id })}
        confirmLabel={t(next ? strings.courier.turnOnLabel : strings.courier.turnOffLabel)}
        onClose={() => setNext(null)}
        onConfirm={async () => {
          await operatorApi.setCourier({ commissionId: id }, Boolean(next));
          await client.invalidateQueries({ queryKey: ["operator", "needs-you"] });
        }}
      >
        <p className="text-body-sm text-app-muted-foreground">{t(strings.courier.storedIn, { path: courier.switchesFile })}</p>
      </Confirm>
    </Section>
  );
}
