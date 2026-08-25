import type { KeyComboStep } from "../consts/key-combos";
import type { GateResult, InputIntent } from "../components/terminal/inputGate";

/** Injectable delay function — default uses setTimeout; tests can substitute a synchronous fake. */
export type DelayFn = (ms: number) => Promise<void>;

const defaultDelay: DelayFn = (ms) =>
  new Promise((resolve) => setTimeout(resolve, ms));

/**
 * Send a multi-step key combo sequence to the terminal via the input
 * gate. Steps with `delayMs > 0` pause before sending. Results are
 * discarded — combo sequences are fire-and-forget; the gate logs
 * queued/rejected outcomes via the per-session pending-input pill.
 */
export async function sendComboSequence(
  sequence: KeyComboStep[],
  onInput: (data: string, intent: Exclude<InputIntent, "control">) => GateResult,
  delay: DelayFn = defaultDelay,
): Promise<void> {
  for (const step of sequence) {
    if (step.delayMs && step.delayMs > 0) {
      await delay(step.delayMs);
    }
    onInput(step.data, "named_key");
  }
}
