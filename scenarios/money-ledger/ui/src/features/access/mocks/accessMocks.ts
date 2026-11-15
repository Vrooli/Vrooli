/**
 * Test doubles for the access boundary (`api/access`).
 *
 * The test setup mocks `api/access` for every test with `makeAccessMocks()`,
 * so a page renders as the signed-in owner by default and no test reaches the
 * network to ask. A test about another caller overrides the answer:
 *
 *   vi.mocked(fetchWriteAccess).mockResolvedValue(makeWriteAccess({ canWrite: false, refusal: WriteRefusal.SIGNED_OUT }));
 */
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import { vi } from "vitest";
import {
  GetWriteAccessResponseSchema,
  SignInResponseSchema,
  SignOutResponseSchema,
  type GetWriteAccessResponse,
  type SignInResponse,
} from "@vrooli/proto-types/money-ledger/v1/access/access_pb";

/** The owner signed in through the console's own sign-in, by default. */
export function makeWriteAccess(overrides: MessageInitShape<typeof GetWriteAccessResponseSchema> = {}): GetWriteAccessResponse {
  const canWrite = overrides.canWrite ?? true;
  return create(GetWriteAccessResponseSchema, {
    canWrite,
    actor: canWrite ? "operator" : "",
    reason: canWrite ? "writes are recorded as operator" : "refused",
    signedInAs: canWrite ? "owner@example.test" : "",
    browserSession: canWrite,
    ...overrides,
  });
}

export function makeSignInResponse(overrides: MessageInitShape<typeof SignInResponseSchema> = {}): SignInResponse {
  return create(SignInResponseSchema, { access: makeWriteAccess(), ...overrides });
}

export function makeAccessMocks() {
  return {
    fetchWriteAccess: vi.fn().mockResolvedValue(makeWriteAccess()),
    signIn: vi.fn().mockResolvedValue(makeSignInResponse()),
    signOut: vi.fn().mockResolvedValue(create(SignOutResponseSchema, { revocationConfirmed: true })),
  };
}
