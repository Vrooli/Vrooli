import { RunProjectSchedule, RunProjectScheduleTranslation } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const runProjectScheduleTranslation: GqlPartial<RunProjectScheduleTranslation> = {
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

export const runProjectSchedule: GqlPartial<RunProjectSchedule> = {
    __typename: 'RunProjectSchedule',
    common: {
        id: true,
        timeZone: true,
        windowStart: true,
        windowEnd: true,
        recurrStart: true,
        recurrEnd: true,
        runProject: async () => rel((await import('./runProject')).runProject, 'nav', { omit: 'runProjectSchedule' }),
        translations: () => rel(runProjectScheduleTranslation, 'full'),
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