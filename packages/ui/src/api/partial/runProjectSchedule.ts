import { RunProjectSchedule, RunProjectScheduleTranslation } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

export const runProjectScheduleTranslationPartial: GqlPartial<RunProjectScheduleTranslation> = {
    __typename: 'RunProjectScheduleTranslation',
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
    full: {},
    list: {},
}

export const runProjectSchedulePartial: GqlPartial<RunProjectSchedule> = {
    __typename: 'RunProjectSchedule',
    common: {
        id: true,
        timeZone: true,
        windowStart: true,
        windowEnd: true,
        recurrStart: true,
        recurrEnd: true,
        runProject: () => relPartial(require('./runProject').runProjectPartial, 'nav', { omit: 'runProjectSchedule' }),
        translations: () => relPartial(runProjectScheduleTranslationPartial, 'full'),
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