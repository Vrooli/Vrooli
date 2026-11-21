import { breakEvenSentence, formatRatio, ratioRangeText, type BreakEven } from "./evidence";

/** What the cost range says: the price and the use were read on one set of units, so it decides. */
export function breakEvenOutcome(b: BreakEven): "cheaper" | "unclear" | "dearer" {
  if (b.verdict === "cheaper") return "cheaper";
  if (b.verdict === "more_expensive") return "dearer";
  return "unclear";
}

const OUTCOME_TEXT = {
  cheaper: "The cost range sits above 1x, so it cost less per accepted outcome on this work.",
  unclear: "The cost range does not settle whether it cost less per accepted outcome.",
  dearer: "The cost range sits below 1x, so it cost more per accepted outcome on this work.",
} as const;

const FILL = { cheaper: "bg-success", unclear: "bg-warning", dearer: "bg-destructive" } as const;

/**
 * Realized price advantage against extra reference tokens, both on the same known
 * units with the same fixed weights. The track runs from 0x to a little past the
 * larger of the two; the marker is break-even, the fill is what it used, and the
 * bracket is the range of that measurement.
 */
export function BreakEvenBar({ value }: { value: BreakEven }) {
  const outcome = breakEvenOutcome(value);
  const top = Math.max(value.advantage, value.used.high) * 1.1;
  const at = (ratio: number) => `${Math.min(100, Math.max(0, (ratio / top) * 100))}%`;
  return (
    <div className="flex flex-col gap-2" data-testid="break-even">
      <p className="m-0 text-sm" data-testid="break-even-sentence">{breakEvenSentence(value)}</p>
      <div role="img" aria-label={`Used ${formatRatio(value.used.value)} the reference tokens per accepted outcome, range ${ratioRangeText(value.used)}; break-even on these units is ${formatRatio(value.advantage)}. ${OUTCOME_TEXT[outcome]}`} className="relative h-6 w-full">
        <div className="absolute inset-x-0 top-2 h-2.5 rounded bg-muted" />
        <div className={`absolute left-0 top-2 h-2.5 rounded-l ${FILL[outcome]}`} style={{ width: at(value.used.value) }} />
        <div className="absolute top-0.5 h-5 border-x-2 border-foreground/70" style={{ left: at(value.used.low), width: `max(2px, calc(${at(value.used.high)} - ${at(value.used.low)}))` }} data-testid="break-even-range" />
        <div className="absolute -top-0.5 h-7 w-0.5 bg-foreground" style={{ left: at(value.advantage) }} data-testid="break-even-marker" />
      </div>
      <div className="flex justify-between gap-2 text-xs text-muted-foreground">
        <span>Used {formatRatio(value.used.value)} ({ratioRangeText(value.used)})</span>
        <span>Break-even on these units {formatRatio(value.advantage)}</span>
      </div>
      <p className="m-0 text-xs text-muted-foreground">{OUTCOME_TEXT[outcome]}{value.prior ? " The points behind it rest on a tier prior for at least one model, so treat the size as a rough one." : ""}</p>
    </div>
  );
}
