import { HistoryResult } from "@shared/consts";
import { relPartial } from "../utils";
import { GqlPartial } from "../types";

export const historyResultPartial: GqlPartial<HistoryResult> = {
    __typename: 'HistoryResult',
    list: {
        __define: {
            0: async () => relPartial((await import('./runProject')).runProjectPartial, 'list'),
            1: async () => relPartial((await import('./runRoutine')).runRoutinePartial, 'list'),
            2: async () => relPartial((await import('./view')).viewPartial, 'list'),
            3: async () => relPartial((await import('./star')).starPartial, 'list'),
        },
        activeRuns: {
            __union: {
                RunProject: 0,
                RunRoutine: 1,
            },
        },
        completedRuns: {
            __union: {
                RunProject: 0,
                RunRoutine: 1,
            },
        },
        recentlyViewed: { __use: 2 },
        recentlyStarred: { __use: 3 },
    }
}