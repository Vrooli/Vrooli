import { Button } from "@vrooli/react-component-library/Button/2";
import type { UsageModeRebaseResult } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { useRunSwitch, useRunSwitchPreview } from "./api";
import { startErrorText } from "./startErrorText";

const STATUS_TEXT: Record<string, string> = {
  switched_at_next_resume: "Switches at its next resume",
  applies_on_next_launch: "Applies on next launch",
  unable: "Cannot switch",
};

/** About how much a switch makes the run re-read, in plain words. */
export function rereadText(tokens: bigint): string {
  if (tokens <= 0n) return "re-read size not recorded";
  const n = Number(tokens);
  return `re-reads about ${n >= 1000 ? `${Math.round(n / 1000)}k` : String(n)} tokens`;
}

function ResultRow({ result }: { result: UsageModeRebaseResult }) {
  const moves = result.toModel ? `${result.fromModel} to ${result.toModel}` : result.fromModel;
  return (
    <li className="space-y-0.5 border-t border-border py-2 first:border-t-0" data-testid="mode-switch-result" data-status={result.status}>
      <p className="break-words text-sm"><b>{STATUS_TEXT[result.status] ?? result.status}</b> <span className="text-muted-foreground">· run {result.runId.slice(0, 8)}{moves ? ` · ${moves}` : ""}</span></p>
      <p className="break-words text-xs text-muted-foreground">{result.reason}{result.status === "switched_at_next_resume" ? ` (${rereadText(result.estimatedRereadTokens)})` : ""}</p>
    </li>
  );
}

/**
 * The explicit "also switch running work" step. It first shows, per run, what
 * would happen and what the switch would cost in re-reading; nothing changes
 * until the operator confirms. Only managed runs can switch, and only at their
 * next resume. Interactive sessions never switch in place.
 */
export function RunSwitchPanel({ modeId }: { modeId: string }) {
  const preview = useRunSwitchPreview(modeId);
  const apply = useRunSwitch();
  const results = apply.data?.results ?? preview.data?.results ?? [];
  const switchable = results.filter((result) => result.status === "switched_at_next_resume");
  const total = switchable.reduce((sum, result) => sum + result.estimatedRereadTokens, 0n);
  const done = apply.isSuccess;
  return (
    <section className="space-y-2 rounded-md border border-border p-3" data-testid="mode-switch-panel" aria-label="Switch running work">
      <h4 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">{done ? "Result" : "Running work"}</h4>
      {preview.isPending ? <p className="text-sm text-muted-foreground">Checking each run</p> : null}
      {preview.error ? <p role="alert" className="text-sm text-destructive" data-testid="mode-switch-error">{startErrorText(preview.error)}</p> : null}
      {!preview.isPending && !preview.error && results.length === 0 ? <p className="text-sm" data-testid="mode-switch-none">No runs are active, so there is nothing to switch.</p> : null}
      {results.length > 0 ? <ul className="m-0 list-none p-0">{results.map((result) => <ResultRow key={result.runId} result={result} />)}</ul> : null}
      {!done && switchable.length > 0 ? (
        <>
          <p className="text-xs text-muted-foreground" data-testid="mode-switch-cost">
            {switchable.length} managed {switchable.length === 1 ? "run" : "runs"} can switch at the next resume. A changed model cannot reuse the cache, so together they {rereadText(total)}. Interactive sessions are not changed.
          </p>
          <Button variant="primary" disabled={apply.isPending} data-testid="mode-switch-confirm"
            onClick={() => apply.mutate({ modeId, runIds: switchable.map((result) => result.runId) })}>
            {apply.isPending ? "Switching" : `Switch ${switchable.length} ${switchable.length === 1 ? "run" : "runs"} at next resume`}
          </Button>
        </>
      ) : null}
      {apply.error ? <p role="alert" className="text-sm text-destructive" data-testid="mode-switch-error">{startErrorText(apply.error)}</p> : null}
    </section>
  );
}
