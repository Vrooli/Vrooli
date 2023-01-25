import { RunRoutineSchedule, RunRoutineScheduleTranslation } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

export const runRoutineScheduleTranslationPartial: GqlPartial<RunRoutineScheduleTranslation> = {
    __typename: 'RunRoutineScheduleTranslation',
    full: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
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
        labels: () => relPartial(require('./label').labelPartial, 'full'),
    },
    list: {
        labels: () => relPartial(require('./label').labelPartial, 'list'),

    }
}