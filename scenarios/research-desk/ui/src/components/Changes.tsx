import { Button } from "@vrooli/react-component-library/Button/2";
import { Input } from "@vrooli/react-component-library/Input/1";
import { useQueryClient } from "@tanstack/react-query";
import { useId, useState } from "react";

import { activityKeys, Baseline, ChangeSubject, useChanges, type GetChangesResponse } from "../api/activity";
import { operatorApi } from "../api/operator";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { kb, toDate } from "../lib/format";
import { Markdown } from "./Markdown";
import { PathText } from "./PathText";
import { ActionError, QueryView } from "./QueryView";
import { Unreadable } from "./Unreadable";
import { When } from "./When";

function changeCount(c: GetChangesResponse): number {
  return c.ledger.length + c.checkpoints.length + c.statusChanges.length + c.files.length;
}

/** The words for each baseline the API reports; an unspecified one reads as a look. */
const LOOK_LINE = { some: strings.changes.countSince, none: strings.changes.noneSince } as const;
type Line = { some: (typeof strings.changes)[keyof typeof strings.changes]; none: (typeof strings.changes)[keyof typeof strings.changes] };
const LINE: Partial<Record<Baseline, Line>> = {
  [Baseline.LOOK]: LOOK_LINE,
  [Baseline.ASKED]: { some: strings.changes.countFrom, none: strings.changes.noneFrom },
  [Baseline.LAST_DAY]: { some: strings.changes.countDay, none: strings.changes.noneDay },
};

/**
 * One line for a list row: "4 changes since you looked 3 hours ago", or,
 * with no look recorded, the last 24 hours, saying so. With
 * `quiet`, a subject with nothing new renders nothing.
 */
export function ChangesLine({ subject, id, since, quiet }: { subject: ChangeSubject; id: string; since?: Date; quiet?: boolean }) {
  const { t } = useTranslation();
  const query = useChanges(subject, id, since);
  if (!query.data) return null;
  const n = changeCount(query.data);
  if (quiet && n === 0) return null;
  const from = toDate(query.data.since);
  const words = LINE[query.data.baselineKind] ?? LOOK_LINE;
  return (
    <p className="text-body-sm">
      {n === 0 ? t(words.none) : t(words.some, { count: n })} {from && query.data.baselineKind !== Baseline.LAST_DAY ? <When date={from} /> : null}
    </p>
  );
}

/** A date as a datetime-local value in the phone's own time zone. */
function localInput(date: Date): string {
  const shifted = new Date(date.getTime() - date.getTimezoneOffset() * 60_000);
  return shifted.toISOString().slice(0, 16);
}

/**
 * What changed since the operator last looked, or since a time they pick
 * (SYSTEM.md §8): new ledger text, changed checkpoints,
 * workstream status changes and touched files, and in the API's words what it
 * cannot show (status changes need a recorded look). Reading never records a look;
 * "Mark as seen" does, in the desk's own state folder.
 */
export function ChangesPanel({ subject, id }: { subject: ChangeSubject; id: string }) {
  const { t } = useTranslation();
  const [mode, setMode] = useState<"look" | "time">("look");
  const [at, setAt] = useState(() => localInput(new Date(Date.now() - 24 * 3_600_000)));
  const picked = new Date(at);
  const since = mode === "time" && !Number.isNaN(picked.getTime()) ? picked : undefined;
  const query = useChanges(subject, id, since);
  const client = useQueryClient();
  const name = useId();
  const atId = useId();
  const [error, setError] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);
  const mark = async () => {
    setBusy(true);
    setError(null);
    try {
      await operatorApi.markSeen(subject, id);
      await client.invalidateQueries({ queryKey: activityKeys.changes(subject, id) });
    } catch (err) {
      setError(err);
    } finally {
      setBusy(false);
    }
  };
  return (
    <div className="flex flex-col gap-space-sm">
      <fieldset className="rd-choice">
        <legend className="sr-only">{t(strings.changes.compareWith)}</legend>
        <label>
          <input type="radio" name={name} checked={mode === "look"} onChange={() => setMode("look")} />
          <span>{t(strings.changes.sinceLook)}</span>
        </label>
        <label>
          <input type="radio" name={name} checked={mode === "time"} onChange={() => setMode("time")} />
          <span>{t(strings.changes.sinceTime)}</span>
        </label>
      </fieldset>
      {mode === "time" ? (
        <label htmlFor={atId} className="flex max-w-xs flex-col gap-space-2xs text-body-sm">
          <span className="font-medium">{t(strings.changes.from)}</span>
          <Input id={atId} type="datetime-local" value={at} onChange={(e) => setAt(e.target.value)} />
        </label>
      ) : null}
      <QueryView query={query}>
        {(c) => {
          const from = toDate(c.since);
          return (
            <div className="flex flex-col gap-space-sm">
              <div className="flex flex-wrap items-center justify-between gap-space-xs">
                <p className="text-body-sm text-app-muted-foreground">
                  {c.baseline ? (
                    c.baseline
                  ) : from ? (
                    <>
                      {t(strings.changes.lastLooked)} <When date={from} />
                    </>
                  ) : null}
                </p>
                <Button variant="secondary" pending={busy} onClick={() => void mark()}>
                  {t(strings.changes.markSeen)}
                </Button>
              </div>
              <ActionError error={error} />
              <Unreadable evidence={c.unreadable} />
              {c.notShown.length > 0 ? (
                <ul className="rd-list text-body-sm text-app-muted-foreground">{c.notShown.map((line) => <li key={line}>{line}</li>)}</ul>
              ) : null}
              {from && changeCount(c) === 0 && c.unreadable.length === 0 ? <p className="text-body-sm">{t(strings.changes.nothing)}</p> : null}
              {c.statusChanges.length > 0 ? (
                <div>
                  <h3 className="rd-h3">{t(strings.changes.statuses)}</h3>
                  <ul className="rd-list">
                    {c.statusChanges.map((s) => (
                      <li key={`${s.topic}/${s.workstream}`}>
                        {t(strings.changes.statusLine, { topic: s.topic, workstream: s.workstream, before: s.before || t(strings.common.none), after: s.after || t(strings.common.none) })}
                      </li>
                    ))}
                  </ul>
                </div>
              ) : null}
              {c.showcaseLines.length > 0 ? (
                <div>
                  <h3 className="rd-h3">{t(strings.changes.showcase)}</h3>
                  <ul className="rd-list">{c.showcaseLines.map((line, n) => <li key={n}>{line}</li>)}</ul>
                </div>
              ) : null}
              {c.ledger.map((entry) => (
                <div key={entry.path}>
                  <h3 className="rd-h3">{t(strings.changes.ledger)}</h3>
                  <PathText path={entry.path} />
                  {entry.fromDatedHeadings ? <p className="text-caption text-app-muted-foreground">{t(strings.changes.fromDatedHeadings)}</p> : null}
                  <Markdown content={entry.text} className="rd-quote" />
                  {entry.truncated ? <p className="text-caption text-app-muted-foreground">{t(strings.changes.truncatedText)}</p> : null}
                </div>
              ))}
              {c.checkpoints.length > 0 ? (
                <div>
                  <h3 className="rd-h3">{t(strings.changes.checkpoints)}</h3>
                  <FileList files={c.checkpoints} />
                </div>
              ) : null}
              {c.files.length > 0 ? (
                <div>
                  <h3 className="rd-h3">{t(strings.changes.files, { count: c.files.length })}</h3>
                  <FileList files={c.files} />
                  {c.truncated ? <p className="text-caption text-app-muted-foreground">{t(strings.changes.truncatedWalk)}</p> : null}
                </div>
              ) : null}
            </div>
          );
        }}
      </QueryView>
    </div>
  );
}

export function FileList({ files }: { files: { path: string; bytes: bigint; modified?: Parameters<typeof toDate>[0] }[] }) {
  return (
    <ul className="rd-list">
      {files.map((f) => {
        const modified = toDate(f.modified);
        return (
          <li key={f.path} className="flex flex-col">
            <PathText path={f.path} />
            <span className="text-caption text-app-muted-foreground">
              {kb(f.bytes)}
              {modified ? (
                <>
                  {" · "}
                  <When date={modified} />
                </>
              ) : null}
            </span>
          </li>
        );
      })}
    </ul>
  );
}
