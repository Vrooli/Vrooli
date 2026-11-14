import { useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { AlertDialog } from "@vrooli/react-component-library/AlertDialog/2";
import { Button } from "@vrooli/react-component-library/Button/2";
import { appDeliveryClient } from "../../api/notifications";
import { QueryError } from "../../components/QueryNotice";
import { ListGroup, ListItem } from "../../components/ListRow";
import { SkeletonRows } from "../../components/Skeleton";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { Notice } from "../settings/Notice";

interface Removal { id: string; app: string; device: string }

/**
 * Which app shows notifications on each device: Chimesail, or a scenario's own
 * app paired from that app's settings. Handing a device back to Chimesail is
 * confirmed first, because pairing again happens on the other app.
 */
export function DeviceAppsPanel() {
  const { t, i18n } = useTranslation();
  const [outcome, setOutcome] = useState<{ ok: boolean; app: string; device: string } | null>(null);
  const [confirming, setConfirming] = useState<Removal | null>(null);
  const devices = useQuery({ queryKey: ["device-apps"], queryFn: () => appDeliveryClient.listDeviceApps({}), refetchOnWindowFocus: true });
  const remove = useMutation({
    mutationFn: async (input: Removal) => {
      await appDeliveryClient.removeDeviceApp({ subscriptionId: input.id });
      return input;
    },
    onSuccess: (input) => setOutcome({ ok: true, app: input.app, device: input.device }),
    onError: (_error, input) => setOutcome({ ok: false, app: input.app, device: input.device }),
    onSettled: () => void devices.refetch(),
  });
  const rows = devices.data?.devices ?? [];
  const lastSeen = (iso: string) => {
    const parsed = new Date(iso);
    return Number.isNaN(parsed.getTime()) ? "" : parsed.toLocaleDateString(i18n.language, { month: "short", day: "numeric" });
  };

  return (
    <section data-testid={selectors.deviceApps.root} aria-labelledby="device-apps-title" className="mt-4 flex flex-col gap-2">
      <h2 id="device-apps-title" className="px-1 text-base font-semibold">{t(strings.reach.deviceApps.title)}</h2>
      <p className="px-1 text-sm text-app-muted-foreground">{t(strings.reach.deviceApps.explanation)}</p>
      {outcome && (
        <Notice tone={outcome.ok ? "success" : "danger"} testId={selectors.deviceApps.message}>
          {t(outcome.ok ? strings.reach.deviceApps.removed : strings.reach.deviceApps.removeError, { app: outcome.app, device: outcome.device })}
        </Notice>
      )}
      {devices.isPending && <SkeletonRows rows={1} />}
      {devices.isError && !devices.data && (
        <QueryError error={devices.error} title={t(strings.reach.deviceApps.loadFailed)} onRetry={() => void devices.refetch()} retrying={devices.isFetching} />
      )}
      {devices.isSuccess && rows.length === 0 && <p className="px-1 text-sm">{t(strings.reach.deviceApps.empty)}</p>}
      {rows.map((device) => (
        <div key={device.group} className="flex flex-col gap-1">
          <div className="flex flex-wrap items-baseline justify-between gap-x-3 px-1">
            <span className="text-sm font-semibold [overflow-wrap:anywhere]">{device.label}</span>
            {lastSeen(device.lastUsed) && <span className="text-xs text-app-muted-foreground">{t(strings.reach.deviceApps.lastSeen, { date: lastSeen(device.lastUsed) })}</span>}
          </div>
          <ListGroup>
            {device.apps.map((app) => (
              <ListItem key={app.subscriptionId} className="flex items-center justify-between gap-3 px-4 py-3">
                <span className="flex min-w-0 flex-col">
                  <span className="font-medium [overflow-wrap:anywhere]">{app.scenario ? app.displayName : t(strings.reach.deviceApps.chimesail)}</span>
                  <span className="text-sm text-app-muted-foreground">{app.status}</span>
                </span>
                {app.scenario && (
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    data-testid={selectors.deviceApps.removeButton}
                    disabled={remove.isPending}
                    onClick={() => setConfirming({ id: app.subscriptionId, app: app.displayName, device: device.label })}
                  >
                    {remove.isPending && remove.variables.id === app.subscriptionId ? t(strings.reach.deviceApps.removing) : t(strings.reach.deviceApps.useChimesail)}
                  </Button>
                )}
              </ListItem>
            ))}
          </ListGroup>
        </div>
      ))}
      <AlertDialog
        open={confirming !== null}
        title={t(strings.reach.deviceApps.confirmTitle, { app: confirming?.app ?? "", device: confirming?.device ?? "" })}
        description={t(strings.reach.deviceApps.confirmBody, { app: confirming?.app ?? "" })}
        confirmLabel={t(strings.reach.deviceApps.useChimesail)}
        cancelLabel={t(strings.settings.action.cancel)}
        closeLabel={t(strings.settings.action.cancel)}
        onConfirm={() => { if (confirming) remove.mutate(confirming); setConfirming(null); }}
        onCancel={() => setConfirming(null)}
      />
    </section>
  );
}
