import { createClient } from "@connectrpc/connect";
import { DeviceService } from "@vrooli/proto-types/web-console/v1/devices/devices_pb";

import { deviceIdentity, setDeviceLabel } from "../lib/deviceIdentity";
import { transport } from "./client";

export const deviceClient = createClient(DeviceService, transport);

export interface DeviceSession {
  sessionId: string;
  sessionName: string;
  holdsLease: boolean;
}

export interface RosterDevice {
  deviceId: string;
  deviceLabel: string;
  deviceClass: string;
  connectionCount: number;
  firstSeenAt: string;
  sessions: DeviceSession[];
  isSelf: boolean;
  reconnecting: boolean;
}

function timestampToString(value: { seconds: bigint | number } | undefined): string {
  if (!value) return "";
  const seconds = Number(value.seconds);
  return Number.isFinite(seconds) && seconds > 0 ? new Date(seconds * 1000).toISOString() : "";
}

export async function listDevices(): Promise<RosterDevice[]> {
  const response = await deviceClient.list({ selfDeviceId: deviceIdentity().id });
  return response.devices.map((device) => ({
    deviceId: device.deviceId,
    deviceLabel: device.deviceLabel,
    deviceClass: device.deviceClass,
    connectionCount: device.connectionCount,
    firstSeenAt: timestampToString(device.firstSeenAt),
    sessions: device.sessions.map((session) => ({ sessionId: session.sessionId, sessionName: session.sessionName, holdsLease: session.holdsLease })),
    isSelf: device.isSelf,
    reconnecting: device.reconnecting,
  }));
}

export async function disconnectDevice(deviceId: string, connectionId = ""): Promise<number> {
  const response = await deviceClient.disconnect({ deviceId, connectionId });
  return response.closedConnections;
}

export async function giveControl(deviceId: string, sessionId = ""): Promise<boolean> {
  const response = await deviceClient.giveControl({ deviceId, sessionId });
  return response.transferred;
}


export function renameOwnDevice(label: string): void {
  setDeviceLabel(label);
}
