import { HistoryResult } from "@shared/consts";
import { rel } from "../utils";
import { GqlPartial } from "../types";

export const historyResult: GqlPartial<HistoryResult> = {
    __typename: 'HistoryResult',
    list: {
        __define: {
            0: async () => rel((await import('./runProject')).runProject, 'list'),
            1: async () => rel((await import('./runRoutine')).runRoutine, 'list'),
            2: async () => rel((await import('./view')).view, 'list'),
            3: async () => rel((await import('./star')).star, 'list'),
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