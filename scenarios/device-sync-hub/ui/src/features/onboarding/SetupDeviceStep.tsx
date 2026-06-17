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
 * "Make this my first device" — shown once the owner is signed in but this
 * browser isn't paired. Calls SetupOwnerDevice, which claims the hub (if
 * unclaimed) and trusts this client directly (no pairing code), then persists
 * the returned device token so the app advances to the paired shell.
 */
export function SetupDeviceStep({
  onJoinInstead,
  onSignOut,
}: {
  onJoinInstead: () => void;
  onSignOut: () => void;
}) {
  const { t } = useTranslation();
  const { ownerEmail, setDeviceCredentials, clearOwnerToken } = useSession();
  const [deviceName, setDeviceName] = useState("");

  const setupMutation = useMutation({
    mutationFn: () =>
      devicesClient.setupOwnerDevice({
        profile: browserDeviceProfile(deviceName.trim() || "This device"),
      }),
    onSuccess: (resp) => setDeviceCredentials(resp.deviceToken, resp.device ?? null),
  });

  const handleSignOut = () => {
    clearOwnerToken();
    onSignOut();
  };

  return (
    <section
      data-testid={selectors.setupDevice.panel}
      aria-labelledby="setup-device-heading"
      className="rounded-panel border border-app-border bg-app-surface p-6 shadow-sm"
    >
      <h1 id="setup-device-heading" className="text-xl font-semibold">
        {t(strings.setupDevice.title)}
      </h1>
      <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.setupDevice.intro)}</p>
      {ownerEmail && (
        <p className="mt-2 text-xs text-app-muted-foreground">
          {t(strings.setupDevice.signedInAs, { owner: ownerEmail })}
        </p>
      )}

      <form
        className="mt-6 flex flex-col gap-4"
        onSubmit={(e) => {
          e.preventDefault();
          setupMutation.mutate();
        }}
      >
        <div className="flex flex-col gap-2">
          <label htmlFor="setup-device-name" className="text-xs font-medium text-app-muted-foreground">
            {t(strings.setupDevice.deviceNameLabel)}
          </label>
          <Input
            id="setup-device-name"
            data-testid={selectors.setupDevice.nameInput}
            value={deviceName}
            onChange={(e) => setDeviceName(e.target.value)}
            placeholder={t(strings.setupDevice.deviceNamePlaceholder)}
          />
        </div>
        <Button
          type="submit"
          data-testid={selectors.setupDevice.submit}
          disabled={setupMutation.isPending}
        >
          {setupMutation.isPending ? t(strings.setupDevice.submitting) : t(strings.setupDevice.submit)}
        </Button>
      </form>

      {setupMutation.error && (
        <p data-testid={selectors.setupDevice.error} className="mt-4 text-sm text-app-danger">
          {errorMessage(setupMutation.error, t)}
        </p>
      )}

      <div className="mt-6 flex flex-wrap items-center gap-4 border-t border-app-border pt-4 text-sm">
        <button
          type="button"
          data-testid={selectors.setupDevice.joinInstead}
          className="font-medium text-app-muted-foreground hover:text-app-foreground"
          onClick={onJoinInstead}
        >
          {t(strings.setupDevice.joinInstead)}
        </button>
        <button
          type="button"
          data-testid={selectors.setupDevice.signOut}
          className="font-medium text-app-muted-foreground hover:text-app-foreground"
          onClick={handleSignOut}
        >
          {t(strings.setupDevice.signOut)}
        </button>
      </div>
    </section>
  );
}
