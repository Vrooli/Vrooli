import { create } from "zustand";
import { ChatConfigObject } from "@vrooli/shared";

export type DrawerView = "chat" | "swarmDetail";

interface DrawerData {
    view: DrawerView;
    chatId?: string | null;
    swarmConfig?: ChatConfigObject | null;
    swarmStatus?: string;
    onApproveToolCall?: (pendingId: string) => void;
    onRejectToolCall?: (pendingId: string, reason?: string) => void;
    onStart?: () => void;
    onPause?: () => void;
    onResume?: () => void;
    onStop?: () => void;
}

interface DrawerState {
    rightDrawer: DrawerData;
    setRightDrawer: (data: Partial<DrawerData>) => void;
    resetRightDrawer: () => void;
}

const defaultDrawerState: DrawerData = {
    view: "chat",
};

export const useDrawerStore = create<DrawerState>((set) => ({
    rightDrawer: defaultDrawerState,
    setRightDrawer: (data) => set((state) => ({
        rightDrawer: { ...state.rightDrawer, ...data },
    })),
    resetRightDrawer: () => set({ rightDrawer: defaultDrawerState }),
}));