import { useEffect, useRef, useState, type FormEvent } from "react";
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
    refetchInterval: 60_000,
    refetchOnWindowFocus: true,
  });
}

/** Hide cached recipient content immediately when any protected API call expires. */
export function useSessionExpiryBoundary() {
  const queryClient = useQueryClient();
  useEffect(() => {
    const expire = () => {
      queryClient.setQueryData(sessionQueryKey, {
        signedIn: false,
        subject: "",
        email: "",
      });
    };
    window.addEventListener("notification-hub:session-expired", expire);
    return () => window.removeEventListener("notification-hub:session-expired", expire);
  }, [queryClient]);
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
  const [confirmingLogout, setConfirmingLogout] = useState(false);
  const confirmationDialog = useRef<HTMLDialogElement>(null);
  const logout = useMutation({
    mutationFn: () => identityClient.logout({}),
    onSuccess: () => {
      setConfirmingLogout(false);
      void queryClient.invalidateQueries({ queryKey: sessionQueryKey });
    },
  });
  useEffect(() => {
    const dialog = confirmationDialog.current;
    if (!dialog) return;
    if (confirmingLogout && !dialog.open) {
      if (typeof dialog.showModal === "function") dialog.showModal();
      else dialog.setAttribute("open", "");
    } else if (!confirmingLogout && dialog.open) {
      if (typeof dialog.close === "function") dialog.close();
      else dialog.removeAttribute("open");
    }
  }, [confirmingLogout]);
  if (session.isPending) return null;
  if (!session.data?.signedIn) return (
    <div className="flex flex-col items-start gap-3">
      <p className="text-sm text-app-muted-foreground">{t(strings.session.hubSessionInactive)}</p>
      <SignInForm />
    </div>
  );
  return (
    <div className="flex flex-col items-start gap-3">
      <p className="text-sm text-app-muted-foreground">{t(strings.session.hubSessionActive)}</p>
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
        onClick={() => setConfirmingLogout(true)}
      >
        {t(strings.session.signOutHub)}
      </Button>
      {logout.isSuccess && (
        <p role="status" className="text-sm text-app-muted-foreground">
          {t(strings.session.signOutSuccess)}
        </p>
      )}
      {confirmingLogout && (
        <dialog
          ref={confirmationDialog}
          onCancel={(event) => {
            event.preventDefault();
            setConfirmingLogout(false);
          }}
          onClose={() => setConfirmingLogout(false)}
          role="alertdialog"
          aria-labelledby="session-signout-title"
          aria-describedby="session-signout-description session-signout-drafts"
          className="fixed inset-0 m-0 h-dvh max-h-none w-screen max-w-none rounded-none border-0 bg-app-surface p-0 text-app-foreground shadow-none backdrop:bg-app-shell/70"
        >
          <div className="flex h-full min-h-0 flex-col">
            <header className="flex min-h-14 items-center justify-between gap-4 border-b border-app-border bg-app-surface-muted px-4 py-3 sm:px-8">
              <button type="button" onClick={() => setConfirmingLogout(false)} className="min-h-11 shrink-0 text-sm font-medium underline">
                ← {t(strings.session.sectionTitle)}
              </button>
              <span className="truncate text-sm font-semibold">{t(strings.app.title)}</span>
              <span aria-hidden="true" className="w-12 shrink-0" />
            </header>
            <main className="mx-auto flex w-full max-w-3xl flex-1 flex-col gap-5 overflow-y-auto px-4 py-5 sm:px-8 sm:py-8 sm:gap-6">
              <p className="text-sm text-app-muted-foreground">{t(strings.session.signOutEyebrow)}</p>
              <h2 id="session-signout-title" className="text-3xl font-bold leading-tight tracking-tight sm:text-5xl">
                {t(strings.session.signOutConfirmTitle)}
              </h2>
              <p className="text-base leading-relaxed sm:text-2xl">
                {t(strings.session.signOutDescription)}
              </p>
              <aside className="border-s-4 border-app-warning py-3 ps-4" aria-label={t(strings.session.signOutOtherAuthTitle)}>
                <p className="text-base font-bold sm:text-xl">{t(strings.session.signOutOtherAuthTitle)}</p>
                <p id="session-signout-description" className="mt-2 text-sm leading-relaxed sm:text-lg">
                  {t(strings.session.signOutCookieScope)}
                </p>
              </aside>
              <p id="session-signout-drafts" className="text-base leading-relaxed sm:text-lg">
                {t(strings.session.signOutDraftNotice)}
              </p>
              {logout.isError && (
                <p role="alert" className="text-sm text-app-danger">
                  {t(strings.session.signOutUnconfirmed)}
                </p>
              )}
            </main>
            <footer className="border-t border-app-border bg-app-surface-muted px-4 py-3 pb-[calc(0.75rem+env(safe-area-inset-bottom))] sm:px-8">
              <div className="mx-auto flex w-full max-w-3xl flex-col gap-2 sm:flex-row">
                <Button
                  className="min-h-12 flex-1"
                  type="button"
                  variant="primary"
                  disabled={logout.isPending}
                  onClick={() => logout.mutate()}
                >
                  {t(strings.session.signOutHub)}
                </Button>
                <Button
                  className="min-h-12 flex-1"
                  type="button"
                  variant="outline"
                  disabled={logout.isPending}
                  autoFocus
                  onClick={() => setConfirmingLogout(false)}
                >
                  {t(strings.session.keepSession)}
                </Button>
              </div>
            </footer>
          </div>
        </dialog>
      )}
    </div>
  );
}
