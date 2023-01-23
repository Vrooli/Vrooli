import { HistoryResult } from "@shared/consts";
import { GqlPartial } from "types";

export const historyResultPartial: GqlPartial<HistoryResult> = {
    __typename: 'HistoryResult',
    list: {
        __define: {
            0: [require('./runProject').runProjectPartial, 'list'],
            1: [require('./runRoutine').runRoutinePartial, 'list'],
            2: [require('./view').viewPartial, 'list'],
            3: [require('./star').starPartial, 'list'],
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