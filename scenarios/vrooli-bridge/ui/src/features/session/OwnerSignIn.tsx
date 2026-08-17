import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { Code, ConnectError } from "@connectrpc/connect";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { identityClient } from "../../api/identity";
import { useSession } from "./SessionProvider";
import {
  generateBrowserKeyMaterial,
  mintBrowserSession,
  saveBrowserEnrollment,
} from "./browser_session";
import { clearEnrollmentBootstrapToken, setEnrollmentBootstrapToken } from "./store";

type Mode = "signin" | "register";

/**
 * Owner sign-in / registration surface. Posts SAME-ORIGIN to the control plane's
 * own IdentityService (`api/identity`) — the bridge forwards to
 * scenario-authenticator via api-core/discovery. The returned provider token is
 * used once to enroll a browser signing key, then discarded; owner RPCs use
 * short-lived locally minted LocalSession credentials. The browser never makes
 * a cross-origin call.
 *
 * When an owner token is already present it renders the signed-in state
 * ("Signed in as …" + Sign out) instead of the form, so the same component is
 * the account surface on the Settings page and the sign-in surface on the
 * unauthenticated gate. A fresh user picks "Create account"; registration
 * auto-issues a token, so either path lands signed-in.
 */
export function OwnerSignIn() {
  const { t } = useTranslation();
  const { isOwner, ownerEmail, setOwnerToken, clearOwnerToken } = useSession();
  const [mode, setMode] = useState<Mode>("signin");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [username, setUsername] = useState("");
  const [localError, setLocalError] = useState<string | null>(null);

  const mutation = useMutation({
    mutationFn: async (): Promise<{ token: string; email: string }> => {
      const trimmed = email.trim();
      const resp =
        mode === "register"
          ? await identityClient.register({ email: trimmed, password, username: username.trim() })
          : await identityClient.login({ email: trimmed, password });
      const keyMaterial = await generateBrowserKeyMaterial();
      setEnrollmentBootstrapToken(resp.token);
      try {
        const enrolled = await identityClient.enrollOperatorSession({
          publicKey: keyMaterial.publicKey,
          mode: "personal",
          requestedScopes: [],
        });
        const enrollment = {
          operatorId: enrolled.operatorId,
          identityProvider: enrolled.identityProvider,
          mode: enrolled.mode,
          reference: enrolled.enrollmentReference,
          enrolledAt: new Date().toISOString(),
          scopeCeiling: enrolled.scopeCeiling,
          privateKeyPkcs8: keyMaterial.privateKeyPkcs8,
        };
        await saveBrowserEnrollment(enrollment);
        return { token: await mintBrowserSession(enrollment), email: resp.email };
      } finally {
        clearEnrollmentBootstrapToken();
      }
    },
    onSuccess: (resp) => {
      setOwnerToken(resp.token, resp.email || email.trim() || null);
      // Drop the password the moment it is no longer needed; it never persists.
      setPassword("");
    },
  });

  if (isOwner) {
    return (
      <div data-testid={selectors.session.owner.panel} className="flex flex-wrap items-center gap-3">
        <p data-testid={selectors.session.owner.status} className="text-sm text-app-success">
          {ownerEmail
            ? t(strings.session.signedInAs, { email: ownerEmail })
            : t(strings.session.signedIn)}
        </p>
        <Button
          data-testid={selectors.session.owner.signOutButton}
          variant="outline"
          size="sm"
          onClick={clearOwnerToken}
        >
          {t(strings.session.signOut)}
        </Button>
      </div>
    );
  }

  const submit = () => {
    setLocalError(null);
    if (!email.trim() || !password) {
      setLocalError(t(strings.session.login.missingFields));
      return;
    }
    mutation.mutate();
  };

  const switchMode = (next: Mode) => {
    setMode(next);
    setLocalError(null);
    mutation.reset();
  };

  const errorText = (): string | null => {
    if (localError) return localError;
    const err = mutation.error;
    if (!err) return null;
    if (err instanceof ConnectError) {
      switch (err.code) {
        case Code.Unauthenticated:
          return t(strings.session.login.invalidCredentials);
        case Code.AlreadyExists:
          return t(strings.session.register.emailTaken);
        case Code.InvalidArgument:
          return t(strings.session.register.invalidInput, { detail: err.rawMessage });
        case Code.Unavailable:
          return t(strings.session.login.unavailable);
        default:
          return t(strings.session.login.unavailable);
      }
    }
    return t(strings.session.login.unavailable);
  };
  const shownError = errorText();

  const tabClass = (active: boolean) =>
    `flex-1 rounded-control px-3 py-2 text-sm font-medium transition-colors ${
      active
        ? "bg-app-primary text-app-primary-foreground"
        : "bg-app-surface-muted text-app-muted-foreground hover:text-app-foreground"
    }`;

  const submitLabel =
    mode === "register"
      ? mutation.isPending
        ? t(strings.session.register.submitting)
        : t(strings.session.register.submit)
      : mutation.isPending
        ? t(strings.session.login.submitting)
        : t(strings.session.login.submit);

  return (
    <section
      data-testid={selectors.session.login.form}
      aria-labelledby="owner-login-heading"
      className="rounded-panel border border-app-border bg-app-surface p-6 shadow-sm"
    >
      <h2 id="owner-login-heading" className="text-xl font-semibold text-app-foreground">
        {t(strings.session.login.title)}
      </h2>
      <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.session.login.intro)}</p>

      <div role="tablist" aria-label={t(strings.session.login.title)} className="mt-5 flex gap-2">
        <button
          type="button"
          role="tab"
          data-testid={selectors.session.login.tabSignIn}
          aria-selected={mode === "signin"}
          className={tabClass(mode === "signin")}
          onClick={() => switchMode("signin")}
        >
          {t(strings.session.login.tabSignIn)}
        </button>
        <button
          type="button"
          role="tab"
          data-testid={selectors.session.login.tabCreate}
          aria-selected={mode === "register"}
          className={tabClass(mode === "register")}
          onClick={() => switchMode("register")}
        >
          {t(strings.session.login.tabCreate)}
        </button>
      </div>

      <form
        className="mt-5 flex flex-col gap-4"
        onSubmit={(e) => {
          e.preventDefault();
          submit();
        }}
      >
        <div className="flex flex-col gap-2">
          <label htmlFor="owner-login-email" className="text-xs font-medium text-app-muted-foreground">
            {t(strings.session.login.emailLabel)}
          </label>
          <Input
            id="owner-login-email"
            data-testid={selectors.session.login.emailInput}
            type="email"
            autoComplete="username"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder={t(strings.session.login.emailPlaceholder)}
          />
        </div>
        <div className="flex flex-col gap-2">
          <label htmlFor="owner-login-password" className="text-xs font-medium text-app-muted-foreground">
            {t(strings.session.login.passwordLabel)}
          </label>
          <Input
            id="owner-login-password"
            data-testid={selectors.session.login.passwordInput}
            type="password"
            autoComplete={mode === "register" ? "new-password" : "current-password"}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder={t(strings.session.login.passwordPlaceholder)}
          />
        </div>
        {mode === "register" && (
          <div className="flex flex-col gap-2">
            <label htmlFor="owner-login-username" className="text-xs font-medium text-app-muted-foreground">
              {t(strings.session.login.usernameLabel)}
            </label>
            <Input
              id="owner-login-username"
              data-testid={selectors.session.login.usernameInput}
              type="text"
              autoComplete="nickname"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder={t(strings.session.login.usernamePlaceholder)}
            />
          </div>
        )}
        <Button type="submit" data-testid={selectors.session.login.submit} disabled={mutation.isPending}>
          {submitLabel}
        </Button>
      </form>

      {shownError && (
        <p data-testid={selectors.session.login.error} role="alert" className="mt-4 text-sm text-app-danger">
          {shownError}
        </p>
      )}
    </section>
  );
}
