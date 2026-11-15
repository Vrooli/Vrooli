import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";

import { signOut } from "../../api/access";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { SignInDialog } from "./SignInDialog";
import { refusalKey, signInCanHelp, useWriteAccess } from "./useWriteAccess";

/**
 * Settings section for who this browser is to Money Ledger: the signed-in
 * writer with a sign-out, or why it is viewing only with a sign-in.
 */
export function AccessPanel() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const access = useWriteAccess();
  const [signingIn, setSigningIn] = useState(false);
  const logout = useMutation({
    mutationFn: signOut,
    onSettled: () => queryClient.invalidateQueries(),
  });
  const data = access.data;
  if (!data) {
    return <p data-testid={selectors.access.status} role="status" className="text-sm text-app-muted-foreground">{access.isError ? t(strings.access.unknown) : t(strings.access.checking)}</p>;
  }
  return (
    <div className="grid gap-3">
      <p data-testid={selectors.access.status} role="status" className="text-sm">
        {data.canWrite
          ? t(data.browserSession ? strings.access.writer : strings.access.tunnelWriter, { person: data.signedInAs || data.actor, actor: data.actor })
          : t(refusalKey(data.refusal), { person: data.signedInAs })}
      </p>
      {!data.canWrite && <p className="text-sm text-app-muted-foreground">{t(strings.access.cliHint)}</p>}
      <div className="flex flex-wrap gap-2">
        {!data.canWrite && signInCanHelp(data) && <Button type="button" data-testid={selectors.access.signIn} onClick={() => setSigningIn(true)}>{t(strings.access.signInAction)}</Button>}
        {data.browserSession && <Button type="button" variant="secondary" data-testid={selectors.access.signOut} disabled={logout.isPending} onClick={() => logout.mutate()}>{t(strings.access.signOutAction)}</Button>}
      </div>
      {logout.isSuccess && <p role="status" className="text-sm text-app-muted-foreground">{logout.data.revocationConfirmed ? t(strings.access.signedOut) : t(strings.access.signOutUnconfirmed)}</p>}
      {logout.isError && <p role="alert" className="text-sm text-app-danger">{t(strings.access.signOutFailed)}</p>}
      <SignInDialog open={signingIn} onClose={() => setSigningIn(false)} />
    </div>
  );
}
