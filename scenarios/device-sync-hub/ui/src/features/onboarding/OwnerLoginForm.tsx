import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { Code, ConnectError } from "@connectrpc/connect";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { identityClient } from "../../api/identity";
import { useSession } from "../session/SessionProvider";

type Mode = "signin" | "register";

/**
 * Owner sign-in / registration. Posts SAME-ORIGIN to the hub's own
 * IdentityService (`api/identity`) — the hub forwards to scenario-authenticator
 * via api-core/discovery and relays the owner JWT, which we store in the
 * session so SetupOwnerDevice (and every owner-gated RPC) becomes reachable. The
 * browser never makes a cross-origin call.
 *
 * Used by the first-run OnboardingScreen (with a Back button) and by Settings
 * re-auth (no Back). A fresh user picks "Create account"; an existing one signs
 * in. Registration auto-issues a token, so either path lands signed-in.
 */
export function OwnerLoginForm({ onBack }: { onBack?: () => void }) {
  const { t } = useTranslation();
  const { setOwnerToken } = useSession();
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
      return { token: resp.token, email: resp.email };
    },
    onSuccess: (resp) => setOwnerToken(resp.token, resp.email || email.trim() || null),
  });

  const submit = () => {
    setLocalError(null);
    if (!email.trim() || !password) {
      setLocalError(t(strings.login.missingFields));
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
          return t(strings.login.invalidCredentials);
        case Code.AlreadyExists:
          return t(strings.register.emailTaken);
        case Code.InvalidArgument:
          return t(strings.register.invalidInput, { detail: err.rawMessage });
        case Code.Unavailable:
          return t(strings.login.unavailable);
        default:
          return t(strings.login.unavailable);
      }
    }
    return t(strings.login.unavailable);
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
        ? t(strings.register.submitting)
        : t(strings.register.submit)
      : mutation.isPending
        ? t(strings.login.submitting)
        : t(strings.login.submit);

  return (
    <section
      data-testid={selectors.login.form}
      aria-labelledby="login-heading"
      className="rounded-panel border border-app-border bg-app-surface p-6 shadow-sm"
    >
      <h1 id="login-heading" className="text-xl font-semibold">
        {t(strings.login.title)}
      </h1>
      <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.login.intro)}</p>

      <div role="tablist" aria-label={t(strings.login.title)} className="mt-5 flex gap-2">
        <button
          type="button"
          role="tab"
          data-testid={selectors.login.tabSignIn}
          aria-selected={mode === "signin"}
          className={tabClass(mode === "signin")}
          onClick={() => switchMode("signin")}
        >
          {t(strings.login.tabSignIn)}
        </button>
        <button
          type="button"
          role="tab"
          data-testid={selectors.login.tabCreate}
          aria-selected={mode === "register"}
          className={tabClass(mode === "register")}
          onClick={() => switchMode("register")}
        >
          {t(strings.login.tabCreate)}
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
          <label htmlFor="login-email" className="text-xs font-medium text-app-muted-foreground">
            {t(strings.login.emailLabel)}
          </label>
          <Input
            id="login-email"
            data-testid={selectors.login.emailInput}
            type="email"
            autoComplete="username"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder={t(strings.login.emailPlaceholder)}
          />
        </div>
        <div className="flex flex-col gap-2">
          <label htmlFor="login-password" className="text-xs font-medium text-app-muted-foreground">
            {t(strings.login.passwordLabel)}
          </label>
          <Input
            id="login-password"
            data-testid={selectors.login.passwordInput}
            type="password"
            autoComplete={mode === "register" ? "new-password" : "current-password"}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder={t(strings.login.passwordPlaceholder)}
          />
        </div>
        {mode === "register" && (
          <div className="flex flex-col gap-2">
            <label htmlFor="login-username" className="text-xs font-medium text-app-muted-foreground">
              {t(strings.register.usernameLabel)}
            </label>
            <Input
              id="login-username"
              data-testid={selectors.login.usernameInput}
              type="text"
              autoComplete="nickname"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder={t(strings.register.usernamePlaceholder)}
            />
          </div>
        )}
        <Button type="submit" data-testid={selectors.login.submit} disabled={mutation.isPending}>
          {submitLabel}
        </Button>
      </form>

      {shownError && (
        <p data-testid={selectors.login.error} className="mt-4 text-sm text-app-danger">
          {shownError}
        </p>
      )}

      {onBack && (
        <Button
          data-testid={selectors.onboarding.back}
          variant="outline"
          size="sm"
          className="mt-6"
          onClick={onBack}
        >
          {t(strings.onboarding.back)}
        </Button>
      )}
    </section>
  );
}
