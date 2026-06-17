import { useState } from "react";
import { useMutation } from "@tanstack/react-query";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { devicesClient } from "../../api/devices";
import { browserDeviceProfile } from "../session/deviceProfile";
import { useSession } from "../session/SessionProvider";

/**
 * "Join an existing hub" path of onboarding. Two ways in: redeem a pairing code
 * (primary — TRUSTED immediately) or request approval (fallback — PENDING until
 * an already-trusted device approves). On success the returned token + device
 * are persisted; the request path then waits for the realtime approval to flip
 * the device to trusted (the app re-renders once paired).
 *
 * Rendered inside OnboardingScreen's card chrome, so this is a panel (no
 * full-screen wrapper). `onBack` returns to the welcome chooser.
 */
export function JoinHubForm({ onBack }: { onBack: () => void }) {
  const { t } = useTranslation();
  const { setDeviceCredentials } = useSession();
  const [code, setCode] = useState("");
  const [deviceName, setDeviceName] = useState("");
  const [localError, setLocalError] = useState<string | null>(null);
  const [requested, setRequested] = useState(false);

  const redeemMutation = useMutation({
    mutationFn: () =>
      devicesClient.redeemPairingCode({
        code: code.trim(),
        profile: browserDeviceProfile(deviceName.trim() || "This device"),
      }),
    onSuccess: (resp) => setDeviceCredentials(resp.deviceToken, resp.device ?? null),
  });

  const requestMutation = useMutation({
    mutationFn: () =>
      devicesClient.requestPairing({
        profile: browserDeviceProfile(deviceName.trim()),
      }),
    onSuccess: (resp) => {
      setDeviceCredentials(resp.deviceToken, resp.device ?? null);
      setRequested(true);
    },
  });

  const handleRedeem = () => {
    setLocalError(null);
    if (!code.trim()) {
      setLocalError(t(strings.join.missingCode));
      return;
    }
    redeemMutation.mutate();
  };

  const handleRequest = () => {
    setLocalError(null);
    if (!deviceName.trim()) {
      setLocalError(t(strings.join.missingName));
      return;
    }
    requestMutation.mutate();
  };

  const mutationError = redeemMutation.error ?? requestMutation.error;

  return (
    <section
      data-testid={selectors.join.screen}
      aria-labelledby="join-heading"
      className="rounded-panel border border-app-border bg-app-surface p-6 shadow-sm"
    >
      <h1 id="join-heading" className="text-xl font-semibold">
        {t(strings.join.title)}
      </h1>
      <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.join.intro)}</p>

      {requested ? (
        <div
          data-testid={selectors.join.waiting}
          className="mt-6 rounded-panel border border-app-border bg-app-surface-muted p-4"
        >
          <h2 className="text-sm font-semibold">{t(strings.join.waitingTitle)}</h2>
          <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.join.waitingIntro)}</p>
        </div>
      ) : (
        <div className="mt-6 flex flex-col gap-4">
          <div className="flex flex-col gap-2">
            <label htmlFor="join-name" className="text-xs font-medium text-app-muted-foreground">
              {t(strings.join.deviceNameLabel)}
            </label>
            <Input
              id="join-name"
              data-testid={selectors.join.deviceNameInput}
              value={deviceName}
              onChange={(e) => setDeviceName(e.target.value)}
              placeholder={t(strings.join.deviceNamePlaceholder)}
            />
          </div>

          <div className="flex flex-col gap-2">
            <label htmlFor="join-code" className="text-xs font-medium text-app-muted-foreground">
              {t(strings.join.codeLabel)}
            </label>
            <Input
              id="join-code"
              data-testid={selectors.join.codeInput}
              value={code}
              onChange={(e) => setCode(e.target.value)}
              placeholder={t(strings.join.codePlaceholder)}
            />
            <Button
              data-testid={selectors.join.redeemButton}
              onClick={handleRedeem}
              disabled={redeemMutation.isPending}
            >
              {t(strings.join.redeem)}
            </Button>
          </div>

          <div className="rounded-panel border border-app-border p-4">
            <h2 className="text-sm font-semibold">{t(strings.join.requestTitle)}</h2>
            <p className="mt-1 text-xs text-app-muted-foreground">{t(strings.join.requestIntro)}</p>
            <Button
              data-testid={selectors.join.requestButton}
              variant="outline"
              className="mt-3"
              onClick={handleRequest}
              disabled={requestMutation.isPending}
            >
              {t(strings.join.request)}
            </Button>
          </div>
        </div>
      )}

      {(localError || mutationError) && (
        <p data-testid={selectors.join.error} className="mt-4 text-sm text-app-danger">
          {localError ?? errorMessage(mutationError, t)}
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
