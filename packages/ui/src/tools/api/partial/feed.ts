import { HomeResult, PopularResult } from "@shared/consts";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const homeResult: GqlPartial<HomeResult> = {
    __typename: 'HomeResult',
    list: {
        __define: {
            0: async () => rel((await import('./meeting')).meeting, 'list'),
            1: async () => rel((await import('./note')).note, 'list'),
            2: async () => rel((await import('./reminder')).reminder, 'list'),
            3: async () => rel((await import('./resource')).resource, 'list'),
            4: async () => rel((await import('./runProjectSchedule')).runProjectSchedule, 'list'),
            5: async () => rel((await import('./runRoutineSchedule')).runRoutineSchedule, 'list'),
        },
        meetings: { __use: 0 },
        notes: { __use: 1 },
        reminders: { __use: 2 },
        resources: { __use: 3 },
        runProjectSchedules: { __use: 4 },
        runRoutineSchedules: { __use: 5 },
    }
}

export const popularResult: GqlPartial<PopularResult> = {
    __typename: 'PopularResult',
    list: {
        __define: {
            0: async () => rel((await import('./api')).api, 'list'),
            1: async () => rel((await import('./note')).note, 'list'),
            2: async () => rel((await import('./organization')).organization, 'list'),
            3: async () => rel((await import('./project')).project, 'list'),
            4: async () => rel((await import('./question')).question, 'list'),
            5: async () => rel((await import('./routine')).routine, 'list'),
            6: async () => rel((await import('./smartContract')).smartContract, 'list'),
            7: async () => rel((await import('./standard')).standard, 'list'),
            8: async () => rel((await import('./user')).user, 'list'),
        },
        apis: { __use: 0 },
        notes: { __use: 1 },
        organizations: { __use: 2 },
        projects: { __use: 3 },
        questions: { __use: 4 },
        routines: { __use: 5 },
        smartContracts: { __use: 6 },
        standards: { __use: 7 },
        users: { __use: 8 },
    }
}