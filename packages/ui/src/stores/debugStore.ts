import { create } from "zustand";

interface TraceData {
    count: number;
    data: any;
}

interface DebugState {
    traces: Record<string, TraceData>;
    addData: (traceId: string, data: any) => void;
}

export const useDebugStore = create<DebugState>((set) => ({
    traces: {},
    addData: (traceId, data) => set((state) => {
        const current = state.traces[traceId] || { count: 0, data: null };
        return {
            traces: {
                ...state.traces,
                [traceId]: {
                    count: current.count + 1,
                    data,
                },
            },
        };
    }),
})); 
