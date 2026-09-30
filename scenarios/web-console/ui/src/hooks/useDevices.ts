import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { disconnectDevice, giveControl, listDevices, renameOwnDevice } from "../api/devices";

export const DEVICES_QUERY_KEY = ["devices", "roster"] as const;

export function useDevices(enabled: boolean) {
  return useQuery({
    queryKey: DEVICES_QUERY_KEY,
    queryFn: listDevices,
    enabled,
  });
}

export function useDeviceMutations() {
  const queryClient = useQueryClient();
  const invalidate = () => queryClient.invalidateQueries({ queryKey: DEVICES_QUERY_KEY });
  return {
    disconnect: useMutation({ mutationFn: ({ deviceId, connectionId }: { deviceId: string; connectionId?: string }) => disconnectDevice(deviceId, connectionId), onSuccess: invalidate }),
    giveControl: useMutation({ mutationFn: ({ deviceId, sessionId }: { deviceId: string; sessionId?: string }) => giveControl(deviceId, sessionId), onSuccess: invalidate }),
    rename: useMutation({ mutationFn: (label: string) => { renameOwnDevice(label); return Promise.resolve(); }, onSuccess: invalidate }),
  };
}
