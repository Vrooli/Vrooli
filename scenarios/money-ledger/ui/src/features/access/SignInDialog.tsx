import { useEffect, useId, useRef, useState, type FormEvent } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Code, ConnectError } from "@connectrpc/connect";
import { Button } from "@vrooli/react-component-library/Button/2";
import { PasswordInput } from "@vrooli/react-component-library/PasswordInput/2";

import { signIn } from "../../api/access";
import { Dialog } from "../../components/ui/dialog";
import { Input } from "@vrooli/react-component-library/Input/1.3.6";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { refusalKey } from "./useWriteAccess";

function signInErrorKey(error: unknown) {
  if (error instanceof ConnectError) {
    switch (error.code) {
      case Code.Unauthenticated:
        return strings.access.signIn.invalid;
      case Code.InvalidArgument:
        return strings.access.signIn.incomplete;
      case Code.ResourceExhausted:
        return strings.access.signIn.throttled;
    }
  }
  return strings.access.signIn.unavailable;
}

/**
 * Signs the owner in through Money Ledger's own API, which forwards the
 * credentials to the installation's sign-in service and keeps the session in a
 * cookie this page cannot read. Signing in proves who someone is; the API
 * still decides whether that person owns these records, and the dialog stays
 * open to say so when they do not.
 */
export function SignInDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const [challenge, setChallenge] = useState("");
  const [refusal, setRefusal] = useState<{ key: ReturnType<typeof refusalKey>; person: string } | null>(null);
  const codeField = useRef<HTMLInputElement>(null);
  const fieldId = useId();

  // The second-factor field appears after the password is accepted; move the
  // reader to it rather than leaving focus on a now-disabled field.
  useEffect(() => {
    if (challenge) codeField.current?.focus();
  }, [challenge]);

  const login = useMutation({
    mutationFn: () => {
      const trimmed = code.trim();
      const recovery = trimmed.length > 0 && !/^\d+$/.test(trimmed);
      return signIn({ email: email.trim(), password, mfaChallenge: challenge, totpCode: recovery ? "" : trimmed, recoveryCode: recovery ? trimmed : "" });
    },
    onSuccess: async (response) => {
      if (response.mfaRequired) {
        setChallenge(response.mfaChallenge);
        return;
      }
      setPassword("");
      setCode("");
      setChallenge("");
      // Everything read before signing in belongs to a signed-out page;
      // refetching it all brings the owner's view back without a reload.
      await queryClient.invalidateQueries();
      const access = response.access;
      if (access && !access.canWrite) {
        setRefusal({ key: refusalKey(access.refusal), person: access.signedInAs });
        return;
      }
      setRefusal(null);
      onClose();
    },
  });

  const close = () => {
    setPassword("");
    setCode("");
    setChallenge("");
    setRefusal(null);
    login.reset();
    onClose();
  };

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setRefusal(null);
    if (!email.trim() || !password || (challenge && !code.trim())) return;
    login.mutate();
  };

  return (
    <Dialog open={open} title={t(strings.access.signIn.title)} description={t(strings.access.signIn.intro)} onClose={close} closeLabel={t(strings.access.signIn.cancel)}>
      <form data-testid={selectors.access.signInForm} className="grid gap-3" onSubmit={submit} aria-label={t(strings.access.signIn.title)}>
        <label className="grid gap-1" htmlFor={`${fieldId}-email`}>
          <span>{t(strings.access.signIn.email)}</span>
          <Input id={`${fieldId}-email`} data-testid={selectors.access.email} type="email" autoComplete="username" value={email} onChange={(event) => setEmail(event.target.value)} disabled={Boolean(challenge)} required />
        </label>
        <PasswordInput id={`${fieldId}-secret`} testId={selectors.access.password} label={t(strings.access.signIn.password)} autoComplete="current-password" value={password} onValueChange={setPassword} disabled={Boolean(challenge)} required revealLabel={t(strings.access.signIn.revealPassword)} concealLabel={t(strings.access.signIn.concealPassword)} capsLockLabel={t(strings.access.signIn.capsLock)} />
        {challenge && (
          <label className="grid gap-1" htmlFor={`${fieldId}-code`}>
            <span>{t(strings.access.signIn.code)}</span>
            <Input ref={codeField} id={`${fieldId}-code`} data-testid={selectors.access.code} autoComplete="one-time-code" value={code} onChange={(event) => setCode(event.target.value)} required />
            <small className="text-app-muted-foreground">{t(strings.access.signIn.codeIntro)}</small>
          </label>
        )}
        {login.isError && <p data-testid={selectors.access.signInError} role="alert" className="text-sm text-app-danger">{t(signInErrorKey(login.error))}</p>}
        {refusal && <p data-testid={selectors.access.signInRefusal} role="alert" className="text-sm text-app-danger">{t(refusal.key, { person: refusal.person })}</p>}
        <div className="flex flex-wrap justify-end gap-2">
          <Button type="button" variant="secondary" onClick={close}>{t(strings.access.signIn.cancel)}</Button>
          <Button type="submit" data-testid={selectors.access.signInSubmit} disabled={login.isPending}>{login.isPending ? t(strings.access.signIn.submitting) : t(strings.access.signIn.submit)}</Button>
        </div>
      </form>
    </Dialog>
  );
}
