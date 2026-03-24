import { create } from "zustand";
import type { WakeLockStatus } from "../hooks/useWakeLock";

/** Non-persisted store so the settings UI can reflect real-time wake lock status. */
export const useWakeLockStatus = create<{
  status: WakeLockStatus;
  setStatus: (s: WakeLockStatus) => void;
}>((set) => ({
  status: "off",
  setStatus: (status) => set({ status }),
}));
