import { create } from "zustand";
/** Non-persisted store so the settings UI can reflect real-time wake lock status. */
export const useWakeLockStatus = create((set) => ({
    status: "off",
    setStatus: (status) => set({ status }),
}));
