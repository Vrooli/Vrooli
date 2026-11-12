import { Code, ConnectError } from "@connectrpc/connect";
import type { TFunction } from "i18next";

import { strings } from "../consts/strings";

/**
 * Why a request failed, in the categories the desk designs a state for:
 * the desk API not reachable, this browser not authorized, or the API's own
 * plain-word reason (which is shown verbatim: the API writes it for people).
 */
export type FailureKind = "unreachable" | "unauthorized" | "refused";

/** Connect's own words for an HTTP failure that carried no API answer (a proxy or a dead API). */
const NO_ANSWER = /^HTTP \d{3}$/;

/** True when the error carries words the desk API wrote, not just a transport status. */
function apiAnswered(err: ConnectError): boolean {
  return err.rawMessage !== "" && !NO_ANSWER.test(err.rawMessage);
}

export function failureKind(err: unknown): FailureKind {
  if (err instanceof ConnectError) {
    if (err.code === Code.Unauthenticated || err.code === Code.PermissionDenied) return "unauthorized";
    // The API answers `unavailable` when one of its dependencies is down (Web
    // Console, Notification Hub): that is the API's reason, not an unreachable desk.
    if (err.code === Code.Unavailable) return apiAnswered(err) ? "refused" : "unreachable";
    if (err.code === Code.Unknown || err.code === Code.DeadlineExceeded) return "unreachable";
    return "refused";
  }
  if (err instanceof TypeError) return "unreachable";
  return "refused";
}

export function errorMessage(err: unknown, t: TFunction): string {
  const kind = failureKind(err);
  // A refusal of an operator action names the caller it refused (rulings F13); show those words.
  if (kind === "unauthorized") return err instanceof ConnectError && err.code === Code.PermissionDenied && apiAnswered(err) ? err.rawMessage : t(strings.errors.unauthorized);
  if (kind === "unreachable") return t(strings.errors.unreachable);
  if (err instanceof ConnectError) return err.rawMessage;
  if (err instanceof Error) return err.message;
  return String(err);
}
