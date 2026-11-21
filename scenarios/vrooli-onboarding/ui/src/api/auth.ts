import { create } from "@bufbuild/protobuf";
import { createClient } from "@connectrpc/connect";
import { AuthService, LoginRequestSchema } from "@vrooli/proto-types/vrooli-onboarding/v1/auth/auth_pb";
import { onboardingTransport } from "./base";

const client = createClient(AuthService, onboardingTransport());

/** Same-origin sign-in facade backed by scenario-authenticator. */
export function loginAuthenticator(email: string, password: string) {
  return client.login(create(LoginRequestSchema, { email: email.trim(), password }));
}
