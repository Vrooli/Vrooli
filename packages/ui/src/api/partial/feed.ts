import { DevelopResult, LearnResult, PopularResult, ResearchResult } from "@shared/consts";
import { GqlPartial } from "types";

export const developResultPartial: GqlPartial<DevelopResult> = {
    __typename: 'DevelopResult',
    list: {
        __define: {
            0: [require('./project').projectPartial, 'list'],
            1: [require('./routine').routinePartial, 'list'],
        },
        completed: {
            __union: {
                Project: 0,
                Routine: 1,
            }
        },
        inProgress: {
            __union: {
                Project: 0,
                Routine: 1,
            }
        },
        recent: {
            __union: {
                Project: 0,
                Routine: 1,
            }
        },
    }
}

export const learnResultPartial: GqlPartial<LearnResult> = {
    __typename: 'LearnResult',
    list: {
        __define: {
            0: [require('./project').projectPartial, 'list'],
            1: [require('./routine').routinePartial, 'list'],
        },
        courses: { __use: 0 },
        tutorials: { __use: 1 },
    }
}

export const popularResultPartial: GqlPartial<PopularResult> = {
    __typename: 'PopularResult',
    list: {
        __define: {
            0: [require('./organization').organizationPartial, 'list'],
            1: [require('./project').projectPartial, 'list'],
            2: [require('./routine').routinePartial, 'list'],
            3: [require('./standard').standardPartial, 'list'],
            4: [require('./user').userPartial, 'list'],
        },
        organizations: { __use: 0 },
        projects: { __use: 1 },
        routines: { __use: 2 },
        standards: { __use: 3 },
        users: { __use: 4 },
    }
}

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