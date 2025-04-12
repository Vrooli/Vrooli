import { create } from "zustand";

export type LayoutComponentId = "navigator" | "primary" | "secondary";
export type LayoutPositionId = "left" | "main" | "right";

interface LayoutState {
    positions: Record<LayoutPositionId, LayoutComponentId>;
    routeControlledPosition: LayoutPositionId;
    swapLeftAndMain: () => void;
    swapMainAndRight: () => void;
    swapLeftAndRight: () => void;
    setRouteControlledComponent: (position: LayoutPositionId, componentId: LayoutComponentId) => void;
}

export const useLayoutStore = create<LayoutState>((set) => ({
    positions: {
        left: "navigator",
        main: "primary",
        right: "secondary",
    },
    routeControlledPosition: "main",

    swapLeftAndMain: () => set((state) => {
        const { left, main } = state.positions;
        return {
            positions: {
                ...state.positions,
                left: main,
                main: left,
            },
        };
    }),

    swapMainAndRight: () => set((state) => {
        const { main, right } = state.positions;
        return {
            positions: {
                ...state.positions,
                main: right,
                right: main,
            },
        };
    }),

    swapLeftAndRight: () => set((state) => {
        const { left, right } = state.positions;
        return {
            positions: {
                ...state.positions,
                left: right,
                right: left,
            },
        };
    }),

    setRouteControlledComponent: (position, componentId) => set((state) => ({
        positions: {
            ...state.positions,
            [position]: componentId,
        },
    })),
}));
