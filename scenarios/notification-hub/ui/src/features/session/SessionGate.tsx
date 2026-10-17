import { useState, type FormEvent, type ReactNode } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Code, ConnectError } from "@connectrpc/connect";
import { Button } from "@vrooli/react-component-library/Button/2";

import { identityClient } from "../../api/notifications";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

export const sessionQueryKey = ["owner-session"] as const;

/**
 * The owner session lives in an HttpOnly cookie, so the page asks the hub who
 * is signed in. GetSession also renews an expired session from the refresh
 * cookie, which keeps a push tap days later signed in.
 */
export function useSession() {
  return useQuery({
    queryKey: sessionQueryKey,
    queryFn: () => identityClient.getSession({}),
    staleTime: 60_000,
  });
}

/** Renders children for a signed-in owner and the sign-in form otherwise. */
export function SessionGate({ children }: { children: ReactNode }) {
  const session = useSession();
  if (session.isPending) return null;
  if (!session.data?.signedIn) return <SignInForm />;
  return <>{children}</>;
}

export function SignInForm() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const login = useMutation({
    mutationFn: () => identityClient.login({ email: email.trim(), password }),
    onSuccess: async () => {
      setPassword("");
      await queryClient.invalidateQueries({ queryKey: sessionQueryKey });
    },
  });
  const submit = (event: FormEvent) => {
    event.preventDefault();
    if (email.trim() && password) login.mutate();
  };
  const error = login.error
    ? login.error instanceof ConnectError &&
      login.error.code === Code.Unauthenticated
      ? t(strings.session.invalid)
      : t(strings.session.unavailable)
    : null;

  return (
    <form
      data-testid={selectors.session.form}
      onSubmit={submit}
      className="flex max-w-sm flex-col gap-3"
      aria-labelledby="session-heading"
    >
      <h2 id="session-heading" className="text-xl font-semibold">
        {t(strings.session.title)}
      </h2>
      <p className="text-sm text-app-muted-foreground">
        {t(strings.session.intro)}
      </p>
      <label className="flex flex-col gap-1 text-sm">
        {t(strings.session.email)}
        <Input
          data-testid={selectors.session.email}
          type="email"
          autoComplete="username"
          value={email}
          onChange={(event) => setEmail(event.target.value)}
        />
      </label>
      <label className="flex flex-col gap-1 text-sm">
        {t(strings.session.password)}
        <Input
          data-testid={selectors.session.password}
          type="password"
          autoComplete="current-password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
        />
      </label>
      {error && (
        <p
          data-testid={selectors.session.error}
          role="alert"
          className="text-sm text-app-danger"
        >
          {error}
        </p>
      )}
      <Button
        data-testid={selectors.session.submit}
        type="submit"
        variant="primary"
        disabled={login.isPending}
      >
        {login.isPending
          ? t(strings.session.submitting)
          : t(strings.session.submit)}
      </Button>
    </form>
  );
}

/** Settings panel: the signed-in owner with sign-out, or the sign-in form. */
export function SessionPanel() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const session = useSession();
  const logout = useMutation({
    mutationFn: () => identityClient.logout({}),
    onSettled: () =>
      queryClient.invalidateQueries({ queryKey: sessionQueryKey }),
  });
  if (session.isPending) return null;
  if (!session.data?.signedIn) return <SignInForm />;
  return (
    <div className="flex flex-wrap items-center gap-3">
      <p data-testid={selectors.session.signedIn} className="text-sm">
        {t(strings.session.signedInAs, {
          email: session.data.email || session.data.subject,
        })}
      </p>
      <Button
        data-testid={selectors.session.signOut}
        type="button"
        variant="outline"
        size="sm"
        onClick={() => logout.mutate()}
      >
        {t(strings.session.signOut)}
      </Button>
    </div>
  );
}
