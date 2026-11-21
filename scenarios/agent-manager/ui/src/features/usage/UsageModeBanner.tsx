import { useState } from "react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import type { ListUsageModesResponse, UsageModeStatus, UsageWindowForecast } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { formatPoints } from "../../lib/format";
import { formatAt } from "./format";
import { startErrorText } from "./startErrorText";
import { RunSwitchPanel } from "./RunSwitchPanel";
import { useUsageModeActions } from "./api";
import { extendChoices, modeEnds, scopeText, secondsLeftText, starterText } from "./modes";

function ModeBanner({ status, windows }: { status: UsageModeStatus; windows: readonly UsageWindowForecast[] }) {
  const mode = status.mode;
  const [extending, setExtending] = useState(false);
  const { end, extend: extendMode } = useUsageModeActions();
  const [switching, setSwitching] = useState(false);
  if (!mode) return null;
  const choices = extendChoices(mode, windows);
  const burn = status.burn.map((b) => `${formatPoints(b.observedIncreasePoints)} on ${b.provider === "anthropic" ? "Claude" : b.provider === "openai" ? "Codex" : b.provider}`).join(", ");
  const error = extendMode.error ?? end.error;
  const extend = (to: string) => extendMode.mutate({ id: mode.id, to }, { onSuccess: () => setExtending(false) });
  return (
    <section aria-label="Active usage mode" data-testid="usage-mode-banner" className="space-y-3 rounded-lg border border-primary/50 bg-card p-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2"><StatusBadge tone="info">Mode on</StatusBadge><h3 className="min-w-0 break-words text-base font-semibold">{mode.label}</h3></div>
          <p className="text-sm text-muted-foreground">New work for {scopeText(mode)} uses a cheaper model. Started {formatAt(mode.startedAt)} by {starterText(mode.startedBy)}.</p>
        </div>
        <div className="flex flex-wrap gap-2">
          {choices.length > 0 ? <Button variant="secondary" aria-expanded={extending} onClick={() => setExtending((v) => !v)} data-testid="mode-extend">Extend</Button> : null}
          <Button variant="secondary" aria-expanded={switching} onClick={() => setSwitching((v) => !v)} data-testid="mode-switch-runs">Switch running work</Button>
          <Button variant="warning" disabled={end.isPending} onClick={() => end.mutate([mode.id])} data-testid="mode-end">End now</Button>
        </div>
      </div>
      <dl className="grid grid-cols-1 gap-x-6 gap-y-2 text-sm sm:grid-cols-3">
        <div><dt className="text-xs text-muted-foreground">Until</dt><dd className="font-semibold" data-testid="mode-until">{modeEnds(mode)}</dd><dd className="text-xs text-muted-foreground">{secondsLeftText(status)}</dd></div>
        <div className="sm:col-span-2"><dt className="text-xs text-muted-foreground">Quota used since start (observed, not a saving)</dt><dd className="font-semibold" data-testid="mode-burn">{burn || "No change in a covered window yet"}</dd></div>
      </dl>
      {extending ? (
        <div className="flex flex-wrap gap-2" data-testid="mode-extend-choices">
          {choices.map((choice) => <Button key={choice.id} variant="secondary" disabled={extendMode.isPending} onClick={() => extend(choice.to)} data-testid={`mode-extend-${choice.id}`}>{choice.label}</Button>)}
          <p className="w-full text-xs text-muted-foreground">Extending moves this mode's end. Its settings and its burn since start carry on unchanged.</p>
        </div>
      ) : null}
      {switching ? <RunSwitchPanel modeId={mode.id} /> : null}
      {error ? <p role="alert" className="text-sm text-destructive" data-testid="mode-action-error">{startErrorText(error)}</p> : null}
    </section>
  );
}

/** One banner per active mode, above the Usage overview. */
export function UsageModeBanners({ modes, windows }: { modes: ListUsageModesResponse | undefined; windows: readonly UsageWindowForecast[] }) {
  if (!modes || modes.modes.length === 0) return null;
  return <div className="flex flex-col gap-3">{modes.modes.map((status) => <ModeBanner key={status.mode?.id} status={status} windows={windows} />)}</div>;
}
