import { Code, ConnectError } from "@connectrpc/connect";

/** Connect metadata key with which the Hub says why it refused a request that was valid when it was sent. */
const ERROR_REASON_HEADER = "Hub-Error-Reason";

/** The requester revised the ask while the operator was answering it. */
export const REASON_ASK_CHANGED = "ask_changed";
/** The ask's requester did not declare that it answers questions (or the ask is no longer open). */
export const REASON_CONTEXT_NOT_ACCEPTED = "context_not_accepted";

/** Whether `error` is the Hub refusing with a FailedPrecondition for exactly this reason. */
export const hasHubReason = (error: unknown, reason: string): boolean =>
  error instanceof ConnectError && error.code === Code.FailedPrecondition && error.metadata.get(ERROR_REASON_HEADER) === reason;
