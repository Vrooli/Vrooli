import { RunRoutineSchedule, RunRoutineScheduleTranslation } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const runRoutineScheduleTranslation: GqlPartial<RunRoutineScheduleTranslation> = {
    __typename: 'RunRoutineScheduleTranslation',
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
    full: {},
    list: {},
}

export const runRoutineSchedule: GqlPartial<RunRoutineSchedule> = {
    __typename: 'RunRoutineSchedule',
    common: {
        id: true,
        attemptAutomatic: true,
        maxAutomaticAttempts: true,
        timeZone: true,
        windowStart: true,
        windowEnd: true,
        recurrStart: true,
        recurrEnd: true,
        runRoutine: async () => rel((await import('./runRoutine')).runRoutine, 'nav', { omit: 'runRoutineSchedule' }),
        translations: () => rel(runRoutineScheduleTranslation, 'full'),
    },
    full: {
        __define: {
            0: async () => rel((await import('./label')).label, 'full'),
        },
        labels: { __use: 0 },
    },
    list: {
        __define: {
            0: async () => rel((await import('./label')).label, 'list'),
        },
        labels: { __use: 0 },
    }
}