import { Code, ConnectError } from "@connectrpc/connect";
import type { ConversationLoadError } from "../stores/useConversationStore";

/**
 * Conversation loading vocabulary, kept in a dependency-light module.
 *
 * `useConversationSession` is mocked by many component tests, so anything the
 * store or the hydration hook needs at runtime cannot live there — a mock that
 * omits one export would break unrelated suites. This module imports only a
 * type and the Connect error class, so it is safe to depend on from anywhere.
 */

/** The page size every conversation fetch windows to. */
export const PAGE_SIZE = 500;

/**
 * The outcome of a refresh, as something the caller can act on.
 *
 * Every one of these paths used to return a bare boolean that callers then
 * discarded, so a dropped connection, a session the server no longer knows
 * about, and "you are already up to date" were indistinguishable — from the
 * code and from the user, who saw a spinner stop and nothing else happen.
 */
export type RefreshOutcome =
  | { ok: true; addedEvents: number }
  | { ok: false; error: ConversationLoadError };

/**
 * describeLoadFailure converts a thrown value into something worth showing.
 *
 * Connect codes carry the distinction that matters most: `not_found` means the
 * session is gone and retrying will never help, while a transport failure is
 * exactly the case where a retry button belongs. Collapsing both into a silent
 * empty state is what made a broken pane look like an idle one.
 */
export function describeLoadFailure(error: unknown): ConversationLoadError {
  if (error instanceof ConnectError) {
    switch (error.code) {
      case Code.NotFound:
        return { message: "This session no longer exists.", code: "not_found", retryable: false };
      case Code.PermissionDenied:
      case Code.Unauthenticated:
        return { message: "You don't have access to this session's messages.", code: "permission_denied", retryable: false };
      case Code.InvalidArgument:
        return { message: "This session's messages couldn't be requested.", code: "invalid_argument", retryable: false };
      case Code.Unavailable:
      case Code.DeadlineExceeded:
        return { message: "Web Console couldn't reach the server.", code: "unavailable", retryable: true };
      default:
        return {
          message: error.rawMessage || "Messages couldn't be loaded.",
          code: Code[error.code],
          retryable: true,
        };
    }
  }
  const message = error instanceof Error && error.message ? error.message : "Messages couldn't be loaded.";
  return { message, code: "network", retryable: true };
}
