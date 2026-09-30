import { useState } from "react";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { Check, Pencil, ShieldOff } from "lucide-react";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { TrustState, type Device } from "../../api/devices";
import {
  useApprovePairingMutation,
  useDevicesQuery,
  useRenameDeviceMutation,
  useRevokeDeviceMutation,
} from "./queries";

const TRUST_LABEL = {
  [TrustState.UNSPECIFIED]: strings.devices.trust.unspecified,
  [TrustState.PENDING]: strings.devices.trust.pending,
  [TrustState.TRUSTED]: strings.devices.trust.trusted,
  [TrustState.REVOKED]: strings.devices.trust.revoked,
} as const satisfies Record<TrustState, string>;

/** Owner-gated device list with rename / revoke / approve actions. Presence is
 * the live `online` flag the server overlays from the realtime hub. */
export function DeviceList() {
  const { t } = useTranslation();
  const devicesQuery = useDevicesQuery(true);
  const rename = useRenameDeviceMutation();
  const revoke = useRevokeDeviceMutation();
  const approve = useApprovePairingMutation();
  const [editingId, setEditingId] = useState<string | null>(null);
  const [draftName, setDraftName] = useState("");

  const startRename = (device: Device) => {
    setEditingId(device.id);
    setDraftName(device.name);
  };

  const commitRename = (deviceId: string) => {
    if (draftName.trim()) {
      rename.mutate({ deviceId, name: draftName.trim() });
    }
    setEditingId(null);
  };

  const handleRevoke = (device: Device) => {
    if (window.confirm(t(strings.devices.revokeConfirm, { name: device.name }))) {
      revoke.mutate(device.id);
    }
  };

  if (devicesQuery.isLoading) {
    return (
      <p data-testid={selectors.devices.loading} className="text-sm text-app-muted-foreground">
        {t(strings.devices.loading)}
      </p>
    );
  }
  if (devicesQuery.error) {
    return <p className="text-sm text-app-danger">{errorMessage(devicesQuery.error, t)}</p>;
  }
  const devices = devicesQuery.data ?? [];
  if (devices.length === 0) {
    return (
      <p data-testid={selectors.devices.empty} className="text-sm text-app-muted-foreground">
        {t(strings.devices.empty)}
      </p>
    );
  }

  return (
    <ul data-testid={selectors.devices.list} className="flex flex-col gap-2">
      {devices.map((device) => (
        <li
          key={device.id}
          data-testid={selectors.devices.row({ id: device.id })}
          className="flex flex-wrap items-center justify-between gap-3 rounded-panel border border-app-border bg-app-background p-3"
        >
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <span
                className={[
                  "inline-block h-2 w-2 shrink-0 rounded-pill",
                  device.online ? "bg-app-success" : "bg-app-muted-foreground",
                ].join(" ")}
                aria-label={device.online ? t(strings.devices.onlineLabel) : t(strings.devices.offlineLabel)}
                role="img"
              />
              {editingId === device.id ? (
                <Input
                  value={draftName}
                  onChange={(e) => setDraftName(e.target.value)}
                  aria-label={t(strings.devices.renameLabel)}
                  className="h-8 w-40"
                />
              ) : (
                <span className="truncate text-sm font-medium text-app-foreground">{device.name}</span>
              )}
            </div>
            <p className="mt-1 text-xs text-app-muted-foreground">
              {device.kind || device.platform} · {t(TRUST_LABEL[device.trustState])}
              {device.lastSeenAt
                ? ` · ${formatDate(timestampDate(device.lastSeenAt), { dateStyle: "short", timeStyle: "short" })}`
                : ""}
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            {device.trustState === TrustState.PENDING && (
              <Button
                data-testid={selectors.devices.approve({ id: device.id })}
                size="sm"
                onClick={() => approve.mutate(device.id)}
                disabled={approve.isPending}
              >
                <Check aria-hidden="true" className="me-1 h-4 w-4" />
                {t(strings.devices.approve)}
              </Button>
            )}
            {editingId === device.id ? (
              <Button size="sm" variant="outline" onClick={() => commitRename(device.id)}>
                {t(strings.devices.renameSave)}
              </Button>
            ) : (
              <Button
                data-testid={selectors.devices.rename({ id: device.id })}
                size="sm"
                variant="outline"
                onClick={() => startRename(device)}
                aria-label={t(strings.devices.rename)}
              >
                <Pencil aria-hidden="true" className="h-4 w-4" />
              </Button>
            )}
            {device.trustState !== TrustState.REVOKED && (
              <Button
                data-testid={selectors.devices.revoke({ id: device.id })}
                size="sm"
                variant="outline"
                onClick={() => handleRevoke(device)}
                disabled={revoke.isPending}
                aria-label={t(strings.devices.revoke)}
              >
                <ShieldOff aria-hidden="true" className="h-4 w-4" />
              </Button>
            )}
          </div>
        </li>
      ))}
    </ul>
  );
}
