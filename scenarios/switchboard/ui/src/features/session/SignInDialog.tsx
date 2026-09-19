import { KeyRound } from "lucide-react";
import { useState, type FormEvent } from "react";

import { Button } from "@vrooli/react-component-library/Button/2";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";

import { LoginError, login } from "../../api/session";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

interface SignInDialogProps {
  onSignedIn: () => void;
  onClose: () => void;
}

/**
 * Owner sign-in, raised only when a write needs it. Credentials go to the
 * same-origin login facade and never touch storage; only the issued token does.
 */
export function SignInDialog({ onSignedIn, onClose }: SignInDialogProps) {
  const { t } = useTranslation();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string>();

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    if (!email.trim() || !password) return;
    setPending(true);
    setError(undefined);
    try {
      await login(email.trim(), password);
      setPassword("");
      onSignedIn();
    } catch (cause) {
      setError(cause instanceof LoginError && cause.status === 401 ? t(strings.console.session.badCredentials) : cause instanceof Error ? cause.message : t(strings.errors.unknown));
    } finally {
      setPending(false);
    }
  };

  return (
    <ResponsiveDialog open onClose={onClose} title={t(strings.console.session.title)} closeLabel={t(strings.console.common.close)} size="sm" testId="session-sign-in">
      <form onSubmit={(event) => void submit(event)} className="flex flex-col gap-4">
        <p className="flex items-start gap-2 text-sm text-app-muted-foreground">
          <KeyRound aria-hidden="true" className="mt-0.5 h-4 w-4 shrink-0" />
          {t(strings.console.session.description)}
        </p>
        <label className="flex flex-col gap-1 text-sm font-medium" htmlFor="session-email">
          {t(strings.console.session.email)}
          <input
            id="session-email"
            data-testid="session-email"
            type="email"
            autoComplete="username"
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            required
            className="min-h-10 rounded-control border border-app-border bg-app-surface px-3 text-base font-normal md:text-sm"
          />
        </label>
        <label className="flex flex-col gap-1 text-sm font-medium" htmlFor="session-password">
          {t(strings.console.session.password)}
          <input
            id="session-password"
            data-testid="session-password"
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            required
            className="min-h-10 rounded-control border border-app-border bg-app-surface px-3 text-base font-normal md:text-sm"
          />
        </label>
        {error ? (
          <p role="alert" className="text-sm text-app-danger">
            {error}
          </p>
        ) : null}
        <div className="flex justify-end gap-2">
          <Button type="button" variant="ghost" onClick={onClose}>
            {t(strings.console.common.cancel)}
          </Button>
          <Button type="submit" data-testid="session-submit" pending={pending} disabled={!email.trim() || !password}>
            {t(strings.console.session.signIn)}
          </Button>
        </div>
      </form>
    </ResponsiveDialog>
  );
}
