import { useState } from "react";
import { Code, ConnectError } from "@connectrpc/connect";
import { create } from "@bufbuild/protobuf";
import { Button } from "./ui/button";
import { LoginRequestSchema } from "@vrooli/proto-types/git-control-tower/v1/auth/auth_pb";
import { authClient } from "../lib/connect";

interface AuthenticatorSignInProps {
  onSignedIn: () => void;
  onContinueReadOnly: () => void;
}

/** Same-origin sign-in surface backed by scenario-authenticator via GCT API. */
export function AuthenticatorSignIn({ onSignedIn, onContinueReadOnly }: AuthenticatorSignInProps) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function submit() {
    if (!email.trim() || !password) {
      setError("Email and password are required.");
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      await authClient.login(create(LoginRequestSchema, {
        email: email.trim(),
        password,
      }));
      setPassword("");
      onSignedIn();
    } catch (cause) {
      if (cause instanceof ConnectError) {
        setError(cause.code === Code.Unauthenticated ? "Invalid email or password." : cause.rawMessage);
      } else {
        setError("Unable to reach scenario-authenticator. Try again.");
      }
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="mt-3 rounded-md border border-slate-700 bg-slate-950/50 p-3" data-testid="authenticator-sign-in">
      <p className="text-xs font-medium text-slate-200">Sign in through Scenario Authenticator</p>
      <p className="mt-1 text-xs text-slate-400">Sign-in is required only for repository mutations. Browsing remains available without it.</p>
      <div
        className="mt-3 space-y-2"
        role="group"
      >
        <input
          aria-label="Email"
          autoComplete="username"
          className="w-full rounded border border-slate-700 bg-slate-900 px-2 py-1.5 text-xs text-slate-100 outline-none focus:border-cyan-500"
          onChange={(event) => setEmail(event.target.value)}
          placeholder="you@example.com"
          type="email"
          value={email}
        />
        <input
          aria-label="Password"
          autoComplete="current-password"
          className="w-full rounded border border-slate-700 bg-slate-900 px-2 py-1.5 text-xs text-slate-100 outline-none focus:border-cyan-500"
          onChange={(event) => setPassword(event.target.value)}
          placeholder="Password"
          type="password"
          value={password}
        />
        {error && <p className="text-xs text-rose-300" role="alert">{error}</p>}
        <div className="flex flex-wrap gap-2">
          <Button disabled={submitting} onClick={() => void submit()} size="sm" type="button" data-testid="authenticator-submit">
            {submitting ? "Signing in…" : "Sign in"}
          </Button>
          <Button onClick={onContinueReadOnly} size="sm" type="button" variant="outline">
            Continue read-only
          </Button>
        </div>
      </div>
    </div>
  );
}
