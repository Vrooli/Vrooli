import { createClient } from "@connectrpc/connect";
import { create } from "@bufbuild/protobuf";
import {
  AccessService,
  GetWriteAccessRequestSchema,
  SignInRequestSchema,
  SignOutRequestSchema,
} from "@vrooli/proto-types/money-ledger/v1/access/access_pb";

import { transport } from "./client";

/**
 * Access is Money Ledger's own answer to "may this browser write, and as
 * whom". The session lives in HttpOnly cookies the API sets; this page never
 * sees a token, so every call here goes to the same origin and the browser
 * attaches the cookies itself.
 */
export const accessClient = createClient(AccessService, transport);

export function fetchWriteAccess() {
  return accessClient.getWriteAccess(create(GetWriteAccessRequestSchema));
}

export function signIn(input: { email: string; password: string; totpCode?: string; recoveryCode?: string; mfaChallenge?: string }) {
  return accessClient.signIn(create(SignInRequestSchema, input));
}

export function signOut() {
  return accessClient.signOut(create(SignOutRequestSchema));
}
