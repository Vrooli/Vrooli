import { useState, type FormEvent } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Code, ConnectError } from "@connectrpc/connect";
import { AlertDialog } from "@vrooli/react-component-library/AlertDialog/2";
import { Button } from "@vrooli/react-component-library/Button/2";

import { identityClient } from "../../api/notifications";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { isRoutedTestMode, livePath } from "../routedTestMode";
import { advanceSessionGeneration } from "../../api/sessionGeneration";
import { clearRecipientContent, sessionQueryKey, useSession } from "./useSession";

export function SignInForm() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [challenge, setChallenge] = useState("");
  const [code, setCode] = useState("");
  const [recovery, setRecovery] = useState(false);
  const login = useMutation({
    mutationFn: () => identityClient.login({ email: email.trim(), password, ...(challenge ? { mfaChallenge: challenge, ...(recovery ? { recoveryCode: code } : { totpCode: code }) } : {}) }),
    onSuccess: async (response) => {
      if (response.mfaRequired) {
        setChallenge(response.mfaChallenge);
        setCode("");
        return;
      }
      advanceSessionGeneration();
      setChallenge("");
      setCode("");
      setPassword("");
      // Everything read while signed out was refused (a query that failed with the
      // sign-in card on screen) or belongs to nobody. Refetching all of it is what
      // brings the page the owner came for, such as a record opened from a push,
      // back after they sign in, without a reload.
      await queryClient.invalidateQueries();
    },
    onError: (cause) => {
      // An expired or already-used challenge cannot be retried. Start a new
      // challenge on the next submission, including after an invalid code.
      if (cause instanceof ConnectError && cause.code === Code.Unauthenticated) {
        setChallenge("");
        setCode("");
      }
    },
  });
  const submit = (event: FormEvent) => {
    event.preventDefault();
    if (window.isSecureContext && email.trim() && password && (!challenge || code.trim())) login.mutate();
  };
  const error = login.error
    ? login.error instanceof ConnectError &&
      login.error.code === Code.Unauthenticated
      ? t(strings.session.invalid)
      : t(strings.session.unavailable)
    : null;

  if (!window.isSecureContext) return <p role="alert">{t(strings.session.secureRequired)}</p>;

  return (
    <form
      data-testid={selectors.session.form}
      onSubmit={submit}
      className="flex w-full max-w-sm flex-col gap-3"
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
      {challenge && <>
        <label className="flex flex-col gap-1 text-sm">
          {t(recovery ? strings.session.recoveryCode : strings.session.totpCode)}
          <Input type="text" autoComplete="one-time-code" value={code} onChange={(event) => setCode(event.target.value)} />
        </label>
        <button type="button" onClick={() => { setRecovery(!recovery); setCode(""); }}>{t(recovery ? strings.session.useTotp : strings.session.useRecovery)}</button>
      </>}
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
  const logout = useMutation({
    mutationFn: () => identityClient.logout({}),
    onSuccess: () => {
      setConfirmingLogout(false);
      advanceSessionGeneration();
      const subject = session.data?.signedIn ? session.data.subject : "";
      queryClient.setQueryData(sessionQueryKey, { signedIn: false, subject: "", email: "" });
      if (subject) clearRecipientContent(queryClient, subject);
    },
  });
  if (session.isPending) return null;
  if (session.isError) return <SessionUnavailable retry={() => { void session.refetch(); }} />;
  if (!session.data.signedIn) return (
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
        onClick={() => setConfirmingLogout(true)}
      >
        {t(strings.session.signOutHub)}
      </Button>
      {logout.isSuccess && (
        <p role="status" className="text-sm text-app-muted-foreground">
          {t(strings.session.signOutSuccess)}
        </p>
      )}
      <AlertDialog
        open={confirmingLogout}
        title={t(strings.session.signOutConfirmTitle)}
        description={t(strings.session.signOutDescription)}
        destructive
        busy={logout.isPending}
        status={logout.isError ? "error" : "default"}
        errorMessage={logout.isError ? t(strings.session.signOutUnconfirmed) : undefined}
        confirmLabel={t(strings.session.signOutHub)}
        cancelLabel={t(strings.session.keepSession)}
        closeLabel={t(strings.session.keepSession)}
        onConfirm={() => logout.mutate()}
        onCancel={() => setConfirmingLogout(false)}
      >
        <div className="flex flex-col gap-3 text-sm">
          <p>
            <span className="font-semibold">{t(strings.session.signOutOtherAuthTitle)}</span>
            {" "}
            {t(strings.session.signOutCookieScope)}
          </p>
          <p className="text-app-muted-foreground">{t(strings.session.signOutDraftNotice)}</p>
        </div>
      </AlertDialog>
    </div>
  );
}

/** A failed status check is not evidence that the owner is signed out. */
export function SessionUnavailable({ retry }: { retry: () => void }) {
  const { t } = useTranslation();
  const retryOrOpenLive = () => {
    if (typeof window !== "undefined" && isRoutedTestMode(window.location.search)) {
      window.location.assign(livePath(`${window.location.pathname}${window.location.search}${window.location.hash}`));
      return;
    }
    retry();
  };
  return (
    <section role="alert" className="flex flex-col items-start gap-3 rounded-panel border border-app-border bg-app-surface p-4">
      <p>{t(strings.errors.unavailable)}</p>
      <Button type="button" variant="secondary" onClick={retryOrOpenLive}>{t(strings.common.retry)}</Button>
    </section>
  );
}
