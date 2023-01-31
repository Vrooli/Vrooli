import { SessionUser } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const sessionUserPartial: GqlPartial<SessionUser> = {
    __typename: 'SessionUser',
    full: {
        handle: true,
        hasPremium: true,
        id: true,
        languages: true,
        name: true,
        schedules: async () => relPartial((await import('./userSchedule')).userSchedulePartial, 'full'),
        theme: true,
    }
}