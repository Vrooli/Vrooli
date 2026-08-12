/**
 * Conversation display density.
 *
 * "comfortable" is the chat reading of a transcript: speaker-aligned bubbles,
 * good for a short back-and-forth. "compact" is the log reading: full-width
 * rows separated by rules, which fits far more turns on screen and keeps long
 * agent output from being squeezed into a narrow column. Long working sessions
 * want the second one, so the choice is remembered per browser rather than
 * reset on every navigation.
 *
 * Deliberately localStorage and not a server preference: this is a per-device
 * viewing choice, like a window size, not part of the session record.
 */
export type ChatDensity = "comfortable" | "compact";

export const CHAT_DENSITIES: readonly ChatDensity[] = ["comfortable", "compact"] as const;

export const CHAT_DENSITY_LABELS: Record<ChatDensity, string> = {
  comfortable: "Bubbles",
  compact: "Compact",
};

/**
 * What each density does, for the icon-button tooltip and accessible name.
 * The toggle is rendered as icons — a two-word label is not worth the width in
 * a toolbar that also carries the profile and run links — so the label alone
 * no longer explains the choice.
 */
export const CHAT_DENSITY_DESCRIPTIONS: Record<ChatDensity, string> = {
  comfortable: "Bubbles — speaker-aligned, easier for short exchanges",
  compact: "Compact — full-width rows, fits more turns on screen",
};

const STORAGE_KEY = "swarm-manager:chat-density:v1";
const DEFAULT_DENSITY: ChatDensity = "comfortable";

function isChatDensity(value: string | null): value is ChatDensity {
  return value === "comfortable" || value === "compact";
}

export function readChatDensity(): ChatDensity {
  if (typeof window === "undefined") return DEFAULT_DENSITY;
  try {
    const stored = window.localStorage.getItem(STORAGE_KEY);
    return isChatDensity(stored) ? stored : DEFAULT_DENSITY;
  } catch {
    // Private-mode or blocked storage must not break the conversation.
    return DEFAULT_DENSITY;
  }
}

export function rememberChatDensity(density: ChatDensity): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(STORAGE_KEY, density);
  } catch {
    // Non-fatal: the operator re-picks next visit.
  }
}
