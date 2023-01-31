import { UserScheduleFilter } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const userScheduleFilterPartial: GqlPartial<UserScheduleFilter> = {
    __typename: 'UserScheduleFilter',
    full: {
        id: true,
        filterType: true,
        tag: async () => relPartial((await import('./tag')).tagPartial, 'list'),
        userSchedule: async () => relPartial((await import('./userSchedule')).userSchedulePartial, 'list', { omit: 'filters' }),
    },
}