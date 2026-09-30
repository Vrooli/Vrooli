import { saveSession } from "../features/session/store";

/**
 * Seed an owner session into localStorage BEFORE a render so `SessionProvider`
 * initialises from it. Mirrors what a successful sign-in persists, so a test can
 * render the signed-in shell (or an owner-gated surface) without driving the
 * sign-in form.
 */
export function seedSession(opts: { ownerToken?: string; ownerEmail?: string | null } = {}): void {
  saveSession({
    ownerToken: opts.ownerToken ?? "owner-token-test",
    ownerEmail: opts.ownerEmail ?? null,
  });
}
