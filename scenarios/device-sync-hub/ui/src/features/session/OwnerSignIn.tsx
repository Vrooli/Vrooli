import { useState } from "react";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { useSession } from "./SessionProvider";

/**
 * Pragmatic owner sign-in: paste an owner JWT to enable the owner-gated device
 * management RPCs. A full login form posting to scenario-authenticator is a
 * future enhancement; v1 accepts a token paste (and lets the owner clear it).
 * The transfer core works with only a device token, so this is secondary.
 */
export function OwnerSignIn() {
  const { t } = useTranslation();
  const { isOwner, setOwnerToken, clearOwnerToken } = useSession();
  const [token, setToken] = useState("");
  const [error, setError] = useState<string | null>(null);

  const handleSave = () => {
    if (!token.trim()) {
      setError(t(strings.owner.missingToken));
      return;
    }
    setError(null);
    setOwnerToken(token.trim());
    setToken("");
  };

  return (
    <div data-testid={selectors.owner.panel} className="flex flex-col gap-3">
      <p className="text-sm text-app-muted-foreground">{t(strings.owner.intro)}</p>
      {isOwner ? (
        <div className="flex flex-wrap items-center gap-3">
          <p data-testid={selectors.owner.status} className="text-sm text-app-success">
            {t(strings.owner.signedIn)}
          </p>
          <Button
            data-testid={selectors.owner.signOutButton}
            variant="outline"
            size="sm"
            onClick={clearOwnerToken}
          >
            {t(strings.owner.signOut)}
          </Button>
        </div>
      ) : (
        <div className="flex flex-col gap-2">
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
            size="sm"
            className="w-fit"
            onClick={handleSave}
          >
            {t(strings.owner.signIn)}
          </Button>
          {error && <p className="text-sm text-app-danger">{error}</p>}
        </div>
      )}
    </div>
  );
}
