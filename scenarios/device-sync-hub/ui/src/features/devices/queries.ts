import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { devicesClient, type Device } from "../../api/devices";

/** Canonical react-query key for the owner-gated device list. */
export const DEVICES_QUERY_KEY = ["devices"] as const;

/**
 * List the owner's devices. Owner-gated, so the caller passes whether an owner
 * token is present; with no token the query stays disabled (the management
 * surface shows a sign-in prompt instead of firing a doomed request).
 */
export function useDevicesQuery(enabled: boolean) {
  return useQuery({
    queryKey: DEVICES_QUERY_KEY,
    queryFn: async (): Promise<Device[]> => {
      const resp = await devicesClient.listDevices({});
      return resp.devices;
    },
    enabled,
  });
}

function useDevicesInvalidate() {
  const queryClient = useQueryClient();
  return () => queryClient.invalidateQueries({ queryKey: DEVICES_QUERY_KEY });
}

export function useRenameDeviceMutation() {
  const invalidate = useDevicesInvalidate();
  return useMutation({
    mutationFn: ({ deviceId, name }: { deviceId: string; name: string }) =>
      devicesClient.renameDevice({ deviceId, name }),
    onSuccess: () => void invalidate(),
  });
}

export function useRevokeDeviceMutation() {
  const invalidate = useDevicesInvalidate();
  return useMutation({
    mutationFn: (deviceId: string) => devicesClient.revokeDevice({ deviceId }),
    onSuccess: () => void invalidate(),
  });
}

export function useApprovePairingMutation() {
  const invalidate = useDevicesInvalidate();
  return useMutation({
    mutationFn: (deviceId: string) => devicesClient.approvePairing({ deviceId }),
    onSuccess: () => void invalidate(),
  });
}

export function useIssuePairingCodeMutation() {
  return useMutation({
    mutationFn: (deviceName: string) => devicesClient.issuePairingCode({ deviceName }),
  });
}

export type { Device };
