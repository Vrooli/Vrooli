import { useQuery } from "@tanstack/react-query";
import { WriteRefusal, type GetWriteAccessResponse } from "@vrooli/proto-types/money-ledger/v1/access/access_pb";

import { fetchWriteAccess } from "../../api/access";
import { strings } from "../../consts/strings";

export const writeAccessQueryKey = ["write-access"] as const;

/**
 * Asks Money Ledger whether this browser may write. The API decides with the
 * same policy that gates every write, and renews an expired browser session
 * while answering, so the page polls gently and re-asks on focus.
 */
export function useWriteAccess() {
  return useQuery({
    queryKey: writeAccessQueryKey,
    queryFn: fetchWriteAccess,
    staleTime: 60_000,
    refetchInterval: 5 * 60_000,
    refetchOnWindowFocus: true,
    retry: false,
  });
}

/**
 * True only once the API has said this browser may write. While the answer is
 * loading, or when the API cannot be reached, write controls stay disabled:
 * offering a form that will be refused is the friction this replaces.
 */
export function useCanWrite(): boolean {
  return useWriteAccess().data?.canWrite === true;
}

/** Whether signing in from this page could change the answer. */
export function signInCanHelp(access: GetWriteAccessResponse): boolean {
  return [WriteRefusal.SIGNED_OUT, WriteRefusal.UNVERIFIED, WriteRefusal.NOT_OWNER].includes(access.refusal);
}

/** The sentence that explains a refusal to the person reading the page. */
export function refusalKey(refusal: WriteRefusal) {
  switch (refusal) {
    case WriteRefusal.SIGNED_OUT:
      return strings.access.refusal.signedOut;
    case WriteRefusal.UNVERIFIED:
      return strings.access.refusal.unverified;
    case WriteRefusal.NOT_OWNER:
      return strings.access.refusal.notOwner;
    case WriteRefusal.AGENT:
      return strings.access.refusal.agent;
    case WriteRefusal.UNDECLARED_SERVICE:
      return strings.access.refusal.undeclaredService;
    case WriteRefusal.CONFLICT:
      return strings.access.refusal.conflict;
    default:
      return strings.access.refusal.unknown;
  }
}
