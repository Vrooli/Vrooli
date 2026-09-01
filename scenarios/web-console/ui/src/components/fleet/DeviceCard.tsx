import { Button } from "../ui/button";
import { DeviceSilhouette } from "../terminal/device/DeviceSilhouette";
import type { DeviceArchetype } from "../../lib/deviceArchetype";
import FleetCard from "./FleetCard";
import type { StatusTone } from "@vrooli/react-component-library/StatusBadge/1";
import { useTranslation } from "react-i18next";
import { strings } from "../../consts/strings";

export type DeviceCardState = "in-control" | "following" | "idle" | "reconnecting" | "not-connected";

export interface RosterDevice {
  deviceId: string;
  deviceLabel: string;
  deviceClass: string;
  connectionCount: number;
  isSelf: boolean;
  reconnecting: boolean;
  sessions: Array<{ sessionId: string; sessionName?: string; holdsLease: boolean }>;
}

function archetype(value: string): DeviceArchetype {
  return ["phone", "tablet", "laptop", "monitor", "ultrawide"].includes(value) ? value as DeviceArchetype : "laptop";
}

function cardState(device: RosterDevice): DeviceCardState {
  if (device.reconnecting) return "reconnecting";
  if (device.sessions.some((session) => session.holdsLease)) return "in-control";
  if (device.sessions.length > 0) return "following";
  return device.connectionCount > 0 ? "idle" : "not-connected";
}

export function DeviceCard({ device, onGiveControl, onRename, onDropOld, onForget }: {
  device: RosterDevice;
  onGiveControl?: (device: RosterDevice) => void;
  onRename?: (device: RosterDevice) => void;
  onDropOld?: (device: RosterDevice) => void;
  onForget?: (device: RosterDevice) => void;
}) {
  const { t } = useTranslation();
  const state = cardState(device);
  const stateCopy: Record<DeviceCardState, { label: string; tone: StatusTone }> = {
    "in-control": { label: t(strings.fleet.inControl), tone: "info" },
    following: { label: t(strings.fleet.following), tone: "neutral" },
    idle: { label: t(strings.fleet.idle), tone: "neutral" },
    reconnecting: { label: t(strings.fleet.reconnecting), tone: "warning" },
    "not-connected": { label: t(strings.fleet.notConnected), tone: "neutral" },
  };
  const copy = stateCopy[state];
  const driving = device.sessions.find((session) => session.holdsLease)?.sessionId;
  return (
    <FleetCard
      testId={`fleet-card-device-${device.deviceId}`}
      title={device.deviceLabel || t(strings.fleet.unnamedScreen)}
      meta={device.isSelf ? t(strings.fleet.you) : undefined}
      status={copy.label}
      statusTone={copy.tone}
      silhouette={<DeviceSilhouette archetype={archetype(device.deviceClass)} keyboardShare={0} kbOpen={false} screenLit={state === "in-control"} />}
      actions={(
        <>
          {state === "following" && onGiveControl && <Button size="sm" className="min-h-11" onClick={() => { onGiveControl(device); }}>{t(strings.fleet.giveControl)}</Button>}
          {state === "reconnecting" && onDropOld && <Button size="sm" variant="outline" className="min-h-11" onClick={() => { onDropOld(device); }}>{t(strings.fleet.dropOld)}</Button>}
          {state === "not-connected" && onForget && <Button size="sm" variant="outline" className="min-h-11" onClick={() => { onForget(device); }}>{t(strings.fleet.forgetScreen)}</Button>}
          {onRename && <Button size="sm" variant="ghost" className="min-h-11" onClick={() => { onRename(device); }}>{t(strings.fleet.rename)}</Button>}
        </>
      )}
    >
      {driving ? t(strings.fleet.driving, { session: driving }) : t(strings.fleet.noSession)}
    </FleetCard>
  );
}

export default DeviceCard;
