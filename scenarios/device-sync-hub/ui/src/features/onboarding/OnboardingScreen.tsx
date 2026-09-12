import { useState } from "react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { useSession } from "../session/SessionProvider";
import { OwnerLoginForm } from "./OwnerLoginForm";
import { SetupDeviceStep } from "./SetupDeviceStep";
import { JoinHubForm } from "./JoinHubForm";

type OnboardingMode = "choose" | "setup" | "join";

/**
 * First-run experience shown until this browser holds a device token. A small
 * state machine over three modes:
 *
 *   choose → setup → (login → make-first-device) → paired
 *          ↘ join → (redeem / request) ───────────→ paired
 *
 * The "setup" mode renders the login form until an owner JWT is present, then
 * the make-this-my-first-device step — so a returning owner (token already
 * stored) skips straight to device setup. The gate in App.tsx swaps this for the
 * routed shell the moment a device token lands.
 */
export function OnboardingScreen() {
  const { t } = useTranslation();
  const { isOwner, isPendingApproval } = useSession();
  const [mode, setMode] = useState<OnboardingMode>(
    isPendingApproval ? "join" : isOwner ? "setup" : "choose",
  );

  return (
    <div
      data-testid={selectors.onboarding.screen}
      className="grid min-h-screen place-items-center bg-app-background p-4 text-app-foreground"
    >
      <div className="w-full max-w-md">
        <header className="mb-6 text-center">
          <h1 className="text-2xl font-semibold">{t(strings.onboarding.welcomeTitle)}</h1>
          <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.onboarding.welcomeIntro)}</p>
        </header>

        {mode === "choose" && (
          <div className="flex flex-col gap-4">
            <button
              type="button"
              data-testid={selectors.onboarding.setupChoice}
              onClick={() => setMode("setup")}
              className="rounded-panel border border-app-border bg-app-surface p-5 text-left shadow-sm transition-colors hover:border-app-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
            >
              <span className="block text-base font-semibold">{t(strings.onboarding.setupTitle)}</span>
              <span className="mt-1 block text-sm text-app-muted-foreground">
                {t(strings.onboarding.setupDescription)}
              </span>
            </button>
            <button
              type="button"
              data-testid={selectors.onboarding.joinChoice}
              onClick={() => setMode("join")}
              className="rounded-panel border border-app-border bg-app-surface p-5 text-left shadow-sm transition-colors hover:border-app-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
            >
              <span className="block text-base font-semibold">{t(strings.onboarding.joinTitle)}</span>
              <span className="mt-1 block text-sm text-app-muted-foreground">
                {t(strings.onboarding.joinDescription)}
              </span>
            </button>
          </div>
        )}

        {mode === "setup" &&
          (isOwner ? (
            <SetupDeviceStep
              onJoinInstead={() => setMode("join")}
              onSignOut={() => setMode("choose")}
            />
          ) : (
            <OwnerLoginForm onBack={() => setMode("choose")} />
          ))}

        {mode === "join" && <JoinHubForm onBack={() => setMode("choose")} />}
      </div>
    </div>
  );
}
