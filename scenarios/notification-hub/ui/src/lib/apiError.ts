import { Code, ConnectError } from "@connectrpc/connect";

/** True when the API refused a read because there is no signed-in Hub session. */
export function isUnauthenticated(error: unknown): boolean {
  return error instanceof ConnectError && error.code === Code.Unauthenticated;
}
