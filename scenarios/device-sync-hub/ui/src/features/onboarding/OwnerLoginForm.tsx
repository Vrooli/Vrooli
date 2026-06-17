import { useMemo, useState } from "react";
import { useMutation } from "@tanstack/react-query";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import {
  AuthError,
  loginOwner,
  resolveAuthenticatorBaseUrl,
} from "../../api/authenticator";
import { useSession } from "../session/SessionProvider";

/**
 * "Set up this hub" → owner sign-in. Posts credentials to scenario-authenticator
 * (resolved via `resolveAuthenticatorBaseUrl`) and stores the returned owner JWT
 * in the session; SetupOwnerDevice then becomes reachable. When the authenticator
 * URL can't be resolved for this deployment, the form degrades to the Advanced
 * owner-token paste, which needs no authenticator URL.
 */
export function OwnerLoginForm({ onBack }: { onBack: () => void }) {
  const { t } = useTranslation();
  const { setOwnerToken } = useSession();
  const authBase = useMemo(() => resolveAuthenticatorBaseUrl(), []);

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [localError, setLocalError] = useState<string | null>(null);
  const [advancedOpen, setAdvancedOpen] = useState(authBase === null);
  const [token, setToken] = useState("");

  const loginMutation = useMutation({
    mutationFn: () => {
      if (authBase === null) {
        throw new AuthError("unavailable", t(strings.login.unconfigured));
      }
      return loginOwner(authBase, { email: email.trim(), password });
    },
    onSuccess: (identity) => setOwnerToken(identity.token, identity.email ?? (email.trim() || null)),
  });

  const handleLogin = () => {
    setLocalError(null);
    if (authBase === null) {
      setLocalError(t(strings.login.unconfigured));
      setAdvancedOpen(true);
      return;
    }
    if (!email.trim() || !password) {
      setLocalError(t(strings.login.missingFields));
      return;
    }
    loginMutation.mutate();
  };

  const handlePasteToken = () => {
    setLocalError(null);
    if (!token.trim()) {
      setLocalError(t(strings.owner.missingToken));
      return;
    }
    setOwnerToken(token.trim());
  };

  const loginErrorMessage = (): string | null => {
    if (localError) return localError;
    const err = loginMutation.error;
    if (!err) return null;
    if (err instanceof AuthError && err.code === "invalid_credentials") {
      return t(strings.login.invalidCredentials);
    }
    return t(strings.login.unavailable);
  };
  const shownError = loginErrorMessage();

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

      {authBase === null && (
        <p className="mt-4 rounded-control border border-app-border bg-app-surface-muted p-3 text-sm text-app-muted-foreground">
          {t(strings.login.unconfigured)}
        </p>
      )}

      {authBase !== null && (
        <form
          className="mt-6 flex flex-col gap-4"
          onSubmit={(e) => {
            e.preventDefault();
            handleLogin();
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
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder={t(strings.login.passwordPlaceholder)}
            />
          </div>
          <Button
            type="submit"
            data-testid={selectors.login.submit}
            disabled={loginMutation.isPending}
          >
            {loginMutation.isPending ? t(strings.login.submitting) : t(strings.login.submit)}
          </Button>
        </form>
      )}

      <div className="mt-6 border-t border-app-border pt-4">
        <button
          type="button"
          data-testid={selectors.login.advancedToggle}
          aria-expanded={advancedOpen}
          className="text-sm font-medium text-app-muted-foreground hover:text-app-foreground"
          onClick={() => setAdvancedOpen((v) => !v)}
        >
          {t(strings.login.advancedToggle)}
        </button>
        {advancedOpen && (
          <div className="mt-3 flex flex-col gap-2">
            <p className="text-xs text-app-muted-foreground">{t(strings.login.advancedIntro)}</p>
            <label htmlFor="owner-token" className="sr-only">
              {t(strings.owner.tokenLabel)}
            </label>
            <Input
              id="owner-token"
              data-testid={selectors.owner.tokenInput}
              type="password"
              value={token}
              onChange={(e) => setToken(e.target.value)}
              placeholder={t(strings.owner.tokenPlaceholder)}
            />
            <Button
              data-testid={selectors.owner.signInButton}
              variant="outline"
              size="sm"
              className="w-fit"
              onClick={handlePasteToken}
            >
              {t(strings.owner.signIn)}
            </Button>
          </div>
        )}
      </div>

      {shownError && (
        <p data-testid={selectors.login.error} className="mt-4 text-sm text-app-danger">
          {shownError}
        </p>
      )}

      <Button
        data-testid={selectors.onboarding.back}
        variant="outline"
        size="sm"
        className="mt-6"
        onClick={onBack}
      >
        {t(strings.onboarding.back)}
      </Button>
    </section>
  );
}
