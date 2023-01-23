import { ResearchResult } from "@shared/consts";
import { GqlPartial } from "types";

export const researchResultPartial: GqlPartial<ResearchResult> = {
    __typename: 'ResearchResult',
    list: {
        __define: {
            0: [require('./routine').routinePartial, 'list'],
            1: [require('./project').projectPartial, 'list'],
            2: [require('./organization').organizationPartial, 'list'],
        },
        processes: { __use: 0 },
        newlyCompleted: { 
            __union: {
                Routine: 0,
                Project: 1,
            },
        },
        needVotes: { __use: 1 },
        needInvestments: { __use: 1 },
        needMembers: { __use: 2 },
    }
}