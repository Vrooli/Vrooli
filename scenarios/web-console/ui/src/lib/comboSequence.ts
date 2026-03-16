import type { KeyComboStep } from "../consts/key-combos";

/** Injectable delay function — default uses setTimeout; tests can substitute a synchronous fake. */
export type DelayFn = (ms: number) => Promise<void>;

const defaultDelay: DelayFn = (ms) =>
  new Promise((resolve) => setTimeout(resolve, ms));

/**
 * Send a multi-step key combo sequence to the terminal.
 * Steps with `delayMs > 0` pause before sending.
 */
export async function sendComboSequence(
  sequence: KeyComboStep[],
  onInput: (data: string) => boolean,
  delay: DelayFn = defaultDelay,
): Promise<void> {
  for (const step of sequence) {
    if (step.delayMs && step.delayMs > 0) {
      await delay(step.delayMs);
    }
    onInput(step.data);
  }
}
