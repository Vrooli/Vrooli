import { UserScheduleFilter } from "@shared/consts";
import { relPartial } from "api/utils";
import { GqlPartial } from "types";

export const userScheduleFilterPartial: GqlPartial<UserScheduleFilter> = {
    __typename: 'UserScheduleFilter',
    full: {
        id: true,
        filterType: true,
        tag: () => relPartial(require('./tag').tagPartial, 'list'),
        userSchedule: () => relPartial(require('./userSchedule').userSchedulePartial, 'nav', { omit: 'filters' }),
    },
}