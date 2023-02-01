import { DevelopResult, LearnResult, PopularResult, ResearchResult } from "@shared/consts";
import { rel } from "../utils";
import { GqlPartial } from "../types";

export const developResult: GqlPartial<DevelopResult> = {
    __typename: 'DevelopResult',
    list: {
        __define: {
            0: async () => rel((await import('./project')).project, 'list'),
            1: async () => rel((await import('./routine')).routine, 'list'),
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

export const learnResult: GqlPartial<LearnResult> = {
    __typename: 'LearnResult',
    list: {
        __define: {
            0: async () => rel((await import('./project')).project, 'list'),
            1: async () => rel((await import('./routine')).routine, 'list'),
        },
        courses: { __use: 0 },
        tutorials: { __use: 1 },
    }
}

export const popularResult: GqlPartial<PopularResult> = {
    __typename: 'PopularResult',
    list: {
        __define: {
            0: async () => rel((await import('./organization')).organization, 'list'),
            1: async () => rel((await import('./project')).project, 'list'),
            2: async () => rel((await import('./routine')).routine, 'list'),
            3: async () => rel((await import('./standard')).standard, 'list'),
            4: async () => rel((await import('./user')).user, 'list'),
        },
        organizations: { __use: 0 },
        projects: { __use: 1 },
        routines: { __use: 2 },
        standards: { __use: 3 },
        users: { __use: 4 },
    }
}

export const researchResult: GqlPartial<ResearchResult> = {
    __typename: 'ResearchResult',
    list: {
        __define: {
            0: async () => rel((await import('./routine')).routine, 'list'),
            1: async () => rel((await import('./project')).project, 'list'),
            2: async () => rel((await import('./organization')).organization, 'list'),
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