import { useState } from "react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Checkbox } from "@vrooli/react-component-library/Checkbox/1";
import { RadioGroup } from "@vrooli/react-component-library/RadioGroup/1";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";
import { Select } from "@vrooli/react-component-library/Select/1";
import type { UsageModeReachRow, UsageWindowForecast } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { formatShare } from "./format";
import { DURATION_LABELS, durationBlock, durationDetail, effectText, RUNNER_NAMES, runnerForProvider, TIME_HOURS, untilValue, type EffectRange, type ModeDuration, type ModeRunner } from "./modes";
import { useUsageModeActions } from "./api";
import { RunSwitchPanel } from "./RunSwitchPanel";
import { startErrorText } from "./startErrorText";

const DURATIONS: ModeDuration[] = ["five_hour", "weekly", "time", "off"];

export interface UsageModeSheetProps {
  open: boolean;
  onClose: () => void;
  /** The coding agent the sheet opens on; empty means everything. */
  runner: ModeRunner | "";
  /** The window the effect figure is stated for, e.g. "Codex · weekly". */
  windowName: string;
  /** Length of the window the sheet was opened for; its reset is the default end. */
  windowMinutes?: number;
  effect?: EffectRange;
  windows: readonly UsageWindowForecast[];
  reach: readonly UsageModeReachRow[];
  reachNotes: readonly string[];
}

function ReachCard({ runner, reach, notes }: { runner: ModeRunner | ""; reach: readonly UsageModeReachRow[]; notes: readonly string[] }) {
  const rows = reach.filter((row) => row.observedIncreasePoints > 0 && (!runner || runnerForProvider(row.provider) === runner));
  const row = rows.length === 1 ? rows[0] : undefined;
  return (
    <section className="space-y-1 rounded-md border border-border p-3" data-testid="mode-reach" aria-label="Reach">
      <h4 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">Reach</h4>
      <p className="text-sm">This changes <b>new</b> work only: terminal and Web Console sessions started through Vrooli, and Agent Manager runs with a role.</p>
      {row ? (
        <p className="text-xs text-muted-foreground" data-testid="mode-reach-shares">
          Of the use in this window so far (estimated): {formatShare(row.operatorShare)} from operator sessions, {formatShare(row.profileShare)} from profile runs, {formatShare(row.unattributedShare)} the mode may not reach (desktop apps and tools started outside Vrooli; still counted).
        </p>
      ) : null}
      <details className="text-xs text-muted-foreground">
        <summary className="inline-flex min-h-11 cursor-pointer items-center sm:min-h-8">What it can and cannot reach</summary>
        <ul className="m-0 list-disc space-y-1 pl-4">{notes.map((note) => <li key={note}>{note}</li>)}</ul>
      </details>
    </section>
  );
}

/** Start a usage mode: the preset is the decision; duration is answered by default; scope is folded under Advanced. */
export function UsageModeSheet({ open, onClose, runner: initialRunner, windowName, windowMinutes, effect, windows, reach, reachNotes }: UsageModeSheetProps) {
  const [runner, setRunner] = useState<ModeRunner | "">(initialRunner);
  const [duration, setDuration] = useState<ModeDuration>(() => {
    const own: ModeDuration | undefined = windowMinutes === 300 ? "five_hour" : windowMinutes === 10080 ? "weekly" : undefined;
    return (own && !durationBlock(own, initialRunner, windows) ? own : DURATIONS.find((value) => !durationBlock(value, initialRunner, windows))) ?? "time";
  });
  const [hours, setHours] = useState<number>(4);
  const [role, setRole] = useState("");
  const [profile, setProfile] = useState("");
  const [switchRunning, setSwitchRunning] = useState(false);
  const [startedId, setStartedId] = useState("");
  const { start } = useUsageModeActions();
  const chosen = duration;
  const blocked = durationBlock(chosen, runner, windows);
  const options = DURATIONS.map((value) => {
    const block = durationBlock(value, runner, windows);
    return { value, label: DURATION_LABELS[value], description: durationDetail(value, runner, windows), disabled: !!block, testId: `mode-duration-${value}` };
  });
  const startLabel = `Start ${chosen === "time" ? `for ${hours} h` : DURATION_LABELS[chosen].toLowerCase()}`;
  // Switching running work is a second, explicit step with its own preview, so a started mode
  // stays open here until the operator has seen what would switch and what it costs.
  const submit = () => start.mutate({ preset: "one-tier-down", runner, role: role.trim(), profile: profile.trim(), until: untilValue(chosen, hours) }, {
    onSuccess: (response) => {
      const id = response.mode?.id ?? "";
      if (switchRunning && id) setStartedId(id);
      else onClose();
    },
  });
  return (
    <ResponsiveDialog
      open={open}
      onOpenChange={(next) => { if (!next) onClose(); }}
      title="Save usage for a while"
      closeLabel="Close"
      size="lg"
      testId="usage-mode-sheet"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} data-testid={startedId ? "mode-done" : undefined}>{startedId ? "Done" : "Cancel"}</Button>
          {startedId ? null : <Button variant="primary" disabled={!!blocked || start.isPending} onClick={submit} data-testid="mode-start">{start.isPending ? "Starting" : startLabel}</Button>}
        </>
      }
    >
      {startedId ? (
        <div className="space-y-3 p-4">
          <p className="text-sm" data-testid="mode-started">The mode is on for new work. Choose whether the work already running should switch too.</p>
          <RunSwitchPanel modeId={startedId} />
        </div>
      ) : (
      <div className="grid gap-4 p-4 md:grid-cols-2">
        <div className="space-y-4">
          <RadioGroup
            label="What to do"
            variant="card"
            value="one-tier-down"
            options={[{ value: "one-tier-down", label: "Save usage: one tier down", description: "New work uses the next cheaper model each coding agent declares.", testId: "mode-preset" }]}
          />
          <RadioGroup label="For how long" variant="card" value={chosen} onValueChange={(value) => setDuration(value as ModeDuration)} options={options} />
          {chosen === "time" ? (
            <label className="flex items-center gap-2 text-sm">
              Hours
              <Select aria-label="Hours" value={String(hours)} options={TIME_HOURS.map((h) => ({ value: String(h), label: `${h} h` }))} onValueChange={(value) => setHours(Number(value))} />
            </label>
          ) : null}
          <details className="rounded-md border border-border p-3" data-testid="mode-advanced">
            <summary className="inline-flex min-h-11 cursor-pointer items-center text-sm font-medium sm:min-h-8">Advanced: runner, role, profile (applies to {runner ? RUNNER_NAMES[runner] : "everything"})</summary>
            <div className="mt-2 grid gap-2 text-sm">
              <label className="grid gap-1">Runner
                <Select
                  aria-label="Runner"
                  value={runner}
                  options={[{ value: "", label: "Everything" }, { value: "claude", label: "Claude" }, { value: "codex", label: "Codex" }]}
                  onValueChange={(value) => { setRunner(value as ModeRunner | ""); if (!value && (chosen === "five_hour" || chosen === "weekly")) setDuration("time"); }}
                />
              </label>
              <label className="grid gap-1">Role (optional)
                <input className="min-h-11 rounded-md border border-border bg-background px-2 sm:min-h-8" value={role} onChange={(event) => setRole(event.target.value)} />
              </label>
              <label className="grid gap-1">Profile (optional)
                <input className="min-h-11 rounded-md border border-border bg-background px-2 sm:min-h-8" value={profile} onChange={(event) => setProfile(event.target.value)} />
              </label>
            </div>
          </details>
        </div>
        <div className="space-y-4">
          <section className="space-y-1 rounded-md border border-border p-3" data-testid="mode-effect" aria-label="Expected effect">
            <h4 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">Expected effect · not measured yet</h4>
            <p className="text-sm">{effectText(effect, windowName)}</p>
            <p className="text-xs text-muted-foreground">A cheaper model can need more attempts, so real savings may be smaller. Measured results appear after a few uses.</p>
          </section>
          <ReachCard runner={runner} reach={reach} notes={reachNotes} />
          <section className="space-y-2 rounded-md border border-border p-3" aria-label="Work already running">
            <h4 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">Work already running</h4>
            <p className="text-sm">Running sessions are left as they are unless you ask.</p>
            <Checkbox label="Also switch running sessions" description="Next you see each run and what it would re-read. Managed runs switch at their next resume; interactive sessions only change on their next launch." checked={switchRunning} data-testid="mode-switch-running" onCheckedChange={(next) => setSwitchRunning(next === true)} />
          </section>
        </div>
        {start.error ? <p role="alert" className="text-sm text-destructive md:col-span-2" data-testid="mode-start-error">{startErrorText(start.error)}</p> : null}
      </div>
      )}
    </ResponsiveDialog>
  );
}
