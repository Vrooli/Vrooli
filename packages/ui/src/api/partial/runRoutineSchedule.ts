import { RunRoutineSchedule, RunRoutineScheduleTranslation } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

export const runRoutineScheduleTranslationPartial: GqlPartial<RunRoutineScheduleTranslation> = {
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

export const runRoutineSchedulePartial: GqlPartial<RunRoutineSchedule> = {
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
        runRoutine: () => relPartial(require('./runRoutine').runRoutinePartial, 'nav', { omit: 'runRoutineSchedule' }),
        translations: () => relPartial(runRoutineScheduleTranslationPartial, 'full'),
    },
    full: {
        __define: {
            0: [require('./label').labelPartial, 'full'],
        },
        labels: { __use: 0 },
    },
    list: {
        __define: {
            0: [require('./label').labelPartial, 'list'],
        },
        labels: { __use: 0 },
    }
}