/**
 * Which single line the Messages pane's status region should show.
 *
 * Lives outside the component file so the pane keeps fast refresh, and so the
 * priority rule can be tested as the pure decision it is.
 *
 * The pane previously rendered its live-stream notice and its refresh result as
 * two independent banners. They stacked and contradicted each other on screen:
 * a full-width "Live updates disconnected — reconnecting" card directly above
 * an equally heavy "Up to date" card, both pushing the transcript down.
 */

export type MessagesPaneStatus =
  | { kind: "error"; text: string }
  | { kind: "disconnected"; text: string }
  | { kind: "success"; text: string };

/**
 * Order is the contract: a failed refresh outranks a dropped stream, and both
 * outrank the transient confirmation that a refresh completed. Anything else
 * lets the pane reassure the user while it is visibly broken.
 */
export function resolveMessagesPaneStatus(input: {
  refreshError: string | null;
  liveInterrupted: boolean;
  liveInterruptedText: string;
  transient: string | null;
}): MessagesPaneStatus | null {
  if (input.refreshError) return { kind: "error", text: input.refreshError };
  if (input.liveInterrupted) return { kind: "disconnected", text: input.liveInterruptedText };
  if (input.transient) return { kind: "success", text: input.transient };
  return null;
}
