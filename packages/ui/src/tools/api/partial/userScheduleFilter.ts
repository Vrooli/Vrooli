import { UserScheduleFilter } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const userScheduleFilter: GqlPartial<UserScheduleFilter> = {
    __typename: 'UserScheduleFilter',
    full: {
        id: true,
        filterType: true,
        tag: async () => rel((await import('./tag')).tag, 'list'),
        userSchedule: async () => rel((await import('./userSchedule')).userSchedule, 'list', { omit: 'filters' }),
    },
}