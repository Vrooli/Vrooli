import { useState } from "react";
import { Code, ConnectError } from "@connectrpc/connect";
import { create } from "@bufbuild/protobuf";
import { useNavigate, useSearchParams } from "react-router-dom";

import { accountsClient } from "../api/client";
import { LoginRequestSchema, RegisterRequestSchema } from "@vrooli/proto-types/scenario-authenticator/v1/accounts/accounts_pb";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";

type Mode = "sign-in" | "register";

/**
 * Hosted end-user entry point. The browser calls only the authenticator's own
 * same-origin AccountsService. Adopting scenarios should use their own
 * same-origin facade for application sessions; this page is the reusable IdP
 * surface and intentionally never places bearer tokens in a redirect URL.
 */
export function HostedLoginPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [mode, setMode] = useState<Mode>("sign-in");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [username, setUsername] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [signedInEmail, setSignedInEmail] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function submit() {
    if (!email.trim() || !password || (mode === "register" && !username.trim())) {
      setError("Email, password, and username are required.");
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      const response = mode === "register"
        ? await accountsClient.register(create(RegisterRequestSchema, {
            email: email.trim(), password, username: username.trim(), realm: "default",
          }))
        : await accountsClient.login(create(LoginRequestSchema, {
            email: email.trim(), password, realm: "default",
          }));
      const tokens = response.tokens;
      if (!tokens?.accessToken) throw new Error("The authenticator returned no access token.");
      window.localStorage.setItem("auth_token", tokens.accessToken);
      if (tokens.refreshToken) window.localStorage.setItem("refresh_token", tokens.refreshToken);
      window.localStorage.setItem("user", JSON.stringify(response.account ?? { email: email.trim() }));
      setSignedInEmail(response.account?.email || email.trim());
      setPassword("");
    } catch (cause) {
      if (cause instanceof ConnectError) {
        setError(cause.code === Code.Unauthenticated ? "Invalid email or password." : cause.rawMessage);
      } else if (cause instanceof Error) {
        setError(cause.message);
      } else {
        setError("Unable to complete sign-in. Try again.");
      }
    } finally {
      setSubmitting(false);
    }
  }

  if (signedInEmail) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-app-background px-4 text-app-foreground">
        <section className="w-full max-w-md rounded-panel border border-app-border bg-app-surface p-8 shadow-sm" data-testid="hosted-login-success">
          <h1 className="text-2xl font-semibold">You are signed in</h1>
          <p className="mt-2 text-app-muted-foreground">Signed in as {signedInEmail}.</p>
          <p className="mt-4 text-sm text-app-muted-foreground">Return to the application that asked you to sign in. Your bearer token was kept out of the URL.</p>
          <Button className="mt-6" onClick={() => navigate("/")} type="button">Open authenticator</Button>
        </section>
      </main>
    );
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-app-background px-4 text-app-foreground">
      <section className="w-full max-w-md rounded-panel border border-app-border bg-app-surface p-8 shadow-sm" data-testid="hosted-login-page">
        <p className="text-xs uppercase tracking-wide text-app-muted-foreground">Scenario Authenticator</p>
        <h1 className="mt-2 text-2xl font-semibold">Sign in</h1>
        <p className="mt-2 text-sm text-app-muted-foreground">Authenticate once, then return to the application that requested access.</p>
        {searchParams.get("return_to") && (
          <p className="mt-3 rounded-control border border-app-border bg-app-surface-muted p-3 text-xs text-app-muted-foreground">Return target: {searchParams.get("return_to")}</p>
        )}
        <div className="mt-6 flex gap-2" role="tablist" aria-label="Account action">
          <Button onClick={() => setMode("sign-in")} size="sm" type="button" variant={mode === "sign-in" ? "default" : "outline"}>Sign in</Button>
          <Button onClick={() => setMode("register")} size="sm" type="button" variant={mode === "register" ? "default" : "outline"}>Create account</Button>
        </div>
        <div className="mt-5 space-y-4">
          <label className="block text-sm">Email<Input autoComplete="username" className="mt-1" onChange={(event) => setEmail(event.target.value)} type="email" value={email} /></label>
          <label className="block text-sm">Password<Input autoComplete={mode === "register" ? "new-password" : "current-password"} className="mt-1" onChange={(event) => setPassword(event.target.value)} type="password" value={password} /></label>
          {mode === "register" && <label className="block text-sm">Username<Input autoComplete="nickname" className="mt-1" onChange={(event) => setUsername(event.target.value)} type="text" value={username} /></label>}
          {error && <p className="text-sm text-red-500" role="alert">{error}</p>}
          <Button className="w-full" disabled={submitting} onClick={() => void submit()} type="button">{submitting ? "Working…" : mode === "register" ? "Create account" : "Sign in"}</Button>
          <Button className="w-full" onClick={() => navigate("/")} type="button" variant="outline">View read-only version</Button>
        </div>
      </section>
    </main>
  );
}
