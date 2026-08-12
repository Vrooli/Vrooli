const defaultDelay = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
/**
 * Send a multi-step key combo sequence to the terminal via the input
 * gate. Steps with `delayMs > 0` pause before sending. Results are
 * discarded — combo sequences are fire-and-forget; the gate logs
 * queued/rejected outcomes via the per-session pending-input pill.
 */
export async function sendComboSequence(sequence, onInput, delay = defaultDelay) {
    for (const step of sequence) {
        if (step.delayMs && step.delayMs > 0) {
            await delay(step.delayMs);
        }
        onInput(step.data, "toolbar-key");
    }
}
