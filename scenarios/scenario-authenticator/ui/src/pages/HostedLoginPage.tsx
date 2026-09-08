import { useState } from "react";
import { Code, ConnectError } from "@connectrpc/connect";
import { create } from "@bufbuild/protobuf";
import { useNavigate, useSearchParams } from "react-router-dom";

import { accountsClient } from "../api/client";
import { LoginRequestSchema, RegisterRequestSchema } from "@vrooli/proto-types/scenario-authenticator/v1/accounts/accounts_pb";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Input } from "@vrooli/react-component-library/Input/1";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

type Mode = "sign-in" | "register";

/**
 * Hosted end-user entry point. The browser calls only the authenticator's own
 * same-origin AccountsService. Adopting scenarios should use their own
 * same-origin facade for application sessions; this page is the reusable IdP
 * surface and intentionally never places bearer tokens in a redirect URL.
 */
export function HostedLoginPage() {
  const { t } = useTranslation();
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
      setError(t(strings.auth.requiredFields));
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
      if (!response.tokens?.accessToken) throw new Error(t(strings.auth.unableToComplete));
      // The same-origin API sets HttpOnly access/refresh cookies. Keep the
      // bearer material out of localStorage and other JavaScript-readable
      // persistence; only display the non-secret account summary in memory.
      setSignedInEmail(response.account?.email || email.trim());
      setPassword("");
    } catch (cause) {
      if (cause instanceof ConnectError) {
        setError(cause.code === Code.Unauthenticated ? t(strings.auth.invalidCredentials) : cause.rawMessage);
      } else if (cause instanceof Error) {
        setError(cause.message);
      } else {
        setError(t(strings.auth.unableToComplete));
      }
    } finally {
      setSubmitting(false);
    }
  }

  if (signedInEmail) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-app-background px-4 text-app-foreground">
        <section className="w-full max-w-md rounded-panel border border-app-border bg-app-surface p-8 shadow-sm" data-testid="hosted-login-success">
          <h1 className="text-2xl font-semibold">{t(strings.auth.signedInHeading)}</h1>
          <p className="mt-2 text-app-muted-foreground">{t(strings.auth.signedInAs, { email: signedInEmail })}</p>
          <p className="mt-4 text-sm text-app-muted-foreground">{t(strings.auth.returnToApplication)}</p>
          <Button className="mt-6" onClick={() => navigate("/")} type="button">{t(strings.app.title)}</Button>
        </section>
      </main>
    );
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-app-background px-4 text-app-foreground">
      <section className="w-full max-w-md rounded-panel border border-app-border bg-app-surface p-8 shadow-sm" data-testid="hosted-login-page">
        <p className="text-xs uppercase tracking-wide text-app-muted-foreground">{t(strings.app.title)}</p>
        <h1 className="mt-2 text-2xl font-semibold">{t(strings.auth.signIn)}</h1>
        <p className="mt-2 text-sm text-app-muted-foreground">{t(strings.auth.signInDescription)}</p>
        {searchParams.get("return_to") && (
          <p data-testid="auth-return-target" className="mt-3 rounded-control border border-app-border bg-app-surface-muted p-3 text-xs text-app-muted-foreground">{t(strings.auth.returnTarget, { target: searchParams.get("return_to") ?? "" })}</p>
        )}
        <div className="mt-6 flex gap-2" role="tablist" aria-label={t(strings.auth.accountAction)}>
          <Button data-testid="auth-sign-in-tab" onClick={() => setMode("sign-in")} size="sm" type="button" variant={mode === "sign-in" ? "primary" : "secondary"}>{t(strings.auth.signIn)}</Button>
          <Button data-testid="auth-register-tab" onClick={() => setMode("register")} size="sm" type="button" variant={mode === "register" ? "primary" : "secondary"}>{t(strings.auth.createAccount)}</Button>
        </div>
        <div className="mt-5 space-y-4">
          <label className="block text-sm" htmlFor="auth-email">{t(strings.auth.email)}<Input data-testid="auth-email" id="auth-email" autoComplete="username" className="mt-1" onChange={(event) => setEmail(event.target.value)} type="email" value={email} /></label>
          <label className="block text-sm" htmlFor="auth-password">{t(strings.auth.password)}<Input data-testid="auth-password" id="auth-password" autoComplete={mode === "register" ? "new-password" : "current-password"} className="mt-1" onChange={(event) => setPassword(event.target.value)} type="password" value={password} /></label>
          {mode === "register" && <label className="block text-sm" htmlFor="auth-username">{t(strings.auth.username)}<Input data-testid="auth-username" id="auth-username" autoComplete="nickname" className="mt-1" onChange={(event) => setUsername(event.target.value)} type="text" value={username} /></label>}
          {error && <p className="text-sm text-red-500" role="alert">{error}</p>}
          <Button className="w-full" data-testid="auth-submit" onClick={() => void submit()} pending={submitting} pendingLabel={t(strings.auth.working)} type="button">{mode === "register" ? t(strings.auth.createAccount) : t(strings.auth.signIn)}</Button>
          <Button className="w-full" data-testid="auth-read-only" onClick={() => navigate("/")} type="button" variant="secondary">{t(strings.auth.viewReadOnly)}</Button>
        </div>
      </section>
    </main>
  );
}
